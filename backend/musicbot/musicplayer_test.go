package musicbot

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

type fakeLink struct {
	disgolink.Client
	player *fakeLavalinkPlayer
}

func (f *fakeLink) ExistingPlayer(snowflake.ID) disgolink.Player { return f.player }

type fakeLavalinkPlayer struct {
	disgolink.Player
	playing string
	fail    bool
	updates int
}

func (f *fakeLavalinkPlayer) Update(_ context.Context, opts ...lavalink.PlayerUpdateOpt) error {
	f.updates++
	if f.fail {
		return errors.New("lavalink down")
	}
	u := lavalink.DefaultPlayerUpdate()
	u.Apply(opts)
	if u.Track != nil && u.Track.Encoded != nil {
		raw, _ := u.Track.Encoded.MarshalJSON()
		var enc *string
		_ = json.Unmarshal(raw, &enc)
		if enc == nil {
			f.playing = ""
		} else {
			f.playing = *enc
		}
	}
	return nil
}

func (f *fakeLavalinkPlayer) Position() lavalink.Duration { return 0 }
func (f *fakeLavalinkPlayer) Volume() int                 { return 100 }
func (f *fakeLavalinkPlayer) ChannelID() *snowflake.ID    { return nil }

func newTestPlayer() (*Player, *fakeLavalinkPlayer) {
	lp := &fakeLavalinkPlayer{}
	return NewPlayer(1, &fakeLink{player: lp}, nil), lp
}

func tr(id string) lavalink.Track {
	return lavalink.Track{Encoded: id, Info: lavalink.TrackInfo{Title: id, Length: lavalink.Minute}}
}

func end(track lavalink.Track, reason lavalink.TrackEndReason) lavalink.TrackEndEvent {
	return lavalink.TrackEndEvent{Track: track, Reason: reason}
}

func mustCurrent(t *testing.T, p *Player, want string) {
	t.Helper()
	cur, ok := p.Current()
	if want == "" {
		if ok {
			t.Fatalf("expected nothing playing, got %q", cur.Encoded)
		}
		return
	}
	if !ok || cur.Encoded != want {
		t.Fatalf("expected current %q, got %q (ok=%v)", want, cur.Encoded, ok)
	}
}

func TestStartAndAdvance(t *testing.T) {
	ctx := context.Background()
	p, lp := newTestPlayer()
	p.Enqueue([]lavalink.Track{tr("a"), tr("b")}, false)

	started, err := p.Start(ctx)
	if err != nil || !started {
		t.Fatalf("start: started=%v err=%v", started, err)
	}
	mustCurrent(t, p, "a")
	if lp.playing != "a" {
		t.Fatalf("lavalink playing %q", lp.playing)
	}

	if started, _ := p.Start(ctx); started {
		t.Fatal("Start must not restart while playing")
	}

	_ = p.OnTrackEnd(ctx, end(tr("a"), lavalink.TrackEndReasonFinished))
	mustCurrent(t, p, "b")

	_ = p.OnTrackEnd(ctx, end(tr("b"), lavalink.TrackEndReasonFinished))
	mustCurrent(t, p, "")
	if lp.playing != "" {
		t.Fatalf("expected null track to be sent, lavalink playing %q", lp.playing)
	}
	if got := len(p.PreviousTracks()); got != 2 {
		t.Fatalf("expected 2 tracks in history, got %d", got)
	}
}

func TestStaleTrackEndIsIgnored(t *testing.T) {
	ctx := context.Background()
	p, _ := newTestPlayer()
	p.Enqueue([]lavalink.Track{tr("a"), tr("b"), tr("c")}, false)
	_, _ = p.Start(ctx)

	if _, err := p.Skip(ctx); err != nil {
		t.Fatal(err)
	}
	mustCurrent(t, p, "b")

	_ = p.OnTrackEnd(ctx, end(tr("a"), lavalink.TrackEndReasonFinished))
	mustCurrent(t, p, "b")

	_ = p.OnTrackEnd(ctx, end(tr("b"), lavalink.TrackEndReasonReplaced))
	mustCurrent(t, p, "b")
}

func TestLoopTrack(t *testing.T) {
	ctx := context.Background()
	p, _ := newTestPlayer()
	p.Enqueue([]lavalink.Track{tr("a"), tr("b")}, false)
	_, _ = p.Start(ctx)
	p.SetLoop(LoopTrack)

	_ = p.OnTrackEnd(ctx, end(tr("a"), lavalink.TrackEndReasonFinished))
	mustCurrent(t, p, "a")

	_, _ = p.Skip(ctx)
	mustCurrent(t, p, "b")

	_ = p.OnTrackEnd(ctx, end(tr("b"), lavalink.TrackEndReasonLoadFailed))
	mustCurrent(t, p, "")
}

