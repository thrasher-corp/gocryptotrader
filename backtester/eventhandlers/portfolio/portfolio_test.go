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
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "binance"

var leet = decimal.NewFromInt(1337)

func TestReset(t *testing.T) {
	t.Parallel()
	p := &Portfolio{
		exchangeAssetPairPortfolioSettings: make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*Settings),
	}
	err := p.Reset()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if p.exchangeAssetPairPortfolioSettings == nil {
		t.Error("expected a map")
	}

	p = nil
	err = p.Reset()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}
}

func TestSetup(t *testing.T) {
	t.Parallel()
	_, err := Setup(nil, nil, decimal.NewFromInt(-1))
	if !errors.Is(err, errSizeManagerUnset) {
		t.Errorf("received: %v, expected: %v", err, errSizeManagerUnset)
	}

	_, err = Setup(&size.Size{}, nil, decimal.NewFromInt(-1))
	if !errors.Is(err, errNegativeRiskFreeRate) {
		t.Errorf("received: %v, expected: %v", err, errNegativeRiskFreeRate)
	}

	_, err = Setup(&size.Size{}, nil, decimal.NewFromInt(1))
	if !errors.Is(err, errRiskManagerUnset) {
		t.Errorf("received: %v, expected: %v", err, errRiskManagerUnset)
	}
	var p *Portfolio
	p, err = Setup(&size.Size{}, &risk.Risk{}, decimal.NewFromInt(1))
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if !p.riskFreeRate.Equal(decimal.NewFromInt(1)) {
		t.Error("expected 1")
	}
}

