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
	Orders []order.Detail
	Time   time.Time
}
