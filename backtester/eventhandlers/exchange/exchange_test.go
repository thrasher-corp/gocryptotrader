package exchange

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
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
	e.SetExchangeAssetCurrencySettings("", "", currency.Pair{}, &Settings{})
	if len(e.CurrencySettings) != 0 {
		t.Error("expected 0")
	}
	cs := &Settings{
		ExchangeName:        testExchange,
		UseRealOrders:       false,
		InitialFunds:        1337,
		CurrencyPair:        currency.NewPair(currency.BTC, currency.USDT),
		AssetType:           asset.Spot,
		ExchangeFee:         0,
		MakerFee:            0,
		TakerFee:            0,
		BuySide:             config.MinMax{},
		SellSide:            config.MinMax{},
		Leverage:            config.Leverage{},
		MinimumSlippageRate: 0,
		MaximumSlippageRate: 0,
	}
	e.SetExchangeAssetCurrencySettings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USDT), cs)
	result, err := e.GetCurrencySettings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
	if result.InitialFunds != 1337 {
		t.Errorf("expected 1337, received %v", result.InitialFunds)
	}

	e.SetExchangeAssetCurrencySettings(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USDT), cs)
	if len(e.CurrencySettings) != 1 {
		t.Error("expected 1")
	}
}

func TestEnsureOrderFitsWithinHLV(t *testing.T) {
	t.Parallel()
	adjustedPrice, adjustedAmount := ensureOrderFitsWithinHLV(123, 1, 100, 99, 100)
	if adjustedAmount != 1 {
		t.Error("expected 1")
	}
	if adjustedPrice != 100 {
		t.Error("expected 100")
	}

	adjustedPrice, adjustedAmount = ensureOrderFitsWithinHLV(123, 1, 100, 99, 80)
	if adjustedAmount != 0.7999999919999999 {
		t.Errorf("expected %v received %v", 0.7999999919999999, adjustedAmount)
	}
	if adjustedPrice != 100 {
		t.Error("expected 100")
	}
}

func TestCalculateExchangeFee(t *testing.T) {
	t.Parallel()
	fee := calculateExchangeFee(1, 1, 0.1)
	if fee != 0.1 {
		t.Error("expected 0.1")
	}
	fee = calculateExchangeFee(2, 1, 0.005)
	if fee != 0.01 {
		t.Error("expected 0.01")
	}
}

