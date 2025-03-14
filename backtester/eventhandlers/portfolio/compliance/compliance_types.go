package compliance

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var errSnapshotNotFound = errors.New("snapshot not found")

// Manager holds a snapshot of all orders at each timeperiod, allowing
// study of all changes across time
type Manager struct {
	Snapshots []Snapshot
}

// Snapshot consists of the timestamp the snapshot is from, along with all orders made
// up until that time
type Snapshot struct {
	Offset    int64           `json:"offset"`
	Timestamp time.Time       `json:"timestamp"`
	Orders    []SnapshotOrder `json:"orders"`
}

// SnapshotOrder adds some additional data that's only relevant for backtesting
// to the order.Detail without adding to order.Detail
type SnapshotOrder struct {
	ClosePrice          decimal.Decimal `json:"close-price"`
	VolumeAdjustedPrice decimal.Decimal `json:"volume-adjusted-price"`
	SlippageRate        decimal.Decimal `json:"slippage-rate"`
	CostBasis           decimal.Decimal `json:"cost-basis"`
	Order               *order.Detail   `json:"order-detail"`
}
