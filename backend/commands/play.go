package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/json"
	"github.com/disgoorg/lavasearch-plugin"
	"github.com/disgoorg/lavasrc-plugin"
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
	Source   string
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

	ctx, cancel := context.WithTimeout(e.Ctx, autocompleteBudget)
	defer cancel()

	source := e.Data.String("source")
	searchType, typeOK := e.Data.OptString("type")

	if typeOK && searchType != "" && searchType != "track" {
		return e.AutocompleteResult(c.lavasearchChoices(ctx, query, source, searchType))
	}

	tracks, err := c.Searcher.SearchTracks(ctx, query, source, 20)
	if err != nil {
		slog.Debug("autocomplete search failed", slog.String("query", query), slog.Any("error", err))
	}

	choices := make([]discord.AutocompleteChoice, 0, len(tracks)+1)
	for _, track := range tracks {
		choices = append(choices, discord.AutocompleteChoiceString{
			Name: choiceName("🎵 %s — %s (%s)",
				musicbot.Trim(musicbot.TrackTitle(track), 60),
				musicbot.Trim(track.Info.Author, 20),
				musicbot.TrackDuration(track)),
			Value: c.Searcher.Remember(track),
		})
	}
	if len(choices) == 0 {
		choices = append(choices, fallbackChoice(query))
	}
	return e.AutocompleteResult(choices)
}

func (c *Commands) lavasearchChoices(ctx context.Context, query, source, searchType string) []discord.AutocompleteChoice {
	prefix := lavalink.SearchType("spsearch")
	if source == "deezer" {
		prefix = "dzsearch"
	}

	node := musicbot.BestNode(c.Lavalink)
	if node == nil {
		return []discord.AutocompleteChoice{fallbackChoice(query)}
	}

	type loaded struct {
		result *lavasearch.SearchResult
		err    error
	}
	done := make(chan loaded, 1)
	go func() {
		result, err := lavasearch.LoadSearch(node.Rest(), prefix.Apply(query), []lavasearch.SearchType{lavasearch.SearchType(searchType)})
		done <- loaded{result, err}
	}()

	var res loaded
	select {
	case res = <-done:
	case <-ctx.Done():
		return []discord.AutocompleteChoice{fallbackChoice(query)}
	}
	if res.err != nil || res.result == nil {
		if res.err != nil && !errors.Is(res.err, lavasearch.ErrEmptySearchResult) {
			slog.Debug("lavasearch failed", slog.Any("error", res.err))
		}
		return []discord.AutocompleteChoice{fallbackChoice(query)}
	}

	choices := make([]discord.AutocompleteChoice, 0, 25)
	add := func(name, value string) {
		if value == "" || len(value) > 100 || len(choices) >= 25 {
			return
		}
		choices = append(choices, discord.AutocompleteChoiceString{Name: name, Value: value})
	}

	for _, artist := range res.result.Artists {
		var info lavasrc.PlaylistInfo
		_ = artist.PluginInfo.Unmarshal(&info)
		add(choiceName("🎤 %s", artist.Info.Name), info.URL)
	}
	for _, album := range res.result.Albums {
		var info lavasrc.PlaylistInfo
		_ = album.PluginInfo.Unmarshal(&info)
		add(choiceName("💿 %s — %s", album.Info.Name, info.Author), info.URL)
	}
	for _, playlist := range res.result.Playlists {
		var info lavasrc.PlaylistInfo
		_ = playlist.PluginInfo.Unmarshal(&info)
		add(choiceName("🎧 %s — %s", playlist.Info.Name, info.Author), info.URL)
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

	playlist := lavalink.Playlist{
		Info: lavalink.PlaylistInfo{
			Name:          dbPlaylist.Name,
			SelectedTrack: -1,
		},
		Tracks: make([]lavalink.Track, 0, len(dbTracks)),
	}
	for _, track := range dbTracks {
		playlist.Tracks = append(playlist.Tracks, track.Track)
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
		return c.Searcher.Resolve(ctx, q, opts.Source)
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

func buildPlaylistEmbed(playlist lavalink.Playlist, requester snowflake.ID) discord.Embed {
	var (
		description  string
		lavasrcInfo  lavasrc.PlaylistInfo
		thumbnailUrl = ""
		playlistType = "playlist"
		numTracks    = len(playlist.Tracks)
		name         = musicbot.EscapeMarkdown(playlist.Info.Name)
	)

	_ = playlist.PluginInfo.Unmarshal(&lavasrcInfo)

	switch lavasrcInfo.Type {
	case lavasrc.PlaylistTypeArtist:
		playlistType = string(lavasrcInfo.Type)
		thumbnailUrl = lavasrcInfo.ArtworkURL
		description = fmt.Sprintf("[%s](%s) `%d tracks`\n\n<@%s>",
			musicbot.EscapeMarkdown(lavasrcInfo.Author), lavasrcInfo.URL, numTracks, requester)
	case lavasrc.PlaylistTypePlaylist, lavasrc.PlaylistTypeAlbum:
		playlistType = string(lavasrcInfo.Type)
		thumbnailUrl = lavasrcInfo.ArtworkURL
		description = fmt.Sprintf("[%s](%s) `%d track(s)`\n%s\n\n<@%s>",
			name, lavasrcInfo.URL, numTracks, musicbot.EscapeMarkdown(lavasrcInfo.Author), requester)
	default:
		description = fmt.Sprintf("%s `%d tracks`\n\n<@%s>", name, numTracks, requester)
	}

	return discord.NewEmbedBuilder().
		SetColor(musicbot.ColorMint).
		SetTitle(strings.ToUpper(playlistType[:1]) + playlistType[1:] + " queued").
		SetDescription(description).
		SetThumbnail(thumbnailUrl).
		Build()
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
	var unavailable *musicbot.SourceUnavailableError
	switch {
	case errors.As(err, &unavailable):
		return unavailable.Error() + "."
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
		case errors.Is(err, musicbot.ErrSelectionExpired):
			updateReply(e, "That search result expired, please search again.")
			return nil
		case errors.Is(err, musicbot.ErrPlaylistNotFound):
			updateReply(e, "Playlist not found.")
			return nil
		case errors.Is(err, musicbot.ErrNoNode):
			updateReply(e, c.voiceErrorMessage(err))
			return nil
		}
		var unavailable *musicbot.SourceUnavailableError
		if errors.As(err, &unavailable) {
			updateReply(e, c.voiceErrorMessage(err))
			return nil
		}
		updateReply(e, "Failed to load that track. Try a different query or source.")
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

		if playOpts.Shuffle != OptFalse {
			rand.Shuffle(len(loadData.Tracks), func(i, j int) {
				loadData.Tracks[i], loadData.Tracks[j] = loadData.Tracks[j], loadData.Tracks[i]
			})
		}
		tracks = loadData.Tracks
		meta.PlaylistName = loadData.Info.Name
		embed = buildPlaylistEmbed(loadData, userID)
	}

	if len(tracks) == 0 {
		updateReply(e, "No matches found for search query.")
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
			Source:   data.String("source"),
			Type:     LavalinkSearch,
			PlayNext: next,
			Loop:     loop,
			Shuffle:  shuffle,
		},
		e,
		cmd)
}
