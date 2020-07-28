package position

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

type Position struct {
	Time time.Time

	Pair currency.Pair

	Quantity       float64
	QuantityBought float64
	QuantitySold   float64

	AveragePrice       float64
	AveragePriceBought float64
	AveragePriceSold   float64

	ExchangeFee float64

	RealisedPNL   float64
	UnrealisedPNL float64
	TotalPNL      float64
}

