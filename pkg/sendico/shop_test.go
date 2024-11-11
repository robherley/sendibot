package sendico_test

import (
	"encoding/json"
	"testing"

	"github.com/robherley/sendibot/pkg/sendico"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalJSON(t *testing.T) {
	tc := []struct {
		data []byte
		want sendico.Shop
		err  error
	}{
		{
			data: []byte(`"ayahoo"`),
			want: sendico.YahooAuctions,
		},
		{
			data: []byte(`"mercari"`),
			want: sendico.Mercari,
		},
		{
			data: []byte(`"rakuma"`),
			want: sendico.Rakuma,
		},
		{
			data: []byte(`"rakuten"`),
			want: sendico.Rakuten,
		},
		{
			data: []byte(`"yahoo"`),
			want: sendico.Yahoo,
		},
		{
			data: []byte(`"garbage"`),
			err:  sendico.ErrInvalidShop,
		},
		{
			data: []byte(`null`),
			err:  sendico.ErrInvalidShop,
		},
	}

	for _, c := range tc {
		var s sendico.Shop

		err := json.Unmarshal(c.data, &s)
		if c.err != nil {
			assert.ErrorIs(t, err, c.err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, c.want, s)
		}
	}
}
