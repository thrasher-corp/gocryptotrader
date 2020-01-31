package supported

import (
	"fmt"
	"strings"
	"unicode"
)

// Supported exchanges to standardise strings
const (
	Binance       = "Binance"
	Bitfinex      = "Bitfinex"
	Biflyer       = "Biflyer"
	Bithumb       = "Bithumb"
	Bitmex        = "Bitmex"
	Bitstamp      = "Bitstamp"
	Bittrex       = "Bittrex"
	Btcmarkets    = "BTC Markets"
	Btse          = "BTSE"
	Coinbasepro   = "CoinbasePro"
	Coinbene      = "Coinbene"
	Coinut        = "Coinut"
	Exmo          = "Exmo"
	Gateio        = "GateIO"
	Gemini        = "Gemini"
	Hitbtc        = "HitBTC"
	Huobi         = "Huobi"
	Itbit         = "Itbit"
	Kraken        = "Kraken"
	Lakebtc       = "LakeBTC"
	Lbank         = "Lbank"
	Localbitcoins = "LocalBitcoins"
	Okcoin        = "OKCOIN"
	Okex          = "OKEX"
	Poloniex      = "Poloniex"
	Yobit         = "Yobit"
	Zb            = "Zb"
)

// Exchanges that are supported on this platform
var Exchanges = []string{
	Binance,
	Bitfinex,
	Biflyer,
	Bithumb,
	Bitmex,
	Bitstamp,
	Bittrex,
	Btcmarkets,
	Btse,
	Coinbasepro,
	Coinbene,
	Coinut,
	Exmo,
	Gateio,
	Gemini,
	Hitbtc,
	Huobi,
	Itbit,
	Kraken,
	Lakebtc,
	Lbank,
	Localbitcoins,
	Okcoin,
	Okex,
	Poloniex,
	Yobit,
	Zb,
}

// CheckExchange checks exchange to supported list
func CheckExchange(name string) (string, error) {
	for i := range Exchanges {
		exch := spaceDelete(strings.ToLower(Exchanges[i]))
		name = spaceDelete(strings.ToLower(name))
		if strings.Contains(exch, name) {
			return Exchanges[i], nil
		}
	}
	return "", fmt.Errorf("%s not found in supported exchange list", name)
}

func spaceDelete(str string) string {
	var b strings.Builder
	b.Grow(len(str))
	for i := range str {
		if !unicode.IsSpace(rune(str[i])) {
			b.WriteRune(rune(str[i]))
		}
	}
	return b.String()
}
