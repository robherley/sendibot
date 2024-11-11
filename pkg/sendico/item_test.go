package sendico_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/robherley/sendibot/pkg/sendico"
	"github.com/stretchr/testify/assert"
)

const (
	ayahoo = `{
	"shop": "ayahoo",
	"code": "e1160102473",
	"name": "\u9001\u6599\u8fbc\u3000\u30b2\u30fc\u30e0\u30dc\u30fc\u30a4\u3000\u30ab\u30e9\u30fc\u3000\u30a2\u30c9\u30d0\u30f3\u30b9\u3000SP 8\u53f0\u30bb\u30c3\u30c8\u3000\u30b8\u30e3\u30f3\u30af\u6271\u3044\u3000\u672c\u4f53\u306e\u307f",
	"category": "2084041581",
	"url": "https:\/\/page.auctions.yahoo.co.jp\/jp\/auction\/e1160102473",
	"img": "https:\/\/back.sendico.com\/proxy-images\/\/i\/auctions.c.yimg.jp\/images.auctions.yahoo.co.jp\/image\/dr000\/auc0511\/user\/07cca48c2428cc7b8cf65d8628519a602cc562431a3c822dc1f9f05d7aed9abf\/i-img783x1200-17309263127805sjnxec32.jpg?pri=l&w=300&h=300&up=0&nf_src=sy&nf_path=images\/auc\/pc\/top\/image\/1.0.3\/na_170x170.png&nf_st=200",
	"price": 29500,
	"buy_out_price": 29500,
	"end_time": "2024-11-14T01:54:36.000000Z",
	"bids": 0,
	"labels": [
		"free_shipping",
		"buy_now"
	],
	"favorite_id": null,
	"seconds_to_end": 593624,
	"converted_price": 194,
	"buy_out_converted_price": 194,
	"short_name": "\u9001\u6599\u8fbc\u3000\u30b2\u30fc\u30e0\u30dc\u30fc\u30a4\u3000\u30ab\u30e9\u30fc\u3000\u30a2\u30c9\u30d0\u30f3\u30b9..."
}`
	mercari = `{
		"shop": "mercari",
		"code": "m69480508468",
		"name": "\u52d5\u4f5c\u54c1\u3000\u4efb\u5929\u5802\u3000\u30b2\u30fc\u30e0\u30dc\u30fc\u30a4\u30ab\u30e9\u30fc\u3000\u30af\u30ea\u30a2\u672c\u4f53\u3000\u30c9\u30f3\u30ad\u30fc\u30b3\u30f3\u30b0",
		"category": 8908,
		"url": "https:\/\/jp.mercari.com\/item\/m69480508468",
		"img": "https:\/\/static.mercdn.net\/c!\/w=240\/thumb\/photos\/m69480508468_1.jpg?1730955439",
		"price": 7800,
		"favorite_id": null,
		"converted_price": 51,
		"short_name": "\u52d5\u4f5c\u54c1\u3000\u4efb\u5929\u5802\u3000\u30b2\u30fc\u30e0\u30dc\u30fc\u30a4\u30ab\u30e9\u30fc\u3000\u30af\u30ea...",
		"labels": [],
		"item_authentication": null
}`
	rakuma = `{
		"shop": "rakuma",
		"code": "5e8d557e7285362d481b72c34d57dcc6",
		"name": "\u30b2\u30fc\u30e0\u30dc\u30fc\u30a4\u30ab\u30e9\u30fc\u3000\u672c\u4f53",
		"category": 789,
		"url": "https:\/\/item.fril.jp\/5e8d557e7285362d481b72c34d57dcc6",
		"img": "https:\/\/img.fril.jp\/img\/720538380\/m\/2412622554.jpg?1729993881",
		"price": 7499,
		"favorite_id": null,
		"converted_price": 49,
		"short_name": "\u30b2\u30fc\u30e0\u30dc\u30fc\u30a4\u30ab\u30e9\u30fc\u3000\u672c\u4f53"
}`
	rakuten = `{
		"shop": "rakuten",
		"code": "centerwave:10000751",
		"name": "\u3010\u30bd\u30d5\u30c8\u30d7\u30ec\u30bc\u30f3\u30c8\u4f01\u753b\uff01\u3011\u30b2\u30fc\u30e0\u30dc\u30fc\u30a4 \u30ab\u30e9\u30fc \u672c\u4f53\u306e\u307f \u96fb\u6c60\u30ab\u30d0\u30fc\u4ed8\u304d 7\u8272...",
		"category": null,
		"url": "",
		"img": "https:\/\/thumbnail.image.rakuten.co.jp\/@0_mall\/centerwave\/cabinet\/08144867\/08164825\/08164837\/imgrc0120359763.jpg?_ex=256x256",
		"price": 10680,
		"favorite_id": null,
		"converted_price": 70,
		"short_name": "\u3010\u30bd\u30d5\u30c8\u30d7\u30ec\u30bc\u30f3\u30c8\u4f01\u753b\uff01\u3011\u30b2\u30fc\u30e0\u30dc\u30fc\u30a4..."
}`
	yahoo = `{
		"shop": "yahoo",
		"code": "cokotokyo_10433",
		"name": "\u30b2\u30fc\u30e0\u30dc\u30fc\u30a4 \u30ab\u30e9\u30fc \u672c\u4f53\u306e\u307f \u96fb\u6c60\u30ab\u30d0\u30fc\u4ed8\u304d 6\u8272\u9078\u3079\u308b\u30ab\u30e9\u30fc \u4efb\u5929\u5802 \u4e2d\u53e4",
		"category": 65458,
		"url": "https:\/\/store.shopping.yahoo.co.jp\/cokotokyo\/10433.html",
		"img": "https:\/\/back.sendico.com\/proxy-images\/yahoo-shopping\/\/i\/j\/cokotokyo_10433",
		"price": 10680,
		"favorite_id": null,
		"converted_price": 70,
		"short_name": "\u30b2\u30fc\u30e0\u30dc\u30fc\u30a4 \u30ab\u30e9\u30fc \u672c\u4f53\u306e\u307f \u96fb\u6c60\u30ab\u30d0\u30fc..."
}`
)

