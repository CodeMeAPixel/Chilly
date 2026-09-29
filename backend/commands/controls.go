package commands

import (
	"fmt"

	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/lavalink"
)

func (c *Commands) Pause(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	player, ok := c.activePlayer(e)
	if !ok {
		return nil
	}
	if err := player.Pause(e.Ctx); err != nil {
		ephemeral(e, "Failed to pause player.")
		return err
	}
	reply(e, "⏸️ Paused player")
	return nil
}

func (c *Commands) Resume(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	player, ok := c.activePlayer(e)
	if !ok {
		return nil
	}
	if err := player.Resume(e.Ctx); err != nil {
		ephemeral(e, "Failed to resume player.")
		return err
	}
	reply(e, "▶️ Resumed player")
	return nil
}

func (c *Commands) Skip(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	player, ok := c.activePlayer(e)
	if !ok {
		return nil
	}
	skipped, err := player.Skip(e.Ctx)
	if err != nil {
		ephemeral(e, "Failed to skip track.")
		return err
	}
	reply(e, fmt.Sprintf("⏭️ Skipped %s", musicbot.TrackLink(skipped)))
	return nil
}

func (c *Commands) Stop(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	disableStay, blocked := c.stayGuard(e)
	if blocked {
		return nil
	}
	player, ok := c.activePlayer(e)
	if !ok {
		return nil
	}
	message := "⏹️ Stopped playing and cleared the queue."
	if disableStay {
		if _, err := c.DisableStay(e.Ctx, *e.GuildID()); err != nil {
			ephemeral(e, "Failed to turn off 24/7 radio.")
			return err
		}
		message += " 24/7 radio is now off."
	}
	if err := player.Stop(e.Ctx); err != nil {
		ephemeral(e, "Failed to stop player.")
		return err
	}
	reply(e, message)
	return nil
}

func (c *Commands) Shuffle(_ discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	player, ok := c.activePlayer(e)
	if !ok {
		return nil
	}
	if player.Shuffle() {
		player.SetShuffle(musicbot.ShuffleOff)
		reply(e, "🔀 Shuffle off")
	} else {
		player.SetShuffle(musicbot.ShuffleOn)
		reply(e, "🔀 Shuffle on")
	}
	return nil
}

func (c *Commands) Loop(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	player, ok := c.activePlayer(e)
	if !ok {
		return nil
	}
	mode, valid := musicbot.ParseLoopMode(data.String("mode"))
	if !valid {
		mode = musicbot.LoopQueue
	}
	player.SetLoop(mode)

	switch mode {
	case musicbot.LoopNone:
		reply(e, "⏭️ Disabled loop")
	case musicbot.LoopTrack:
		reply(e, "🔂 Enabled track loop")
	default:
		reply(e, "🔁 Enabled queue loop")
	}
	return nil
}

func (c *Commands) Seek(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	player, ok := c.activePlayer(e)
	if !ok {
		return nil
	}

	h, m, s, err := musicbot.ParseTime(data.String("position"))
	if err != nil {
		ephemeral(e, "Invalid position. Use `MM:SS` or `HH:MM:SS`.")
		return nil
	}
	if track, ok := player.Current(); ok && track.Info.IsStream {
		ephemeral(e, "This track can't be seeked.")
		return nil
	}

	newPosition := lavalink.Duration(s)*lavalink.Second + lavalink.Duration(m)*lavalink.Minute + lavalink.Duration(h)*lavalink.Hour
	if err := player.Seek(e.Ctx, newPosition); err != nil {
		ephemeral(e, "Failed to seek to position.")
		return err
	}
	reply(e, fmt.Sprintf("⏩ Player moved to `%s`", musicbot.FormatTime(newPosition)))
	return nil
}
