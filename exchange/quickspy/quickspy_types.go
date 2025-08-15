package quickspy

import (
	"context"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/alert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

var bookNames = []string{"book", "depth", "ob", "orderbook"}

type CredKey struct {
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
	CredKey *CredKey
	// Focuses is a map of focus types to focus options
	Focuses map[FocusType]FocusData
	// shutUP is a channel for shutdown signaling
	shutUP chan any
	// dataHandlerChannel is used for receiving data from websockets
	dataHandlerChannel chan any
	// RWMutex is used for concurrent read/write operations
	RWMutex *sync.RWMutex
	// wg is used for synchronizing goroutines
	wg sync.WaitGroup
	// alert is used for notifications
	alert alert.Notice
	// Data contains all the market data
	Data Data
}

type KlineChartData struct {
	ChartName   string
	ChartColour string
	CandleData  []kline.Candle
	VolumeData  []kline.Candle
}

type OrderBookEntry struct {
	Price            float64
	Amount           float64
	OrderAmount      int64
	Total            float64
	ContractDecimals float64
}

type Data struct {
	Key *CredKey
	//  contract stuff
	UnderlyingBase         *currency.Item
	UnderlyingQuote        *currency.Item
	ContractExpirationTime time.Time
	ContractDecimals       float64
	ContractType           futures.ContractType
	//open interest
	OpenInterest float64
	// ticker stuff
	LastPrice   float64
	MarkPrice   float64
	IndexPrice  float64
	QuoteVolume float64
	Volume      float64
	// ob stuff
	OB            *orderbook.Depth
	AskLiquidity  float64
	AskValue      float64
	BidLiquidity  float64
	BidValue      float64
	Spread        float64
	SpreadPercent float64
	// fr stuff
	FundingRate            float64
	NextFundingRateTime    time.Time
	CurrentFundingRateTime time.Time
	EstimatedFundingRate   float64
	// trade stuff
	LastTradePrice float64
	LastTradeSize  float64
	// account stuff
	Holdings []account.Holdings
	// orders stuff
	Orders []order.Detail
	// kline stuff
	Klines []kline.Candle
	Bids   orderbook.Levels
	Asks   orderbook.Levels
	// order execution limits
	ExecutionLimits order.MinMaxLevel
	// url stuff
	Url string
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
	Bids                   []OrderBookEntry      `json:"bids,omitempty"`
	Asks                   []OrderBookEntry      `json:"asks,omitempty"`
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