func TestSetupCurrencySettingsMap(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	err := p.SetCurrencySettingsMap(nil)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	err = p.SetCurrencySettingsMap(&exchange.Settings{})
	if !errors.Is(err, errExchangeUnset) {
		t.Errorf("received: %v, expected: %v", err, errExchangeUnset)
	}

	ff := &binance.Binance{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff})
	if !errors.Is(err, errAssetUnset) {
		t.Errorf("received: %v, expected: %v", err, errAssetUnset)
	}

	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot})
	if !errors.Is(err, errCurrencyPairUnset) {
		t.Errorf("received: %v, expected: %v", err, errCurrencyPairUnset)
	}

	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USDT)})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestSetHoldings(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}

	err := p.SetHoldingsForTimestamp(&holdings.Holding{})
	if !errors.Is(err, errHoldingsNoTimestamp) {
		t.Errorf("received: %v, expected: %v", err, errHoldingsNoTimestamp)
	}
	tt := time.Now()

	err = p.SetHoldingsForTimestamp(&holdings.Holding{Timestamp: tt})
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	ff := &binance.Binance{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USDT)})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		Timestamp: tt})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		Timestamp: tt})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
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
	err := p.SetHoldingsForTimestamp(&holdings.Holding{
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		Timestamp: tt})
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	ff := &binance.Binance{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USDT)})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	h = p.GetLatestHoldingsForAllCurrencies()
	if len(h) != 0 {
		t.Errorf("received %v, expected %v", len(h), 0)
	}
	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Offset:    1,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		Timestamp: tt})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	h = p.GetLatestHoldingsForAllCurrencies()
	if len(h) != 1 {
		t.Errorf("received %v, expected %v", len(h), 1)
	}
	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Offset:    1,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		Timestamp: tt})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
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
		Base: &event.Base{
			Time:         tt,
			Exchange:     testExchange,
			AssetType:    asset.Spot,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
		},
	}
	_, err := p.ViewHoldingAtTimePeriod(s)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	ff := &binance.Binance{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USDT)})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	_, err = p.ViewHoldingAtTimePeriod(s)
	if !errors.Is(err, errNoHoldings) {
		t.Errorf("received: %v, expected: %v", err, errNoHoldings)
	}

	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Offset:    1,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		Timestamp: tt})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Offset:    2,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		Timestamp: tt.Add(time.Hour)})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
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
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}

	err = p.UpdateHoldings(&kline.Kline{}, nil)
	if !errors.Is(err, funding.ErrFundsNotFound) {
		t.Errorf("received '%v' expected '%v'", err, funding.ErrFundsNotFound)
	}
	bc, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.NewFromInt(1), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	qc, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, decimal.NewFromInt(100), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := funding.CreatePair(bc, qc)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	b := &event.Base{}
	err = p.UpdateHoldings(&kline.Kline{
		Base: b,
	}, pair)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received '%v' expected '%v'", err, errNoPortfolioSettings)
	}

	tt := time.Now()
	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Offset:    1,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		Timestamp: tt,
	})
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	ff := &binance.Binance{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USDT)})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	b.Time = tt
	b.Exchange = testExchange
	b.CurrencyPair = currency.NewPair(currency.BTC, currency.USDT)
	b.AssetType = asset.Spot
	err = p.UpdateHoldings(&kline.Kline{
		Base: b,
	}, pair)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestGetComplianceManager(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	_, err := p.getComplianceManager("", asset.Empty, currency.EMPTYPAIR)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	ff := &binance.Binance{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USDT)})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	var cm *compliance.Manager
	cm, err = p.getComplianceManager(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
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
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}

	err = p.addComplianceSnapshot(&fill.Fill{
		Base: &event.Base{},
	})
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	ff := &binance.Binance{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USDT)})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = p.addComplianceSnapshot(&fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
			AssetType:    asset.Spot,
		},
		Order: &gctorder.Detail{
			Exchange:  testExchange,
			Pair:      currency.NewPair(currency.BTC, currency.USDT),
			AssetType: asset.Spot,
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestOnFill(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	_, err := p.OnFill(nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}

	f := &fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
			AssetType:    asset.Spot,
		},
		Order: &gctorder.Detail{
			Exchange:  testExchange,
			Pair:      currency.NewPair(currency.BTC, currency.USDT),
			AssetType: asset.Spot,
		},
	}
	_, err = p.OnFill(f, nil)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}
	ff := &binance.Binance{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USDT)})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
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
	if !errors.Is(err, errHoldingsNoTimestamp) {
		t.Errorf("received: %v, expected: %v", err, errHoldingsNoTimestamp)
	}

	f.Time = time.Now()
	_, err = p.OnFill(f, pair)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	f.Direction = gctorder.Buy
	_, err = p.OnFill(f, pair)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestOnSignal(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	_, err := p.OnSignal(nil, nil, nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Error(err)
	}
	b := &event.Base{}
	s := &signal.Signal{
		Base: b,
	}
	_, err = p.OnSignal(s, &exchange.Settings{}, nil)
	if !errors.Is(err, errSizeManagerUnset) {
		t.Errorf("received: %v, expected: %v", err, errSizeManagerUnset)
	}
	p.sizeManager = &size.Size{}

	_, err = p.OnSignal(s, &exchange.Settings{}, nil)
	if !errors.Is(err, errRiskManagerUnset) {
		t.Errorf("received: %v, expected: %v", err, errRiskManagerUnset)
	}

	p.riskManager = &risk.Risk{}

	_, err = p.OnSignal(s, &exchange.Settings{}, nil)
	if !errors.Is(err, funding.ErrFundsNotFound) {
		t.Errorf("received: %v, expected: %v", err, funding.ErrFundsNotFound)
	}
	bc, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, leet, decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	qc, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, leet, decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	funds, err := funding.CreatePair(bc, qc)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.OnSignal(s, &exchange.Settings{}, funds)
	if !errors.Is(err, errInvalidDirection) {
		t.Errorf("received: %v, expected: %v", err, errInvalidDirection)
	}

	s.Direction = gctorder.Buy
	_, err = p.OnSignal(s, &exchange.Settings{}, funds)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}
	ff := &binance.Binance{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	b.Exchange = testExchange
	b.CurrencyPair = currency.NewPair(currency.BTC, currency.USD)
	b.AssetType = asset.Spot
	s = &signal.Signal{
		Base:      b,
		Direction: gctorder.Buy,
	}
	var resp *order.Order
	resp, err = p.OnSignal(s, &exchange.Settings{}, funds)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Reasons) != 2 {
		t.Error("expected issue")
	}

	s.Direction = gctorder.Sell
	_, err = p.OnSignal(s, &exchange.Settings{}, funds)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(resp.Reasons) != 4 {
		t.Error("expected issue")
	}

	s.Direction = gctorder.MissingData
	_, err = p.OnSignal(s, &exchange.Settings{}, funds)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	s.Direction = gctorder.Buy
	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Exchange:  "lol",
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		Timestamp: time.Now(),
		QuoteSize: leet,
	})
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}
	cs := &exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)}
	err = p.SetCurrencySettingsMap(cs)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	resp, err = p.OnSignal(s, &exchange.Settings{}, funds)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if resp.Direction != gctorder.CouldNotBuy {
		t.Errorf("expected common.CouldNotBuy, received %v", resp.Direction)
	}

	s.ClosePrice = decimal.NewFromInt(10)
	s.Direction = gctorder.Buy
	s.Amount = decimal.NewFromInt(1)
	resp, err = p.OnSignal(s, &exchange.Settings{}, funds)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if resp.Amount.IsZero() {
		t.Error("expected an amount to be sized")
	}

	bc, err = funding.CreateItem(testExchange, asset.Futures, currency.BTC, leet, decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	qc, err = funding.CreateItem(testExchange, asset.Futures, currency.USD, leet, decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	collateralFunds, err := funding.CreateCollateral(bc, qc)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	s.AssetType = asset.Futures
	cs.Asset = asset.Futures

	err = p.SetCurrencySettingsMap(cs)
	if !errors.Is(err, gctcommon.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNotYetImplemented)
	}
	s.Direction = gctorder.Long
	_, err = p.OnSignal(s, cs, collateralFunds)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	_, err = p.getSettings(testExchange, asset.Futures, cp)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}
	exchangeSettings := &Settings{}
	exchangeSettings.FuturesTracker, err = gctorder.SetupMultiPositionTracker(&gctorder.MultiPositionTrackerSetup{
		Exchange:           testExchange,
		Asset:              asset.Futures,
		Pair:               cp,
		Underlying:         currency.USD,
		CollateralCurrency: currency.USD,
		OfflineCalculation: true,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = exchangeSettings.FuturesTracker.TrackNewOrder(&gctorder.Detail{
		Price:         1337,
		Amount:        1337,
		Exchange:      testExchange,
		OrderID:       "1337",
		ClientOrderID: "1337",
		Type:          gctorder.Market,
		Side:          gctorder.Long,
		Status:        gctorder.AnyStatus,
		AssetType:     asset.Futures,
		Date:          time.Now(),
		Pair:          currency.NewPair(currency.BTC, currency.USD),
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	s.Direction = gctorder.ClosePosition
	_, err = p.OnSignal(s, cs, collateralFunds)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}
}

func TestGetLatestHoldings(t *testing.T) {
	t.Parallel()
	s := &Settings{
		HoldingsSnapshots: make(map[int64]*holdings.Holding),
	}
	_, err := s.GetLatestHoldings()
	if !errors.Is(err, errNoHoldings) {
		t.Errorf("received: %v, expected: %v", err, errNoHoldings)
	}

	tt := time.Now()
	s.HoldingsSnapshots[tt.UnixNano()] = &holdings.Holding{Timestamp: tt}

	h, err := s.GetLatestHoldings()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if !h.Timestamp.Equal(tt) {
		t.Errorf("expected %v, received %v", tt, h.Timestamp)
	}
}

func TestGetSnapshotAtTime(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	b := &event.Base{}
	_, err := p.GetLatestOrderSnapshotForEvent(&kline.Kline{
		Base: b,
	})
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}
	cp := currency.NewPair(currency.XRP, currency.DOGE)
	ff := &binance.Binance{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: cp})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	tt := time.Now()
	s, ok := p.exchangeAssetPairPortfolioSettings[testExchange][asset.Spot][cp.Base.Item][cp.Quote.Item]
	if !ok {
		t.Fatal("couldn't get settings")
	}
	err = s.ComplianceManager.AddSnapshot(&compliance.Snapshot{
		Orders: []compliance.SnapshotOrder{
			{
				Order: &gctorder.Detail{
					Exchange:  testExchange,
					AssetType: asset.Spot,
					Pair:      cp,
					Amount:    1337,
				},
			},
		},
	}, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	b.Exchange = testExchange
	b.Time = tt
	b.Interval = gctkline.OneDay
	b.CurrencyPair = cp
	b.AssetType = asset.Spot
	e := &kline.Kline{
		Base: b,
	}

	ss, err := p.GetLatestOrderSnapshotForEvent(e)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(ss.Orders) != 1 {
		t.Fatal("expected 1")
	}
	if ss.Orders[0].Order.Amount != 1337 {
		t.Error("expected 1")
	}
}

func TestGetLatestSnapshot(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	_, err := p.GetLatestOrderSnapshots()
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}
	cp := currency.NewPair(currency.XRP, currency.DOGE)
	ff := &binance.Binance{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.XRP, currency.DOGE)})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	s, ok := p.exchangeAssetPairPortfolioSettings[testExchange][asset.Spot][cp.Base.Item][cp.Quote.Item]
	if !ok {
		t.Fatal("couldn't get settings")
	}
	err = s.ComplianceManager.AddSnapshot(&compliance.Snapshot{
		Orders: []compliance.SnapshotOrder{
			{
				Order: &gctorder.Detail{
					Exchange:  testExchange,
					AssetType: asset.Spot,
					Pair:      cp,
					Amount:    1337,
				},
			},
		},
	}, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	_, err = p.GetLatestOrderSnapshots()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = s.ComplianceManager.AddSnapshot(&compliance.Snapshot{
		Orders: []compliance.SnapshotOrder{
			{
				Order: &gctorder.Detail{
					Exchange:  testExchange,
					AssetType: asset.Spot,
					Pair:      cp,
					Amount:    1337,
				},
			},
		},
	}, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	ss, err := p.GetLatestOrderSnapshots()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(ss) != 1 {
		t.Fatal("expected 1")
	}
	if len(ss[0].Orders) != 1 {
		t.Errorf("expected 1, received %v", len(ss[0].Orders))
	}
}

func TestCalculatePNL(t *testing.T) {
	p := &Portfolio{}
	ev := &kline.Kline{
		Base: &event.Base{},
	}
	err := p.UpdatePNL(ev, decimal.Zero)
	if !errors.Is(err, gctorder.ErrNotFuturesAsset) {
		t.Errorf("received: %v, expected: %v", err, gctorder.ErrNotFuturesAsset)
	}

	exch := &binance.Binance{}
	exch.Name = testExchange
	a := asset.Futures
	pair, err := currency.NewPairFromStrings("BTC", "1231")
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = p.SetCurrencySettingsMap(&exchange.Settings{
		Exchange:      exch,
		UseRealOrders: false,
		Pair:          pair,
		Asset:         a,
	})
	if !errors.Is(err, gctcommon.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNotYetImplemented)
	}
	tt := time.Now().Add(time.Hour)
	tt0 := time.Now().Add(-time.Hour)
	ev.Exchange = exch.Name
	ev.AssetType = a
	ev.CurrencyPair = pair
	ev.Time = tt0

	err = p.UpdatePNL(ev, decimal.Zero)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	od := &gctorder.Detail{
		Price:     1336,
		Amount:    20,
		Exchange:  exch.Name,
		Side:      gctorder.Short,
		AssetType: a,
		Date:      tt0,
		Pair:      pair,
		OrderID:   "lol",
	}
	mpt, err := gctorder.SetupMultiPositionTracker(&gctorder.MultiPositionTrackerSetup{
		Exchange:           testExchange,
		Asset:              ev.AssetType,
		Pair:               ev.Pair(),
		Underlying:         currency.USDT,
		CollateralCurrency: currency.USDT,
		OfflineCalculation: true,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	s := &Settings{
		FuturesTracker: mpt,
	}

	p.exchangeAssetPairPortfolioSettings = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange] = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange][ev.AssetType] = make(map[*currency.Item]map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange][ev.AssetType][pair.Base.Item] = make(map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange][ev.AssetType][pair.Base.Item][pair.Quote.Item] = s
	ev.Close = leet
	err = s.ComplianceManager.AddSnapshot(&compliance.Snapshot{
		Timestamp: tt0,
		Orders: []compliance.SnapshotOrder{
			{
				Order: od,
			},
		},
	}, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	odCp := od.Copy()
	odCp.Price = od.Price - 1
	odCp.Side = gctorder.Long
	err = s.ComplianceManager.AddSnapshot(&compliance.Snapshot{
		Offset:    1,
		Timestamp: tt,
		Orders: []compliance.SnapshotOrder{
			{
				Order: od,
			},
			{
				Order: &odCp,
			},
		},
	}, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = s.FuturesTracker.TrackNewOrder(od)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = p.UpdatePNL(ev, decimal.NewFromInt(1))
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	pos := s.FuturesTracker.GetPositions()
	if len(pos) != 1 {
		t.Fatalf("expected one position, received '%v'", len(pos))
	}
	if len(pos[0].PNLHistory) == 0 {
		t.Fatal("expected a pnl entry ( ͡° ͜ʖ ͡°)")
	}
	if !pos[0].UnrealisedPNL.Equal(decimal.NewFromInt(26700)) {
		// 20 orders * $1 difference * 1x leverage
		t.Errorf("expected 26700, received '%v'", pos[0].UnrealisedPNL)
	}
}

func TestTrackFuturesOrder(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	_, err := p.TrackFuturesOrder(nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received '%v' expected '%v", err, common.ErrNilEvent)
	}
	_, err = p.TrackFuturesOrder(&fill.Fill{}, nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v", err, gctcommon.ErrNilPointer)
	}
	fundPair := &funding.SpotPair{}
	_, err = p.TrackFuturesOrder(&fill.Fill{}, fundPair)
	if !errors.Is(err, gctorder.ErrSubmissionIsNil) {
		t.Errorf("received '%v' expected '%v", err, gctorder.ErrSubmissionIsNil)
	}

	od := &gctorder.Detail{}
	_, err = p.TrackFuturesOrder(&fill.Fill{
		Order: od,
	}, fundPair)
	if !errors.Is(err, gctorder.ErrNotFuturesAsset) {
		t.Errorf("received '%v' expected '%v", err, gctorder.ErrNotFuturesAsset)
	}

	od.AssetType = asset.Futures
	_, err = p.TrackFuturesOrder(&fill.Fill{
		Order: od,
	}, fundPair)
	if !errors.Is(err, funding.ErrNotCollateral) {
		t.Errorf("received '%v' expected '%v", err, funding.ErrNotCollateral)
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	od.Pair = cp
	od.Exchange = testExchange
	od.Side = gctorder.Short
	od.AssetType = asset.Futures
	od.Amount = 1
	od.Price = 0
	od.OrderID = od.Exchange
	od.Date = time.Now()
	contract, err := funding.CreateItem(od.Exchange, od.AssetType, od.Pair.Base, decimal.NewFromInt(9999), decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	collateral, err := funding.CreateItem(od.Exchange, od.AssetType, od.Pair.Quote, decimal.NewFromInt(9999), decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	err = collateral.IncreaseAvailable(decimal.NewFromInt(9999))
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	err = contract.IncreaseAvailable(decimal.NewFromInt(9999))
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	err = collateral.Reserve(leet)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	err = contract.Reserve(leet)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	collat, err := funding.CreateCollateral(contract, collateral)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	_, err = p.TrackFuturesOrder(&fill.Fill{
		Order: od,
	}, collat)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received '%v' expected '%v", err, errNoPortfolioSettings)
	}

	ff := &binance.Binance{}
	ff.Name = od.Exchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Futures, Pair: cp})
	if !errors.Is(err, gctcommon.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNotYetImplemented)
	}

	_, err = p.TrackFuturesOrder(&fill.Fill{
		Order: od,
		Base: &event.Base{
			Exchange:     od.Exchange,
			AssetType:    asset.Futures,
			CurrencyPair: cp,
		},
	}, collat)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received '%v' expected '%v", err, errNoPortfolioSettings)
	}

	od.Side = gctorder.Long
	_, err = p.TrackFuturesOrder(&fill.Fill{
		Order: od,
		Base: &event.Base{
			Exchange:     od.Exchange,
			AssetType:    asset.Futures,
			CurrencyPair: cp,
			Time:         od.Date,
		},
	}, collat)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received '%v' expected '%v", err, errNoPortfolioSettings)
	}

	_, err = p.TrackFuturesOrder(&fill.Fill{
		Order:      od,
		Liquidated: true,
		Base: &event.Base{
			Exchange:     od.Exchange,
			AssetType:    asset.Futures,
			CurrencyPair: cp,
			Time:         od.Date,
		},
	}, collat)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received '%v' expected '%v", err, errNoPortfolioSettings)
	}
}