func TestLoopQueueSingleTrack(t *testing.T) {
	ctx := context.Background()
	p, _ := newTestPlayer()
	p.Enqueue([]lavalink.Track{tr("a")}, false)
	_, _ = p.Start(ctx)
	p.SetLoop(LoopQueue)

	for i := 0; i < 3; i++ {
		_ = p.OnTrackEnd(ctx, end(tr("a"), lavalink.TrackEndReasonFinished))
		mustCurrent(t, p, "a")
	}
}

func TestFailedStartRestoresQueue(t *testing.T) {
	ctx := context.Background()
	p, lp := newTestPlayer()
	lp.fail = true
	p.Enqueue([]lavalink.Track{tr("a"), tr("b")}, false)

	if _, err := p.Start(ctx); err == nil {
		t.Fatal("expected error")
	}
	mustCurrent(t, p, "")
	if q := p.Queue(); len(q) != 2 || q[0].Encoded != "a" {
		t.Fatalf("queue not restored: %+v", q)
	}

	lp.fail = false
	if started, err := p.Start(ctx); err != nil || !started {
		t.Fatalf("restart: started=%v err=%v", started, err)
	}
	mustCurrent(t, p, "a")
}

func TestFailedSkipKeepsCurrent(t *testing.T) {
	ctx := context.Background()
	p, lp := newTestPlayer()
	p.Enqueue([]lavalink.Track{tr("a"), tr("b")}, false)
	_, _ = p.Start(ctx)

	lp.fail = true
	if _, err := p.Skip(ctx); err == nil {
		t.Fatal("expected error")
	}
	mustCurrent(t, p, "a")
	if q := p.Queue(); len(q) != 1 || q[0].Encoded != "b" {
		t.Fatalf("queue not restored: %+v", q)
	}
}

func TestPreviousRequeuesCurrent(t *testing.T) {
	ctx := context.Background()
	p, _ := newTestPlayer()
	p.Enqueue([]lavalink.Track{tr("a"), tr("b"), tr("c")}, false)
	_, _ = p.Start(ctx)
	_, _ = p.Skip(ctx)
	mustCurrent(t, p, "b")

	if err := p.PlayPrevious(ctx); err != nil {
		t.Fatal(err)
	}
	mustCurrent(t, p, "a")
	if q := p.Queue(); len(q) != 2 || q[0].Encoded != "b" || q[1].Encoded != "c" {
		t.Fatalf("unexpected queue after previous: %+v", q)
	}
}

func TestPlayNextEmptyQueueWithTrackLoopDoesNotPanic(t *testing.T) {
	p, _ := newTestPlayer()
	p.SetLoop(LoopTrack)
	if err := p.PlayNext(context.Background()); err != nil {
		t.Fatal(err)
	}
	mustCurrent(t, p, "")
}

func TestEnqueueNextAndMove(t *testing.T) {
	p, _ := newTestPlayer()
	p.Enqueue([]lavalink.Track{tr("a"), tr("b")}, false)
	p.Enqueue([]lavalink.Track{tr("x")}, true)
	if q := p.Queue(); q[0].Encoded != "x" {
		t.Fatalf("expected x first, got %+v", q)
	}
	if !p.MoveInQueue(0, 2) {
		t.Fatal("move failed")
	}
	if q := p.Queue(); q[0].Encoded != "a" || q[2].Encoded != "x" {
		t.Fatalf("unexpected order %+v", q)
	}
	if p.MoveInQueue(0, 5) {
		t.Fatal("out of range move should fail")
	}
}

func TestTrackEndWithReencodedTrackAdvances(t *testing.T) {
	ctx := context.Background()
	p, _ := newTestPlayer()
	p.Enqueue([]lavalink.Track{tr("a"), tr("b")}, false)
	_, _ = p.Start(ctx)

	current, _ := p.Current()
	fromLavalink := current
	fromLavalink.Encoded = "a-reencoded-with-position"

	if err := p.OnTrackEnd(ctx, end(fromLavalink, lavalink.TrackEndReasonFinished)); err != nil {
		t.Fatal(err)
	}
	mustCurrent(t, p, "b")
}

func TestTrackEndWithoutUserDataFallsBackToIdentifier(t *testing.T) {
	ctx := context.Background()
	p, _ := newTestPlayer()
	a := tr("a")
	a.Info.Identifier, a.Info.SourceName = "vid-a", "youtube"
	p.Enqueue([]lavalink.Track{a, tr("b")}, false)
	_, _ = p.Start(ctx)

	fromLavalink := lavalink.Track{Encoded: "different", Info: lavalink.TrackInfo{Identifier: "vid-a", SourceName: "youtube"}}
	_ = p.OnTrackEnd(ctx, end(fromLavalink, lavalink.TrackEndReasonFinished))
	mustCurrent(t, p, "b")
}
