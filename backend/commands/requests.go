package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/CodeMeAPixel/Chilly/azuracast"
	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/lavalink"
)

var RequestCommands = []discord.ApplicationCommandCreate{
	discord.SlashCommandCreate{
		Name:        "request",
		Description: "Ask a station to play a song from the library soon",
		Contexts:    []discord.InteractionContextType{discord.InteractionContextTypeGuild},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:         "song",
				Description:  "Song from the library",
				Required:     true,
				Autocomplete: true,
			},
		},
	},
	discord.SlashCommandCreate{
		Name:        "suggest",
		Description: "Suggest a song that isn't in the library yet",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{Name: "artist", Description: "Artist name", Required: true, MaxLength: intPtr(200)},
			discord.ApplicationCommandOptionString{Name: "title", Description: "Song title", Required: true, MaxLength: intPtr(200)},
			discord.ApplicationCommandOptionString{Name: "link", Description: "A link that helps us find it (optional)", MaxLength: intPtr(500)},
			discord.ApplicationCommandOptionString{Name: "note", Description: "Anything we should know (optional)", MaxLength: intPtr(500)},
		},
	},
	discord.SlashCommandCreate{
		Name:        "requests",
		Description: "See your song requests and suggestions",
	},
}

func intPtr(v int) *int {
	return &v
}

var statusIcons = map[string]string{
	musicbot.RequestQueued:       "🕒",
	musicbot.RequestPlaying:      "🔊",
	musicbot.RequestPlayed:       "✅",
	musicbot.RequestExpired:      "⌛",
	musicbot.SuggestionPending:   "🕒",
	musicbot.SuggestionReviewing: "👀",
	musicbot.SuggestionAdded:     "✅",
	musicbot.SuggestionDeclined:  "✖️",
}

func (c *Commands) RequestAutocomplete(e *handler.AutocompleteEvent) error {
	query := strings.TrimSpace(e.Data.String("song"))
	if query == "" {
		return e.AutocompleteResult(nil)
	}
	tracks := c.Searcher.Search(query, 25)
	choices := make([]discord.AutocompleteChoice, 0, len(tracks))
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
	return e.AutocompleteResult(choices)
}

func (c *Commands) Request(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if c.Requests == nil {
		ephemeral(e, musicbot.ErrRequestsUnavailable.Error()+".")
		return nil
	}
	key := data.String("song")
	if _, _, ok := musicbot.ParseLibraryKey(key); !ok {
		matches := c.Searcher.Search(key, 1)
		if len(matches) == 0 {
			ephemeral(e, fmt.Sprintf("That isn't in the library yet. Suggest it with `/suggest`, or browse everything at %s", c.SiteURL("/tracks")))
			return nil
		}
		key = matches[0].Key()
	}

	if err := e.DeferCreateMessage(true); err != nil {
		return err
	}
	req, err := c.Requests.Submit(e.Ctx, e.User().ID, *e.GuildID(), key, "discord")
	if err != nil {
		var (
			limit   *musicbot.LimitError
			station *azuracast.RequestError
		)
		switch {
		case errors.As(err, &limit):
			updateReply(e, limit.Message)
		case errors.As(err, &station):
			updateReply(e, "The station couldn't take that request: "+station.Message)
		case errors.Is(err, musicbot.ErrSelectionExpired), errors.Is(err, musicbot.ErrRequestsUnavailable):
			updateReply(e, err.Error()+".")
		default:
			updateReply(e, "Couldn't reach the station right now. Try again in a moment.")
			return err
		}
		return nil
	}

	stationName := req.Station
	if c.Radio != nil {
		if np, ok := c.Radio.Station(req.Station); ok {
			stationName = np.Station.Name
		}
	}
	embed := discord.NewEmbedBuilder().
		SetColor(musicbot.ColorPink).
		SetTitle("Request sent").
		SetDescription(fmt.Sprintf("**%s** by %s will play on **%s** soon. Tune in with `/radio play`.\n\nTrack it at %s",
			musicbot.EscapeMarkdown(req.Title), musicbot.EscapeMarkdown(req.Artist), musicbot.EscapeMarkdown(stationName),
			c.SiteURL("/dashboard/requests")))
	if req.Art != "" {
		embed.SetThumbnail(req.Art)
	}
	if _, err := e.UpdateInteractionResponse(discord.MessageUpdate{Embeds: &[]discord.Embed{embed.Build()}}); err != nil {
		musicbot.LogUpdateError(err, e.GuildID().String(), e.User().ID.String())
	}
	return nil
}

