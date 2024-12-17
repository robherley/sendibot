package cmd

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/robherley/sendibot/internal/bot/emoji"
	"github.com/robherley/sendibot/internal/db"
)

func NewSubscriptions(db db.DB, emojis *emoji.Store) Handler {
	return &Subscriptions{db, emojis}
}

type Subscriptions struct {
	db     db.DB
	emojis *emoji.Store
}

func (cmd *Subscriptions) Name() string {
	return "subscriptions"
}

func (cmd *Subscriptions) Description() string {
	return "View active subscriptions."
}

func (cmd *Subscriptions) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if i.Type != discordgo.InteractionApplicationCommand {
		return nil
	}

	userID := UserID(i)
	if userID == "" {
		return nil
	}

	subs, err := cmd.db.GetUserSubscriptions(userID)
	if err != nil {
		return err
	}

	builder := strings.Builder{}
	builder.WriteString("You have ")
	builder.WriteString(strconv.Itoa(len(subs)))
	builder.WriteString(" subscription(s)")

	if len(subs) > 0 {
		builder.WriteString(":\n")
		for _, sub := range subs {
			builder.WriteString("- \"")
			builder.WriteString(sub.Term.EN)
			builder.WriteString("\" (")
			builder.WriteString(sub.Term.JP)
			builder.WriteString(") ")

			if sub.Subscription.MinPrice != nil || sub.Subscription.MaxPrice != nil {
				builder.WriteString("¥")
				if sub.Subscription.MinPrice != nil {
					builder.WriteString(strconv.Itoa(*sub.Subscription.MinPrice))
				} else {
					builder.WriteString("0")
				}
				builder.WriteString(" - ¥")
				if sub.Subscription.MaxPrice != nil {
					builder.WriteString(strconv.Itoa(*sub.Subscription.MaxPrice))
				} else {
					builder.WriteString("∞")
				}
				builder.WriteString(" ")
			}

			for i, shop := range sub.Subscription.Shops() {
				if cmd.emojis.Has(shop.Identifier()) {
					builder.WriteString(cmd.emojis.For(shop.Identifier()))
				} else {
					builder.WriteString(shop.Name())
					if i < len(sub.Subscription.Shops())-1 {
						builder.WriteString(", ")
					}
				}
			}
			builder.WriteString("\n")
		}
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			CustomID: cmd.Name(),
			Content:  builder.String(),
		},
	})
}
