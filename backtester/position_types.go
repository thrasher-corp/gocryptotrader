package backtest

import "time"

type Position struct {
	timestamp    time.Time
	Amount       float64
	AmountBought float64
	AmountSold   float64

	avgPrice    float64
	avgPriceNet float64
	avgPriceBought float64
	avgPriceSold float64

	value    float64
	valueBought float64
	valueSold float64

	netValue    float64
	netValueBought float64
	netValueSold float64

	marketValue float64
	exchangeFee float64
	cost        float64
	costBasis   float64

	realProfitLoss   float64
	unrealProfitLoss float64
	totalProfitLoss  float64
}
