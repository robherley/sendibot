package db

import "github.com/google/uuid"

func newID() string {
	return uuid.Must(uuid.NewV7()).String()
}