func TestGetHoldingsForTime(t *testing.T) {
	t.Parallel()
	s := &Settings{
		HoldingsSnapshots: make(map[int64]*holdings.Holding),
	}
	_, err := s.GetHoldingsForTime(time.Now())
	if !errors.Is(err, errNoHoldings) {
		t.Errorf("received '%v' expected '%v", err, errNoHoldings)
	}
	tt := time.Now()
	s.HoldingsSnapshots[tt.UnixNano()] = &holdings.Holding{
		Timestamp: tt,
		Offset:    1337,
	}
	_, err = s.GetHoldingsForTime(time.Unix(1337, 0))
	if !errors.Is(err, errNoHoldings) {
		t.Errorf("received '%v' expected '%v", err, errNoHoldings)
	}

	h, err := s.GetHoldingsForTime(tt)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v", err, nil)
	}
	if h.Timestamp.IsZero() && h.Offset != 1337 {
		t.Error("expected set holdings")
	}
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	var expectedError = common.ErrNilEvent
	_, err := p.GetPositions(nil)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	ev := &fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
			AssetType:    asset.Futures,
		},
	}
	ff := &binance.Binance{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: ev.AssetType, Pair: ev.Pair()})
	if !errors.Is(err, gctcommon.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNotYetImplemented)
	}
	expectedError = errNoPortfolioSettings
	_, err = p.GetPositions(ev)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
}

