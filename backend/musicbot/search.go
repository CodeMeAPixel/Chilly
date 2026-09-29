package musicbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sync"
	"time"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
)

const (
	LibrarySource   = "azuracast"
	maxGroupTracks  = 200
	loadConcurrency = 6
)

var (
	ErrNoNode           = errors.New("no lavalink node available")
	ErrSelectionExpired = errors.New("that song is no longer in the library, please search again")
	ErrExternalSource   = errors.New("Chilly only plays songs from its own library. Search by song or artist name instead")

	urlPattern = regexp.MustCompile(`^https?://[-a-zA-Z0-9+&@#/%?=~_|!:,.;]*[-a-zA-Z0-9+&@#/%=~_|]?`)
)

func IsURL(s string) bool {
	return urlPattern.MatchString(s)
}

type Searcher struct {
	link    disgolink.Client
	library *Library
	signer  *MediaSigner
	cache   *MediaCache
}

func NewSearcher(link disgolink.Client, library *Library, signer *MediaSigner) *Searcher {
	return &Searcher{link: link, library: library, signer: signer}
}

func (s *Searcher) UseMediaCache(cache *MediaCache) {
	s.cache = cache
}

func (s *Searcher) Library() *Library {
	return s.library
}

func (s *Searcher) available() error {
	if s.library == nil || s.signer == nil || !s.library.Ready() {
		return ErrLibraryUnavailable
	}
	return nil
}

func (s *Searcher) Search(query string, limit int) []LibraryTrack {
	if s.library == nil {
		return nil
	}
	return s.library.Search(query, limit)
}

func (s *Searcher) Resolve(ctx context.Context, query string, kind string) (*lavalink.LoadResult, error) {
	if _, _, ok := ParseLibraryKey(query); ok {
		if err := s.available(); err != nil {
			return nil, err
		}
		track, found := s.library.Get(query)
		if !found {
			return nil, ErrSelectionExpired
		}
		return s.single(ctx, track)
	}
	if IsURL(query) {
		return nil, ErrExternalSource
	}
	if err := s.available(); err != nil {
		return nil, err
	}

	switch LibraryGroup(kind) {
	case GroupAlbum, GroupArtist, GroupPlaylist:
		name, tracks := s.library.Group(LibraryGroup(kind), query)
		if len(tracks) == 0 {
			return &lavalink.LoadResult{LoadType: lavalink.LoadTypeEmpty, Data: lavalink.Empty{}}, nil
		}
		if len(tracks) > maxGroupTracks {
			tracks = tracks[:maxGroupTracks]
		}
		loaded, err := s.LoadLibrary(ctx, tracks)
		if err != nil {
			return nil, err
		}
		return &lavalink.LoadResult{
			LoadType: lavalink.LoadTypePlaylist,
			Data:     lavalink.Playlist{Info: lavalink.PlaylistInfo{Name: name, SelectedTrack: -1}, Tracks: loaded},
		}, nil
	}

	matches := s.library.Search(query, 1)
	if len(matches) == 0 {
		return &lavalink.LoadResult{LoadType: lavalink.LoadTypeEmpty, Data: lavalink.Empty{}}, nil
	}
	return s.single(ctx, matches[0])
}

func (s *Searcher) single(ctx context.Context, track LibraryTrack) (*lavalink.LoadResult, error) {
	loaded, err := s.LoadLibrary(ctx, []LibraryTrack{track})
	if err != nil {
		return nil, err
	}
	return &lavalink.LoadResult{LoadType: lavalink.LoadTypeTrack, Data: loaded[0]}, nil
}

func (s *Searcher) LoadLibrary(ctx context.Context, tracks []LibraryTrack) ([]lavalink.Track, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	node := BestNode(s.link)
	if node == nil {
		return nil, ErrNoNode
	}

	loaded := make([]*lavalink.Track, len(tracks))
	errs := make([]error, len(tracks))
	sem := make(chan struct{}, loadConcurrency)
	var wg sync.WaitGroup
	for i, t := range tracks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			track, err := s.loadOne(ctx, node, t)
			if err != nil {
				errs[i] = err
				return
			}
			loaded[i] = &track
		}()
	}
	wg.Wait()

	out := make([]lavalink.Track, 0, len(tracks))
	for _, t := range loaded {
		if t != nil {
			out = append(out, *t)
		}
	}
	if len(out) == 0 {
		if err := errors.Join(errs...); err != nil {
			return nil, err
		}
		return nil, errors.New("nothing could be loaded")
	}
	return out, nil
}

func (s *Searcher) loadOne(ctx context.Context, node disgolink.Node, lt LibraryTrack) (lavalink.Track, error) {
	s.warm(ctx, lt)
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	result, err := node.LoadTracks(ctx, s.signer.URL(lt.Station, lt.ID))
	if err != nil {
		return lavalink.Track{}, err
	}
	track, ok := result.Data.(lavalink.Track)
	if !ok {
		if ex, isErr := result.Data.(lavalink.Exception); isErr {
			if cause := LoadFailureCause(ex); cause != "" {
				return lavalink.Track{}, fmt.Errorf("lavalink node %s could not load %q: %s (cause: %s)", node.Config().Name, lt.Title, ex.Message, Trim(cause, 500))
			}
			return lavalink.Track{}, fmt.Errorf("lavalink node %s could not load %q: %s", node.Config().Name, lt.Title, ex.Message)
		}
		return lavalink.Track{}, fmt.Errorf("unexpected load result %q for %q (is the lavalink http source enabled and MEDIA_BASE_URL reachable?)", result.LoadType, lt.Title)
	}
	return withLibraryInfo(track, lt), nil
}

func (s *Searcher) warm(ctx context.Context, lt LibraryTrack) {
	if s.cache == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if _, err := s.cache.Warm(ctx, lt.Station, lt.ID); err != nil {
		slog.Warn("failed to cache library song, lavalink will stream it from AzuraCast",
			slog.String("station", lt.Station), slog.Int("id", lt.ID), slog.Any("error", err))
	}
}

func withLibraryInfo(track lavalink.Track, lt LibraryTrack) lavalink.Track {
	track.Info.Title = lt.Title
	track.Info.Author = lt.Artist
	track.Info.Identifier = lt.Key()
	track.Info.SourceName = LibrarySource
	track.Info.URI = nil
	track.Info.ArtworkURL = nil
	if lt.ArtURL != "" {
		art := lt.ArtURL
		track.Info.ArtworkURL = &art
	}
	return track
}

func (s *Searcher) LoadURL(ctx context.Context, identifier string) (*lavalink.LoadResult, error) {
	node := BestNode(s.link)
	if node == nil {
		return nil, ErrNoNode
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return node.LoadTracks(ctx, identifier)
}

func (s *Searcher) LoadPlaylistTracks(ctx context.Context, rows []PlaylistTrack) ([]lavalink.Track, int, error) {
	if err := s.available(); err != nil {
		return nil, 0, err
	}
	missing := 0
	available := make([]LibraryTrack, 0, len(rows))
	for _, row := range rows {
		if row.LibraryKey == "" {
			missing++
			continue
		}
		track, ok := s.library.Get(row.LibraryKey)
		if !ok {
			missing++
			continue
		}
		available = append(available, track)
	}
	if len(available) > maxGroupTracks {
		available = available[:maxGroupTracks]
	}
	if len(available) == 0 {
		return nil, missing, nil
	}
	tracks, err := s.LoadLibrary(ctx, available)
	missing += len(available) - len(tracks)
	return tracks, missing, err
}
