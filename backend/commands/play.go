package commands

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
)

const autocompleteBudget = 2500 * time.Millisecond

type SearchType int

const (
	LavalinkSearch SearchType = 0
	PlaylistSearch SearchType = 1
)

type OptBool int

const (
	OptTrue  OptBool = 1
	OptFalse OptBool = 0
	OptUnset OptBool = -1
)

type UserData = musicbot.TrackMeta

type PlayOpts struct {
	Query    any
	Kind     string
	Type     SearchType
	PlayNext OptBool
	Loop     OptBool
	Shuffle  OptBool
}

func optBoolValue(b bool, ok bool) OptBool {
	if !ok {
		return OptUnset
	}
	if b {
		return OptTrue
	}
	return OptFalse
}

func readOptBools(data discord.SlashCommandInteractionData) (next, loop, shuffle OptBool) {
	n, ok := data.OptBool("next")
	next = optBoolValue(n, ok)
	l, ok := data.OptBool("loop")
	loop = optBoolValue(l, ok)
	s, ok := data.OptBool("shuffle")
	shuffle = optBoolValue(s, ok)
	return
}

func choiceName(format string, args ...any) string {
	return musicbot.Trim(fmt.Sprintf(format, args...), 100)
}

func fallbackChoice(query string) discord.AutocompleteChoice {
	return discord.AutocompleteChoiceString{
		Name:  choiceName("🔎 Search: %s", query),
		Value: musicbot.Trim(query, 100),
	}
}

func (c *Commands) SearchAutocomplete(e *handler.AutocompleteEvent) error {
	query := strings.TrimSpace(e.Data.String("query"))
	if query == "" {
		return e.AutocompleteResult(nil)
	}

	kind := e.Data.String("type")
	if kind == string(musicbot.GroupAlbum) || kind == string(musicbot.GroupArtist) || kind == string(musicbot.GroupPlaylist) {
		return e.AutocompleteResult(c.groupChoices(query, musicbot.LibraryGroup(kind)))
	}

	tracks := c.Searcher.Search(query, 25)
	choices := make([]discord.AutocompleteChoice, 0, len(tracks)+1)
	for _, track := range tracks {
		name := musicbot.Trim(track.Title, 60)
		if track.Artist != "" {
			name += " — " + musicbot.Trim(track.Artist, 25)
		}
		choices = append(choices, discord.AutocompleteChoiceString{
			Name:  choiceName("🎵 %s (%s)", name, musicbot.FormatTime(lavalink.Duration(track.LengthMs))),
			Value: track.Key(),
		})
	}
	if len(choices) == 0 {
		choices = append(choices, fallbackChoice(query))
	}
	return e.AutocompleteResult(choices)
}

func (c *Commands) groupChoices(query string, kind musicbot.LibraryGroup) []discord.AutocompleteChoice {
	icon := map[musicbot.LibraryGroup]string{musicbot.GroupAlbum: "💿", musicbot.GroupArtist: "🎤", musicbot.GroupPlaylist: "🎧"}[kind]
	seen := make(map[string]bool)
	choices := make([]discord.AutocompleteChoice, 0, 25)
	for _, track := range c.Searcher.Search(query, 200) {
		var names []string
		switch kind {
		case musicbot.GroupAlbum:
			names = []string{track.Album}
		case musicbot.GroupArtist:
			names = []string{track.Artist}
		case musicbot.GroupPlaylist:
			names = track.Playlists
		}
		for _, name := range names {
			if name == "" || seen[strings.ToLower(name)] || len(name) > 100 || len(choices) >= 25 {
				continue
			}
			seen[strings.ToLower(name)] = true
			label := name
			if kind == musicbot.GroupAlbum && track.Artist != "" {
				label += " — " + track.Artist
			}
			choices = append(choices, discord.AutocompleteChoiceString{Name: choiceName("%s %s", icon, label), Value: name})
		}
	}
	if len(choices) == 0 {
		choices = append(choices, fallbackChoice(query))
	}
	return choices
}

func SearchPlaylist(ctx context.Context, playlistId int, c *Commands, userId snowflake.ID) (*lavalink.LoadResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	dbPlaylist, dbTracks, err := c.Db.GetPlaylist(ctx, userId, playlistId)
	if err != nil {
		return nil, err
	}
	if len(dbTracks) == 0 {
		return &lavalink.LoadResult{LoadType: lavalink.LoadTypeEmpty, Data: lavalink.Empty{}}, nil
	}

	tracks, missing, err := c.Searcher.LoadPlaylistTracks(ctx, dbTracks)
	if err != nil {
		return nil, err
	}
	if len(tracks) == 0 {
		return nil, ErrPlaylistNotInLibrary
	}
	playlist := lavalink.Playlist{
		Info:   lavalink.PlaylistInfo{Name: dbPlaylist.Name, SelectedTrack: -1},
		Tracks: tracks,
	}
	if missing > 0 {
		playlist.PluginInfo = lavalink.RawData(fmt.Sprintf(`{"missing":%d}`, missing))
	}

	return &lavalink.LoadResult{
		LoadType: lavalink.LoadTypePlaylist,
		Data:     playlist,
	}, nil
}

