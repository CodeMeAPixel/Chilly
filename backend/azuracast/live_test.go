package azuracast

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func npJSON(code, songID, title string, shID int) string {
	return fmt.Sprintf(`{"station":{"shortcode":%q,"name":%q},"is_online":true,"now_playing":{"sh_id":%d,"duration":200.5,"song":{"id":%q,"title":%q}}}`,
		code, code, shID, songID, title)
}

func TestParseLive(t *testing.T) {
	connect := `{"connect":{"subs":{"station:a":{"publications":[{"data":{"np":` + npJSON("a", "s1", "One", 1) + `}}]},"station:b":{}}}}`
	if got := parseLive([]byte(connect)); len(got) != 1 || got[0].NowPlaying.Song.Title != "One" {
		t.Fatalf("connect message: %+v", got)
	}
	pub := `{"channel":"station:a","pub":{"data":{"np":` + npJSON("a", "s2", "Two", 2) + `,"current_time":1}}}`
	if got := parseLive([]byte(pub)); len(got) != 1 || got[0].NowPlaying.Duration != 201 {
		t.Fatalf("pub message: %+v", got)
	}
	for _, junk := range []string{"", "{}", "not json"} {
		if got := parseLive([]byte(junk)); len(got) != 0 {
			t.Errorf("%q should produce nothing, got %+v", junk, got)
		}
	}
}

func TestLiveAppliesUpdatesAndNotifies(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("cf_connect")
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		fmt.Fprintf(w, "data: %s\n\n", `{"connect":{"subs":{"station:a":{"publications":[{"data":{"np":`+npJSON("a", "s1", "One", 1)+`}}]}}}}`)
		flusher.Flush()
		fmt.Fprint(w, "data: {}\n\n")
		fmt.Fprintf(w, "data: %s\n\n", `{"channel":"station:a","pub":{"data":{"np":`+npJSON("a", "s2", "Two", 2)+`}}}`)
		flusher.Flush()
	}))
	defer srv.Close()

	s := NewService(Config{URL: srv.URL})
	s.stations["a"] = NowPlaying{Station: Station{Shortcode: "a"}}
	s.order = []string{"a"}
	changes := make(chan string, 4)
	s.OnSongChange(func(np NowPlaying) { changes <- np.NowPlaying.Song.Title })
	updates, unsubscribe := s.Subscribe()
	defer unsubscribe()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.live(ctx, []string{"a"}); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("expected the stream to end, got %v", err)
	}

	var subs struct {
		Subs map[string]map[string]bool `json:"subs"`
	}
	if err := json.Unmarshal([]byte(gotQuery), &subs); err != nil || !subs.Subs["station:a"]["recover"] {
		t.Errorf("subscription should ask for recovery, got %q", gotQuery)
	}
	np, _ := s.Station("a")
	if np.NowPlaying == nil || np.NowPlaying.Song.Title != "Two" {
		t.Fatalf("latest update not applied: %+v", np.NowPlaying)
	}
	if len(changes) != 2 {
		t.Errorf("expected 2 song changes, got %d", len(changes))
	}
	select {
	case <-updates:
	default:
		t.Error("subscribers should be notified")
	}
	if healthy, _ := s.Healthy(); !healthy {
		t.Error("live updates should mark the service healthy")
	}
}

func TestLiveURLEscapesSubscriptions(t *testing.T) {
	raw, err := liveURL("https://radio.example", []string{"247_hip-hop"})
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(raw)
	if u.Path != "/api/live/nowplaying/sse" || !strings.Contains(u.Query().Get("cf_connect"), `"station:247_hip-hop":{"recover":true}`) {
		t.Errorf("unexpected url %s", raw)
	}
}

func TestApplyIgnoresFilteredStations(t *testing.T) {
	s := NewService(Config{URL: "http://x", Stations: []string{"a"}})
	s.apply(NowPlaying{Station: Station{Shortcode: "b"}})
	if len(s.Stations()) != 0 {
		t.Error("stations outside the allow-list must be ignored")
	}
	s.apply(NowPlaying{Station: Station{Shortcode: "a"}})
	if len(s.Stations()) != 1 {
		t.Error("allowed stations should be added")
	}
}
