package sendico_test

import (
	"context"
	"testing"

	"github.com/robherley/sendibot/pkg/sendico"
	"github.com/stretchr/testify/assert"
)

func TestClientIntegration(t *testing.T) {
	ctx := context.Background()

	client, err := sendico.New(ctx)
	assert.NoError(t, err)

	t.Run("Search", func(t *testing.T) {
		_, err := client.Search(ctx, sendico.Mercari, sendico.SearchOptions{
			TermJP: "ゲームボーイsp",
		})
		assert.NoError(t, err)
	})

	t.Run("Search with min and max price", func(t *testing.T) {
		min := 2000
		max := 4000
		_, err := client.Search(ctx, sendico.Mercari, sendico.SearchOptions{
			TermJP:   "ゲームボーイsp",
			MinPrice: &min,
			MaxPrice: &max,
		})
		assert.NoError(t, err)
	})

	t.Run("Translate", func(t *testing.T) {
		_, err := client.Translate(ctx, "gameboy sp")
		assert.NoError(t, err)
	})
}
