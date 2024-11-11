package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/robherley/sendibot/internal/db"
)

func NewUnsubscribe(db db.DB) Handler {
	return &Unsubscribe{db}
}

type Unsubscribe struct {
	db db.DB
}

func (cmd *Unsubscribe) Name() string {
	return "unsubscribe"
}

func (cmd *Unsubscribe) Description() string {
	return "Unsubscribe to Sendico updates"
}

func (cmd *Unsubscribe) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		userID := UserID(i)
		if userID == "" {
			return nil
		}

		subs, err := cmd.db.GetUserSubscriptions(userID)
		if err != nil {
			return err
		}

		if len(subs) == 0 {
			return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "ℹ️ You have no subscriptions to unsubscribe from.",
				},
			})
		}

		options := make([]discordgo.SelectMenuOption, 0, len(subs))
		for _, sub := range subs {
			options = append(options, discordgo.SelectMenuOption{
				Label: sub.Term.EN,
				Value: sub.Subscription.ID,
			})
		}

		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				CustomID: cmd.Name(),
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.SelectMenu{
								CustomID:    cmd.Name() + ":remove",
								Placeholder: "⏹️ What subscription(s) would you like to remove?",
								Options:     options,
								MaxValues:   len(options),
							},
						},
					},
				},
			},
		})
	case discordgo.InteractionMessageComponent:
		subIDs := i.MessageComponentData().Values
		userID := UserID(i)

		subscriptions, err := cmd.db.GetUserSubscriptions(userID)
		if err != nil {
			return err
		}

		if err := cmd.db.DeleteUserSubscriptions(userID, subIDs...); err != nil {
			return err
		}

		deleted := make([]db.Term, 0, len(subIDs))
		for _, subID := range subIDs {
			for _, sub := range subscriptions {
				if sub.Subscription.ID == subID {
					deleted = append(deleted, sub.Term)
				}
			}
		}

		builder := strings.Builder{}
		builder.WriteString("🔕 Unsubscribed from ")
		builder.WriteString(strconv.Itoa(len(deleted)))
		builder.WriteString(" terms(s):\n")

		for _, t := range deleted {
			builder.WriteString("- \"")
			builder.WriteString(t.EN)
			builder.WriteString("\"\n")
		}

		dm, err := s.UserChannelCreate(userID)
		if err != nil {
			return err
		}

		_, err = s.ChannelMessageSend(dm.ID, builder.String())
		if err != nil {
			return err
		}

		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ Unsubscribed from %d term(s)!", len(deleted)),
			},
		})
	default:
		return nil
	}
}
