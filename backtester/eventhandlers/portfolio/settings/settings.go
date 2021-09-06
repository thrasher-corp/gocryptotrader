package settings

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
)

// GetLatestHoldings returns the latest holdings after being sorted by time
func (e *Settings) GetLatestHoldings() *holdings.Holding {
	if len(e.HoldingsSnapshots) == 0 {
		return nil
	}

	return e.HoldingsSnapshots[len(e.HoldingsSnapshots)-1]
}

// GetHoldingsForTime returns the holdings for a time period, or an empty holding if not found
func (e *Settings) GetHoldingsForTime(t time.Time) *holdings.Holding {
	if e.HoldingsSnapshots == nil {
		// no holdings yet
		return nil
	}
	for i := len(e.HoldingsSnapshots) - 1; i >= 0; i-- {
		if e.HoldingsSnapshots[i].Timestamp.Equal(t) {
			return e.HoldingsSnapshots[i]
		}
	}
	return nil
}

// Value returns the total value of the latest holdings
func (e *Settings) Value() decimal.Decimal {
	latest := e.GetLatestHoldings()
	if latest == nil {
		return decimal.Zero
	}
	return latest.TotalValue
}
