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
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "binance"

type fakeFund struct{}

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
	e.SetExchangeAssetCurrencySettings("", "", currency.EMPTYPAIR, &Settings{})
	if len(e.CurrencySettings) != 0 {
		t.Error("expected 0")
	}
	cs := &Settings{
		Exchange:            testExchange,
		UseRealOrders:       true,
		Pair:                currency.NewPair(currency.BTC, currency.USDT),
		Asset:               asset.Spot,
		ExchangeFee:         decimal.Zero,
		MakerFee:            decimal.Zero,
		TakerFee:            decimal.Zero,
		BuySide:             MinMax{},
		SellSide:            MinMax{},
		Leverage:            Leverage{},
		MinimumSlippageRate: decimal.Zero,
		MaximumSlippageRate: decimal.Zero,
	}
	e.SetExchangeAssetCurrencySettings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USDT), cs)
	result, err := e.GetCurrencySettings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
	if !result.UseRealOrders {
		t.Error("expected true")
	}
	e.SetExchangeAssetCurrencySettings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USDT), cs)
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

func TestSizeOrder(t *testing.T) {
	t.Parallel()
	e := Exchange{}
	_, _, err := e.sizeOfflineOrder(decimal.Zero, decimal.Zero, decimal.Zero, nil, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Error(err)
	}
	cs := &Settings{}
	f := &fill.Fill{
		ClosePrice: decimal.NewFromInt(1337),
		Amount:     decimal.NewFromInt(1),
	}
	_, _, err = e.sizeOfflineOrder(decimal.Zero, decimal.Zero, decimal.Zero, cs, f)
	if !errors.Is(err, errDataMayBeIncorrect) {
		t.Errorf("received: %v, expected: %v", err, errDataMayBeIncorrect)
	}
	var p, a decimal.Decimal
	p, a, err = e.sizeOfflineOrder(decimal.NewFromInt(10), decimal.NewFromInt(2), decimal.NewFromInt(10), cs, f)
	if err != nil {
		t.Error(err)
	}
	if !p.Equal(decimal.NewFromInt(10)) {
		t.Error("expected 10")
	}
	if !a.Equal(decimal.NewFromInt(1)) {
		t.Error("expected 1")
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
	cfg, err := exch.GetDefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	err = exch.Setup(cfg)
	if err != nil {
		t.Fatal(err)
	}
	em.Add(exch)
	bot.ExchangeManager = em
	bot.OrderManager, err = engine.SetupOrderManager(em, &engine.CommunicationManager{}, &bot.ServicesWG, false)
	if err != nil {
		t.Error(err)
	}
	err = bot.OrderManager.Start()
	if err != nil {
		t.Error(err)
	}
	e := Exchange{}
	_, err = e.placeOrder(context.Background(), decimal.NewFromInt(1), decimal.NewFromInt(1), false, true, nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	f := &fill.Fill{}
	_, err = e.placeOrder(context.Background(), decimal.NewFromInt(1), decimal.NewFromInt(1), false, true, f, bot.OrderManager)
	if !errors.Is(err, engine.ErrExchangeNameIsEmpty) {
		t.Errorf("received: %v, expected: %v", err, engine.ErrExchangeNameIsEmpty)
	}

	f.Exchange = testExchange
	_, err = e.placeOrder(context.Background(), decimal.NewFromInt(1), decimal.NewFromInt(1), false, true, f, bot.OrderManager)
	if !errors.Is(err, gctorder.ErrPairIsEmpty) {
		t.Errorf("received: %v, expected: %v", err, gctorder.ErrPairIsEmpty)
	}
	f.CurrencyPair = currency.NewPair(currency.BTC, currency.USDT)
	f.AssetType = asset.Spot
	f.Direction = gctorder.Buy
	_, err = e.placeOrder(context.Background(), decimal.NewFromInt(1), decimal.NewFromInt(1), false, true, f, bot.OrderManager)
	if err != nil {
		t.Error(err)
	}

	_, err = e.placeOrder(context.Background(), decimal.NewFromInt(1), decimal.NewFromInt(1), true, true, f, bot.OrderManager)
	if !errors.Is(err, exchange.ErrAuthenticationSupportNotEnabled) {
		t.Errorf("received: %v but expected: %v", err, exchange.ErrAuthenticationSupportNotEnabled)
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
	cfg, err := exch.GetDefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	err = exch.Setup(cfg)
	if err != nil {
		t.Fatal(err)
	}
	em.Add(exch)
	bot.ExchangeManager = em
	bot.OrderManager, err = engine.SetupOrderManager(em, &engine.CommunicationManager{}, &bot.ServicesWG, false)
	if err != nil {
		t.Error(err)
	}
	err = bot.OrderManager.Start()
	if err != nil {
		t.Error(err)
	}

	p := currency.NewPair(currency.BTC, currency.USDT)
	a := asset.Spot
	_, err = exch.FetchOrderbook(context.Background(), p, a)
	if err != nil {
		t.Fatal(err)
	}

	cs := Settings{
		Exchange:            testExchange,
		UseRealOrders:       false,
		Pair:                p,
		Asset:               a,
		ExchangeFee:         decimal.NewFromFloat(0.01),
		MakerFee:            decimal.NewFromFloat(0.01),
		TakerFee:            decimal.NewFromFloat(0.01),
		BuySide:             MinMax{},
		SellSide:            MinMax{},
		Leverage:            Leverage{},
		MinimumSlippageRate: decimal.Zero,
		MaximumSlippageRate: decimal.NewFromInt(1),
	}
	e := Exchange{
		CurrencySettings: []Settings{cs},
	}
	ev := event.Base{
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
			Asset:    "",
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
	if err != nil {
		t.Error(err)
	}
	d.Next()

	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	if err != nil {
		t.Error(err)
	}

	cs.UseRealOrders = true
	cs.CanUseExchangeLimits = true
	o.Direction = gctorder.Sell
	e.CurrencySettings = []Settings{cs}
	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	if !errors.Is(err, exchange.ErrAuthenticationSupportNotEnabled) {
		t.Errorf("received: %v but expected: %v", err, exchange.ErrAuthenticationSupportNotEnabled)
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
	cfg, err := exch.GetDefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	err = exch.Setup(cfg)
	if err != nil {
		t.Fatal(err)
	}

	em.Add(exch)
	bot.ExchangeManager = em
	bot.OrderManager, err = engine.SetupOrderManager(em, &engine.CommunicationManager{}, &bot.ServicesWG, false)
	if err != nil {
		t.Error(err)
	}
	err = bot.OrderManager.Start()
	if err != nil {
		t.Error(err)
	}
	p := currency.NewPair(currency.BTC, currency.USDT)
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

	cs := Settings{
		Exchange:      testExchange,
		UseRealOrders: false,
		Pair:          p,
		Asset:         a,
		ExchangeFee:   decimal.NewFromFloat(0.01),
		MakerFee:      decimal.NewFromFloat(0.01),
		TakerFee:      decimal.NewFromFloat(0.01),
		BuySide: MinMax{
			MaximumSize: decimal.NewFromFloat(0.01),
			MinimumSize: decimal.Zero,
		},
		SellSide: MinMax{
			MaximumSize: decimal.NewFromFloat(0.1),
			MinimumSize: decimal.Zero,
		},
		Leverage:            Leverage{},
		MinimumSlippageRate: decimal.Zero,
		MaximumSlippageRate: decimal.NewFromInt(1),
		Limits:              limits,
	}
	e := Exchange{
		CurrencySettings: []Settings{cs},
	}
	ev := event.Base{
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
			Asset:    "",
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
	if err != nil {
		t.Error(err)
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
	}
	cs.SellSide.MaximumSize = decimal.Zero
	cs.SellSide.MinimumSize = decimal.NewFromFloat(0.01)

	cs.UseRealOrders = true
	cs.CanUseExchangeLimits = true
	o.Direction = gctorder.Sell
	e.CurrencySettings = []Settings{cs}
	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	if !errors.Is(err, exchange.ErrAuthenticationSupportNotEnabled) {
		t.Errorf("received: %v but expected: %v", err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

func TestApplySlippageToPrice(t *testing.T) {
	t.Parallel()
	resp := applySlippageToPrice(gctorder.Buy, decimal.NewFromInt(1), decimal.NewFromFloat(0.9))
	if !resp.Equal(decimal.NewFromFloat(1.1)) {
		t.Errorf("received: %v, expected: %v", resp, decimal.NewFromFloat(1.1))
	}
	resp = applySlippageToPrice(gctorder.Sell, decimal.NewFromInt(1), decimal.NewFromFloat(0.9))
	if !resp.Equal(decimal.NewFromFloat(0.9)) {
		t.Errorf("received: %v, expected: %v", resp, decimal.NewFromFloat(0.9))
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
