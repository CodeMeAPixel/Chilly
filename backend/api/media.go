package api

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Chilly/azuracast"
)

var mediaHeaders = []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Last-Modified", "ETag"}

type byteRange struct {
	start, end int64
}

type rangeResult int

const (
	rangeIgnored rangeResult = iota
	rangeValid
	rangeUnsatisfiable
)

func parseRange(header string, total int64) (byteRange, rangeResult) {
	spec, ok := strings.CutPrefix(strings.TrimSpace(header), "bytes=")
	if !ok || strings.Contains(spec, ",") || total <= 0 {
		return byteRange{}, rangeIgnored
	}
	first, last, ok := strings.Cut(spec, "-")
	if !ok {
		return byteRange{}, rangeIgnored
	}
	if first == "" {
		n, err := strconv.ParseInt(last, 10, 64)
		if err != nil || n <= 0 {
			return byteRange{}, rangeIgnored
		}
		return byteRange{start: max(total-n, 0), end: total - 1}, rangeValid
	}
	start, err := strconv.ParseInt(first, 10, 64)
	if err != nil || start < 0 {
		return byteRange{}, rangeIgnored
	}
	end := total - 1
	if last != "" {
		if end, err = strconv.ParseInt(last, 10, 64); err != nil || end < start {
			return byteRange{}, rangeIgnored
		}
	}
	if start >= total {
		return byteRange{}, rangeUnsatisfiable
	}
	return byteRange{start: start, end: min(end, total-1)}, rangeValid
}

func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	if s.bot.Media == nil || s.bot.Radio == nil {
		writeError(w, http.StatusNotFound, "not_found", "library playback is not configured")
		return
	}
	station := r.PathValue("station")
	id, err := strconv.Atoi(r.PathValue("mediaID"))
	if err != nil || id <= 0 || !s.bot.Media.Verify(station, id, r.URL.Query().Get("sig")) {
		writeError(w, http.StatusForbidden, "forbidden", "invalid media link")
		return
	}

	if s.serveCachedMedia(w, r, station, id) {
		return
	}

	rangeHeader := r.Header.Get("Range")
	_, noRanges := s.mediaNoRange.Load(station)
	emulate := rangeHeader != "" && noRanges

	client := s.bot.Radio.Client()
	forwarded := rangeHeader
	if emulate {
		forwarded = ""
	}
	upstream, err := client.PlayFile(r.Context(), station, id, forwarded)
	if err == nil && upstream.StatusCode >= 500 && forwarded != "" {
		upstream.Body.Close()
		upstream, err = client.PlayFile(r.Context(), station, id, "")
		if err == nil && upstream.StatusCode == http.StatusOK {
			s.mediaNoRange.Store(station, true)
			slog.Info("music library can't serve partial content for this station; serving ranges from full downloads", slog.String("station", station))
			emulate = true
		}
	}
	if err != nil {
		if !errors.Is(r.Context().Err(), err) {
			slog.Warn("failed to fetch library media", slog.String("station", station), slog.Int("id", id), slog.Any("error", err))
		}
		writeError(w, http.StatusBadGateway, "upstream_error", "couldn't reach the music library")
		return
	}
	defer upstream.Body.Close()

	switch upstream.StatusCode {
	case http.StatusOK, http.StatusPartialContent:
	case http.StatusNotFound:
		writeError(w, http.StatusNotFound, "not_found", "that song is no longer in the library")
		return
	case http.StatusRequestedRangeNotSatisfiable:
		w.WriteHeader(upstream.StatusCode)
		return
	default:
		body, _ := io.ReadAll(io.LimitReader(upstream.Body, 1024))
		slog.Warn("music library returned an error for media",
			slog.String("station", station), slog.Int("id", id), slog.Int("status", upstream.StatusCode),
			slog.String("body", strings.TrimSpace(string(body))))
		writeError(w, http.StatusBadGateway, "upstream_error", "the music library returned an error")
		return
	}

	if emulate && upstream.StatusCode == http.StatusOK {
		serveRangeFromFull(w, r, upstream, rangeHeader)
		return
	}

	for _, h := range mediaHeaders {
		if v := upstream.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(upstream.StatusCode)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = io.Copy(w, upstream.Body)
}

func (s *Server) serveCachedMedia(w http.ResponseWriter, r *http.Request, station string, id int) bool {
	cache := s.bot.MediaCache
	if cache == nil {
		return false
	}
	f, err := cache.Open(r.Context(), station, id)
	if err != nil {
		var status *azuracast.StatusError
		switch {
		case errors.As(err, &status) && status.Status == http.StatusNotFound:
			writeError(w, http.StatusNotFound, "not_found", "that song is no longer in the library")
			return true
		case r.Context().Err() != nil:
			return true
		}
		slog.Warn("media cache unavailable, streaming from the music library",
			slog.String("station", station), slog.Int("id", id), slog.Any("error", err))
		return false
	}
	defer f.Close()
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeContent(w, r, "", time.Time{}, f)
	return true
}

func serveRangeFromFull(w http.ResponseWriter, r *http.Request, upstream *http.Response, rangeHeader string) {
	total := upstream.ContentLength
	h := w.Header()
	h.Set("Cache-Control", "private, no-store")
	h.Set("Accept-Ranges", "bytes")
	if ct := upstream.Header.Get("Content-Type"); ct != "" {
		h.Set("Content-Type", ct)
	}

	rng, result := parseRange(rangeHeader, total)
	switch result {
	case rangeUnsatisfiable:
		h.Set("Content-Range", fmt.Sprintf("bytes */%d", total))
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	case rangeIgnored:
		if total > 0 {
			h.Set("Content-Length", strconv.FormatInt(total, 10))
		}
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			_, _ = io.Copy(w, upstream.Body)
		}
		return
	}

	length := rng.end - rng.start + 1
	h.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", rng.start, rng.end, total))
	h.Set("Content-Length", strconv.FormatInt(length, 10))
	w.WriteHeader(http.StatusPartialContent)
	if r.Method == http.MethodHead {
		return
	}
	if _, err := io.CopyN(io.Discard, upstream.Body, rng.start); err != nil {
		return
	}
	_, _ = io.CopyN(w, upstream.Body, length)
}
