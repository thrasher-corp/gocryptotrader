package exchange

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	gctconfig "github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binanceus"
	"github.com/thrasher-corp/gocryptotrader/exchanges/btcmarkets"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "okx"

type fakeFund struct{}

var leet = decimal.NewFromInt(1337)

func (f *fakeFund) GetPairReader() (funding.IPairReader, error) {
	i, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, leet, leet)
	if err != nil {
		return nil, err
	}
	j, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, leet, leet)
	if err != nil {
		return nil, err
	}
	return funding.CreatePair(i, j)
}

func (f *fakeFund) GetCollateralReader() (funding.ICollateralReader, error) {
	i, err := funding.CreateItem(testExchange, asset.Futures, currency.BTC, leet, leet)
	if err != nil {
		return nil, err
	}
	j, err := funding.CreateItem(testExchange, asset.Futures, currency.USDT, leet, leet)
	if err != nil {
		return nil, err
	}
	return funding.CreateCollateral(i, j)
}

func (f *fakeFund) PairReleaser() (funding.IPairReleaser, error) {
	btc, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, leet, leet)
	if err != nil {
		return nil, err
	}
	usdt, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, leet, leet)
	if err != nil {
		return nil, err
	}
	p, err := funding.CreatePair(btc, usdt)
	if err != nil {
		return nil, err
	}
	err = p.Reserve(leet, gctorder.Buy)
	if err != nil {
		return nil, err
	}
	err = p.Reserve(leet, gctorder.Sell)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (f *fakeFund) CollateralReleaser() (funding.ICollateralReleaser, error) {
	i, err := funding.CreateItem(testExchange, asset.Futures, currency.BTC, decimal.Zero, decimal.Zero)
	if err != nil {
		return nil, err
	}
	j, err := funding.CreateItem(testExchange, asset.Futures, currency.USDT, decimal.Zero, decimal.Zero)
	if err != nil {
		return nil, err
	}
	return funding.CreateCollateral(i, j)
}

func (f *fakeFund) IncreaseAvailable(decimal.Decimal, gctorder.Side) {}
func (f *fakeFund) Release(decimal.Decimal, decimal.Decimal, gctorder.Side) error {
	return nil
}

func TestReset(t *testing.T) {
	t.Parallel()
	e := &Exchange{
		CurrencySettings: []Settings{
			{},
		},
	}
	err := e.Reset()
	assert.NoError(t, err)

	if len(e.CurrencySettings) > 0 {
		t.Error("expected no entries")
	}

	e = nil
	err = e.Reset()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestSetCurrency(t *testing.T) {
	t.Parallel()
	e := Exchange{}
	e.SetExchangeAssetCurrencySettings(asset.Empty, currency.EMPTYPAIR, &Settings{})
	if len(e.CurrencySettings) != 0 {
		t.Error("expected 0")
	}
	f := &binance.Exchange{}
	f.Name = testExchange
	cs := &Settings{
		Exchange:      f,
		UseRealOrders: true,
		Pair:          currency.NewBTCUSDT(),
		Asset:         asset.Spot,
	}
	e.SetExchangeAssetCurrencySettings(asset.Spot, currency.NewBTCUSDT(), cs)
	result, err := e.GetCurrencySettings(testExchange, asset.Spot, currency.NewBTCUSDT())
	assert.NoError(t, err)

	if !result.UseRealOrders {
		t.Error("expected true")
	}
	e.SetExchangeAssetCurrencySettings(asset.Spot, currency.NewBTCUSDT(), cs)
	if len(e.CurrencySettings) != 1 {
		t.Error("expected 1")
	}
}

func TestEnsureOrderFitsWithinHLV(t *testing.T) {
	t.Parallel()
	adjustedPrice, adjustedAmount := ensureOrderFitsWithinHLV(decimal.NewFromInt(123), decimal.NewFromInt(1), decimal.NewFromInt(100), decimal.NewFromInt(99), decimal.NewFromInt(10))
	if !adjustedAmount.Equal(decimal.NewFromInt(1)) {
		t.Error("expected 1")
	}
	if !adjustedPrice.Equal(decimal.NewFromInt(100)) {
		t.Error("expected 100")
	}

	adjustedPrice, adjustedAmount = ensureOrderFitsWithinHLV(decimal.NewFromInt(123), decimal.NewFromInt(1), decimal.NewFromInt(100), decimal.NewFromInt(99), decimal.NewFromFloat(0.8))
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

	em := engine.NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err, "NewExchangeByName must not error")

	exch.SetDefaults()
	exchB := exch.GetBase()
	exchB.States = currencystate.NewCurrencyStates()
	err = em.Add(exch)
	require.NoError(t, err, "Add exchange must not error")
	bot.OrderManager, err = engine.SetupOrderManager(em, &engine.CommunicationManager{}, &bot.ServicesWG, &gctconfig.OrderManager{})
	require.NoError(t, err, "SetupOrderManager must not error")
	err = bot.OrderManager.Start()
	require.NoError(t, err, "Start must not error")

	e := Exchange{}
	_, err = e.placeOrder(t.Context(), decimal.NewFromInt(1), decimal.NewFromInt(1), decimal.Zero, false, true, nil, nil)
	assert.ErrorIs(t, err, common.ErrNilEvent)
	f := &fill.Fill{
		Base: &event.Base{},
	}
	_, err = e.placeOrder(t.Context(), decimal.NewFromInt(1), decimal.NewFromInt(1), decimal.Zero, false, true, f, bot.OrderManager)
	assert.ErrorIs(t, err, gctcommon.ErrExchangeNameNotSet)

	f.Exchange = testExchange
	require.NoError(t, exch.UpdateOrderExecutionLimits(t.Context(), asset.Spot), "UpdateOrderExecutionLimits must not error")

	_, err = e.placeOrder(t.Context(), decimal.NewFromInt(1), decimal.NewFromInt(1), decimal.Zero, false, true, f, bot.OrderManager)
	assert.ErrorIs(t, err, gctorder.ErrPairIsEmpty)

	f.CurrencyPair = currency.NewBTCUSDT()
	f.AssetType = asset.Spot
	f.Direction = gctorder.Buy
	_, err = e.placeOrder(t.Context(), decimal.NewFromInt(1), decimal.NewFromInt(1), decimal.Zero, false, true, f, bot.OrderManager)
	assert.NoError(t, err, "placeOrder should not error")
	_, err = e.placeOrder(t.Context(), decimal.NewFromInt(1), decimal.NewFromInt(1), decimal.Zero, true, true, f, bot.OrderManager)
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)
}

