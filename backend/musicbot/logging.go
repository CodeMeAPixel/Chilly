package musicbot

import (
	"log/slog"
)

func LogPlayerInteraction(button string, guildID string, userID string) {
	slog.Info("player interaction",
		slog.String("button", button),
		slog.String("guild_id", guildID),
		slog.String("user_id", userID),
	)
}

func LogCommand(command string, guildID string, userID string) {
	slog.Info("command executed",
		slog.String("command", command),
		slog.String("guild_id", guildID),
		slog.String("user_id", userID),
	)
}

func LogSendError(err error, guildID string, userID string, ephemeral bool) {
	slog.Error("failed to send message",
		slog.Any("error", err),
		slog.String("guild_id", guildID),
		slog.String("user_id", userID),
		slog.Bool("ephemeral", ephemeral),
	)
}

func LogUpdateError(err error, guildID string, userID string) {
	slog.Error("failed to update message",
		slog.Any("error", err),
		slog.String("guild_id", guildID),
		slog.String("user_id", userID),
	)
}

func LogCommandError(err error, command string, guildID string, userID string) {
	slog.Error("failed to run command",
		slog.Any("error", err),
		slog.String("command", command),
		slog.String("guild_id", guildID),
		slog.String("user_id", userID),
	)
}

func LogDeleteError(err error, guildID string, channelID string, messageID string) {
	slog.Error("failed to delete message",
		slog.Any("error", err),
		slog.String("guild_id", guildID),
		slog.String("channel_id", channelID),
		slog.String("message_id", messageID),
	)
}

func LogJoinDebug(stage string, guildID string, userID string, channelID string, err error) {
	slog.Error("voice join debug",
		slog.String("stage", stage),
		slog.String("guild_id", guildID),
		slog.String("user_id", userID),
		slog.String("channel_id", channelID),
		slog.Any("error", err),
	)
}
