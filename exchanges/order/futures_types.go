package order

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	errExchangeNameEmpty              = errors.New("exchange name empty")
	errNotFutureAsset                 = errors.New("asset type is not futures")
	errTimeUnset                      = errors.New("time unset")
	errMissingPNLCalculationFunctions = errors.New("futures tracker requires exchange PNL calculation functions")
	errOrderNotEqualToTracker         = errors.New("order does not match tracker data")
	errPositionClosed                 = errors.New("the position is closed")
	errPositionNotClosed              = errors.New("the position is not closed")
	errPositionDiscrepancy            = errors.New("there is a position considered open, but it is not the latest, please review")
	errAssetMismatch                  = errors.New("provided asset does not match")
	errEmptyUnderlying                = errors.New("underlying asset unset")
	errNilSetup                       = errors.New("nil setup received")
	errNilOrder                       = errors.New("nil order received")
	errNoPNLHistory                   = errors.New("no pnl history")
	errCannotCalculateUnrealisedPNL   = errors.New("cannot calculate unrealised PNL, order is not open")

	// ErrNilPNLCalculator is raised when pnl calculation is requested for
	// an exchange, but the fields are not set properly
	ErrNilPNLCalculator = errors.New("nil pnl calculator received")
	// ErrPositionLiquidated is raised when checking PNL status only for
	// it to be liquidated
	ErrPositionLiquidated = errors.New("position liquidated")
)

// PNLCalculation is an interface to allow multiple
// ways of calculating PNL to be used for futures positions
type PNLCalculation interface {
	CalculatePNL(*PNLCalculator) (*PNLResult, error)
}

// CollateralManagement is an interface that allows
// multiple ways of calculating the size of collateral
// on an exchange
type CollateralManagement interface {
	ScaleCollateral(*CollateralCalculator) (decimal.Decimal, error)
	CalculateTotalCollateral([]CollateralCalculator) (decimal.Decimal, error)
}

// PositionController manages all futures orders
// across all exchanges assets and pairs
// its purpose is to handle the minutia of tracking
// and so all you need to do is send all orders to
// the position controller and its all tracked happily
type PositionController struct {
	positionTrackerControllers map[string]map[asset.Item]map[currency.Pair]*MultiPositionTracker
}

// MultiPositionTracker will track the performance of
// futures positions over time. If an old position tracker
// is closed, then the position controller will create a new one
// to track the current positions
type MultiPositionTracker struct {
	exchange   string
	asset      asset.Item
	pair       currency.Pair
	underlying currency.Code
	positions  []*PositionTracker
	// order positions allows for an easier time knowing which order is
	// part of which position tracker
	orderPositions             map[string]*PositionTracker
	pnl                        decimal.Decimal
	offlinePNLCalculation      bool
	useExchangePNLCalculations bool
	exchangePNLCalculation     PNLCalculation
}

// PositionControllerSetup holds the parameters
// required to set up a position controller
type PositionControllerSetup struct {
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
	exchange              string
	asset                 asset.Item
	contractPair          currency.Pair
	underlyingAsset       currency.Code
	exposure              decimal.Decimal
	currentDirection      Side
	openingDirection      Side
	status                Status
	averageLeverage       decimal.Decimal
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
	EntryPrice                float64
	Underlying                currency.Code
	Asset                     asset.Item
	Side                      Side
	UseExchangePNLCalculation bool
}

// CollateralCalculator is used to determine
// the size of collateral holdings for an exchange
// eg on FTX, the collateral is scaled depending on what
// currency it is
type CollateralCalculator struct {
	CollateralCurrency currency.Code
	Asset              asset.Item
	Side               Side
	CollateralAmount   decimal.Decimal
	USDPrice           decimal.Decimal
	IsLiquidating      bool
}

// PNLCalculator is used to calculate PNL values
// for an open position
type PNLCalculator struct {
	OrderBasedCalculation    *Detail
	TimeBasedCalculation     *TimeBasedCalculation
	ExchangeBasedCalculation *ExchangeBasedCalculation
}

// TimeBasedCalculation will update PNL values
// based on the current time
type TimeBasedCalculation struct {
	Time         time.Time
	CurrentPrice float64
}

// ExchangeBasedCalculation are the fields required to
// calculate PNL using an exchange's custom PNL calculations
// eg FTX uses a different method than Binance to calculate PNL
// values
type ExchangeBasedCalculation struct {
	Pair             currency.Pair
	CalculateOffline bool
	Underlying       currency.Code
	Asset            asset.Item
	Side             Side
	Leverage         float64
	EntryPrice       float64
	EntryAmount      float64
	Amount           float64
	CurrentPrice     float64
	PreviousPrice    float64
	Time             time.Time
	OrderID          string
	Fee              decimal.Decimal
}

// PNLResult stores pnl history at a point in time
type PNLResult struct {
	Time          time.Time
	UnrealisedPNL decimal.Decimal
	RealisedPNL   decimal.Decimal
	Price         decimal.Decimal
	Exposure      decimal.Decimal
	Fee           decimal.Decimal
}
