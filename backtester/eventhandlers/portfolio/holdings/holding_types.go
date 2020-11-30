package holdings

import (
	"time"
)

type Snapshots struct {
	Hodlings []Holding
}

type Holding struct {
	Timestamp      time.Time
	InitialFunds   float64
	PositionsSize  float64
	PositionsValue float64
	SoldAmount     float64
	SoldValue      float64
	BoughtAmount   float64
	BoughtValue    float64
	RemainingFunds float64
	TotalValue     float64
	TotalFees      float64
}
