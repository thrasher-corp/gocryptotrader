package main

import (
	"fmt"
	"log"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/okex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func main() {
	var o okex.OKEX
	o.SetDefaults()

	o.API.Credentials.Key = "your_key"
	o.API.Credentials.Secret = "your_secret"
	o.API.Credentials.ClientID = "your_clientid"

	ord := &order.Submit{
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     50000,
		Amount:    0.1,
		AssetType: asset.Spot,
	}
	resp, err := o.SubmitOrder(ord)
	if err != nil {
		log.Printf("Unable to place order: %s", err)
	}
	fmt.Println(resp.OrderID)
}
