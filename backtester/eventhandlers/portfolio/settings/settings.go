package settings

import (
	"sort"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
)

func (e *Settings) GetLatestHoldings() holdings.Holding {
	if e.HoldingsSnapshots.Holdings == nil {
		// no holdings yet
		return holdings.Holding{}
	}
	sort.SliceStable(e.HoldingsSnapshots.Holdings, func(i, j int) bool {
		return e.HoldingsSnapshots.Holdings[i].Timestamp.Before(e.HoldingsSnapshots.Holdings[j].Timestamp)
	})

	return e.HoldingsSnapshots.Holdings[len(e.HoldingsSnapshots.Holdings)-1]
}

func (e *Settings) Value() float64 {
	latest := e.GetLatestHoldings()
	return latest.TotalValue
}
