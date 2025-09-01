package quickspy

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
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
)

// CredentialsKey is a struct that holds credentials and exchange/pair/asset info
type CredentialsKey struct {
	// Credentials is optional, if nil public data only
	Credentials       *account.Credentials  `json:"credentials"`
	ExchangeAssetPair key.ExchangeAssetPair `json:"exchangeAssetPair"`
}

// QuickSpy is a struct that holds data on a single exchange pair asset
// Its purpose is to continuously generate metadata on the market
// it is write-side oriented vs the comparer
type QuickSpy struct {
	// credContext is the context for credentials
	// also used for cancelling goroutines
	credContext context.Context
	// Exch is the exchange interface
	Exch exchange.IBotExchange
	// Key contains exchange, pair, and asset information
	Key *CredentialsKey
	// Focuses is a map of focus types to focus options
	// Don't access directly, use functions to handle locking
	Focuses *FocusStore
	// dataHandlerChannel is used for receiving data from websockets
	dataHandlerChannel chan any
	// m is used for concurrent read/write operations
	m *sync.RWMutex
	// wg is used for synchronizing goroutines
	wg sync.WaitGroup
	// alert is used for notifications
	alert alert.Notice
	// Data contains all the market data
	Data    *Data
	verbose bool
}

// Data holds the GCT types that QuickSpy gathers
type Data struct {
	Key             *CredentialsKey
	Contract        *futures.Contract
	Orderbook       *orderbook.Book
	Ticker          *ticker.Price
	Kline           []websocket.KlineData
	Account         *account.Holdings
	Orders          []order.Detail
	FundingRate     *fundingrate.LatestRateResponse
	Trades          []trade.Data
	ExecutionLimits *limits.MinMaxLevel
	URL             string
	OpenInterest    float64
}

// ExportedData is a struct that collates all the data QuickSpy has gathered
type ExportedData struct {
	Key                    key.ExchangeAssetPair `json:"CredentialsKey"`
	UnderlyingBase         *currency.Item        `json:"UnderlyingBase,omitzero"`
	UnderlyingQuote        *currency.Item        `json:"underlyingQuote,omitzero"`
	ContractExpirationTime time.Time             `json:"contractExpirationTime,omitzero"`
	ContractType           string                `json:"contractType,omitzero"`
	ContractDecimals       float64               `json:"contractDecimals,omitzero"`
	ContractSettlement     string                `json:"contractSettlement,omitzero"`
	HasValidCredentials    bool                  `json:"hasValidCredentials"`
	LastPrice              float64               `json:"lastPrice,omitzero"`
	IndexPrice             float64               `json:"indexPrice,omitzero"`
	MarkPrice              float64               `json:"markPrice,omitzero"`
	Volume                 float64               `json:"volume,omitzero"`
	Spread                 float64               `json:"spread,omitzero"`
	SpreadPercent          float64               `json:"spreadPercent,omitzero"`
	FundingRate            float64               `json:"fundingRate,omitzero"`
	EstimatedFundingRate   float64               `json:"estimatedFundingRate,omitzero"`
	LastTradePrice         float64               `json:"lastTradePrice,omitzero"`
	LastTradeSize          float64               `json:"lastTradeSize,omitzero"`
	Holdings               []account.SubAccount  `json:"holdings,omitzero"`
	Orders                 []order.Detail        `json:"orders,omitzero"`
	Bids                   orderbook.Levels      `json:"bids,omitzero"`
	Asks                   orderbook.Levels      `json:"asks,omitzero"`
	OpenInterest           float64               `json:"openInterest,omitzero"`
	NextFundingRateTime    time.Time             `json:"nextFundingRateTime,omitzero"`
	CurrentFundingRateTime time.Time             `json:"currentFundingRateTime,omitzero"`
	ExecutionLimits        limits.MinMaxLevel    `json:"executionLimits,omitzero"`
	URL                    string                `json:"url,omitzero"`
}