func TestGetLatestPNLForEvent(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	var expectedError = common.ErrNilEvent
	_, err := p.GetLatestPNLForEvent(nil)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	ev := &fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
			AssetType:    asset.Futures,
		},
	}
	ff := &binance.Binance{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: ev.AssetType, Pair: ev.Pair()})
	if !errors.Is(err, gctcommon.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNotYetImplemented)
	}
	expectedError = errNoPortfolioSettings
	_, err = p.GetLatestPNLForEvent(ev)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}

	mpt, err := gctorder.SetupMultiPositionTracker(&gctorder.MultiPositionTrackerSetup{
		Exchange:           testExchange,
		Asset:              ev.AssetType,
		Pair:               ev.Pair(),
		Underlying:         currency.USDT,
		CollateralCurrency: currency.USDT,
		OfflineCalculation: true,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	s := &Settings{
		FuturesTracker: mpt,
	}

	p.exchangeAssetPairPortfolioSettings = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange] = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange][ev.AssetType] = make(map[*currency.Item]map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange][ev.AssetType][ev.Pair().Base.Item] = make(map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange][ev.AssetType][ev.Pair().Base.Item][ev.Pair().Quote.Item] = s
	expectedError = nil
	err = s.FuturesTracker.TrackNewOrder(&gctorder.Detail{
		Exchange:  ev.GetExchange(),
		AssetType: ev.AssetType,
		Pair:      ev.Pair(),
		Amount:    1,
		Price:     1,
		OrderID:   "one",
		Date:      time.Now(),
		Side:      gctorder.Buy,
	})
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	latest, err := p.GetLatestPNLForEvent(ev)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	if latest == nil {
		t.Error("unexpected")
	}
}

