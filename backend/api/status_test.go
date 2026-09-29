package api

import (
	"context"
	"testing"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"
)

var _ StatusStore = (*musicbot.DB)(nil)

func TestRecordOpensIncidentAfterRepeatedFailures(t *testing.T) {
	tr := newStatusTracker()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	if ev := tr.record("node:a", "node-a", "Audio nodes", false, "offline", now); ev != nil {
		t.Fatal("a single failed check must not open an incident")
	}
	ev := tr.record("node:a", "node-a", "Audio nodes", false, "offline", now.Add(statusSampleEvery))
	if ev == nil || !ev.open || ev.name != "node-a" {
		t.Fatalf("expected an open event, got %+v", ev)
	}
	if again := tr.record("node:a", "node-a", "Audio nodes", false, "offline", now.Add(2*statusSampleEvery)); again != nil {
		t.Error("an incident that is already opening must not be opened twice")
	}

	c := tr.components["node:a"]
	c.opening, c.incidentID = false, 42
	resolved := tr.record("node:a", "node-a", "Audio nodes", true, "connected", now.Add(3*statusSampleEvery))
	if resolved == nil || resolved.open || resolved.incidentID != 42 {
		t.Fatalf("expected a resolve event for incident 42, got %+v", resolved)
	}
	if c.incidentID != 0 || c.fails != 0 {
		t.Errorf("component should be reset after recovery: %+v", c)
	}
	if ev := tr.record("node:a", "node-a", "Audio nodes", true, "connected", now.Add(4*statusSampleEvery)); ev != nil {
		t.Error("healthy checks must not produce events")
	}
}

func TestRecordRestoresOpenIncidentsAndPreloadedHistory(t *testing.T) {
	tr := newStatusTracker()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tr.openIncidents["database"] = 7
	tr.preload["database"] = []statusBucket{{start: now.Truncate(statusBucketSize).Add(-statusBucketSize), up: 30, total: 30}}

	ev := tr.record("database", "Database", "Core", true, "responding", now)
	if ev == nil || ev.incidentID != 7 {
		t.Fatalf("a recovered component should resolve its restored incident, got %+v", ev)
	}
	if got := len(tr.components["database"].buckets); got != 2 {
		t.Errorf("preloaded history should be kept, got %d buckets", got)
	}
}

func TestSnapshotCollectingAndDailyUptime(t *testing.T) {
	tr := newStatusTracker()
	now := time.Now()
	tr.record("database", "Database", "Core", true, "responding", now)

	components, overall := tr.snapshot(now)
	if overall != "operational" || len(components) != 1 {
		t.Fatalf("unexpected snapshot %v %d", overall, len(components))
	}
	c := components[0]
	if !c.Collecting || c.Uptime != nil || c.Uptime24h == nil {
		t.Errorf("a new component should be collecting data: %+v", c)
	}
	if len(c.History) != statusBucketCount || len(c.Daily) != statusDays {
		t.Errorf("unexpected history lengths %d %d", len(c.History), len(c.Daily))
	}

	tr.components["database"].daily[dayStart(now).Unix()] = statusBucket{start: dayStart(now), up: 95, total: 100}
	components, _ = tr.snapshot(now)
	if c := components[0]; c.Collecting || c.Uptime == nil || *c.Uptime != 95 {
		t.Errorf("daily totals should drive uptime once there is enough data: %+v", c)
	}

	tr.record("node:a", "node-a", "Audio nodes", false, "offline", now)
	if _, overall := tr.snapshot(now); overall != "outage" {
		t.Errorf("all nodes down should be an outage, got %s", overall)
	}
	tr.record("node:b", "node-b", "Audio nodes", true, "connected", now)
	if _, overall := tr.snapshot(now); overall != "degraded" {
		t.Errorf("one node down should be degraded, got %s", overall)
	}
}

type fakeStatusStore struct {
	created  []musicbot.Incident
	resolved []int64
}

func (f *fakeStatusStore) AddStatusSamples(context.Context, []musicbot.StatusSample) error {
	return nil
}
func (f *fakeStatusStore) StatusBuckets(context.Context, time.Time) ([]musicbot.StatusBucket, error) {
	return nil, nil
}
func (f *fakeStatusStore) StatusDaily(context.Context, time.Time) ([]musicbot.StatusBucket, error) {
	return nil, nil
}
func (f *fakeStatusStore) FirstStatusSample(context.Context) (time.Time, error) {
	return time.Time{}, nil
}
func (f *fakeStatusStore) PruneStatus(context.Context, time.Time) error { return nil }
func (f *fakeStatusStore) CreateIncident(_ context.Context, in *musicbot.Incident) error {
	in.ID = int64(len(f.created) + 1)
	f.created = append(f.created, *in)
	return nil
}
func (f *fakeStatusStore) ResolveIncident(_ context.Context, id int64, _ time.Time) error {
	f.resolved = append(f.resolved, id)
	return nil
}
func (f *fakeStatusStore) OpenAutoIncidents(context.Context) ([]musicbot.Incident, error) {
	return nil, nil
}
func (f *fakeStatusStore) StatusIncidents(context.Context, time.Time, int) ([]musicbot.Incident, error) {
	return nil, nil
}

func TestHandleIncidentEvent(t *testing.T) {
	s := &Server{status: newStatusTracker()}
	store := &fakeStatusStore{}
	now := time.Now()

	s.status.record("station:chill", "Chill Mixes", "Stations", false, "off air", now)
	ev := s.status.record("station:chill", "Chill Mixes", "Stations", false, "off air", now.Add(statusSampleEvery))
	s.handleIncidentEvent(context.Background(), store, ev)
	if len(store.created) != 1 || store.created[0].Title != "Chill Mixes off air" || store.created[0].Kind != musicbot.IncidentAuto {
		t.Fatalf("unexpected incident %+v", store.created)
	}
	if s.status.components["station:chill"].incidentID != 1 {
		t.Error("the open incident id should be remembered")
	}

	resolve := s.status.record("station:chill", "Chill Mixes", "Stations", true, "3 listening", now.Add(2*statusSampleEvery))
	s.handleIncidentEvent(context.Background(), store, resolve)
	if len(store.resolved) != 1 || store.resolved[0] != 1 {
		t.Errorf("incident should be resolved, got %v", store.resolved)
	}
}

func TestIncidentViewsSplitNotices(t *testing.T) {
	now := time.Now()
	resolved := now.Add(-time.Hour)
	history, notices := incidentViews([]musicbot.Incident{
		{ID: 1, Kind: musicbot.IncidentAuto, StartedAt: now.Add(-2 * time.Hour), ResolvedAt: &resolved},
		{ID: 2, Kind: musicbot.IncidentMaintenance, StartedAt: now.Add(-time.Minute)},
		{ID: 3, Kind: musicbot.IncidentNotice, StartedAt: now.Add(-3 * time.Hour), ResolvedAt: &resolved},
		{ID: 4, Kind: musicbot.IncidentAuto, StartedAt: now.Add(-time.Minute)},
	}, now)
	if len(notices) != 1 || notices[0].ID != 2 {
		t.Errorf("only ongoing notices belong in notices: %+v", notices)
	}
	if len(history) != 3 || history[0].DurationSeconds != 3600 || !history[2].Ongoing {
		t.Errorf("unexpected history %+v", history)
	}
}
