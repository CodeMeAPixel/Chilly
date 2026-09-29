package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Chilly/azuracast"
	"github.com/CodeMeAPixel/Chilly/musicbot"
)

func (s *Server) handleLibraryTracks(w http.ResponseWriter, r *http.Request) {
	lib, ok := s.library(w)
	if !ok {
		return
	}
	q := r.URL.Query()
	perPage, err := strconv.Atoi(q.Get("per_page"))
	if err != nil || perPage <= 0 || perPage > 100 {
		perPage = 50
	}
	page, err := strconv.Atoi(q.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	tracks, total := lib.Browse(musicbot.LibraryQuery{
		Query:    q.Get("q"),
		Artist:   q.Get("artist"),
		Album:    q.Get("album"),
		Playlist: q.Get("playlist"),
		Sort:     q.Get("sort"),
		Offset:   (page - 1) * perPage,
		Limit:    perPage,
	})
	results := make([]libraryResult, len(tracks))
	for i, t := range tracks {
		results[i] = newLibraryResult(t)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tracks":   results,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

func (s *Server) handleLibraryGroups(kind musicbot.LibraryGroup) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lib, ok := s.library(w)
		if !ok {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": lib.Groups(kind)})
	}
}

func (s *Server) handleLibrarySummary(w http.ResponseWriter, r *http.Request) {
	lib, ok := s.library(w)
	if !ok {
		return
	}
	count, lastSync, _ := lib.Status()
	resp := map[string]any{
		"tracks":    count,
		"artists":   len(lib.Groups(musicbot.GroupArtist)),
		"albums":    len(lib.Groups(musicbot.GroupAlbum)),
		"playlists": len(lib.Playlists()),
		"synced_at": lastSync,
	}
	if s.bot.Requests != nil {
		cfg := s.bot.Requests.Config()
		resp["requests"] = map[string]any{
			"cooldown_seconds":     int(cfg.Cooldown / time.Second),
			"max_pending":          cfg.MaxPending,
			"max_open_suggestions": cfg.MaxOpenSuggestions,
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) requestError(w http.ResponseWriter, err error) {
	var (
		limit     *musicbot.LimitError
		station   *azuracast.RequestError
		inLibrary *musicbot.AlreadyInLibraryError
	)
	switch {
	case errors.As(err, &limit):
		writeError(w, http.StatusTooManyRequests, "limited", limit.Message)
	case errors.As(err, &station):
		writeError(w, http.StatusConflict, "station_refused", station.Message)
	case errors.As(err, &inLibrary):
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": apiError{Code: "in_library", Message: inLibrary.Error()},
			"track": newLibraryResult(inLibrary.Track),
		})
	case errors.Is(err, musicbot.ErrSelectionExpired):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, musicbot.ErrSuggestionNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, musicbot.ErrInvalidStatus):
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
	case errors.Is(err, musicbot.ErrRequestsUnavailable), errors.Is(err, musicbot.ErrLibraryUnavailable):
		writeError(w, http.StatusServiceUnavailable, "unavailable", err.Error())
	default:
		writeError(w, http.StatusBadGateway, "request_failed", "couldn't reach the station right now")
	}
}

func (s *Server) requests(w http.ResponseWriter) (*musicbot.Requests, bool) {
	if s.bot.Requests == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", musicbot.ErrRequestsUnavailable.Error())
		return nil, false
	}
	return s.bot.Requests, true
}

func (s *Server) handleCreateRequest(w http.ResponseWriter, r *http.Request, sess *Session) {
	reqs, ok := s.requests(w)
	if !ok {
		return
	}
	var body struct {
		Key string `json:"key"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	req, err := reqs.Submit(r.Context(), sess.UserID, 0, strings.TrimSpace(body.Key), "web")
	if err != nil {
		s.requestError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"request": req})
}

func (s *Server) handleMyRequests(w http.ResponseWriter, r *http.Request, sess *Session) {
	requests, err := s.bot.Db.UserRequests(r.Context(), sess.UserID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "database error")
		return
	}
	suggestions, err := s.bot.Db.UserSuggestions(r.Context(), sess.UserID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": requests, "suggestions": suggestions})
}

func (s *Server) handleCreateSuggestion(w http.ResponseWriter, r *http.Request, sess *Session) {
	reqs, ok := s.requests(w)
	if !ok {
		return
	}
	var body struct {
		Artist string `json:"artist"`
		Title  string `json:"title"`
		Link   string `json:"link"`
		Note   string `json:"note"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	suggestion, err := reqs.Suggest(r.Context(), sess.UserID, body.Artist, body.Title, body.Link, body.Note)
	if err != nil {
		s.requestError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"suggestion": suggestion})
}

func suggestionID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("suggestionID"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid suggestion id")
		return 0, false
	}
	return id, true
}

func (s *Server) handleWithdrawSuggestion(w http.ResponseWriter, r *http.Request, sess *Session) {
	reqs, ok := s.requests(w)
	if !ok {
		return
	}
	id, ok := suggestionID(w, r)
	if !ok {
		return
	}
	if err := reqs.Withdraw(r.Context(), id, sess.UserID); err != nil {
		s.requestError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminSuggestions(w http.ResponseWriter, r *http.Request, _ *Session) {
	status := r.URL.Query().Get("status")
	suggestions, err := s.bot.Db.ListSuggestions(r.Context(), status, 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"suggestions": suggestions})
}

func (s *Server) handleAdminReviewSuggestion(w http.ResponseWriter, r *http.Request, sess *Session) {
	reqs, ok := s.requests(w)
	if !ok {
		return
	}
	id, ok := suggestionID(w, r)
	if !ok {
		return
	}
	var body struct {
		Status     string `json:"status"`
		Reason     string `json:"reason"`
		LibraryKey string `json:"library_key"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if len(body.Reason) > 500 {
		writeError(w, http.StatusBadRequest, "bad_request", "reason must be 500 characters or fewer")
		return
	}
	suggestion, err := reqs.Review(r.Context(), id, body.Status, body.Reason, body.LibraryKey, sess.UserID)
	if err != nil {
		s.requestError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"suggestion": suggestion})
}

func (s *Server) handleAdminRequests(w http.ResponseWriter, r *http.Request, _ *Session) {
	requests, err := s.bot.Db.RecentRequests(r.Context(), 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": requests})
}