func (c *Commands) Suggest(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if c.Requests == nil {
		ephemeral(e, musicbot.ErrRequestsUnavailable.Error()+".")
		return nil
	}
	s, err := c.Requests.Suggest(e.Ctx, e.User().ID, data.String("artist"), data.String("title"), data.String("link"), data.String("note"))
	if err != nil {
		var (
			limit     *musicbot.LimitError
			inLibrary *musicbot.AlreadyInLibraryError
		)
		switch {
		case errors.As(err, &limit):
			ephemeral(e, limit.Message)
		case errors.As(err, &inLibrary):
			ephemeral(e, fmt.Sprintf("Good news: **%s** by %s is already in the library! Play it with `/play` or request it with `/request`.",
				musicbot.EscapeMarkdown(inLibrary.Track.Title), musicbot.EscapeMarkdown(inLibrary.Track.Artist)))
		default:
			ephemeral(e, "Couldn't save your suggestion. Try again in a moment.")
			return err
		}
		return nil
	}
	ephemeral(e, fmt.Sprintf("Thanks! **%s** by %s is in the review queue. I'll DM you when it's added or reviewed, and you can follow it at %s",
		musicbot.EscapeMarkdown(s.Title), musicbot.EscapeMarkdown(s.Artist), c.SiteURL("/dashboard/requests")))
	return nil
}

func (c *Commands) MyRequests(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	requests, err := c.Db.UserRequests(e.Ctx, e.User().ID, 5)
	if err != nil {
		ephemeral(e, "Couldn't load your requests.")
		return err
	}
	suggestions, err := c.Db.UserSuggestions(e.Ctx, e.User().ID, 5)
	if err != nil {
		ephemeral(e, "Couldn't load your suggestions.")
		return err
	}

	builder := discord.NewEmbedBuilder().SetColor(musicbot.ColorMint).SetTitle("Your requests")
	var lines strings.Builder
	for _, r := range requests {
		fmt.Fprintf(&lines, "%s **%s** — %s · %s <t:%d:R>\n", statusIcons[r.Status], musicbot.EscapeMarkdown(musicbot.Trim(r.Title, 60)),
			musicbot.EscapeMarkdown(musicbot.Trim(r.Artist, 30)), r.Status, r.CreatedAt.Unix())
	}
	if lines.Len() == 0 {
		lines.WriteString("No requests yet. Try `/request`.")
	}
	builder.AddField("Requests", musicbot.Trim(lines.String(), 1024), false)

	lines.Reset()
	for _, s := range suggestions {
		fmt.Fprintf(&lines, "%s **%s** — %s · %s\n", statusIcons[s.Status], musicbot.EscapeMarkdown(musicbot.Trim(s.Title, 60)),
			musicbot.EscapeMarkdown(musicbot.Trim(s.Artist, 30)), s.Status)
	}
	if lines.Len() == 0 {
		lines.WriteString("No suggestions yet. Missing a song? Try `/suggest`.")
	}
	builder.AddField("Suggestions", musicbot.Trim(lines.String(), 1024), false)

	return e.CreateMessage(discord.MessageCreate{
		Embeds: []discord.Embed{builder.Build()},
		Components: []discord.ContainerComponent{
			discord.NewActionRow(discord.NewLinkButton("See everything", c.SiteURL("/dashboard/requests"))),
		},
		Flags: discord.MessageFlagEphemeral,
	})
}
