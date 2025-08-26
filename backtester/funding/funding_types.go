package funding

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	// ErrFundsNotFound used when funds are requested but the funding is not found in the manager
	ErrFundsNotFound = errors.New("funding not found")
	// ErrAlreadyExists used when a matching item or pair is already in the funding manager
	ErrAlreadyExists = errors.New("funding already exists")
	// ErrUSDTrackingDisabled used when attempting to track USD values when disabled
	ErrUSDTrackingDisabled = errors.New("USD tracking disabled")

	errCannotAllocate             = errors.New("cannot allocate funds")
	errZeroAmountReceived         = errors.New("amount received less than or equal to zero")
	errNegativeAmountReceived     = errors.New("received negative decimal")
	errNotEnoughFunds             = errors.New("not enough funds")
	errCannotTransferToSameFunds  = errors.New("cannot send funds to self")
	errTransferMustBeSameCurrency = errors.New("cannot transfer to different currency")
	errCannotMatchTrackingToItem  = errors.New("cannot match tracking data to funding items")
	errNotFutures                 = errors.New("item linking collateral currencies must be a futures asset")
	errExchangeManagerRequired    = errors.New("exchange manager required")
)

// IFundingManager limits funding usage for portfolio event handling
type IFundingManager interface {
	Reset() error
	IsUsingExchangeLevelFunding() bool
	GetFundingForEvent(common.Event) (IFundingPair, error)
	Transfer(decimal.Decimal, *Item, *Item, bool) error
	GenerateReport() (*Report, error)
	AddUSDTrackingData(*kline.DataFromKline) error
	CreateSnapshot(time.Time) error
	USDTrackingDisabled() bool
	Liquidate(common.Event) error
	GetAllFunding() ([]BasicItem, error)
	UpdateCollateralForEvent(common.Event, bool) error
	UpdateAllCollateral(isLive, hasUpdateFunding bool) error
	UpdateFundingFromLiveData(hasUpdatedFunding bool) error
	HasFutures() bool
	HasExchangeBeenLiquidated(handler common.Event) bool
	RealisePNL(receivingExchange string, receivingAsset asset.Item, receivingCurrency currency.Code, realisedPNL decimal.Decimal) error
	SetFunding(string, asset.Item, *accounts.Balance, bool) error
}

// IFundingTransferer allows for funding amounts to be transferred
// implementation can be swapped for live transferring
type IFundingTransferer interface {
	IsUsingExchangeLevelFunding() bool
	Transfer(decimal.Decimal, *Item, *Item, bool) error
	GetFundingForEvent(common.Event) (IFundingPair, error)
	HasExchangeBeenLiquidated(handler common.Event) bool
}

// IFundingReader is a simple interface of
// IFundingManager for readonly access at portfolio
// manager
type IFundingReader interface {
	GetFundingForEvent(common.Event) (IFundingPair, error)
	GetAllFunding() []BasicItem
}

// IFundingPair allows conversion into various
// funding interfaces
type IFundingPair interface {
	FundReader() IFundReader
	FundReserver() IFundReserver
	FundReleaser() IFundReleaser
}

// IFundReader can read
// either collateral or pair details
type IFundReader interface {
	GetPairReader() (IPairReader, error)
	GetCollateralReader() (ICollateralReader, error)
}

// IFundReserver limits funding usage for portfolio event handling
type IFundReserver interface {
	IFundReader
	CanPlaceOrder(order.Side) bool
	Reserve(decimal.Decimal, order.Side) error
}

// IFundReleaser can read
// or release pair or collateral funds
type IFundReleaser interface {
	IFundReader
	PairReleaser() (IPairReleaser, error)
	CollateralReleaser() (ICollateralReleaser, error)
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
	ContractCurrency() currency.Code
	CollateralCurrency() currency.Code
	InitialFunds() decimal.Decimal
	AvailableFunds() decimal.Decimal
	CurrentHoldings() decimal.Decimal
}

// IPairReleaser limits funding usage for exchange event handling
type IPairReleaser interface {
	IPairReader
	IncreaseAvailable(decimal.Decimal, order.Side) error
	Release(decimal.Decimal, decimal.Decimal, order.Side) error
	Liquidate()
}

// ICollateralReleaser limits funding usage for exchange event handling
type ICollateralReleaser interface {
	ICollateralReader
	UpdateContracts(order.Side, decimal.Decimal) error
	TakeProfit(contracts, positionReturns decimal.Decimal) error
	ReleaseContracts(decimal.Decimal) error
	Liquidate()
}

// FundManager is the benevolent holder of all funding levels across all
// currencies used in the backtester
type FundManager struct {
	usingExchangeLevelFunding bool
	disableUSDTracking        bool
	items                     []*Item
	exchangeManager           *engine.ExchangeManager
	verbose                   bool
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
	snapshot          map[int64]ItemSnapshot
	isCollateral      bool
	isLiquidated      bool
	appendedViaAPI    bool
	collateralCandles map[currency.Code]kline.DataFromKline
}

// SpotPair holds two currencies that are associated with each other
type SpotPair struct {
	base  *Item
	quote *Item
}

// CollateralPair consists of a currency pair for a futures contract
// and associates it with an addition collateral pair to take funding from
type CollateralPair struct {
	currentDirection *order.Side
	contract         *Item
	collateral       *Item
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

// Report holds all funding data for result reporting
type Report struct {
	DisableUSDTracking        bool
	UsingExchangeLevelFunding bool
	Items                     []ReportItem
	USDTotalsOverTime         []ItemSnapshot
	InitialFunds              decimal.Decimal
	FinalFunds                decimal.Decimal
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
	IsCollateral         bool
	AppendedViaAPI       bool
	PairedWith           currency.Code
}

// ItemSnapshot holds USD values to allow for tracking
// across backtesting results
type ItemSnapshot struct {
	Time          time.Time
	Available     decimal.Decimal
	USDClosePrice decimal.Decimal
	USDValue      decimal.Decimal
	Breakdown     []CurrencyContribution
}

// CurrencyContribution helps breakdown how a USD value
// determines its number
type CurrencyContribution struct {
	Currency        currency.Code
	USDContribution decimal.Decimal
}
