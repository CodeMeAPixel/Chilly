package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Chilly/azuracast"
	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/lavalink"
)

var RadioCommand = discord.SlashCommandCreate{
	Name:        "radio",
	Description: "Listen to our radio stations",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionSubCommand{
			Name:        "play",
			Description: "Play a radio station in your voice channel",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:         "station",
					Description:  "Station to play",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "now",
			Description: "Show what's playing on a station",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:         "station",
					Description:  "Station (defaults to the one playing here)",
					Required:     false,
					Autocomplete: true,
				},
			},
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "stations",
			Description: "List available radio stations",
		},
	},
}

func RadioEmbed(np azuracast.NowPlaying) discord.Embed {
	builder := discord.NewEmbedBuilder().
		SetAuthor("📻 "+np.Station.Name, np.Station.PublicPlayerURL, "").
		SetColor(musicbot.ColorPink)

	if !np.IsOnline {
		return builder.SetDescription("This station is currently offline.").Build()
	}

	var description strings.Builder
	if np.NowPlaying != nil {
		song := np.NowPlaying.Song
		fmt.Fprintf(&description, "**%s**\n%s", musicbot.EscapeMarkdown(songTitle(song)), musicbot.EscapeMarkdown(song.Artist))
		if song.Album != "" {
			fmt.Fprintf(&description, " · *%s*", musicbot.EscapeMarkdown(song.Album))
		}
		if np.NowPlaying.Duration > 0 {
			elapsed := lavalink.Duration(np.NowPlaying.Elapsed) * lavalink.Second
			total := lavalink.Duration(np.NowPlaying.Duration) * lavalink.Second
			fmt.Fprintf(&description, "\n%s `%s | %s`", musicbot.ProgressBar(float32(elapsed)/float32(total)),
				musicbot.FormatTime(elapsed), musicbot.FormatTime(total))
		}
		if song.Art != "" {
			builder.SetThumbnail(song.Art)
		}
	}
	if np.Live.IsLive {
		fmt.Fprintf(&description, "\n\n🔴 Live: **%s**", musicbot.EscapeMarkdown(np.Live.StreamerName))
	}
	if np.PlayingNext != nil && np.PlayingNext.Song.Text != "" {
		fmt.Fprintf(&description, "\n\n**Up next:** %s", musicbot.EscapeMarkdown(np.PlayingNext.Song.Text))
	}

	return builder.
		SetDescription(musicbot.Trim(description.String(), 4000)).
		SetFooterText(fmt.Sprintf("%d listening", np.Listeners.Current)).
		Build()
}

func songTitle(song azuracast.Song) string {
	if song.Title != "" {
		return song.Title
	}
	if song.Text != "" {
		return song.Text
	}
	return "Unknown title"
}

func (c *Commands) RadioStationAutocomplete(e *handler.AutocompleteEvent) error {
	if c.Radio == nil {
		return e.AutocompleteResult(nil)
	}
	filter := strings.ToLower(strings.TrimSpace(e.Data.String("station")))
	choices := make([]discord.AutocompleteChoice, 0, 25)
	for _, np := range c.Radio.Stations() {
		if len(choices) >= 25 {
			break
		}
		if filter != "" && !strings.Contains(strings.ToLower(np.Station.Name), filter) &&
			!strings.Contains(strings.ToLower(np.Station.Shortcode), filter) {
			continue
		}
		name := "📻 " + np.Station.Name
		if !np.IsOnline {
			name += " (offline)"
		} else if np.NowPlaying != nil && np.NowPlaying.Song.Text != "" {
			name += " — " + np.NowPlaying.Song.Text
		}
		choices = append(choices, discord.AutocompleteChoiceString{
			Name:  musicbot.Trim(name, 100),
			Value: np.Station.Shortcode,
		})
	}
	return e.AutocompleteResult(choices)
}

