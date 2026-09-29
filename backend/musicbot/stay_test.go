package musicbot

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
)

func TestStayStoreSetGetDelete(t *testing.T) {
	store := NewStayStore(nil)
	ctx := context.Background()
	guild := snowflake.ID(1)

	if store.Enabled(guild) {
		t.Fatal("new store should have no settings")
	}
	if err := store.Set(ctx, StaySetting{GuildID: guild, VoiceChannelID: 2, Station: "chill"}); err != nil {
		t.Fatal(err)
	}
	got, ok := store.Get(guild)
	if !ok || got.Station != "chill" || got.VoiceChannelID != 2 || got.EnabledAt.IsZero() {
		t.Fatalf("unexpected setting %+v (ok=%v)", got, ok)
	}

	existed, err := store.Delete(ctx, guild)
	if err != nil || !existed {
		t.Fatalf("delete = %v, %v; want true, nil", existed, err)
	}
	if existed, _ := store.Delete(ctx, guild); existed {
		t.Error("second delete should report nothing removed")
	}
}

func TestStayStoreAllSortedByEnabledAt(t *testing.T) {
	store := NewStayStore(nil)
	ctx := context.Background()
	now := time.Now()
	_ = store.Set(ctx, StaySetting{GuildID: 3, Station: "c", EnabledAt: now.Add(2 * time.Minute)})
	_ = store.Set(ctx, StaySetting{GuildID: 1, Station: "a", EnabledAt: now})
	_ = store.Set(ctx, StaySetting{GuildID: 2, Station: "b", EnabledAt: now.Add(time.Minute)})

	all := store.All()
	if len(all) != 3 || all[0].Station != "a" || all[1].Station != "b" || all[2].Station != "c" {
		t.Fatalf("unexpected order %+v", all)
	}
}

func TestStayBackoff(t *testing.T) {
	store := NewStayStore(nil)
	ctx := context.Background()
	guild := snowflake.ID(1)
	now := time.Now()
	_ = store.Set(ctx, StaySetting{GuildID: guild, Station: "chill"})

	if !store.due(guild, now) {
		t.Fatal("a new setting should be due immediately")
	}
	h := store.recordFailure(guild, errors.New("offline"), now)
	if h.Failures != 1 || !h.RetryAt.Equal(now.Add(stayBaseBackoff)) {
		t.Fatalf("first failure = %+v", h)
	}
	if store.due(guild, now.Add(stayBaseBackoff-time.Second)) {
		t.Error("should wait for the backoff to pass")
	}
	if !store.due(guild, now.Add(stayBaseBackoff)) {
		t.Error("should be due once the backoff passes")
	}

	for i := 0; i < 20; i++ {
		h = store.recordFailure(guild, errors.New("offline"), now)
	}
	if got := h.RetryAt.Sub(now); got != stayMaxBackoff {
		t.Errorf("backoff = %s, want capped at %s", got, stayMaxBackoff)
	}

	store.recordSuccess(guild)
	if store.Health(guild).Failures != 0 || !store.due(guild, now) {
		t.Error("success should reset the backoff")
	}
}

func TestStayFailureIgnoredAfterDelete(t *testing.T) {
	store := NewStayStore(nil)
	ctx := context.Background()
	_ = store.Set(ctx, StaySetting{GuildID: 1, Station: "chill"})
	_, _ = store.Delete(ctx, 1)
	if h := store.recordFailure(1, errors.New("x"), time.Now()); h.Failures != 0 {
		t.Errorf("failure recorded for a deleted setting: %+v", h)
	}
}