func TestGetFuturesSettingsFromEvent(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	_, err := p.getFuturesSettingsFromEvent(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Fatalf("received '%v' expected '%v'", err, common.ErrNilEvent)
	}
	b := &event.Base{}
	_, err = p.getFuturesSettingsFromEvent(&fill.Fill{
		Base: b,
	})
	if !errors.Is(err, gctorder.ErrNotFuturesAsset) {
		t.Fatalf("received '%v' expected '%v'", err, gctorder.ErrNotFuturesAsset)
	}
	b.Exchange = testExchange
	b.CurrencyPair = currency.NewPair(currency.BTC, currency.USDT)
	b.AssetType = asset.Futures
	ev := &fill.Fill{
		Base: b,
	}
	_, err = p.getFuturesSettingsFromEvent(ev)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Fatalf("received '%v' expected '%v'", err, errNoPortfolioSettings)
	}

	ff := &binance.Binance{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: ev.AssetType, Pair: ev.Pair()})
	if !errors.Is(err, gctcommon.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNotYetImplemented)
	}
	_, err = p.getFuturesSettingsFromEvent(ev)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Fatalf("received '%v' expected '%v'", err, errNoPortfolioSettings)
	}

	_, err = p.getFuturesSettingsFromEvent(ev)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Fatalf("received '%v' expected '%v'", err, errNoPortfolioSettings)
	}
}

