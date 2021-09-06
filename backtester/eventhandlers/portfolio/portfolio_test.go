package portfolio

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/settings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "binance"

func TestReset(t *testing.T) {
	t.Parallel()
	p := Portfolio{
		exchangeAssetPairSettings: make(map[string]map[asset.Item]map[currency.Pair]*settings.Settings),
	}
	p.Reset()
	if p.exchangeAssetPairSettings != nil {
		t.Error("expected nil")
	}
}

func TestSetup(t *testing.T) {
	t.Parallel()
	_, err := Setup(nil, nil, decimal.NewFromInt(-1))
	if !errors.Is(err, errSizeManagerUnset) {
		t.Errorf("expected: %v, received %v", errSizeManagerUnset, err)
	}

	_, err = Setup(&size.Size{}, nil, decimal.NewFromInt(-1))
	if !errors.Is(err, errNegativeRiskFreeRate) {
		t.Errorf("expected: %v, received %v", errNegativeRiskFreeRate, err)
	}

	_, err = Setup(&size.Size{}, nil, decimal.NewFromInt(1))
	if !errors.Is(err, errRiskManagerUnset) {
		t.Errorf("expected: %v, received %v", errRiskManagerUnset, err)
	}
	var p *Portfolio
	p, err = Setup(&size.Size{}, &risk.Risk{}, decimal.NewFromInt(1))
	if err != nil {
		t.Error(err)
	}
	if !p.riskFreeRate.Equal(decimal.NewFromInt(1)) {
		t.Error("expected 1")
	}
}

func TestSetupCurrencySettingsMap(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	_, err := p.SetupCurrencySettingsMap("", "", currency.Pair{})
	if !errors.Is(err, errExchangeUnset) {
		t.Errorf("expected: %v, received %v", errExchangeUnset, err)
	}

	_, err = p.SetupCurrencySettingsMap("hi", "", currency.Pair{})
	if !errors.Is(err, errAssetUnset) {
		t.Errorf("expected: %v, received %v", errAssetUnset, err)
	}

	_, err = p.SetupCurrencySettingsMap("hi", asset.Spot, currency.Pair{})
	if !errors.Is(err, errCurrencyPairUnset) {
		t.Errorf("expected: %v, received %v", errCurrencyPairUnset, err)
	}

	_, err = p.SetupCurrencySettingsMap("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if err != nil {
		t.Error(err)
	}
}

func TestSetHoldings(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}

	err := p.setHoldingsForOffset("", "", currency.Pair{}, &holdings.Holding{}, false)
	if !errors.Is(err, errHoldingsNoTimestamp) {
		t.Errorf("expected: %v, received %v", errHoldingsNoTimestamp, err)
	}
	tt := time.Now()

	err = p.setHoldingsForOffset("", "", currency.Pair{}, &holdings.Holding{Timestamp: tt}, false)
	if !errors.Is(err, errExchangeUnset) {
		t.Errorf("expected: %v, received %v", errExchangeUnset, err)
	}

	err = p.setHoldingsForOffset(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD), &holdings.Holding{Timestamp: tt}, false)
	if err != nil {
		t.Error(err)
	}

	err = p.setHoldingsForOffset(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD), &holdings.Holding{Timestamp: tt}, true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLatestHoldingsForAllCurrencies(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	h := p.GetLatestHoldingsForAllCurrencies()
	if len(h) != 0 {
		t.Error("expected 0")
	}
	tt := time.Now()
	err := p.setHoldingsForOffset(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD), &holdings.Holding{Timestamp: tt}, true)
	if !errors.Is(err, errNoHoldings) {
		t.Errorf("expected: %v, received %v", errNoHoldings, err)
	}
	h = p.GetLatestHoldingsForAllCurrencies()
	if len(h) != 0 {
		t.Errorf("received %v, expected %v", len(h), 0)
	}
	err = p.setHoldingsForOffset(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.DOGE), &holdings.Holding{Offset: 1, Timestamp: tt}, false)
	if err != nil {
		t.Error(err)
	}
	h = p.GetLatestHoldingsForAllCurrencies()
	if len(h) != 1 {
		t.Errorf("received %v, expected %v", len(h), 1)
	}
	err = p.setHoldingsForOffset(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.DOGE), &holdings.Holding{Offset: 1, Timestamp: tt}, false)
	if !errors.Is(err, errHoldingsAlreadySet) {
		t.Errorf("expected: %v, received %v", errHoldingsAlreadySet, err)
	}
	err = p.setHoldingsForOffset(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.DOGE), &holdings.Holding{Offset: 1, Timestamp: tt}, true)
	if !errors.Is(err, nil) {
		t.Errorf("expected: %v, received %v", nil, err)
	}
	h = p.GetLatestHoldingsForAllCurrencies()
	if len(h) != 1 {
		t.Errorf("received %v, expected %v", len(h), 1)
	}
}

