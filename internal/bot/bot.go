package bot

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/robherley/sendibot/internal/bot/cmd"
	"github.com/robherley/sendibot/internal/db"
	"github.com/robherley/sendibot/pkg/sendico"
)

// MaxMessagesPerNotify is the maximum number of messages to send in a single notify.
// This number was based on the discord maximum of 10 embeds per message.
const MaxMessagesPerNotify = 10

type Bot struct {
	DB      db.DB
	Sendico *sendico.Client

	session  *discordgo.Session
	emojis   map[string]string
	handlers map[string]cmd.Handler
}

func New(token string, db db.DB, sendico *sendico.Client) (*Bot, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	session.UserAgent = "sendibot (https://github.com/robherley/sendibot)"

	b := &Bot{
		DB:      db,
		Sendico: sendico,
		session: session,
		emojis:  make(map[string]string),
	}

	b.handlers = buildHandlers(
		cmd.NewPing(),
		cmd.NewSubscribe(db, sendico, b.emojis),
		cmd.NewSubscriptions(db, b.emojis),
		cmd.NewUnsubscribe(db),
	)

	return b, nil
}

func (b *Bot) Start() error {
	b.addHandlers()
	if err := b.session.Open(); err != nil {
		return err
	}

	if err := b.fetchEmojis(); err != nil {
		return err
	}

	return nil
}

func (b *Bot) Close() error {
	return b.session.Close()
}

func (b *Bot) NotifyNewItems(termEN, userID string, items []sendico.Item) error {
	dm, err := b.session.UserChannelCreate(userID)
	if err != nil {
		return err
	}

	total := len(items)
	truncated := false
	if len(items) > MaxMessagesPerNotify {
		items = items[:MaxMessagesPerNotify]
		truncated = true
	}

	embeds := make([]*discordgo.MessageEmbed, 0, len(items))
	for _, item := range items {
		// TODOs:
		// - auction specific fields
		// - translate???
		embeds = append(embeds, &discordgo.MessageEmbed{
			Title: item.Name,
			Image: &discordgo.MessageEmbedImage{
				URL: item.Image,
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Price",
					Value:  fmt.Sprintf("¥%d ($%d)", item.PriceYen, item.PriceUSD),
					Inline: true,
				},
				{
					Name:   "Shop",
					Value:  fmt.Sprintf("<:%s:%s> %s", item.Shop.Identifier(), b.emojis[item.Shop.Identifier()], item.Shop.Name()),
					Inline: true,
				},
				{
					Name:   "Category",
					Value:  item.Category.String(),
					Inline: true,
				},
			},
			URL: item.SendicoLink(),
		})
	}

	msg, err := b.session.ChannelMessageSendComplex(dm.ID, &discordgo.MessageSend{
		Content: fmt.Sprintf("🔔 New items for %q!", termEN),
		Embeds:  embeds,
	})
	if err != nil {
		return err
	}

	if truncated {
		content := fmt.Sprintf("⚠️ BTW! I only sent %d out of %d items. This means there were a lot results from when I last checked. Try refining your search terms a bit more or listen to less shops!", MaxMessagesPerNotify, total)
		_, err = b.session.ChannelMessageSendReply(dm.ID, content, msg.Reference())
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Bot) addHandlers() error {
	b.session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		slog.Info("ready to go", "bot_user", r.User.String())
	})

	b.session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		log := LogWith(i, "interaction_type", i.Type.String())

		defer func() {
			if r := recover(); r != nil {
				log.Error("panic", "err", r)
			}
		}()

		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			handler, ok := b.handlers[i.ApplicationCommandData().Name]
			if !ok {
				log.Warn("no handler found")
				return
			}

			log.Info("invoking command")
			if err := handler.Handle(s, i); err != nil {
				log.Error("failed", "err", err)
			}
		case discordgo.InteractionMessageComponent:
			customID := i.MessageComponentData().CustomID
			log = log.With("custom_id", customID)

			cmd, _ := cmd.FromCustomID(customID)
			handler, ok := b.handlers[cmd]
			if !ok {
				log.Warn("no handler found")
				return
			}

			log.Info("invoking command")
			if err := handler.Handle(s, i); err != nil {
				log.Error("failed", "err", err)
			}
		default:
			log.Warn("unknown interaction type")
			spew.Dump(i)
		}
	})

	return nil
}

func (b *Bot) Unregister(guild string) error {
	if guild == "" {
		return nil
	}

	if guild == "global" {
		guild = ""
	}

	appID := b.session.State.User.ID
	existing, err := b.session.ApplicationCommands(appID, guild)
	if err != nil {
		return err
	}

	for _, cmd := range existing {
		log := slog.With("cmd", cmd.Name, "guild_id", guild)
		if err := b.session.ApplicationCommandDelete(appID, guild, cmd.ID); err != nil {
			log.Error("failed to unregister")
			return err
		}
		log.Info("unregistered")
	}

	return nil
}

func (b *Bot) Register(guild string) error {
	if guild == "" {
		return nil
	}

	if guild == "global" {
		guild = ""
	}

	appID := b.session.State.User.ID
	for _, h := range b.handlers {
		log := slog.With("cmd", h.Name(), "guild_id", guild)
		_, err := b.session.ApplicationCommandCreate(
			appID,
			guild,
			cmd.ToApplicationCommand(h),
		)
		if err != nil {
			log.Error("failed to register")
			return err
		}
		log.Info("registered")
	}

	return nil
}

func (b *Bot) fetchEmojis() (err error) {
	appID := b.session.State.Application.ID
	body, err := b.session.Request("GET", discordgo.EndpointApplication(appID)+"/emojis", nil)
	if err != nil {
		return
	}

	response := struct {
		Items []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}{}

	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	for _, item := range response.Items {
		b.emojis[item.Name] = item.ID
	}

	slog.Info("fetched emojis", "count", len(b.emojis))
	return nil
}

func buildHandlers(handlers ...cmd.Handler) map[string]cmd.Handler {
	m := make(map[string]cmd.Handler, len(handlers))
	for _, h := range handlers {
		m[h.Name()] = h
	}
	return m
}
