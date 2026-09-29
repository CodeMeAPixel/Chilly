package musicbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgolink/v3/lavalink"
)

var ErrLyricsNotFound = errors.New("no lyrics found for this song")

type LyricLine struct {
	TimeMs int64  `json:"time_ms"`
	Text   string `json:"text"`
}

type Lyrics struct {
	Title        string      `json:"title"`
	Artist       string      `json:"artist"`
	Album        string      `json:"album,omitempty"`
	DurationMs   int64       `json:"duration_ms"`
	Instrumental bool        `json:"instrumental"`
	Plain        string      `json:"plain"`
	Synced       []LyricLine `json:"synced,omitempty"`
	Source       string      `json:"source"`
}

type SongQuery struct {
	Title      string
	Artist     string
	DurationMs int64
}

type LyricsClient struct {
	baseURL string
	http    *http.Client
	cache   *ttlCache[*Lyrics]
	misses  *ttlCache[bool]
}

func NewLyricsClient(baseURL string) *LyricsClient {
	if baseURL == "" {
		baseURL = "https://lrclib.net"
	}
	return &LyricsClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 8 * time.Second},
		cache:   newTTLCache[*Lyrics](6*time.Hour, 2000),
		misses:  newTTLCache[bool](30*time.Minute, 2000),
	}
}

var (
	bracketNoise  = regexp.MustCompile(`(?i)\s*[\(\[][^\)\]]*\b(official|video|audio|lyrics?|visuali[sz]er|remaster(ed)?|hd|hq|4k|explicit|clean|live|prod\.?)\b[^\)\]]*[\)\]]`)
	bracketFeat   = regexp.MustCompile(`(?i)\s*[\(\[]\s*(ft\.?|feat\.?|featuring)\s[^\)\]]*[\)\]]`)
	trailingFeat  = regexp.MustCompile(`(?i)\s+\b(ft\.?|feat\.?|featuring)\s.*$`)
	channelSuffix = regexp.MustCompile(`(?i)\s*(-\s*topic|vevo|official)$`)
	spaces        = regexp.MustCompile(`\s+`)
	anyBrackets   = regexp.MustCompile(`\s*[\(\[][^\)\]]*[\)\]]`)
)

const (
	strictLengthTolerance = 15.0
	looseLengthTolerance  = 45.0
)

func cleanPart(s string) string {
	s = bracketNoise.ReplaceAllString(s, "")
	s = bracketFeat.ReplaceAllString(s, "")
	s = trailingFeat.ReplaceAllString(s, "")
	s = strings.Trim(s, " -–—|\"'")
	return spaces.ReplaceAllString(s, " ")
}

func QueryFromTrack(track lavalink.Track) SongQuery {
	title := track.Info.Title
	artist := channelSuffix.ReplaceAllString(strings.TrimSpace(track.Info.Author), "")

	if track.Info.SourceName == "youtube" || track.Info.SourceName == "soundcloud" {
		for _, sep := range []string{" - ", " – ", " — "} {
			if left, right, ok := strings.Cut(title, sep); ok {
				artist, title = left, right
				break
			}
		}
	}

	q := SongQuery{Title: cleanPart(title), Artist: cleanPart(artist)}
	if !track.Info.IsStream {
		q.DurationMs = int64(track.Info.Length)
	}
	return q
}

