package azuracast

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateLyrics(t *testing.T) {
	var method, path, key, body string
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, key = r.Method, r.URL.Path, r.Header.Get("X-API-Key")
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		w.WriteHeader(status)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	if err := c.UpdateLyrics(context.Background(), "chill", 7, "[00:01.00]hi"); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPut || path != "/api/station/chill/file/7" || key != "k" || body != `{"lyrics":"[00:01.00]hi"}` {
		t.Errorf("unexpected request %s %s key=%q body=%s", method, path, key, body)
	}

	status = http.StatusForbidden
	var statusErr *StatusError
	if err := c.UpdateLyrics(context.Background(), "chill", 7, "x"); !errors.As(err, &statusErr) || statusErr.Status != http.StatusForbidden {
		t.Errorf("expected a 403 StatusError, got %v", err)
	}
}

func TestNowPlayingAcceptsFractionalDurations(t *testing.T) {
	payload := `{
		"now_playing": {"duration": 212.4, "elapsed": 30, "remaining": 182.6, "song": {"title": "a"}},
		"playing_next": {"cued_at": 1790000000, "duration": 5379.7920206649, "song": {"title": "b"}},
		"song_history": [{"played_at": 1789999000, "duration": null, "song": {"title": "c"}}]
	}`

	var np NowPlaying
	if err := json.Unmarshal([]byte(payload), &np); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if np.NowPlaying.Duration != 212 || np.NowPlaying.Elapsed != 30 || np.NowPlaying.Remaining != 183 {
		t.Errorf("now playing = %d/%d/%d, want 212/30/183", np.NowPlaying.Duration, np.NowPlaying.Elapsed, np.NowPlaying.Remaining)
	}
	if np.PlayingNext.Duration != 5380 {
		t.Errorf("playing next duration = %d, want 5380", np.PlayingNext.Duration)
	}
	if np.SongHistory[0].Duration != 0 {
		t.Errorf("history duration = %d, want 0", np.SongHistory[0].Duration)
	}
}

func TestSecondsRejectsNonNumbers(t *testing.T) {
	var s Seconds
	if err := json.Unmarshal([]byte(`"12"`), &s); err == nil {
		t.Error("expected an error for a string value")
	}
}

func TestSubmitRequest(t *testing.T) {
	var path, agent string
	fail := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, agent = r.URL.Path, r.Header.Get("User-Agent")
		if fail {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"code":500,"message":"This song or artist has been played too recently.","success":false}`)
			return
		}
		_, _ = io.WriteString(w, `{"success":true}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	if err := c.SubmitRequest(context.Background(), "chill", "abc123"); err != nil {
		t.Fatal(err)
	}
	if path != "/api/station/chill/request/abc123" {
		t.Errorf("unexpected path %q", path)
	}
	if agent != userAgent || strings.Contains(strings.ToLower(agent), "bot") {
		t.Errorf("user agent %q must not look like a crawler", agent)
	}

	fail = true
	var reqErr *RequestError
	if err := c.SubmitRequest(context.Background(), "chill", "abc123"); !errors.As(err, &reqErr) || reqErr.Message != "This song or artist has been played too recently." {
		t.Errorf("AzuraCast's message should come through, got %v", err)
	}
}
