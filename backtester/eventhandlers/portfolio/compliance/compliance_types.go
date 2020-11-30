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
	Orders []SnapshotOrder
	Time   time.Time
}

// SnapshotOrder adds some additional data that's only relevant for backtesting
// to the order.Detail without adding to order.Detail
type SnapshotOrder struct {
	ClosePrice          float64
	VolumeAdjustedPrice float64
	SlippageRate        float64
	CostBasis           float64
	*order.Detail
}
