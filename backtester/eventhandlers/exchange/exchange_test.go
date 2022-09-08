package exchange

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ftx"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "ftx"

type fakeFund struct{}

func (f *fakeFund) GetPairReader() (funding.IPairReader, error) {
	return nil, nil
}

func (f *fakeFund) GetCollateralReader() (funding.ICollateralReader, error) {
	return nil, nil
}

func (f *fakeFund) PairReleaser() (funding.IPairReleaser, error) {
	btc, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.NewFromInt(9999), decimal.NewFromInt(9999))
	if err != nil {
		return nil, err
	}
	usd, err := funding.CreateItem(testExchange, asset.Spot, currency.USD, decimal.NewFromInt(9999), decimal.NewFromInt(9999))
	if err != nil {
		return nil, err
	}
	p, err := funding.CreatePair(btc, usd)
	if err != nil {
		return nil, err
	}
	err = p.Reserve(decimal.NewFromInt(1337), gctorder.Buy)
	if err != nil {
		return nil, err
	}
	err = p.Reserve(decimal.NewFromInt(1337), gctorder.Sell)
	if err != nil {
		return nil, err
	}
	return p, nil
}
func (f *fakeFund) CollateralReleaser() (funding.ICollateralReleaser, error) {
	return nil, nil
}

func (f *fakeFund) IncreaseAvailable(decimal.Decimal, gctorder.Side) {}
func (f *fakeFund) Release(decimal.Decimal, decimal.Decimal, gctorder.Side) error {
	return nil
}

func TestReset(t *testing.T) {
	t.Parallel()
	e := Exchange{
		CurrencySettings: []Settings{},
	}
	e.Reset()
	if e.CurrencySettings != nil {
		t.Error("expected nil")
	}
}

func TestSetCurrency(t *testing.T) {
	t.Parallel()
	e := Exchange{}
	e.SetExchangeAssetCurrencySettings(asset.Empty, currency.EMPTYPAIR, &Settings{})
	if len(e.CurrencySettings) != 0 {
		t.Error("expected 0")
	}
	f := &ftx.FTX{}
	f.Name = testExchange
	cs := &Settings{
		Exchange:      f,
		UseRealOrders: true,
		Pair:          currency.NewPair(currency.BTC, currency.USD),
		Asset:         asset.Spot,
	}
	e.SetExchangeAssetCurrencySettings(asset.Spot, currency.NewPair(currency.BTC, currency.USD), cs)
	result, err := e.GetCurrencySettings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if !result.UseRealOrders {
		t.Error("expected true")
	}
	e.SetExchangeAssetCurrencySettings(asset.Spot, currency.NewPair(currency.BTC, currency.USD), cs)
	if len(e.CurrencySettings) != 1 {
		t.Error("expected 1")
	}
}

func TestEnsureOrderFitsWithinHLV(t *testing.T) {
	t.Parallel()
	adjustedPrice, adjustedAmount := ensureOrderFitsWithinHLV(decimal.NewFromInt(123), decimal.NewFromInt(1), decimal.NewFromInt(100), decimal.NewFromInt(99), decimal.NewFromInt(100))
	if !adjustedAmount.Equal(decimal.NewFromInt(1)) {
		t.Error("expected 1")
	}
	if !adjustedPrice.Equal(decimal.NewFromInt(100)) {
		t.Error("expected 100")
	}

	adjustedPrice, adjustedAmount = ensureOrderFitsWithinHLV(decimal.NewFromInt(123), decimal.NewFromInt(1), decimal.NewFromInt(100), decimal.NewFromInt(99), decimal.NewFromInt(80))
	if !adjustedAmount.Equal(decimal.NewFromFloat(0.799999992)) {
		t.Errorf("received: %v, expected: %v", adjustedAmount, decimal.NewFromFloat(0.799999992))
	}
	if !adjustedPrice.Equal(decimal.NewFromInt(100)) {
		t.Error("expected 100")
	}
}

