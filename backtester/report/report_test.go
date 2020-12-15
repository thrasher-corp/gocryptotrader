package report

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
)

func TestGenerateReport(t *testing.T) {
	err := GenerateReport(statistics.Statistic{StrategyName: "butts"})
	if err != nil {
		t.Error(err)
	}
}
