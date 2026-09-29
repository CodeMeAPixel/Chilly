package musicbot

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"slices"
	"sync"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

type LoopMode string

const (
	LoopNone  LoopMode = "none"
	LoopTrack LoopMode = "track"
	LoopQueue LoopMode = "queue"
)

func ParseLoopMode(s string) (LoopMode, bool) {
	switch LoopMode(s) {
	case LoopNone, LoopTrack, LoopQueue:
		return LoopMode(s), true
	}
	return "", false
}

type ShuffleMode bool

const (
	ShuffleOn  ShuffleMode = true
	ShuffleOff ShuffleMode = false
)

const (
	maxHistory         = 50
	maxRadioRetries    = 5
	previousRestartPos = 5 * lavalink.Second
)

var ErrNothingPlaying = errors.New("nothing is playing")

type Player struct {
	mu               sync.Mutex
	link             disgolink.Client
	guildID          snowflake.ID
	notify           func(snowflake.ID)
	channelID        snowflake.ID
	playerMessage    *discord.Message
	sessionMessage   *discord.Message
	current          *lavalink.Track
	startedAt        time.Time
	tracks           []lavalink.Track
	prevtracks       []lavalink.Track
	paused           bool
	loop             LoopMode
	shuffle          ShuffleMode
	sessionStartTime time.Time
	radioRetries     int
}

func NewPlayer(guildID snowflake.ID, link disgolink.Client, notify func(snowflake.ID)) *Player {
	if notify == nil {
		notify = func(snowflake.ID) {}
	}
	return &Player{
		link:             link,
		guildID:          guildID,
		notify:           notify,
		tracks:           make([]lavalink.Track, 0),
		prevtracks:       make([]lavalink.Track, 0),
		loop:             LoopNone,
		shuffle:          ShuffleOff,
		sessionStartTime: time.Now(),
	}
}

func (p *Player) lp() disgolink.Player {
	return lavalinkPlayer(p.link, p.guildID)
}

func (p *Player) restoreOn(ctx context.Context, position lavalink.Duration, volume int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current == nil {
		return nil
	}
	opts := []lavalink.PlayerUpdateOpt{
		lavalink.WithTrack(*p.current),
		lavalink.WithPaused(p.paused),
		lavalink.WithVolume(volume),
	}
	if !p.current.Info.IsStream && position > 0 {
		opts = append(opts, lavalink.WithPosition(position))
	}
	return p.lp().Update(ctx, opts...)
}

func (p *Player) GuildID() snowflake.ID {
	return p.guildID
}

func (p *Player) changed() {
	go p.notify(p.guildID)
}

func (p *Player) Pause(ctx context.Context) error {
	return p.SetPaused(ctx, true)
}

func (p *Player) Resume(ctx context.Context) error {
	return p.SetPaused(ctx, false)
}

func (p *Player) SetPaused(ctx context.Context, paused bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current == nil {
		return ErrNothingPlaying
	}
	if err := p.lp().Update(ctx, lavalink.WithPaused(paused)); err != nil {
		return err
	}
	p.paused = paused
	p.changed()
	return nil
}

func (p *Player) SetVolume(ctx context.Context, volume int) error {
	volume = max(0, min(volume, 1000))
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.lp().Update(ctx, lavalink.WithVolume(volume)); err != nil {
		return err
	}
	p.changed()
	return nil
}

func (p *Player) Volume() int {
	return p.lp().Volume()
}

func (p *Player) Seek(ctx context.Context, position lavalink.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current == nil {
		return ErrNothingPlaying
	}
	if p.current.Info.IsStream {
		return errors.New("track is not seekable")
	}
	position = max(0, min(position, p.current.Info.Length))
	if err := p.lp().Update(ctx, lavalink.WithPosition(position)); err != nil {
		return err
	}
	p.changed()
	return nil
}

func (p *Player) Start(ctx context.Context) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current != nil {
		return false, nil
	}
	if len(p.tracks) == 0 {
		return false, nil
	}
	if err := p.advanceLocked(ctx, true); err != nil {
		return false, err
	}
	return p.current != nil, nil
}

func (p *Player) Skip(ctx context.Context) (lavalink.Track, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current == nil {
		return lavalink.Track{}, ErrNothingPlaying
	}
	skipped := *p.current
	return skipped, p.advanceLocked(ctx, true)
}

func (p *Player) PlayNext(ctx context.Context) error {
	_, err := p.Skip(ctx)
	if errors.Is(err, ErrNothingPlaying) {
		return nil
	}
	return err
}

