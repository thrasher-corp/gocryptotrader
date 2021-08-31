package holdings

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// ErrInitialFundsZero is an error when initial funds are zero or less
var ErrInitialFundsZero = errors.New("initial funds < 0")

// the concept of a holding here doesn't work in the context of item funding
// we cannot track order fees along with bought and sold amounts via the way its been changed
// portfolio snapshotting needs to be more nuanced?
//
/*
type Holding struct {
Offset         int64
	Item           currency.Code
	Asset          asset.Item      `json:"asset"`
	Exchange       string          `json:"exchange"`
	Timestamp      time.Time       `json:"timestamp"`
	InitialFunds   decimal.Decimal `json:"initial-funds"`
	AvailableFunds decimal.Decimal
	PurchasedAmount decimal.Decimal
	SoldAmount decimal.Decimal


*/

// Holding contains pricing statistics for a given time
// for a given exchange asset pair
type Holding struct {
	Offset         int64
	Item           currency.Code
	Pair           currency.Pair
	Asset          asset.Item      `json:"asset"`
	Exchange       string          `json:"exchange"`
	Timestamp      time.Time       `json:"timestamp"`
	InitialFunds   decimal.Decimal `json:"initial-funds"`
	PositionsSize  decimal.Decimal `json:"positions-size"`
	PositionsValue decimal.Decimal `json:"positions-value"`
	SoldAmount     decimal.Decimal `json:"sold-amount"`
	SoldValue      decimal.Decimal `json:"sold-value"`
	BoughtAmount   decimal.Decimal `json:"bought-amount"`
	BoughtValue    decimal.Decimal `json:"bought-value"`
	RemainingFunds decimal.Decimal `json:"remaining-funds"`
	CommittedFunds decimal.Decimal `json:"committed-funds"`

	TotalValueDifference      decimal.Decimal
	ChangeInTotalValuePercent decimal.Decimal
	BoughtValueDifference     decimal.Decimal
	SoldValueDifference       decimal.Decimal
	PositionsValueDifference  decimal.Decimal

	TotalValue                   decimal.Decimal `json:"total-value"`
	TotalFees                    decimal.Decimal `json:"total-fees"`
	TotalValueLostToVolumeSizing decimal.Decimal `json:"total-value-lost-to-volume-sizing"`
	TotalValueLostToSlippage     decimal.Decimal `json:"total-value-lost-to-slippage"`
	TotalValueLost               decimal.Decimal `json:"total-value-lost"`

	RiskFreeRate decimal.Decimal `json:"risk-free-rate"`
}
