package exchange

import "strings"

// IsSupported returns whether or not a specific exchange is supported
func IsSupported(exchangeName string) bool {
	for x := range Exchanges {
		if strings.EqualFold(exchangeName, Exchanges[x]) {
			return true
		}
	}
	return false
}

// Exchanges stores a list of supported exchanges
var Exchanges = []string{
	"binance",
	"binanceus",
	"bitfinex",
	"bithumb",
	"bitflyer",
	"bitmex",
	"bitstamp",
	"btc markets",
	"btse",
	"bybit",
	"coinbasepro",
	"coinut",
	"deribit",
	"exmo",
	"gateio",
	"gemini",
	"hitbtc",
	"huobi",
	"kraken",
	"kucoin",
	"lbank",
	"okx",
	"poloniex",
	"yobit",
}
