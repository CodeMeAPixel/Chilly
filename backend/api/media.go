package api

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

var mediaHeaders = []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Last-Modified", "ETag"}

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

	upstream, err := s.bot.Radio.Client().PlayFile(r.Context(), station, id, r.Header.Get("Range"))
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
		slog.Warn("music library returned an error for media",
			slog.String("station", station), slog.Int("id", id), slog.Int("status", upstream.StatusCode))
		writeError(w, http.StatusBadGateway, "upstream_error", "the music library returned an error")
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
