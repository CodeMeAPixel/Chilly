package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgolink/v3/disgolink"
)

const (
	statusSampleEvery = 30 * time.Second
	statusBucketSize  = 15 * time.Minute
	statusBucketCount = 96
)

type statusBucket struct {
	start time.Time
	up    int
	total int
}

type componentHistory struct {
	id      string
	name    string
	group   string
	ok      bool
	detail  string
	buckets []statusBucket
}

type statusTracker struct {
	mu         sync.RWMutex
	components map[string]*componentHistory
	order      []string
	since      time.Time
	checkedAt  time.Time
}

func newStatusTracker() *statusTracker {
	return &statusTracker{components: make(map[string]*componentHistory), since: time.Now()}
}

func (t *statusTracker) record(id, name, group string, ok bool, detail string, now time.Time) {
	c, exists := t.components[id]
	if !exists {
		c = &componentHistory{id: id, name: name, group: group}
		t.components[id] = c
		t.order = append(t.order, id)
	}
	c.ok, c.detail = ok, detail

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
}

func (s *Server) RunStatus(ctx context.Context) {
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

func (s *Server) sampleStatus(ctx context.Context) {
	now := time.Now()

	gatewayOK, gatewayDetail := false, "disconnected"
	if s.bot.Client.HasGateway() {
		gw := s.bot.Client.Gateway()
		if gw.Status() == gateway.StatusReady {
			gatewayOK = true
			gatewayDetail = gw.Latency().Round(time.Millisecond).String() + " latency"
		} else {
			gatewayDetail = gw.Status().String()
		}
	}

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	dbErr := s.bot.Db.Ping(pingCtx)
	cancel()
	dbDetail := "responding"
	if dbErr != nil {
		dbDetail = "unreachable"
	}

	nodes := s.nodes()

	t := s.status
	t.mu.Lock()
	defer t.mu.Unlock()
	t.checkedAt = now
	t.record("discord", "Discord connection", "Core", gatewayOK, gatewayDetail, now)
	t.record("database", "Database", "Core", dbErr == nil, dbDetail, now)
	for _, node := range nodes {
		ok := node.Status == string(disgolink.StatusConnected)
		t.record("node:"+node.Name, node.Name, "Audio nodes", ok, statusLabel(node.Status), now)
	}
	if s.bot.Radio != nil {
		healthy, _ := s.bot.Radio.Healthy()
		detail := "streaming"
		if !healthy {
			detail = "unreachable"
		}
		t.record("radio", "Radio", "Services", healthy, detail, now)
	}
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
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Group   string        `json:"group"`
	Status  string        `json:"status"`
	Detail  string        `json:"detail"`
	Uptime  *float64      `json:"uptime"`
	History []statusPoint `json:"history"`
}

type statusResponse struct {
	Status        string              `json:"status"`
	CheckedAt     time.Time           `json:"checked_at"`
	TrackingSince time.Time           `json:"tracking_since"`
	BucketMinutes int                 `json:"bucket_minutes"`
	Components    []statusComponent   `json:"components"`
	Nodes         []musicbot.NodeInfo `json:"nodes"`
	Bot           map[string]any      `json:"bot"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	t := s.status
	t.mu.RLock()
	current := time.Now().Truncate(statusBucketSize)
	first := current.Add(-statusBucketSize * (statusBucketCount - 1))

	components := make([]statusComponent, 0, len(t.order))
	coreDown, nodesUp, nodesTotal, anyDown := false, 0, 0, false
	for _, id := range t.order {
		c := t.components[id]
		byStart := make(map[int64]statusBucket, len(c.buckets))
		up, total := 0, 0
		for _, b := range c.buckets {
			byStart[b.start.Unix()] = b
			up += b.up
			total += b.total
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
		component := statusComponent{
			ID: c.id, Name: c.name, Group: c.group, Detail: c.detail, History: history,
			Status: "operational",
		}
		if total > 0 {
			v := float64(up) / float64(total) * 100
			component.Uptime = &v
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
	resp := statusResponse{
		CheckedAt:     t.checkedAt,
		TrackingSince: t.since,
		BucketMinutes: int(statusBucketSize.Minutes()),
		Components:    components,
	}
	t.mu.RUnlock()

	switch {
	case coreDown || (nodesTotal > 0 && nodesUp == 0):
		resp.Status = "outage"
	case anyDown:
		resp.Status = "degraded"
	default:
		resp.Status = "operational"
	}

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
	resp.Bot = bot
	resp.Nodes = s.nodes()

	w.Header().Set("Cache-Control", "public, max-age=10")
	writeJSON(w, http.StatusOK, resp)
}
