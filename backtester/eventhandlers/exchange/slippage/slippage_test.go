package slippage

import "testing"

func TestRandomSlippage(t *testing.T) {
	resp := EstimateSlippagePercentage(80, 100, gctcommon.BuySide)
	if resp < 0.8 || resp > 1 {
		t.Error("expected result > 0.8 and < 100")
	}
}
