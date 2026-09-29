package musicbot

import (
	"context"
	"errors"
	"log/slog"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CodeMeAPixel/Chilly/azuracast"
)

const libraryKeyPrefix = "lib:"

var ErrLibraryUnavailable = errors.New("the music library isn't available right now")

type LibraryTrack struct {
	Station   string   `json:"station"`
	ID        int      `json:"id"`
	UniqueID  string   `json:"unique_id"`
	SongID    string   `json:"song_id"`
	Title     string   `json:"title"`
	Artist    string   `json:"artist"`
	Album     string   `json:"album"`
	Genre     string   `json:"genre"`
	ArtURL    string   `json:"art"`
	LengthMs  int64    `json:"length_ms"`
	Playlists []string `json:"playlists"`
	Lyrics    string   `json:"-"`
	Path      string   `json:"-"`

	index string
}

func (t LibraryTrack) Key() string {
	return libraryKeyPrefix + t.Station + ":" + strconv.Itoa(t.ID)
}

func ParseLibraryKey(key string) (station string, id int, ok bool) {
	rest, found := strings.CutPrefix(key, libraryKeyPrefix)
	if !found {
		return "", 0, false
	}
	station, rawID, found := strings.Cut(rest, ":")
	if !found || station == "" {
		return "", 0, false
	}
	id, err := strconv.Atoi(rawID)
	if err != nil || id <= 0 {
		return "", 0, false
	}
	return station, id, true
}

type LibraryFetcher interface {
	Files(ctx context.Context, station string) ([]azuracast.MediaFile, error)
}

type Library struct {
	fetcher  LibraryFetcher
	stations func() []string

	mu       sync.RWMutex
	tracks   []LibraryTrack
	byKey    map[string]int
	lastSync time.Time
	lastErr  error
	onSync   []func()
}

func NewLibrary(fetcher LibraryFetcher, stations func() []string) *Library {
	return &Library{fetcher: fetcher, stations: stations, byKey: make(map[string]int)}
}

func (l *Library) Run(ctx context.Context, interval time.Duration) {
	retry := 15 * time.Second
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if err := l.Sync(ctx); err != nil {
				slog.Warn("failed to sync music library", slog.Any("error", err))
				timer.Reset(retry)
				continue
			}
			timer.Reset(interval)
		}
	}
}

func (l *Library) Sync(ctx context.Context) error {
	stations := l.stations()
	if len(stations) == 0 {
		return errors.New("no stations known yet")
	}

	var (
		tracks   []LibraryTrack
		bySong   = make(map[string]int)
		failures []error
	)
	for _, station := range stations {
		files, err := l.fetcher.Files(ctx, station)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		for _, f := range files {
			playlists := make([]string, 0, len(f.Playlists))
			for _, p := range f.Playlists {
				playlists = append(playlists, p.Name)
			}
			if i, seen := bySong[f.SongID]; seen && f.SongID != "" {
				for _, p := range playlists {
					if !slices.Contains(tracks[i].Playlists, p) {
						tracks[i].Playlists = append(tracks[i].Playlists, p)
					}
				}
				continue
			}
			t := LibraryTrack{
				Station:   station,
				ID:        f.ID,
				UniqueID:  f.UniqueID,
				SongID:    f.SongID,
				Title:     strings.TrimSpace(f.Title),
				Artist:    strings.TrimSpace(f.Artist),
				Album:     strings.TrimSpace(f.Album),
				Genre:     strings.TrimSpace(f.Genre),
				ArtURL:    f.Art,
				LengthMs:  int64(f.Length) * 1000,
				Playlists: playlists,
				Path:      f.Path,
			}
			if t.Title == "" {
				t.Title = strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path))
			}
			if f.Lyrics != nil {
				t.Lyrics = *f.Lyrics
			}
			t.index = normalizeSearch(t.Artist + " " + t.Title + " " + t.Album)
			bySong[f.SongID] = len(tracks)
			tracks = append(tracks, t)
		}
	}
	if len(failures) == len(stations) {
		l.mu.Lock()
		l.lastErr = failures[0]
		l.mu.Unlock()
		return failures[0]
	}

	byKey := make(map[string]int, len(tracks))
	for i, t := range tracks {
		byKey[t.Key()] = i
	}
	l.mu.Lock()
	l.tracks = tracks
	l.byKey = byKey
	l.lastSync = time.Now()
	l.lastErr = errors.Join(failures...)
	hooks := slices.Clone(l.onSync)
	l.mu.Unlock()
	for _, fn := range hooks {
		fn()
	}
	slog.Info("music library synced", slog.Int("tracks", len(tracks)), slog.Int("stations", len(stations)))
	return nil
}