func TestGetUnrealisedPNL(t *testing.T) {
	t.Parallel()
	p := PNLSummary{
		Exchange:           testExchange,
		Asset:              asset.Futures,
		Pair:               currency.NewPair(currency.BTC, currency.USDT),
		CollateralCurrency: currency.USDT,
		Offset:             1,
		Result: gctorder.PNLResult{
			Time:                  time.Now(),
			UnrealisedPNL:         leet,
			RealisedPNLBeforeFees: decimal.NewFromInt(1338),
			RealisedPNL:           decimal.NewFromInt(1339),
			Price:                 decimal.NewFromInt(1331),
			Exposure:              decimal.NewFromInt(1332),
			Direction:             gctorder.Short,
			Fee:                   decimal.NewFromInt(1333),
			IsLiquidated:          true,
		},
	}
	result := p.GetUnrealisedPNL()
	if !result.PNL.Equal(p.Result.UnrealisedPNL) {
		t.Errorf("received '%v' expected '%v'", result.PNL, p.Result.UnrealisedPNL)
	}
	if !result.Time.Equal(p.Result.Time) {
		t.Errorf("received '%v' expected '%v'", result.Time, p.Result.Time)
	}
	if !result.Currency.Equal(p.CollateralCurrency) {
		t.Errorf("received '%v' expected '%v'", result.Currency, p.CollateralCurrency)
	}
}

func TestGetRealisedPNL(t *testing.T) {
	t.Parallel()
	p := PNLSummary{
		Exchange:           testExchange,
		Asset:              asset.Futures,
		Pair:               currency.NewPair(currency.BTC, currency.USDT),
		CollateralCurrency: currency.USDT,
		Offset:             1,
		Result: gctorder.PNLResult{
			Time:                  time.Now(),
			UnrealisedPNL:         leet,
			RealisedPNLBeforeFees: decimal.NewFromInt(1338),
			RealisedPNL:           decimal.NewFromInt(1339),
			Price:                 decimal.NewFromInt(1331),
			Exposure:              decimal.NewFromInt(1332),
			Direction:             gctorder.Short,
			Fee:                   decimal.NewFromInt(1333),
			IsLiquidated:          true,
		},
	}
	result := p.GetRealisedPNL()
	if !result.PNL.Equal(p.Result.RealisedPNL) {
		t.Errorf("received '%v' expected '%v'", result.PNL, p.Result.RealisedPNL)
	}
	if !result.Time.Equal(p.Result.Time) {
		t.Errorf("received '%v' expected '%v'", result.Time, p.Result.Time)
	}
	if !result.Currency.Equal(p.CollateralCurrency) {
		t.Errorf("received '%v' expected '%v'", result.Currency, p.CollateralCurrency)
	}
}

func TestGetExposure(t *testing.T) {
	t.Parallel()
	p := PNLSummary{
		Exchange:           testExchange,
		Asset:              asset.Futures,
		Pair:               currency.NewPair(currency.BTC, currency.USDT),
		CollateralCurrency: currency.USDT,
		Offset:             1,
		Result: gctorder.PNLResult{
			Time:                  time.Now(),
			UnrealisedPNL:         leet,
			RealisedPNLBeforeFees: decimal.NewFromInt(1338),
			RealisedPNL:           decimal.NewFromInt(1339),
			Price:                 decimal.NewFromInt(1331),
			Exposure:              decimal.NewFromInt(1332),
			Direction:             gctorder.Short,
			Fee:                   decimal.NewFromInt(1333),
			IsLiquidated:          true,
		},
	}
	if !p.GetExposure().Equal(p.Result.Exposure) {
		t.Errorf("received '%v' expected '%v'", p.GetExposure(), p.Result.Exposure)
	}
}

func TestGetCollateralCurrency(t *testing.T) {
	t.Parallel()
	p := PNLSummary{
		Exchange:           testExchange,
		Asset:              asset.Futures,
		Pair:               currency.NewPair(currency.BTC, currency.USDT),
		CollateralCurrency: currency.USDT,
		Offset:             1,
		Result: gctorder.PNLResult{
			Time:                  time.Now(),
			UnrealisedPNL:         leet,
			RealisedPNLBeforeFees: decimal.NewFromInt(1338),
			RealisedPNL:           decimal.NewFromInt(1339),
			Price:                 decimal.NewFromInt(1331),
			Exposure:              decimal.NewFromInt(1332),
			Direction:             gctorder.Short,
			Fee:                   decimal.NewFromInt(1333),
			IsLiquidated:          true,
		},
	}
	result := p.GetCollateralCurrency()
	if !result.Equal(p.CollateralCurrency) {
		t.Errorf("received '%v' expected '%v'", result, p.CollateralCurrency)
	}
}

