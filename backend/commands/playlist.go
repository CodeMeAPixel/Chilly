package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/lavasrc-plugin"
)

var playlist = discord.SlashCommandCreate{
	Name:        "list",
	Description: "playlist commands",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionSubCommand{
			Name:        "create",
			Description: "Create new playlist",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "playlist_name",
					Description: "Playlist name",
					Required:    true,
				},
			}},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "delete",
			Description: "Delete playlist",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionInt{
					Name:         "playlist",
					Description:  "Playlist name",
					Required:     true,
					Autocomplete: true,
				},
			}},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "list",
			Description: "List playlist",
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "add",
			Description: "Add track(s) to playlist",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:         "query",
					Description:  "Search query",
					Required:     true,
					Autocomplete: true,
				},
				discord.ApplicationCommandOptionInt{
					Name:         "playlist",
					Description:  "Playlist name",
					Required:     true,
					Autocomplete: true,
				},
				discord.ApplicationCommandOptionString{
					Name:        "source",
					Description: "Source to search from",
					Required:    false,
					Choices:     searchSourceChoices,
				},
				discord.ApplicationCommandOptionString{
					Name:        "type",
					Description: "Type of search",
					Required:    false,
					Choices:     searchTypeChoices,
				},
			}},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "remove",
			Description: "Remove track from playlist",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionInt{
					Name:         "playlist",
					Description:  "Playlist to remove track from",
					Required:     true,
					Autocomplete: true,
				},
				discord.ApplicationCommandOptionInt{
					Name:         "track",
					Description:  "Track to remove",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
	}}

func (c *Commands) PlaylistAutocomplete(e *handler.AutocompleteEvent) error {
	playlists, err := c.Db.SearchPlaylist(e.Ctx, e.User().ID, e.Data.String("playlist"), 25)
	if err != nil {
		return e.AutocompleteResult(nil)
	}

	choices := make([]discord.AutocompleteChoice, 0, len(playlists))
	for _, playlist := range playlists {
		choices = append(choices, discord.AutocompleteChoiceInt{
			Name:  choiceName("%s (%d tracks)", playlist.Name, playlist.TrackCount),
			Value: playlist.ID,
		})
	}
	return e.AutocompleteResult(choices)
}

func (c *Commands) PlaylistTrackAutocomplete(e *handler.AutocompleteEvent) error {
	playlistId := e.Data.Int("playlist")

	if playlistId <= 0 {
		return e.AutocompleteResult(nil)
	}

	_, playlistTracks, err := c.Db.GetPlaylist(e.Ctx, e.User().ID, playlistId)
	if err != nil || len(playlistTracks) == 0 {
		return e.AutocompleteResult(nil)
	}

	filter := strings.ToLower(e.Data.String("track"))
	choices := make([]discord.AutocompleteChoice, 0, 25)
	for _, playlistTrack := range playlistTracks {
		if len(choices) >= 25 {
			break
		}
		if filter != "" && !strings.Contains(strings.ToLower(playlistTrack.TrackTitle), filter) {
			continue
		}
		choices = append(choices, discord.AutocompleteChoiceInt{
			Name:  choiceName("%s", playlistTrack.TrackTitle),
			Value: playlistTrack.ID,
		})
	}
	return e.AutocompleteResult(choices)
}

func (c *Commands) AddPlaylistTrackAutocomplete(e *handler.AutocompleteEvent) error {
	switch e.Data.Focused().Name {
	case "playlist":
		return c.PlaylistAutocomplete(e)
	case "query":
		return c.SearchAutocomplete(e)
	}
	return e.AutocompleteResult(nil)
}

func (c *Commands) RemovePlaylistTrackAutocomplete(e *handler.AutocompleteEvent) error {
	switch e.Data.Focused().Name {
	case "playlist":
		return c.PlaylistAutocomplete(e)
	case "track":
		return c.PlaylistTrackAutocomplete(e)
	}
	return e.AutocompleteResult(nil)
}

func (c *Commands) CreatePlaylist(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	playlistName := strings.TrimSpace(data.String("playlist_name"))
	if playlistName == "" || len([]rune(playlistName)) > 100 {
		ephemeral(e, "Playlist names must be between 1 and 100 characters.")
		return nil
	}

	if _, err := c.Db.CreatePlaylist(e.Ctx, e.User().ID, playlistName); err != nil {
		if errors.Is(err, musicbot.ErrPlaylistExists) {
			ephemeral(e, fmt.Sprintf("You already have a playlist named `%s`.", playlistName))
			return nil
		}
		ephemeral(e, "Failed to create playlist.")
		return err
	}
	reply(e, fmt.Sprintf("📋 Playlist `%s` created", musicbot.EscapeMarkdown(playlistName)))
	return nil
}