func (l *Library) Status() (count int, lastSync time.Time, err error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.tracks), l.lastSync, l.lastErr
}

func (l *Library) Ready() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return !l.lastSync.IsZero()
}

func (l *Library) All() []LibraryTrack {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return slices.Clone(l.tracks)
}

func (l *Library) Get(key string) (LibraryTrack, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	i, ok := l.byKey[key]
	if !ok {
		return LibraryTrack{}, false
	}
	return l.tracks[i], true
}

func (l *Library) BySongID(songID string) (LibraryTrack, bool) {
	if songID == "" {
		return LibraryTrack{}, false
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, t := range l.tracks {
		if t.SongID == songID {
			return t, true
		}
	}
	return LibraryTrack{}, false
}

func (t LibraryTrack) StoredLyrics() *Lyrics {
	text := strings.TrimSpace(t.Lyrics)
	if text == "" {
		return nil
	}
	lyrics := &Lyrics{Title: t.Title, Artist: t.Artist, Album: t.Album, DurationMs: t.LengthMs, Source: "Chilly Library"}
	if synced := ParseLRC(text); len(synced) > 0 {
		lyrics.Synced = synced
		lines := make([]string, len(synced))
		for i, line := range synced {
			lines[i] = line.Text
		}
		lyrics.Plain = strings.Join(lines, "\n")
	} else {
		lyrics.Plain = text
	}
	return lyrics
}

func normalizeSearch(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

func (l *Library) Search(query string, limit int) []LibraryTrack {
	q := normalizeSearch(query)
	tokens := strings.Fields(q)
	if len(tokens) == 0 {
		return nil
	}

	type scored struct {
		track LibraryTrack
		score int
	}
	l.mu.RLock()
	matches := make([]scored, 0, 32)
	for _, t := range l.tracks {
		if !containsAll(t.index, tokens) {
			continue
		}
		matches = append(matches, scored{track: t, score: matchScore(t, q)})
	}
	l.mu.RUnlock()

	slices.SortStableFunc(matches, func(a, b scored) int {
		if a.score != b.score {
			return b.score - a.score
		}
		if c := strings.Compare(strings.ToLower(a.track.Artist), strings.ToLower(b.track.Artist)); c != 0 {
			return c
		}
		return strings.Compare(strings.ToLower(a.track.Title), strings.ToLower(b.track.Title))
	})
	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}
	out := make([]LibraryTrack, len(matches))
	for i, m := range matches {
		out[i] = m.track
	}
	return out
}

func containsAll(haystack string, tokens []string) bool {
	for _, t := range tokens {
		if !strings.Contains(haystack, t) {
			return false
		}
	}
	return true
}

func matchScore(t LibraryTrack, q string) int {
	title := normalizeSearch(t.Title)
	artist := normalizeSearch(t.Artist)
	score := 0
	switch {
	case title == q:
		score += 100
	case strings.HasPrefix(title, q):
		score += 60
	case strings.Contains(title, q):
		score += 40
	}
	switch {
	case artist == q:
		score += 50
	case artist != "" && strings.Contains(artist, q):
		score += 20
	}
	if title != "" && title != q && strings.Contains(q, title) {
		score += 30
	}
	if artist != "" && artist != q && strings.Contains(q, artist) {
		score += 15
	}
	return score
}

type LibraryGroup string

const (
	GroupAlbum    LibraryGroup = "album"
	GroupArtist   LibraryGroup = "artist"
	GroupPlaylist LibraryGroup = "playlist"
)

func (l *Library) Group(kind LibraryGroup, query string) (string, []LibraryTrack) {
	q := normalizeSearch(query)
	if q == "" {
		return "", nil
	}

	l.mu.RLock()
	groups := make(map[string][]LibraryTrack)
	names := make(map[string]string)
	for _, t := range l.tracks {
		for _, name := range groupNames(kind, t) {
			key := normalizeSearch(name)
			if key == "" {
				continue
			}
			groups[key] = append(groups[key], t)
			if _, ok := names[key]; !ok {
				names[key] = name
			}
		}
	}
	l.mu.RUnlock()

	best, bestScore := "", 0
	tokens := strings.Fields(q)
	for key := range groups {
		score := 0
		switch {
		case key == q:
			score = 4
		case strings.HasPrefix(key, q):
			score = 3
		case strings.Contains(key, q):
			score = 2
		case containsAll(key, tokens):
			score = 1
		}
		if score > bestScore || (score == bestScore && score > 0 && key < best) {
			best, bestScore = key, score
		}
	}
	if bestScore == 0 {
		return "", nil
	}
	tracks := groups[best]
	slices.SortStableFunc(tracks, func(a, b LibraryTrack) int {
		if kind == GroupArtist {
			if c := strings.Compare(strings.ToLower(a.Album), strings.ToLower(b.Album)); c != 0 {
				return c
			}
		}
		return strings.Compare(a.Path, b.Path)
	})
	return names[best], tracks
}

func groupNames(kind LibraryGroup, t LibraryTrack) []string {
	switch kind {
	case GroupAlbum:
		return []string{t.Album}
	case GroupArtist:
		return []string{t.Artist}
	case GroupPlaylist:
		return t.Playlists
	default:
		return nil
	}
}

func (l *Library) OnSync(fn func()) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onSync = append(l.onSync, fn)
}

