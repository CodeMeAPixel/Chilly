package musicbot

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func flakyServer(t *testing.T, failures int32) (*httptest.Server, *atomic.Int32, *[]string) {
	t.Helper()
	var calls atomic.Int32
	bodies := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		if calls.Add(1) <= failures {
			http.Error(w, "upstream connect error or disconnect/reset before headers. reset reason: overflow", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls, &bodies
}

func TestRetriesInteractionCallbackOn503(t *testing.T) {
	srv, calls, bodies := flakyServer(t, 1)
	client := NewDiscordHTTPClient()

	resp, err := client.Post(srv.URL+"/api/v10/interactions/1/token/callback", "application/json", strings.NewReader(`{"type":5}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent || calls.Load() != 2 {
		t.Fatalf("expected success after one retry, got status %d after %d calls", resp.StatusCode, calls.Load())
	}
	if (*bodies)[1] != `{"type":5}` {
		t.Fatalf("body was not replayed on retry: %q", (*bodies)[1])
	}
}

func TestDoesNotRetryMessageCreate(t *testing.T) {
	srv, calls, _ := flakyServer(t, 1)
	client := NewDiscordHTTPClient()

	resp, err := client.Post(srv.URL+"/api/v10/channels/1/messages", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable || calls.Load() != 1 {
		t.Fatalf("message creates must not be retried, got %d calls", calls.Load())
	}
}

func TestGivesUpAfterMaxRetries(t *testing.T) {
	srv, calls, _ := flakyServer(t, 10)
	client := NewDiscordHTTPClient()

	resp, err := client.Get(srv.URL + "/api/v10/users/@me")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if calls.Load() != discordRetries+1 {
		t.Fatalf("expected %d attempts, got %d", discordRetries+1, calls.Load())
	}
}
