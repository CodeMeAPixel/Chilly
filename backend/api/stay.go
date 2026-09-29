package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"
)

type stayView struct {
	musicbot.StaySetting
	StationName string              `json:"station_name"`
	Health      musicbot.StayHealth `json:"health"`
}

func (s *Server) stayView(setting musicbot.StaySetting) stayView {
	view := stayView{StaySetting: setting, StationName: setting.Station, Health: s.bot.Stays.Health(setting.GuildID)}
	if s.bot.Radio != nil {
		if np, ok := s.bot.Radio.Station(setting.Station); ok {
			view.StationName = np.Station.Name
		}
	}
	return view
}

func (s *Server) handleGetStay(w http.ResponseWriter, r *http.Request, sess *Session) {
	a, ok := s.guildAccess(w, r, sess)
	if !ok {
		return
	}
	setting, ok := s.bot.Stays.Get(a.guildID)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"stay": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stay": s.stayView(setting)})
}

func (s *Server) handleEnableStay(w http.ResponseWriter, r *http.Request, sess *Session) {
	if s.bot.Radio == nil {
		writeError(w, http.StatusNotFound, "radio_disabled", "radio is not enabled")
		return
	}
	a, ok := s.guildAccess(w, r, sess)
	if !ok {
		return
	}
	if !a.canManage {
		writeError(w, http.StatusForbidden, "forbidden", "you need the Manage Server permission to set up 24/7 radio")
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
	if !np.IsOnline {
		writeError(w, http.StatusConflict, "station_offline", np.Station.Name+" is offline right now")
		return
	}
	voiceChannel, ok := s.voiceTarget(a)
	if !ok {
		writeError(w, http.StatusConflict, "not_in_voice", "join the voice channel Chilly should stay in first")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	setting := musicbot.StaySetting{
		GuildID:        a.guildID,
		VoiceChannelID: voiceChannel,
		Station:        np.Station.Shortcode,
		EnabledBy:      sess.UserID,
	}
	if p, ok := s.bot.PlayerManager.GetPlayer(a.guildID); ok {
		setting.TextChannelID = p.ChannelID()
	}
	if _, err := s.bot.EnableStay(ctx, setting); err != nil {
		s.playerError(w, err)
		return
	}
	saved, _ := s.bot.Stays.Get(a.guildID)
	writeJSON(w, http.StatusOK, map[string]any{"stay": s.stayView(saved)})
}

func (s *Server) handleDisableStay(w http.ResponseWriter, r *http.Request, sess *Session) {
	a, ok := s.guildAccess(w, r, sess)
	if !ok {
		return
	}
	if !a.canManage {
		writeError(w, http.StatusForbidden, "forbidden", "you need the Manage Server permission to change 24/7 radio")
		return
	}
	if _, err := s.bot.DisableStay(r.Context(), a.guildID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "failed to turn off 24/7 radio")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stay": nil})
}

func (s *Server) stayStopGuard(w http.ResponseWriter, r *http.Request, a guildAccess) bool {
	if !s.bot.Stays.Enabled(a.guildID) {
		return true
	}
	if !a.canManage {
		writeError(w, http.StatusForbidden, "stay_enabled", musicbot.ErrStayEnabled.Error()+"; someone with Manage Server can turn it off")
		return false
	}
	if _, err := s.bot.DisableStay(r.Context(), a.guildID); err != nil && !errors.Is(err, context.Canceled) {
		writeError(w, http.StatusInternalServerError, "internal", "failed to turn off 24/7 radio")
		return false
	}
	return true
}

func (s *Server) handleStepStation(w http.ResponseWriter, r *http.Request, sess *Session) {
	a, _, ok := s.controlAccess(w, r, sess)
	if !ok {
		return
	}
	var req struct {
		Direction string `json:"direction"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	step := 1
	switch req.Direction {
	case "next":
	case "previous":
		step = -1
	default:
		writeError(w, http.StatusBadRequest, "bad_request", "direction must be next or previous")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if _, err := s.bot.StepStation(ctx, a.guildID, sess.UserID, step, a.canManage); err != nil {
		switch {
		case errors.Is(err, musicbot.ErrNotRadio), errors.Is(err, musicbot.ErrNoOtherStation):
			writeError(w, http.StatusConflict, "not_radio", err.Error())
		default:
			s.playerError(w, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, s.snapshot(a, queueLimit(r)))
}
