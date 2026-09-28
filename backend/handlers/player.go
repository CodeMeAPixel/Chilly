package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/CodeMeAPixel/Chilly/commands"
	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

type ButtonID string

const (
	PlayPrevious ButtonID = "play_previous"
	PlayNext     ButtonID = "play_next"
	PausePlayer  ButtonID = "pause_player"
	ResumePlayer ButtonID = "resume_player"
	StopPlayer   ButtonID = "stop_player"
	LoopQueue    ButtonID = "loop_queue"
	LoopTrack    ButtonID = "loop_track"
	LoopOff      ButtonID = "loop_off"
	ShuffleOn    ButtonID = "shuffle_on"
	ShuffleOff   ButtonID = "shuffle_off"
	ShowLyrics   ButtonID = "show_lyrics"
)

var playerButtons = map[ButtonID]bool{
	PlayPrevious: true, PlayNext: true, PausePlayer: true, ResumePlayer: true, StopPlayer: true,
	LoopQueue: true, LoopTrack: true, LoopOff: true, ShuffleOn: true, ShuffleOff: true,
	ShowLyrics: true,
}

func (h *Handlers) OnPlayerInteraction(event *events.ComponentInteractionCreate) {
	if event.GuildID() == nil || event.Data.Type() != discord.ComponentTypeButton {
		return
	}
	buttonID := ButtonID(event.Data.CustomID())
	if !playerButtons[buttonID] {
		return
	}
	guildID := *event.GuildID()

	if buttonID == ShowLyrics {
		h.showLyrics(event, guildID)
		return
	}

	player, ok := h.PlayerManager.GetPlayer(guildID)
	if !ok || !player.IsPlaying() {
		_ = event.DeferUpdateMessage()
		if err := h.Client.Rest().DeleteMessage(event.Message.ChannelID, event.Message.ID); err != nil {
			musicbot.LogDeleteError(err, guildID.String(), event.Message.ChannelID.String(), event.Message.ID.String())
		}
		return
	}
	if !h.InSameVoice(guildID, event.User().ID) {
		_ = event.CreateMessage(discord.MessageCreate{
			Content: "You need to be in my voice channel to control the player.",
			Flags:   discord.MessageFlagEphemeral,
		})
		return
	}

	musicbot.LogPlayerInteraction(string(buttonID), guildID.String(), event.User().ID.String())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	switch buttonID {
	case PlayNext, StopPlayer, PlayPrevious:

		_ = event.DeferUpdateMessage()
		switch buttonID {
		case PlayNext:
			_, err = player.Skip(ctx)
		case StopPlayer:
			err = player.Stop(ctx)
		case PlayPrevious:
			err = player.PlayPrevious(ctx)
		}
		if err != nil {
			musicbot.LogCommandError(err, "button/"+string(buttonID), guildID.String(), event.User().ID.String())
		}
		return
	case ResumePlayer:
		err = player.Resume(ctx)
	case PausePlayer:
		err = player.Pause(ctx)
	case ShuffleOn:
		player.SetShuffle(musicbot.ShuffleOn)
	case ShuffleOff:
		player.SetShuffle(musicbot.ShuffleOff)
	case LoopOff:
		player.SetLoop(musicbot.LoopNone)
	case LoopTrack:
		player.SetLoop(musicbot.LoopTrack)
	case LoopQueue:
		player.SetLoop(musicbot.LoopQueue)
	}
	if err != nil {
		musicbot.LogCommandError(err, "button/"+string(buttonID), guildID.String(), event.User().ID.String())
	}

	if track, ok := player.Current(); ok {
		embed, rows := h.playerView(player, track)
		if err := event.UpdateMessage(discord.MessageUpdate{Embeds: &[]discord.Embed{embed}, Components: &rows}); err != nil {
			musicbot.LogUpdateError(err, guildID.String(), event.User().ID.String())
			return
		}
		h.rememberView(guildID, embed, rows)
		return
	}
	_ = event.DeferUpdateMessage()
}

func (h *Handlers) createPlayerMessage(player *musicbot.Player, track lavalink.Track) discord.MessageCreate {
	embed, rows := h.playerView(player, track)
	h.rememberView(player.GuildID(), embed, rows)
	return discord.MessageCreate{
		Embeds:     []discord.Embed{embed},
		Components: rows,
	}
}

func (h *Handlers) playerView(player *musicbot.Player, track lavalink.Track) (discord.Embed, []discord.ContainerComponent) {
	rows := h.buttonRows(player)
	if station := musicbot.GetTrackMeta(track).Radio; station != "" && h.Radio != nil {
		if np, ok := h.Radio.Station(station); ok {
			return commands.RadioEmbed(np), rows
		}
	}
	return h.trackEmbed(player, track), rows
}

