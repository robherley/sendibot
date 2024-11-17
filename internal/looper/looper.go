package looper

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robherley/sendibot/internal/bot"
	"github.com/robherley/sendibot/internal/db"
	"github.com/robherley/sendibot/pkg/sendico"
)

const (
	TickNotify  = 2 * time.Minute
	TickCleanup = 1 * time.Hour

	WindowNotify  = 5 * time.Minute
	WindowCleanup = 72 * time.Hour
)

type Looper struct {
	db      db.DB
	sendico *sendico.Client
	bot     *bot.Bot
}

func New(db db.DB, sendico *sendico.Client, bot *bot.Bot) *Looper {
	return &Looper{db, sendico, bot}
}

func (l *Looper) Notify(ctx context.Context) {
	ticker := time.NewTicker(TickNotify)
	defer ticker.Stop()

	log := slog.With("component", "looper.notify")
	log.Info("starting loop", "tick", TickNotify)

	for {
		select {
		case <-ctx.Done():
			log.Info("context done, stopping")
			return
		case <-ticker.C:
			// the following search/check/notify flow will probably not scale well. should be fine for low volume though
			termSubs, err := l.db.FindSubscriptionsToNotify(WindowNotify, 100)
			if err != nil {
				log.Error("failed to find terms to update", "err", err)
				continue
			}

			for _, termSub := range termSubs {
				// let's be nice to sendico
				time.Sleep(2 * time.Second)

				results, err := l.sendico.BulkSearch(ctx, termSub.Subscription.Shops(), sendico.SearchOptions{
					TermJP:   termSub.Term.JP,
					MinPrice: termSub.Subscription.MinPrice,
					MaxPrice: termSub.Subscription.MaxPrice,
				})
				if err != nil {
					log.Error("failed to bulk search", "err", err, "term_id", termSub.Term.ID)
					continue
				}

				itemMap := make(map[string]sendico.Item)
				items := make([]db.Item, 0, len(results))
				for _, item := range results {
					items = append(items, db.Item{
						Shop:           item.Shop,
						Code:           item.Code,
						SubscriptionID: termSub.Subscription.ID,
					})

					itemMap[fmt.Sprintf("%s:%s", item.Shop.Identifier(), item.Code)] = item
				}

				newItems, err := l.db.FilterBySeenItems(items)
				if err != nil {
					log.Error("failed to filter by seen items", "err", err)
					continue
				}

				if len(newItems) == 0 {
					log.Info("no new items found", "term_id", termSub.Term.ID)
					continue
				}

				log.Info("new items found", "term_id", termSub.Term.ID, "count", len(newItems))
				if err := l.db.TrackItems(newItems...); err != nil {
					log.Error("failed to track items", "err", err)
					continue
				}

				itemsToNotify := make([]sendico.Item, 0, len(newItems))
				for _, item := range newItems {
					key := fmt.Sprintf("%s:%s", item.Shop.Identifier(), item.Code)
					if item, found := itemMap[key]; found {
						itemsToNotify = append(itemsToNotify, item)
					}
				}

				if err := l.bot.NotifyNewItems(termSub.Term.EN, termSub.Subscription.UserID, itemsToNotify); err != nil {
					log.Error("failed to notify new items", "err", err, "term_id", termSub.Term.ID, "user_id", termSub.Subscription.UserID)
					continue
				}
			}

			subIDs := make([]string, 0, len(termSubs))
			for _, termSub := range termSubs {
				subIDs = append(subIDs, termSub.Subscription.ID)
			}

			if err := l.db.SetNotified(subIDs...); err != nil {
				log.Error("failed to set notified", "err", err)
				continue
			}
		}
	}
}

func (l *Looper) Cleanup(ctx context.Context) {
	ticker := time.NewTicker(TickCleanup)
	defer ticker.Stop()

	log := slog.With("component", "looper.cleanup")
	log.Info("starting loop", "tick", TickCleanup)

	for {
		select {
		case <-ctx.Done():
			log.Info("context done, stopping")
			return
		case <-ticker.C:
			if err := l.db.CleanupItems(WindowCleanup); err != nil {
				log.Error("failed to cleanup items", "err", err)
				continue
			}
		}
	}
}