type lrclibRecord struct {
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	AlbumName    string  `json:"albumName"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
}

func (c *LyricsClient) Lookup(ctx context.Context, q SongQuery) (*Lyrics, error) {
	if strings.TrimSpace(q.Title) == "" {
		return nil, ErrLyricsNotFound
	}
	key := strings.ToLower(q.Artist + "|" + q.Title + "|" + strconv.FormatInt(q.DurationMs/5000, 10))
	if cached, ok := c.cache.Get(key); ok {
		return cached, nil
	}
	if _, missed := c.misses.Get(key); missed {
		return nil, ErrLyricsNotFound
	}

	record, err := c.find(ctx, q)
	if err != nil {
		if errors.Is(err, ErrLyricsNotFound) {
			c.misses.Set(key, true)
		}
		return nil, err
	}
	lyrics := &Lyrics{
		Title:        record.TrackName,
		Artist:       record.ArtistName,
		Album:        record.AlbumName,
		DurationMs:   int64(record.Duration * 1000),
		Instrumental: record.Instrumental,
		Plain:        strings.TrimSpace(record.PlainLyrics),
		Synced:       ParseLRC(record.SyncedLyrics),
		Source:       "LRCLIB",
	}
	if lyrics.Plain == "" && len(lyrics.Synced) > 0 {
		lines := make([]string, len(lyrics.Synced))
		for i, l := range lyrics.Synced {
			lines[i] = l.Text
		}
		lyrics.Plain = strings.Join(lines, "\n")
	}
	c.cache.Set(key, lyrics)
	return lyrics, nil
}

const maxExactLookupMs = 3600 * 1000

func (c *LyricsClient) find(ctx context.Context, q SongQuery) (*lrclibRecord, error) {
	if q.Artist != "" && q.DurationMs >= 1000 && q.DurationMs <= maxExactLookupMs {
		params := url.Values{"track_name": {q.Title}, "artist_name": {q.Artist}, "duration": {strconv.FormatInt(q.DurationMs/1000, 10)}}
		var record lrclibRecord
		if err := c.get(ctx, "/api/get", params, &record); err == nil && hasLyrics(record) {
			return &record, nil
		} else if err != nil && !errors.Is(err, ErrLyricsNotFound) {
			return nil, err
		}
	}

	titles := []string{q.Title}
	if bare := strings.TrimSpace(spaces.ReplaceAllString(anyBrackets.ReplaceAllString(q.Title, ""), " ")); bare != "" && bare != q.Title {
		titles = append(titles, bare)
	}

	var searches []url.Values
	for _, title := range titles {
		if q.Artist != "" {
			searches = append(searches, url.Values{"track_name": {title}, "artist_name": {q.Artist}})
		}
		searches = append(searches, url.Values{"q": {strings.TrimSpace(q.Artist + " " + title)}})
	}

	var candidates []lrclibRecord
	for _, params := range searches {
		var records []lrclibRecord
		if err := c.get(ctx, "/api/search", params, &records); err != nil {
			if errors.Is(err, ErrLyricsNotFound) {
				continue
			}
			return nil, err
		}
		if best := pickBest(records, q.DurationMs, strictLengthTolerance); best != nil {
			return best, nil
		}
		candidates = append(candidates, records...)
	}
	if best := pickBest(candidates, q.DurationMs, looseLengthTolerance); best != nil {
		return best, nil
	}
	return nil, ErrLyricsNotFound
}

func hasLyrics(r lrclibRecord) bool {
	return r.Instrumental || r.PlainLyrics != "" || r.SyncedLyrics != ""
}

func pickBest(records []lrclibRecord, durationMs int64, tolerance float64) *lrclibRecord {
	var (
		best      *lrclibRecord
		bestScore = math.MaxFloat64
	)
	for i := range records {
		r := &records[i]
		if !hasLyrics(*r) {
			continue
		}
		score := 0.0
		if durationMs > 0 {
			diff := math.Abs(r.Duration - float64(durationMs)/1000)
			if diff > tolerance {
				continue
			}
			score = diff
		}
		if r.SyncedLyrics == "" {
			score += 5
		}
		if score < bestScore {
			best, bestScore = r, score
		}
	}
	return best
}

func (c *LyricsClient) get(ctx context.Context, path string, params url.Values, out any) error {
	err := c.getOnce(ctx, path, params, out)
	if err == nil || errors.Is(err, ErrLyricsNotFound) || ctx.Err() != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return err
	case <-time.After(400 * time.Millisecond):
	}
	return c.getOnce(ctx, path, params, out)
}

func (c *LyricsClient) getOnce(ctx context.Context, path string, params url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path+"?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Chilly (https://chillybot.space)")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("lyrics request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrLyricsNotFound
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
		slog.Debug("lyrics provider rejected a query", slog.String("path", path), slog.Int("status", resp.StatusCode))
		return ErrLyricsNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("lyrics provider returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

var lrcLine = regexp.MustCompile(`^\[(\d{1,2}):(\d{2})(?:[.:](\d{1,3}))?\]\s?(.*)$`)

func ParseLRC(lrc string) []LyricLine {
	if strings.TrimSpace(lrc) == "" {
		return nil
	}
	var lines []LyricLine
	for _, raw := range strings.Split(lrc, "\n") {
		m := lrcLine.FindStringSubmatch(strings.TrimSpace(raw))
		if m == nil {
			continue
		}
		minutes, _ := strconv.Atoi(m[1])
		seconds, _ := strconv.Atoi(m[2])
		fraction := 0
		if m[3] != "" {
			fraction, _ = strconv.Atoi((m[3] + "00")[:3])
		}
		lines = append(lines, LyricLine{
			TimeMs: int64(minutes)*60_000 + int64(seconds)*1000 + int64(fraction),
			Text:   strings.TrimSpace(m[4]),
		})
	}
	return lines
}

func (b *Bot) LyricsForTrack(ctx context.Context, track lavalink.Track) (*Lyrics, error) {
	if b.Lyrics == nil {
		return nil, ErrLyricsNotFound
	}
	q := QueryFromTrack(track)
	var lib *Library
	if b.Searcher != nil {
		lib = b.Searcher.Library()
	}
	if lib != nil && track.Info.SourceName == LibrarySource {
		if lt, ok := lib.Get(track.Info.Identifier); ok {
			if stored := lt.StoredLyrics(); stored != nil {
				return stored, nil
			}
		}
	}
	if station := GetTrackMeta(track).Radio; station != "" && b.Radio != nil {
		np, ok := b.Radio.Station(station)
		if !ok || np.NowPlaying == nil {
			return nil, ErrLyricsNotFound
		}
		song := np.NowPlaying.Song
		if lib != nil {
			if lt, ok := lib.BySongID(song.ID); ok {
				if stored := lt.StoredLyrics(); stored != nil {
					return stored, nil
				}
			}
		}
		q = SongQuery{Title: cleanPart(song.Title), Artist: cleanPart(song.Artist), DurationMs: int64(np.NowPlaying.Duration) * 1000}
	}
	return b.Lyrics.Lookup(ctx, q)
}
