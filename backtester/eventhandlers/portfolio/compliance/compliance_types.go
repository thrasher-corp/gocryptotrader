package compliance

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errSnapshotNotFound = errors.New("snapshot not found")
)

// Manager holds a snapshot of all orders at each timeperiod, allowing
// study of all changes across time
type Manager struct {
	Snapshots []Snapshot
}

// Snapshot consists of the timestamp the snapshot is from, along with all orders made
// up until that time
type Snapshot struct {
	Orders    []SnapshotOrder `json:"orders"`
	Timestamp time.Time       `json:"timestamp"`
	Offset    int64           `json:"offset"`
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
