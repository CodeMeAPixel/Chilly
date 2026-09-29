package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgolink/v3/disgolink"
)

const (
	statusSampleEvery    = 30 * time.Second
	statusBucketSize     = 15 * time.Minute
	statusBucketCount    = 96
	statusDays           = 90
	statusIncidentWindow = 14 * 24 * time.Hour
	incidentAfterFails   = 2
	collectingMinSamples = 60
	dailyRefreshEvery    = 5 * time.Minute
	libraryStaleAfter    = 45 * time.Minute
)

type StatusStore interface {
	AddStatusSamples(ctx context.Context, samples []musicbot.StatusSample) error
	StatusBuckets(ctx context.Context, since time.Time) ([]musicbot.StatusBucket, error)
	StatusDaily(ctx context.Context, since time.Time) ([]musicbot.StatusBucket, error)
	FirstStatusSample(ctx context.Context) (time.Time, error)
	PruneStatus(ctx context.Context, before time.Time) error
	CreateIncident(ctx context.Context, in *musicbot.Incident) error
	ResolveIncident(ctx context.Context, id int64, at time.Time) error
	OpenAutoIncidents(ctx context.Context) ([]musicbot.Incident, error)
	StatusIncidents(ctx context.Context, since time.Time, limit int) ([]musicbot.Incident, error)
}

type statusBucket struct {
	start time.Time
	up    int
	total int
}

type componentHistory struct {
	id         string
	name       string
	group      string
	ok         bool
	detail     string
	buckets    []statusBucket
	daily      map[int64]statusBucket
	fails      int
	incidentID int64
	opening    bool
}

type incidentEvent struct {
	open       bool
	component  *componentHistory
	incidentID int64
	name       string
	group      string
	detail     string
	at         time.Time
}

type statusTracker struct {
	mu         sync.RWMutex
	components map[string]*componentHistory
	order      []string
	since      time.Time
	checkedAt  time.Time

	preload       map[string][]statusBucket
	openIncidents map[string]int64
	incidents     []musicbot.Incident
	lastDaily     time.Time
	lastPrune     time.Time
	firstRound    bool
}

func newStatusTracker() *statusTracker {
	return &statusTracker{
		components:    make(map[string]*componentHistory),
		since:         time.Now(),
		preload:       make(map[string][]statusBucket),
		openIncidents: make(map[string]int64),
		firstRound:    true,
	}
}

func (t *statusTracker) record(id, name, group string, ok bool, detail string, now time.Time) *incidentEvent {
	c, exists := t.components[id]
	if !exists {
		c = &componentHistory{id: id, name: name, group: group, buckets: t.preload[id], daily: make(map[int64]statusBucket)}
		delete(t.preload, id)
		if incidentID, open := t.openIncidents[id]; open {
			c.incidentID = incidentID
			c.fails = incidentAfterFails
			delete(t.openIncidents, id)
		}
		t.components[id] = c
		t.order = append(t.order, id)
	}
	c.name, c.group, c.ok, c.detail = name, group, ok, detail

	start := now.Truncate(statusBucketSize)
	if n := len(c.buckets); n == 0 || !c.buckets[n-1].start.Equal(start) {
		c.buckets = append(c.buckets, statusBucket{start: start})
	}
	last := &c.buckets[len(c.buckets)-1]
	last.total++
	if ok {
		last.up++
	}
	cutoff := start.Add(-statusBucketSize * (statusBucketCount - 1))
	for len(c.buckets) > 0 && c.buckets[0].start.Before(cutoff) {
		c.buckets = c.buckets[1:]
	}

	if ok {
		c.fails = 0
		if c.incidentID != 0 {
			ev := &incidentEvent{component: c, incidentID: c.incidentID, at: now}
			c.incidentID = 0
			return ev
		}
		return nil
	}
	c.fails++
	if c.fails >= incidentAfterFails && c.incidentID == 0 && !c.opening {
		c.opening = true
		return &incidentEvent{open: true, component: c, name: name, group: group, detail: detail, at: now}
	}
	return nil
}

func (s *Server) statusStore() StatusStore {
	if s.bot == nil || s.bot.Db == nil {
		return nil
	}
	return s.bot.Db
}

