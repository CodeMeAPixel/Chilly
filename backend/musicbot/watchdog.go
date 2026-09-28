package musicbot

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

const (
	voiceStrikesBeforeRecovery = 3
	voiceRecoveryResetAfter    = 2 * time.Minute
)

type voiceHealth struct {
	strikes     int
	attempts    int
	lastAttempt time.Time
	busy        bool
}

func (s *NodeSupervisor) checkVoice(ctx context.Context, player *Player, lp disgolink.Player) {
	guildID := player.GuildID()
	playing := player.IsPlaying()

	s.mu.Lock()
	defer s.mu.Unlock()

	if !playing {
		delete(s.health, guildID)
		return
	}
	h, ok := s.health[guildID]
	if !ok {
		h = &voiceHealth{}
		s.health[guildID] = h
	}
	if h.busy {
		return
	}
	if lp.State().Connected {
		h.strikes = 0
		if time.Since(h.lastAttempt) > voiceRecoveryResetAfter {
			h.attempts = 0
		}
		return
	}

	h.strikes++
	if h.strikes < voiceStrikesBeforeRecovery {
		return
	}
	h.strikes = 0
	h.attempts++
	h.lastAttempt = time.Now()
	h.busy = true
	attempt := h.attempts

	go func() {
		s.recoverVoice(ctx, player, lp, attempt)
		s.mu.Lock()
		if h, ok := s.health[guildID]; ok {
			h.busy = false
		}
		s.mu.Unlock()
	}()
}

func (s *NodeSupervisor) recoverVoice(ctx context.Context, player *Player, lp disgolink.Player, attempt int) {
	guildID := player.GuildID()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	switch {
	case attempt == 1 && lp.Node() != nil:
		slog.Warn("player is not connected to voice, resending voice credentials", slog.String("guild_id", guildID.String()))
		s.migrate(ctx, player, lp.Node())
	case attempt <= 2 && s.Rejoin != nil:
		slog.Warn("player is still not connected to voice, rejoining the channel", slog.String("guild_id", guildID.String()))
		if err := s.Rejoin(ctx, guildID); err != nil {
			slog.Error("failed to rejoin voice channel", slog.String("guild_id", guildID.String()), slog.Any("error", err))
		}
	default:
		slog.Error("voice connection could not be recovered, stopping player", slog.String("guild_id", guildID.String()))
		if err := player.Stop(ctx); err != nil {
			slog.Error("failed to stop player", slog.String("guild_id", guildID.String()), slog.Any("error", err))
		}
	}
}

func (b *Bot) RejoinVoice(ctx context.Context, guildID snowflake.ID) error {
	player, ok := b.PlayerManager.GetPlayer(guildID)
	if !ok {
		return nil
	}
	channelID, ok := b.BotVoiceChannel(guildID)
	if !ok {
		creds, hasCreds := b.PlayerManager.voiceCreds(guildID)
		if !hasCreds {
			return errors.New("unknown voice channel")
		}
		channelID = creds.channelID
	}

	lp := lavalinkPlayer(b.Lavalink, guildID)
	position, volume := lp.Position(), lp.Volume()
	state := player.exportState()

	if err := b.Client.UpdateVoiceState(ctx, guildID, nil, false, true); err != nil {
		return err
	}
	for i := 0; i < 30; i++ {
		if _, inVoice := b.BotVoiceChannel(guildID); !inVoice {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err := b.EnsureVoice(ctx, guildID, channelID); err != nil {
		return err
	}
	fresh := b.PlayerManager.GetOrCreatePlayer(guildID)
	fresh.importState(state)
	return fresh.restoreOn(ctx, position, volume)
}

type playerState struct {
	current      *lavalink.Track
	queue        []lavalink.Track
	history      []lavalink.Track
	loop         LoopMode
	shuffle      ShuffleMode
	paused       bool
	channelID    snowflake.ID
	sessionStart time.Time
}

func (p *Player) exportState() playerState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return playerState{
		current:      p.current,
		queue:        slices.Clone(p.tracks),
		history:      slices.Clone(p.prevtracks),
		loop:         p.loop,
		shuffle:      p.shuffle,
		paused:       p.paused,
		channelID:    p.channelID,
		sessionStart: p.sessionStartTime,
	}
}

func (p *Player) importState(s playerState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = s.current
	p.tracks = s.queue
	p.prevtracks = s.history
	p.loop = s.loop
	p.shuffle = s.shuffle
	p.paused = s.paused
	p.channelID = s.channelID
	p.sessionStartTime = s.sessionStart
	p.changed()
}
