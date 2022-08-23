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
	"binanceus",
	"binance",
	"bitfinex",
	"bitflyer",
	"bithumb",
	"bitmex",
	"bitstamp",
	"bittrex",
	"btc markets",
	"btse",
	"bybit",
	"coinbasepro",
	"coinut",
	"exmo",
	"ftx",
	"gateio",
	"gemini",
	"hitbtc",
	"huobi",
	"itbit",
	"kraken",
	"lbank",
	"localbitcoins",
	"okcoin international",
	"okx",
	"poloniex",
	"yobit",
	"zb",
}
