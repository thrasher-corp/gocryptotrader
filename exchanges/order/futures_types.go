package order

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
)

var (
	// ErrPositionClosed returned when attempting to amend a closed position
	ErrPositionClosed = errors.New("the position is closed")
	// ErrNilPNLCalculator is raised when pnl calculation is requested for
	// an exchange, but the fields are not set properly
	ErrNilPNLCalculator = errors.New("nil pnl calculator received")
	// ErrPositionLiquidated is raised when checking PNL status only for
	// it to be liquidated
	ErrPositionLiquidated = errors.New("position liquidated")
	// ErrNotFuturesAsset returned when futures data is requested on a non-futures asset
	ErrNotFuturesAsset = errors.New("asset type is not futures")
	// ErrUSDValueRequired returned when usd value unset
	ErrUSDValueRequired = errors.New("USD value required")
	// ErrOfflineCalculationSet is raised when collateral calculation is set to be offline, yet is attempted online
	ErrOfflineCalculationSet = errors.New("offline calculation set")
	// ErrPositionNotFound is raised when a position is not found
	ErrPositionNotFound = errors.New("position not found")
	// ErrNotPerpetualFuture is returned when a currency is not a perpetual future
	ErrNotPerpetualFuture = errors.New("not a perpetual future")
	// ErrNoPositionsFound returned when there is no positions returned
	ErrNoPositionsFound = errors.New("no positions found")
	// ErrGetFundingDataRequired is returned when requesting funding rate data without the prerequisite
	ErrGetFundingDataRequired = errors.New("getfundingdata is a prerequisite")

	errExchangeNameEmpty              = errors.New("exchange name empty")
	errExchangeNameMismatch           = errors.New("exchange name mismatch")
	errTimeUnset                      = errors.New("time unset")
	errMissingPNLCalculationFunctions = errors.New("futures tracker requires exchange PNL calculation functions")
	errOrderNotEqualToTracker         = errors.New("order does not match tracker data")
	errPositionDiscrepancy            = errors.New("there is a position considered open, but it is not the latest, please review")
	errAssetMismatch                  = errors.New("provided asset does not match")
	errEmptyUnderlying                = errors.New("underlying asset unset")
	errNilSetup                       = errors.New("nil setup received")
	errNilOrder                       = errors.New("nil order received")
	errNoPNLHistory                   = errors.New("no pnl history")
	errCannotCalculateUnrealisedPNL   = errors.New("cannot calculate unrealised PNL")
	errDoesntMatch                    = errors.New("doesn't match")
	errCannotTrackInvalidParams       = errors.New("parameters set incorrectly, cannot track")
)

// PNLCalculation is an interface to allow multiple
// ways of calculating PNL to be used for futures positions
type PNLCalculation interface {
	CalculatePNL(context.Context, *PNLCalculatorRequest) (*PNLResult, error)
	GetCurrencyForRealisedPNL(realisedAsset asset.Item, realisedPair currency.Pair) (currency.Code, asset.Item, error)
}

// TotalCollateralResponse holds all collateral
type TotalCollateralResponse struct {
	CollateralCurrency                          currency.Code
	TotalValueOfPositiveSpotBalances            decimal.Decimal
	CollateralContributedByPositiveSpotBalances decimal.Decimal
	UsedCollateral                              decimal.Decimal
	UsedBreakdown                               *UsedCollateralBreakdown
	AvailableCollateral                         decimal.Decimal
	AvailableMaintenanceCollateral              decimal.Decimal
	UnrealisedPNL                               decimal.Decimal
	BreakdownByCurrency                         []CollateralByCurrency
	BreakdownOfPositions                        []CollateralByPosition
}

// CollateralByPosition shows how much collateral is used
// from positions
type CollateralByPosition struct {
	PositionCurrency currency.Pair
	Size             decimal.Decimal
	OpenOrderSize    decimal.Decimal
	PositionSize     decimal.Decimal
	MarkPrice        decimal.Decimal
	RequiredMargin   decimal.Decimal
	CollateralUsed   decimal.Decimal
}

// CollateralByCurrency individual collateral contribution
// along with what the potentially scaled collateral
// currency it is represented as
// eg in Bybit ScaledCurrency is USDC
type CollateralByCurrency struct {
	Currency                    currency.Code
	SkipContribution            bool
	TotalFunds                  decimal.Decimal
	AvailableForUseAsCollateral decimal.Decimal
	CollateralContribution      decimal.Decimal
	AdditionalCollateralUsed    decimal.Decimal
	FairMarketValue             decimal.Decimal
	Weighting                   decimal.Decimal
	ScaledCurrency              currency.Code
	UnrealisedPNL               decimal.Decimal
	ScaledUsed                  decimal.Decimal
	ScaledUsedBreakdown         *UsedCollateralBreakdown
	Error                       error
}

