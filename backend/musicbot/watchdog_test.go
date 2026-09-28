package musicbot

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

type stateLavalinkPlayer struct {
	*fakeLavalinkPlayer
	connected bool
}

func (p *stateLavalinkPlayer) State() lavalink.PlayerState {
	return lavalink.PlayerState{Connected: p.connected}
}

func (p *stateLavalinkPlayer) Node() disgolink.Node { return nil }

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met in time")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestVoiceWatchdogRejoinsSilentPlayer(t *testing.T) {
	ctx := context.Background()
	p, fake := newTestPlayer()
	p.Enqueue([]lavalink.Track{tr("a")}, false)
	if _, err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}

	var rejoins atomic.Int32
	s := &NodeSupervisor{health: make(map[snowflake.ID]*voiceHealth)}
	s.Rejoin = func(context.Context, snowflake.ID) error {
		rejoins.Add(1)
		return nil
	}
	lp := &stateLavalinkPlayer{fakeLavalinkPlayer: fake}

	for i := 0; i < voiceStrikesBeforeRecovery-1; i++ {
		s.checkVoice(ctx, p, lp)
	}
	if rejoins.Load() != 0 {
		t.Fatal("recovered too early")
	}
	s.checkVoice(ctx, p, lp)
	waitFor(t, func() bool { return rejoins.Load() == 1 })

	lp.connected = true
	waitFor(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.health[p.GuildID()].busy
	})
	s.checkVoice(ctx, p, lp)
	if s.health[p.GuildID()].strikes != 0 {
		t.Fatal("strikes should reset once connected")
	}
}

func TestVoiceWatchdogGivesUpAndStops(t *testing.T) {
	ctx := context.Background()
	p, fake := newTestPlayer()
	p.Enqueue([]lavalink.Track{tr("a"), tr("b")}, false)
	_, _ = p.Start(ctx)

	s := &NodeSupervisor{health: make(map[snowflake.ID]*voiceHealth)}
	s.Rejoin = func(context.Context, snowflake.ID) error { return nil }
	lp := &stateLavalinkPlayer{fakeLavalinkPlayer: fake}

	for attempt := 0; attempt < 3; attempt++ {
		waitFor(t, func() bool {
			s.mu.Lock()
			defer s.mu.Unlock()
			h, ok := s.health[p.GuildID()]
			return !ok || !h.busy
		})
		for i := 0; i < voiceStrikesBeforeRecovery; i++ {
			s.checkVoice(ctx, p, lp)
		}
	}
	waitFor(t, func() bool { return !p.IsPlaying() })
}
