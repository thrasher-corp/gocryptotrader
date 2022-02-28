package order

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	// ErrPositionClosed returned when attempting to amend a closed position
	ErrPositionClosed = errors.New("the position is closed")
	// ErrPositionsNotLoadedForExchange returned when no position data exists for an exchange
	ErrPositionsNotLoadedForExchange = errors.New("no positions loaded for exchange")
	// ErrPositionsNotLoadedForAsset returned when no position data exists for an asset
	ErrPositionsNotLoadedForAsset = errors.New("no positions loaded for asset")
	// ErrPositionsNotLoadedForPair returned when no position data exists for a pair
	ErrPositionsNotLoadedForPair = errors.New("no positions loaded for pair")
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

	errExchangeNameEmpty              = errors.New("exchange name empty")
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
)

// PNLCalculation is an interface to allow multiple
// ways of calculating PNL to be used for futures positions
type PNLCalculation interface {
	CalculatePNL(context.Context, *PNLCalculatorRequest) (*PNLResult, error)
}

// CollateralManagement is an interface that allows
// multiple ways of calculating the size of collateral
// on an exchange
type CollateralManagement interface {
	ScaleCollateral(ctx context.Context, calculator *CollateralCalculator) (*CollateralByCurrency, error)
	CalculateTotalCollateral(context.Context, *TotalCollateralCalculator) (*TotalCollateralResponse, error)
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
// eg in FTX ScaledCurrency is USD
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
	m                          sync.Mutex
	positionTrackerControllers map[string]map[asset.Item]map[currency.Pair]*MultiPositionTracker
}

// MultiPositionTracker will track the performance of
// futures positions over time. If an old position tracker
// is closed, then the position controller will create a new one
// to track the current positions
type MultiPositionTracker struct {
	m          sync.Mutex
	exchange   string
	asset      asset.Item
	pair       currency.Pair
	underlying currency.Code
	positions  []*PositionTracker
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
	m                     sync.Mutex
	exchange              string
	asset                 asset.Item
	contractPair          currency.Pair
	underlyingAsset       currency.Code
	exposure              decimal.Decimal
	currentDirection      Side
	openingDirection      Side
	status                Status
	unrealisedPNL         decimal.Decimal
	realisedPNL           decimal.Decimal
	shortPositions        []Detail
	longPositions         []Detail
	pnlHistory            []PNLResult
	entryPrice            decimal.Decimal
	closingPrice          decimal.Decimal
	offlinePNLCalculation bool
	PNLCalculation
	latestPrice               decimal.Decimal
	useExchangePNLCalculation bool
}

// PositionTrackerSetup contains all required fields to
// setup a position tracker
type PositionTrackerSetup struct {
	Pair                      currency.Pair
	EntryPrice                decimal.Decimal
	Underlying                currency.Code
	Asset                     asset.Item
	Side                      Side
	UseExchangePNLCalculation bool
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
// eg on FTX, the collateral is scaled depending on what
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
	Time                  time.Time
	UnrealisedPNL         decimal.Decimal
	RealisedPNLBeforeFees decimal.Decimal
	Price                 decimal.Decimal
	Exposure              decimal.Decimal
	Fee                   decimal.Decimal
	IsLiquidated          bool
}

// PositionStats is a basic holder
// for position information
type PositionStats struct {
	Exchange         string
	Asset            asset.Item
	Pair             currency.Pair
	Underlying       currency.Code
	Orders           []Detail
	RealisedPNL      decimal.Decimal
	UnrealisedPNL    decimal.Decimal
	LatestDirection  Side
	Status           Status
	OpeningDirection Side
	OpeningPrice     decimal.Decimal
	LatestPrice      decimal.Decimal
	PNLHistory       []PNLResult
}