func (s *Server) RunStatus(ctx context.Context) {
	s.loadStatus(ctx)
	s.sampleStatus(ctx)
	ticker := time.NewTicker(statusSampleEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sampleStatus(ctx)
		}
	}
}

func (s *Server) loadStatus(ctx context.Context) {
	store := s.statusStore()
	if store == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	t := s.status

	now := time.Now()
	buckets, err := store.StatusBuckets(ctx, now.Truncate(statusBucketSize).Add(-statusBucketSize*(statusBucketCount-1)))
	if err != nil {
		slog.Warn("failed to load status history", slog.Any("error", err))
	}
	first, _ := store.FirstStatusSample(ctx)
	open, err := store.OpenAutoIncidents(ctx)
	if err != nil {
		slog.Warn("failed to load open incidents", slog.Any("error", err))
	}

	t.mu.Lock()
	for _, b := range buckets {
		t.preload[b.Component] = append(t.preload[b.Component], statusBucket{start: b.Start, up: b.Up, total: b.Total})
	}
	if !first.IsZero() && first.Before(t.since) {
		t.since = first
	}
	for _, in := range open {
		t.openIncidents[in.Component] = in.ID
	}
	t.mu.Unlock()
}

type statusObservation struct {
	id, name, group string
	ok              bool
	detail          string
}

func gatewayHealthy(status gateway.Status, latency time.Duration) bool {
	switch status {
	case gateway.StatusReady:
		return true
	case gateway.StatusResuming:
		return latency > 0
	default:
		return false
	}
}

func (s *Server) observe(ctx context.Context) []statusObservation {
	var obs []statusObservation

	gatewayOK, gatewayDetail := false, "disconnected"
	if s.bot.Client.HasGateway() {
		gw := s.bot.Client.Gateway()
		if gatewayHealthy(gw.Status(), gw.Latency()) {
			gatewayOK = true
			gatewayDetail = gw.Latency().Round(time.Millisecond).String() + " latency"
		} else {
			gatewayDetail = gw.Status().String()
		}
	}
	obs = append(obs, statusObservation{"discord", "Discord connection", "Core", gatewayOK, gatewayDetail})

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	dbErr := s.bot.Db.Ping(pingCtx)
	cancel()
	dbDetail := "responding"
	if dbErr != nil {
		dbDetail = "unreachable"
	}
	obs = append(obs, statusObservation{"database", "Database", "Core", dbErr == nil, dbDetail})

	if s.bot.Radio != nil {
		for _, np := range s.bot.Radio.Stations() {
			detail := "off air"
			if np.IsOnline {
				detail = fmt.Sprintf("%d listening", np.Listeners.Current)
				if np.Live.IsLive {
					detail += " · live DJ"
				}
			}
			obs = append(obs, statusObservation{"station:" + np.Station.Shortcode, np.Station.Name, "Stations", np.IsOnline, detail})
		}

		healthy, _ := s.bot.Radio.Healthy()
		detail := "responding"
		if !healthy {
			detail = "unreachable"
		}
		obs = append(obs, statusObservation{"azuracast", "AzuraCast", "Services", healthy, detail})
	}

	if lib := s.bot.Searcher.Library(); lib != nil {
		count, lastSync, _ := lib.Status()
		ok := !lastSync.IsZero() && time.Since(lastSync) < libraryStaleAfter
		detail := "not synced yet"
		switch {
		case ok:
			detail = fmt.Sprintf("%d songs", count)
		case !lastSync.IsZero():
			detail = "sync failing"
		}
		obs = append(obs, statusObservation{"library", "Music library", "Services", ok, detail})
	}

	for _, node := range s.nodes() {
		ok := node.Status == string(disgolink.StatusConnected)
		obs = append(obs, statusObservation{"node:" + node.Name, node.Name, "Audio nodes", ok, statusLabel(node.Status)})
	}
	return obs
}