func (p *Player) PlayPrevious(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current == nil {
		return ErrNothingPlaying
	}
	if len(p.prevtracks) == 0 || p.lp().Position() > previousRestartPos {
		if p.current.Info.IsStream {
			return nil
		}
		return p.lp().Update(ctx, lavalink.WithPosition(0))
	}

	last := p.prevtracks[len(p.prevtracks)-1]
	current := *p.current
	if err := p.playLocked(ctx, last); err != nil {
		return err
	}
	p.prevtracks = p.prevtracks[:len(p.prevtracks)-1]
	p.tracks = append([]lavalink.Track{current}, p.tracks...)
	p.changed()
	return nil
}

func (p *Player) PlayNow(ctx context.Context, track lavalink.Track) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	prev := p.current
	if err := p.playLocked(ctx, track); err != nil {
		return err
	}
	if prev != nil {
		p.pushHistory(*prev)
	}
	p.radioRetries = 0
	p.changed()
	return nil
}

func (p *Player) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.resetLocked()
	p.changed()
	return p.lp().Update(ctx, lavalink.WithNullTrack())
}

func (p *Player) StopAudio(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = nil
	p.changed()
	return p.lp().Update(ctx, lavalink.WithNullTrack())
}

func (p *Player) ClearState() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.resetLocked()
	p.changed()
}

func (p *Player) resetLocked() {
	p.current = nil
	p.paused = false
	p.loop = LoopNone
	p.shuffle = ShuffleOff
	p.tracks = make([]lavalink.Track, 0)
	p.prevtracks = make([]lavalink.Track, 0)
	p.radioRetries = 0
}

func (p *Player) OnTrackStart(track lavalink.Track) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.startedAt = time.Now()
	p.paused = false
	p.changed()
}

func (p *Player) OnTrackEnd(ctx context.Context, event lavalink.TrackEndEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.current == nil || !SameTrack(*p.current, event.Track) {
		return nil
	}

	switch event.Reason {
	case lavalink.TrackEndReasonFinished, lavalink.TrackEndReasonLoadFailed:
		if meta := GetTrackMeta(*p.current); meta.Radio != "" {
			if time.Since(p.startedAt) > time.Minute {
				p.radioRetries = 0
			}
			if p.radioRetries < maxRadioRetries {
				p.radioRetries++
				p.scheduleRetry(*p.current, time.Duration(p.radioRetries)*2*time.Second)
				return nil
			}
		}
		return p.advanceLocked(ctx, event.Reason == lavalink.TrackEndReasonLoadFailed)
	case lavalink.TrackEndReasonCleanup:
		p.current = nil
		p.changed()
	}
	return nil
}

func (p *Player) IsCurrent(track lavalink.Track) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.current != nil && SameTrack(*p.current, track)
}

func (p *Player) scheduleRetry(track lavalink.Track, delay time.Duration) {
	time.AfterFunc(delay, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.current == nil || !SameTrack(*p.current, track) {
			return
		}
		if err := p.playLocked(ctx, track); err != nil {
			slog.Error("failed to restart radio stream", slog.Any("error", err), slog.String("guild_id", p.guildID.String()))
			p.current = nil
			p.changed()
		}
	})
}

func (p *Player) AddToQueue(tracks ...lavalink.Track) {
	p.Enqueue(tracks, false)
}

func (p *Player) AddToQueueNext(tracks ...lavalink.Track) {
	p.Enqueue(tracks, true)
}

func (p *Player) Enqueue(tracks []lavalink.Track, next bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if next {
		p.tracks = append(slices.Clone(tracks), p.tracks...)
	} else {
		p.tracks = append(p.tracks, tracks...)
	}
	p.changed()
}

func (p *Player) RemoveFromQueue(index int) (lavalink.Track, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if index < 0 || index >= len(p.tracks) {
		return lavalink.Track{}, false
	}
	track := p.tracks[index]
	p.tracks = slices.Delete(p.tracks, index, index+1)
	p.changed()
	return track, true
}

func (p *Player) MoveInQueue(from, to int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if from < 0 || from >= len(p.tracks) || to < 0 || to >= len(p.tracks) {
		return false
	}
	track := p.tracks[from]
	p.tracks = slices.Delete(p.tracks, from, from+1)
	p.tracks = slices.Insert(p.tracks, to, track)
	p.changed()
	return true
}

func (p *Player) ClearQueue() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tracks = make([]lavalink.Track, 0)
	p.changed()
}

func (p *Player) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.current != nil
}

func (p *Player) IsPaused() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.paused
}

func (p *Player) Current() (lavalink.Track, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current == nil {
		return lavalink.Track{}, false
	}
	return *p.current, true
}

func (p *Player) Queue() []lavalink.Track {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.tracks)
}

