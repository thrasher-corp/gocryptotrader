package hodlings

import (
	"time"
)

type Snapshots struct {
	Hodlings []Hodling
}

type Hodling struct {
	Timestamp time.Time
	//ScottStuff
	InitialFunds  float64
	PositionSize  float64
	PositionValue float64
	TotalWorth    float64
	AmountSold    float64
	AmountBought  float64

	// ==================
	Amount float64
	//AmountBought       float64
	//AmountSold         float64
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
	Cost               float64
	CostBasis          float64

	RealProfitLoss   float64
	UnrealProfitLoss float64
	TotalProfitLoss  float64
}
