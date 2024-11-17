package db

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"ariga.io/atlas/sql/migrate"
	aschema "ariga.io/atlas/sql/schema"
	asqlite "ariga.io/atlas/sql/sqlite"
	"github.com/mattn/go-sqlite3"
)

//go:embed schema.hcl
var schema []byte

type SQLite struct {
	*sql.DB
}

func NewSQLite(dsn string) (DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	return &SQLite{db}, nil
}

func (s *SQLite) Migrate(ctx context.Context) error {
	driver, err := asqlite.Open(s.DB)
	if err != nil {
		return err
	}

	want := &aschema.Schema{}
	if err := asqlite.EvalHCLBytes(schema, want, nil); err != nil {
		return err
	}

	got, err := driver.InspectSchema(ctx, "", nil)
	if err != nil {
		return err
	}

	changes, err := driver.SchemaDiff(got, want)
	if err != nil {
		return err
	}

	return driver.ApplyChanges(ctx, changes, []migrate.PlanOption{}...)
}

func (s *SQLite) CreateTerm(term *Term) error {
	const checkQuery = `SELECT id FROM terms WHERE en = ?`
	const insertQuery = `INSERT INTO terms (id, en, jp) VALUES (?, ?, ?)`

	var existingID string
	err := s.DB.QueryRow(checkQuery, term.EN).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if existingID != "" {
		term.ID = existingID
		return nil
	}

	term.ID = newID()
	_, err = s.DB.Exec(insertQuery, term.ID, term.EN, term.JP)
	if err != nil {
		return err
	}

	return nil
}

func (s *SQLite) GetTerm(id string) (*Term, error) {
	const query = `SELECT id, en, jp FROM terms WHERE id = ?`

	row := s.DB.QueryRow(query, id)

	term := &Term{}
	if err := row.Scan(&term.ID, &term.EN, &term.JP); err != nil {
		return nil, err
	}

	return term, nil
}

func (s *SQLite) CreateSubscription(subscription *Subscription) error {
	const query = `INSERT INTO subscriptions (
		id, user_id, term_id, last_notified_at, shops, min_price, max_price
	) VALUES (?, ?, ?, ?, ?, ?, ?)`
	subscription.ID = newID()

	_, err := s.DB.Exec(query,
		subscription.ID,
		subscription.UserID,
		subscription.TermID,
		time.Now().UTC(),
		subscription.ShopsBitField,
		subscription.MinPrice,
		subscription.MaxPrice,
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
			return ErrConstraintUnique
		}
		return err
	}

	return nil
}

func (s *SQLite) GetSubscription(id string) (*Subscription, error) {
	const query = `
		SELECT id, user_id, term_id, last_notified_at, shops, min_price, max_price
		FROM subscriptions
		WHERE id = ?
	`

	row := s.DB.QueryRow(query, id)

	subscription := &Subscription{}
	if err := row.Scan(
		&subscription.ID,
		&subscription.UserID,
		&subscription.TermID,
		&subscription.LastNotifiedAt,
		&subscription.ShopsBitField,
		&subscription.MinPrice,
		&subscription.MaxPrice,
	); err != nil {
		return nil, err
	}

	return subscription, nil
}

