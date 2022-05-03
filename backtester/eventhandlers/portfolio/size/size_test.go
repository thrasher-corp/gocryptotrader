package size

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
)

func TestSizingAccuracy(t *testing.T) {
	t.Parallel()
	globalMinMax := exchange.MinMax{
		MaximumSize:  decimal.NewFromInt(1),
		MaximumTotal: decimal.NewFromInt(10),
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := decimal.NewFromInt(10)
	availableFunds := decimal.NewFromInt(11)
	feeRate := decimal.NewFromFloat(0.02)
	buyLimit := decimal.NewFromInt(1)
	amountWithoutFee, err := sizer.calculateBuySize(price, availableFunds, feeRate, buyLimit, globalMinMax)
	if err != nil {
		t.Error(err)
	}
	totalWithFee := (price.Mul(amountWithoutFee)).Add(globalMinMax.MaximumTotal.Mul(feeRate))
	if !totalWithFee.Equal(globalMinMax.MaximumTotal) {
		t.Errorf("expected %v received %v", globalMinMax.MaximumTotal, totalWithFee)
	}
}

func TestSizingOverMaxSize(t *testing.T) {
	t.Parallel()
	globalMinMax := exchange.MinMax{
		MaximumSize:  decimal.NewFromFloat(0.5),
		MaximumTotal: decimal.NewFromInt(1337),
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := decimal.NewFromInt(1338)
	availableFunds := decimal.NewFromInt(1338)
	feeRate := decimal.NewFromFloat(0.02)
	buyLimit := decimal.NewFromInt(1)
	amount, err := sizer.calculateBuySize(price, availableFunds, feeRate, buyLimit, globalMinMax)
	if err != nil {
		t.Error(err)
	}
	if amount.GreaterThan(globalMinMax.MaximumSize) {
		t.Error("greater than max")
	}
}

func TestSizingUnderMinSize(t *testing.T) {
	t.Parallel()
	globalMinMax := exchange.MinMax{
		MinimumSize:  decimal.NewFromInt(1),
		MaximumSize:  decimal.NewFromInt(2),
		MaximumTotal: decimal.NewFromInt(1337),
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := decimal.NewFromInt(1338)
	availableFunds := decimal.NewFromInt(1338)
	feeRate := decimal.NewFromFloat(0.02)
	buyLimit := decimal.NewFromInt(1)
	_, err := sizer.calculateBuySize(price, availableFunds, feeRate, buyLimit, globalMinMax)
	if !errors.Is(err, errLessThanMinimum) {
		t.Errorf("received: %v, expected: %v", err, errLessThanMinimum)
	}
}

func TestMaximumBuySizeEqualZero(t *testing.T) {
	t.Parallel()
	globalMinMax := exchange.MinMax{
		MinimumSize:  decimal.NewFromInt(1),
		MaximumTotal: decimal.NewFromInt(1437),
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := decimal.NewFromInt(1338)
	availableFunds := decimal.NewFromInt(13380)
	feeRate := decimal.NewFromFloat(0.02)
	buyLimit := decimal.NewFromInt(1)
	amount, err := sizer.calculateBuySize(price, availableFunds, feeRate, buyLimit, globalMinMax)
	if amount != buyLimit || err != nil {
		t.Errorf("expected: %v, received %v, err: %+v", buyLimit, amount, err)
	}
}
func TestMaximumSellSizeEqualZero(t *testing.T) {
	t.Parallel()
	globalMinMax := exchange.MinMax{
		MinimumSize:  decimal.NewFromInt(1),
		MaximumTotal: decimal.NewFromInt(1437),
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := decimal.NewFromInt(1338)
	availableFunds := decimal.NewFromInt(13380)
	feeRate := decimal.NewFromFloat(0.02)
	sellLimit := decimal.NewFromInt(1)
	amount, err := sizer.calculateSellSize(price, availableFunds, feeRate, sellLimit, globalMinMax)
	if amount != sellLimit || err != nil {
		t.Errorf("expected: %v, received %v, err: %+v", sellLimit, amount, err)
	}
}

func TestSizingErrors(t *testing.T) {
	t.Parallel()
	globalMinMax := exchange.MinMax{
		MinimumSize:  decimal.NewFromInt(1),
		MaximumSize:  decimal.NewFromInt(2),
		MaximumTotal: decimal.NewFromInt(1337),
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := decimal.NewFromInt(1338)
	availableFunds := decimal.Zero
	feeRate := decimal.NewFromFloat(0.02)
	buyLimit := decimal.NewFromInt(1)
	_, err := sizer.calculateBuySize(price, availableFunds, feeRate, buyLimit, globalMinMax)
	if !errors.Is(err, errNoFunds) {
		t.Errorf("received: %v, expected: %v", err, errNoFunds)
	}
}

func TestCalculateSellSize(t *testing.T) {
	t.Parallel()
	globalMinMax := exchange.MinMax{
		MinimumSize:  decimal.NewFromInt(1),
		MaximumSize:  decimal.NewFromInt(2),
		MaximumTotal: decimal.NewFromInt(1337),
	}
	sizer := Size{
		BuySide:  globalMinMax,
		SellSide: globalMinMax,
	}
	price := decimal.NewFromInt(1338)
	availableFunds := decimal.Zero
	feeRate := decimal.NewFromFloat(0.02)
	sellLimit := decimal.NewFromInt(1)
	_, err := sizer.calculateSellSize(price, availableFunds, feeRate, sellLimit, globalMinMax)
	if !errors.Is(err, errNoFunds) {
		t.Errorf("received: %v, expected: %v", err, errNoFunds)
	}
	availableFunds = decimal.NewFromInt(1337)
	_, err = sizer.calculateSellSize(price, availableFunds, feeRate, sellLimit, globalMinMax)
	if !errors.Is(err, errLessThanMinimum) {
		t.Errorf("received: %v, expected: %v", err, errLessThanMinimum)
	}
	price = decimal.NewFromInt(12)
	availableFunds = decimal.NewFromInt(1339)
	_, err = sizer.calculateSellSize(price, availableFunds, feeRate, sellLimit, globalMinMax)
	if err != nil {
		t.Error(err)
	}
}

/*
func TestSizeOrder(t *testing.T) {
	t.Parallel()
	s := Size{}
	_, err := s.SizeOrder(nil, decimal.Zero, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Error(err)
	}
	o := &order.Order{}
	cs := &exchange.Settings{}
	_, err = s.SizeOrder(o, decimal.Zero, cs)
	if !errors.Is(err, errNoFunds) {
		t.Errorf("received: %v, expected: %v", err, errNoFunds)
	}

	_, err = s.SizeOrder(o, decimal.NewFromInt(1337), cs)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received: %v, expected: %v", err, errCannotAllocate)
	}
	o.Direction = gctorder.Buy
	_, err = s.SizeOrder(o, decimal.NewFromInt(1337), cs)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received: %v, expected: %v", err, errCannotAllocate)
	}

	o.ClosePrice = decimal.NewFromInt(1)
	s.BuySide.MaximumSize = decimal.NewFromInt(1)
	s.BuySide.MinimumSize = decimal.NewFromInt(1)
	_, err = s.SizeOrder(o, decimal.NewFromInt(1337), cs)
	if err != nil {
		t.Error(err)
	}

	o.Amount = decimal.NewFromInt(1)
	o.Direction = gctorder.Sell
	_, err = s.SizeOrder(o, decimal.NewFromInt(1337), cs)
	if err != nil {
		t.Error(err)
	}

	s.SellSide.MaximumSize = decimal.NewFromInt(1)
	s.SellSide.MinimumSize = decimal.NewFromInt(1)
	_, err = s.SizeOrder(o, decimal.NewFromInt(1337), cs)
	if err != nil {
		t.Error(err)
	}

	o.Direction = common.ClosePosition
	_, err = s.SizeOrder(o, decimal.NewFromInt(1337), cs)
	if err != nil {
		t.Error(err)
	}
}


*/
func TestFuckYou(t *testing.T) {
	t.Parallel()
	weight := 0.95
	//availableFunds := 438.0
	spotPrice := 39122.0
	futurePrice := 43195.0

	for i := 1.0; i < 100000.0; i++ {
		step1 := (i * weight) / futurePrice
		futuresNotionalPosition := step1 * futurePrice
		spotNotionalPosition := step1 * spotPrice
		spotCollateralSize := spotNotionalPosition * weight
		extraFundingRequired := futuresNotionalPosition - spotCollateralSize
		excess := i - spotNotionalPosition
		leftover := excess - extraFundingRequired
		t.Log(leftover)
	}

	//t.Logf("step1 %v\n futuresNotionalPosition %v\n spotNotionalPosition %v\n spotCollateralSize %v\n excess %v\n extraFundingRequired %v\n leftover %v\n", step1, futuresNotionalPosition, spotNotionalPosition, spotCollateralSize, excess, extraFundingRequired, leftover)
}
