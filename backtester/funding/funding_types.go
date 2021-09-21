package funding

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// FundManager is the benevolent holder of all funding levels across all
// currencies used in the backtester
type FundManager struct {
	usingExchangeLevelFunding bool
	items                     []*Item
	reportMap                 map[Item]map[Item]ReportItem
}

// Report holds all funding data for result reporting
type Report struct {
	Items   []ReportItem
	ItemMap map[Item][]ReportItem
}

// ReportItem holds reporting fields
type ReportItem struct {
	Exchange     string
	Asset        asset.Item
	Currency     currency.Code
	InitialFunds decimal.Decimal
	TransferFee  decimal.Decimal
	FinalFunds   decimal.Decimal
	PairedWith   currency.Code
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
	BuildReportFromExchangeAssetPair(string, asset.Item, currency.Pair) error
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
	exchange     string
	asset        asset.Item
	currency     currency.Code
	initialFunds decimal.Decimal
	available    decimal.Decimal
	reserved     decimal.Decimal
	transferFee  decimal.Decimal
	pairedWith   *Item
}

// Pair holds two currencies that are associated with each other
type Pair struct {
	Base  *Item
	Quote *Item
}
