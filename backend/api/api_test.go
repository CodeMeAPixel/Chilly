package api

import (
	"testing"
	"time"

	"github.com/disgoorg/disgo/gateway"
)

func TestSafeRedirect(t *testing.T) {
	cases := map[string]string{
		"":                   "/",
		"/guilds/1":          "/guilds/1",
		"//evil.com":         "/",
		"https://evil.com":   "/",
		"/\\evil.com":        "/",
		"guilds":             "/",
		"/playlists?tab=all": "/playlists?tab=all",
	}
	for in, want := range cases {
		if got := safeRedirect(in); got != want {
			t.Errorf("safeRedirect(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSessionGuildCanManage(t *testing.T) {
	if !(SessionGuild{Owner: true}).CanManage() {
		t.Error("owner should manage")
	}
	if !(SessionGuild{Permissions: "8"}).CanManage() {
		t.Error("administrator should manage")
	}
	if !(SessionGuild{Permissions: "32"}).CanManage() {
		t.Error("manage guild should manage")
	}
	if (SessionGuild{Permissions: "1024"}).CanManage() {
		t.Error("view channel alone should not manage")
	}
}

func TestRateLimiter(t *testing.T) {
	l := newRateLimiter(1, 3)
	for i := 0; i < 3; i++ {
		if !l.allow("ip") {
			t.Fatalf("request %d should pass", i)
		}
	}
	if l.allow("ip") {
		t.Fatal("burst exceeded, should be limited")
	}
	if !l.allow("other") {
		t.Fatal("other keys are independent")
	}
}

func TestOriginOf(t *testing.T) {
	if got := originOf("https://dash.example.com/app"); got != "https://dash.example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestStatusTrackerBuckets(t *testing.T) {
	tr := newStatusTracker()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	tr.record("db", "Database", "Core", true, "", base)
	tr.record("db", "Database", "Core", false, "", base.Add(time.Minute))
	tr.record("db", "Database", "Core", true, "", base.Add(statusBucketSize))

	c := tr.components["db"]
	if len(c.buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(c.buckets))
	}
	if c.buckets[0].up != 1 || c.buckets[0].total != 2 {
		t.Fatalf("unexpected first bucket %+v", c.buckets[0])
	}

	tr.record("db", "Database", "Core", true, "", base.Add(statusBucketSize*statusBucketCount))
	if len(c.buckets) > statusBucketCount {
		t.Fatalf("history should be capped at %d buckets, got %d", statusBucketCount, len(c.buckets))
	}
	if c.buckets[0].start.Equal(base) {
		t.Fatal("oldest bucket should have been dropped")
	}
}

func TestGatewayHealthy(t *testing.T) {
	cases := []struct {
		status  gateway.Status
		latency time.Duration
		want    bool
	}{
		{gateway.StatusReady, 40 * time.Millisecond, true},
		{gateway.StatusResuming, 40 * time.Millisecond, true},
		{gateway.StatusResuming, -2 * time.Second, false},
		{gateway.StatusResuming, 0, false},
		{gateway.StatusDisconnected, 40 * time.Millisecond, false},
		{gateway.StatusWaitingForReady, 40 * time.Millisecond, false},
	}
	for _, c := range cases {
		if got := gatewayHealthy(c.status, c.latency); got != c.want {
			t.Errorf("gatewayHealthy(%s, %s) = %v, want %v", c.status, c.latency, got, c.want)
		}
	}
}
