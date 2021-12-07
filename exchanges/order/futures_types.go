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
	errPositionClosed                 = errors.New("the position is closed, time for a new one")
	errPositionDiscrepancy            = errors.New("there is a position considered open, but it is not the latest, please review")
	errAssetMismatch                  = errors.New("provided asset does not match")
)

type PNLManagement interface {
	CalculatePNL(*PNLCalculator) (*PNLResult, error)
}

type CollateralManagement interface {
	ScaleCollateral(*CollateralCalculator) (decimal.Decimal, error)
	CalculateCollateral(calculator *CollateralCalculator) (decimal.Decimal, error)
}

type CollateralCalculator struct {
	CollateralCurrency currency.Code
	Asset              asset.Item
	Side               Side
	CollateralAmount   decimal.Decimal
	EntryPrice         float64
}

type PNLCalculator struct {
	CalculateOffline bool
	Underlying       currency.Code
	OrderID          string
	Asset            asset.Item
	Side             Side
	Leverage         float64
	EntryPrice       float64
	OpeningAmount    float64
	Amount           float64
	MarkPrice        float64
	PrevMarkPrice    float64
	CurrentPrice     float64
}

type PNLResult struct {
	MarginFraction            decimal.Decimal
	EstimatedLiquidationPrice decimal.Decimal
	UnrealisedPNL             decimal.Decimal
	RealisedPNL               decimal.Decimal
	Collateral                decimal.Decimal
	IsLiquidated              bool
}

// ExchangeAssetPositionTracker will track the performance of
// po
type ExchangeAssetPositionTracker struct {
	Exchange              string
	Asset                 asset.Item
	Positions             []*PositionTracker
	PNL                   decimal.Decimal
	PNLCalculation        PNLManagement
	OfflinePNLCalculation bool
}

// PositionTracker tracks futures orders until the overall position is considered closed
// eg a user can open a short position, append to it via two more shorts, reduce via a small long and
// finally close off the remainder via another long. All of these actions are to be
// captured within one position tracker. It allows for a user to understand their PNL
// specifically for futures positions. Utilising spot/futures arbitrage will not be tracked
// completely within this position tracker, however, can still provide a good
// timeline of performance until the position is closed
type PositionTracker struct {
	Exchange              string
	Asset                 asset.Item
	ContractPair          currency.Pair
	UnderlyingAsset       currency.Code
	Exposure              decimal.Decimal
	CurrentDirection      Side
	Status                Status
	AverageLeverage       decimal.Decimal
	UnrealisedPNL         decimal.Decimal
	RealisedPNL           decimal.Decimal
	ShortPositions        []Detail
	LongPositions         []Detail
	PNLHistory            []PNLHistory
	EntryPrice            decimal.Decimal
	ClosingPrice          decimal.Decimal
	OfflinePNLCalculation bool
	PNLCalculation        PNLManagement
}

// PNLHistory tracks how a futures contract
// pnl is going over the history of exposure
type PNLHistory struct {
	Time          time.Time
	UnrealisedPNL decimal.Decimal
	RealisedPNL   decimal.Decimal
}
