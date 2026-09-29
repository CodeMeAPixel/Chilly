package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

func (s *Server) adminOnly(h authedHandler) http.Handler {
	return s.authed(func(w http.ResponseWriter, r *http.Request, sess *Session) {
		if !s.isAdmin(sess.UserID) {
			writeError(w, http.StatusForbidden, "forbidden", "admin access required")
			return
		}
		h(w, r, sess)
	})
}

func (s *Server) adminRoutes(mux *http.ServeMux) {
	p := apiPrefix + "/admin"
	mux.Handle("GET "+p+"/overview", s.adminOnly(s.handleAdminOverview))
	mux.Handle("GET "+p+"/guilds", s.adminOnly(s.handleAdminGuilds))
	mux.Handle("GET "+p+"/guilds/{guildID}", s.adminOnly(s.handleAdminGuild))
	mux.Handle("POST "+p+"/guilds/{guildID}/disconnect", s.adminOnly(s.handleAdminDisconnect))
	mux.Handle("POST "+p+"/guilds/{guildID}/move", s.adminOnly(s.handleAdminMove))
	mux.Handle("POST "+p+"/guilds/{guildID}/leave", s.adminOnly(s.handleAdminLeave))
	mux.Handle("POST "+p+"/search", s.adminOnly(s.handleAdminSearch))
	mux.Handle("GET "+p+"/logs", s.adminOnly(s.handleAdminLogs))
}

type adminRadio struct {
	Enabled  bool      `json:"enabled"`
	Healthy  bool      `json:"healthy"`
	LastPoll time.Time `json:"last_poll,omitempty"`
	Stations int       `json:"stations"`
	Online   int       `json:"online"`
}

type adminOverview struct {
	Version          string              `json:"version"`
	GoVersion        string              `json:"go_version"`
	StartedAt        time.Time           `json:"started_at"`
	UptimeSeconds    int64               `json:"uptime_seconds"`
	Goroutines       int                 `json:"goroutines"`
	HeapBytes        uint64              `json:"heap_bytes"`
	SysBytes         uint64              `json:"sys_bytes"`
	GatewayStatus    string              `json:"gateway_status"`
	GatewayLatencyMs int64               `json:"gateway_latency_ms"`
	Guilds           int                 `json:"guilds"`
	Members          int                 `json:"members"`
	Players          int                 `json:"players"`
	Playing          int                 `json:"playing"`
	Radio            adminRadio          `json:"radio"`
	Stays            []stayView          `json:"stays"`
	Nodes            []musicbot.NodeInfo `json:"nodes"`
}

func (s *Server) handleAdminOverview(w http.ResponseWriter, r *http.Request, _ *Session) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	players, playing := s.bot.PlayerManager.Count()

	resp := adminOverview{
		Version:       s.bot.Version,
		GoVersion:     runtime.Version(),
		StartedAt:     s.bot.StartedAt,
		UptimeSeconds: int64(time.Since(s.bot.StartedAt).Seconds()),
		Goroutines:    runtime.NumGoroutine(),
		HeapBytes:     mem.HeapAlloc,
		SysBytes:      mem.Sys,
		Guilds:        s.bot.Client.Caches().GuildsLen(),
		Players:       players,
		Playing:       playing,
		Stays:         make([]stayView, 0),
		Nodes:         s.nodes(),
	}
	s.bot.Client.Caches().GuildsForEach(func(g discord.Guild) { resp.Members += g.MemberCount })
	if s.bot.Client.HasGateway() {
		gw := s.bot.Client.Gateway()
		resp.GatewayStatus = gw.Status().String()
		resp.GatewayLatencyMs = gw.Latency().Milliseconds()
	}
	if s.bot.Radio != nil {
		healthy, lastPoll := s.bot.Radio.Healthy()
		resp.Radio = adminRadio{Enabled: true, Healthy: healthy, LastPoll: lastPoll}
		for _, np := range s.bot.Radio.Stations() {
			resp.Radio.Stations++
			if np.IsOnline {
				resp.Radio.Online++
			}
		}
	}
	for _, setting := range s.bot.Stays.All() {
		resp.Stays = append(resp.Stays, s.stayView(setting))
	}
	writeJSON(w, http.StatusOK, resp)
}