func TestSizeOrder(t *testing.T) {
	t.Parallel()
	e := Exchange{}
	_, _, err := e.sizeOfflineOrder(0, 0, 0, nil, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Error(err)
	}
	cs := &Settings{}
	f := &fill.Fill{
		ClosePrice: 1337,
		Amount:     1,
	}
	_, _, err = e.sizeOfflineOrder(0, 0, 0, cs, f)
	if !errors.Is(err, errDataMayBeIncorrect) {
		t.Errorf("expected: %v, received %v", errDataMayBeIncorrect, err)
	}
	var p, a float64
	p, a, err = e.sizeOfflineOrder(10, 2, 10, cs, f)
	if err != nil {
		t.Error(err)
	}
	if p != 10 {
		t.Error("expected 10")
	}
	if a != 1 {
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
	_, err = e.placeOrder(1, 1, false, true, nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, received %v", common.ErrNilEvent, err)
	}
	f := &fill.Fill{}
	_, err = e.placeOrder(1, 1, false, true, f, bot)
	if err != nil && err.Error() != "order exchange name must be specified" {
		t.Error(err)
	}

	f.Exchange = testExchange
	_, err = e.placeOrder(1, 1, false, true, f, bot)
	if !errors.Is(err, gctorder.ErrPairIsEmpty) {
		t.Errorf("expected: %v, received %v", gctorder.ErrPairIsEmpty, err)
	}
	f.CurrencyPair = currency.NewPair(currency.BTC, currency.USDT)
	f.AssetType = asset.Spot
	f.Direction = gctorder.Buy
	_, err = e.placeOrder(1, 1, false, true, f, bot)
	if err != nil {
		t.Error(err)
	}

	_, err = e.placeOrder(1, 1, true, true, f, bot)
	if err != nil && !strings.Contains(err.Error(), "unset/default API keys") {
		t.Error(err)
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
	_, err = exch.FetchOrderbook(p, a)
	if err != nil {
		t.Fatal(err)
	}

	cs := Settings{
		ExchangeName:        testExchange,
		UseRealOrders:       false,
		InitialFunds:        1337,
		CurrencyPair:        p,
		AssetType:           a,
		ExchangeFee:         0.01,
		MakerFee:            0.01,
		TakerFee:            0.01,
		BuySide:             config.MinMax{},
		SellSide:            config.MinMax{},
		Leverage:            config.Leverage{},
		MinimumSlippageRate: 0,
		MaximumSlippageRate: 1,
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
		Base:      ev,
		Direction: gctorder.Buy,
		Amount:    10,
		Funds:     1337,
	}

	d := &kline.DataFromKline{
		Item: gctkline.Item{
			Exchange: "",
			Pair:     currency.Pair{},
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
	_, err = e.ExecuteOrder(o, d, bot)
	if err != nil {
		t.Error(err)
	}

	cs.UseRealOrders = true
	cs.CanUseExchangeLimits = true
	o.Direction = gctorder.Sell
	e.CurrencySettings = []Settings{cs}
	_, err = e.ExecuteOrder(o, d, bot)
	if err != nil && !strings.Contains(err.Error(), "unset/default API keys") {
		t.Error(err)
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
	_, err = exch.FetchOrderbook(p, a)
	if err != nil {
		t.Fatal(err)
	}

	err = exch.UpdateOrderExecutionLimits(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	limits, err := exch.GetOrderExecutionLimits(a, p)
	if err != nil {
		t.Fatal(err)
	}

	cs := Settings{
		ExchangeName:  testExchange,
		UseRealOrders: false,
		InitialFunds:  1337,
		CurrencyPair:  p,
		AssetType:     a,
		ExchangeFee:   0.01,
		MakerFee:      0.01,
		TakerFee:      0.01,
		BuySide: config.MinMax{
			MaximumSize: 0.01,
			MinimumSize: 0,
		},
		SellSide: config.MinMax{
			MaximumSize: 0.1,
			MinimumSize: 0,
		},
		Leverage:            config.Leverage{},
		MinimumSlippageRate: 0,
		MaximumSlippageRate: 1,
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
		Base:      ev,
		Direction: gctorder.Buy,
		Amount:    10,
		Funds:     1337,
	}

	d := &kline.DataFromKline{
		Item: gctkline.Item{
			Exchange: "",
			Pair:     currency.Pair{},
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
	_, err = e.ExecuteOrder(o, d, bot)
	if !errors.Is(err, errExceededPortfolioLimit) {
		t.Errorf("received %v expected %v", err, errExceededPortfolioLimit)
	}
	o = &order.Order{
		Base:      ev,
		Direction: gctorder.Buy,
		Amount:    10,
		Funds:     1337,
	}
	cs.BuySide.MaximumSize = 0
	cs.BuySide.MinimumSize = 0.01
	e.CurrencySettings = []Settings{cs}
	_, err = e.ExecuteOrder(o, d, bot)
	if err != nil && !strings.Contains(err.Error(), "exceed minimum size") {
		t.Error(err)
	}
	if err != nil {
		t.Error("limitReducedAmount adjusted to 0.99999999, direction BUY, should fall in  buyside {MinimumSize:0.01 MaximumSize:0 MaximumTotal:0}")
	}
	o = &order.Order{
		Base:      ev,
		Direction: gctorder.Sell,
		Amount:    10,
		Funds:     1337,
	}
	cs.SellSide.MaximumSize = 0
	cs.SellSide.MinimumSize = 0.01
	e.CurrencySettings = []Settings{cs}
	_, err = e.ExecuteOrder(o, d, bot)
	if err != nil && !strings.Contains(err.Error(), "exceed minimum size") {
		t.Error(err)
	}
	if err != nil {
		t.Error("limitReducedAmount adjust to 0.99999999, should fall in sell size {MinimumSize:0.01 MaximumSize:0 MaximumTotal:0}")
	}

	o = &order.Order{
		Base:      ev,
		Direction: gctorder.Sell,
		Amount:    0.5,
		Funds:     1337,
	}
	cs.SellSide.MaximumSize = 0
	cs.SellSide.MinimumSize = 1
	e.CurrencySettings = []Settings{cs}
	_, err = e.ExecuteOrder(o, d, bot)
	if !errors.Is(err, errExceededPortfolioLimit) {
		t.Errorf("received %v expected %v", err, errExceededPortfolioLimit)
	}

	o = &order.Order{
		Base:      ev,
		Direction: gctorder.Sell,
		Amount:    0.02,
		Funds:     0.01337,
	}
	cs.SellSide.MaximumSize = 0
	cs.SellSide.MinimumSize = 0.01

	cs.UseRealOrders = true
	cs.CanUseExchangeLimits = true
	o.Direction = gctorder.Sell
	e.CurrencySettings = []Settings{cs}
	_, err = e.ExecuteOrder(o, d, bot)
	if !errors.Is(err, exchange.ErrAuthenticatedRequestWithoutCredentialsSet) {
		t.Errorf("received %v expected %v", err, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
}

func TestApplySlippageToPrice(t *testing.T) {
	t.Parallel()
	resp := applySlippageToPrice(gctorder.Buy, 1, 0.9)
	if resp != 1.1 {
		t.Errorf("expected 1.1, received %v", resp)
	}
	resp = applySlippageToPrice(gctorder.Sell, 1, 0.9)
	if resp != 0.9 {
		t.Errorf("expected 0.9, received %v", resp)
	}
}

func TestReduceAmountToFitPortfolioLimit(t *testing.T) {
	t.Parallel()
	initialPrice := 1003.37
	initialAmount := 1337 / initialPrice
	portfolioAdjustedTotal := initialAmount * initialPrice
	adjustedPrice := 1000.0
	amount := 2.0
	finalAmount := reduceAmountToFitPortfolioLimit(adjustedPrice, amount, portfolioAdjustedTotal, gctorder.Buy)
	if finalAmount*adjustedPrice != portfolioAdjustedTotal {
		t.Errorf("expected value %v to match portfolio total %v", finalAmount*adjustedPrice, portfolioAdjustedTotal)
	}
	finalAmount = reduceAmountToFitPortfolioLimit(adjustedPrice, 133333333337, portfolioAdjustedTotal, gctorder.Sell)
	if finalAmount != portfolioAdjustedTotal {
		t.Errorf("expected value %v to match portfolio total %v", finalAmount, portfolioAdjustedTotal)
	}
	finalAmount = reduceAmountToFitPortfolioLimit(adjustedPrice, 1, portfolioAdjustedTotal, gctorder.Sell)
	if finalAmount != 1 {
		t.Errorf("expected value %v to match portfolio total %v", finalAmount, portfolioAdjustedTotal)
	}
}

func TestVerifyOrderWithinLimits(t *testing.T) {
	t.Parallel()
	err := verifyOrderWithinLimits(nil, 0, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received %v expected %v", err, common.ErrNilEvent)
	}

	err = verifyOrderWithinLimits(&fill.Fill{}, 0, nil)
	if !errors.Is(err, errNilCurrencySettings) {
		t.Errorf("received %v expected %v", err, errNilCurrencySettings)
	}

	err = verifyOrderWithinLimits(&fill.Fill{}, 0, &Settings{})
	if !errors.Is(err, errInvalidDirection) {
		t.Errorf("received %v expected %v", err, errInvalidDirection)
	}
	f := &fill.Fill{
		Direction: gctorder.Buy,
	}
	err = verifyOrderWithinLimits(f, 0, &Settings{})
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	s := &Settings{
		BuySide: config.MinMax{
			MinimumSize: 1,
			MaximumSize: 1,
		},
	}
	err = verifyOrderWithinLimits(f, 0.5, s)
	if !errors.Is(err, errExceededPortfolioLimit) {
		t.Errorf("received %v expected %v", err, errExceededPortfolioLimit)
	}
	f.Direction = gctorder.Buy
	err = verifyOrderWithinLimits(f, 2, s)
	if !errors.Is(err, errExceededPortfolioLimit) {
		t.Errorf("received %v expected %v", err, errExceededPortfolioLimit)
	}

	f.Direction = gctorder.Sell
	s.SellSide = config.MinMax{
		MinimumSize: 1,
		MaximumSize: 1,
	}
	err = verifyOrderWithinLimits(f, 0.5, s)
	if !errors.Is(err, errExceededPortfolioLimit) {
		t.Errorf("received %v expected %v", err, errExceededPortfolioLimit)
	}
	f.Direction = gctorder.Sell
	err = verifyOrderWithinLimits(f, 2, s)
	if !errors.Is(err, errExceededPortfolioLimit) {
		t.Errorf("received %v expected %v", err, errExceededPortfolioLimit)
	}
}
