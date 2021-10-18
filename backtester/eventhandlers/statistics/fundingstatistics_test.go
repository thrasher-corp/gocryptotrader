package statistics

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestCalculateTotalUSDFundingStatistics(t *testing.T) {
	t.Parallel()
	_, err := CalculateTotalUSDFundingStatistics(nil, nil)
	if !errors.Is(err, funding.ErrFundsNotFound) {
		t.Errorf("received %v expected %v", err, funding.ErrFundsNotFound)
	}
	f := funding.SetupFundingManager(true, false)
	item, err := funding.CreateItem("binance", asset.Spot, currency.BTC, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	err = f.AddItem(item)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}

	_, err = CalculateTotalUSDFundingStatistics(f, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received %v expected %v", err, common.ErrNilArguments)
	}
}

func TestCalculateIndividualFundingStatistics(t *testing.T) {

}