type adminGuild struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	IconURL        *string   `json:"icon_url"`
	MemberCount    int       `json:"member_count"`
	OwnerID        string    `json:"owner_id"`
	JoinedAt       time.Time `json:"joined_at"`
	VoiceChannelID string    `json:"voice_channel_id,omitempty"`
	Listeners      int       `json:"listeners"`
	Playing        bool      `json:"playing"`
	Paused         bool      `json:"paused"`
	Current        string    `json:"current,omitempty"`
	Source         string    `json:"source,omitempty"`
	Radio          string    `json:"radio,omitempty"`
	QueueLength    int       `json:"queue_length"`
	Node           string    `json:"node,omitempty"`
	Stay           *stayView `json:"stay"`
}

func (s *Server) adminGuildView(g discord.Guild) adminGuild {
	view := adminGuild{
		ID:          g.ID.String(),
		Name:        g.Name,
		IconURL:     g.IconURL(discord.WithSize(128)),
		MemberCount: g.MemberCount,
		OwnerID:     g.OwnerID.String(),
		JoinedAt:    g.JoinedAt,
	}
	if ch, ok := s.bot.BotVoiceChannel(g.ID); ok {
		view.VoiceChannelID = ch.String()
		view.Listeners = s.bot.ListenerCount(g.ID, ch)
	}
	if p, ok := s.bot.PlayerManager.GetPlayer(g.ID); ok {
		if t, ok := p.Current(); ok {
			view.Playing = true
			view.Paused = p.IsPaused()
			view.Current = musicbot.TrackTitle(t)
			view.Source = t.Info.SourceName
			view.Radio = musicbot.GetTrackMeta(t).Radio
		}
		view.QueueLength = len(p.Queue())
	}
	if lp := s.bot.Lavalink.ExistingPlayer(g.ID); lp != nil && lp.Node() != nil {
		view.Node = lp.Node().Config().Name
	}
	if setting, ok := s.bot.Stays.Get(g.ID); ok {
		stay := s.stayView(setting)
		view.Stay = &stay
	}
	return view
}

func (s *Server) handleAdminGuilds(w http.ResponseWriter, r *http.Request, _ *Session) {
	guilds := make([]adminGuild, 0, s.bot.Client.Caches().GuildsLen())
	s.bot.Client.Caches().GuildsForEach(func(g discord.Guild) {
		guilds = append(guilds, s.adminGuildView(g))
	})
	slices.SortFunc(guilds, func(a, b adminGuild) int {
		if a.Playing != b.Playing {
			if a.Playing {
				return -1
			}
			return 1
		}
		return b.MemberCount - a.MemberCount
	})
	writeJSON(w, http.StatusOK, map[string]any{"guilds": guilds})
}

func (s *Server) adminGuildID(w http.ResponseWriter, r *http.Request) (discord.Guild, bool) {
	guildID, err := snowflake.Parse(r.PathValue("guildID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid guild id")
		return discord.Guild{}, false
	}
	g, ok := s.bot.Client.Caches().Guild(guildID)
	if !ok {
		writeError(w, http.StatusNotFound, "guild_not_found", "the bot is not in that server")
		return discord.Guild{}, false
	}
	return g, true
}

