package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

type guildAccess struct {
	guildID   snowflake.ID
	sess      *Session
	canManage bool
}

func (s *Server) canControl(a guildAccess) bool {
	return a.canManage || s.bot.InSameVoice(a.guildID, a.sess.UserID)
}

func (s *Server) guildAccess(w http.ResponseWriter, r *http.Request, sess *Session) (guildAccess, bool) {
	guildID, err := snowflake.Parse(r.PathValue("guildID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid guild id")
		return guildAccess{}, false
	}
	if _, ok := s.bot.Client.Caches().Guild(guildID); !ok {
		writeError(w, http.StatusNotFound, "guild_not_found", "the bot is not in that server")
		return guildAccess{}, false
	}
	admin := s.isAdmin(sess.UserID)
	g, member := sess.Guild(guildID)
	if !member && !admin {
		writeError(w, http.StatusForbidden, "forbidden", "you are not a member of that server")
		return guildAccess{}, false
	}
	return guildAccess{guildID: guildID, sess: sess, canManage: admin || (member && g.CanManage())}, true
}

func (s *Server) controlAccess(w http.ResponseWriter, r *http.Request, sess *Session) (guildAccess, *musicbot.Player, bool) {
	a, ok := s.guildAccess(w, r, sess)
	if !ok {
		return a, nil, false
	}
	if !s.canControl(a) {
		writeError(w, http.StatusForbidden, "not_in_voice", "join the bot's voice channel to control the player")
		return a, nil, false
	}
	player, ok := s.bot.PlayerManager.GetPlayer(a.guildID)
	if !ok {
		writeError(w, http.StatusConflict, "no_player", "nothing is playing in that server")
		return a, nil, false
	}
	return a, player, true
}

type guildView struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Icon          *string `json:"icon"`
	CanManage     bool    `json:"can_manage"`
	BotVoiceID    string  `json:"bot_voice_channel_id,omitempty"`
	UserVoiceID   string  `json:"user_voice_channel_id,omitempty"`
	Playing       bool    `json:"playing"`
	CurrentTitle  string  `json:"current_title,omitempty"`
	ListeningHere bool    `json:"listening_with_bot"`
}

func (s *Server) handleGuilds(w http.ResponseWriter, r *http.Request, sess *Session) {
	guilds := make([]guildView, 0)
	for _, g := range sess.Guilds {
		if _, ok := s.bot.Client.Caches().Guild(g.ID); !ok {
			continue
		}
		view := guildView{ID: g.ID.String(), Name: g.Name, Icon: g.Icon, CanManage: g.CanManage() || s.isAdmin(sess.UserID)}
		if ch, ok := s.bot.BotVoiceChannel(g.ID); ok {
			view.BotVoiceID = ch.String()
		}
		if ch, ok := s.bot.UserVoiceChannel(g.ID, sess.UserID); ok {
			view.UserVoiceID = ch.String()
		}
		view.ListeningHere = view.BotVoiceID != "" && view.BotVoiceID == view.UserVoiceID
		if p, ok := s.bot.PlayerManager.GetPlayer(g.ID); ok {
			if t, ok := p.Current(); ok {
				view.Playing = true
				view.CurrentTitle = musicbot.TrackTitle(t)
			}
		}
		guilds = append(guilds, view)
	}
	writeJSON(w, http.StatusOK, map[string]any{"guilds": guilds})
}

type playerResponse struct {
	musicbot.PlayerSnapshot
	CanControl bool      `json:"can_control"`
	CanManage  bool      `json:"can_manage"`
	Stay       *stayView `json:"stay"`
}

func (s *Server) snapshot(a guildAccess, limit int) playerResponse {
	resp := playerResponse{CanControl: s.canControl(a), CanManage: a.canManage}
	if setting, ok := s.bot.Stays.Get(a.guildID); ok {
		view := s.stayView(setting)
		resp.Stay = &view
	}
	if p, ok := s.bot.PlayerManager.GetPlayer(a.guildID); ok {
		resp.PlayerSnapshot = p.Snapshot(limit)
	} else {
		resp.PlayerSnapshot = musicbot.PlayerSnapshot{
			GuildID:   a.guildID.String(),
			Loop:      musicbot.LoopNone,
			Queue:     []musicbot.TrackView{},
			History:   []musicbot.TrackView{},
			UpdatedAt: time.Now(),
		}
	}
	if resp.VoiceChannelID == "" {
		if ch, ok := s.bot.BotVoiceChannel(a.guildID); ok {
			resp.VoiceChannelID = ch.String()
		}
	}
	return resp
}

func queueLimit(r *http.Request) int {
	limit, err := strconv.Atoi(r.URL.Query().Get("queue_limit"))
	if err != nil || limit <= 0 {
		return 100
	}
	return min(limit, 1000)
}

func (s *Server) handlePlayer(w http.ResponseWriter, r *http.Request, sess *Session) {
	a, ok := s.guildAccess(w, r, sess)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.snapshot(a, queueLimit(r)))
}

type playerUpdateRequest struct {
	Paused     *bool   `json:"paused"`
	Loop       *string `json:"loop"`
	Shuffle    *bool   `json:"shuffle"`
	Volume     *int    `json:"volume"`
	PositionMs *int64  `json:"position_ms"`
}

