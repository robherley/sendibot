package db

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"

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
	const query = `INSERT INTO subscriptions (id, user_id, term_id, last_notified_at, shops) VALUES (?, ?, ?, ?, ?)`
	subscription.ID = newID()

	_, err := s.DB.Exec(query,
		subscription.ID,
		subscription.UserID,
		subscription.TermID,
		subscription.LastNotifiedAt,
		subscription.shops,
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
			return ErrConstraintUnique
		}
	}

	return nil
}

func (s *SQLite) GetUserSubscriptions(userID string) ([]TermSubscription, error) {
	const query = `
		SELECT t.id, t.en, t.jp, s.id, s.last_notified_at, s.shops
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
		if err := rows.Scan(&term.ID, &term.EN, &term.JP, &subscription.ID, &subscription.LastNotifiedAt, &subscription.shops); err != nil {
			return nil, err
		}

		subscriptions = append(subscriptions, TermSubscription{
			Term:         term,
			Subscription: subscription,
		})
	}

	return subscriptions, nil
}

func (s *SQLite) DeleteUserSubscriptions(userID string, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}

	query := `
	DELETE FROM
		subscriptions
	WHERE
		id IN (%s) AND user_id = ?`

	query = fmt.Sprintf(query, strings.Repeat("?,", len(ids)-1)+"?")

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	args = append(args, userID)

	_, err := s.DB.Exec(query, args...)
	if err != nil {
		return err
	}

	return nil
}