func TestExecuteOrder(t *testing.T) {
	t.Parallel()
	bot := &engine.Engine{}

	em := engine.NewExchangeManager()
	const testExchange = "binanceus"
	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err, "NewExchangeByName must not error")
	exch.SetDefaults()
	exchB := exch.GetBase()
	exchB.States = currencystate.NewCurrencyStates()
	err = em.Add(exch)
	require.NoError(t, err, "ExchangeManager.Add exchange must not error")
	bot.OrderManager, err = engine.SetupOrderManager(em, &engine.CommunicationManager{}, &bot.ServicesWG, &gctconfig.OrderManager{})
	require.NoError(t, err, "engine.SetupOrderManager must not error")
	err = bot.OrderManager.Start()
	require.NoError(t, err, "OrderManager.Start must not error")

	p := currency.NewBTCUSDT()
	a := asset.Spot
	require.NoError(t, exchB.CurrencyPairs.SetAssetEnabled(a, true), "SetAssetEnabled must not error")
	_, err = exch.UpdateOrderbook(t.Context(), p, a)
	require.NoError(t, err, "UpdateOrderbook must not error")
	f := &binanceus.Exchange{}
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

	item := &gctkline.Item{
		Exchange: testExchange,
		Pair:     p,
		Asset:    a,
		Interval: 0,
		Candles: []gctkline.Candle{
			{
				Close: 1,
				High:  1,
				Low:   1,
				Time:  time.Now(),
			},
		},
	}
	d := &kline.DataFromKline{
		Base: &data.Base{},
		Item: item,
	}
	err = d.Load()
	assert.NoError(t, err)

	_, err = d.Next()
	assert.NoError(t, err)

	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	assert.ErrorIs(t, err, errNoCurrencySettingsFound)

	cs.UseRealOrders = true
	cs.CanUseExchangeLimits = true
	o.Direction = gctorder.Sell
	e.CurrencySettings = []Settings{cs}
	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)

	o.LiquidatingPosition = true
	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	assert.NoError(t, err)

	o.AssetType = asset.Futures
	e.CurrencySettings[0].Asset = asset.Futures
	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	assert.NoError(t, err)

	o.LiquidatingPosition = false
	o.Amount = decimal.Zero
	o.AssetType = asset.Spot
	e.CurrencySettings[0].Asset = asset.Spot
	e.CurrencySettings[0].UseRealOrders = false
	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	assert.ErrorIs(t, err, gctorder.ErrAmountIsInvalid)
}

