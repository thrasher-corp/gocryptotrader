package exchange

import "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"

func Seed() error {
	allExchanges := []exchange.Details{
		{
			Name:      "Alphapoint",
		},
		{
			Name:      "Binance",
		},
		{
			Name:      "Bitfinex",
		},
		{
			Name:      "Bitflyer",
		},
		{
			Name:      "Bithumb",
		},
		{
			Name:      "Bitmex",
		},
		{
			Name:      "Bitstamp",
		},
		{
			Name:      "Bittrex",
		},
		{
			Name:      "BTC Markets",
		},
		{
			Name:     "BTSE",
		},
		{
			Name:      "Coinbase Pro",
		},
		{
			Name:      "Coinbene",
		},
		{
			Name:      "Coinut",
		},
		{
			Name:      "Exmo",
		},
		{
			Name:      "GateIO",
		},
		{
			Name:      "Gemini",
		},
		{
			Name:      "HitBTC",
		},
		{
			Name:      "Huobi",
		},
		{
			Name:      "itBit",
		},
		{
			Name:      "Kraken",
		},
		{
			Name:      "lakeBTC",
		},
		{
			Name:      "lBank",
		},
		{
			Name:      "Local Bitcoins",
		},
		{
			Name:      "alphapoint",
		},
		{
			Name:      "OKCoin",
		},
		{
			Name:      "OKEX",
		},
		{
			Name:      "Poloniex",
		},
		{
			Name:      "YoBit",
		},
		{
			Name:      "ZB",
		},
	}
	return exchange.InsertMany(allExchanges)
}