func (s *Server) sampleStatus(ctx context.Context) {
	now := time.Now()
	obs := s.observe(ctx)

	t := s.status
	samples := make([]musicbot.StatusSample, 0, len(obs))
	var events []*incidentEvent
	t.mu.Lock()
	t.checkedAt = now
	for _, o := range obs {
		if ev := t.record(o.id, o.name, o.group, o.ok, o.detail, now); ev != nil {
			events = append(events, ev)
		}
		samples = append(samples, musicbot.StatusSample{Component: o.id, Start: now.Truncate(statusBucketSize), Up: o.ok})
	}
	var orphaned []int64
	if t.firstRound {
		for _, id := range t.openIncidents {
			orphaned = append(orphaned, id)
		}
		t.openIncidents = make(map[string]int64)
		t.firstRound = false
	}
	refreshDaily := now.Sub(t.lastDaily) >= dailyRefreshEvery
	prune := now.Sub(t.lastPrune) >= time.Hour
	t.mu.Unlock()

	store := s.statusStore()
	if store == nil {
		return
	}
	if err := store.AddStatusSamples(ctx, samples); err != nil {
		slog.Warn("failed to save status samples", slog.Any("error", err))
	}
	for _, id := range orphaned {
		if err := store.ResolveIncident(ctx, id, now); err != nil {
			slog.Debug("failed to resolve orphaned incident", slog.Int64("id", id), slog.Any("error", err))
		}
	}
	for _, ev := range events {
		s.handleIncidentEvent(ctx, store, ev)
	}
	if refreshDaily {
		s.refreshDaily(ctx, store, now)
	}
	if prune {
		if err := store.PruneStatus(ctx, now.Add(-(statusDays+1)*24*time.Hour)); err != nil {
			slog.Warn("failed to prune status history", slog.Any("error", err))
		}
		t.mu.Lock()
		t.lastPrune = now
		t.mu.Unlock()
	}

	incidents, err := store.StatusIncidents(ctx, now.Add(-statusIncidentWindow), 30)
	if err != nil {
		slog.Warn("failed to load incidents", slog.Any("error", err))
		return
	}
	t.mu.Lock()
	t.incidents = incidents
	t.mu.Unlock()
}

func (s *Server) handleIncidentEvent(ctx context.Context, store StatusStore, ev *incidentEvent) {
	t := s.status
	if !ev.open {
		if err := store.ResolveIncident(ctx, ev.incidentID, ev.at); err != nil {
			slog.Warn("failed to resolve incident", slog.Int64("id", ev.incidentID), slog.Any("error", err))
		}
		return
	}

	title := ev.name + " unavailable"
	if ev.group == "Stations" {
		title = ev.name + " off air"
	}
	in := musicbot.Incident{
		Component:     ev.component.id,
		ComponentName: ev.name,
		Kind:          musicbot.IncidentAuto,
		Title:         title,
		Message:       ev.detail,
		StartedAt:     ev.at.Add(-statusSampleEvery * (incidentAfterFails - 1)),
	}
	err := store.CreateIncident(ctx, &in)

	t.mu.Lock()
	ev.component.opening = false
	recovered := ev.component.ok
	if err == nil && !recovered {
		ev.component.incidentID = in.ID
	}
	t.mu.Unlock()

	if err != nil {
		slog.Warn("failed to open incident", slog.String("component", ev.component.id), slog.Any("error", err))
		return
	}
	slog.Warn("status incident opened", slog.String("component", ev.component.id), slog.String("title", title))
	if recovered {
		_ = store.ResolveIncident(ctx, in.ID, time.Now())
	}
}

func (s *Server) refreshDaily(ctx context.Context, store StatusStore, now time.Time) {
	daily, err := store.StatusDaily(ctx, dayStart(now).AddDate(0, 0, -(statusDays-1)))
	if err != nil {
		slog.Warn("failed to load daily status", slog.Any("error", err))
		return
	}
	byComponent := make(map[string]map[int64]statusBucket)
	for _, b := range daily {
		m, ok := byComponent[b.Component]
		if !ok {
			m = make(map[int64]statusBucket)
			byComponent[b.Component] = m
		}
		m[dayStart(b.Start).Unix()] = statusBucket{start: b.Start, up: b.Up, total: b.Total}
	}

	t := s.status
	t.mu.Lock()
	for id, c := range t.components {
		if m, ok := byComponent[id]; ok {
			c.daily = m
		}
	}
	t.lastDaily = now
	t.mu.Unlock()
}

func dayStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func statusLabel(nodeStatus string) string {
	switch disgolink.Status(nodeStatus) {
	case disgolink.StatusConnected:
		return "connected"
	case disgolink.StatusConnecting:
		return "connecting"
	case disgolink.StatusReconnecting:
		return "reconnecting"
	default:
		return "offline"
	}
}

