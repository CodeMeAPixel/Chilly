package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CodeMeAPixel/Chilly/azuracast"
	"github.com/CodeMeAPixel/Chilly/musicbot"
)

func TestMediaProxy(t *testing.T) {
	var gotKey, gotRange, gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey, gotRange, gotPath = r.Header.Get("X-API-Key"), r.Header.Get("Range"), r.URL.Path
		if strings.HasSuffix(r.URL.Path, "/file/404/play") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Range", "bytes 0-3/10")
		w.Header().Set("Content-Length", "4")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.WriteString(w, "ID3!")
	}))
	defer upstream.Close()

	signer := musicbot.NewMediaSigner("secret", "http://bot")
	s := &Server{bot: &musicbot.Bot{
		Media: signer,
		Radio: azuracast.NewService(azuracast.Config{URL: upstream.URL, APIKey: "key-123"}),
	}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/media/{station}/{mediaID}", s.handleMedia)

	serve := func(target, rangeHeader string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		if rangeHeader != "" {
			req.Header.Set("Range", rangeHeader)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	path := func(link string) string { return strings.TrimPrefix(link, "http://bot") }

	if rec := serve("/api/v1/media/chill/7?sig=nope", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("bad signature: got %d, want 403", rec.Code)
	}
	if rec := serve(path(signer.URL("chill", 8))+"&x=1", ""); rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("valid link: got %d", rec.Code)
	}

	rec := serve(path(signer.URL("chill", 7)), "bytes=0-3")
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("got %d, want 206", rec.Code)
	}
	if gotKey != "key-123" || gotRange != "bytes=0-3" || gotPath != "/api/station/chill/file/7/play" {
		t.Errorf("upstream got key=%q range=%q path=%q", gotKey, gotRange, gotPath)
	}
	if rec.Body.String() != "ID3!" || rec.Header().Get("Content-Range") != "bytes 0-3/10" || rec.Header().Get("Content-Length") != "4" {
		t.Errorf("response not passed through: body=%q headers=%v", rec.Body.String(), rec.Header())
	}

	if rec := serve(path(signer.URL("chill", 404)), ""); rec.Code != http.StatusNotFound {
		t.Errorf("missing file: got %d, want 404", rec.Code)
	}

	disabled := &Server{bot: &musicbot.Bot{}}
	rec = httptest.NewRecorder()
	disabled.handleMedia(rec, httptest.NewRequest(http.MethodGet, "/api/v1/media/chill/7", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("unconfigured proxy: got %d, want 404", rec.Code)
	}
}
