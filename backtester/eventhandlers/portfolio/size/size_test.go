package size

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestSizingAccuracy(t *testing.T) {
	t.Parallel()
	globalMinMax := config.MinMax{
		MinimumSize:  0,
		MaximumSize:  1,
		MaximumTotal: 1337,
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 1338.0
	feeRate := 0.02
	var buylimit float64 = 1
	amountWithoutFee, err := sizer.calculateBuySize(price, availableFunds, feeRate, buylimit, globalMinMax)
	if err != nil {
		t.Error(err)
	}
	totalWithFee := (price * amountWithoutFee) + (globalMinMax.MaximumTotal * feeRate)
	if totalWithFee != globalMinMax.MaximumTotal {
		t.Error("incorrect amount calculation")
	}
}

func TestSizingOverMaxSize(t *testing.T) {
	t.Parallel()
	globalMinMax := config.MinMax{
		MinimumSize:  0,
		MaximumSize:  0.5,
		MaximumTotal: 1337,
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 1338.0
	feeRate := 0.02
	var buylimit float64 = 1
	amount, err := sizer.calculateBuySize(price, availableFunds, feeRate, buylimit, globalMinMax)
	if err != nil {
		t.Error(err)
	}
	if amount > globalMinMax.MaximumSize {
		t.Error("greater than max")
	}
}

func TestSizingUnderMinSize(t *testing.T) {
	t.Parallel()
	globalMinMax := config.MinMax{
		MinimumSize:  1,
		MaximumSize:  2,
		MaximumTotal: 1337,
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 1338.0
	feeRate := 0.02
	var buylimit float64 = 1
	_, err := sizer.calculateBuySize(price, availableFunds, feeRate, buylimit, globalMinMax)
	if !errors.Is(err, errLessThanMinimum) {
		t.Errorf("expected: %v, received %v", errLessThanMinimum, err)
	}
}

func TestMaximumBuySizeEqualZero(t *testing.T) {
	t.Parallel()
	globalMinMax := config.MinMax{
		MinimumSize:  1,
		MaximumSize:  0,
		MaximumTotal: 1437,
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 13380.0
	feeRate := 0.02
	var buylimit float64 = 1
	amount, err := sizer.calculateBuySize(price, availableFunds, feeRate, buylimit, globalMinMax)
	if amount != buylimit || err != nil {
		t.Errorf("expected: %v, received %v, err: %+v", buylimit, amount, err)
	}
}
func TestMaximumSellSizeEqualZero(t *testing.T) {
	t.Parallel()
	globalMinMax := config.MinMax{
		MinimumSize:  1,
		MaximumSize:  0,
		MaximumTotal: 1437,
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 13380.0
	feeRate := 0.02
	var selllimit float64 = 1
	amount, err := sizer.calculateSellSize(price, availableFunds, feeRate, selllimit, globalMinMax)
	if amount != selllimit || err != nil {
		t.Errorf("expected: %v, received %v, err: %+v", selllimit, amount, err)
	}
}

func TestSizingErrors(t *testing.T) {
	t.Parallel()
	globalMinMax := config.MinMax{
		MinimumSize:  1,
		MaximumSize:  2,
		MaximumTotal: 1337,
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 0.0
	feeRate := 0.02
	var buylimit float64 = 1
	_, err := sizer.calculateBuySize(price, availableFunds, feeRate, buylimit, globalMinMax)
	if !errors.Is(err, errNoFunds) {
		t.Errorf("expected: %v, received %v", errNoFunds, err)
	}
}

func TestCalculateSellSize(t *testing.T) {
	t.Parallel()
	globalMinMax := config.MinMax{
		MinimumSize:  1,
		MaximumSize:  2,
		MaximumTotal: 1337,
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := 1338.0
	availableFunds := 0.0
	feeRate := 0.02
	var sellLimit float64 = 1
	_, err := sizer.calculateSellSize(price, availableFunds, feeRate, sellLimit, globalMinMax)
	if !errors.Is(err, errNoFunds) {
		t.Errorf("expected: %v, received %v", errNoFunds, err)
	}
	availableFunds = 1337
	_, err = sizer.calculateSellSize(price, availableFunds, feeRate, sellLimit, globalMinMax)
	if !errors.Is(err, errLessThanMinimum) {
		t.Errorf("expected: %v, received %v", errLessThanMinimum, err)
	}
	price = 12
	availableFunds = 1339
	_, err = sizer.calculateSellSize(price, availableFunds, feeRate, sellLimit, globalMinMax)
	if err != nil {
		t.Error(err)
	}
}

func TestSizeOrder(t *testing.T) {
	t.Parallel()
	s := Size{}
	_, err := s.SizeOrder(nil, 0, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Error(err)
	}
	o := &order.Order{}
	cs := &exchange.Settings{}
	_, err = s.SizeOrder(o, 0, cs)
	if !errors.Is(err, errNoFunds) {
		t.Errorf("expected: %v, received %v", errNoFunds, err)
	}

	_, err = s.SizeOrder(o, 1337, cs)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("expected: %v, received %v", errCannotAllocate, err)
	}

	o.Direction = gctorder.Buy
	o.Price = 1
	s.BuySide.MaximumSize = 1
	s.BuySide.MinimumSize = 1
	_, err = s.SizeOrder(o, 1337, cs)
	if err != nil {
		t.Error(err)
	}

	o.Direction = gctorder.Sell
	_, err = s.SizeOrder(o, 1337, cs)
	if err != nil {
		t.Error(err)
	}

	s.SellSide.MaximumSize = 1
	s.SellSide.MinimumSize = 1
	_, err = s.SizeOrder(o, 1337, cs)
	if err != nil {
		t.Error(err)
	}
}