func TestViewHoldingAtTimePeriod(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	tt := time.Now()
	s := &signal.Signal{
		Base: event.Base{
			Time:         tt,
			Exchange:     testExchange,
			AssetType:    asset.Spot,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
		},
	}
	_, err := p.ViewHoldingAtTimePeriod(s)
	if !errors.Is(err, errNoHoldings) {
		t.Errorf("expected: %v, received %v", errNoHoldings, err)
	}

	err = p.setHoldingsForOffset(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD), &holdings.Holding{Offset: 1, Timestamp: tt}, false)
	if err != nil {
		t.Error(err)
	}
	err = p.setHoldingsForOffset(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD), &holdings.Holding{Offset: 2, Timestamp: tt.Add(time.Hour)}, false)
	if err != nil {
		t.Error(err)
	}
	var h *holdings.Holding
	h, err = p.ViewHoldingAtTimePeriod(s)
	if err != nil {
		t.Fatal(err)
	}
	if !h.Timestamp.Equal(tt) {
		t.Errorf("expected %v received %v", tt, h.Timestamp)
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	err := p.UpdateHoldings(nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, received %v", common.ErrNilEvent, err)
	}

	err = p.UpdateHoldings(&kline.Kline{}, nil)
	if !errors.Is(err, funding.ErrFundsNotFound) {
		t.Errorf("received '%v' expected '%v'", err, funding.ErrFundsNotFound)
	}
	b, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.NewFromInt(1), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	q, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, decimal.NewFromInt(100), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := funding.CreatePair(b, q)
	if err != nil {
		t.Fatal(err)
	}
	err = p.UpdateHoldings(&kline.Kline{}, pair)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received '%v' expected '%v'", err, errNoPortfolioSettings)
	}

	tt := time.Now()
	err = p.setHoldingsForOffset(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD), &holdings.Holding{Offset: 1, Timestamp: tt}, false)
	if err != nil {
		t.Error(err)
	}

	err = p.UpdateHoldings(&kline.Kline{
		Base: event.Base{
			Time:         tt,
			Exchange:     testExchange,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
			AssetType:    asset.Spot,
		},
	}, pair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	f := p.GetFee("", "", currency.Pair{})
	if !f.IsZero() {
		t.Error("expected 0")
	}

	_, err := p.SetupCurrencySettingsMap("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if err != nil {
		t.Error(err)
	}

	p.SetFee("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD), decimal.NewFromInt(1337))
	f = p.GetFee("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if !f.Equal(decimal.NewFromInt(1337)) {
		t.Errorf("expected %v received %v", 1337, f)
	}
}

func TestGetComplianceManager(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	_, err := p.GetComplianceManager("", "", currency.Pair{})
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("expected: %v, received %v", errNoPortfolioSettings, err)
	}

	_, err = p.SetupCurrencySettingsMap("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if err != nil {
		t.Error(err)
	}
	var cm *compliance.Manager
	cm, err = p.GetComplianceManager("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if err != nil {
		t.Error(err)
	}
	if cm == nil {
		t.Error("expected not nil")
	}
}