func (s *Server) handlePlayerUpdate(w http.ResponseWriter, r *http.Request, sess *Session) {
	a, player, ok := s.controlAccess(w, r, sess)
	if !ok {
		return
	}
	var req playerUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	var loop musicbot.LoopMode
	if req.Loop != nil {
		if loop, ok = musicbot.ParseLoopMode(*req.Loop); !ok {
			writeError(w, http.StatusBadRequest, "bad_request", "loop must be one of none, track, queue")
			return
		}
	}
	if req.Volume != nil && (*req.Volume < 0 || *req.Volume > 200) {
		writeError(w, http.StatusBadRequest, "bad_request", "volume must be between 0 and 200")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if req.Loop != nil {
		player.SetLoop(loop)
	}
	if req.Shuffle != nil {
		player.SetShuffle(musicbot.ShuffleMode(*req.Shuffle))
	}
	if req.Paused != nil {
		if err := player.SetPaused(ctx, *req.Paused); err != nil {
			s.playerError(w, err)
			return
		}
	}
	if req.Volume != nil {
		if err := player.SetVolume(ctx, *req.Volume); err != nil {
			s.playerError(w, err)
			return
		}
	}
	if req.PositionMs != nil {
		if err := player.Seek(ctx, lavalink.Duration(*req.PositionMs)); err != nil {
			s.playerError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, s.snapshot(a, queueLimit(r)))
}

func (s *Server) playerError(w http.ResponseWriter, err error) {
	var busy *musicbot.BusyError
	switch {
	case errors.Is(err, musicbot.ErrExternalSource):
		writeError(w, http.StatusUnprocessableEntity, "external_source", err.Error())
	case errors.Is(err, musicbot.ErrLibraryUnavailable):
		writeError(w, http.StatusServiceUnavailable, "library_unavailable", err.Error())
	case errors.Is(err, musicbot.ErrNothingPlaying):
		writeError(w, http.StatusConflict, "not_playing", "nothing is playing")
	case errors.As(err, &busy):
		writeError(w, http.StatusConflict, "busy", "the bot is playing in another voice channel")
	case errors.Is(err, musicbot.ErrVoiceTimeout):
		writeError(w, http.StatusGatewayTimeout, "voice_timeout", err.Error())
	case errors.Is(err, musicbot.ErrNoNode):
		writeError(w, http.StatusServiceUnavailable, "no_node", "the music server is unavailable")
	case errors.Is(err, musicbot.ErrSelectionExpired), errors.Is(err, musicbot.ErrPlaylistNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	default:
		writeError(w, http.StatusBadGateway, "player_error", err.Error())
	}
}

func (s *Server) handleSkip(w http.ResponseWriter, r *http.Request, sess *Session) {
	a, player, ok := s.controlAccess(w, r, sess)
	if !ok {
		return
	}
	if _, err := player.Skip(r.Context()); err != nil {
		s.playerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.snapshot(a, queueLimit(r)))
}

func (s *Server) handlePrevious(w http.ResponseWriter, r *http.Request, sess *Session) {
	a, player, ok := s.controlAccess(w, r, sess)
	if !ok {
		return
	}
	if err := player.PlayPrevious(r.Context()); err != nil {
		s.playerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.snapshot(a, queueLimit(r)))
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request, sess *Session) {
	a, player, ok := s.controlAccess(w, r, sess)
	if !ok {
		return
	}
	if !s.stayStopGuard(w, r, a) {
		return
	}
	if err := player.Stop(r.Context()); err != nil {
		s.playerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.snapshot(a, queueLimit(r)))
}

type enqueueRequest struct {
	Query      string `json:"query"`
	Type       string `json:"type"`
	PlaylistID int    `json:"playlist_id"`
	Next       bool   `json:"next"`
	PlayNow    bool   `json:"play_now"`
	Shuffle    bool   `json:"shuffle"`
}

func (s *Server) voiceTarget(a guildAccess) (snowflake.ID, bool) {
	if botCh, ok := s.bot.BotVoiceChannel(a.guildID); ok && s.canControl(a) {
		return botCh, true
	}
	return s.bot.UserVoiceChannel(a.guildID, a.sess.UserID)
}

func (s *Server) handleEnqueue(w http.ResponseWriter, r *http.Request, sess *Session) {
	a, ok := s.guildAccess(w, r, sess)
	if !ok {
		return
	}
	var req enqueueRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" && req.PlaylistID == 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "query or playlist_id is required")
		return
	}

	voiceChannel, ok := s.voiceTarget(a)
	if !ok {
		writeError(w, http.StatusConflict, "not_in_voice", "join a voice channel first")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	meta := musicbot.TrackMeta{Requester: sess.UserID}
	var (
		tracks  []lavalink.Track
		missing int
	)

	if req.PlaylistID != 0 {
		playlist, dbTracks, err := s.bot.Db.GetPlaylist(ctx, sess.UserID, req.PlaylistID)
		if err != nil {
			s.playerError(w, err)
			return
		}
		tracks, missing, err = s.bot.Searcher.LoadPlaylistTracks(ctx, dbTracks)
		if err != nil {
			s.playerError(w, err)
			return
		}
		meta.PlaylistName = playlist.Name
	} else {
		result, err := s.bot.Searcher.Resolve(ctx, req.Query, req.Type)
		if err != nil {
			s.playerError(w, err)
			return
		}
		switch data := result.Data.(type) {
		case lavalink.Track:
			tracks = []lavalink.Track{data}
		case lavalink.Search:
			if len(data) > 0 {
				tracks = []lavalink.Track{data[0]}
			}
		case lavalink.Playlist:
			tracks = data.Tracks
			meta.PlaylistName = data.Info.Name
		}
	}
	if len(tracks) == 0 {
		writeError(w, http.StatusNotFound, "no_results", "that isn't in the library yet")
		return
	}
	if req.Shuffle {
		rand.Shuffle(len(tracks), func(i, j int) { tracks[i], tracks[j] = tracks[j], tracks[i] })
	}
	tracks = musicbot.WithTrackMeta(tracks, meta)

	if _, err := s.bot.Enqueue(ctx, musicbot.EnqueueRequest{
		GuildID:        a.guildID,
		VoiceChannelID: voiceChannel,
		Tracks:         tracks,
		Next:           req.Next,
		PlayNow:        req.PlayNow,
	}); err != nil {
		s.playerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"added":   len(tracks),
		"missing": missing,
		"tracks":  musicbot.NewTrackViews(tracks[:min(len(tracks), 50)]),
		"player":  s.snapshot(a, queueLimit(r)),
	})
}

func (s *Server) handleClearQueue(w http.ResponseWriter, r *http.Request, sess *Session) {
	a, player, ok := s.controlAccess(w, r, sess)
	if !ok {
		return
	}
	player.ClearQueue()
	writeJSON(w, http.StatusOK, s.snapshot(a, queueLimit(r)))
}

func (s *Server) handleRemoveQueue(w http.ResponseWriter, r *http.Request, sess *Session) {
	a, player, ok := s.controlAccess(w, r, sess)
	if !ok {
		return
	}
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid index")
		return
	}
	if _, ok := player.RemoveFromQueue(index); !ok {
		writeError(w, http.StatusNotFound, "not_found", "no track at that position")
		return
	}
	writeJSON(w, http.StatusOK, s.snapshot(a, queueLimit(r)))
}

func (s *Server) handleMoveQueue(w http.ResponseWriter, r *http.Request, sess *Session) {
	a, player, ok := s.controlAccess(w, r, sess)
	if !ok {
		return
	}
	var req struct {
		From int `json:"from"`
		To   int `json:"to"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if !player.MoveInQueue(req.From, req.To) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid positions")
		return
	}
	writeJSON(w, http.StatusOK, s.snapshot(a, queueLimit(r)))
}

func (s *Server) handlePlayRadio(w http.ResponseWriter, r *http.Request, sess *Session) {
	if s.bot.Radio == nil {
		writeError(w, http.StatusNotFound, "radio_disabled", "radio is not enabled")
		return
	}
	a, ok := s.guildAccess(w, r, sess)
	if !ok {
		return
	}
	var req struct {
		Station string `json:"station"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	np, ok := s.bot.Radio.Station(req.Station)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "unknown station")
		return
	}
	voiceChannel, ok := s.voiceTarget(a)
	if !ok {
		writeError(w, http.StatusConflict, "not_in_voice", "join a voice channel first")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	track, err := s.bot.LoadRadioTrack(ctx, np, musicbot.TrackMeta{Requester: sess.UserID})
	if err != nil {
		s.playerError(w, err)
		return
	}
	if _, err := s.bot.Enqueue(ctx, musicbot.EnqueueRequest{
		GuildID:        a.guildID,
		VoiceChannelID: voiceChannel,
		Tracks:         []lavalink.Track{track},
		PlayNow:        true,
	}); err != nil {
		s.playerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.snapshot(a, queueLimit(r)))
}

func (s *Server) handlePlayerEvents(w http.ResponseWriter, r *http.Request, sess *Session) {
	a, ok := s.guildAccess(w, r, sess)
	if !ok {
		return
	}
	rc := http.NewResponseController(w)

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	signal, unsubscribe := s.bot.PlayerManager.Hub.Subscribe(a.guildID)
	defer unsubscribe()

	limit := queueLimit(r)
	send := func() error {
		data, err := json.Marshal(s.snapshot(a, limit))
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "event: player\ndata: %s\n\n", data); err != nil {
			return err
		}
		return rc.Flush()
	}
	if err := send(); err != nil {
		return
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	expiry := time.NewTimer(time.Until(sess.ExpiresAt))
	defer expiry.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-expiry.C:
			_, _ = fmt.Fprint(w, "event: session_expired\ndata: {}\n\n")
			_ = rc.Flush()
			return
		case <-signal:

			time.Sleep(150 * time.Millisecond)
			select {
			case <-signal:
			default:
			}
			if err := send(); err != nil {
				return
			}
		case <-heartbeat.C:
			if err := send(); err != nil {
				return
			}
		}
	}
}
