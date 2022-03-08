package funding

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// FundManager is the benevolent holder of all funding levels across all
// currencies used in the backtester
type FundManager struct {
	usingExchangeLevelFunding bool
	disableUSDTracking        bool
	items                     []*Item
	exchangeManager           *engine.ExchangeManager
}

// IFundingManager limits funding usage for portfolio event handling
type IFundingManager interface {
	Reset()
	IsUsingExchangeLevelFunding() bool
	GetFundingForEAC(string, asset.Item, currency.Code) (*Item, error)
	GetFundingForEvent(common.EventHandler) (IFundingPair, error)
	GetFundingForEAP(string, asset.Item, currency.Pair) (IFundingPair, error)
	Transfer(decimal.Decimal, *Item, *Item, bool) error
	GenerateReport() *Report
	AddUSDTrackingData(*kline.DataFromKline) error
	CreateSnapshot(time.Time)
	USDTrackingDisabled() bool
	LiquidateByCollateral(currency.Code) error
	GetAllFunding() []BasicItem
	UpdateCollateral(common.EventHandler) error
}

// IFundingReader is a simple interface of
// IFundingManager for readonly access at portfolio
// manager
type IFundingReader interface {
	GetFundingForEvent(common.EventHandler) (IFundingPair, error)
	GetFundingForEAP(string, asset.Item, currency.Pair) (IFundingPair, error)
	GetAllFunding() []BasicItem
}

// IFundingPair allows conversion into various
// funding interfaces
type IFundingPair interface {
	FundReader() IFundReader
	FundReserver() IFundReserver
	FundReleaser() IFundReleaser
}

// IFundReader allows a connoisseur to read
// either collateral or pair details
type IFundReader interface {
	GetPairReader() (IPairReader, error)
	GetCollateralReader() (ICollateralReader, error)
}

// IFundTransferer allows for funding amounts to be transferred
// implementation can be swapped for live transferring
type IFundTransferer interface {
	IsUsingExchangeLevelFunding() bool
	Transfer(decimal.Decimal, *Item, *Item, bool) error
	GetFundingForEAC(string, asset.Item, currency.Code) (*Item, error)
	GetFundingForEvent(common.EventHandler) (IFundingPair, error)
	GetFundingForEAP(string, asset.Item, currency.Pair) (IFundingPair, error)
}

// IFundReserver limits funding usage for portfolio event handling
type IFundReserver interface {
	IFundReader
	CanPlaceOrder(order.Side) bool
	Reserve(decimal.Decimal, order.Side) error
}

// IFundReleaser allows a connoisseur to read
// or release pair or collateral funds
type IFundReleaser interface {
	IFundReader
	GetPairReleaser() (IPairReleaser, error)
	GetCollateralReleaser() (ICollateralReleaser, error)
}

// IPairReader is used to limit pair funding functions
// to readonly
type IPairReader interface {
	BaseInitialFunds() decimal.Decimal
	QuoteInitialFunds() decimal.Decimal
	BaseAvailable() decimal.Decimal
	QuoteAvailable() decimal.Decimal
}

// ICollateralReader is used to read data from
// collateral pairs
type ICollateralReader interface {
	UnderlyingAsset() currency.Code
	ContractCurrency() currency.Code
	CollateralCurrency() currency.Code
	InitialFunds() decimal.Decimal
	AvailableFunds() decimal.Decimal
	CurrentHoldings() decimal.Decimal
}

// IPairReleaser limits funding usage for exchange event handling
type IPairReleaser interface {
	IPairReader
	IncreaseAvailable(decimal.Decimal, order.Side)
	Release(decimal.Decimal, decimal.Decimal, order.Side) error
}

type ICollateralReleaser interface {
	ICollateralReader
	UpdateCollateral(decimal.Decimal) error
	UpdateContracts(order.Side, decimal.Decimal) error
	TakeProfit(contracts, positionReturns decimal.Decimal) error
	ReleaseContracts(decimal.Decimal) error
	Liquidate()
}

// Item holds funding data per currency item
type Item struct {
	exchange          string
	asset             asset.Item
	currency          currency.Code
	initialFunds      decimal.Decimal
	available         decimal.Decimal
	reserved          decimal.Decimal
	transferFee       decimal.Decimal
	pairedWith        *Item
	trackingCandles   *kline.DataFromKline
	snapshot          map[time.Time]ItemSnapshot
	isCollateral      bool
	collateralCandles map[currency.Code]kline.DataFromKline
}

// BasicItem is a representation of Item
type BasicItem struct {
	Exchange     string
	Asset        asset.Item
	Currency     currency.Code
	InitialFunds decimal.Decimal
	Available    decimal.Decimal
	Reserved     decimal.Decimal
	USDPrice     decimal.Decimal
}

// Pair holds two currencies that are associated with each other
type Pair struct {
	Base  *Item
	Quote *Item
}

// Collateral consists of a currency pair for a futures contract
// and associates it with an addition collateral pair to take funding from
type Collateral struct {
	currentDirection *order.Side
	Contract         *Item
	Collateral       *Item
}

// Report holds all funding data for result reporting
type Report struct {
	DisableUSDTracking        bool
	UsingExchangeLevelFunding bool
	Items                     []ReportItem
	USDTotalsOverTime         map[time.Time]ItemSnapshot
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
	USDPairCandle        *kline.DataFromKline
	Difference           decimal.Decimal
	ShowInfinite         bool
	PairedWith           currency.Code
	IsCollateral         bool
}

// ItemSnapshot holds USD values to allow for tracking
// across backtesting results
type ItemSnapshot struct {
	Time          time.Time
	Available     decimal.Decimal
	USDClosePrice decimal.Decimal
	USDValue      decimal.Decimal
}