func TestExecuteOrderBuySellSizeLimit(t *testing.T) {
	t.Parallel()
	bot := &engine.Engine{}
	em := engine.NewExchangeManager()
	const testExchange = "BTC Markets"
	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err, "NewExchangeByName must not error")
	exch.SetDefaults()
	exchB := exch.GetBase()
	exchB.CurrencyPairs = currency.PairsManager{
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot: {
				AssetEnabled: true,
			},
		},
	}
	exchB.States = currencystate.NewCurrencyStates()
	err = em.Add(exch)
	require.NoError(t, err, "Add exchange must not error")
	bot.OrderManager, err = engine.SetupOrderManager(em, &engine.CommunicationManager{}, &bot.ServicesWG, &gctconfig.OrderManager{})
	require.NoError(t, err, "SetupOrderManager must not error")
	err = bot.OrderManager.Start()
	require.NoError(t, err, "Start must not error")

	p := currency.NewPair(currency.BTC, currency.AUD)
	a := asset.Spot
	_, err = exch.UpdateOrderbook(t.Context(), p, a)
	require.NoError(t, err, "UpdateOrderbook must not error")

	err = exch.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateOrderExecutionLimits must not error")

	l, err := exch.GetOrderExecutionLimits(a, p)
	require.NoError(t, err, "GetOrderExecutionLimits must not error")

	f := &btcmarkets.Exchange{}
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
		Limits:              l,
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
		Base: &data.Base{},
		Item: &gctkline.Item{
			Exchange: testExchange,
			Pair:     p,
			Asset:    asset.Spot,
			Interval: gctkline.FifteenMin,
			Candles: []gctkline.Candle{
				{
					Close:  1,
					High:   1,
					Low:    1,
					Volume: 1,
					Time:   time.Now(),
				},
			},
		},
	}
	err = d.Load()
	require.NoError(t, err, "Load must not error")

	_, err = d.Next()
	require.NoError(t, err, "Next must not error")

	_, err = e.ExecuteOrder(o, d, bot.OrderManager, &fakeFund{})
	assert.ErrorIs(t, err, errExceededPortfolioLimit)

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
	assert.NoError(t, err, "ExecuteOrder should not error when limitReducedAmount adjusted to 0.99999999, direction BUY {MinimumSize:0.01 MaximumSize:0 MaximumTotal:0}")

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
	assert.NoError(t, err, "ExecuteOrder should not error when limitReducedAmount adjusted to 0.99999999, direction SELL {MinimumSize:0.01 MaximumSize:0 MaximumTotal:0}")

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
	assert.ErrorIs(t, err, errExceededPortfolioLimit)

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
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)
}

func TestApplySlippageToPrice(t *testing.T) {
	t.Parallel()
	resp, err := applySlippageToPrice(gctorder.Buy, decimal.NewFromInt(1), decimal.NewFromFloat(0.9))
	assert.NoError(t, err)

	if !resp.Equal(decimal.NewFromFloat(1.1)) {
		t.Errorf("received: %v, expected: %v", resp, decimal.NewFromFloat(1.1))
	}

	resp, err = applySlippageToPrice(gctorder.Sell, decimal.NewFromInt(1), decimal.NewFromFloat(0.9))
	assert.NoError(t, err)

	if !resp.Equal(decimal.NewFromFloat(0.9)) {
		t.Errorf("received: %v, expected: %v", resp, decimal.NewFromFloat(0.9))
	}

	resp, err = applySlippageToPrice(gctorder.Sell, decimal.NewFromInt(1), decimal.Zero)
	assert.NoError(t, err)

	if !resp.Equal(decimal.NewFromFloat(1)) {
		t.Errorf("received: %v, expected: %v", resp, decimal.NewFromFloat(1))
	}

	_, err = applySlippageToPrice(gctorder.UnknownSide, decimal.NewFromInt(1), decimal.NewFromFloat(0.9))
	assert.ErrorIs(t, err, gctorder.ErrSideIsInvalid)
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
	assert.ErrorIs(t, err, common.ErrNilEvent)

	err = verifyOrderWithinLimits(&fill.Fill{}, decimal.Zero, nil)
	assert.ErrorIs(t, err, errNilCurrencySettings)

	err = verifyOrderWithinLimits(&fill.Fill{}, decimal.Zero, &Settings{})
	assert.ErrorIs(t, err, errInvalidDirection)

	f := &fill.Fill{
		Direction: gctorder.Buy,
	}
	err = verifyOrderWithinLimits(f, decimal.Zero, &Settings{})
	assert.NoError(t, err)

	s := &Settings{
		BuySide: MinMax{
			MinimumSize: decimal.NewFromInt(1),
			MaximumSize: decimal.NewFromInt(1),
		},
	}
	f.Base = &event.Base{}
	err = verifyOrderWithinLimits(f, decimal.NewFromFloat(0.5), s)
	assert.ErrorIs(t, err, errExceededPortfolioLimit)

	f.Direction = gctorder.Buy
	err = verifyOrderWithinLimits(f, decimal.NewFromInt(2), s)
	assert.ErrorIs(t, err, errExceededPortfolioLimit)

	f.Direction = gctorder.Sell
	s.SellSide = MinMax{
		MinimumSize: decimal.NewFromInt(1),
		MaximumSize: decimal.NewFromInt(1),
	}
	err = verifyOrderWithinLimits(f, decimal.NewFromFloat(0.5), s)
	assert.ErrorIs(t, err, errExceededPortfolioLimit)

	f.Direction = gctorder.Sell
	err = verifyOrderWithinLimits(f, decimal.NewFromInt(2), s)
	assert.ErrorIs(t, err, errExceededPortfolioLimit)
}

