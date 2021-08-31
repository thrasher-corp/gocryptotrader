package settings

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
)

// GetLatestHoldings returns the latest holdings after being sorted by time
func (e *Settings) GetLatestHoldings() holdings.Holding {
	if e.HoldingHolder == nil {
		// no holdings yet
		return holdings.Holding{Offset: 1}
	}

	return e.HoldingHolder[len(e.HoldingHolder)-1]
}

// GetHoldingsForTime returns the holdings for a time period, or an empty holding if not found
func (e *Settings) GetHoldingsForTime(t time.Time) holdings.Holding {
	if e.HoldingHolder == nil {
		// no holdings yet
		return holdings.Holding{}
	}
	for i := len(e.HoldingHolder) - 1; i >= 0; i-- {
		if e.HoldingHolder[i].Timestamp.Equal(t) {
			return e.HoldingHolder[i]
		}
	}
	return holdings.Holding{}
}

// Value returns the total value of the latest holdings
func (e *Settings) Value() decimal.Decimal {
	latest := e.GetLatestHoldings()
	return latest.TotalValue
}
