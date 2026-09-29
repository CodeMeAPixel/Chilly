package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

var bot = discord.SlashCommandCreate{
	Name:        "bot",
	Description: "bot commands",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionSubCommand{
			Name:        "ping",
			Description: "[test] Ping command",
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "join",
			Description: "Joins voice chat channel",
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "leave",
			Description: "Leaves voice chat channel",
		},
	}}

func (c *Commands) Connect(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	channelID, ok := c.UserVoiceChannel(*e.GuildID(), e.User().ID)
	if !ok {
		ephemeral(e, "You need to be in a voice channel to use this command.")
		return nil
	}
	if botCh, inVoice := c.BotVoiceChannel(*e.GuildID()); inVoice && botCh != channelID {
		if p, exists := c.PlayerManager.GetPlayer(*e.GuildID()); exists && p.IsPlaying() {
			ephemeral(e, fmt.Sprintf("I'm already playing in <#%s>.", botCh))
			return nil
		}
	}

	if err := e.DeferCreateMessage(false); err != nil {
		return err
	}
	defer musicbot.AutoRemove(e)

	ctx, cancel := context.WithTimeout(e.Ctx, 15*time.Second)
	defer cancel()
	if err := c.EnsureVoice(ctx, *e.GuildID(), channelID); err != nil {
		musicbot.LogJoinDebug("ensure_voice_failed", e.GuildID().String(), e.User().ID.String(), channelID.String(), err)
		updateReply(e, c.voiceErrorMessage(err))
		return err
	}
	c.PlayerManager.GetOrCreatePlayer(*e.GuildID()).SetChannelID(e.Channel().ID())
	updateReply(e, fmt.Sprintf("Joined <#%s>.", channelID))
	return nil
}

func (c *Commands) Disconnect(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if _, inVoice := c.BotVoiceChannel(*e.GuildID()); !inVoice {
		ephemeral(e, "I'm not in a voice channel.")
		return nil
	}
	if p, exists := c.PlayerManager.GetPlayer(*e.GuildID()); exists && p.IsPlaying() && !c.InSameVoice(*e.GuildID(), e.User().ID) {
		ephemeral(e, "You need to be in my voice channel to disconnect me.")
		return nil
	}
	disableStay, blocked := c.stayGuard(e)
	if blocked {
		return nil
	}
	message := "Left voice channel."
	if disableStay {
		if _, err := c.DisableStay(e.Ctx, *e.GuildID()); err != nil {
			ephemeral(e, "Failed to turn off 24/7 radio.")
			return err
		}
		message = "Left voice channel. 24/7 radio is now off."
	}
	if err := c.Client.UpdateVoiceState(e.Ctx, *e.GuildID(), nil, false, true); err != nil {
		ephemeral(e, "Failed to leave voice channel.")
		return err
	}
	reply(e, message)
	return nil
}

func (c *Commands) Ping(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if sendErr := e.CreateMessage(discord.MessageCreate{
		Content: "Pong!",
	}); sendErr != nil {
		musicbot.LogSendError(sendErr, e.GuildID().String(), e.User().ID.String(), false)
	}
	return nil
}
