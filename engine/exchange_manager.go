package engine

import (
	"errors"
	"fmt"
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

// ExchangeManager manages what exchanges are loaded
type ExchangeManager struct {
	m         sync.Mutex
	exchanges map[string]exchange.IBotExchange
}

// SetupExchangeManager creates a new exchange manager
func SetupExchangeManager() *ExchangeManager {
	return &ExchangeManager{
		exchanges: make(map[string]exchange.IBotExchange),
	}
}

// Add adds or replaces an exchange
func (m *ExchangeManager) Add(exch exchange.IBotExchange) {
	if exch == nil {
		return
	}
	m.m.Lock()
	m.exchanges[strings.ToLower(exch.GetName())] = exch
	m.m.Unlock()
}

// GetExchanges returns all stored exchanges
func (m *ExchangeManager) GetExchanges() []exchange.IBotExchange {
	m.m.Lock()
	defer m.m.Unlock()
	var exchs []exchange.IBotExchange
	for _, x := range m.exchanges {
		exchs = append(exchs, x)
	}
	return exchs
}

// RemoveExchange removes an exchange from the manager
func (m *ExchangeManager) RemoveExchange(exchName string) error {
	if m.Len() == 0 {
		return ErrNoExchangesLoaded
	}
	exch := m.GetExchangeByName(exchName)
	if exch == nil {
		return ErrExchangeNotFound
	}
	m.m.Lock()
	defer m.m.Unlock()
	delete(m.exchanges, strings.ToLower(exchName))
	log.Infof(log.ExchangeSys, "%s exchange unloaded successfully.\n", exchName)
	return nil
}

// GetExchangeByName returns an exchange by its name if it exists
func (m *ExchangeManager) GetExchangeByName(exchangeName string) exchange.IBotExchange {
	if m == nil {
		return nil
	}
	m.m.Lock()
	defer m.m.Unlock()
	exch, ok := m.exchanges[strings.ToLower(exchangeName)]
	if !ok {
		return nil
	}
	return exch
}

// Len says how many exchanges are loaded
func (m *ExchangeManager) Len() int {
	m.m.Lock()
	defer m.m.Unlock()
	return len(m.exchanges)
}

// NewExchangeByName helps create a new exchange to be loaded
func (m *ExchangeManager) NewExchangeByName(name string) (exchange.IBotExchange, error) {
	if m == nil {
		return nil, fmt.Errorf("exchange manager %w", ErrNilSubsystem)
	}
	nameLower := strings.ToLower(name)
	if m.GetExchangeByName(nameLower) != nil {
		return nil, fmt.Errorf("%s %w", name, ErrExchangeAlreadyLoaded)
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
		return nil, fmt.Errorf("%s, %w", nameLower, ErrExchangeNotFound)
	}

	return exch, nil
}
