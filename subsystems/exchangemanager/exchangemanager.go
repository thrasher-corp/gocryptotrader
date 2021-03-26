package exchangemanager

import (
	"errors"
	"strings"
	"sync"

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ftx"
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
	"github.com/thrasher-corp/gocryptotrader/log"
)

// vars related to exchange functions
var (
	ErrNoExchangesLoaded     = errors.New("no exchanges have been loaded")
	ErrExchangeNotFound      = errors.New("exchange not found")
	ErrExchangeAlreadyLoaded = errors.New("exchange already loaded")
	ErrExchangeFailedToLoad  = errors.New("exchange failed to load")
)

type ExchangeManager struct {
	m         sync.Mutex
	exchanges map[string]exchange.IBotExchange
}

func (e *ExchangeManager) Add(exch exchange.IBotExchange) {
	e.m.Lock()
	if e.exchanges == nil {
		e.exchanges = make(map[string]exchange.IBotExchange)
	}
	e.exchanges[strings.ToLower(exch.GetName())] = exch
	e.m.Unlock()
}

func (e *ExchangeManager) GetExchanges() []exchange.IBotExchange {
	if e.Len() == 0 {
		return nil
	}

	e.m.Lock()
	defer e.m.Unlock()
	var exchs []exchange.IBotExchange
	for x := range e.exchanges {
		exchs = append(exchs, e.exchanges[x])
	}
	return exchs
}

func (e *ExchangeManager) RemoveExchange(exchName string) error {
	if e.Len() == 0 {
		return ErrNoExchangesLoaded
	}
	exch := e.GetExchangeByName(exchName)
	if exch == nil {
		return ErrExchangeNotFound
	}
	e.m.Lock()
	defer e.m.Unlock()
	delete(e.exchanges, strings.ToLower(exchName))
	log.Infof(log.ExchangeSys, "%s exchange unloaded successfully.\n", exchName)
	return nil
}

func (e *ExchangeManager) GetExchangeByName(exchangeName string) exchange.IBotExchange {
	if e.Len() == 0 {
		return nil
	}
	e.m.Lock()
	defer e.m.Unlock()
	exch, ok := e.exchanges[strings.ToLower(exchangeName)]
	if !ok {
		return nil
	}
	return exch
}

func (e *ExchangeManager) Len() int {
	e.m.Lock()
	defer e.m.Unlock()
	return len(e.exchanges)
}

func (e *ExchangeManager) NewExchangeByName(name string) (exchange.IBotExchange, error) {
	nameLower := strings.ToLower(name)
	if e.GetExchangeByName(nameLower) != nil {
		return nil, ErrExchangeAlreadyLoaded
	}
	var exch exchange.IBotExchange

	switch nameLower {
	case "binance":
		exch = new(binance.Binance)
	case "bitfinex":
		exch = new(bitfinex.Bitfinex)
	case "bitflyer":
		exch = new(bitflyer.Bitflyer)
	case "bithumb":
		exch = new(bithumb.Bithumb)
	case "bitmex":
		exch = new(bitmex.Bitmex)
	case "bitstamp":
		exch = new(bitstamp.Bitstamp)
	case "bittrex":
		exch = new(bittrex.Bittrex)
	case "btc markets":
		exch = new(btcmarkets.BTCMarkets)
	case "btse":
		exch = new(btse.BTSE)
	case "coinbene":
		exch = new(coinbene.Coinbene)
	case "coinut":
		exch = new(coinut.COINUT)
	case "exmo":
		exch = new(exmo.EXMO)
	case "coinbasepro":
		exch = new(coinbasepro.CoinbasePro)
	case "ftx":
		exch = new(ftx.FTX)
	case "gateio":
		exch = new(gateio.Gateio)
	case "gemini":
		exch = new(gemini.Gemini)
	case "hitbtc":
		exch = new(hitbtc.HitBTC)
	case "huobi":
		exch = new(huobi.HUOBI)
	case "itbit":
		exch = new(itbit.ItBit)
	case "kraken":
		exch = new(kraken.Kraken)
	case "lakebtc":
		exch = new(lakebtc.LakeBTC)
	case "lbank":
		exch = new(lbank.Lbank)
	case "localbitcoins":
		exch = new(localbitcoins.LocalBitcoins)
	case "okcoin international":
		exch = new(okcoin.OKCoin)
	case "okex":
		exch = new(okex.OKEX)
	case "poloniex":
		exch = new(poloniex.Poloniex)
	case "yobit":
		exch = new(yobit.Yobit)
	case "zb":
		exch = new(zb.ZB)
	default:
		return nil, ErrExchangeNotFound
	}

	return exch, nil

}
