package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/json"
)

var StayCommand = discord.SlashCommandCreate{
	Name:                     "247",
	Description:              "Keep a radio station playing in a voice channel around the clock",
	DefaultMemberPermissions: json.NewNullablePtr(discord.PermissionManageGuild),
	Contexts:                 []discord.InteractionContextType{discord.InteractionContextTypeGuild},
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionSubCommand{
			Name:        "on",
			Description: "Play a station 24/7 in your voice channel, even when nobody is listening",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:         "station",
					Description:  "Station to keep playing",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "off",
			Description: "Turn off 24/7 radio in this server",
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "status",
			Description: "Show the 24/7 radio setting for this server",
		},
	},
}

func (c *Commands) canManageStay(e *handler.CommandEvent) bool {
	return c.CanManageStay(*e.GuildID(), e.User().ID, e.Member())
}

func (c *Commands) StayOn(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.canManageStay(e) {
		ephemeral(e, "You need the **Manage Server** permission to set up 24/7 radio.")
		return nil
	}
	voiceChannelID, ok := c.UserVoiceChannel(*e.GuildID(), e.User().ID)
	if !ok {
		ephemeral(e, "Join the voice channel Chilly should stay in, then run this again.")
		return nil
	}
	if c.Radio == nil {
		ephemeral(e, "Radio is not enabled.")
		return nil
	}
	station, ok := c.Radio.Station(data.String("station"))
	if !ok {
		ephemeral(e, "Unknown station. Pick one from the list.")
		return nil
	}
	if !station.IsOnline {
		ephemeral(e, fmt.Sprintf("**%s** is offline right now. Pick another station or try again later.", station.Station.Name))
		return nil
	}

	if err := e.DeferCreateMessage(false); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(e.Ctx, 30*time.Second)
	defer cancel()

	setting := musicbot.StaySetting{
		GuildID:        *e.GuildID(),
		VoiceChannelID: voiceChannelID,
		TextChannelID:  e.Channel().ID(),
		Station:        station.Station.Shortcode,
		EnabledBy:      e.User().ID,
	}
	if _, err := c.EnableStay(ctx, setting); err != nil {
		var busy *musicbot.BusyError
		if errors.As(err, &busy) {
			updateReply(e, fmt.Sprintf("I'm playing in <#%s> right now. Stop that first or join that channel.", busy.ChannelID))
			return nil
		}
		updateReply(e, "Couldn't start 24/7 radio: "+c.voiceErrorMessage(err))
		return err
	}

	embed := RadioEmbed(station)
	embed.Title = "24/7 radio is on"
	embed.Description = fmt.Sprintf("I'll stay in <#%s> playing **%s** around the clock, even when nobody's listening, and come back after restarts.\n\nTurn it off with `/247 off`.\n\n%s",
		voiceChannelID, musicbot.EscapeMarkdown(station.Station.Name), embed.Description)
	if _, err := e.UpdateInteractionResponse(discord.MessageUpdate{Embeds: &[]discord.Embed{embed}}); err != nil {
		musicbot.LogUpdateError(err, e.GuildID().String(), e.User().ID.String())
	}
	return nil
}

func (c *Commands) StayOff(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	if !c.canManageStay(e) {
		ephemeral(e, "You need the **Manage Server** permission to change 24/7 radio.")
		return nil
	}
	existed, err := c.DisableStay(e.Ctx, *e.GuildID())
	if err != nil {
		ephemeral(e, "Couldn't turn off 24/7 radio. Try again in a moment.")
		return err
	}
	if !existed {
		ephemeral(e, "24/7 radio isn't on in this server.")
		return nil
	}

	message := "📻 24/7 radio is off. The current station keeps playing until you stop it or everyone leaves."
	if botCh, inVoice := c.BotVoiceChannel(*e.GuildID()); inVoice && c.ListenerCount(*e.GuildID(), botCh) == 0 {
		if err := c.Client.UpdateVoiceState(e.Ctx, *e.GuildID(), nil, false, true); err != nil {
			musicbot.LogCommandError(err, "/247/off", e.GuildID().String(), e.User().ID.String())
		}
		message = "📻 24/7 radio is off. Nobody was listening, so I left the voice channel."
	}
	reply(e, message)
	return nil
}

func (c *Commands) StayStatus(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	setting, ok := c.Stays.Get(*e.GuildID())
	if !ok {
		ephemeral(e, "24/7 radio is off. Turn it on with `/247 on` while you're in a voice channel.")
		return nil
	}
	name := setting.Station
	if c.Radio != nil {
		if np, ok := c.Radio.Station(setting.Station); ok {
			name = np.Station.Name
		}
	}
	builder := discord.NewEmbedBuilder().
		SetColor(musicbot.ColorPink).
		SetTitle("📻 24/7 radio is on").
		AddField("Station", musicbot.EscapeMarkdown(name), true).
		AddField("Channel", fmt.Sprintf("<#%s>", setting.VoiceChannelID), true).
		AddField("Turned on by", fmt.Sprintf("<@%s> <t:%d:R>", setting.EnabledBy, setting.EnabledAt.Unix()), false)
	if health := c.Stays.Health(*e.GuildID()); health.Failures > 0 {
		builder.AddField("Having trouble", fmt.Sprintf("%s\nRetrying <t:%d:R>.", musicbot.Trim(health.LastError, 300), health.RetryAt.Unix()), false)
	}
	return e.CreateMessage(discord.MessageCreate{
		Embeds: []discord.Embed{builder.SetFooterText("Turn it off with /247 off").Build()},
		Flags:  discord.MessageFlagEphemeral,
	})
}

func (c *Commands) stayGuard(e *handler.CommandEvent) (disable bool, blocked bool) {
	if !c.Stays.Enabled(*e.GuildID()) {
		return false, false
	}
	if c.canManageStay(e) {
		return true, false
	}
	ephemeral(e, "24/7 radio is on in this server. Someone with **Manage Server** can turn it off with `/247 off`.")
	return false, true
}
