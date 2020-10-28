package positions

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

type Positions struct {
	Timestamp          time.Time
	Pair               currency.Pair
	Amount             float64
	AmountBought       float64
	AmountSold         float64
	AveragePrice       float64
	AveragePriceNet    float64
	AveragePriceBought float64
	AveragePriceSold   float64
	Value              float64
	ValueBought        float64
	ValueSold          float64
	NetValue           float64
	NetValueBought     float64
	NetValueSold       float64
	MarketPrice        float64
	MarketValue        float64
	Commission         float64
	ExchangeFee        float64
	Cost               float64
	CostBasis          float64

	RealProfitLoss   float64
	UnrealProfitLoss float64
	TotalProfitLoss  float64
}
