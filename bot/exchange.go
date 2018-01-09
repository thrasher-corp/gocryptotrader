package bot

import (
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/exchanges/anx"
	"github.com/thrasher-/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-/gocryptotrader/exchanges/bitstamp"
	"github.com/thrasher-/gocryptotrader/exchanges/bittrex"
	"github.com/thrasher-/gocryptotrader/exchanges/btcc"
	"github.com/thrasher-/gocryptotrader/exchanges/btcmarkets"
	"github.com/thrasher-/gocryptotrader/exchanges/coinut"
	"github.com/thrasher-/gocryptotrader/exchanges/gdax"
	"github.com/thrasher-/gocryptotrader/exchanges/gemini"
	"github.com/thrasher-/gocryptotrader/exchanges/huobi"
	"github.com/thrasher-/gocryptotrader/exchanges/itbit"
	"github.com/thrasher-/gocryptotrader/exchanges/kraken"
	"github.com/thrasher-/gocryptotrader/exchanges/lakebtc"
	"github.com/thrasher-/gocryptotrader/exchanges/liqui"
	"github.com/thrasher-/gocryptotrader/exchanges/localbitcoins"
	"github.com/thrasher-/gocryptotrader/exchanges/okcoin"
	"github.com/thrasher-/gocryptotrader/exchanges/poloniex"
	"github.com/thrasher-/gocryptotrader/exchanges/wex"
)

// ExchangeMain contains all the necessary exchange packages
type ExchangeMain struct {
	anx           anx.ANX
	btcc          btcc.BTCC
	bitstamp      bitstamp.Bitstamp
	bitfinex      bitfinex.Bitfinex
	bittrex       bittrex.Bittrex
	wex           wex.WEX
	btcmarkets    btcmarkets.BTCMarkets
	coinut        coinut.COINUT
	gdax          gdax.GDAX
	gemini        gemini.Gemini
	okcoinChina   okcoin.OKCoin
	okcoinIntl    okcoin.OKCoin
	itbit         itbit.ItBit
	lakebtc       lakebtc.LakeBTC
	liqui         liqui.Liqui
	localbitcoins localbitcoins.LocalBitcoins
	poloniex      poloniex.Poloniex
	huobi         huobi.HUOBI
	kraken        kraken.Kraken
}

// SetupBotExchanges setup exchange defaults
func (b *Bot) SetupBotExchanges() {
	for _, exch := range b.Config.Exchanges {
		for i := 0; i < len(b.Exchanges); i++ {
			if b.Exchanges[i] != nil {
				if b.Exchanges[i].GetName() == exch.Name {
					b.Exchanges[i].Setup(exch)
					if b.Exchanges[i].IsEnabled() {
						log.Printf("%s: Exchange support: %s (Authenticated API support: %s - Verbose mode: %s).\n",
							exch.Name, common.IsEnabled(exch.Enabled),
							common.IsEnabled(exch.AuthenticatedAPISupport),
							common.IsEnabled(exch.Verbose),
						)
						b.Exchanges[i].Start()
					} else {
						log.Printf(
							"%s: Exchange support: %s\n", exch.Name,
							common.IsEnabled(exch.Enabled),
						)
					}
				}
			}
		}
	}
}
