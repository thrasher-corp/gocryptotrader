package holdings

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// ErrInitialFundsZero is an error when initial funds are zero or less
var ErrInitialFundsZero = errors.New("initial funds <= 0")

// Holding contains pricing statistics for a given time
// for a given exchange asset pair
type Holding struct {
	Offset         int64
	Pair           currency.Pair `json:"pair"`
	Asset          asset.Item    `json:"asset"`
	Exchange       string        `json:"exchange"`
	Timestamp      time.Time     `json:"timestamp"`
	InitialFunds   float64       `json:"initial-funds"`
	PositionsSize  float64       `json:"positions-size"`
	PositionsValue float64       `json:"postions-value"`
	SoldAmount     float64       `json:"sold-amount"`
	SoldValue      float64       `json:"sold-value"`
	BoughtAmount   float64       `json:"bought-amount"`
	BoughtValue    float64       `json:"bought-value"`
	RemainingFunds float64       `json:"remaining-funds"`
	CommittedFunds float64       `json:"committed-funds"`

	TotalValueDifference      float64
	ChangeInTotalValuePercent float64
	BoughtValueDifference     float64
	SoldValueDifference       float64
	PositionsValueDifference  float64

	TotalValue                   float64 `json:"total-value"`
	TotalFees                    float64 `json:"total-fees"`
	TotalValueLostToVolumeSizing float64 `json:"total-value-lost-to-volume-sizing"`
	TotalValueLostToSlippage     float64 `json:"total-value-lost-to-slippage"`
	TotalValueLost               float64 `json:"total-value-lost"`

	RiskFreeRate float64 `json:"risk-free-rate"`
}
