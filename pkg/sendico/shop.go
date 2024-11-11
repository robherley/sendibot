package sendico

import (
	"encoding/json"
)

var Shops = map[string]Shop{
	YahooAuctions.Identifier(): YahooAuctions,
	Mercari.Identifier():       Mercari,
	Rakuma.Identifier():        Rakuma,
	Rakuten.Identifier():       Rakuten,
	Yahoo.Identifier():         Yahoo,
}

type Shop int

const (
	YahooAuctions Shop = 0b00001
	Mercari       Shop = 0b00010
	Rakuma        Shop = 0b00100
	Rakuten       Shop = 0b01000
	Yahoo         Shop = 0b10000
)

func (s Shop) Identifier() string {
	switch s {
	case YahooAuctions:
		return "ayahoo"
	case Mercari:
		return "mercari"
	case Rakuma:
		return "rakuma"
	case Rakuten:
		return "rakuten"
	case Yahoo:
		return "yahoo"
	default:
		return ""
	}
}

func (s *Shop) Name() string {
	switch *s {
	case YahooAuctions:
		return "Yahoo Auctions"
	case Mercari:
		return "Mercari"
	case Rakuma:
		return "Rakuma"
	case Rakuten:
		return "Rakuten"
	case Yahoo:
		return "Yahoo Shopping"
	default:
		return ""
	}
}

func (s Shop) IsAuction() bool {
	return s == YahooAuctions
}

func (s Shop) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Identifier())
}

func (s *Shop) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	if shop, ok := Shops[str]; ok {
		*s = shop
	} else {
		return NewInvalidShopError(str)
	}

	return nil
}
