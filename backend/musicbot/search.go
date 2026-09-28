package musicbot

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
)

var (
	ErrNoNode           = errors.New("no lavalink node available")
	ErrSelectionExpired = errors.New("that search result expired, please search again")

	urlPattern = regexp.MustCompile(`^https?://[-a-zA-Z0-9+&@#/%?=~_|!:,.;]*[-a-zA-Z0-9+&@#/%=~_|]?`)
)

const (
	maxChoiceValue  = 100
	selectionPrefix = "sel:"
)

var sourceAliases = map[string]lavalink.SearchType{
	"youtube":      "ytsearch",
	"youtubemusic": "ytmsearch",
	"ytmusic":      "ytmsearch",
	"soundcloud":   "scsearch",
	"spotify":      "spsearch",
	"deezer":       "dzsearch",
	"applemusic":   "amsearch",
}

func IsURL(s string) bool {
	return urlPattern.MatchString(s)
}

type SourceUnavailableError struct {
	Source string
}

var sourceNames = map[string]string{
	"youtube":    "YouTube",
	"soundcloud": "SoundCloud",
	"spotify":    "Spotify",
	"deezer":     "Deezer",
	"applemusic": "Apple Music",
}

func (e *SourceUnavailableError) Error() string {
	name := sourceNames[e.Source]
	if name == "" {
		name = e.Source
	}
	return name + " isn't available on the music server right now"
}

var urlSources = map[string]string{
	"youtube.com":        "youtube",
	"www.youtube.com":    "youtube",
	"m.youtube.com":      "youtube",
	"music.youtube.com":  "youtube",
	"youtu.be":           "youtube",
	"open.spotify.com":   "spotify",
	"spotify.link":       "spotify",
	"www.deezer.com":     "deezer",
	"deezer.com":         "deezer",
	"deezer.page.link":   "deezer",
	"link.deezer.com":    "deezer",
	"music.apple.com":    "applemusic",
	"soundcloud.com":     "soundcloud",
	"m.soundcloud.com":   "soundcloud",
	"on.soundcloud.com":  "soundcloud",
	"www.soundcloud.com": "soundcloud",
}

func sourceOfURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return urlSources[strings.ToLower(u.Hostname())]
}

type Searcher struct {
	link       disgolink.Client
	mu         sync.RWMutex
	available  map[string]bool
	providers  []lavalink.SearchType
	selections *ttlCache[lavalink.Track]
	results    *ttlCache[[]lavalink.Track]
}

func NewSearcher(link disgolink.Client, providers []string) *Searcher {
	s := &Searcher{
		link:       link,
		selections: newTTLCache[lavalink.Track](30*time.Minute, 10000),
		results:    newTTLCache[[]lavalink.Track](5*time.Minute, 2000),
	}
	for _, p := range providers {
		p = strings.TrimSpace(strings.TrimSuffix(p, ":"))
		if alias, ok := sourceAliases[strings.ToLower(p)]; ok {
			s.providers = append(s.providers, alias)
		} else if p != "" {
			s.providers = append(s.providers, lavalink.SearchType(p))
		}
	}
	if len(s.providers) == 0 {
		s.providers = []lavalink.SearchType{"scsearch", "spsearch", "dzsearch"}
	}
	return s
}

func (s *Searcher) node() (disgolink.Node, error) {
	node := BestNode(s.link)
	if node == nil {
		return nil, ErrNoNode
	}
	return node, nil
}

func (s *Searcher) sourceAvailable(source string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.available == nil || source == "" || s.available[source]
}

func (s *Searcher) providersFor(source string) ([]lavalink.SearchType, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return s.providers, nil
	}
	provider, ok := sourceAliases[source]
	if !ok {
		provider = lavalink.SearchType(source)
	}
	if name, known := providerSources[provider]; known && !s.sourceAvailable(name) {
		return nil, &SourceUnavailableError{Source: name}
	}
	return []lavalink.SearchType{provider}, nil
}

func (s *Searcher) load(ctx context.Context, identifier string) (*lavalink.LoadResult, error) {
	node, err := s.node()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return node.LoadTracks(ctx, identifier)
}

func (s *Searcher) Resolve(ctx context.Context, query string, source string) (*lavalink.LoadResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return &lavalink.LoadResult{LoadType: lavalink.LoadTypeEmpty, Data: lavalink.Empty{}}, nil
	}

	if track, ok := s.selections.Get(query); ok {
		return &lavalink.LoadResult{LoadType: lavalink.LoadTypeTrack, Data: track}, nil
	}
	if strings.HasPrefix(query, selectionPrefix) {
		return nil, ErrSelectionExpired
	}

	if IsURL(query) {
		if source := sourceOfURL(query); !s.sourceAvailable(source) {
			return nil, &SourceUnavailableError{Source: source}
		}
		result, err := s.load(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to load %q: %w", query, err)
		}
		if ex, ok := result.Data.(lavalink.Exception); ok {
			return nil, fmt.Errorf("failed to load %q: %s", query, ex.Message)
		}
		return result, nil
	}

	providers, err := s.providersFor(source)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for _, provider := range providers {
		result, err := s.load(ctx, provider.Apply(query))
		if err != nil {
			lastErr = err
			slog.Debug("search provider failed", slog.String("provider", string(provider)), slog.Any("error", err))
			continue
		}
		switch data := result.Data.(type) {
		case lavalink.Track, lavalink.Playlist:
			return result, nil
		case lavalink.Search:
			if len(data) > 0 {
				return result, nil
			}
		case lavalink.Exception:
			lastErr = errors.New(data.Message)
			slog.Debug("search provider exception", slog.String("provider", string(provider)), slog.String("message", data.Message))
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("no provider returned results for %q: %w", query, lastErr)
	}
	return &lavalink.LoadResult{LoadType: lavalink.LoadTypeEmpty, Data: lavalink.Empty{}}, nil
}