func TestGetDirection(t *testing.T) {
	t.Parallel()
	p := PNLSummary{
		Exchange:           testExchange,
		Asset:              asset.Futures,
		Pair:               currency.NewPair(currency.BTC, currency.USDT),
		CollateralCurrency: currency.USDT,
		Offset:             1,
		Result: gctorder.PNLResult{
			Time:                  time.Now(),
			UnrealisedPNL:         leet,
			RealisedPNLBeforeFees: decimal.NewFromInt(1338),
			RealisedPNL:           decimal.NewFromInt(1339),
			Price:                 decimal.NewFromInt(1331),
			Exposure:              decimal.NewFromInt(1332),
			Direction:             gctorder.Short,
			Fee:                   decimal.NewFromInt(1333),
			IsLiquidated:          true,
		},
	}
	if p.GetDirection() != (p.Result.Direction) {
		t.Errorf("received '%v' expected '%v'", p.GetDirection(), p.Result.Direction)
	}
}

func TestCannotPurchase(t *testing.T) {
	t.Parallel()
	var expectedError = common.ErrNilEvent
	_, err := cannotPurchase(nil, nil)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}

	s := &signal.Signal{
		Base: &event.Base{},
	}
	expectedError = gctcommon.ErrNilPointer
	_, err = cannotPurchase(s, nil)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}

	o := &order.Order{
		Base: &event.Base{},
	}
	s.Direction = gctorder.Buy
	expectedError = nil
	result, err := cannotPurchase(s, o)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	if result.Direction != gctorder.CouldNotBuy {
		t.Errorf("received '%v' expected '%v'", result.Direction, gctorder.CouldNotBuy)
	}

	s.Direction = gctorder.Sell
	expectedError = nil
	result, err = cannotPurchase(s, o)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	if result.Direction != gctorder.CouldNotSell {
		t.Errorf("received '%v' expected '%v'", result.Direction, gctorder.CouldNotSell)
	}

	s.Direction = gctorder.Short
	expectedError = nil
	result, err = cannotPurchase(s, o)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	if result.Direction != gctorder.CouldNotShort {
		t.Errorf("received '%v' expected '%v'", result.Direction, gctorder.CouldNotShort)
	}

	s.Direction = gctorder.Long
	expectedError = nil
	result, err = cannotPurchase(s, o)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	if result.Direction != gctorder.CouldNotLong {
		t.Errorf("received '%v' expected '%v'", result.Direction, gctorder.CouldNotLong)
	}

	s.Direction = gctorder.UnknownSide
	expectedError = nil
	result, err = cannotPurchase(s, o)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	if result.Direction != gctorder.DoNothing {
		t.Errorf("received '%v' expected '%v'", result.Direction, gctorder.DoNothing)
	}
}

func TestCreateLiquidationOrdersForExchange(t *testing.T) {
	t.Parallel()

	p := &Portfolio{}
	var expectedError = common.ErrNilEvent
	_, err := p.CreateLiquidationOrdersForExchange(nil, nil)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}

	b := &event.Base{}

	ev := &kline.Kline{
		Base: b,
	}
	expectedError = gctcommon.ErrNilPointer
	_, err = p.CreateLiquidationOrdersForExchange(ev, nil)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}

	funds := &funding.FundManager{}
	expectedError = config.ErrExchangeNotFound
	_, err = p.CreateLiquidationOrdersForExchange(ev, funds)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}

	ff := &binance.Binance{}
	ff.Name = testExchange
	cp := currency.NewPair(currency.BTC, currency.USDT)
	expectedError = nil
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Futures, Pair: cp})
	if !errors.Is(err, gctcommon.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNotYetImplemented)
	}
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: cp})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	ev.Exchange = ff.Name
	_, err = p.CreateLiquidationOrdersForExchange(ev, funds)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}

	_, err = p.getSettings(ff.Name, asset.Futures, cp)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Fatalf("received '%v' expected '%v'", err, errNoPortfolioSettings)
	}

	od := &gctorder.Detail{
		Exchange:  ff.Name,
		AssetType: asset.Futures,
		Pair:      cp,
		Side:      gctorder.Long,
		OrderID:   "lol",
		Date:      time.Now(),
		Amount:    1337,
		Price:     1337,
	}

	mpt, err := gctorder.SetupMultiPositionTracker(&gctorder.MultiPositionTrackerSetup{
		Exchange:           testExchange,
		Asset:              od.AssetType,
		Pair:               cp,
		Underlying:         currency.USDT,
		CollateralCurrency: currency.USDT,
		OfflineCalculation: true,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	settings := &Settings{
		FuturesTracker: mpt,
	}

	err = settings.FuturesTracker.TrackNewOrder(od)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v', expected '%v'", err, expectedError)
	}
	p.exchangeAssetPairPortfolioSettings = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange] = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange][ev.AssetType] = make(map[*currency.Item]map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange][ev.AssetType][ev.Pair().Base.Item] = make(map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange][ev.AssetType][ev.Pair().Base.Item][ev.Pair().Quote.Item] = settings

	ev.Exchange = ff.Name
	ev.AssetType = asset.Futures
	ev.CurrencyPair = cp
	_, err = p.CreateLiquidationOrdersForExchange(ev, funds)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}

	// spot order
	item, err := funding.CreateItem(ff.Name, asset.Spot, currency.BTC, decimal.Zero, decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	err = funds.AddItem(item)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	err = item.IncreaseAvailable(leet)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	orders, err := p.CreateLiquidationOrdersForExchange(ev, funds)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	if len(orders) != 0 {
		t.Errorf("expected two orders generated, received '%v'", len(orders))
	}
}

