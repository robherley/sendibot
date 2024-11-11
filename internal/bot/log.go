package bot

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

func LogWith(v any, more ...any) *slog.Logger {
	switch v := v.(type) {
	case *discordgo.MessageCreate:
		return slog.With(
			"guild_id", v.GuildID,
			"channel_id", v.ChannelID,
			"user", v.Author.Username,
		)
	case *discordgo.InteractionCreate:
		args := []any{
			"guild_id", v.GuildID,
			"channel_id", v.ChannelID,
		}
		if v.User != nil {
			args = append(args, "user", v.User.String())
		} else if v.Member != nil {
			args = append(args, "user", v.Member.User.String())
		}
		switch v.Type {
		case discordgo.InteractionApplicationCommand:
			args = append(args, "cmd", v.ApplicationCommandData().Name)
		case discordgo.InteractionModalSubmit:
			args = append(args, "custom_id", v.ModalSubmitData().CustomID)
		}
		return slog.With(append(args, more...)...)
	default:
		return slog.With(more...)
	}
}
