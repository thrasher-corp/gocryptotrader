package size

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/config"
)

func TestSizingAccuracy(t *testing.T) {
	globalMinMax := config.MinMax{
		MinimumSize:  0,
		MaximumSize:  1,
		MaximumTotal: 1337,
	}
	sizer := Size{
		Leverage: config.Leverage{},
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 1338.0
	feeRate := 0.02

	amountWithoutFee, err := sizer.calculateSize(price, availableFunds, feeRate, globalMinMax)
	if err != nil {
		t.Error(err)
	}
	totalWithFee := (price * amountWithoutFee) + (globalMinMax.MaximumTotal * feeRate)
	if totalWithFee != globalMinMax.MaximumTotal {
		t.Log("incorrect amount calculation")
	}
}

func TestSizingOverMaxSize(t *testing.T) {
	globalMinMax := config.MinMax{
		MinimumSize:  0,
		MaximumSize:  0.5,
		MaximumTotal: 1337,
	}
	sizer := Size{
		Leverage: config.Leverage{},
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 1338.0
	feeRate := 0.02

	amount, err := sizer.calculateSize(price, availableFunds, feeRate, globalMinMax)
	if err != nil {
		t.Error(err)
	}
	if amount > globalMinMax.MaximumSize {
		t.Error("greater than max")
	}
	if amount+feeRate > globalMinMax.MaximumSize {
		t.Error("greater than max")

	}
}

func TestSizingUnderMinSize(t *testing.T) {
	globalMinMax := config.MinMax{
		MinimumSize:  1,
		MaximumSize:  2,
		MaximumTotal: 1337,
	}
	sizer := Size{
		Leverage: config.Leverage{},
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 1338.0
	feeRate := 0.02

	_, err := sizer.calculateSize(price, availableFunds, feeRate, globalMinMax)
	if err != nil && err.Error() != "sized amount less than minimum 1" {
		t.Error(err)
	}
}

func TestSizingErrors(t *testing.T) {
	globalMinMax := config.MinMax{
		MinimumSize:  1,
		MaximumSize:  2,
		MaximumTotal: 1337,
	}
	sizer := Size{
		Leverage: config.Leverage{},
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 0.0
	feeRate := 0.02

	_, err := sizer.calculateSize(price, availableFunds, feeRate, globalMinMax)
	if err != nil && err.Error() != "no fund available" {
		t.Error(err)
	}
}
