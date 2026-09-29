package azuracast

import (
	"context"
	"log/slog"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	URL           string
	APIKey        string
	Stations      []string
	PollInterval  time.Duration
	StreamBaseURL string
}

type SongChangeFunc func(np NowPlaying)

type Service struct {
	client *Client
	cfg    Config

	mu        sync.RWMutex
	stations  map[string]NowPlaying
	order     []string
	lastErr   error
	lastPoll  time.Time
	listeners []SongChangeFunc
}

func NewService(cfg Config) *Service {
	if cfg.PollInterval < 5*time.Second {
		cfg.PollInterval = 5 * time.Second
	}
	return &Service{
		client:   NewClient(cfg.URL, cfg.APIKey),
		cfg:      cfg,
		stations: make(map[string]NowPlaying),
	}
}

func (s *Service) OnSongChange(fn SongChangeFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, fn)
}

func (s *Service) Run(ctx context.Context) {
	s.poll(ctx)
	ticker := time.NewTicker(s.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.poll(ctx)
		}
	}
}

func (s *Service) poll(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	all, err := s.client.NowPlayingAll(ctx)
	s.mu.Lock()
	s.lastPoll = time.Now()
	s.lastErr = err
	if err != nil {
		s.mu.Unlock()
		slog.Warn("failed to poll azuracast", slog.Any("error", err))
		return
	}

	var changed []NowPlaying
	order := make([]string, 0, len(all))
	next := make(map[string]NowPlaying, len(all))
	for _, np := range all {
		code := np.Station.Shortcode
		if len(s.cfg.Stations) > 0 && !slices.Contains(s.cfg.Stations, code) {
			continue
		}
		order = append(order, code)
		next[code] = np
		if prev, ok := s.stations[code]; ok && songID(prev) != songID(np) {
			changed = append(changed, np)
		}
	}
	s.stations = next
	s.order = order
	listeners := slices.Clone(s.listeners)
	s.mu.Unlock()

	for _, np := range changed {
		for _, fn := range listeners {
			fn(np)
		}
	}
}

func songID(np NowPlaying) string {
	if np.NowPlaying == nil {
		return ""
	}
	return strconv.Itoa(np.NowPlaying.ShID) + ":" + np.NowPlaying.Song.ID
}

func (s *Service) Stations() []NowPlaying {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]NowPlaying, 0, len(s.order))
	for _, code := range s.order {
		out = append(out, s.stations[code])
	}
	return out
}

func (s *Service) Station(key string) (NowPlaying, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if np, ok := s.stations[key]; ok {
		return np, true
	}
	for _, np := range s.stations {
		if strconv.Itoa(np.Station.ID) == key || strings.EqualFold(np.Station.Name, key) {
			return np, true
		}
	}
	return NowPlaying{}, false
}

func (s *Service) Healthy() (bool, time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastErr == nil && !s.lastPoll.IsZero(), s.lastPoll
}

func (s *Service) StreamURL(np NowPlaying) string {
	stream := np.Station.ListenURL
	for _, m := range np.Station.Mounts {
		if m.IsDefault && m.URL != "" {
			stream = m.URL
			break
		}
	}
	if stream == "" && len(np.Station.Mounts) > 0 {
		stream = np.Station.Mounts[0].URL
	}
	if s.cfg.StreamBaseURL == "" || stream == "" {
		return stream
	}
	base, err := url.Parse(s.cfg.StreamBaseURL)
	if err != nil {
		return stream
	}
	u, err := url.Parse(stream)
	if err != nil {
		return stream
	}
	u.Scheme = base.Scheme
	u.Host = base.Host
	return u.String()
}

func (s *Service) Client() *Client {
	return s.client
}

func (s *Service) Shortcodes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string(nil), s.order...)
}
