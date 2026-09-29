package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/CodeMeAPixel/Chilly/azuracast"
	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgolink/v3/disgolink"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	dbOK := s.bot.Db.Ping(ctx) == nil
	nodes := musicbot.Nodes(s.bot.Lavalink)
	connected := 0
	for _, n := range nodes {
		if n.Status == string(disgolink.StatusConnected) {
			connected++
		}
	}

	resp := map[string]any{
		"status":          "ok",
		"database":        dbOK,
		"nodes":           nodes,
		"nodes_connected": connected,
		"uptime_seconds":  int64(time.Since(s.bot.StartedAt).Seconds()),
	}
	if s.bot.Radio != nil {
		healthy, lastPoll := s.bot.Radio.Healthy()
		resp["radio"] = map[string]any{"healthy": healthy, "last_poll": lastPoll}
	}

	status := http.StatusOK
	if !dbOK || connected == 0 {
		resp["status"] = "degraded"
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, resp)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	players, playing := s.bot.PlayerManager.Count()
	resp := map[string]any{
		"guilds":         s.bot.Client.Caches().GuildsLen(),
		"players":        players,
		"playing":        playing,
		"uptime_seconds": int64(time.Since(s.bot.StartedAt).Seconds()),
	}
	if s.bot.Radio != nil {
		resp["radio_stations"] = len(s.bot.Radio.Stations())
	}
	botInfo := map[string]any{"id": s.bot.Client.ApplicationID().String()}
	if self, ok := s.bot.Client.Caches().SelfUser(); ok {
		botInfo["username"] = self.Username
		botInfo["avatar_url"] = self.EffectiveAvatarURL()
	}
	resp["bot"] = botInfo
	writeJSON(w, http.StatusOK, resp)
}

type libraryResult struct {
	musicbot.TrackView

	Value     string   `json:"value"`
	Album     string   `json:"album,omitempty"`
	Station   string   `json:"station"`
	Playlists []string `json:"playlists,omitempty"`
}

func newLibraryResult(t musicbot.LibraryTrack) libraryResult {
	return libraryResult{
		TrackView: musicbot.TrackView{
			Title:      t.Title,
			Author:     t.Artist,
			ArtworkURL: t.ArtURL,
			SourceName: musicbot.LibrarySource,
			Identifier: t.Key(),
			LengthMs:   t.LengthMs,
		},
		Value:     t.Key(),
		Album:     t.Album,
		Station:   t.Station,
		Playlists: t.Playlists,
	}
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request, _ *Session) {
	q := r.URL.Query()
	query := q.Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "q is required")
		return
	}
	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil || limit <= 0 || limit > 50 {
		limit = 20
	}

	tracks := s.bot.Searcher.Search(query, limit)
	results := make([]libraryResult, len(tracks))
	for i, t := range tracks {
		results[i] = newLibraryResult(t)
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

type stationView struct {
	ID          int                      `json:"id"`
	Shortcode   string                   `json:"shortcode"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Online      bool                     `json:"online"`
	Listeners   int                      `json:"listeners"`
	PlayerURL   string                   `json:"public_player_url"`
	StreamURL   string                   `json:"stream_url"`
	Live        azuracast.Live           `json:"live"`
	NowPlaying  *azuracast.CurrentSong   `json:"now_playing"`
	PlayingNext *azuracast.StationQueue  `json:"playing_next,omitempty"`
	History     []azuracast.HistoryEntry `json:"song_history,omitempty"`
}

func newStationView(np azuracast.NowPlaying, withHistory bool) stationView {
	v := stationView{
		ID:          np.Station.ID,
		Shortcode:   np.Station.Shortcode,
		Name:        np.Station.Name,
		Description: np.Station.Description,
		Online:      np.IsOnline,
		Listeners:   np.Listeners.Current,
		PlayerURL:   np.Station.PublicPlayerURL,
		StreamURL:   np.Station.ListenURL,
		Live:        np.Live,
		NowPlaying:  np.NowPlaying,
		PlayingNext: np.PlayingNext,
	}
	if withHistory {
		v.History = np.SongHistory
	}
	return v
}

func (s *Server) handleRadioStations(w http.ResponseWriter, r *http.Request) {
	if s.bot.Radio == nil {
		writeError(w, http.StatusNotFound, "radio_disabled", "radio is not enabled")
		return
	}
	stations := s.bot.Radio.Stations()
	views := make([]stationView, len(stations))
	for i, np := range stations {
		views[i] = newStationView(np, false)
	}
	writeJSON(w, http.StatusOK, map[string]any{"stations": views})
}

func (s *Server) handleRadioStation(w http.ResponseWriter, r *http.Request) {
	if s.bot.Radio == nil {
		writeError(w, http.StatusNotFound, "radio_disabled", "radio is not enabled")
		return
	}
	np, ok := s.bot.Radio.Station(r.PathValue("station"))
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "unknown station")
		return
	}
	writeJSON(w, http.StatusOK, newStationView(np, true))
}