func (h *Handlers) botAvatar() string {
	if self, ok := h.Client.Caches().SelfUser(); ok {
		return self.EffectiveAvatarURL()
	}
	return ""
}

func (h *Handlers) trackEmbed(player *musicbot.Player, track lavalink.Track) discord.Embed {
	meta := musicbot.GetTrackMeta(track)

	status := "Now playing"
	if player.IsPaused() {
		status = "Paused"
	}

	description := fmt.Sprintf("%s · `%s`", musicbot.EscapeMarkdown(track.Info.Author), musicbot.TrackDuration(track))
	if meta.Requester != 0 {
		description += fmt.Sprintf("\nRequested by <@%s>", meta.Requester)
		if meta.PlaylistName != "" {
			description += fmt.Sprintf(" from **%s**", musicbot.EscapeMarkdown(meta.PlaylistName))
		}
	}

	builder := discord.NewEmbedBuilder().
		SetColor(musicbot.ColorMint).
		SetAuthor(status, "", h.botAvatar()).
		SetTitle(musicbot.Trim(musicbot.TrackTitle(track), 256)).
		SetURL(musicbot.TrackURL(track)).
		SetDescription(description).
		SetThumbnail(musicbot.TrackArtwork(track))

	if queue := player.Queue(); len(queue) > 0 {
		next := musicbot.TrackLink(queue[0])
		if len(queue) > 1 {
			next += fmt.Sprintf(" · +%d more", len(queue)-1)
		}
		builder.AddField("Up next", next, false)
	}

	loop := map[musicbot.LoopMode]string{musicbot.LoopNone: "off", musicbot.LoopTrack: "track", musicbot.LoopQueue: "queue"}[player.Loop()]
	shuffle := "off"
	if player.Shuffle() {
		shuffle = "on"
	}
	builder.SetFooter(fmt.Sprintf("%s · Loop %s · Shuffle %s · %s", musicbot.SourceLabel(track.Info.SourceName), loop, shuffle, h.SiteHost()), "")
	return builder.Build()
}

func emojiButton(id ButtonID, emojiID int) discord.ButtonComponent {
	return discord.NewSecondaryButton("", string(id)).WithEmoji(discord.ComponentEmoji{ID: snowflake.ID(emojiID)})
}

func (h *Handlers) buttonRows(player *musicbot.Player) []discord.ContainerComponent {
	playPause := emojiButton(PausePlayer, musicbot.PAUSE_PLAYER_EMOJI_ID)
	if player.IsPaused() {
		playPause = emojiButton(ResumePlayer, musicbot.RESUME_PLAYER_EMOJI_ID)
	}

	var repeat discord.ButtonComponent
	switch player.Loop() {
	case musicbot.LoopQueue:
		repeat = emojiButton(LoopTrack, musicbot.LOOP_QUEUE_EMOJI_ID)
	case musicbot.LoopTrack:
		repeat = emojiButton(LoopOff, musicbot.LOOP_TRACK_EMOJI_ID)
	default:
		repeat = emojiButton(LoopQueue, musicbot.LOOP_OFF_EMOJI_ID)
	}

	shuffleButton := emojiButton(ShuffleOn, musicbot.SHUFFLE_OFF_EMOJI_ID)
	if player.Shuffle() {
		shuffleButton = emojiButton(ShuffleOff, musicbot.SHUFFLE_ON_EMOJI_ID)
	}

	return []discord.ContainerComponent{
		discord.NewActionRow(
			emojiButton(PlayPrevious, musicbot.PLAYER_PREVIOUS_EMOJI_ID),
			playPause,
			emojiButton(PlayNext, musicbot.PLAYER_NEXT_EMOJI_ID),
			discord.NewSecondaryButton("Lyrics", string(ShowLyrics)),
		),
		discord.NewActionRow(
			emojiButton(StopPlayer, musicbot.STOP_PLAYER_EMOJI_ID),
			repeat,
			shuffleButton,
			discord.NewLinkButton("Dashboard", h.SiteURL("/dashboard/"+player.GuildID().String())),
		),
	}
}

func viewSignature(embed discord.Embed, rows []discord.ContainerComponent) string {
	data, _ := json.Marshal(struct {
		Embed discord.Embed                `json:"embed"`
		Rows  []discord.ContainerComponent `json:"rows"`
	}{embed, rows})
	return string(data)
}

