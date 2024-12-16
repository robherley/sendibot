package emoji

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type Store struct {
	store sync.Map
}

func (e *Store) Set(name, id string) {
	e.store.Store(name, id)
}

func (e *Store) ID(name string) string {
	v, ok := e.store.Load(name)
	if !ok {
		slog.Warn("emoji not found", "name", name)
		return ""
	}
	return v.(string)
}

func (e *Store) For(name string) string {
	return "<:" + name + ":" + e.ID(name) + ">"
}

func (e *Store) Has(name string) bool {
	_, ok := e.store.Load(name)
	return ok
}

func Fetch(session *discordgo.Session) (*Store, error) {
	appID := session.State.Application.ID
	body, err := session.Request("GET", discordgo.EndpointApplication(appID)+"/emojis", nil)
	if err != nil {
		return nil, err
	}

	response := struct {
		Items []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}{}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	var emoji *Store
	for _, item := range response.Items {
		emoji.Set(item.Name, item.ID)
	}

	slog.Info("fetched emojis", "count", len(response.Items))
	return emoji, nil
}