func TestGetPositionStatus(t *testing.T) {
	t.Parallel()
	p := PNLSummary{
		Result: gctorder.PNLResult{
			Status: gctorder.Rejected,
		},
	}
	status := p.GetPositionStatus()
	if gctorder.Rejected != status {
		t.Errorf("expected '%v' received '%v'", gctorder.Rejected, status)
	}
}

func TestCheckLiquidationStatus(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	var expectedError = common.ErrNilEvent
	err := p.CheckLiquidationStatus(nil, nil, nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v', expected '%v'", err, expectedError)
	}

	ev := &kline.Kline{
		Base: &event.Base{},
	}
	expectedError = gctcommon.ErrNilPointer
	err = p.CheckLiquidationStatus(ev, nil, nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v', expected '%v'", err, expectedError)
	}

	item := asset.Futures
	pair := currency.NewPair(currency.BTC, currency.USDT)
	expectedError = nil
	contract, err := funding.CreateItem(testExchange, item, pair.Base, decimal.NewFromInt(100), decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}
	collateral, err := funding.CreateItem(testExchange, item, pair.Quote, decimal.NewFromInt(100), decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}
	collat, err := funding.CreateCollateral(contract, collateral)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}

	expectedError = gctcommon.ErrNilPointer
	err = p.CheckLiquidationStatus(ev, collat, nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v', expected '%v'", err, expectedError)
	}

	pnl := &PNLSummary{}
	expectedError = gctorder.ErrNotFuturesAsset
	err = p.CheckLiquidationStatus(ev, collat, pnl)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v', expected '%v'", err, expectedError)
	}

	pnl.Asset = asset.Futures
	ev.AssetType = asset.Futures
	ev.Exchange = testExchange
	ev.CurrencyPair = pair
	exch := &binance.Binance{}
	exch.Name = ev.Exchange
	expectedError = nil
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: exch, Asset: asset.Futures, Pair: pair})
	if !errors.Is(err, gctcommon.ErrNotYetImplemented) {
		t.Errorf("received '%v', expected '%v'", err, gctcommon.ErrNotYetImplemented)
	}
	_, err = p.getSettings(ev.Exchange, ev.AssetType, ev.Pair())
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received '%v', expected '%v'", err, errNoPortfolioSettings)
	}
	od := &gctorder.Detail{
		Price:     1336,
		Amount:    20,
		Exchange:  exch.Name,
		Side:      gctorder.Short,
		AssetType: ev.AssetType,
		Date:      time.Now(),
		Pair:      pair,
		OrderID:   "lol",
	}
	mpt, err := gctorder.SetupMultiPositionTracker(&gctorder.MultiPositionTrackerSetup{
		Exchange:           testExchange,
		Asset:              ev.AssetType,
		Pair:               ev.Pair(),
		Underlying:         currency.USDT,
		CollateralCurrency: currency.USDT,
		OfflineCalculation: true,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	settings := &Settings{
		FuturesTracker: mpt,
	}

	err = settings.FuturesTracker.TrackNewOrder(od)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v', expected '%v'", err, expectedError)
	}
	p.exchangeAssetPairPortfolioSettings = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange] = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange][ev.AssetType] = make(map[*currency.Item]map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange][ev.AssetType][pair.Base.Item] = make(map[*currency.Item]*Settings)
	p.exchangeAssetPairPortfolioSettings[testExchange][ev.AssetType][pair.Base.Item][pair.Quote.Item] = settings
	err = p.CheckLiquidationStatus(ev, collat, pnl)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v', expected '%v'", err, expectedError)
	}
}

func TestSetHoldingsForEvent(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	err := p.SetHoldingsForEvent(nil, nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v', expected '%v'", err, gctcommon.ErrNilPointer)
	}

	item, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	cp, err := funding.CreatePair(item, item)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	err = p.SetHoldingsForEvent(cp.FundReader(), nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received '%v', expected '%v'", err, common.ErrNilEvent)
	}

	err = p.SetHoldingsForEvent(cp.FundReader(), nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received '%v', expected '%v'", err, common.ErrNilEvent)
	}

	tt := time.Now()
	ev := &signal.Signal{
		Base: &event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewPair(currency.BTC, currency.BTC),
			AssetType:    asset.Spot,
			Time:         tt,
		},
	}
	f := &binance.Binance{}
	f.SetDefaults()
	err = p.SetCurrencySettingsMap(&exchange.Settings{
		Exchange: f,
		Pair:     ev.Pair(),
		Asset:    ev.AssetType,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	err = p.SetHoldingsForEvent(cp.FundReader(), ev)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}

	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Item:      currency.BTC,
		Pair:      ev.Pair(),
		Asset:     ev.AssetType,
		Exchange:  ev.Exchange,
		Timestamp: tt,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
	err = p.SetHoldingsForEvent(cp.FundReader(), ev)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected '%v'", err, nil)
	}
}