func TestAllocateFundsPostOrder(t *testing.T) {
	t.Parallel()
	err := allocateFundsPostOrder(nil, nil, nil, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero)
	assert.ErrorIs(t, err, common.ErrNilEvent)

	f := &fill.Fill{
		Base: &event.Base{
			AssetType: asset.Spot,
		},
		Direction: gctorder.Buy,
	}
	err = allocateFundsPostOrder(f, nil, nil, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	one := decimal.NewFromInt(1)
	item, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.NewFromInt(1337), decimal.Zero)
	require.NoError(t, err, "CreateItem must not error")

	item2, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, decimal.NewFromInt(1337), decimal.Zero)
	require.NoError(t, err, "CreateItem must not error")

	err = item.Reserve(one)
	require.NoError(t, err, "Reserve must not error")

	err = item2.Reserve(one)
	require.NoError(t, err, "Reserve must not error")

	fundPair, err := funding.CreatePair(item, item2)
	require.NoError(t, err, "CreatePair must not error")

	f.Order = &gctorder.Detail{}
	err = allocateFundsPostOrder(f, fundPair, nil, one, one, one, one, decimal.Zero)
	require.NoError(t, err, "allocateFundsPostOrder must not error")

	f.SetDirection(gctorder.Sell)
	err = allocateFundsPostOrder(f, fundPair, nil, one, one, one, one, decimal.Zero)
	require.NoError(t, err, "allocateFundsPostOrder must not error")

	err = allocateFundsPostOrder(f, fundPair, gctorder.ErrSubmissionIsNil, one, one, one, one, decimal.Zero)
	assert.ErrorIs(t, err, gctorder.ErrSubmissionIsNil)

	f.AssetType = asset.Futures
	f.SetDirection(gctorder.Short)
	item3, err := funding.CreateItem(testExchange, asset.Futures, currency.BTC, decimal.NewFromInt(1337), decimal.Zero)
	require.NoError(t, err, "CreateItem must not error")

	item4, err := funding.CreateItem(testExchange, asset.Futures, currency.USDT, decimal.NewFromInt(1337), decimal.Zero)
	require.NoError(t, err, "CreateItem must not error")

	err = item3.Reserve(one)
	require.NoError(t, err, "Reserve must not error")

	err = item4.Reserve(one)
	require.NoError(t, err, "Reserve must not error")

	collateralPair, err := funding.CreateCollateral(item, item2)
	require.NoError(t, err, "CreateCollateral must not error")

	err = allocateFundsPostOrder(f, collateralPair, gctorder.ErrSubmissionIsNil, one, one, one, one, decimal.Zero)
	assert.ErrorIs(t, err, gctorder.ErrSubmissionIsNil)

	err = allocateFundsPostOrder(f, collateralPair, nil, one, one, one, one, decimal.Zero)
	require.NoError(t, err, "allocateFundsPostOrder must not error")

	f.SetDirection(gctorder.Long)
	err = allocateFundsPostOrder(f, collateralPair, gctorder.ErrSubmissionIsNil, one, one, one, one, decimal.Zero)
	assert.ErrorIs(t, err, gctorder.ErrSubmissionIsNil)

	err = allocateFundsPostOrder(f, collateralPair, nil, one, one, one, one, decimal.Zero)
	assert.NoError(t, err, "allocateFundsPostOrder should not error")

	f.AssetType = asset.Margin
	err = allocateFundsPostOrder(f, collateralPair, nil, one, one, one, one, decimal.Zero)
	assert.ErrorIs(t, err, common.ErrInvalidDataType)
}
