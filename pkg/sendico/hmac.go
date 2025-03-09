package sendico

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type HMACInput struct {
	Secret    string
	Path      string
	Payload   *orderedmap.OrderedMap[string, any]
	Timestamp int64
	Nonce     string
}

type HMACAttributes struct {
	Signature string
	Nonce     string
	Timestamp int64
}

func BuildHMAC(in HMACInput) (*HMACAttributes, error) {
	if in.Timestamp == 0 {
		in.Timestamp = time.Now().Unix()
	}

	if in.Nonce == "" {
		in.Nonce = uuid.New().String()
	}

	payload := orderedmap.New[string, any]()
	payload.Set("url", in.Path)
	payload.Set("body", in.Payload)
	payload.Set("nonce", in.Nonce)
	payload.Set("timestamp", in.Timestamp)

	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	hash := hmac.New(sha256.New, []byte(in.Secret))
	hash.Write(bytes)
	signature := hash.Sum(nil)

	return &HMACAttributes{
		Signature: fmt.Sprintf("%x", signature),
		Nonce:     in.Nonce,
		Timestamp: in.Timestamp,
	}, nil
}
