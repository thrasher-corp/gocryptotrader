package main

import (
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitflyer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bithumb"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitmex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitstamp"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bittrex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/btcmarkets"
	"github.com/thrasher-corp/gocryptotrader/exchanges/btse"
	"github.com/thrasher-corp/gocryptotrader/exchanges/coinbasepro"
	"github.com/thrasher-corp/gocryptotrader/exchanges/coinbene"
	"github.com/thrasher-corp/gocryptotrader/exchanges/coinut"
	"github.com/thrasher-corp/gocryptotrader/exchanges/exmo"
	"github.com/thrasher-corp/gocryptotrader/exchanges/gateio"
	"github.com/thrasher-corp/gocryptotrader/exchanges/gemini"
	"github.com/thrasher-corp/gocryptotrader/exchanges/hitbtc"
	"github.com/thrasher-corp/gocryptotrader/exchanges/huobi"
	"github.com/thrasher-corp/gocryptotrader/exchanges/itbit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kraken"
	"github.com/thrasher-corp/gocryptotrader/exchanges/lakebtc"
	"github.com/thrasher-corp/gocryptotrader/exchanges/lbank"
	"github.com/thrasher-corp/gocryptotrader/exchanges/localbitcoins"
	"github.com/thrasher-corp/gocryptotrader/exchanges/okcoin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/okex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/poloniex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/yobit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/zb"
)

const defaultSleepTime = time.Minute

// SupportedExchanges exchanges are supported exchanges
var SupportedExchanges = []exchange.IBotExchange{
	new(binance.Binance),
	new(bitfinex.Bitfinex),
	new(bitflyer.Bitflyer),
	new(bithumb.Bithumb),
	new(bitmex.Bitmex),
	new(bitstamp.Bitstamp),
	new(bittrex.Bittrex),
	new(btcmarkets.BTCMarkets),
	new(btse.BTSE),
	new(coinbene.Coinbene),
	new(coinut.COINUT),
	new(exmo.EXMO),
	new(coinbasepro.CoinbasePro),
	new(gateio.Gateio),
	new(gemini.Gemini),
	new(hitbtc.HitBTC),
	new(huobi.HUOBI),
	new(itbit.ItBit),
	new(kraken.Kraken),
	new(lakebtc.LakeBTC),
	new(lbank.Lbank),
	new(localbitcoins.LocalBitcoins),
	new(okcoin.OKCoin),
	new(okex.OKEX),
	new(poloniex.Poloniex),
	new(yobit.Yobit),
	new(zb.ZB),
}

const configFilename = "timestamp.json"

// TimestampConfiguration defines the indidual exchange configuration
type TimestampConfiguration struct {
	Enabled bool `json:"enabled"`
	Keys
	Report *Report `json:"-"`
}

// Keys defines the individual access keys for the exchange
type Keys struct {
	Key      string `json:"key"`
	Secret   string `json:"secret"`
	ClientID string `json:"clientID"`
	PEMKey   string `json:"pemKey"`
	OTP      string `json:"oneTimePassword"`
}

// Report is a reported error
type Report struct {
	Error error
}

var (
	verbose bool
	configs map[string]*TimestampConfiguration
)
