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
	"github.com/disgoorg/snowflake/v2"
)

const maxLyricsLength = 3900

func LyricsMessage(bot *musicbot.Bot, lyrics *musicbot.Lyrics, guildID snowflake.ID, artwork string) (discord.Embed, []discord.ContainerComponent) {
	body := lyrics.Plain
	switch {
	case lyrics.Instrumental:
		body = "*This track is instrumental.*"
	case body == "":
		body = "*No lyrics text available.*"
	case len([]rune(body)) > maxLyricsLength:
		body = string([]rune(body)[:maxLyricsLength])
		if cut := strings.LastIndex(body, "\n"); cut > maxLyricsLength/2 {
			body = body[:cut]
		}
		body += "\n\n*…continued on the dashboard*"
	}

	footer := "Lyrics from " + lyrics.Source
	if len(lyrics.Synced) > 0 {
		footer += " · synced lyrics on " + bot.SiteHost()
	}

	embed := discord.NewEmbedBuilder().
		SetColor(musicbot.ColorPeach).
		SetAuthor("Lyrics", "", "").
		SetTitle(musicbot.Trim(fmt.Sprintf("%s — %s", lyrics.Title, lyrics.Artist), 256)).
		SetDescription(musicbot.EscapeMarkdown(body)).
		SetThumbnail(artwork).
		SetFooter(footer, "").
		Build()

	var rows []discord.ContainerComponent
	if guildID != 0 {
		rows = []discord.ContainerComponent{
			discord.NewActionRow(discord.NewLinkButton("Sing along on the dashboard", bot.SiteURL("/dashboard/"+guildID.String()+"?lyrics=1"))),
		}
	}
	return embed, rows
}

func parseSongQuery(query string) musicbot.SongQuery {
	for _, sep := range []string{" - ", " – ", " — ", " by "} {
		if left, right, ok := strings.Cut(query, sep); ok {
			if sep == " by " {
				return musicbot.SongQuery{Title: strings.TrimSpace(left), Artist: strings.TrimSpace(right)}
			}
			return musicbot.SongQuery{Title: strings.TrimSpace(right), Artist: strings.TrimSpace(left)}
		}
	}
	return musicbot.SongQuery{Title: strings.TrimSpace(query)}
}

func (c *Commands) Lyrics(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	query := strings.TrimSpace(data.String("query"))

	if query == "" {
		player, ok := c.PlayerManager.GetPlayer(*e.GuildID())
		if !ok || !player.IsPlaying() {
			ephemeral(e, "Nothing is playing. Try `/lyrics query:artist - song`.")
			return nil
		}
	}

	if err := e.DeferCreateMessage(false); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(e.Ctx, 15*time.Second)
	defer cancel()

	var (
		lyrics  *musicbot.Lyrics
		err     error
		artwork string
	)
	if query != "" {
		lyrics, err = c.Bot.Lyrics.Lookup(ctx, parseSongQuery(query))
	} else if player, ok := c.PlayerManager.GetPlayer(*e.GuildID()); ok {
		if track, playing := player.Current(); playing {
			artwork = musicbot.TrackArtwork(track)
			lyrics, err = c.LyricsForTrack(ctx, track)
		} else {
			err = musicbot.ErrLyricsNotFound
		}
	}

	if err != nil {
		if errors.Is(err, musicbot.ErrLyricsNotFound) {
			updateReply(e, "Couldn't find lyrics for that song. Try `/lyrics query:artist - song`.")
			return nil
		}
		updateReply(e, "The lyrics service isn't responding right now. Try again in a moment.")
		return err
	}

	embed, rows := LyricsMessage(c.Bot, lyrics, *e.GuildID(), artwork)
	_, updateErr := e.UpdateInteractionResponse(discord.MessageUpdate{
		Embeds:     &[]discord.Embed{embed},
		Components: &rows,
	})
	if updateErr != nil {
		musicbot.LogUpdateError(updateErr, e.GuildID().String(), e.User().ID.String())
	}
	return nil
}
