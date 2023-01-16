package engine

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binanceus"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitflyer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bithumb"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitmex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitstamp"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bittrex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/btcmarkets"
	"github.com/thrasher-corp/gocryptotrader/exchanges/btse"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bybit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/coinbasepro"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/okx"
	"github.com/thrasher-corp/gocryptotrader/exchanges/poloniex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
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
	ErrExchangeNameIsEmpty   = errors.New("exchange name is empty")

	errExchangeIsNil = errors.New("exchange is nil")
)

// CustomExchangeBuilder interface allows external applications to create
// custom/unsupported exchanges that satisfy the IBotExchange interface.
type CustomExchangeBuilder interface {
	NewExchangeByName(name string) (exchange.IBotExchange, error)
}

// ExchangeManager manages what exchanges are loaded
type ExchangeManager struct {
	mtx       sync.Mutex
	exchanges map[string]exchange.IBotExchange
	Builder   CustomExchangeBuilder
}

// NewExchangeManager creates a new exchange manager
func NewExchangeManager() *ExchangeManager {
	return &ExchangeManager{
		exchanges: make(map[string]exchange.IBotExchange),
	}
}

// Add adds an exchange
func (m *ExchangeManager) Add(exch exchange.IBotExchange) error {
	if m == nil {
		return fmt.Errorf("exchange manager: %w", ErrNilSubsystem)
	}
	if exch == nil {
		return errExchangeIsNil
	}
	m.mtx.Lock()
	defer m.mtx.Unlock()
	_, ok := m.exchanges[strings.ToLower(exch.GetName())]
	if ok {
		return errors.New("exchange already loaded")
	}
	m.exchanges[strings.ToLower(exch.GetName())] = exch
	return nil
}

// GetExchanges returns all stored exchanges
func (m *ExchangeManager) GetExchanges() ([]exchange.IBotExchange, error) {
	if m == nil {
		return nil, fmt.Errorf("exchange manager: %w", ErrNilSubsystem)
	}
	m.mtx.Lock()
	defer m.mtx.Unlock()
	exchs := make([]exchange.IBotExchange, 0, len(m.exchanges))
	for _, exch := range m.exchanges {
		exchs = append(exchs, exch)
	}
	return exchs, nil
}

// RemoveExchange removes an exchange from the manager
func (m *ExchangeManager) RemoveExchange(exchangeName string) error {
	if m == nil {
		return fmt.Errorf("exchange manager: %w", ErrNilSubsystem)
	}
	if m.Len() == 0 {
		return ErrNoExchangesLoaded
	}
	m.mtx.Lock()
	defer m.mtx.Unlock()
	exch, err := m.getExchangeByName(exchangeName)
	if err != nil {
		return err
	}
	return m.unload(exchangeName, exch)
}

// unload shutsdown and cleans up exchange and removes its reference from memory
// NOTE: This requires a lock.
func (m *ExchangeManager) unload(name string, exch exchange.IBotExchange) error {
	err := exch.GetBase().Websocket.Shutdown()
	if err != nil && !errors.Is(err, stream.ErrNotConnected) {
		return err
	}
	err = exch.GetBase().Requester.Shutdown()
	if err != nil {
		return err
	}
	delete(m.exchanges, strings.ToLower(name))
	log.Infof(log.ExchangeSys, "%s exchange unloaded successfully.\n", name)

	return nil
}

// GetExchangeByName returns an exchange by its name if it exists
func (m *ExchangeManager) GetExchangeByName(exchangeName string) (exchange.IBotExchange, error) {
	if m == nil {
		return nil, fmt.Errorf("exchange manager: %w", ErrNilSubsystem)
	}
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.getExchangeByName(exchangeName)
}

// getExchangeByName finds and returns an exchange by its name.
// NOTE: This requires a mutex lock.
func (m *ExchangeManager) getExchangeByName(exchName string) (exchange.IBotExchange, error) {
	if exchName == "" {
		return nil, fmt.Errorf("exchange manager: %w", ErrExchangeNameIsEmpty)
	}
	exch, ok := m.exchanges[strings.ToLower(exchName)]
	if !ok {
		return nil, fmt.Errorf("%s %w", exchName, ErrExchangeNotFound)
	}
	return exch, nil
}

// Len says how many exchanges are loaded
func (m *ExchangeManager) Len() int {
	if m == nil {
		return 0
	}
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return len(m.exchanges)
}

// NewExchangeByName helps create a new exchange to be loaded
func (m *ExchangeManager) NewExchangeByName(name string) (exchange.IBotExchange, error) {
	if m == nil {
		return nil, fmt.Errorf("exchange manager %w", ErrNilSubsystem)
	}
	nameLower := strings.ToLower(name)
	if exch, _ := m.GetExchangeByName(nameLower); exch != nil {
		return nil, fmt.Errorf("%s %w", name, ErrExchangeAlreadyLoaded)
	}
	var exch exchange.IBotExchange

	switch nameLower {
	case "binanceus":
		exch = new(binanceus.Binanceus)
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
	case "bybit":
		exch = new(bybit.Bybit)
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
	case "okx":
		exch = new(okx.Okx)
	case "poloniex":
		exch = new(poloniex.Poloniex)
	case "yobit":
		exch = new(yobit.Yobit)
	case "zb":
		exch = new(zb.ZB)
	default:
		if m.Builder != nil {
			return m.Builder.NewExchangeByName(nameLower)
		}
		return nil, fmt.Errorf("%s, %w", nameLower, ErrExchangeNotFound)
	}
	return exch, nil
}

// Shutdown shuts down all exchanges and unloads them
func (m *ExchangeManager) Shutdown() error {
	if m == nil {
		return fmt.Errorf("exchange manager %w", ErrNilSubsystem)
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()
	for _, exch := range m.exchanges {
		err := m.unload(exch.GetName(), exch)
		if err != nil {
			return err
		}
	}
	return nil
}