// UsedCollateralBreakdown provides a detailed
// breakdown of where collateral is currently being allocated
type UsedCollateralBreakdown struct {
	LockedInStakes                  decimal.Decimal
	LockedInNFTBids                 decimal.Decimal
	LockedInFeeVoucher              decimal.Decimal
	LockedInSpotMarginFundingOffers decimal.Decimal
	LockedInSpotOrders              decimal.Decimal
	LockedAsCollateral              decimal.Decimal
	UsedInPositions                 decimal.Decimal
	UsedInSpotMarginBorrows         decimal.Decimal
}

// PositionController manages all futures orders
// across all exchanges assets and pairs
// its purpose is to handle the minutia of tracking
// and so all you need to do is send all orders to
// the position controller and its all tracked happily
type PositionController struct {
	m                     sync.Mutex
	multiPositionTrackers map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*MultiPositionTracker
	updated               time.Time
}

// MultiPositionTracker will track the performance of
// futures positions over time. If an old position tracker
// is closed, then the position controller will create a new one
// to track the current positions
type MultiPositionTracker struct {
	m                  sync.Mutex
	exchange           string
	asset              asset.Item
	pair               currency.Pair
	underlying         currency.Code
	collateralCurrency currency.Code
	positions          []*PositionTracker
	// order positions allows for an easier time knowing which order is
	// part of which position tracker
	orderPositions             map[string]*PositionTracker
	offlinePNLCalculation      bool
	useExchangePNLCalculations bool
	exchangePNLCalculation     PNLCalculation
}

// MultiPositionTrackerSetup holds the parameters
// required to set up a multi position tracker
type MultiPositionTrackerSetup struct {
	Exchange                  string
	Asset                     asset.Item
	Pair                      currency.Pair
	Underlying                currency.Code
	CollateralCurrency        currency.Code
	OfflineCalculation        bool
	UseExchangePNLCalculation bool
	ExchangePNLCalculation    PNLCalculation
}

// PositionTracker tracks futures orders until the overall position is considered closed
// eg a user can open a short position, append to it via two more shorts, reduce via a small long and
// finally close off the remainder via another long. All of these actions are to be
// captured within one position tracker. It allows for a user to understand their PNL
// specifically for futures positions. Utilising spot/futures arbitrage will not be tracked
// completely within this position tracker, however, can still provide a good
// timeline of performance until the position is closed
type PositionTracker struct {
	m                         sync.Mutex
	useExchangePNLCalculation bool
	collateralCurrency        currency.Code
	offlinePNLCalculation     bool
	PNLCalculation
	exchange           string
	asset              asset.Item
	contractPair       currency.Pair
	underlying         currency.Code
	exposure           decimal.Decimal
	openingDirection   Side
	openingPrice       decimal.Decimal
	openingSize        decimal.Decimal
	openingDate        time.Time
	latestDirection    Side
	latestPrice        decimal.Decimal
	lastUpdated        time.Time
	unrealisedPNL      decimal.Decimal
	realisedPNL        decimal.Decimal
	status             Status
	closingPrice       decimal.Decimal
	closingDate        time.Time
	shortPositions     []Detail
	longPositions      []Detail
	pnlHistory         []PNLResult
	fundingRateDetails *fundingrate.Rates
}

// PositionTrackerSetup contains all required fields to
// setup a position tracker
type PositionTrackerSetup struct {
	Exchange                  string
	Asset                     asset.Item
	Pair                      currency.Pair
	EntryPrice                decimal.Decimal
	Underlying                currency.Code
	CollateralCurrency        currency.Code
	Side                      Side
	UseExchangePNLCalculation bool
	OfflineCalculation        bool
	PNLCalculator             PNLCalculation
}

// TotalCollateralCalculator holds many collateral calculators
// to calculate total collateral standing with one struct
type TotalCollateralCalculator struct {
	CollateralAssets []CollateralCalculator
	CalculateOffline bool
	FetchPositions   bool
}

