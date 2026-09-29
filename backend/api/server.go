package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"
)

const apiPrefix = "/api/v1"

type Server struct {
	bot      *musicbot.Bot
	cfg      musicbot.APIConfig
	sessions *SessionStore
	http     *http.Server
	oauth    *discordOAuth
	limiter  *rateLimiter
	strict   *rateLimiter
	status   *statusTracker
}

func New(bot *musicbot.Bot) *Server {
	cfg := bot.Cfg.API
	if cfg.ClientID == "" {
		cfg.ClientID = bot.Client.ApplicationID().String()
	}

	s := &Server{
		bot:      bot,
		cfg:      cfg,
		sessions: NewSessionStore(bot.Db),
		oauth:    newDiscordOAuth(cfg),
		limiter:  newRateLimiter(10, 40),
		strict:   newRateLimiter(1, 5),
		status:   newStatusTracker(),
	}

	mux := http.NewServeMux()
	s.routes(mux)

	s.http = &http.Server{
		Addr:              cfg.Address,
		Handler:           s.middleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return s
}

func (s *Server) routes(mux *http.ServeMux) {
	p := apiPrefix

	mux.HandleFunc("GET "+p+"/health", s.handleHealth)
	mux.HandleFunc("GET "+p+"/stats", s.handleStats)
	mux.HandleFunc("GET "+p+"/status", s.handleStatus)

	mux.Handle("GET "+p+"/auth/login", s.limitStrict(http.HandlerFunc(s.handleLogin)))
	mux.Handle("GET "+p+"/auth/callback", s.limitStrict(http.HandlerFunc(s.handleCallback)))
	mux.HandleFunc("POST "+p+"/auth/logout", s.handleLogout)
	mux.Handle("GET "+p+"/auth/me", s.authed(s.handleMe))

	mux.Handle("GET "+p+"/search", s.authed(s.handleSearch))

	mux.Handle("GET "+p+"/guilds", s.authed(s.handleGuilds))
	mux.Handle("GET "+p+"/guilds/{guildID}/player", s.authed(s.handlePlayer))
	mux.Handle("PATCH "+p+"/guilds/{guildID}/player", s.authed(s.handlePlayerUpdate))
	mux.Handle("GET "+p+"/guilds/{guildID}/player/events", s.authed(s.handlePlayerEvents))
	mux.Handle("GET "+p+"/guilds/{guildID}/player/lyrics", s.authed(s.handleLyrics))
	mux.Handle("POST "+p+"/guilds/{guildID}/player/skip", s.authed(s.handleSkip))
	mux.Handle("POST "+p+"/guilds/{guildID}/player/previous", s.authed(s.handlePrevious))
	mux.Handle("POST "+p+"/guilds/{guildID}/player/stop", s.authed(s.handleStop))
	mux.Handle("POST "+p+"/guilds/{guildID}/queue", s.authed(s.handleEnqueue))
	mux.Handle("DELETE "+p+"/guilds/{guildID}/queue", s.authed(s.handleClearQueue))
	mux.Handle("POST "+p+"/guilds/{guildID}/queue/move", s.authed(s.handleMoveQueue))
	mux.Handle("DELETE "+p+"/guilds/{guildID}/queue/{index}", s.authed(s.handleRemoveQueue))
	mux.Handle("POST "+p+"/guilds/{guildID}/radio", s.authed(s.handlePlayRadio))
	mux.Handle("GET "+p+"/guilds/{guildID}/radio/247", s.authed(s.handleGetStay))
	mux.Handle("PUT "+p+"/guilds/{guildID}/radio/247", s.authed(s.handleEnableStay))
	mux.Handle("DELETE "+p+"/guilds/{guildID}/radio/247", s.authed(s.handleDisableStay))

	mux.Handle("GET "+p+"/playlists", s.authed(s.handleListPlaylists))
	mux.Handle("POST "+p+"/playlists", s.authed(s.handleCreatePlaylist))
	mux.Handle("GET "+p+"/playlists/{playlistID}", s.authed(s.handleGetPlaylist))
	mux.Handle("PATCH "+p+"/playlists/{playlistID}", s.authed(s.handleRenamePlaylist))
	mux.Handle("DELETE "+p+"/playlists/{playlistID}", s.authed(s.handleDeletePlaylist))
	mux.Handle("POST "+p+"/playlists/{playlistID}/tracks", s.authed(s.handleAddPlaylistTracks))
	mux.Handle("DELETE "+p+"/playlists/{playlistID}/tracks/{trackID}", s.authed(s.handleRemovePlaylistTrack))

	mux.HandleFunc("GET "+p+"/radio/stations", s.handleRadioStations)
	mux.HandleFunc("GET "+p+"/radio/stations/{station}", s.handleRadioStation)

	s.adminRoutes(mux)
	mux.HandleFunc(p+"/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "route not found")
	})
}

func (s *Server) Start() {
	slog.Info("api listening", slog.String("address", s.cfg.Address))
	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("api server failed", slog.Any("error", err))
	}
}

func (s *Server) Shutdown(ctx context.Context) {
	if err := s.http.Shutdown(ctx); err != nil {
		slog.Error("failed to shut down api server", slog.Any("error", err))
	}
}

func (s *Server) RunJanitor(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n, err := s.sessions.DeleteExpired(ctx); err != nil {
				slog.Warn("failed to delete expired sessions", slog.Any("error", err))
			} else if n > 0 {
				slog.Debug("deleted expired sessions", slog.Int64("count", n))
			}
		}
	}
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Debug("failed to write response", slog.Any("error", err))
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]apiError{"error": {Code: code, Message: message}})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "bad_request", "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body: "+err.Error())
		}
		return false
	}
	return true
}
