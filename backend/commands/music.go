package commands

import (
	"github.com/disgoorg/disgo/discord"
)

var searchTypeChoices = []discord.ApplicationCommandOptionChoiceString{
	{Name: "Song", Value: "track"},
	{Name: "Album", Value: "album"},
	{Name: "Artist", Value: "artist"},
	{Name: "Playlist", Value: "playlist"},
}

var loopModeChoices = []discord.ApplicationCommandOptionChoiceString{
	{
		Name:  "none",
		Value: "none",
	},
	{
		Name:  "track",
		Value: "track",
	},
	{
		Name:  "queue",
		Value: "queue",
	},
}

var music = discord.SlashCommandCreate{
	Name:        "music",
	Description: "music commands",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionSubCommand{
			Name:        "play",
			Description: "Play a song, album, artist or playlist from the library",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:         "query",
					Description:  "Song, album, artist or playlist name",
					Required:     true,
					Autocomplete: true,
				},
				discord.ApplicationCommandOptionString{
					Name:        "type",
					Description: "What to look for (defaults to a song)",
					Required:    false,
					Choices:     searchTypeChoices,
				},
				discord.ApplicationCommandOptionBool{
					Name:        "next",
					Description: "Play next instead of adding to the end of the queue",
					Required:    false,
				},
				discord.ApplicationCommandOptionBool{
					Name:        "loop",
					Description: "Loop the queue",
					Required:    false,
				},
				discord.ApplicationCommandOptionBool{
					Name:        "shuffle",
					Description: "Shuffle what gets added",
					Required:    false,
				},
			}},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "search",
			Description: "Search the library and pick the exact song",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:         "query",
					Description:  "Song, album, artist or playlist name",
					Required:     true,
					Autocomplete: true,
				},
				discord.ApplicationCommandOptionString{
					Name:        "type",
					Description: "What to look for (defaults to a song)",
					Required:    false,
					Choices:     searchTypeChoices,
				},
				discord.ApplicationCommandOptionBool{
					Name:        "next",
					Description: "Play next instead of adding to the end of the queue",
					Required:    false,
				},
			}},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "playlist",
			Description: "Add & play your saved playlists",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionInt{
					Name:         "playlist",
					Description:  "Playlist name",
					Required:     true,
					Autocomplete: true,
				},
				discord.ApplicationCommandOptionBool{
					Name:        "next",
					Description: "Play next instead of adding to the end of the queue",
					Required:    false,
				},
				discord.ApplicationCommandOptionBool{
					Name:        "loop",
					Description: "Enable loop for query",
					Required:    false,
				},
				discord.ApplicationCommandOptionBool{
					Name:        "shuffle",
					Description: "Enable shuffle for query",
					Required:    false,
				},
			}},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "queue",
			Description: "Display queue",
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "resume",
			Description: "Resume player",
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "pause",
			Description: "Pause player",
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "now",
			Description: "Current track",
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "lyrics",
			Description: "Show lyrics for the current song or any song",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "query",
					Description: "Song to look up, e.g. artist - title (defaults to what's playing)",
					Required:    false,
				},
			},
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "stop",
			Description: "Stop player",
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "skip",
			Description: "Skip current track",
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "shuffle",
			Description: "Shuffle queue",
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "loop",
			Description: "Loop track/queue",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "mode",
					Description: "Loop mode",
					Required:    true,
					Choices:     loopModeChoices,
				},
			},
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "seek",
			Description: "Seek player to a position",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "position",
					Description: "Postion to seek to (format: [HH:MM:SS] or [MM:SS])",
					Required:    true,
				},
			},
		},
		discord.ApplicationCommandOptionSubCommand{
			Name:        "remove",
			Description: "Remove track from queue",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionInt{
					Name:         "track",
					Description:  "Track to remove from queue",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
	},
}