// CollateralCalculator is used to determine
// the size of collateral holdings for an exchange
// eg on Bybit, the collateral is scaled depending on what
// currency it is
type CollateralCalculator struct {
	CalculateOffline   bool
	CollateralCurrency currency.Code
	Asset              asset.Item
	Side               Side
	USDPrice           decimal.Decimal
	IsLiquidating      bool
	IsForNewPosition   bool
	FreeCollateral     decimal.Decimal
	LockedCollateral   decimal.Decimal
	UnrealisedPNL      decimal.Decimal
}

// PNLCalculator implements the PNLCalculation interface
// to call CalculatePNL and is used when a user wishes to have a
// consistent method of calculating PNL across different exchanges
type PNLCalculator struct{}

// PNLCalculatorRequest is used to calculate PNL values
// for an open position
type PNLCalculatorRequest struct {
	Pair             currency.Pair
	CalculateOffline bool
	Underlying       currency.Code
	Asset            asset.Item
	Leverage         decimal.Decimal
	EntryPrice       decimal.Decimal
	EntryAmount      decimal.Decimal
	Amount           decimal.Decimal
	CurrentPrice     decimal.Decimal
	PreviousPrice    decimal.Decimal
	Time             time.Time
	OrderID          string
	Fee              decimal.Decimal
	PNLHistory       []PNLResult
	Exposure         decimal.Decimal
	OrderDirection   Side
	OpeningDirection Side
	CurrentDirection Side
}

// PNLResult stores a PNL result from a point in time
type PNLResult struct {
	Status                Status
	Time                  time.Time
	UnrealisedPNL         decimal.Decimal
	RealisedPNLBeforeFees decimal.Decimal
	RealisedPNL           decimal.Decimal
	Price                 decimal.Decimal
	Exposure              decimal.Decimal
	Direction             Side
	Fee                   decimal.Decimal
	IsLiquidated          bool
	// Is event is supposed to show that something has happened and it isn't just tracking in time
	IsOrder bool
}

// Position is a basic holder for position information
type Position struct {
	Exchange           string
	Asset              asset.Item
	Pair               currency.Pair
	Underlying         currency.Code
	CollateralCurrency currency.Code
	RealisedPNL        decimal.Decimal
	UnrealisedPNL      decimal.Decimal
	Status             Status
	OpeningDate        time.Time
	OpeningPrice       decimal.Decimal
	OpeningSize        decimal.Decimal
	OpeningDirection   Side
	LatestPrice        decimal.Decimal
	LatestSize         decimal.Decimal
	LatestDirection    Side
	LastUpdated        time.Time
	CloseDate          time.Time
	Orders             []Detail
	PNLHistory         []PNLResult
	FundingRates       fundingrate.Rates
}

// PositionSummaryRequest is used to request a summary of an open position
type PositionSummaryRequest struct {
	Asset asset.Item
	Pair  currency.Pair

	// offline calculation requirements below
	CalculateOffline          bool
	Direction                 Side
	FreeCollateral            decimal.Decimal
	TotalCollateral           decimal.Decimal
	OpeningPrice              decimal.Decimal
	CurrentPrice              decimal.Decimal
	OpeningSize               decimal.Decimal
	CurrentSize               decimal.Decimal
	CollateralUsed            decimal.Decimal
	NotionalPrice             decimal.Decimal
	Leverage                  decimal.Decimal
	MaxLeverageForAccount     decimal.Decimal
	TotalAccountValue         decimal.Decimal
	TotalOpenPositionNotional decimal.Decimal
}

// PositionSummary returns basic details on an open position
type PositionSummary struct {
	MaintenanceMarginRequirement decimal.Decimal
	InitialMarginRequirement     decimal.Decimal
	EstimatedLiquidationPrice    decimal.Decimal
	CollateralUsed               decimal.Decimal
	MarkPrice                    decimal.Decimal
	CurrentSize                  decimal.Decimal
	BreakEvenPrice               decimal.Decimal
	AverageOpenPrice             decimal.Decimal
	RecentPNL                    decimal.Decimal
	MarginFraction               decimal.Decimal
	FreeCollateral               decimal.Decimal
	TotalCollateral              decimal.Decimal
}

// PositionDetails are used to track open positions
// in the order manager
type PositionDetails struct {
	Exchange string
	Asset    asset.Item
	Pair     currency.Pair
	Orders   []Detail
}

// PositionsRequest defines the request to
// retrieve futures position data
type PositionsRequest struct {
	Asset     asset.Item
	Pairs     currency.Pairs
	StartDate time.Time
}
