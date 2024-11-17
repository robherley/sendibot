package db

import (
	"context"
	"errors"
	"time"

	"github.com/robherley/sendibot/pkg/sendico"
)

var (
	ErrConstraintUnique = errors.New("failed unique constraint")
)

type DB interface {
	Close() error
	Migrate(context.Context) error
	CreateSubscription(*Subscription) error
	GetSubscription(id string) (*Subscription, error)
	UpdateSubscription(*Subscription) error
	GetUserSubscriptions(userID string) ([]TermSubscription, error)
	FindSubscriptionsToNotify(window time.Duration, limit int) ([]TermSubscription, error)
	SetNotified(subIDs ...string) error
	DeleteUserSubscriptions(userID string, ids ...string) error
	CreateTerm(*Term) error
	GetTerm(id string) (*Term, error)
	FilterBySeenItems(items []Item) ([]Item, error)
	TrackItems(items ...Item) error
	CleanupItems(window time.Duration) error
}

type Term struct {
	ID string
	EN string
	JP string
}

type Subscription struct {
	ID             string
	UserID         string
	TermID         string
	LastNotifiedAt time.Time
	ShopsBitField  int
	MinPrice       *int
	MaxPrice       *int
}

func (s *Subscription) AddShop(shop sendico.Shop) {
	s.ShopsBitField |= int(shop)
}

func (s *Subscription) Shops() []sendico.Shop {
	return sendico.ShopsFromBits(s.ShopsBitField)
}

type TermSubscription struct {
	Term         Term
	Subscription Subscription
}

type Item struct {
	ID             string
	Shop           sendico.Shop
	Code           string
	SubscriptionID string
}
