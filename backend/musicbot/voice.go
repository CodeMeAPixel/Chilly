package musicbot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

var (
	ErrUserNotInVoice = errors.New("you need to be in a voice channel")
	ErrVoiceTimeout   = errors.New("timed out connecting to the voice channel")
)

type BusyError struct {
	ChannelID snowflake.ID
}

func (e *BusyError) Error() string {
	return fmt.Sprintf("already playing in <#%s>", e.ChannelID)
}

const voiceConnectTimeout = 10 * time.Second

func (b *Bot) UserVoiceChannel(guildID, userID snowflake.ID) (snowflake.ID, bool) {
	vs, ok := b.Client.Caches().VoiceState(guildID, userID)
	if !ok || vs.ChannelID == nil {
		return 0, false
	}
	return *vs.ChannelID, true
}

func (b *Bot) BotVoiceChannel(guildID snowflake.ID) (snowflake.ID, bool) {
	return b.UserVoiceChannel(guildID, b.Client.ApplicationID())
}

func (b *Bot) InSameVoice(guildID, userID snowflake.ID) bool {
	botCh, ok := b.BotVoiceChannel(guildID)
	if !ok {
		return false
	}
	userCh, ok := b.UserVoiceChannel(guildID, userID)
	return ok && userCh == botCh
}

func (b *Bot) EnsureVoice(ctx context.Context, guildID, channelID snowflake.ID) error {
	if ch, ok := b.PlayerManager.VoiceChannel(guildID); ok && ch == channelID {
		if botCh, inVoice := b.BotVoiceChannel(guildID); inVoice && botCh == channelID {
			return nil
		}
	}

	if botCh, inVoice := b.BotVoiceChannel(guildID); inVoice && botCh == channelID {
		if err := b.Client.UpdateVoiceState(ctx, guildID, nil, false, true); err != nil {
			return err
		}
		time.Sleep(500 * time.Millisecond)
	}

	wait := b.PlayerManager.voiceWaiter(guildID)
	if err := b.Client.UpdateVoiceState(ctx, guildID, &channelID, false, true); err != nil {
		return fmt.Errorf("failed to join voice channel: %w", err)
	}

	timer := time.NewTimer(voiceConnectTimeout)
	defer timer.Stop()
	for {
		select {
		case <-wait:
			if ch, ok := b.PlayerManager.VoiceChannel(guildID); ok && ch == channelID {
				return nil
			}
			wait = b.PlayerManager.voiceWaiter(guildID)
		case <-timer.C:
			return ErrVoiceTimeout
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

type EnqueueRequest struct {
	GuildID        snowflake.ID
	VoiceChannelID snowflake.ID

	TextChannelID snowflake.ID
	Tracks        []lavalink.Track
	Next          bool
	Loop          *LoopMode
	Shuffle       *ShuffleMode

	PlayNow bool
}

func (b *Bot) Enqueue(ctx context.Context, req EnqueueRequest) (*Player, error) {
	if len(req.Tracks) == 0 {
		return nil, errors.New("no tracks to queue")
	}
	if botCh, ok := b.BotVoiceChannel(req.GuildID); ok && botCh != req.VoiceChannelID {
		if p, exists := b.PlayerManager.GetPlayer(req.GuildID); exists && p.IsPlaying() {
			return nil, &BusyError{ChannelID: botCh}
		}
	}

	if err := b.EnsureVoice(ctx, req.GuildID, req.VoiceChannelID); err != nil {
		return nil, err
	}

	player := b.PlayerManager.GetOrCreatePlayer(req.GuildID)
	if req.TextChannelID != 0 {
		player.SetChannelID(req.TextChannelID)
	}
	if req.Loop != nil {
		player.SetLoop(*req.Loop)
	}
	if req.Shuffle != nil {
		player.SetShuffle(*req.Shuffle)
	}

	if req.PlayNow {
		if err := player.PlayNow(ctx, req.Tracks[0]); err != nil {
			return player, err
		}
		if len(req.Tracks) > 1 {
			player.Enqueue(req.Tracks[1:], true)
		}
		return player, nil
	}

	player.Enqueue(req.Tracks, req.Next)
	if _, err := player.Start(ctx); err != nil {
		return player, err
	}
	return player, nil
}

func (b *Bot) ListenerCount(guildID, channelID snowflake.ID) int {
	count := 0
	b.Client.Caches().VoiceStatesForEach(guildID, func(vs discord.VoiceState) {
		if vs.ChannelID == nil || *vs.ChannelID != channelID || vs.UserID == b.Client.ApplicationID() {
			return
		}
		if member, ok := b.Client.Caches().Member(guildID, vs.UserID); ok && member.User.Bot {
			return
		}
		count++
	})
	return count
}