func (c *Commands) RadioPlay(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if c.Radio == nil {
		ephemeral(e, "Radio is not enabled.")
		return nil
	}
	voiceChannelID, ok := c.UserVoiceChannel(*e.GuildID(), e.User().ID)
	if !ok {
		ephemeral(e, "You need to be in a voice channel to use this command.")
		return nil
	}
	np, ok := c.Radio.Station(data.String("station"))
	if !ok {
		ephemeral(e, "Unknown station. Pick one from the list.")
		return nil
	}
	if !np.IsOnline {
		ephemeral(e, fmt.Sprintf("**%s** is currently offline.", np.Station.Name))
		return nil
	}

	if err := e.DeferCreateMessage(false); err != nil {
		return err
	}
	defer musicbot.AutoRemove(e)

	ctx, cancel := context.WithTimeout(e.Ctx, 30*time.Second)
	defer cancel()

	track, err := c.LoadRadioTrack(ctx, np, musicbot.TrackMeta{Requester: e.User().ID})
	if err != nil {
		updateReply(e, "Failed to load the radio stream.")
		return err
	}

	if _, err := c.Enqueue(ctx, musicbot.EnqueueRequest{
		GuildID:        *e.GuildID(),
		VoiceChannelID: voiceChannelID,
		TextChannelID:  e.Channel().ID(),
		Tracks:         []lavalink.Track{track},
		PlayNow:        true,
	}); err != nil {
		updateReply(e, c.voiceErrorMessage(err))
		var busy *musicbot.BusyError
		if errors.As(err, &busy) {
			return nil
		}
		return err
	}

	if _, err := e.UpdateInteractionResponse(discord.MessageUpdate{
		Embeds: &[]discord.Embed{RadioEmbed(np)},
	}); err != nil {
		musicbot.LogUpdateError(err, e.GuildID().String(), e.User().ID.String())
	}
	return nil
}

func (c *Commands) RadioNow(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if c.Radio == nil {
		ephemeral(e, "Radio is not enabled.")
		return nil
	}

	key := data.String("station")
	if key == "" {
		if p, ok := c.PlayerManager.GetPlayer(*e.GuildID()); ok {
			if track, ok := p.Current(); ok {
				key = musicbot.GetTrackMeta(track).Radio
			}
		}
	}

	var (
		np azuracast.NowPlaying
		ok bool
	)
	if key != "" {
		np, ok = c.Radio.Station(key)
	} else if stations := c.Radio.Stations(); len(stations) > 0 {
		np, ok = stations[0], true
	}
	if !ok {
		ephemeral(e, "Station not found.")
		return nil
	}

	return e.CreateMessage(discord.MessageCreate{Embeds: []discord.Embed{RadioEmbed(np)}})
}

func (c *Commands) RadioStations(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if c.Radio == nil {
		ephemeral(e, "Radio is not enabled.")
		return nil
	}
	stations := c.Radio.Stations()
	if len(stations) == 0 {
		ephemeral(e, "No stations available right now.")
		return nil
	}

	var b strings.Builder
	for _, np := range stations {
		status := "🟢"
		if !np.IsOnline {
			status = "🔴"
		}
		fmt.Fprintf(&b, "%s **%s** `%s` · %d listening\n", status, musicbot.EscapeMarkdown(np.Station.Name), np.Station.Shortcode, np.Listeners.Current)
		if np.IsOnline && np.NowPlaying != nil && np.NowPlaying.Song.Text != "" {
			fmt.Fprintf(&b, "└ %s\n", musicbot.EscapeMarkdown(np.NowPlaying.Song.Text))
		}
	}

	return e.CreateMessage(discord.MessageCreate{
		Embeds: []discord.Embed{discord.NewEmbedBuilder().
			SetTitle("📻 Radio stations").
			SetDescription(musicbot.Trim(b.String(), 4000)).
			SetFooterText("Use /radio play to tune in · listen on the web at " + c.SiteHost()).
			Build()},
		Components: []discord.ContainerComponent{
			discord.NewActionRow(discord.NewLinkButton("Listen on the web", c.SiteURL("/radio"))),
		},
	})
}
