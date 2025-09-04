package quickspy

import (
	"context"
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/alert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

var (
	errKeyNotFound                    = errors.New("key not found")
	errNoKey                          = errors.New("no key provided")
	errNoFocus                        = errors.New("no focuses provided")
	errValidationFailed               = errors.New("validation failed")
	errNoCredentials                  = errors.New("credentials required but none provided")
	errFocusDataTimeout               = errors.New("focus did not receive data in time")
	errNoRateLimits                   = errors.New("exchange has no rate limits set")
	errNoWebsocketSupportForFocusType = errors.New("quickspy does not support websocket for this focus type")
	errNoSubSwitchingToREST           = errors.New("no subscription found, switching to REST")
	errTimerNotSet                    = errors.New("timer not set")
	errNoDataYet                      = errors.New("no data received yet")
)

// CredentialsKey is a struct that holds credentials and exchange/pair/asset info
type CredentialsKey struct {
	// Credentials is optional, if nil public data only
	Credentials       *account.Credentials  `json:"credentials,omitempty"`
	ExchangeAssetPair key.ExchangeAssetPair `json:"exchangeAssetPair"`
}

// QuickSpy is a struct that holds data on a single exchange pair asset
// Its purpose is to continuously generate metadata on the market
// it is write-side oriented vs the comparer
type QuickSpy struct {
	// credContext is the context for credentials
	// also used for cancelling goroutines
	credContext context.Context
	// exch is the exchange interface
	exch exchange.IBotExchange
	// Key contains exchange, pair, and asset information
	key *CredentialsKey
	// focuses is a map of focus types to focus options
	// Don't access directly, use functions to handle locking
	focuses *FocusStore
	// dataHandlerChannel is used for receiving data from websockets
	dataHandlerChannel chan any
	// m is used for concurrent read/write operations
	m *sync.RWMutex
	// wg is used for synchronizing goroutines
	wg sync.WaitGroup
	// alert is used for notifications
	alert alert.Notice
	// Data contains all the market data
	data *Data
}

// Data holds the GCT types that QuickSpy gathers
type Data struct {
	Key             *CredentialsKey                 `json:"key"`
	Contract        *futures.Contract               `json:"contract,omitempty"`
	Orderbook       *orderbook.Book                 `json:"orderbook,omitempty"`
	Ticker          *ticker.Price                   `json:"ticker,omitempty"`
	Kline           []websocket.KlineData           `json:"kline,omitempty"`
	AccountBalance  []account.Balance               `json:"accountBalance,omitempty"`
	Orders          []order.Detail                  `json:"orders,omitempty"`
	FundingRate     *fundingrate.LatestRateResponse `json:"fundingRate,omitempty"`
	Trades          []trade.Data                    `json:"trades,omitempty"`
	ExecutionLimits *limits.MinMaxLevel             `json:"executionLimits,omitempty"`
	URL             string                          `json:"url,omitzero"`
	OpenInterest    float64                         `json:"openInterest,omitzero"`
}
