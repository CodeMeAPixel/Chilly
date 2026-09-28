package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgolink/v3/lavalink"
)

type playlistTrackView struct {
	ID      int                `json:"id"`
	AddedAt time.Time          `json:"added_at"`
	AddedBy string             `json:"added_by"`
	Track   musicbot.TrackView `json:"track"`
}

func playlistID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("playlistID"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid playlist id")
		return 0, false
	}
	return id, true
}

func validPlaylistName(w http.ResponseWriter, name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 100 {
		writeError(w, http.StatusBadRequest, "bad_request", "name must be between 1 and 100 characters")
		return "", false
	}
	return name, true
}

func (s *Server) playlistError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, musicbot.ErrPlaylistNotFound):
		writeError(w, http.StatusNotFound, "not_found", "playlist not found")
	case errors.Is(err, musicbot.ErrPlaylistExists):
		writeError(w, http.StatusConflict, "exists", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal", "database error")
	}
}

func (s *Server) handleListPlaylists(w http.ResponseWriter, r *http.Request, sess *Session) {
	playlists, err := s.bot.Db.SearchPlaylist(r.Context(), sess.UserID, r.URL.Query().Get("q"), 100)
	if err != nil {
		s.playlistError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"playlists": playlists})
}

func (s *Server) handleCreatePlaylist(w http.ResponseWriter, r *http.Request, sess *Session) {
	var req struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	name, ok := validPlaylistName(w, req.Name)
	if !ok {
		return
	}
	playlist, err := s.bot.Db.CreatePlaylist(r.Context(), sess.UserID, name)
	if err != nil {
		s.playlistError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, playlist)
}

func (s *Server) handleGetPlaylist(w http.ResponseWriter, r *http.Request, sess *Session) {
	id, ok := playlistID(w, r)
	if !ok {
		return
	}
	playlist, tracks, err := s.bot.Db.GetPlaylist(r.Context(), sess.UserID, id)
	if err != nil {
		s.playlistError(w, err)
		return
	}
	views := make([]playlistTrackView, len(tracks))
	for i, t := range tracks {
		views[i] = playlistTrackView{ID: t.ID, AddedAt: t.AddedAt, AddedBy: t.AddedBy.String(), Track: musicbot.NewTrackView(t.Track)}
	}
	writeJSON(w, http.StatusOK, map[string]any{"playlist": playlist, "tracks": views})
}

func (s *Server) handleRenamePlaylist(w http.ResponseWriter, r *http.Request, sess *Session) {
	id, ok := playlistID(w, r)
	if !ok {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	name, ok := validPlaylistName(w, req.Name)
	if !ok {
		return
	}
	if err := s.bot.Db.RenamePlaylist(r.Context(), sess.UserID, id, name); err != nil {
		s.playlistError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeletePlaylist(w http.ResponseWriter, r *http.Request, sess *Session) {
	id, ok := playlistID(w, r)
	if !ok {
		return
	}
	if err := s.bot.Db.DeletePlaylist(r.Context(), sess.UserID, id); err != nil {
		s.playlistError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAddPlaylistTracks(w http.ResponseWriter, r *http.Request, sess *Session) {
	id, ok := playlistID(w, r)
	if !ok {
		return
	}
	var req struct {
		Query  string `json:"query"`
		Source string `json:"source"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "query is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if _, _, err := s.bot.Db.GetPlaylist(ctx, sess.UserID, id); err != nil {
		s.playlistError(w, err)
		return
	}
	result, err := s.bot.Searcher.Resolve(ctx, req.Query, req.Source)
	if err != nil {
		s.playerError(w, err)
		return
	}
	var tracks []lavalink.Track
	switch data := result.Data.(type) {
	case lavalink.Track:
		tracks = []lavalink.Track{data}
	case lavalink.Search:
		if len(data) > 0 {
			tracks = []lavalink.Track{data[0]}
		}
	case lavalink.Playlist:
		tracks = data.Tracks
	}
	if len(tracks) == 0 {
		writeError(w, http.StatusNotFound, "no_results", "no tracks found")
		return
	}
	if err := s.bot.Db.AddTracksToPlaylist(ctx, id, sess.UserID, tracks); err != nil {
		s.playlistError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"added": len(tracks), "tracks": musicbot.NewTrackViews(tracks[:min(len(tracks), 50)])})
}

func (s *Server) handleRemovePlaylistTrack(w http.ResponseWriter, r *http.Request, sess *Session) {
	id, ok := playlistID(w, r)
	if !ok {
		return
	}
	trackID, err := strconv.Atoi(r.PathValue("trackID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid track id")
		return
	}
	if err := s.bot.Db.RemoveTrackFromPlaylist(r.Context(), sess.UserID, id, trackID); err != nil {
		s.playlistError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
