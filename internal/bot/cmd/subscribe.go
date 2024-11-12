package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/robherley/sendibot/internal/db"
	"github.com/robherley/sendibot/pkg/sendico"
)

func NewSubscribe(db db.DB, sendico *sendico.Client, emojis map[string]string) Handler {
	return &Subscribe{db, sendico, emojis, nil}
}

type Subscribe struct {
	db      db.DB
	sendico *sendico.Client
	emojis  map[string]string
	opts    []discordgo.SelectMenuOption
}

func (cmd *Subscribe) Name() string {
	return "subscribe"
}

func (cmd *Subscribe) Description() string {
	return "Subscribe to a search term and shops."
}

func (cmd *Subscribe) Options() []*discordgo.ApplicationCommandOption {
	termMinLength := 1
	termMaxLength := 100
	return []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "search",
			Description: "What items do you want to look for?",
			MinLength:   &termMinLength,
			MaxLength:   termMaxLength,
			Required:    true,
		},
	}
}

func (cmd *Subscribe) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		data := i.ApplicationCommandData()

		searchTermEN := data.Options[0].StringValue()
		searchTermJP, err := cmd.sendico.Translate(context.Background(), searchTermEN)
		if err != nil {
			return err
		}

		term := db.Term{
			EN: searchTermEN,
			JP: searchTermJP,
		}

		err = cmd.db.CreateTerm(&term)
		if err != nil {
			return err
		}

		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				CustomID: cmd.Name(),
				Content:  fmt.Sprintf("🔍 Will search for: %q (%s)", term.EN, term.JP),
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.SelectMenu{
								CustomID:    cmd.Name() + ":shops:" + term.ID,
								Placeholder: "🛒 What shops would you like to check?",
								Options:     cmd.options(),
								MaxValues:   len(cmd.options()),
							},
						},
					},
				},
			},
		})
	case discordgo.InteractionMessageComponent:
		_, args := FromCustomID(i.MessageComponentData().CustomID)
		if len(args) != 2 {
			return nil
		}

		userID := UserID(i)
		if userID == "" {
			return nil
		}

		term, err := cmd.db.GetTerm(args[1])
		if err != nil {
			return err
		}

		subscription := &db.Subscription{
			UserID: userID,
			TermID: term.ID,
		}

		for _, shop := range i.MessageComponentData().Values {
			found, ok := sendico.ShopMap[shop]
			if !ok {
				continue
			}
			subscription.AddShop(found)
		}

		if len(subscription.Shops()) == 0 {
			return nil
		}

		if err = cmd.db.CreateSubscription(subscription); err != nil {
			if errors.Is(err, db.ErrConstraintUnique) {
				return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("⛔ Already subscribed to shops for search: %q.\nSee subscriptions with `/subscriptions` and `/unsubscribe` if you wish to change your configured subscriptions.", term.EN),
					},
				})
			}

			return err
		}

		err = cmd.seedCurrentItems(term, subscription)
		if err != nil {
			slog.Error("failed to seed current items", "err", err)
			// this is best effort
		}

		shops := make([]string, 0, len(subscription.Shops()))
		for _, shop := range subscription.Shops() {
			shops = append(shops, fmt.Sprintf("<:%s:%s> %s", shop.Identifier(), cmd.emojis[shop.Identifier()], shop.Name()))
		}

		msg := fmt.Sprintf("🔔 Subscribed for term: %q (%s)\nWill check shops: %s", term.EN, term.JP, strings.Join(shops, ", "))

		dm, err := s.UserChannelCreate(userID)
		if err != nil {
			return err
		}

		_, err = s.ChannelMessageSend(dm.ID, msg)
		if err != nil {
			return err
		}

		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ Subscribed, <@%s>! You will receive a DM when new items are found.", userID),
			},
		})
	default:
		return nil
	}
}

func (cmd *Subscribe) seedCurrentItems(term *db.Term, sub *db.Subscription) error {
	results, err := cmd.sendico.BulkSearch(context.Background(), term.JP, sub.Shops()...)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		return nil
	}

	items := make([]db.Item, 0, len(results))
	for _, result := range results {
		items = append(items, db.Item{
			Shop:           result.Shop,
			Code:           result.Code,
			SubscriptionID: sub.ID,
		})
	}

	return cmd.db.TrackItems(items...)
}

func (cmd *Subscribe) options() []discordgo.SelectMenuOption {
	if cmd.opts != nil {
		return cmd.opts
	}

	cmd.opts = make([]discordgo.SelectMenuOption, 0, len(sendico.Shops))
	for _, shop := range sendico.Shops {
		cmd.opts = append(cmd.opts, discordgo.SelectMenuOption{
			Label: shop.Name(),
			Value: shop.Identifier(),
			Emoji: &discordgo.ComponentEmoji{
				ID: cmd.emojis[shop.Identifier()],
			},
		})
	}

	return cmd.opts
}
