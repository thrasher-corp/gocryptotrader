package quickspy

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/alert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

var errKeyNotFound = errors.New("key not found")

type Key struct {
	Exchange    string               `json:"exchange"`
	Pair        currency.Pair        `json:"pair"`
	Asset       asset.Item           `json:"asset"`
	Credentials *account.Credentials `json:"credentials"`
}

// QuickSpy is a struct that holds data on a single exchange pair asset
// Its purpose is to continuously generate metadata on the market
// it is write-side oriented vs the comparer
type QuickSpy struct {
	// credContext is the context for credentials
	credContext context.Context
	// Exch is the exchange interface
	Exch exchange.IBotExchange
	// Key contains exchange, pair, and asset information
	Key *Key
	// Focuses is a map of focus types to focus options
	Focuses *FocusStore
	// shutdown is a channel for shutdown signaling
	shutdown chan any
	// dataHandlerChannel is used for receiving data from websockets
	dataHandlerChannel chan any
	// m is used for concurrent read/write operations
	m *sync.RWMutex
	// wg is used for synchronizing goroutines
	wg sync.WaitGroup
	// alert is used for notifications
	alert alert.Notice
	// Data contains all the market data
	Data *Data
}

type FocusStore struct {
	s map[FocusType]*FocusData
	m *sync.RWMutex
}

type Data struct {
	Key *Key
	//  contract stuff
	Contract        *futures.Contract
	Orderbook       *orderbook.Book
	Ticker          *ticker.Price
	Kline           []websocket.KlineData
	Account         []account.Holdings
	Orders          []order.Detail
	FundingRate     *fundingrate.Rate
	LastTrade       *trade.Data
	ExecutionLimits *order.MinMaxLevel
	URL             string
}

type ExportedData struct {
	Key                    key.ExchangePairAsset `json:"Key"`
	UnderlyingBase         *currency.Item        `json:"UnderlyingBase,omitempty"`
	UnderlyingQuote        *currency.Item        `json:"underlyingQuote,omitempty"`
	ContractExpirationTime time.Time             `json:"contractExpirationTime,omitempty"`
	ContractType           string                `json:"contractType,omitempty"`
	ContractDecimals       float64               `json:"contractDecimals,omitempty"`
	HasValidCredentials    bool                  `json:"hasValidCredentials"`
	LastPrice              float64               `json:"lastPrice,omitempty"`
	IndexPrice             float64               `json:"indexPrice,omitempty"`
	MarkPrice              float64               `json:"markPrice,omitempty"`
	Volume                 float64               `json:"volume,omitempty"`
	AskLiquidity           float64               `json:"askLiquidity,omitempty"`
	AskValue               float64               `json:"askValue,omitempty"`
	BidLiquidity           float64               `json:"bidLiquidity,omitempty"`
	BidValue               float64               `json:"bidValue,omitempty"`
	Spread                 float64               `json:"spread,omitempty"`
	SpreadPercent          float64               `json:"spreadPercent,omitempty"`
	FundingRate            float64               `json:"fundingRate,omitempty"`
	EstimatedFundingRate   float64               `json:"estimatedFundingRate,omitempty"`
	LastTradePrice         float64               `json:"lastTradePrice,omitempty"`
	LastTradeSize          float64               `json:"lastTradeSize,omitempty"`
	Holdings               []account.Holdings    `json:"holdings,omitempty"`
	Orders                 []order.Detail        `json:"orders,omitempty"`
	Bids                   orderbook.Levels      `json:"bids,omitempty"`
	Asks                   orderbook.Levels      `json:"asks,omitempty"`
	OpenInterest           float64               `json:"openInterest,omitempty"`
	NextFundingRateTime    time.Time             `json:"nextFundingRateTime,omitempty"`
	CurrentFundingRateTime time.Time             `json:"currentFundingRateTime,omitempty"`
	ExecutionLimits        order.MinMaxLevel     `json:"executionLimits,omitempty"`
	URL                    string                `json:"url,omitempty"`
	ContractDenomination   string                `json:"contractValueDenomination,omitempty"`
}

type FocusType int

type FocusData struct {
	Type                  FocusType
	Enabled               bool
	UseWebsocket          bool
	RESTPollTime          time.Duration
	m                     *sync.RWMutex
	IsOnceOff             bool
	hasBeenSuccessful     bool
	HasBeenSuccessfulChan chan any
	Stream                chan any
}

// FocusTypes are what quickspy uses to grant permission for it to grab data
const (
	UnsetFocusType FocusType = iota
	OpenInterestFocusType
	TickerFocusType
	OrderBookFocusType
	FundingRateFocusType
	TradesFocusType
	AccountHoldingsFocusType
	OrdersFocusType
	OrderPlacementFocusType
	KlineFocusType
	ContractFocusType
	OrderExecutionFocusType
	URLFocusType
	HistoricalContractKlineFocusType
)

func (f FocusType) String() string {
	switch f {
	case OpenInterestFocusType:
		return "OpenInterestFocusType"
	case TickerFocusType:
		return "TickerFocusType"
	case OrderBookFocusType:
		return "OrderBookFocusType"
	case FundingRateFocusType:
		return "FundingRateFocusType"
	case TradesFocusType:
		return "TradesFocusType"
	case AccountHoldingsFocusType:
		return "AccountHoldingsFocusType"
	case OrdersFocusType:
		return "OrdersFocusType"
	case OrderPlacementFocusType:
		return "OrderPlacementFocusType"
	case KlineFocusType:
		return "KlineFocusType"
	case ContractFocusType:
		return "ContractFocusType"
	case OrderExecutionFocusType:
		return "OrderExecutionFocusType"
	case URLFocusType:
		return "URLFocusType"
	default:
		return "Unset/Unknown FocusType"
	}
}