func TestCalculateExchangeFee(t *testing.T) {
	t.Parallel()
	fee := calculateExchangeFee(decimal.NewFromInt(1), decimal.NewFromInt(1), decimal.NewFromFloat(0.1))
	if !fee.Equal(decimal.NewFromFloat(0.1)) {
		t.Error("expected 0.1")
	}
	fee = calculateExchangeFee(decimal.NewFromInt(2), decimal.NewFromFloat(1), decimal.NewFromFloat(0.005))
	if !fee.Equal(decimal.NewFromFloat(0.01)) {
		t.Error("expected 0.01")
	}
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	bot := &engine.Engine{}
	var err error
	em := engine.SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	exchB := exch.GetBase()
	exchB.States = currencystate.NewCurrencyStates()
	em.Add(exch)
	bot.ExchangeManager = em
	bot.OrderManager, err = engine.SetupOrderManager(em, &engine.CommunicationManager{}, &bot.ServicesWG, false, false, 0)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = bot.OrderManager.Start()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	e := Exchange{}
	_, err = e.placeOrder(context.Background(), decimal.NewFromInt(1), decimal.NewFromInt(1), decimal.Zero, false, true, nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	f := &fill.Fill{
		Base: &event.Base{},
	}
	_, err = e.placeOrder(context.Background(), decimal.NewFromInt(1), decimal.NewFromInt(1), decimal.Zero, false, true, f, bot.OrderManager)
	if !errors.Is(err, engine.ErrExchangeNameIsEmpty) {
		t.Errorf("received: %v, expected: %v", err, engine.ErrExchangeNameIsEmpty)
	}

	f.Exchange = testExchange
	_, err = e.placeOrder(context.Background(), decimal.NewFromInt(1), decimal.NewFromInt(1), decimal.Zero, false, true, f, bot.OrderManager)
	if !errors.Is(err, gctorder.ErrPairIsEmpty) {
		t.Errorf("received: %v, expected: %v", err, gctorder.ErrPairIsEmpty)
	}
	f.CurrencyPair = currency.NewPair(currency.BTC, currency.USD)
	f.AssetType = asset.Spot
	f.Direction = gctorder.Buy
	_, err = e.placeOrder(context.Background(), decimal.NewFromInt(1), decimal.NewFromInt(1), decimal.Zero, false, true, f, bot.OrderManager)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	_, err = e.placeOrder(context.Background(), decimal.NewFromInt(1), decimal.NewFromInt(1), decimal.Zero, true, true, f, bot.OrderManager)
	if !errors.Is(err, exchange.ErrCredentialsAreEmpty) {
		t.Errorf("received: %v but expected: %v", err, exchange.ErrCredentialsAreEmpty)
	}
}

func TestExecuteOrder(t *testing.T) {
	t.Parallel()
	bot := &engine.Engine{}
	var err error
	em := engine.SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	exchB := exch.GetBase()
	exchB.States = currencystate.NewCurrencyStates()
	em.Add(exch)
	bot.ExchangeManager = em
	bot.OrderManager, err = engine.SetupOrderManager(em, &engine.CommunicationManager{}, &bot.ServicesWG, false, false, 0)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = bot.OrderManager.Start()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	p := currency.NewPair(currency.BTC, currency.USD)
	a := asset.Spot
	_, err = exch.FetchOrderbook(context.Background(), p, a)
	if err != nil {
		t.Fatal(err)
	}
	f := &ftx.FTX{}
	f.Name = testExchange
	cs := Settings{
		Exchange:            f,
		UseRealOrders:       false,
		Pair:                p,
		Asset:               a,
		MakerFee:            decimal.NewFromFloat(0.01),
		TakerFee:            decimal.NewFromFloat(0.01),
		MaximumSlippageRate: decimal.NewFromInt(1),
	}
	e := Exchange{}
	ev := &event.Base{
		Exchange:     testExchange,
		Time:         time.Now(),
		Interval:     gctkline.FifteenMin,
		CurrencyPair: p,
		AssetType:    a,
	}
	o := &order.Order{
		Base:           ev,
		Direction:      gctorder.Buy,
		Amount:         decimal.NewFromInt(10),
		AllocatedFunds: decimal.NewFromInt(1337),
		ClosePrice:     decimal.NewFromInt(1),
	}

	item := gctkline.Item{
		Exchange: testExchange,
		Pair:     p,
		Asset:    a,
		Interval: 0,
		Candles: []gctkline.Candle{
			{
				Close:  1,
				High:   1,
				Low:    1,
				Volume: 1,
			},
		},
	}
	d := &kline.DataFromKline{
		Item: item,
	}
	err = d.Load()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	d.Next()
	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	if !errors.Is(err, errNoCurrencySettingsFound) {
		t.Error(err)
	}

	cs.UseRealOrders = true
	cs.CanUseExchangeLimits = true
	o.Direction = gctorder.Sell
	e.CurrencySettings = []Settings{cs}
	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	if !errors.Is(err, exchange.ErrCredentialsAreEmpty) {
		t.Errorf("received: %v but expected: %v", err, exchange.ErrCredentialsAreEmpty)
	}
}

func TestExecuteOrderBuySellSizeLimit(t *testing.T) {
	t.Parallel()
	bot := &engine.Engine{}
	var err error
	em := engine.SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	exchB := exch.GetBase()
	exchB.States = currencystate.NewCurrencyStates()
	em.Add(exch)
	bot.ExchangeManager = em
	bot.OrderManager, err = engine.SetupOrderManager(em, &engine.CommunicationManager{}, &bot.ServicesWG, false, false, 0)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = bot.OrderManager.Start()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	p := currency.NewPair(currency.BTC, currency.USD)
	a := asset.Spot
	_, err = exch.FetchOrderbook(context.Background(), p, a)
	if err != nil {
		t.Fatal(err)
	}

	err = exch.UpdateOrderExecutionLimits(context.Background(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	limits, err := exch.GetOrderExecutionLimits(a, p)
	if err != nil {
		t.Fatal(err)
	}
	f := &ftx.FTX{}
	f.Name = testExchange
	cs := Settings{
		Exchange:      f,
		UseRealOrders: false,
		Pair:          p,
		Asset:         a,
		MakerFee:      decimal.NewFromFloat(0.01),
		TakerFee:      decimal.NewFromFloat(0.01),
		BuySide: MinMax{
			MaximumSize: decimal.NewFromFloat(0.01),
		},
		SellSide: MinMax{
			MaximumSize: decimal.NewFromFloat(0.1),
		},
		MaximumSlippageRate: decimal.NewFromInt(1),
		Limits:              limits,
	}
	e := Exchange{
		CurrencySettings: []Settings{cs},
	}
	ev := &event.Base{
		Exchange:     testExchange,
		Time:         time.Now(),
		Interval:     gctkline.FifteenMin,
		CurrencyPair: p,
		AssetType:    a,
	}
	o := &order.Order{
		Base:           ev,
		Direction:      gctorder.Buy,
		Amount:         decimal.NewFromInt(10),
		AllocatedFunds: decimal.NewFromInt(1337),
	}

	d := &kline.DataFromKline{
		Item: gctkline.Item{
			Exchange: "",
			Pair:     currency.EMPTYPAIR,
			Asset:    asset.Empty,
			Interval: 0,
			Candles: []gctkline.Candle{
				{
					Close:  1,
					High:   1,
					Low:    1,
					Volume: 1,
				},
			},
		},
	}
	err = d.Load()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	d.Next()
	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	if !errors.Is(err, errExceededPortfolioLimit) {
		t.Errorf("received %v expected %v", err, errExceededPortfolioLimit)
	}
	o = &order.Order{
		Base:           ev,
		Direction:      gctorder.Buy,
		Amount:         decimal.NewFromInt(10),
		AllocatedFunds: decimal.NewFromInt(1337),
	}
	cs.BuySide.MaximumSize = decimal.Zero
	cs.BuySide.MinimumSize = decimal.NewFromFloat(0.01)
	e.CurrencySettings = []Settings{cs}
	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	if err != nil && !strings.Contains(err.Error(), "exceed minimum size") {
		t.Error(err)
	}
	if err != nil {
		t.Error("limitReducedAmount adjusted to 0.99999999, direction BUY, should fall in  buyside {MinimumSize:0.01 MaximumSize:0 MaximumTotal:0}")
	}
	o = &order.Order{
		Base:           ev,
		Direction:      gctorder.Sell,
		Amount:         decimal.NewFromInt(10),
		AllocatedFunds: decimal.NewFromInt(1337),
	}
	cs.SellSide.MaximumSize = decimal.Zero
	cs.SellSide.MinimumSize = decimal.NewFromFloat(0.01)
	e.CurrencySettings = []Settings{cs}
	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	if err != nil && !strings.Contains(err.Error(), "exceed minimum size") {
		t.Error(err)
	}
	if err != nil {
		t.Error("limitReducedAmount adjust to 0.99999999, should fall in sell size {MinimumSize:0.01 MaximumSize:0 MaximumTotal:0}")
	}

	o = &order.Order{
		Base:           ev,
		Direction:      gctorder.Sell,
		Amount:         decimal.NewFromFloat(0.5),
		AllocatedFunds: decimal.NewFromInt(1337),
	}
	cs.SellSide.MaximumSize = decimal.Zero
	cs.SellSide.MinimumSize = decimal.NewFromInt(1)
	e.CurrencySettings = []Settings{cs}
	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	if !errors.Is(err, errExceededPortfolioLimit) {
		t.Errorf("received %v expected %v", err, errExceededPortfolioLimit)
	}

	o = &order.Order{
		Base:           ev,
		Direction:      gctorder.Sell,
		Amount:         decimal.NewFromFloat(0.02),
		AllocatedFunds: decimal.NewFromFloat(0.01337),
		ClosePrice:     decimal.NewFromFloat(1337),
	}
	cs.SellSide.MaximumSize = decimal.Zero
	cs.SellSide.MinimumSize = decimal.NewFromFloat(0.01)

	cs.UseRealOrders = true
	cs.CanUseExchangeLimits = true
	o.Direction = gctorder.Sell

	e.CurrencySettings = []Settings{cs}
	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	if !errors.Is(err, exchange.ErrCredentialsAreEmpty) {
		t.Errorf("received: %v but expected: %v", err, exchange.ErrCredentialsAreEmpty)
	}
}

func TestApplySlippageToPrice(t *testing.T) {
	t.Parallel()
	resp, err := applySlippageToPrice(gctorder.Buy, decimal.NewFromInt(1), decimal.NewFromFloat(0.9))
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !resp.Equal(decimal.NewFromFloat(1.1)) {
		t.Errorf("received: %v, expected: %v", resp, decimal.NewFromFloat(1.1))
	}

	resp, err = applySlippageToPrice(gctorder.Sell, decimal.NewFromInt(1), decimal.NewFromFloat(0.9))
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !resp.Equal(decimal.NewFromFloat(0.9)) {
		t.Errorf("received: %v, expected: %v", resp, decimal.NewFromFloat(0.9))
	}

	resp, err = applySlippageToPrice(gctorder.Sell, decimal.NewFromInt(1), decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !resp.Equal(decimal.NewFromFloat(1)) {
		t.Errorf("received: %v, expected: %v", resp, decimal.NewFromFloat(1))
	}

	_, err = applySlippageToPrice(gctorder.UnknownSide, decimal.NewFromInt(1), decimal.NewFromFloat(0.9))
	if !errors.Is(err, gctorder.ErrSideIsInvalid) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

func TestReduceAmountToFitPortfolioLimit(t *testing.T) {
	t.Parallel()
	initialPrice := decimal.NewFromInt(100)
	initialAmount := decimal.NewFromInt(10).Div(initialPrice)
	portfolioAdjustedTotal := initialAmount.Mul(initialPrice)
	adjustedPrice := decimal.NewFromInt(1000)
	amount := decimal.NewFromInt(2)
	finalAmount := reduceAmountToFitPortfolioLimit(adjustedPrice, amount, portfolioAdjustedTotal, gctorder.Buy)
	if !finalAmount.Mul(adjustedPrice).Equal(portfolioAdjustedTotal) {
		t.Errorf("expected value %v to match portfolio total %v", finalAmount.Mul(adjustedPrice), portfolioAdjustedTotal)
	}
	finalAmount = reduceAmountToFitPortfolioLimit(adjustedPrice, decimal.NewFromInt(133333333337), portfolioAdjustedTotal, gctorder.Sell)
	if finalAmount != portfolioAdjustedTotal {
		t.Errorf("expected value %v to match portfolio total %v", finalAmount, portfolioAdjustedTotal)
	}
	finalAmount = reduceAmountToFitPortfolioLimit(adjustedPrice, decimal.NewFromInt(1), portfolioAdjustedTotal, gctorder.Sell)
	if !finalAmount.Equal(decimal.NewFromInt(1)) {
		t.Errorf("expected value %v to match portfolio total %v", finalAmount, portfolioAdjustedTotal)
	}
}

func TestVerifyOrderWithinLimits(t *testing.T) {
	t.Parallel()
	err := verifyOrderWithinLimits(nil, decimal.Zero, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received %v expected %v", err, common.ErrNilEvent)
	}

	err = verifyOrderWithinLimits(&fill.Fill{}, decimal.Zero, nil)
	if !errors.Is(err, errNilCurrencySettings) {
		t.Errorf("received %v expected %v", err, errNilCurrencySettings)
	}

	err = verifyOrderWithinLimits(&fill.Fill{}, decimal.Zero, &Settings{})
	if !errors.Is(err, errInvalidDirection) {
		t.Errorf("received %v expected %v", err, errInvalidDirection)
	}
	f := &fill.Fill{
		Direction: gctorder.Buy,
	}
	err = verifyOrderWithinLimits(f, decimal.Zero, &Settings{})
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	s := &Settings{
		BuySide: MinMax{
			MinimumSize: decimal.NewFromInt(1),
			MaximumSize: decimal.NewFromInt(1),
		},
	}
	f.Base = &event.Base{}
	err = verifyOrderWithinLimits(f, decimal.NewFromFloat(0.5), s)
	if !errors.Is(err, errExceededPortfolioLimit) {
		t.Errorf("received %v expected %v", err, errExceededPortfolioLimit)
	}
	f.Direction = gctorder.Buy
	err = verifyOrderWithinLimits(f, decimal.NewFromInt(2), s)
	if !errors.Is(err, errExceededPortfolioLimit) {
		t.Errorf("received %v expected %v", err, errExceededPortfolioLimit)
	}

	f.Direction = gctorder.Sell
	s.SellSide = MinMax{
		MinimumSize: decimal.NewFromInt(1),
		MaximumSize: decimal.NewFromInt(1),
	}
	err = verifyOrderWithinLimits(f, decimal.NewFromFloat(0.5), s)
	if !errors.Is(err, errExceededPortfolioLimit) {
		t.Errorf("received %v expected %v", err, errExceededPortfolioLimit)
	}
	f.Direction = gctorder.Sell
	err = verifyOrderWithinLimits(f, decimal.NewFromInt(2), s)
	if !errors.Is(err, errExceededPortfolioLimit) {
		t.Errorf("received %v expected %v", err, errExceededPortfolioLimit)
	}
}

func TestAllocateFundsPostOrder(t *testing.T) {
	t.Parallel()
	expectedError := common.ErrNilEvent
	err := allocateFundsPostOrder(nil, nil, nil, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	expectedError = gctcommon.ErrNilPointer
	f := &fill.Fill{
		Base: &event.Base{
			AssetType: asset.Spot,
		},
		Direction: gctorder.Buy,
	}
	err = allocateFundsPostOrder(f, nil, nil, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	expectedError = nil
	one := decimal.NewFromInt(1)
	item, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	item2, err := funding.CreateItem(testExchange, asset.Spot, currency.USD, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	err = item.Reserve(one)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	err = item2.Reserve(one)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	fundPair, err := funding.CreatePair(item, item2)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	f.Order = &gctorder.Detail{}
	err = allocateFundsPostOrder(f, fundPair, nil, one, one, one, one, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	f.SetDirection(gctorder.Sell)
	err = allocateFundsPostOrder(f, fundPair, nil, one, one, one, one, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	expectedError = gctorder.ErrSubmissionIsNil
	orderError := gctorder.ErrSubmissionIsNil
	err = allocateFundsPostOrder(f, fundPair, orderError, one, one, one, one, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	f.AssetType = asset.Futures
	f.SetDirection(gctorder.Short)
	expectedError = nil
	item3, err := funding.CreateItem(testExchange, asset.Futures, currency.BTC, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	item4, err := funding.CreateItem(testExchange, asset.Futures, currency.USD, decimal.NewFromInt(1337), decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	err = item3.Reserve(one)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	err = item4.Reserve(one)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	collateralPair, err := funding.CreateCollateral(item, item2)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	expectedError = gctorder.ErrSubmissionIsNil
	err = allocateFundsPostOrder(f, collateralPair, orderError, one, one, one, one, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	expectedError = nil
	err = allocateFundsPostOrder(f, collateralPair, nil, one, one, one, one, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	expectedError = gctorder.ErrSubmissionIsNil
	f.SetDirection(gctorder.Long)
	err = allocateFundsPostOrder(f, collateralPair, orderError, one, one, one, one, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	expectedError = nil
	err = allocateFundsPostOrder(f, collateralPair, nil, one, one, one, one, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	f.AssetType = asset.Margin
	expectedError = common.ErrInvalidDataType
	err = allocateFundsPostOrder(f, collateralPair, nil, one, one, one, one, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
}