func (c *Commands) DeletePlaylist(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if err := c.Db.DeletePlaylist(e.Ctx, e.User().ID, data.Int("playlist")); err != nil {
		if errors.Is(err, musicbot.ErrPlaylistNotFound) {
			ephemeral(e, "Playlist not found.")
			return nil
		}
		ephemeral(e, "Failed to delete playlist.")
		return err
	}
	reply(e, "📋 Playlist deleted")
	return nil
}

func (c *Commands) ListPlaylists(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	playlists, err := c.Db.SearchPlaylist(e.Ctx, e.User().ID, "", 25)
	if err != nil {
		ephemeral(e, "Failed to retrieve your playlists.")
		return err
	}
	if len(playlists) == 0 {
		ephemeral(e, fmt.Sprintf("You don't have any playlists yet. Create one with `/list create` or at %s.", c.SiteURL("/dashboard/playlists")))
		return nil
	}

	content := fmt.Sprintf("<@%s>'s playlists\n", e.User().ID)
	for _, playlist := range playlists {
		content += fmt.Sprintf("- `%s` %d tracks, created <t:%d:R>\n",
			playlist.Name, playlist.TrackCount, playlist.CreatedAt.Unix())
	}

	if err := e.CreateMessage(discord.MessageCreate{
		Embeds: []discord.Embed{discord.NewEmbedBuilder().
			SetColor(musicbot.ColorMint).
			SetTitle("Playlists").
			SetDescription(musicbot.Trim(content, 4000)).
			SetFooter("Edit your playlists at "+c.SiteHost(), "").
			Build()},
		Components: []discord.ContainerComponent{
			discord.NewActionRow(discord.NewLinkButton("Manage playlists", c.SiteURL("/dashboard/playlists"))),
		},
	}); err != nil {
		musicbot.LogSendError(err, e.GuildID().String(), e.User().ID.String(), false)
	}
	return nil
}

func (c *Commands) AddPlaylistTrack(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	var (
		playlistID = data.Int("playlist")
		query      = data.String("query")
	)

	if err := e.DeferCreateMessage(false); err != nil {
		return err
	}
	defer musicbot.AutoRemove(e)

	ctx, cancel := context.WithTimeout(e.Ctx, 30*time.Second)
	defer cancel()

	playlist, _, err := c.Db.GetPlaylist(ctx, e.User().ID, playlistID)
	if err != nil {
		if errors.Is(err, musicbot.ErrPlaylistNotFound) {
			updateReply(e, "Playlist not found.")
			return nil
		}
		updateReply(e, "Failed to retrieve playlist.")
		return err
	}

	result, err := c.Searcher.Resolve(ctx, query, data.String("source"))
	if err != nil {
		updateReply(e, "Failed to load that track.")
		if errors.Is(err, musicbot.ErrSelectionExpired) {
			return nil
		}
		return err
	}

	var (
		tracks      []lavalink.Track
		description string
	)
	switch loadData := result.Data.(type) {
	case lavalink.Track:
		tracks = []lavalink.Track{loadData}
		description = fmt.Sprintf("%s added to playlist `%s`", musicbot.TrackLink(loadData), playlist.Name)
	case lavalink.Search:
		if len(loadData) > 0 {
			tracks = []lavalink.Track{loadData[0]}
			description = fmt.Sprintf("%s added to playlist `%s`", musicbot.TrackLink(loadData[0]), playlist.Name)
		}
	case lavalink.Playlist:
		tracks = loadData.Tracks
		var info lavasrc.PlaylistInfo
		if err := loadData.PluginInfo.Unmarshal(&info); err == nil && info.URL != "" {
			description = fmt.Sprintf("Playlist [%s](%s) `%d tracks` added to playlist `%s`",
				musicbot.EscapeMarkdown(loadData.Info.Name), info.URL, len(loadData.Tracks), playlist.Name)
		} else {
			description = fmt.Sprintf("Playlist %s `%d tracks` added to playlist `%s`",
				musicbot.EscapeMarkdown(loadData.Info.Name), len(loadData.Tracks), playlist.Name)
		}
	}

	if len(tracks) == 0 {
		updateReply(e, "No matches found for that query.")
		return nil
	}
	if err := c.Db.AddTracksToPlaylist(ctx, playlistID, e.User().ID, tracks); err != nil {
		updateReply(e, "Failed to save tracks to the playlist.")
		return err
	}

	if _, updateErr := e.UpdateInteractionResponse(discord.MessageUpdate{
		Embeds: &[]discord.Embed{{Description: description}},
	}); updateErr != nil {
		musicbot.LogUpdateError(updateErr, e.GuildID().String(), e.User().ID.String())
	}
	return nil
}

func (c *Commands) RemovePlaylistTrack(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	err := c.Db.RemoveTrackFromPlaylist(e.Ctx, e.User().ID, data.Int("playlist"), data.Int("track"))
	if err != nil {
		if errors.Is(err, musicbot.ErrPlaylistNotFound) {
			ephemeral(e, "Track not found in that playlist.")
			return nil
		}
		ephemeral(e, "Failed to remove track.")
		return err
	}
	reply(e, "Track removed from playlist.")
	return nil
}
