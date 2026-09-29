package azuracast

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	liveIdleTimeout = 90 * time.Second
	liveMinBackoff  = 5 * time.Second
	liveMaxBackoff  = 5 * time.Minute
)

type livePayload struct {
	NowPlaying *NowPlaying `json:"np"`
}

type liveMessage struct {
	Connect *struct {
		Subs map[string]struct {
			Publications []struct {
				Data livePayload `json:"data"`
			} `json:"publications"`
		} `json:"subs"`
	} `json:"connect"`
	Pub *struct {
		Data livePayload `json:"data"`
	} `json:"pub"`
}

func (s *Service) runLive(ctx context.Context) {
	backoff := liveMinBackoff
	failures := 0
	for {
		codes := s.Shortcodes()
		if len(codes) == 0 {
			if !sleepCtx(ctx, liveMinBackoff) {
				return
			}
			continue
		}

		started := time.Now()
		err := s.live(ctx, codes)
		if ctx.Err() != nil {
			return
		}
		if time.Since(started) > time.Minute {
			backoff, failures = liveMinBackoff, 0
		}
		if err != nil {
			failures++
			if failures == 1 || failures%10 == 0 {
				slog.Warn("azuracast live updates unavailable, falling back to polling", slog.Any("error", err), slog.Duration("retry_in", backoff))
			}
		}
		if !sleepCtx(ctx, backoff) {
			return
		}
		backoff = min(backoff*2, liveMaxBackoff)
	}
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func liveURL(base string, codes []string) (string, error) {
	subs := make(map[string]map[string]bool, len(codes))
	for _, code := range codes {
		subs["station:"+code] = map[string]bool{"recover": true}
	}
	raw, err := json.Marshal(map[string]any{"subs": subs})
	if err != nil {
		return "", err
	}
	return base + "/api/live/nowplaying/sse?cf_connect=" + url.QueryEscape(string(raw)), nil
}

func (s *Service) live(ctx context.Context, codes []string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	idle := time.AfterFunc(liveIdleTimeout, cancel)
	defer idle.Stop()

	target, err := liveURL(s.client.baseURL, codes)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", userAgent)
	resp, err := s.client.stream.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("live endpoint returned %d", resp.StatusCode)
	}
	slog.Info("connected to azuracast live updates", slog.Int("stations", len(codes)))

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		idle.Reset(liveIdleTimeout)
		line := scanner.Text()
		data, ok := strings.CutPrefix(line, "data:")
		if !ok {
			continue
		}
		for _, np := range parseLive([]byte(strings.TrimSpace(data))) {
			s.apply(np)
		}
		if !slices.Equal(codes, s.Shortcodes()) {
			return nil
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return errors.New("live stream closed")
}

func parseLive(data []byte) []NowPlaying {
	if len(data) == 0 {
		return nil
	}
	var msg liveMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		slog.Debug("ignoring unreadable azuracast live message", slog.Any("error", err))
		return nil
	}
	var out []NowPlaying
	if msg.Connect != nil {
		for _, sub := range msg.Connect.Subs {
			for _, p := range sub.Publications {
				if p.Data.NowPlaying != nil {
					out = append(out, *p.Data.NowPlaying)
				}
			}
		}
	}
	if msg.Pub != nil && msg.Pub.Data.NowPlaying != nil {
		out = append(out, *msg.Pub.Data.NowPlaying)
	}
	return out
}