func SearchQuery(ctx context.Context, opts PlayOpts, c *Commands, userId snowflake.ID) (*lavalink.LoadResult, error) {
	switch opts.Type {
	case LavalinkSearch:
		q, ok := opts.Query.(string)
		if !ok {
			return nil, fmt.Errorf("query should be a string for Lavalink search, got %T", opts.Query)
		}
		return c.Searcher.Resolve(ctx, q, opts.Kind)
	case PlaylistSearch:
		q, ok := opts.Query.(int)
		if !ok {
			return nil, fmt.Errorf("query should be an int for Playlist search, got %T", opts.Query)
		}
		return SearchPlaylist(ctx, q, c, userId)
	default:
		return nil, fmt.Errorf("unknown search type: %v", opts.Type)
	}
}

func buildTrackEmbed(track lavalink.Track, queued bool, position int) discord.Embed {
	description := fmt.Sprintf("Starting %s · `%s`", musicbot.TrackLink(track), musicbot.TrackDuration(track))
	if queued {
		description = fmt.Sprintf("Queued %s · `%s`\nPosition **#%d** in the queue", musicbot.TrackLink(track), musicbot.TrackDuration(track), position)
	}
	return discord.NewEmbedBuilder().
		SetColor(musicbot.ColorMint).
		SetDescription(description).
		Build()
}

var ErrPlaylistNotInLibrary = errors.New("none of the songs in that playlist are in the library yet")

func buildPlaylistEmbed(playlist lavalink.Playlist, kind string, requester snowflake.ID) discord.Embed {
	label := "Playlist"
	switch musicbot.LibraryGroup(kind) {
	case musicbot.GroupAlbum:
		label = "Album"
	case musicbot.GroupArtist:
		label = "Artist"
	}
	description := fmt.Sprintf("**%s** `%d tracks`", musicbot.EscapeMarkdown(playlist.Info.Name), len(playlist.Tracks))
	var extra struct {
		Missing int `json:"missing"`
	}
	if len(playlist.PluginInfo) > 0 && playlist.PluginInfo.Unmarshal(&extra) == nil && extra.Missing > 0 {
		description += fmt.Sprintf("\n%d song(s) skipped because they aren't in the library yet.", extra.Missing)
	}
	builder := discord.NewEmbedBuilder().
		SetColor(musicbot.ColorMint).
		SetTitle(label + " queued").
		SetDescription(fmt.Sprintf("%s\n\n<@%s>", description, requester))
	if len(playlist.Tracks) > 0 {
		builder.SetThumbnail(musicbot.TrackArtwork(playlist.Tracks[0]))
	}
	return builder.Build()
}

func updateReply(e *handler.CommandEvent, content string) {
	if _, err := e.UpdateInteractionResponse(discord.MessageUpdate{
		Content: json.Ptr(content),
		Embeds:  &[]discord.Embed{},
	}); err != nil {
		musicbot.LogUpdateError(err, e.GuildID().String(), e.User().ID.String())
	}
}

func (c *Commands) voiceErrorMessage(err error) string {
	var busy *musicbot.BusyError
	switch {
	case errors.As(err, &busy):
		return fmt.Sprintf("I'm already playing in <#%s>. Join that channel to add tracks.", busy.ChannelID)
	case errors.Is(err, musicbot.ErrVoiceTimeout):
		return "Couldn't connect to your voice channel in time. Check my permissions and try again."
	case errors.Is(err, musicbot.ErrNoNode):
		return fmt.Sprintf("The music server is currently unavailable. Check %s for updates.", c.SiteURL("/status"))
	default:
		return "Failed to start playback."
	}
}