func (l *Library) Match(q SongQuery) (LibraryTrack, bool) {
	title := normalizeSearch(cleanPart(q.Title))
	artist := normalizeSearch(cleanPart(q.Artist))
	if title == "" {
		return LibraryTrack{}, false
	}

	l.mu.RLock()
	defer l.mu.RUnlock()
	var candidates []LibraryTrack
	for _, t := range l.tracks {
		if normalizeSearch(cleanPart(t.Title)) != title {
			continue
		}
		candidates = append(candidates, t)
		libArtist := normalizeSearch(cleanPart(t.Artist))
		if artist != "" && libArtist != "" && (strings.Contains(libArtist, artist) || strings.Contains(artist, libArtist)) {
			return t, true
		}
	}
	if len(candidates) == 1 && q.DurationMs > 0 && candidates[0].LengthMs > 0 {
		diff := candidates[0].LengthMs - q.DurationMs
		if diff < 0 {
			diff = -diff
		}
		if diff <= 15_000 {
			return candidates[0], true
		}
	}
	return LibraryTrack{}, false
}

type LibraryPlaylist struct {
	Name       string   `json:"name"`
	TrackCount int      `json:"track_count"`
	Art        string   `json:"art,omitempty"`
	Artists    []string `json:"artists"`
}

func (l *Library) Playlists() []LibraryPlaylist {
	l.mu.RLock()
	byName := make(map[string]*LibraryPlaylist)
	order := make([]string, 0)
	for _, t := range l.tracks {
		for _, name := range t.Playlists {
			key := strings.ToLower(name)
			p, ok := byName[key]
			if !ok {
				p = &LibraryPlaylist{Name: name, Artists: []string{}}
				byName[key] = p
				order = append(order, key)
			}
			p.TrackCount++
			if p.Art == "" && t.ArtURL != "" {
				p.Art = t.ArtURL
			}
			if t.Artist != "" && len(p.Artists) < 5 && !slices.Contains(p.Artists, t.Artist) {
				p.Artists = append(p.Artists, t.Artist)
			}
		}
	}
	l.mu.RUnlock()

	out := make([]LibraryPlaylist, 0, len(order))
	for _, key := range order {
		out = append(out, *byName[key])
	}
	slices.SortFunc(out, func(a, b LibraryPlaylist) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return out
}

func (l *Library) PlaylistTracks(name string) []LibraryTrack {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]LibraryTrack, 0)
	for _, t := range l.tracks {
		for _, p := range t.Playlists {
			if strings.EqualFold(p, name) {
				out = append(out, t)
				break
			}
		}
	}
	slices.SortStableFunc(out, func(a, b LibraryTrack) int { return strings.Compare(a.Path, b.Path) })
	return out
}

