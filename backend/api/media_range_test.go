package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/CodeMeAPixel/Chilly/azuracast"
	"github.com/CodeMeAPixel/Chilly/musicbot"
)

func TestParseRange(t *testing.T) {
	cases := []struct {
		header     string
		total      int64
		start, end int64
		result     rangeResult
	}{
		{"bytes=0-15", 100, 0, 15, rangeValid},
		{"bytes=10-", 100, 10, 99, rangeValid},
		{"bytes=90-500", 100, 90, 99, rangeValid},
		{"bytes=-20", 100, 80, 99, rangeValid},
		{"bytes=-500", 100, 0, 99, rangeValid},
		{"bytes=100-", 100, 0, 0, rangeUnsatisfiable},
		{"bytes=5-2", 100, 0, 0, rangeIgnored},
		{"bytes=0-1,5-6", 100, 0, 0, rangeIgnored},
		{"items=0-1", 100, 0, 0, rangeIgnored},
		{"bytes=0-10", 0, 0, 0, rangeIgnored},
	}
	for _, c := range cases {
		got, result := parseRange(c.header, c.total)
		if result != c.result || (result == rangeValid && (got.start != c.start || got.end != c.end)) {
			t.Errorf("parseRange(%q, %d) = %+v %v, want %d-%d %v", c.header, c.total, got, result, c.start, c.end, c.result)
		}
	}
}

func TestMediaProxyEmulatesRangesWhenUpstreamCannot(t *testing.T) {
	const file = "0123456789abcdefghij"
	var rangeRequests atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "" {
			rangeRequests.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"message":"stream is not seekable"}`)
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Content-Length", "20")
		_, _ = io.WriteString(w, file)
	}))
	defer upstream.Close()

	signer := musicbot.NewMediaSigner("secret", "http://bot")
	s := &Server{bot: &musicbot.Bot{Media: signer, Radio: azuracast.NewService(azuracast.Config{URL: upstream.URL})}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/media/{station}/{mediaID}", s.handleMedia)
	path := strings.TrimPrefix(signer.URL("s3station", 1), "http://bot")

	fetch := func(rangeHeader string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if rangeHeader != "" {
			req.Header.Set("Range", rangeHeader)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	rec := fetch("bytes=5-9")
	if rec.Code != http.StatusPartialContent || rec.Body.String() != "56789" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Range") != "bytes 5-9/20" || rec.Header().Get("Content-Length") != "5" || rec.Header().Get("Accept-Ranges") != "bytes" {
		t.Errorf("unexpected headers %v", rec.Header())
	}

	rec = fetch("bytes=15-")
	if rec.Code != http.StatusPartialContent || rec.Body.String() != "fghij" {
		t.Errorf("open-ended range: %d %q", rec.Code, rec.Body.String())
	}
	if n := rangeRequests.Load(); n != 1 {
		t.Errorf("the station should be remembered after the first failure, got %d range attempts", n)
	}

	if rec := fetch("bytes=50-"); rec.Code != http.StatusRequestedRangeNotSatisfiable || rec.Header().Get("Content-Range") != "bytes */20" {
		t.Errorf("out of range: %d %v", rec.Code, rec.Header())
	}
	if rec := fetch(""); rec.Code != http.StatusOK || rec.Body.String() != file {
		t.Errorf("plain request: %d %q", rec.Code, rec.Body.String())
	}
}
