package holdings

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type Snapshots struct {
	Holdings []Holding
}

type Holding struct {
	Pair           currency.Pair
	Asset          asset.Item
	Exchange       string
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
