package compliance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Manager struct {
	Snapshots []Snapshot
}

type Snapshot struct {
	Orders []order.Detail
	Time   time.Time
}
