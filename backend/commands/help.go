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
	const permissions = 36775152
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
		SetDescription(fmt.Sprintf("Everything Chilly can do. Control the player and manage playlists from the web at **[%s](%s)**.", c.SiteHost(), c.SiteURL("/dashboard"))).
		AddField("Music", "`/play` `/search` `/playlist` `/queue` `/now` `/lyrics` `/pause` `/resume` `/seek` `/skip` `/stop` `/shuffle` `/loop` `/remove`", false).
		AddField("Playlists", "`/list create` `/list add` `/list remove` `/list delete` `/list list`", false)
	if c.Radio != nil {
		embed.AddField("Radio", "`/radio play` `/radio now` `/radio stations`", false)
	}
	embed.AddField("Bot", "`/join` `/leave` `/help` `/invite`", false).
		SetFooter(c.SiteHost(), "")

	return e.CreateMessage(discord.MessageCreate{
		Embeds: []discord.Embed{embed.Build()},
		Components: []discord.ContainerComponent{
			discord.NewActionRow(
				discord.NewLinkButton("Dashboard", c.SiteURL("/dashboard")),
				discord.NewLinkButton("Status", c.SiteURL("/status")),
				discord.NewLinkButton("Website", c.SiteURL("")),
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
			SetDescription(fmt.Sprintf("Add Chilly to any server you manage, then run `/play` in a voice channel. Learn more at **[%s](%s)**.", c.SiteHost(), c.SiteURL(""))).
			Build()},
		Components: []discord.ContainerComponent{
			discord.NewActionRow(
				discord.NewLinkButton("Add to server", c.inviteURL()),
				discord.NewLinkButton("Website", c.SiteURL("")),
			),
		},
	})
}
