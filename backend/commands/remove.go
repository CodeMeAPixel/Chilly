package commands

import (
	"fmt"

	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

func (c *Commands) RemoveQueueTrackAutocomplete(e *handler.AutocompleteEvent) error {
	player, ok := c.PlayerManager.GetPlayer(*e.GuildID())
	if !ok || !player.IsPlaying() {
		return e.AutocompleteResult(nil)
	}

	var (
		queue   = player.Queue()
		choices = make([]discord.AutocompleteChoice, 0)
		limit   = min(25, len(queue))
	)
	for i, track := range queue[:limit] {
		choices = append(choices, discord.AutocompleteChoiceInt{
			Name:  choiceName("%d. %s — %s", i+1, musicbot.TrackTitle(track), track.Info.Author),
			Value: i,
		})
	}
	return e.AutocompleteResult(choices)
}

func (c *Commands) RemoveQueueTrack(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	player, ok := c.activePlayer(e)
	if !ok {
		return nil
	}

	track, ok := player.RemoveFromQueue(data.Int("track"))
	if !ok {
		ephemeral(e, "Invalid index or no track to remove.")
		return nil
	}
	reply(e, fmt.Sprintf("Removed %s from the queue.", musicbot.TrackLink(track)))
	return nil
}
