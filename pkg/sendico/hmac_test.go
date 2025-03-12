package sendico_test

import (
	"testing"

	"github.com/robherley/sendibot/pkg/sendico"
	"github.com/stretchr/testify/assert"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestBuildHMAC(t *testing.T) {
	body := orderedmap.New[string, any]()
	body.Set("global", "1")
	body.Set("page", "1")
	body.Set("search", "セイコー")

	in := sendico.HMACInput{
		Secret:    "correct horse battery staple",
		Path:      "/api/mercari/items",
		Payload:   body,
		Timestamp: 1741408783,
		Nonce:     "66df7588-dedc-471d-9ffa-263ed1666cd0",
	}

	out, err := sendico.BuildHMAC(in)
	assert.NoError(t, err)
	assert.Equal(t, out.Signature, "2545fcd9e8b72d3e7be95affebcb080e444e2a2516329727c6bd9f3587833d4a")
}

func TestDecodeHMACKey(t *testing.T) {
	key := "pdwwkhzv pduvkdoo orfnv ehq dpholdexujk"
	decoded := sendico.DecodeHMACKey(key)
	assert.Equal(t, "ameliaburgh ben locks marshall matthews", decoded)
}