type LibraryQuery struct {
	Query    string
	Artist   string
	Album    string
	Playlist string
	Sort     string
	Offset   int
	Limit    int
}

func (l *Library) Browse(q LibraryQuery) ([]LibraryTrack, int) {
	var tracks []LibraryTrack
	if strings.TrimSpace(q.Query) != "" {
		tracks = l.Search(q.Query, 0)
	} else {
		tracks = l.All()
		key := func(t LibraryTrack) string { return strings.ToLower(t.Artist + "\x00" + t.Album + "\x00" + t.Path) }
		switch q.Sort {
		case "title":
			key = func(t LibraryTrack) string { return strings.ToLower(t.Title + "\x00" + t.Artist) }
		case "album":
			key = func(t LibraryTrack) string { return strings.ToLower(t.Album + "\x00" + t.Path) }
		}
		slices.SortStableFunc(tracks, func(a, b LibraryTrack) int { return strings.Compare(key(a), key(b)) })
	}

	filtered := tracks[:0]
	for _, t := range tracks {
		if q.Artist != "" && !strings.EqualFold(t.Artist, q.Artist) {
			continue
		}
		if q.Album != "" && !strings.EqualFold(t.Album, q.Album) {
			continue
		}
		if q.Playlist != "" && !slices.ContainsFunc(t.Playlists, func(p string) bool { return strings.EqualFold(p, q.Playlist) }) {
			continue
		}
		filtered = append(filtered, t)
	}

	total := len(filtered)
	start := min(max(q.Offset, 0), total)
	end := total
	if q.Limit > 0 {
		end = min(start+q.Limit, total)
	}
	return filtered[start:end], total
}

type LibraryGroupSummary struct {
	Name       string `json:"name"`
	Artist     string `json:"artist,omitempty"`
	TrackCount int    `json:"track_count"`
	Art        string `json:"art,omitempty"`
}

func (l *Library) Groups(kind LibraryGroup) []LibraryGroupSummary {
	l.mu.RLock()
	byKey := make(map[string]*LibraryGroupSummary)
	for _, t := range l.tracks {
		name := t.Artist
		if kind == GroupAlbum {
			name = t.Album
		}
		if strings.TrimSpace(name) == "" {
			continue
		}
		key := strings.ToLower(name)
		if kind == GroupAlbum {
			key += "\x00" + strings.ToLower(t.Artist)
		}
		g, ok := byKey[key]
		if !ok {
			g = &LibraryGroupSummary{Name: name}
			if kind == GroupAlbum {
				g.Artist = t.Artist
			}
			byKey[key] = g
		}
		g.TrackCount++
		if g.Art == "" && t.ArtURL != "" {
			g.Art = t.ArtURL
		}
	}
	l.mu.RUnlock()

	out := make([]LibraryGroupSummary, 0, len(byKey))
	for _, g := range byKey {
		out = append(out, *g)
	}
	slices.SortFunc(out, func(a, b LibraryGroupSummary) int {
		if c := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); c != 0 {
			return c
		}
		return strings.Compare(strings.ToLower(a.Artist), strings.ToLower(b.Artist))
	})
	return out
}
