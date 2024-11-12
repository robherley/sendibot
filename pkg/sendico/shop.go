package sendico

import (
	"encoding/json"
)

var ShopMap = map[string]Shop{
	YahooAuctions.Identifier(): YahooAuctions,
	Mercari.Identifier():       Mercari,
	Rakuma.Identifier():        Rakuma,
	Rakuten.Identifier():       Rakuten,
	Yahoo.Identifier():         Yahoo,
}

var Shops = []Shop{
	Mercari,
	Rakuma,
	Rakuten,
	YahooAuctions,
	Yahoo,
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

	if shop, ok := ShopMap[str]; ok {
		*s = shop
	} else {
		return NewInvalidShopError(str)
	}

	return nil
}

func ShopsFromBits(bits int) []Shop {
	shops := make([]Shop, 0, len(Shops))
	for _, shop := range Shops {
		if bits&int(shop) != 0 {
			shops = append(shops, shop)
		}
	}
	return shops
}