func (h *Handlers) rememberView(guildID snowflake.ID, embed discord.Embed, rows []discord.ContainerComponent) {
	h.viewMu.Lock()
	h.views[guildID] = viewSignature(embed, rows)
	h.viewMu.Unlock()
}

func (h *Handlers) RefreshPlayerMessage(guildID snowflake.ID) {
	player, ok := h.PlayerManager.GetPlayer(guildID)
	if !ok {
		return
	}
	msg := player.PlayerMessage()
	track, playing := player.Current()
	if msg == nil || !playing {
		return
	}

	embed, rows := h.playerView(player, track)
	signature := viewSignature(embed, rows)
	h.viewMu.Lock()
	unchanged := h.views[guildID] == signature
	h.viewMu.Unlock()
	if unchanged {
		return
	}

	if _, err := h.Client.Rest().UpdateMessage(msg.ChannelID, msg.ID, discord.MessageUpdate{
		Embeds:     &[]discord.Embed{embed},
		Components: &rows,
	}); err != nil {
		slog.Debug("failed to refresh player message", slog.Any("error", err), slog.String("guild_id", guildID.String()))
		return
	}
	h.rememberView(guildID, embed, rows)
}

func (h *Handlers) RunPlayerMessageSync(ctx context.Context) {
	updates, unsubscribe := h.PlayerManager.Hub.SubscribeAll()
	defer unsubscribe()

	pending := make(map[snowflake.ID]struct{})
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case guildID := <-updates:
			pending[guildID] = struct{}{}
		case <-ticker.C:
			for guildID := range pending {
				delete(pending, guildID)
				h.RefreshPlayerMessage(guildID)
			}
		}
	}
}

func createRecentlyPlayedEmbed(prevTracks []lavalink.Track, startTime time.Time) discord.MessageCreate {
	var (
		inline         = true
		requesterCount = make(map[snowflake.ID]int)
		recent         string
		requesters     string
	)

	for i, track := range prevTracks {
		if i < 5 {
			recent += fmt.Sprintf("\n%d. %s", i+1, musicbot.TrackLink(track))
		}
		if r := musicbot.GetTrackMeta(track).Requester; r != 0 {
			requesterCount[r]++
		}
	}

	keys := make([]snowflake.ID, 0, len(requesterCount))
	for k := range requesterCount {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return requesterCount[keys[i]] > requesterCount[keys[j]]
	})
	for i := 0; i < min(5, len(keys)); i++ {
		requesters += fmt.Sprintf("%d. <@%s>\n", i+1, keys[i])
	}
	if requesters == "" {
		requesters = "-"
	}

	return discord.NewMessageCreateBuilder().
		SetEmbeds(discord.NewEmbedBuilder().
			SetTitle("Current Session").
			SetDescription(fmt.Sprintf("**Started:** <t:%d:R>", startTime.Unix())).
			SetFields(
				discord.EmbedField{Name: "Recent Tracks", Value: musicbot.Trim(recent, 1024), Inline: &inline},
				discord.EmbedField{Name: "Top Requesters", Value: requesters, Inline: &inline},
			).
			SetThumbnail(musicbot.TrackArtwork(prevTracks[0])).
			Build()).
		Build()
}

func (h *Handlers) showLyrics(event *events.ComponentInteractionCreate, guildID snowflake.ID) {
	player, ok := h.PlayerManager.GetPlayer(guildID)
	var track lavalink.Track
	if ok {
		track, ok = player.Current()
	}
	if !ok {
		_ = event.CreateMessage(discord.MessageCreate{Content: "Nothing is playing right now.", Flags: discord.MessageFlagEphemeral})
		return
	}
	if err := event.DeferCreateMessage(true); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	update := discord.MessageUpdate{}
	lyrics, err := h.LyricsForTrack(ctx, track)
	switch {
	case err == nil:
		embed, rows := commands.LyricsMessage(h.Bot, lyrics, guildID, musicbot.TrackArtwork(track))
		update.Embeds = &[]discord.Embed{embed}
		update.Components = &rows
	case errors.Is(err, musicbot.ErrLyricsNotFound):
		content := fmt.Sprintf("Couldn't find lyrics for **%s**. Try `/lyrics query:artist - song`.", musicbot.EscapeMarkdown(musicbot.TrackTitle(track)))
		update.Content = &content
	default:
		content := "The lyrics service isn't responding right now. Try again in a moment."
		update.Content = &content
		slog.Warn("lyrics lookup failed", slog.Any("error", err), slog.String("guild_id", guildID.String()))
	}

	if _, err := h.Client.Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), update); err != nil {
		musicbot.LogUpdateError(err, guildID.String(), event.User().ID.String())
	}
}