func (s *SQLite) UpdateSubscription(subscription *Subscription) error {
	const query = `
	UPDATE subscriptions
	SET last_notified_at = ?, shops = ?, min_price = ?, max_price = ?
	WHERE id = ?
	`

	_, err := s.DB.Exec(query,
		subscription.LastNotifiedAt,
		subscription.ShopsBitField,
		subscription.MinPrice,
		subscription.MaxPrice,
		subscription.ID,
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *SQLite) GetUserSubscriptions(userID string) ([]TermSubscription, error) {
	const query = `
		SELECT t.id, t.en, t.jp, s.id, s.user_id, s.term_id, s.last_notified_at, s.shops, s.min_price, s.max_price
		FROM subscriptions s
		JOIN terms t ON t.id = s.term_id
		WHERE s.user_id = ?
	`

	rows, err := s.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []TermSubscription
	for rows.Next() {
		var term Term
		var subscription Subscription
		if err := rows.Scan(
			&term.ID,
			&term.EN,
			&term.JP,
			&subscription.ID,
			&subscription.UserID,
			&subscription.TermID,
			&subscription.LastNotifiedAt,
			&subscription.ShopsBitField,
			&subscription.MinPrice,
			&subscription.MaxPrice,
		); err != nil {
			return nil, err
		}

		subscriptions = append(subscriptions, TermSubscription{
			Term:         term,
			Subscription: subscription,
		})
	}

	return subscriptions, nil
}

func (s *SQLite) FindSubscriptionsToNotify(window time.Duration, limit int) ([]TermSubscription, error) {
	const query = `
		SELECT t.id, t.en, t.jp, s.id, s.user_id, s.term_id, s.last_notified_at, s.shops, s.min_price, s.max_price
		FROM subscriptions s
		JOIN terms t ON t.id = s.term_id
		WHERE s.last_notified_at < ?
		LIMIT ?
	`

	rows, err := s.DB.Query(query, time.Now().UTC().Add(-window), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []TermSubscription
	for rows.Next() {
		var term Term
		var subscription Subscription
		if err := rows.Scan(
			&term.ID,
			&term.EN,
			&term.JP,
			&subscription.ID,
			&subscription.UserID,
			&subscription.TermID,
			&subscription.LastNotifiedAt,
			&subscription.ShopsBitField,
			&subscription.MinPrice,
			&subscription.MaxPrice,
		); err != nil {
			return nil, err
		}

		subscriptions = append(subscriptions, TermSubscription{
			Term:         term,
			Subscription: subscription,
		})
	}

	return subscriptions, nil
}

func (s *SQLite) SetNotified(subIDs ...string) error {
	if len(subIDs) == 0 {
		return nil
	}

	query := `
	UPDATE subscriptions
	SET last_notified_at = ?
	WHERE id IN (%s)`

	query = fmt.Sprintf(query, strings.Repeat("?,", len(subIDs)-1)+"?")
	args := []any{time.Now().UTC()}
	for _, id := range subIDs {
		args = append(args, id)
	}

	_, err := s.DB.Exec(query, args...)
	if err != nil {
		return err
	}

	return nil
}

func (s *SQLite) DeleteUserSubscriptions(userID string, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}

	subscriptionsDeleteQuery := `
	DELETE FROM
		subscriptions
	WHERE
		id IN (%s) AND user_id = ?`

	subscriptionsDeleteQuery = fmt.Sprintf(subscriptionsDeleteQuery, strings.Repeat("?,", len(ids)-1)+"?")

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	_, err = tx.Exec(subscriptionsDeleteQuery, append(args, userID)...)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	itemsDeleteQuery := `
	DELETE FROM
		items
	WHERE
		subscription_id IN (%s)`
	itemsDeleteQuery = fmt.Sprintf(itemsDeleteQuery, strings.Repeat("?,", len(ids)-1)+"?")

	_, err = tx.Exec(itemsDeleteQuery, args...)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (s *SQLite) TrackItems(items ...Item) error {
	const query = `
	INSERT INTO
		items (id, shop, code, subscription_id, created_at)
	VALUES (?, ?, ?, ?, ?)`

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}

	for _, item := range items {
		item.ID = newID()
		_, err = tx.Exec(query, item.ID, item.Shop, item.Code, item.SubscriptionID, time.Now().UTC())
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLite) FilterBySeenItems(items []Item) ([]Item, error) {
	if len(items) == 0 {
		return nil, nil
	}

	query := `
	SELECT
			subscription_id, shop, code
	FROM
			items
	WHERE
			(subscription_id, shop, code) IN (`

	var args []interface{}
	for i, item := range items {
		if i > 0 {
			query += ","
		}
		query += "(?, ?, ?)"
		args = append(args, item.SubscriptionID, item.Shop, item.Code)
	}
	query += ")"

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	foundItems := make(map[Item]struct{})
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.SubscriptionID, &item.Shop, &item.Code); err != nil {
			return nil, err
		}
		foundItems[item] = struct{}{}
	}

	var notFoundItems []Item
	for _, item := range items {
		if _, found := foundItems[item]; !found {
			notFoundItems = append(notFoundItems, item)
		}
	}

	return notFoundItems, nil
}

func (s *SQLite) CleanupItems(window time.Duration) error {
	_, err := s.DB.Exec("DELETE FROM items WHERE created_at < ?", time.Now().UTC().Add(-window))
	return err
}
