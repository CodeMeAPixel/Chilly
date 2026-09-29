package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/CodeMeAPixel/Chilly/azuracast"
	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

const aloneTimeout = 30 * time.Second

type Handlers struct {
	*musicbot.Bot
	voiceMu     sync.Mutex
	serverSeen  map[snowflake.ID]bool
	aloneMu     sync.Mutex
	aloneTimers map[snowflake.ID]*time.Timer

	viewMu sync.Mutex
	views  map[snowflake.ID]string
}

func New(b *musicbot.Bot) *Handlers {
	return &Handlers{
		Bot:         b,
		serverSeen:  make(map[snowflake.ID]bool),
		aloneTimers: make(map[snowflake.ID]*time.Timer),
		views:       make(map[snowflake.ID]string),
	}
}

func (h *Handlers) OnVoiceStateUpdate(event *events.GuildVoiceStateUpdate) {
	guildID := event.VoiceState.GuildID
	if event.VoiceState.UserID == h.Client.ApplicationID() {
		h.onBotVoiceStateUpdate(event)
		return
	}

	botChannel, ok := h.BotVoiceChannel(guildID)
	if !ok {
		return
	}
	touchesBot := (event.VoiceState.ChannelID != nil && *event.VoiceState.ChannelID == botChannel) ||
		(event.OldVoiceState.ChannelID != nil && *event.OldVoiceState.ChannelID == botChannel)
	if !touchesBot {
		return
	}

	listeners := h.listenersIn(guildID, botChannel)
	if listeners == 0 {
		h.scheduleLeave(guildID)
		return
	}
	h.cancelLeave(guildID)

	if listeners == 1 && event.VoiceState.ChannelID != nil && *event.VoiceState.ChannelID == botChannel {
		player, ok := h.PlayerManager.GetPlayer(guildID)
		if !ok || !player.IsPlaying() {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if event.VoiceState.SelfDeaf && !event.OldVoiceState.SelfDeaf {
			_ = player.Pause(ctx)
		} else if !event.VoiceState.SelfDeaf && event.OldVoiceState.SelfDeaf {
			_ = player.Resume(ctx)
		}
	}
}

func (h *Handlers) listenersIn(guildID, channelID snowflake.ID) int {
	return h.ListenerCount(guildID, channelID)
}

func (h *Handlers) scheduleLeave(guildID snowflake.ID) {
	if h.Stays.Enabled(guildID) {
		return
	}
	h.aloneMu.Lock()
	defer h.aloneMu.Unlock()
	if _, pending := h.aloneTimers[guildID]; pending {
		return
	}
	h.aloneTimers[guildID] = time.AfterFunc(aloneTimeout, func() {
		h.aloneMu.Lock()
		delete(h.aloneTimers, guildID)
		h.aloneMu.Unlock()

		botChannel, ok := h.BotVoiceChannel(guildID)
		if !ok || h.listenersIn(guildID, botChannel) > 0 || h.Stays.Enabled(guildID) {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := h.Client.UpdateVoiceState(ctx, guildID, nil, false, false); err != nil {
			slog.Error("failed to disconnect from voice channel",
				slog.Any("error", err), slog.String("guild_id", guildID.String()))
		}
	})
}

func (h *Handlers) cancelLeave(guildID snowflake.ID) {
	h.aloneMu.Lock()
	defer h.aloneMu.Unlock()
	if t, ok := h.aloneTimers[guildID]; ok {
		t.Stop()
		delete(h.aloneTimers, guildID)
	}
}

func (h *Handlers) onBotVoiceStateUpdate(event *events.GuildVoiceStateUpdate) {
	guildID := event.VoiceState.GuildID
	channelID := event.VoiceState.ChannelID

	slog.Info("received bot voice state update",
		slog.String("guild_id", guildID.String()),
		slog.Any("channel_id", channelID),
		slog.String("session_id", event.VoiceState.SessionID),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h.voiceMu.Lock()

	if channelID == nil {
		if lp := h.Lavalink.ExistingPlayer(guildID); lp != nil {
			lp.OnVoiceStateUpdate(ctx, nil, event.VoiceState.SessionID)
		}
	} else {
		h.PlayerManager.LavalinkPlayer(guildID).OnVoiceStateUpdate(ctx, channelID, event.VoiceState.SessionID)
		h.PlayerManager.SetVoiceSession(guildID, *channelID, event.VoiceState.SessionID)
	}
	if channelID == nil {
		delete(h.serverSeen, guildID)
		h.PlayerManager.ClearVoice(guildID)
	} else if h.serverSeen[guildID] {
		h.PlayerManager.MarkVoiceReady(guildID, *channelID)
	}
	h.voiceMu.Unlock()

	if channelID != nil {
		return
	}

	h.cancelLeave(guildID)
	player, ok := h.PlayerManager.GetPlayer(guildID)
	if !ok {
		return
	}
	player.ClearState()
	h.PlayerManager.DeletePlayer(guildID)
	playerMessage, sessionMessage := player.TakePlayerMessage(), player.TakeSessionMessage()
	go func() {
		if playerMessage != nil {
			h.deleteMessage(guildID, playerMessage)
		}
		if sessionMessage != nil {
			h.deleteMessage(guildID, sessionMessage)
		}
	}()
}

func (h *Handlers) OnVoiceServerUpdate(event *events.VoiceServerUpdate) {
	if event.Endpoint == nil {
		return
	}
	slog.Info("received voice server update",
		slog.String("guild_id", event.GuildID.String()),
		slog.String("endpoint", *event.Endpoint),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h.voiceMu.Lock()
	defer h.voiceMu.Unlock()
	lp := h.PlayerManager.LavalinkPlayer(event.GuildID)
	lp.OnVoiceServerUpdate(ctx, event.Token, *event.Endpoint)
	h.PlayerManager.SetVoiceServer(event.GuildID, event.Token, *event.Endpoint)
	h.serverSeen[event.GuildID] = true
	if ch := lp.ChannelID(); ch != nil {
		h.PlayerManager.MarkVoiceReady(event.GuildID, *ch)
	}
}

func (h *Handlers) OnTrackStart(p disgolink.Player, event lavalink.TrackStartEvent) {
	player, ok := h.PlayerManager.GetPlayer(p.GuildID())
	if !ok {
		return
	}
	player.OnTrackStart(event.Track)

	go h.postPlayerMessage(player)
}

func (h *Handlers) postPlayerMessage(player *musicbot.Player) {
	guildID := player.GuildID()
	if msg := player.TakeSessionMessage(); msg != nil {
		h.deleteMessage(guildID, msg)
	}
	if msg := player.TakePlayerMessage(); msg != nil {
		h.deleteMessage(guildID, msg)
	}

	channelID := player.ChannelID()
	track, ok := player.Current()
	if channelID == 0 || !ok {
		return
	}

	playerMessage, err := h.Client.Rest().CreateMessage(channelID, h.createPlayerMessage(player, track))
	if err != nil {
		slog.Error("failed to send player embed",
			slog.Any("error", err), slog.String("guild_id", guildID.String()))
		return
	}
	if current, ok := player.Current(); ok && musicbot.SameTrack(current, track) {
		player.SetMessage(playerMessage)
	} else {
		h.deleteMessage(guildID, playerMessage)
	}
}

func (h *Handlers) OnTrackEnd(p disgolink.Player, event lavalink.TrackEndEvent) {
	player, ok := h.PlayerManager.GetPlayer(p.GuildID())
	if !ok {
		return
	}

	if event.Reason.MayStartNext() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := player.OnTrackEnd(ctx, event); err != nil {
			slog.Error("failed to play next track in queue",
				slog.Any("error", err), slog.String("guild_id", p.GuildID().String()))
		}
		cancel()
	} else {
		_ = player.OnTrackEnd(context.Background(), event)
	}

	oldMessage := player.TakePlayerMessage()
	go func() {
		if oldMessage != nil {
			h.deleteMessage(p.GuildID(), oldMessage)
		}
		if event.Reason.MayStartNext() && !player.IsPlaying() && player.ChannelID() != 0 {
			if prev := player.PreviousTracks(); len(prev) > 0 {
				msg, err := h.Client.Rest().CreateMessage(player.ChannelID(), createRecentlyPlayedEmbed(prev, player.SessionStartTime()))
				if err != nil {
					slog.Error("failed to create session message",
						slog.Any("error", err), slog.String("guild_id", p.GuildID().String()))
					return
				}
				player.SetSessionMessage(msg)
			}
		}
	}()
}

func (h *Handlers) OnTrackException(p disgolink.Player, event lavalink.TrackExceptionEvent) {
	attrs := []any{
		slog.String("guild_id", p.GuildID().String()),
		slog.String("node", p.Node().Config().Name),
		slog.String("title", event.Track.Info.Title),
		slog.String("source", event.Track.Info.SourceName),
		slog.String("message", event.Exception.Message),
	}
	if !strings.Contains(event.Exception.Cause, event.Exception.Message) {
		attrs = append(attrs, slog.String("cause", event.Exception.Cause))
	}
	slog.Warn("track exception", attrs...)
	player, ok := h.PlayerManager.GetPlayer(p.GuildID())
	if !ok || player.ChannelID() == 0 {
		return
	}
	go h.tempMessage(p.GuildID(), player.ChannelID(), fmt.Sprintf("⚠️ Couldn't play %s: %s",
		musicbot.TrackLink(event.Track), musicbot.Trim(event.Exception.Message, 200)))
}

func (h *Handlers) OnTrackStuck(p disgolink.Player, event lavalink.TrackStuckEvent) {
	node := ""
	if n := p.Node(); n != nil {
		node = n.Config().Name
	}
	slog.Warn("track stuck",
		slog.String("guild_id", p.GuildID().String()),
		slog.String("node", node),
		slog.String("title", event.Track.Info.Title),
		slog.Any("threshold", event.Threshold),
	)
	player, ok := h.PlayerManager.GetPlayer(p.GuildID())
	if !ok {
		return
	}
	if current, ok := player.Current(); !ok || !musicbot.SameTrack(current, event.Track) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := player.Skip(ctx); err != nil {
		slog.Error("failed to skip stuck track", slog.Any("error", err))
	}
}

func (h *Handlers) OnWebSocketClosed(p disgolink.Player, event lavalink.WebSocketClosedEvent) {
	slog.Warn("lavalink voice websocket closed",
		slog.String("guild_id", p.GuildID().String()),
		slog.Int("code", event.Code),
		slog.String("reason", event.Reason),
		slog.Bool("by_remote", event.ByRemote),
	)
	if event.Code == 4017 {
		slog.Error("discord requires the DAVE (E2EE) voice protocol, upgrade lavalink to a version that supports it",
			slog.String("guild_id", p.GuildID().String()))
	}
}

func (h *Handlers) OnRadioSongChange(np azuracast.NowPlaying) {
	h.PlayerManager.ForEach(func(player *musicbot.Player) {
		track, ok := player.Current()
		if !ok || musicbot.GetTrackMeta(track).Radio != np.Station.Shortcode {
			return
		}
		h.RefreshPlayerMessage(player.GuildID())
	})
}

func (h *Handlers) deleteMessage(guildID snowflake.ID, msg *discord.Message) {
	if err := h.Client.Rest().DeleteMessage(msg.ChannelID, msg.ID); err != nil {
		musicbot.LogDeleteError(err, guildID.String(), msg.ChannelID.String(), msg.ID.String())
	}
}

func (h *Handlers) tempMessage(guildID, channelID snowflake.ID, content string) {
	msg, err := h.Client.Rest().CreateMessage(channelID, discord.MessageCreate{
		Embeds: []discord.Embed{{Description: content}},
	})
	if err != nil {
		return
	}
	time.AfterFunc(15*time.Second, func() { h.deleteMessage(guildID, msg) })
}
