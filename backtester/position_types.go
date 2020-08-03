package backtest

import "time"

type Position struct {
	timestamp    time.Time
	Amount       float64
	AmountBought float64
	AmountSold   float64

	avgPrice    float64
	avgPriceNet float64
	avgPriceBOT float64
	avgPriceSLD float64

	value    float64
	valueBOT float64
	valueSLD float64

	netValue    float64
	netValueBOT float64
	netValueSLD float64

	marketValue float64
	exchangeFee float64
	cost        float64
	costBasis   float64

	realProfitLoss   float64
	unrealProfitLoss float64
	totalProfitLoss  float64
}