func HandlePlay(playOpts PlayOpts, e *handler.CommandEvent, c *Commands) error {
	guildID := *e.GuildID()
	userID := e.User().ID

	voiceChannelID, ok := c.UserVoiceChannel(guildID, userID)
	if !ok {
		updateReply(e, "You need to be in a voice channel to use this command.")
		return nil
	}

	ctx, cancel := context.WithTimeout(e.Ctx, 30*time.Second)
	defer cancel()

	result, err := SearchQuery(ctx, playOpts, c, userID)
	if err != nil {
		switch {
		case errors.Is(err, musicbot.ErrSelectionExpired), errors.Is(err, musicbot.ErrExternalSource), errors.Is(err, musicbot.ErrLibraryUnavailable):
			updateReply(e, err.Error()+".")
			return nil
		case errors.Is(err, musicbot.ErrPlaylistNotFound):
			updateReply(e, "Playlist not found.")
			return nil
		case errors.Is(err, ErrPlaylistNotInLibrary):
			updateReply(e, "None of the songs in that playlist are in the library yet. They'll start working as soon as they're added.")
			return nil
		case errors.Is(err, musicbot.ErrNoNode):
			updateReply(e, c.voiceErrorMessage(err))
			return nil
		}
		updateReply(e, "Failed to load that song. Try again in a moment.")
		return err
	}

	var (
		tracks []lavalink.Track
		embed  discord.Embed
		meta   = musicbot.TrackMeta{Requester: userID}
	)

	switch loadData := result.Data.(type) {
	case lavalink.Track:
		tracks = []lavalink.Track{loadData}
	case lavalink.Search:
		if len(loadData) > 0 {
			tracks = []lavalink.Track{loadData[0]}
		}
	case lavalink.Playlist:

		if playOpts.Shuffle == OptTrue || (playOpts.Shuffle == OptUnset && playOpts.Kind != string(musicbot.GroupAlbum)) {
			rand.Shuffle(len(loadData.Tracks), func(i, j int) {
				loadData.Tracks[i], loadData.Tracks[j] = loadData.Tracks[j], loadData.Tracks[i]
			})
		}
		tracks = loadData.Tracks
		meta.PlaylistName = loadData.Info.Name
		embed = buildPlaylistEmbed(loadData, playOpts.Kind, userID)
	}

	if len(tracks) == 0 {
		updateReply(e, "That isn't in the library yet. Try another search, or browse everything on the website.")
		return nil
	}
	tracks = musicbot.WithTrackMeta(tracks, meta)

	req := musicbot.EnqueueRequest{
		GuildID:        guildID,
		VoiceChannelID: voiceChannelID,
		TextChannelID:  e.Channel().ID(),
		Tracks:         tracks,
		Next:           playOpts.PlayNext == OptTrue,
	}
	switch playOpts.Loop {
	case OptTrue:
		req.Loop = ptr(musicbot.LoopQueue)
	case OptFalse:
		req.Loop = ptr(musicbot.LoopNone)
	}
	switch playOpts.Shuffle {
	case OptTrue:
		req.Shuffle = ptr(musicbot.ShuffleOn)
	case OptFalse:
		req.Shuffle = ptr(musicbot.ShuffleOff)
	}

	wasPlaying := false
	if p, exists := c.PlayerManager.GetPlayer(guildID); exists {
		wasPlaying = p.IsPlaying()
	}

	player, err := c.Enqueue(ctx, req)
	if err != nil {
		updateReply(e, c.voiceErrorMessage(err))
		var busy *musicbot.BusyError
		if errors.As(err, &busy) {
			return nil
		}
		return err
	}

	if len(tracks) == 1 {
		position := 1
		if !req.Next && player != nil {
			position = len(player.Queue())
		}
		embed = buildTrackEmbed(tracks[0], wasPlaying, position)
	}
	if _, updateErr := e.UpdateInteractionResponse(discord.MessageUpdate{
		Embeds: &[]discord.Embed{embed},
	}); updateErr != nil {
		musicbot.LogUpdateError(updateErr, guildID.String(), userID.String())
	}
	return nil
}

func ptr[T any](v T) *T {
	return &v
}

func (cmd *Commands) PlayPlaylist(data discord.SlashCommandInteractionData, event *handler.CommandEvent) error {
	if !requireUserVoice(cmd, event) {
		return nil
	}

	if err := event.DeferCreateMessage(false); err != nil {
		return err
	}

	defer musicbot.AutoRemove(event)

	next, loop, shuffle := readOptBools(data)
	return HandlePlay(
		PlayOpts{
			Query:    data.Int("playlist"),
			Type:     PlaylistSearch,
			PlayNext: next,
			Loop:     loop,
			Shuffle:  shuffle,
		},
		event,
		cmd)
}

func (cmd *Commands) Play(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !requireUserVoice(cmd, e) {
		return nil
	}

	if err := e.DeferCreateMessage(false); err != nil {
		return err
	}

	defer musicbot.AutoRemove(e)

	next, loop, shuffle := readOptBools(data)
	return HandlePlay(
		PlayOpts{
			Query:    data.String("query"),
			Kind:     data.String("type"),
			Type:     LavalinkSearch,
			PlayNext: next,
			Loop:     loop,
			Shuffle:  shuffle,
		},
		e,
		cmd)
}
