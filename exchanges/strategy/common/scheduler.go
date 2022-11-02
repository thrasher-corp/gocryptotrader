package common

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// Scheduler defines scheduling assistance for strategies
type Scheduler struct {
	// start defines scheduled start time
	start time.Time
	// end defines scheduled end time
	end time.Time
	// alignmentToUTC allows the heartbeat of strategy to occur at actual
	// candle close
	alignmentToUTC bool
	// interval defines the actual lowest interval as heart beat to execute
	// strategy.
	interval kline.Interval
	// TODO: // offset allows for the shift in heartbeat
	// offset kline.Interval
}
