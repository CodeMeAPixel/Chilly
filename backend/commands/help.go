package commands

import (
	"fmt"

	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

var help = discord.SlashCommandCreate{
	Name:        "help",
	Description: "Show available commands",
}

var invite = discord.SlashCommandCreate{
	Name:        "invite",
	Description: "Get a link to invite the bot",
}

func (c *Commands) inviteURL() string {
	const permissions = 37047296
	return fmt.Sprintf(
		"https://discord.com/oauth2/authorize?client_id=%s&scope=bot%%20applications.commands&permissions=%d",
		c.Client.ApplicationID(), permissions)
}

func (c *Commands) botAvatar() string {
	if self, ok := c.Client.Caches().SelfUser(); ok {
		return self.EffectiveAvatarURL()
	}
	return ""
}

func (c *Commands) Help(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	embed := discord.NewEmbedBuilder().
		SetColor(musicbot.ColorMint).
		SetAuthor("Chilly", c.SiteURL(""), c.botAvatar()).
		SetTitle("Commands").
		SetDescription(fmt.Sprintf("24/7 radio for your Discord server. Browse stations and control the player from the web at **[%s](%s)**.", c.SiteHost(), c.SiteURL("/radio")))
	if c.Radio != nil {
		embed.AddField("📻 Radio", "`/radio play` `/radio now` `/radio stations`", false).
			AddField("🌙 24/7", "`/247 on` `/247 off` `/247 status` · keep a station playing in a channel around the clock", false).
			AddField("🙋 Requests", "`/request` `/suggest` `/requests` · ask a station to play a song, or suggest one for the library", false)
	}
	embed.AddField("🎵 Music", "`/play` `/search` `/playlist` `/queue` `/now` `/lyrics` `/pause` `/resume` `/seek` `/skip` `/stop` `/shuffle` `/loop` `/remove`", false).
		AddField("Playlists", "`/list create` `/list add` `/list remove` `/list delete` `/list list`", false).
		AddField("Bot", "`/join` `/leave` `/help` `/invite`", false).
		SetFooter(c.SiteHost(), "")

	return e.CreateMessage(discord.MessageCreate{
		Embeds: []discord.Embed{embed.Build()},
		Components: []discord.ContainerComponent{
			discord.NewActionRow(
				discord.NewLinkButton("Stations", c.SiteURL("/radio")),
				discord.NewLinkButton("Dashboard", c.SiteURL("/dashboard")),
				discord.NewLinkButton("Status", c.SiteURL("/status")),
			),
		},
	})
}

func (c *Commands) Invite(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	return e.CreateMessage(discord.MessageCreate{
		Embeds: []discord.Embed{discord.NewEmbedBuilder().
			SetColor(musicbot.ColorMint).
			SetAuthor("Chilly", c.SiteURL(""), c.botAvatar()).
			SetTitle("Bring Chilly to your server").
			SetDescription(fmt.Sprintf("Add Chilly to any server you manage, join a voice channel and run `/radio play`. Use `/247 on` to keep a station playing around the clock. Learn more at **[%s](%s)**.", c.SiteHost(), c.SiteURL(""))).
			Build()},
		Components: []discord.ContainerComponent{
			discord.NewActionRow(
				discord.NewLinkButton("Add to server", c.inviteURL()),
				discord.NewLinkButton("Website", c.SiteURL("")),
			),
		},
	})
}