func TestAddComplianceSnapshot(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	err := p.addComplianceSnapshot(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, received %v", common.ErrNilEvent, err)
	}

	err = p.addComplianceSnapshot(&fill.Fill{})
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("expected: %v, received %v", errNoPortfolioSettings, err)
	}

	_, err = p.SetupCurrencySettingsMap("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if err != nil {
		t.Error(err)
	}

	err = p.addComplianceSnapshot(&fill.Fill{
		Base: event.Base{
			Exchange:     "hi",
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
			AssetType:    asset.Spot,
		},
		Order: &gctorder.Detail{
			Exchange:  "hi",
			Pair:      currency.NewPair(currency.BTC, currency.USD),
			AssetType: asset.Spot,
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestOnFill(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	_, err := p.OnFill(nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, received %v", common.ErrNilEvent, err)
	}

	f := &fill.Fill{
		Base: event.Base{
			Exchange:     "hi",
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
			AssetType:    asset.Spot,
		},
		Order: &gctorder.Detail{
			Exchange:  "hi",
			Pair:      currency.NewPair(currency.BTC, currency.USD),
			AssetType: asset.Spot,
		},
	}
	_, err = p.OnFill(f, nil)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("expected: %v, received %v", errNoPortfolioSettings, err)
	}
	_, err = p.SetupCurrencySettingsMap("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if err != nil {
		t.Error(err)
	}

	b, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.NewFromInt(1), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	q, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, decimal.NewFromInt(100), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := funding.CreatePair(b, q)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.OnFill(f, pair)
	if !errors.Is(err, nil) {
		t.Errorf("expected: %v, received %v", nil, err)
	}

	f.Direction = gctorder.Buy
	_, err = p.OnFill(f, pair)
	if err != nil {
		t.Error(err)
	}
}

func TestOnSignal(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	_, err := p.OnSignal(nil, nil, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Error(err)
	}

	s := &signal.Signal{}
	_, err = p.OnSignal(s, &exchange.Settings{}, nil)
	if !errors.Is(err, errSizeManagerUnset) {
		t.Errorf("expected: %v, received %v", errSizeManagerUnset, err)
	}
	p.sizeManager = &size.Size{}

	_, err = p.OnSignal(s, &exchange.Settings{}, nil)
	if !errors.Is(err, errRiskManagerUnset) {
		t.Errorf("expected: %v, received %v", errRiskManagerUnset, err)
	}

	p.riskManager = &risk.Risk{}

	_, err = p.OnSignal(s, &exchange.Settings{}, nil)
	if !errors.Is(err, funding.ErrFundsNotFound) {
		t.Errorf("expected: %v, received %v", funding.ErrFundsNotFound, err)
	}
	b, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.NewFromInt(1337), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	q, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, decimal.NewFromInt(1337), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := funding.CreatePair(b, q)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.OnSignal(s, &exchange.Settings{}, pair)
	if !errors.Is(err, errInvalidDirection) {
		t.Errorf("expected: %v, received %v", errInvalidDirection, err)
	}

	s.Direction = gctorder.Buy
	_, err = p.OnSignal(s, &exchange.Settings{}, pair)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("expected: %v, received %v", errNoPortfolioSettings, err)
	}
	_, err = p.SetupCurrencySettingsMap("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD))
	if err != nil {
		t.Error(err)
	}
	s = &signal.Signal{
		Base: event.Base{
			Exchange:     "hi",
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
			AssetType:    asset.Spot,
		},
		Direction: gctorder.Buy,
	}
	var resp *order.Order
	resp, err = p.OnSignal(s, &exchange.Settings{}, pair)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Reason == "" {
		t.Error("expected issue")
	}

	s.Direction = gctorder.Sell
	_, err = p.OnSignal(s, &exchange.Settings{}, pair)
	if err != nil {
		t.Error(err)
	}
	if resp.Reason == "" {
		t.Error("expected issue")
	}

	s.Direction = common.MissingData
	_, err = p.OnSignal(s, &exchange.Settings{}, pair)
	if err != nil {
		t.Error(err)
	}

	s.Direction = gctorder.Buy
	err = p.setHoldingsForOffset("hi", asset.Spot, currency.NewPair(currency.BTC, currency.USD), &holdings.Holding{Timestamp: time.Now(), QuoteSize: decimal.NewFromInt(1337)}, false)
	if err != nil {
		t.Error(err)
	}
	resp, err = p.OnSignal(s, &exchange.Settings{}, pair)
	if err != nil {
		t.Error(err)
	}
	if resp.Direction != common.CouldNotBuy {
		t.Errorf("expected common.CouldNotBuy, received %v", resp.Direction)
	}

	s.ClosePrice = decimal.NewFromInt(10)
	s.Direction = gctorder.Buy
	resp, err = p.OnSignal(s, &exchange.Settings{}, pair)
	if err != nil {
		t.Error(err)
	}
	if resp.Amount.IsZero() {
		t.Error("expected an amount to be sized")
	}
}
