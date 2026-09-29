package commands

import (
	"fmt"
	"strings"

	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

func queueLine(i int, track lavalink.Track) string {
	line := fmt.Sprintf("\n%d. %s `%s`", i+1, musicbot.TrackLink(track), musicbot.TrackDuration(track))
	if track.Info.SourceName == musicbot.LibrarySource && track.Info.Author != "" {
		line += " " + musicbot.EscapeMarkdown(track.Info.Author)
	}
	return line
}

func nowPlayingContent(player *musicbot.Player, track lavalink.Track) string {
	meta := musicbot.GetTrackMeta(track)
	return fmt.Sprintf("%s\n%s\n%s\n\nRequested: <@%s>\n",
		musicbot.TrackLink(track), musicbot.EscapeMarkdown(track.Info.Author),
		musicbot.PlayerBar(player.IsPaused(), track, player.Position()), meta.Requester)
}

func (c *Commands) Now(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	player, ok := c.PlayerManager.GetPlayer(*e.GuildID())
	if !ok {
		ephemeral(e, "Player is not playing.")
		return nil
	}
	track, ok := player.Current()
	if !ok {
		ephemeral(e, "Player is not playing.")
		return nil
	}

	content := nowPlayingContent(player, track)
	if queue := player.Queue(); len(queue) > 0 {
		content += "\n**Up next:**" + strings.TrimPrefix(queueLine(0, queue[0]), "\n1.")
	}

	embed := discord.NewEmbedBuilder().
		SetTitle("Now playing").
		SetDescription(content).
		SetThumbnail(musicbot.TrackArtwork(track)).
		SetColor(musicbot.ColorMint).
		SetFooter("Manage the queue at "+c.SiteHost(), "")

	if sendErr := e.CreateMessage(discord.MessageCreate{
		Embeds:     []discord.Embed{embed.Build()},
		Components: c.dashboardRow(*e.GuildID()),
	}); sendErr != nil {
		musicbot.LogSendError(sendErr, e.GuildID().String(), e.User().ID.String(), false)
	}
	return nil
}

func (c *Commands) Queue(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	player, ok := c.PlayerManager.GetPlayer(*e.GuildID())
	if !ok {
		ephemeral(e, "Player is not playing.")
		return nil
	}
	track, ok := player.Current()
	if !ok {
		ephemeral(e, "Player is not playing.")
		return nil
	}

	queue := player.Queue()
	content := nowPlayingContent(player, track)
	if len(queue) > 0 {
		content += fmt.Sprintf("\n**Up next:** `%d track(s)`", len(queue))
		for i, t := range queue[:min(10, len(queue))] {
			content += queueLine(i, t)
		}
	}
	if loop := player.Loop(); loop != musicbot.LoopNone {
		content += fmt.Sprintf("\n\n🔁 Loop: `%s`", loop)
	}
	if player.Shuffle() {
		content += "\n🔀 Shuffle: `on`"
	}

	embed := discord.NewEmbedBuilder().
		SetTitle("Queue").
		SetDescription(musicbot.Trim(content, 4000)).
		SetThumbnail(musicbot.TrackArtwork(track)).
		SetColor(musicbot.ColorMint).
		SetFooter("Manage the queue at "+c.SiteHost(), "")

	if sendErr := e.CreateMessage(discord.MessageCreate{
		Embeds:     []discord.Embed{embed.Build()},
		Components: c.dashboardRow(*e.GuildID()),
	}); sendErr != nil {
		musicbot.LogSendError(sendErr, e.GuildID().String(), e.User().ID.String(), false)
	}
	return nil
}

func (c *Commands) dashboardRow(guildID snowflake.ID) []discord.ContainerComponent {
	return []discord.ContainerComponent{
		discord.NewActionRow(discord.NewLinkButton("Open in dashboard", c.SiteURL("/dashboard/"+guildID.String()))),
	}
}