func (s *Server) nodes() []musicbot.NodeInfo {
	if s.bot.Nodes != nil {
		return s.bot.Nodes.Nodes()
	}
	return musicbot.Nodes(s.bot.Lavalink)
}

type statusPoint struct {
	Start  time.Time `json:"start"`
	Uptime *float64  `json:"uptime"`
}

type statusComponent struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Group      string        `json:"group"`
	Status     string        `json:"status"`
	Detail     string        `json:"detail"`
	Uptime     *float64      `json:"uptime"`
	Uptime24h  *float64      `json:"uptime_24h"`
	Collecting bool          `json:"collecting"`
	History    []statusPoint `json:"history"`
	Daily      []statusPoint `json:"daily"`
}

type incidentView struct {
	musicbot.Incident
	DurationSeconds int64 `json:"duration_seconds"`
	Ongoing         bool  `json:"ongoing"`
}

type statusResponse struct {
	Status        string              `json:"status"`
	CheckedAt     time.Time           `json:"checked_at"`
	TrackingSince time.Time           `json:"tracking_since"`
	BucketMinutes int                 `json:"bucket_minutes"`
	HistoryDays   int                 `json:"history_days"`
	Components    []statusComponent   `json:"components"`
	Incidents     []incidentView      `json:"incidents"`
	Notices       []incidentView      `json:"notices"`
	Nodes         []musicbot.NodeInfo `json:"nodes"`
	Bot           map[string]any      `json:"bot"`
}

func percent(up, total int) *float64 {
	if total == 0 {
		return nil
	}
	v := float64(up) / float64(total) * 100
	return &v
}

func (t *statusTracker) snapshot(now time.Time) ([]statusComponent, string) {
	current := now.Truncate(statusBucketSize)
	first := current.Add(-statusBucketSize * (statusBucketCount - 1))
	today := dayStart(now)
	firstDay := today.AddDate(0, 0, -(statusDays - 1))

	components := make([]statusComponent, 0, len(t.order))
	coreDown, nodesUp, nodesTotal, anyDown := false, 0, 0, false
	for _, id := range t.order {
		c := t.components[id]

		byStart := make(map[int64]statusBucket, len(c.buckets))
		up24, total24 := 0, 0
		for _, b := range c.buckets {
			byStart[b.start.Unix()] = b
			up24 += b.up
			total24 += b.total
		}
		history := make([]statusPoint, statusBucketCount)
		for i := range history {
			start := first.Add(statusBucketSize * time.Duration(i))
			history[i].Start = start
			if b, ok := byStart[start.Unix()]; ok && b.total > 0 {
				v := float64(b.up) / float64(b.total)
				history[i].Uptime = &v
			}
		}

		daily := make([]statusPoint, statusDays)
		upAll, totalAll := 0, 0
		for i := range daily {
			day := firstDay.AddDate(0, 0, i)
			daily[i].Start = day
			if b, ok := c.daily[day.Unix()]; ok && b.total > 0 {
				v := float64(b.up) / float64(b.total)
				daily[i].Uptime = &v
				upAll += b.up
				totalAll += b.total
			}
		}
		if totalAll < total24 {
			upAll, totalAll = up24, total24
		}

		component := statusComponent{
			ID: c.id, Name: c.name, Group: c.group, Detail: c.detail,
			History: history, Daily: daily, Status: "operational",
			Uptime24h:  percent(up24, total24),
			Collecting: totalAll < collectingMinSamples,
		}
		if !component.Collecting {
			component.Uptime = percent(upAll, totalAll)
		}
		if !c.ok {
			component.Status = "down"
			anyDown = true
			if c.group == "Core" {
				coreDown = true
			}
		}
		if c.group == "Audio nodes" {
			nodesTotal++
			if c.ok {
				nodesUp++
			}
		}
		components = append(components, component)
	}

	switch {
	case coreDown || (nodesTotal > 0 && nodesUp == 0):
		return components, "outage"
	case anyDown:
		return components, "degraded"
	default:
		return components, "operational"
	}
}

