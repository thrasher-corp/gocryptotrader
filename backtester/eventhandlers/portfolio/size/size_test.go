package size

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
	amountWithoutFee, _, err := sizer.calculateBuySize(price, availableFunds, feeRate, buyLimit, globalMinMax)
	assert.NoError(t, err)

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
	amount, _, err := sizer.calculateBuySize(price, availableFunds, feeRate, buyLimit, globalMinMax)
	assert.NoError(t, err)

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
	_, _, err := sizer.calculateBuySize(price, availableFunds, feeRate, buyLimit, globalMinMax)
	assert.ErrorIs(t, err, errLessThanMinimum)
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
	amount, _, err := sizer.calculateBuySize(price, availableFunds, feeRate, buyLimit, globalMinMax)
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
	amount, _, err := sizer.calculateSellSize(price, availableFunds, feeRate, sellLimit, globalMinMax)
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
	_, _, err := sizer.calculateBuySize(price, availableFunds, feeRate, buyLimit, globalMinMax)
	assert.ErrorIs(t, err, errNoFunds)
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
	_, _, err := sizer.calculateSellSize(price, availableFunds, feeRate, sellLimit, globalMinMax)
	assert.ErrorIs(t, err, errNoFunds)

	availableFunds = decimal.NewFromInt(1337)
	_, _, err = sizer.calculateSellSize(price, availableFunds, feeRate, sellLimit, globalMinMax)
	assert.ErrorIs(t, err, errLessThanMinimum)

	price = decimal.NewFromInt(12)
	availableFunds = decimal.NewFromInt(1339)
	amount, fee, err := sizer.calculateSellSize(price, availableFunds, feeRate, sellLimit, globalMinMax)
	assert.NoError(t, err)

	if !amount.Equal(sellLimit) {
		t.Errorf("received '%v' expected '%v'", amount, sellLimit)
	}
	if !amount.Mul(price).Mul(feeRate).Equal(fee) {
		t.Errorf("received '%v' expected '%v'", amount.Mul(price).Mul(feeRate), fee)
	}
}

func TestSizeOrder(t *testing.T) {
	t.Parallel()
	s := Size{}
	_, _, err := s.SizeOrder(nil, decimal.Zero, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	o := &order.Order{
		Base: &event.Base{
			Offset:         1,
			Exchange:       "binance",
			Time:           time.Now(),
			CurrencyPair:   currency.NewBTCUSDT(),
			UnderlyingPair: currency.NewBTCUSDT(),
			AssetType:      asset.Spot,
		},
	}
	cs := &exchange.Settings{}
	_, _, err = s.SizeOrder(o, decimal.Zero, cs)
	assert.ErrorIs(t, err, errNoFunds)

	_, _, err = s.SizeOrder(o, decimal.NewFromInt(1337), cs)
	assert.ErrorIs(t, err, errCannotAllocate)

	o.Direction = gctorder.Buy
	_, _, err = s.SizeOrder(o, decimal.NewFromInt(1337), cs)
	assert.ErrorIs(t, err, errCannotAllocate)

	o.ClosePrice = decimal.NewFromInt(1)
	s.BuySide.MaximumSize = decimal.NewFromInt(1)
	s.BuySide.MinimumSize = decimal.NewFromInt(1)
	_, _, err = s.SizeOrder(o, decimal.NewFromInt(1337), cs)
	assert.NoError(t, err)

	o.Amount = decimal.NewFromInt(1)
	o.Direction = gctorder.Sell
	_, _, err = s.SizeOrder(o, decimal.NewFromInt(1337), cs)
	assert.NoError(t, err)

	s.SellSide.MaximumSize = decimal.NewFromInt(1)
	s.SellSide.MinimumSize = decimal.NewFromInt(1)
	_, _, err = s.SizeOrder(o, decimal.NewFromInt(1337), cs)
	assert.NoError(t, err)

	o.Direction = gctorder.ClosePosition
	_, _, err = s.SizeOrder(o, decimal.NewFromInt(1337), cs)
	assert.NoError(t, err)

	// spot futures sizing
	o.FillDependentEvent = &signal.Signal{
		Base:               o.Base,
		MatchesOrderAmount: true,
		ClosePrice:         decimal.NewFromInt(1337),
	}
	exch := binance.Exchange{}
	// TODO adjust when Binance futures wrappers are implemented
	cs.Exchange = &exch
	_, _, err = s.SizeOrder(o, decimal.NewFromInt(1337), cs)
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)

	o.ClosePrice = decimal.NewFromInt(1000000000)
	o.Amount = decimal.NewFromInt(1000000000)
	_, _, err = s.SizeOrder(o, decimal.NewFromInt(1337), cs)
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)
}
