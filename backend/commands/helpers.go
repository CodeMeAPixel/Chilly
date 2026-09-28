package commands

import (
	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

func ephemeral(e *handler.CommandEvent, content string) {
	if err := e.CreateMessage(discord.MessageCreate{
		Content: content,
		Flags:   discord.MessageFlagEphemeral,
	}); err != nil {
		musicbot.LogSendError(err, e.GuildID().String(), e.User().ID.String(), true)
	}
}

func reply(e *handler.CommandEvent, description string) {
	if err := e.CreateMessage(discord.MessageCreate{
		Embeds: []discord.Embed{{Description: description}},
	}); err != nil {
		musicbot.LogSendError(err, e.GuildID().String(), e.User().ID.String(), false)
		return
	}
	musicbot.AutoRemove(e)
}

func requireUserVoice(c *Commands, e *handler.CommandEvent) bool {
	if _, ok := c.UserVoiceChannel(*e.GuildID(), e.User().ID); !ok {
		ephemeral(e, "You need to be in a voice channel to use this command.")
		return false
	}
	return true
}

func (c *Commands) activePlayer(e *handler.CommandEvent) (*musicbot.Player, bool) {
	player, ok := c.PlayerManager.GetPlayer(*e.GuildID())
	if !ok || !player.IsPlaying() {
		ephemeral(e, "Player is not playing.")
		return nil, false
	}
	if !c.InSameVoice(*e.GuildID(), e.User().ID) {
		ephemeral(e, "You need to be in my voice channel to control the player.")
		return nil, false
	}
	return player, true
}
