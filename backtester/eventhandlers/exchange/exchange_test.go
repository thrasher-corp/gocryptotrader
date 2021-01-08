package exchange

import (
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestReset(t *testing.T) {
	e := Exchange{
		CurrencySettings: []Settings{},
	}
	e.Reset()
	if e.CurrencySettings != nil {
		t.Error("expected nil")
	}
}

func TestSetCurrency(t *testing.T) {
	e := Exchange{}
	e.SetCurrency("", "", currency.Pair{}, Settings{})
	if len(e.CurrencySettings) != 0 {
		t.Error("expected 0")
	}
	cs := Settings{
		ExchangeName:        "binance",
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
	e.SetCurrency("binance", asset.Spot, currency.NewPair(currency.BTC, currency.USDT), cs)
	result, err := e.GetCurrencySettings("binance", asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
	if result.InitialFunds != 1337 {
		t.Errorf("expected 1337, received %v", result.InitialFunds)
	}

	e.SetCurrency("binance", asset.Spot, currency.NewPair(currency.BTC, currency.USDT), cs)
	if len(e.CurrencySettings) != 1 {
		t.Error("expected 1")
	}
}

func TestEnsureOrderFitsWithinHLV(t *testing.T) {
	adjustedPrice, adjustedAmount := ensureOrderFitsWithinHLV(123, 1, 100, 99, 100)
	if adjustedAmount != 1 {
		t.Error("expected 1")
	}
	if adjustedPrice != 100 {
		t.Error("expected 100")
	}

	adjustedPrice, adjustedAmount = ensureOrderFitsWithinHLV(123, 1, 100, 99, 80)
	if adjustedAmount != 0.7999999992619746 {
		t.Errorf("expected %v", adjustedAmount)
	}
	if adjustedPrice != 100 {
		t.Error("expected 100")
	}
}

func TestCalculateExchangeFee(t *testing.T) {
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
	e := Exchange{}
	_, _, err := e.sizeOrder(0, 0, 0, nil, nil)
	if err != nil && err.Error() != "received nil arguments" {
		t.Error(err)
	}
	cs := &Settings{}
	f := &fill.Fill{
		ClosePrice: 1337,
		Amount:     1,
	}
	var p, a float64
	p, a, err = e.sizeOrder(0, 0, 0, cs, f)
	if err != nil && err.Error() != "amount set to 0, data may be incorrect" {
		t.Error(err)
	}
	p, a, err = e.sizeOrder(10, 2, 10, cs, f)
	if err != nil && err.Error() != "amount set to 0, data may be incorradsect" {
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
	var err error
	engine.Bot, err = engine.New()
	if err != nil {
		t.Error(err)
	}
	err = engine.Bot.OrderManager.Start()
	if err != nil {
		t.Error(err)
	}
	err = engine.Bot.LoadExchange("binance", false, nil)
	if err != nil {
		t.Error(err)
	}
	e := Exchange{}
	_, err = e.placeOrder(1, 1, false, nil)
	if err != nil && err.Error() != "received nil event" {
		t.Error(err)
	}
	f := &fill.Fill{}
	_, err = e.placeOrder(1, 1, false, f)
	if err != nil && err.Error() != "order exchange name must be specified" {
		t.Error(err)
	}

	f.Exchange = "binance"
	_, err = e.placeOrder(1, 1, false, f)
	if err != nil && err.Error() != "order pair is empty" {
		t.Error(err)
	}
	f.CurrencyPair = currency.NewPair(currency.BTC, currency.USDT)
	f.AssetType = asset.Spot
	f.Direction = gctorder.Buy
	_, err = e.placeOrder(1, 1, false, f)
	if err != nil {
		t.Error(err)
	}

	_, err = e.placeOrder(1, 1, true, f)
	if err != nil && !strings.Contains(err.Error(), "unset/default API keys") {
		t.Error(err)
	}
}

func TestExecuteOrder(t *testing.T) {
	cs := Settings{
		ExchangeName:        "binance",
		UseRealOrders:       false,
		InitialFunds:        1337,
		CurrencyPair:        currency.NewPair(currency.BTC, currency.USDT),
		AssetType:           asset.Spot,
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
	ev := event.Event{
		Exchange:     "binance",
		Time:         time.Now(),
		Interval:     gctkline.FifteenMin,
		CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
		AssetType:    asset.Spot,
	}
	o := &order.Order{
		Event:     ev,
		Direction: gctorder.Buy,
		Amount:    1,
	}

	var err error
	engine.Bot, err = engine.New()
	if err != nil {
		t.Error(err)
	}
	err = engine.Bot.OrderManager.Start()
	if err != nil {
		t.Error(err)
	}
	err = engine.Bot.LoadExchange("binance", false, nil)
	if err != nil {
		t.Error(err)
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
	d.Load()
	d.Next()
	_, err = e.ExecuteOrder(o, d)
	if err != nil {
		t.Error(err)
	}

	cs.UseRealOrders = true
	o.Direction = gctorder.Sell
	e.CurrencySettings = []Settings{cs}
	_, err = e.ExecuteOrder(o, d)
	if err != nil && !strings.Contains(err.Error(), "unset/default API keys") {
		t.Error(err)
	}

}

func TestApplySlippageToPrice(t *testing.T) {
	resp := applySlippageToPrice(gctorder.Buy, 1, 0.9)
	if resp != 1.1 {
		t.Errorf("expected 1.1, received %v", resp)
	}
	resp = applySlippageToPrice(gctorder.Sell, 1, 0.9)
	if resp != 0.9 {
		t.Errorf("expected 0.9, received %v", resp)
	}
}