func (s *Searcher) SearchTracks(ctx context.Context, query string, source string, limit int) ([]lavalink.Track, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	cacheKey := strings.ToLower(source) + "|" + strings.ToLower(query)
	if tracks, ok := s.results.Get(cacheKey); ok {
		return capTracks(tracks, limit), nil
	}

	if IsURL(query) {
		result, err := s.Resolve(ctx, query, "")
		if err != nil {
			return nil, err
		}
		tracks := tracksFromResult(result)
		s.results.Set(cacheKey, tracks)
		return capTracks(tracks, limit), nil
	}

	providers, err := s.providersFor(source)
	if err != nil {
		return nil, err
	}
	type res struct {
		idx    int
		tracks []lavalink.Track
		err    error
	}
	ch := make(chan res, len(providers))
	for i, provider := range providers {
		go func(i int, provider lavalink.SearchType) {
			result, err := s.load(ctx, provider.Apply(query))
			if err != nil {
				ch <- res{idx: i, err: err}
				return
			}
			if ex, ok := result.Data.(lavalink.Exception); ok {
				ch <- res{idx: i, err: errors.New(ex.Message)}
				return
			}
			ch <- res{idx: i, tracks: tracksFromResult(result)}
		}(i, provider)
	}

	done := make([]*res, len(providers))
	var lastErr error
	for received := 0; received < len(providers); {
		select {
		case r := <-ch:
			received++
			done[r.idx] = &r
			if r.err != nil {
				lastErr = r.err
			}
		case <-ctx.Done():
			received = len(providers)
		}

		for _, r := range done {
			if r == nil {
				break
			}
			if len(r.tracks) > 0 {
				s.results.Set(cacheKey, r.tracks)
				return capTracks(r.tracks, limit), nil
			}
		}
	}

	for _, r := range done {
		if r != nil && len(r.tracks) > 0 {
			return capTracks(r.tracks, limit), nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return nil, nil
}

func (s *Searcher) Remember(track lavalink.Track) string {
	value := TrackURL(track)
	if value == "" || len(value) > maxChoiceValue {
		sum := sha1.Sum([]byte(track.Encoded))
		value = selectionPrefix + hex.EncodeToString(sum[:12])
	}
	s.selections.Set(value, track)
	return value
}

func tracksFromResult(result *lavalink.LoadResult) []lavalink.Track {
	switch data := result.Data.(type) {
	case lavalink.Track:
		return []lavalink.Track{data}
	case lavalink.Search:
		return data
	case lavalink.Playlist:
		return data.Tracks
	}
	return nil
}

func capTracks(tracks []lavalink.Track, limit int) []lavalink.Track {
	if limit > 0 && len(tracks) > limit {
		return tracks[:limit]
	}
	return tracks
}

var sourcePrefixes = map[string]lavalink.SearchType{
	"youtube":    "ytsearch",
	"soundcloud": "scsearch",
	"spotify":    "spsearch",
	"deezer":     "dzsearch",
	"applemusic": "amsearch",
}

const alternativeLengthTolerance = 15 * lavalink.Second

func (s *Searcher) FindAlternative(ctx context.Context, failed lavalink.Track) (lavalink.Track, bool) {
	query := strings.TrimSpace(failed.Info.Author + " " + failed.Info.Title)
	if query == "" {
		return lavalink.Track{}, false
	}
	failedPrefix := sourcePrefixes[failed.Info.SourceName]

	s.mu.RLock()
	providers := s.providers
	s.mu.RUnlock()
	for _, provider := range providers {
		if provider == failedPrefix || (failedPrefix == "ytsearch" && provider == "ytmsearch") {
			continue
		}
		result, err := s.load(ctx, provider.Apply(query))
		if err != nil {
			if ctx.Err() != nil {
				return lavalink.Track{}, false
			}
			continue
		}
		for _, candidate := range capTracks(tracksFromResult(result), 5) {
			if candidate.Encoded == failed.Encoded || candidate.Info.SourceName == failed.Info.SourceName {
				continue
			}
			if failed.Info.Length > 0 && !failed.Info.IsStream {
				diff := candidate.Info.Length - failed.Info.Length
				if diff < -alternativeLengthTolerance || diff > alternativeLengthTolerance {
					continue
				}
			}
			return candidate, true
		}
	}
	return lavalink.Track{}, false
}

var providerSources = map[lavalink.SearchType]string{
	"ytsearch":  "youtube",
	"ytmsearch": "youtube",
	"scsearch":  "soundcloud",
	"spsearch":  "spotify",
	"dzsearch":  "deezer",
	"amsearch":  "applemusic",
}

func (s *Searcher) FilterSupported(sourceManagers []string) (kept, dropped []lavalink.SearchType) {
	available := make(map[string]bool, len(sourceManagers))
	for _, name := range sourceManagers {
		available[strings.ToLower(name)] = true
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.available = available
	for _, provider := range s.providers {
		if source, known := providerSources[provider]; known && !available[source] {
			dropped = append(dropped, provider)
			continue
		}
		kept = append(kept, provider)
	}
	if len(kept) == 0 {
		return s.providers, nil
	}
	s.providers = kept
	return kept, dropped
}
