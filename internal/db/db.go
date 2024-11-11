package db

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/robherley/sendibot/pkg/sendico"
)

var (
	ErrConstraintUnique = errors.New("failed unique constraint")
)

type DB interface {
	Close() error
	Migrate(context.Context) error
	CreateTerm(*Term) error
	GetTerm(id string) (*Term, error)
	CreateSubscription(*Subscription) error
	GetUserSubscriptions(userID string) ([]TermSubscription, error)
	DeleteUserSubscriptions(userID string, ids ...string) error
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
	shops          int
}

func (s *Subscription) AddShop(shop sendico.Shop) {
	s.shops |= int(shop)
}

func (s *Subscription) Shops() []sendico.Shop {
	shops := []sendico.Shop{}
	for _, shop := range sendico.Shops {
		if s.shops&int(shop) != 0 {
			shops = append(shops, shop)
		}
	}

	sort.Slice(shops, func(i, j int) bool {
		return shops[i].Name() < shops[j].Name()
	})

	return shops
}

type TermSubscription struct {
	Term
	Subscription
}
