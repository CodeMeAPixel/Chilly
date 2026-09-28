package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"
)

type lyricsResponse struct {
	Track  musicbot.TrackView `json:"track"`
	Lyrics *musicbot.Lyrics   `json:"lyrics"`
}

func (s *Server) handleLyrics(w http.ResponseWriter, r *http.Request, sess *Session) {
	a, ok := s.guildAccess(w, r, sess)
	if !ok {
		return
	}
	player, exists := s.bot.PlayerManager.GetPlayer(a.guildID)
	if !exists {
		writeError(w, http.StatusConflict, "not_playing", "nothing is playing")
		return
	}
	track, playing := player.Current()
	if !playing {
		writeError(w, http.StatusConflict, "not_playing", "nothing is playing")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	lyrics, err := s.bot.LyricsForTrack(ctx, track)
	if err != nil {
		if errors.Is(err, musicbot.ErrLyricsNotFound) {
			writeError(w, http.StatusNotFound, "no_lyrics", "no lyrics found for this song")
			return
		}
		slog.Warn("lyrics lookup failed", slog.Any("error", err))
		writeError(w, http.StatusBadGateway, "lyrics_unavailable", "the lyrics service isn't responding")
		return
	}
	writeJSON(w, http.StatusOK, lyricsResponse{Track: musicbot.NewTrackView(track), Lyrics: lyrics})
}