func ptr[T any](n T) *T {
	return &n
}

func ts(str string) time.Time {
	ts, err := time.Parse(time.RFC3339, str)
	if err != nil {
		panic(err)
	}
	return ts
}

func TestItemUnmarshalJSON(t *testing.T) {
	tc := []struct {
		name string
		data string
		want sendico.Item
	}{
		{
			name: "ayahoo",
			data: ayahoo,
			want: sendico.Item{
				Auction: &sendico.Auction{
					BuyOutPriceYen: ptr(29500),
					BuyOutPriceUSD: ptr(194),
					EndTime:        ts("2024-11-14T01:54:36.000000Z"),
				},
				Shop:     sendico.YahooAuctions,
				Code:     "e1160102473",
				Name:     "送料込　ゲームボーイ　カラー　アドバンス　SP 8台セット　ジャンク扱い　本体のみ",
				Category: ptr(json.Number("2084041581")),
				URL:      "https://page.auctions.yahoo.co.jp/jp/auction/e1160102473",
				Image:    "https://back.sendico.com/proxy-images//i/auctions.c.yimg.jp/images.auctions.yahoo.co.jp/image/dr000/auc0511/user/07cca48c2428cc7b8cf65d8628519a602cc562431a3c822dc1f9f05d7aed9abf/i-img783x1200-17309263127805sjnxec32.jpg?pri=l&w=300&h=300&up=0&nf_src=sy&nf_path=images/auc/pc/top/image/1.0.3/na_170x170.png&nf_st=200",
				PriceYen: 29500,
				PriceUSD: 194,
				Labels:   []string{"free_shipping", "buy_now"},
			},
		},
		{
			name: "mercari",
			data: mercari,
			want: sendico.Item{
				Auction:  nil,
				Shop:     sendico.Mercari,
				Code:     "m69480508468",
				Name:     "動作品　任天堂　ゲームボーイカラー　クリア本体　ドンキーコング",
				Category: ptr(json.Number("8908")),
				URL:      "https://jp.mercari.com/item/m69480508468",
				Image:    "https://static.mercdn.net/c!/w=240/thumb/photos/m69480508468_1.jpg?1730955439",
				PriceYen: 7800,
				PriceUSD: 51,
				Labels:   []string{},
			},
		},
		{
			name: "rakuma",
			data: rakuma,
			want: sendico.Item{
				Auction:  nil,
				Shop:     sendico.Rakuma,
				Code:     "5e8d557e7285362d481b72c34d57dcc6",
				Name:     "ゲームボーイカラー　本体",
				Category: ptr(json.Number("789")),
				URL:      "https://item.fril.jp/5e8d557e7285362d481b72c34d57dcc6",
				Image:    "https://img.fril.jp/img/720538380/m/2412622554.jpg?1729993881",
				PriceYen: 7499,
				PriceUSD: 49,
			},
		},
		{
			name: "rakuten",
			data: rakuten,
			want: sendico.Item{
				Auction:  nil,
				Shop:     sendico.Rakuten,
				Code:     "centerwave:10000751",
				Name:     "【ソフトプレゼント企画！】ゲームボーイ カラー 本体のみ 電池カバー付き 7色...",
				Category: nil,
				URL:      "",
				Image:    "https://thumbnail.image.rakuten.co.jp/@0_mall/centerwave/cabinet/08144867/08164825/08164837/imgrc0120359763.jpg?_ex=256x256",
				PriceYen: 10680,
				PriceUSD: 70,
			},
		},
		{
			name: "yahoo",
			data: yahoo,
			want: sendico.Item{
				Auction:  nil,
				Shop:     sendico.Yahoo,
				Code:     "cokotokyo_10433",
				Name:     "ゲームボーイ カラー 本体のみ 電池カバー付き 6色選べるカラー 任天堂 中古",
				Category: ptr(json.Number("65458")),
				URL:      "https://store.shopping.yahoo.co.jp/cokotokyo/10433.html",
				Image:    "https://back.sendico.com/proxy-images/yahoo-shopping//i/j/cokotokyo_10433",
				PriceYen: 10680,
				PriceUSD: 70,
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			var i sendico.Item
			err := json.Unmarshal([]byte(tt.data), &i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.want, i)
		})
	}
}

func TestItemSendicoLink(t *testing.T) {
	i := sendico.Item{
		Shop: sendico.YahooAuctions,
		Code: "e1160102473",
	}

	assert.Equal(t, "https://sendico.com/shop/ayahoo/catalog/e1160102473", i.SendicoLink())
}
