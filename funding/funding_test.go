package funding

import "testing"

func TestGetFundingRates(t *testing.T) {
	err := GetFundingRates("https://www.binance.com/en/futures/funding-history/0")
	if err != nil {
		t.Error(err)
	}
}
