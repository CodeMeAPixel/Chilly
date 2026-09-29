package api

import (
	"net/http"
	"strings"

	"github.com/CodeMeAPixel/Chilly/musicbot"
)

func (s *Server) library(w http.ResponseWriter) (*musicbot.Library, bool) {
	lib := s.bot.Searcher.Library()
	if lib == nil || !lib.Ready() {
		writeError(w, http.StatusServiceUnavailable, "library_unavailable", musicbot.ErrLibraryUnavailable.Error())
		return nil, false
	}
	return lib, true
}

func (s *Server) handleLibraryPlaylists(w http.ResponseWriter, r *http.Request) {
	lib, ok := s.library(w)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"playlists": lib.Playlists()})
}

func (s *Server) handleLibraryPlaylistTracks(w http.ResponseWriter, r *http.Request) {
	lib, ok := s.library(w)
	if !ok {
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name is required")
		return
	}
	tracks := lib.PlaylistTracks(name)
	if len(tracks) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "playlist not found")
		return
	}
	results := make([]libraryResult, len(tracks))
	for i, t := range tracks {
		results[i] = newLibraryResult(t)
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": name, "tracks": results})
}