func (p *Player) PreviousTracks() []lavalink.Track {
	p.mu.Lock()
	defer p.mu.Unlock()
	reversed := slices.Clone(p.prevtracks)
	slices.Reverse(reversed)
	return reversed
}

func (p *Player) Position() lavalink.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current == nil {
		return 0
	}
	return p.lp().Position()
}

func (p *Player) Loop() LoopMode {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.loop
}

func (p *Player) SetLoop(loop LoopMode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.loop = loop
	p.changed()
}

func (p *Player) Shuffle() ShuffleMode {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.shuffle
}

func (p *Player) SetShuffle(shuffle ShuffleMode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.shuffle = shuffle
	p.changed()
}

func (p *Player) PlayerMessage() *discord.Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playerMessage
}

func (p *Player) SetMessage(message *discord.Message) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.playerMessage = message
}

func (p *Player) TakePlayerMessage() *discord.Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	msg := p.playerMessage
	p.playerMessage = nil
	return msg
}

func (p *Player) ChannelID() snowflake.ID {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.channelID
}

func (p *Player) SetChannelID(channelID snowflake.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.channelID = channelID
}

func (p *Player) SessionMessage() *discord.Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sessionMessage
}

func (p *Player) SetSessionMessage(message *discord.Message) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sessionMessage = message
}

func (p *Player) TakeSessionMessage() *discord.Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	msg := p.sessionMessage
	p.sessionMessage = nil
	return msg
}

func (p *Player) SessionStartTime() time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sessionStartTime
}

func (p *Player) Snapshot(queueLimit int) PlayerSnapshot {
	lp := p.lp()

	p.mu.Lock()
	defer p.mu.Unlock()

	queue := p.tracks
	if queueLimit > 0 && len(queue) > queueLimit {
		queue = queue[:queueLimit]
	}
	history := slices.Clone(p.prevtracks)
	slices.Reverse(history)
	if len(history) > 10 {
		history = history[:10]
	}

	snap := PlayerSnapshot{
		GuildID:      p.guildID.String(),
		Playing:      p.current != nil,
		Paused:       p.paused,
		Volume:       lp.Volume(),
		Loop:         p.loop,
		Shuffle:      bool(p.shuffle),
		Queue:        NewTrackViews(queue),
		QueueLength:  len(p.tracks),
		History:      NewTrackViews(history),
		SessionStart: p.sessionStartTime,
		UpdatedAt:    time.Now(),
	}
	if p.channelID != 0 {
		snap.TextChannelID = p.channelID.String()
	}
	if ch := lp.ChannelID(); ch != nil {
		snap.VoiceChannelID = ch.String()
	}
	if p.current != nil {
		view := NewTrackView(*p.current)
		snap.Current = &view
		snap.PositionMs = int64(lp.Position())
	}
	return snap
}

func (p *Player) playLocked(ctx context.Context, track lavalink.Track) error {
	track = stampPlay(track)
	if err := p.lp().Update(ctx, lavalink.WithTrack(track), lavalink.WithPaused(false)); err != nil {
		return err
	}
	p.current = &track
	p.paused = false
	p.startedAt = time.Now()
	return nil
}

func (p *Player) pushHistory(track lavalink.Track) {
	p.prevtracks = append(p.prevtracks, track)
	if over := len(p.prevtracks) - maxHistory; over > 0 {
		p.prevtracks = slices.Delete(p.prevtracks, 0, over)
	}
}

func (p *Player) advanceLocked(ctx context.Context, skipped bool) error {
	prev := p.current
	savedQueue := slices.Clone(p.tracks)
	savedHistory := slices.Clone(p.prevtracks)

	err := p.doAdvance(ctx, prev, skipped)
	if err != nil {
		p.tracks = savedQueue
		p.prevtracks = savedHistory
		if skipped && prev != nil {
			p.current = prev
		} else {
			p.current = nil
		}
	}
	p.changed()
	return err
}

func (p *Player) doAdvance(ctx context.Context, prev *lavalink.Track, skipped bool) error {
	if prev != nil {
		if p.loop == LoopTrack && !skipped {
			return p.playLocked(ctx, *prev)
		}
		p.pushHistory(*prev)
		if p.loop == LoopQueue {
			p.tracks = append(p.tracks, *prev)
		}
	}

	if len(p.tracks) == 0 {
		p.current = nil
		if prev == nil {
			return nil
		}
		return p.lp().Update(ctx, lavalink.WithNullTrack())
	}

	idx := 0
	if p.shuffle == ShuffleOn && len(p.tracks) > 1 {
		idx = rand.Intn(len(p.tracks))
	}
	next := p.tracks[idx]
	p.tracks = slices.Delete(p.tracks, idx, idx+1)
	p.radioRetries = 0
	return p.playLocked(ctx, next)
}
