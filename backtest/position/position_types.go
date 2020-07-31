package position

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

type Position struct {
	Time time.Time

	Pair currency.Pair

	Amount       float64
	AmountBought float64
	AmountSold   float64

	AveragePrice       float64
	AveragePriceNet    float64
	AveragePriceBought float64
	AveragePriceSold   float64

	Value       float64
	ValueBought float64
	ValueSold   float64

	NetValue     float64
	NetValueBUY  float64
	NetValueSold float64

	MarketLastPrice float64
	MarketValue     float64

	Price      float64
	PriceBasis float64

	ExchangeFee float64

	RealisedPNL   float64
	UnrealisedPNL float64
	TotalPNL      float64
}
