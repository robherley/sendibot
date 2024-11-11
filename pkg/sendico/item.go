package sendico

import (
	"encoding/json"
	"fmt"
	"time"
)

type Auction struct {
	BuyOutPriceYen *int      `json:"buy_out_price"`
	BuyOutPriceUSD *int      `json:"buy_out_converted_price"`
	EndTime        time.Time `json:"end_time"`
	Bids           int       `json:"bids"`
}

func (a *Auction) Ends() time.Duration {
	return time.Until(a.EndTime)
}

func (a *Auction) IsEnded() bool {
	return a.Ends() <= 0
}

type Item struct {
	*Auction
	Shop     Shop         `json:"shop"`
	Code     string       `json:"code"`
	Name     string       `json:"name"`
	Category *json.Number `json:"category"`
	URL      string       `json:"url"`
	Image    string       `json:"img"`
	PriceYen int          `json:"price"`
	PriceUSD int          `json:"converted_price"`
	Labels   []string     `json:"labels"`
}

func (i *Item) SendicoLink() string {
	return fmt.Sprintf("%s/shop/%s/catalog/%s", DefaultBaseURL, i.Shop.Identifier(), i.Code)
}

func (i *Item) IsAuction() bool {
	return i.Auction != nil
}
