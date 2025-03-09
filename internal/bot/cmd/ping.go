package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/robherley/sendibot/internal/meta"
)

func NewPing() Handler {
	return &ping{}
}

type ping struct{}

func (cm *ping) Name() string {
	return "ping"
}

func (cmd *ping) Description() string {
	return "Pings the bot!"
}

func (cmd *ping) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if i.Type != discordgo.InteractionApplicationCommand {
		return nil
	}

	user := "unknown"
	if i.User != nil {
		user = i.User.String()
	} else if i.Member != nil {
		user = i.Member.User.String()
	}

	payload := map[string]interface{}{
		"user":       user,
		"guild_id":   i.GuildID,
		"dm":         i.GuildID == "",
		"channel_id": i.ChannelID,
		"version":    meta.Version,
	}

	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🏓 pong!\n```json\n%s\n```", bytes),
		},
	})
}
