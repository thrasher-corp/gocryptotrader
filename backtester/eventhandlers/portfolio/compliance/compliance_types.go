package compliance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Manager struct {
	Interval  kline.Interval
	Snapshots []Snapshot
}

type Snapshot struct {
	Orders    []SnapshotOrder `json:"orders"`
	Timestamp time.Time       `json:"timestamp"`
}

// SnapshotOrder adds some additional data that's only relevant for backtesting
// to the order.Detail without adding to order.Detail
type SnapshotOrder struct {
	ClosePrice          float64 `json:"close-price"`
	VolumeAdjustedPrice float64 `json:"volume-adjusted-price"`
	SlippageRate        float64 `json:"slippage-rate"`
	CostBasis           float64 `json:"cost-basis"`
	*order.Detail       `json:"order-detail"`
}
