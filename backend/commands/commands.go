package commands

import (
	"github.com/CodeMeAPixel/Chilly/musicbot"

	"github.com/disgoorg/disgo/discord"
)

var CommandCreates = []discord.ApplicationCommandCreate{
	playlist,
	help,
	invite,
}

func init() {
	CommandCreates = append(CommandCreates, botCommands()...)
	CommandCreates = append(CommandCreates, musicCommands()...)
}

func botCommands() []discord.ApplicationCommandCreate {
	commands := make([]discord.ApplicationCommandCreate, 0, len(bot.Options))
	for _, option := range bot.Options {
		subcommand, ok := option.(discord.ApplicationCommandOptionSubCommand)
		if !ok {
			continue
		}
		commands = append(commands, discord.SlashCommandCreate{
			Name:        subcommand.Name,
			Description: subcommand.Description,
			Options:     subcommand.Options,
		})
	}
	return commands
}

func musicCommands() []discord.ApplicationCommandCreate {
	commands := make([]discord.ApplicationCommandCreate, 0, len(music.Options))
	for _, option := range music.Options {
		subcommand, ok := option.(discord.ApplicationCommandOptionSubCommand)
		if !ok {
			continue
		}
		commands = append(commands, discord.SlashCommandCreate{
			Name:        subcommand.Name,
			Description: subcommand.Description,
			Options:     subcommand.Options,
		})
	}
	return commands
}

type Commands struct {
	*musicbot.Bot
}
