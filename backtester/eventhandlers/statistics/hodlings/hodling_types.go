package hodlings

import (
	"time"
)

type Snapshots struct {
	Hodlings []Hodling
}

type Hodling struct {
	Timestamp      time.Time
	InitialFunds   float64
	PositionsSize  float64
	PositionsValue float64
	AmountSold     float64
	AmountBought   float64
	RemainingFunds float64
	TotalValue     float64
}