func incidentViews(incidents []musicbot.Incident, now time.Time) (history, notices []incidentView) {
	history, notices = make([]incidentView, 0), make([]incidentView, 0)
	for _, in := range incidents {
		end := now
		if in.ResolvedAt != nil {
			end = *in.ResolvedAt
		}
		view := incidentView{Incident: in, DurationSeconds: int64(end.Sub(in.StartedAt).Seconds()), Ongoing: in.ResolvedAt == nil}
		if in.Kind != musicbot.IncidentAuto && view.Ongoing {
			notices = append(notices, view)
			continue
		}
		history = append(history, view)
	}
	return history, notices
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	t := s.status
	t.mu.RLock()
	components, overall := t.snapshot(now)
	history, notices := incidentViews(t.incidents, now)
	resp := statusResponse{
		Status:        overall,
		CheckedAt:     t.checkedAt,
		TrackingSince: t.since,
		BucketMinutes: int(statusBucketSize.Minutes()),
		HistoryDays:   statusDays,
		Components:    components,
		Incidents:     history,
		Notices:       notices,
	}
	t.mu.RUnlock()

	players, playing := s.bot.PlayerManager.Count()
	bot := map[string]any{
		"guilds":         s.bot.Client.Caches().GuildsLen(),
		"players":        players,
		"playing":        playing,
		"uptime_seconds": int64(time.Since(s.bot.StartedAt).Seconds()),
	}
	if s.bot.Client.HasGateway() {
		bot["gateway_latency_ms"] = s.bot.Client.Gateway().Latency().Milliseconds()
	}
	if s.bot.Radio != nil {
		listeners := 0
		for _, np := range s.bot.Radio.Stations() {
			listeners += np.Listeners.Current
		}
		bot["listeners"] = listeners
	}
	resp.Bot = bot
	resp.Nodes = s.nodes()

	w.Header().Set("Cache-Control", "public, max-age=10")
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) reloadIncidents(ctx context.Context) {
	store := s.statusStore()
	if store == nil {
		return
	}
	incidents, err := store.StatusIncidents(ctx, time.Now().Add(-statusIncidentWindow), 30)
	if err != nil {
		slog.Warn("failed to reload incidents", slog.Any("error", err))
		return
	}
	s.status.mu.Lock()
	s.status.incidents = incidents
	s.status.mu.Unlock()
}

func (s *Server) handleAdminCreateNotice(w http.ResponseWriter, r *http.Request, sess *Session) {
	var req struct {
		Kind    string `json:"kind"`
		Title   string `json:"title"`
		Message string `json:"message"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Kind != musicbot.IncidentNotice && req.Kind != musicbot.IncidentMaintenance {
		writeError(w, http.StatusBadRequest, "bad_request", "kind must be notice or maintenance")
		return
	}
	if req.Title == "" || len(req.Title) > 200 || len(req.Message) > 2000 {
		writeError(w, http.StatusBadRequest, "bad_request", "title is required (200 characters max) and messages are limited to 2000 characters")
		return
	}
	in := musicbot.Incident{Kind: req.Kind, Title: req.Title, Message: strings.TrimSpace(req.Message), CreatedBy: sess.UserID}
	if err := s.bot.Db.CreateIncident(r.Context(), &in); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "database error")
		return
	}
	s.reloadIncidents(r.Context())
	writeJSON(w, http.StatusCreated, map[string]any{"incident": in})
}

func incidentID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("incidentID"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid incident id")
		return 0, false
	}
	return id, true
}

func (s *Server) handleAdminResolveIncident(w http.ResponseWriter, r *http.Request, _ *Session) {
	id, ok := incidentID(w, r)
	if !ok {
		return
	}
	if err := s.bot.Db.ResolveIncident(r.Context(), id, time.Now()); err != nil {
		if errors.Is(err, musicbot.ErrIncidentNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "no open incident with that id")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "database error")
		return
	}
	s.reloadIncidents(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminDeleteIncident(w http.ResponseWriter, r *http.Request, _ *Session) {
	id, ok := incidentID(w, r)
	if !ok {
		return
	}
	if err := s.bot.Db.DeleteIncident(r.Context(), id); err != nil {
		if errors.Is(err, musicbot.ErrIncidentNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "incident not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "database error")
		return
	}
	s.reloadIncidents(r.Context())
	w.WriteHeader(http.StatusNoContent)
}
