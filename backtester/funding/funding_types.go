package funding

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// FundManager is the benevolent holder of all funding levels across all
// currencies used in the backtester
type FundManager struct {
	usingExchangeLevelFunding bool
	disableUSDTracking        bool
	items                     []*Item
}

type ItemSnapshot struct {
	Time          time.Time
	Available     decimal.Decimal
	USDClosePrice decimal.Decimal
	USDValue      decimal.Decimal
}

// IFundingManager limits funding usage for portfolio event handling
type IFundingManager interface {
	Reset()
	IsUsingExchangeLevelFunding() bool
	GetFundingForEAC(string, asset.Item, currency.Code) (*Item, error)
	GetFundingForEvent(common.EventHandler) (*Pair, error)
	GetFundingForEAP(string, asset.Item, currency.Pair) (*Pair, error)
	Transfer(decimal.Decimal, *Item, *Item, bool) error
	GenerateReport() *Report
	AddUSDTrackingData(*kline.DataFromKline) error
	CreateSnapshot(time.Time)
	USDTrackingDisabled() bool
}

// IFundTransferer allows for funding amounts to be transferred
// implementation can be swapped for live transferring
type IFundTransferer interface {
	IsUsingExchangeLevelFunding() bool
	Transfer(decimal.Decimal, *Item, *Item, bool) error
	GetFundingForEAC(string, asset.Item, currency.Code) (*Item, error)
	GetFundingForEvent(common.EventHandler) (*Pair, error)
	GetFundingForEAP(string, asset.Item, currency.Pair) (*Pair, error)
}

// IPairReader is used to limit pair funding functions
// to readonly
type IPairReader interface {
	BaseInitialFunds() decimal.Decimal
	QuoteInitialFunds() decimal.Decimal
	BaseAvailable() decimal.Decimal
	QuoteAvailable() decimal.Decimal
}

// IPairReserver limits funding usage for portfolio event handling
type IPairReserver interface {
	IPairReader
	CanPlaceOrder(order.Side) bool
	Reserve(decimal.Decimal, order.Side) error
}

// IPairReleaser limits funding usage for exchange event handling
type IPairReleaser interface {
	IncreaseAvailable(decimal.Decimal, order.Side)
	Release(decimal.Decimal, decimal.Decimal, order.Side) error
}

// Item holds funding data per currency item
type Item struct {
	exchange           string
	asset              asset.Item
	currency           currency.Code
	initialFunds       decimal.Decimal
	available          decimal.Decimal
	reserved           decimal.Decimal
	transferFee        decimal.Decimal
	pairedWith         *Item
	usdTrackingCandles *kline.DataFromKline
	snapshot           map[time.Time]ItemSnapshot
}

// Pair holds two currencies that are associated with each other
type Pair struct {
	Base  *Item
	Quote *Item
}

// Report holds all funding data for result reporting
type Report struct {
	DisableUSDTracking bool
	USDInitialTotal    decimal.Decimal
	USDFinalTotal      decimal.Decimal
	Difference         decimal.Decimal
	Items              []ReportItem
	USDTotals          map[time.Time]ItemSnapshot
}

// ReportItem holds reporting fields
type ReportItem struct {
	Exchange             string
	Asset                asset.Item
	Currency             currency.Code
	TransferFee          decimal.Decimal
	InitialFunds         decimal.Decimal
	FinalFunds           decimal.Decimal
	USDInitialFunds      decimal.Decimal
	USDInitialCostForOne decimal.Decimal
	USDFinalFunds        decimal.Decimal
	USDFinalCostForOne   decimal.Decimal
	Snapshots            []ItemSnapshot

	USDPairCandle *kline.DataFromKline
	Difference    decimal.Decimal
	ShowInfinite  bool
	PairedWith    currency.Code
}
