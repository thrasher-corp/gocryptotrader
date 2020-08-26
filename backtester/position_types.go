package backtest

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

type Positions struct {
	timestamp      time.Time
	pair           currency.Pair
	amount         float64
	amountBought   float64
	amountSold     float64
	avgPrice       float64
	avgPriceNet    float64
	avgPriceBought float64
	avgPriceSold   float64
	value          float64
	valueBought    float64
	valueSold      float64
	netValue       float64
	netValueBought float64
	netValueSold   float64
	marketPrice    float64
	marketValue    float64
	commission     float64
	exchangeFee    float64
	cost           float64
	costBasis      float64

	realProfitLoss   float64
	unrealProfitLoss float64
	totalProfitLoss  float64
}
