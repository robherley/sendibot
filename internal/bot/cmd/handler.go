package cmd

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

type Handler interface {
	Name() string
	Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error
}

func ToApplicationCommand(h Handler) *discordgo.ApplicationCommand {
	cmd := &discordgo.ApplicationCommand{
		Name: h.Name(),
		Type: discordgo.ChatApplicationCommand,
	}

	if h, ok := h.(interface {
		Type() discordgo.ApplicationCommandType
	}); ok {
		cmd.Type = h.Type()
	}

	if h, ok := h.(interface {
		Description() string
	}); ok {
		cmd.Description = h.Description()
	}

	if h, ok := h.(interface {
		Options() []*discordgo.ApplicationCommandOption
	}); ok {
		cmd.Options = h.Options()
	}

	if cmd.Type != discordgo.ChatApplicationCommand {
		// these are only allowed for chat commands
		cmd.Description = ""
		cmd.Options = nil
	}

	return cmd
}

func FromCustomID(customID string) (string, []string) {
	parts := strings.Split(customID, ":")
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}

func UserID(i *discordgo.InteractionCreate) string {
	if i == nil {
		return ""
	}
	if i.User != nil {
		return i.User.ID
	} else if i.Member != nil {
		return i.Member.User.ID
	}
	return ""
}