func (s *Server) handleAdminGuild(w http.ResponseWriter, r *http.Request, _ *Session) {
	g, ok := s.adminGuildID(w, r)
	if !ok {
		return
	}
	resp := map[string]any{"guild": s.adminGuildView(g)}
	if p, ok := s.bot.PlayerManager.GetPlayer(g.ID); ok {
		resp["player"] = p.Snapshot(25)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminDisconnect(w http.ResponseWriter, r *http.Request, sess *Session) {
	g, ok := s.adminGuildID(w, r)
	if !ok {
		return
	}
	if _, err := s.bot.DisableStay(r.Context(), g.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "failed to turn off 24/7 radio")
		return
	}
	if p, ok := s.bot.PlayerManager.GetPlayer(g.ID); ok {
		_ = p.Stop(r.Context())
	}
	if err := s.bot.Client.UpdateVoiceState(r.Context(), g.ID, nil, false, true); err != nil {
		writeError(w, http.StatusBadGateway, "discord_error", err.Error())
		return
	}
	slog.Info("admin disconnected player", slog.String("guild_id", g.ID.String()), slog.String("admin_id", sess.UserID.String()))
	writeJSON(w, http.StatusOK, map[string]any{"guild": s.adminGuildView(g)})
}

func (s *Server) handleAdminMove(w http.ResponseWriter, r *http.Request, sess *Session) {
	g, ok := s.adminGuildID(w, r)
	if !ok {
		return
	}
	var req struct {
		Node string `json:"node"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	player, ok := s.bot.PlayerManager.GetPlayer(g.ID)
	if !ok || s.bot.Nodes == nil {
		writeError(w, http.StatusConflict, "no_player", "nothing is playing in that server")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := s.bot.Nodes.MoveTo(ctx, player, req.Node); err != nil {
		if errors.Is(err, musicbot.ErrNodeNotFound) {
			writeError(w, http.StatusNotFound, "node_not_found", err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, "move_failed", err.Error())
		return
	}
	slog.Info("admin moved player", slog.String("guild_id", g.ID.String()), slog.String("node", req.Node), slog.String("admin_id", sess.UserID.String()))
	writeJSON(w, http.StatusOK, map[string]any{"guild": s.adminGuildView(g)})
}

func (s *Server) handleAdminLeave(w http.ResponseWriter, r *http.Request, sess *Session) {
	g, ok := s.adminGuildID(w, r)
	if !ok {
		return
	}
	if _, err := s.bot.DisableStay(r.Context(), g.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "failed to turn off 24/7 radio")
		return
	}
	if err := s.bot.Client.Rest().LeaveGuild(g.ID); err != nil {
		writeError(w, http.StatusBadGateway, "discord_error", err.Error())
		return
	}
	slog.Warn("admin removed bot from server",
		slog.String("guild_id", g.ID.String()), slog.String("guild_name", g.Name), slog.String("admin_id", sess.UserID.String()))
	w.WriteHeader(http.StatusNoContent)
}

type adminTrack struct {
	Title    string `json:"title"`
	Author   string `json:"author"`
	LengthMs int64  `json:"length_ms"`
	URI      string `json:"uri,omitempty"`
	Source   string `json:"source"`
	IsStream bool   `json:"is_stream"`
}

func (s *Server) handleAdminSearch(w http.ResponseWriter, r *http.Request, _ *Session) {
	var req struct {
		Query  string `json:"query"`
		Source string `json:"source"`
		Node   string `json:"node"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "query is required")
		return
	}

	var node disgolink.Node
	if req.Node != "" {
		node = s.bot.Lavalink.Node(req.Node)
	} else {
		node = musicbot.BestNode(s.bot.Lavalink)
	}
	if node == nil || node.Status() != disgolink.StatusConnected {
		writeError(w, http.StatusServiceUnavailable, "no_node", "that node is not connected")
		return
	}

	identifier := req.Query
	if !strings.HasPrefix(identifier, "http://") && !strings.HasPrefix(identifier, "https://") {
		source := req.Source
		if source == "" {
			source = "ytsearch"
		}
		identifier = source + ":" + identifier
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	started := time.Now()
	result, err := node.LoadTracks(ctx, identifier)
	resp := map[string]any{
		"node":       node.Config().Name,
		"identifier": identifier,
		"took_ms":    time.Since(started).Milliseconds(),
		"tracks":     []adminTrack{},
	}
	if err != nil {
		resp["load_type"] = "error"
		resp["error"] = err.Error()
		writeJSON(w, http.StatusOK, resp)
		return
	}
	resp["load_type"] = result.LoadType

	var tracks []lavalink.Track
	switch data := result.Data.(type) {
	case lavalink.Track:
		tracks = []lavalink.Track{data}
	case lavalink.Playlist:
		resp["playlist"] = data.Info.Name
		tracks = data.Tracks
	case lavalink.Search:
		tracks = data
	case lavalink.Exception:
		resp["error"] = data.Message
		if data.Cause != "" {
			resp["cause"] = data.Cause
		}
	}
	views := make([]adminTrack, 0, min(len(tracks), 10))
	for _, t := range tracks[:min(len(tracks), 10)] {
		view := adminTrack{
			Title:    t.Info.Title,
			Author:   t.Info.Author,
			LengthMs: int64(t.Info.Length),
			Source:   t.Info.SourceName,
			IsStream: t.Info.IsStream,
		}
		if t.Info.URI != nil {
			view.URI = *t.Info.URI
		}
		views = append(views, view)
	}
	resp["tracks"] = views
	resp["total"] = len(tracks)
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminLogs(w http.ResponseWriter, r *http.Request, _ *Session) {
	if s.bot.Logs == nil {
		writeJSON(w, http.StatusOK, map[string]any{"entries": []musicbot.LogEntry{}})
		return
	}
	level := slog.LevelInfo
	if raw := r.URL.Query().Get("level"); raw != "" {
		if err := level.UnmarshalText([]byte(raw)); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "level must be debug, info, warn or error")
			return
		}
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 {
		limit = 200
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": s.bot.Logs.Entries(level, min(limit, 1000))})
}
