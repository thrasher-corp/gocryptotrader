package portfolio

import (
	"errors"
	"strings"
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
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ftx"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "ftx"

func TestReset(t *testing.T) {
	t.Parallel()
	p := Portfolio{
		exchangeAssetPairSettings: make(map[string]map[asset.Item]map[currency.Pair]*Settings),
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
	err := p.SetupCurrencySettingsMap(nil)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	err = p.SetupCurrencySettingsMap(&exchange.Settings{})
	if !errors.Is(err, errExchangeUnset) {
		t.Errorf("received: %v, expected: %v", err, errExchangeUnset)
	}

	ff := &ftx.FTX{}
	ff.Name = testExchange
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff})
	if !errors.Is(err, errAssetUnset) {
		t.Errorf("received: %v, expected: %v", err, errAssetUnset)
	}

	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot})
	if !errors.Is(err, errCurrencyPairUnset) {
		t.Errorf("received: %v, expected: %v", err, errCurrencyPairUnset)
	}

	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)})
	if err != nil {
		t.Error(err)
	}
}

func TestSetHoldings(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}

	err := p.setHoldingsForOffset(&holdings.Holding{}, false)
	if !errors.Is(err, errHoldingsNoTimestamp) {
		t.Errorf("received: %v, expected: %v", err, errHoldingsNoTimestamp)
	}
	tt := time.Now()

	err = p.setHoldingsForOffset(&holdings.Holding{Timestamp: tt}, false)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	ff := &ftx.FTX{}
	ff.Name = testExchange
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)})
	if err != nil {
		t.Error(err)
	}
	err = p.setHoldingsForOffset(&holdings.Holding{
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		Timestamp: tt}, false)
	if err != nil {
		t.Error(err)
	}

	err = p.setHoldingsForOffset(&holdings.Holding{
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		Timestamp: tt}, true)
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
	err := p.setHoldingsForOffset(&holdings.Holding{
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		Timestamp: tt}, true)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	ff := &ftx.FTX{}
	ff.Name = testExchange
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)})
	if err != nil {
		t.Error(err)
	}
	h = p.GetLatestHoldingsForAllCurrencies()
	if len(h) != 0 {
		t.Errorf("received %v, expected %v", len(h), 0)
	}
	err = p.setHoldingsForOffset(&holdings.Holding{
		Offset:    1,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		Timestamp: tt}, false)
	if err != nil {
		t.Error(err)
	}
	h = p.GetLatestHoldingsForAllCurrencies()
	if len(h) != 1 {
		t.Errorf("received %v, expected %v", len(h), 1)
	}
	err = p.setHoldingsForOffset(&holdings.Holding{
		Offset:    1,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		Timestamp: tt}, false)
	if !errors.Is(err, errHoldingsAlreadySet) {
		t.Errorf("received: %v, expected: %v", err, errHoldingsAlreadySet)
	}
	err = p.setHoldingsForOffset(&holdings.Holding{
		Offset:    1,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		Timestamp: tt}, true)
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
		Base: event.Base{
			Time:         tt,
			Exchange:     testExchange,
			AssetType:    asset.Spot,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
		},
	}
	_, err := p.ViewHoldingAtTimePeriod(s)
	if !errors.Is(err, errNoHoldings) {
		t.Errorf("received: %v, expected: %v", err, errNoHoldings)
	}

	ff := &ftx.FTX{}
	ff.Name = testExchange
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)})
	if err != nil {
		t.Error(err)
	}

	err = p.setHoldingsForOffset(&holdings.Holding{
		Offset:    1,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		Timestamp: tt}, false)
	if err != nil {
		t.Error(err)
	}
	err = p.setHoldingsForOffset(&holdings.Holding{
		Offset:    2,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		Timestamp: tt.Add(time.Hour)}, false)
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
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
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
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = p.UpdateHoldings(&kline.Kline{}, pair)
	if !errors.Is(err, errExchangeUnset) {
		t.Errorf("received '%v' expected '%v'", err, errExchangeUnset)
	}

	tt := time.Now()
	err = p.setHoldingsForOffset(&holdings.Holding{
		Offset:    1,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		Timestamp: tt}, false)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	ff := &ftx.FTX{}
	ff.Name = testExchange
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)})
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
	f := p.GetFee("", asset.Empty, currency.EMPTYPAIR)
	if !f.IsZero() {
		t.Error("expected 0")
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	ff := &ftx.FTX{}
	ff.Name = testExchange
	err := p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: cp})
	if err != nil {
		t.Error(err)
	}

	p.exchangeAssetPairSettings[testExchange][asset.Spot][cp].Fee = decimal.NewFromInt(1337)
	f = p.GetFee(testExchange, asset.Spot, cp)
	if !f.Equal(decimal.NewFromInt(1337)) {
		t.Errorf("expected %v received %v", 1337, f)
	}
}

func TestGetComplianceManager(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	_, err := p.GetComplianceManager("", asset.Empty, currency.EMPTYPAIR)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	ff := &ftx.FTX{}
	ff.Name = testExchange
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)})
	if err != nil {
		t.Error(err)
	}
	var cm *compliance.Manager
	cm, err = p.GetComplianceManager(testExchange, asset.Spot, currency.NewPair(currency.BTC, currency.USD))
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
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}

	err = p.addComplianceSnapshot(&fill.Fill{})
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	ff := &ftx.FTX{}
	ff.Name = testExchange
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)})
	if err != nil {
		t.Error(err)
	}

	err = p.addComplianceSnapshot(&fill.Fill{
		Base: event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
			AssetType:    asset.Spot,
		},
		Order: &gctorder.Detail{
			Exchange:  testExchange,
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
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}

	f := &fill.Fill{
		Base: event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
			AssetType:    asset.Spot,
		},
		Order: &gctorder.Detail{
			Exchange:  testExchange,
			Pair:      currency.NewPair(currency.BTC, currency.USD),
			AssetType: asset.Spot,
		},
	}
	_, err = p.OnFill(f, nil)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}
	ff := &ftx.FTX{}
	ff.Name = testExchange
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)})
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
		t.Errorf("received: %v, expected: %v", err, nil)
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
		t.Errorf("received: %v, expected: %v", err, errInvalidDirection)
	}

	s.Direction = gctorder.Buy
	_, err = p.OnSignal(s, &exchange.Settings{}, pair)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}
	ff := &ftx.FTX{}
	ff.Name = testExchange
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)})
	if err != nil {
		t.Error(err)
	}
	s = &signal.Signal{
		Base: event.Base{
			Exchange:     testExchange,
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
	err = p.setHoldingsForOffset(&holdings.Holding{
		Exchange:  "lol",
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		Timestamp: time.Now(),
		QuoteSize: decimal.NewFromInt(1337)}, false)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)})
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
	s.Amount = decimal.NewFromInt(1)
	resp, err = p.OnSignal(s, &exchange.Settings{}, pair)
	if err != nil {
		t.Error(err)
	}
	if resp.Amount.IsZero() {
		t.Error("expected an amount to be sized")
	}
}

func TestGetLatestHoldings(t *testing.T) {
	t.Parallel()
	cs := Settings{}
	h := cs.GetLatestHoldings()
	if !h.Timestamp.IsZero() {
		t.Error("expected unset holdings")
	}
	tt := time.Now()
	cs.HoldingsSnapshots = append(cs.HoldingsSnapshots, holdings.Holding{Timestamp: tt})

	h = cs.GetLatestHoldings()
	if !h.Timestamp.Equal(tt) {
		t.Errorf("expected %v, received %v", tt, h.Timestamp)
	}
}

func TestGetSnapshotAtTime(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	_, err := p.GetLatestOrderSnapshotForEvent(&kline.Kline{})
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}
	cp := currency.NewPair(currency.XRP, currency.DOGE)
	ff := &ftx.FTX{}
	ff.Name = testExchange
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: cp})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	tt := time.Now()
	s, ok := p.exchangeAssetPairSettings[testExchange][asset.Spot][cp]
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
	e := &kline.Kline{
		Base: event.Base{
			Exchange:     testExchange,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: cp,
			AssetType:    asset.Spot,
		},
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
	ff := &ftx.FTX{}
	ff.Name = testExchange
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.XRP, currency.DOGE)})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	s, ok := p.exchangeAssetPairSettings[testExchange][asset.Spot][cp]
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
	ev := &kline.Kline{}
	err := p.UpdatePNL(ev, decimal.Zero)
	if !errors.Is(err, gctorder.ErrNotFuturesAsset) {
		t.Errorf("received: %v, expected: %v", err, gctorder.ErrNotFuturesAsset)
	}

	exch := &ftx.FTX{}
	exch.Name = testExchange
	a := asset.Futures
	pair, err := currency.NewPairFromStrings("BTC", "1231")
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = p.SetupCurrencySettingsMap(&exchange.Settings{
		Exchange:      exch,
		UseRealOrders: false,
		Pair:          pair,
		Asset:         a,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	tt := time.Now().Add(time.Hour)
	tt0 := time.Now().Add(-time.Hour)
	ev.Exchange = exch.Name
	ev.AssetType = a
	ev.CurrencyPair = pair
	ev.Time = tt0

	err = p.UpdatePNL(ev, decimal.Zero)
	if !errors.Is(err, gctorder.ErrPositionsNotLoadedForPair) {
		t.Errorf("received: %v, expected: %v", err, gctorder.ErrPositionsNotLoadedForPair)
	}

	od := &gctorder.Detail{
		Price:     1336,
		Amount:    20,
		Exchange:  exch.Name,
		Side:      gctorder.Short,
		AssetType: a,
		Date:      tt0,
		Pair:      pair,
		ID:        "lol",
	}

	s, ok := p.exchangeAssetPairSettings[strings.ToLower(exch.Name)][a][pair]
	if !ok {
		t.Fatal("couldn't get settings")
	}
	ev.Close = decimal.NewFromInt(1337)
	err = s.ComplianceManager.AddSnapshot(&compliance.Snapshot{
		Offset:    0,
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
	var expectedError = common.ErrNilEvent
	_, err := p.TrackFuturesOrder(nil, nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}
	expectedError = common.ErrNilArguments
	_, err = p.TrackFuturesOrder(&fill.Fill{}, nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}
	fundPair := &funding.SpotPair{}
	expectedError = gctorder.ErrSubmissionIsNil
	_, err = p.TrackFuturesOrder(&fill.Fill{}, fundPair)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}

	expectedError = gctorder.ErrNotFuturesAsset
	od := &gctorder.Detail{}
	_, err = p.TrackFuturesOrder(&fill.Fill{
		Order: od,
	}, fundPair)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}

	od.AssetType = asset.Futures
	expectedError = funding.ErrNotCollateral
	_, err = p.TrackFuturesOrder(&fill.Fill{
		Order: od,
	}, fundPair)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}

	expectedError = nil
	contract, err := funding.CreateItem(od.Exchange, od.AssetType, od.Pair.Base, decimal.NewFromInt(100), decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}
	collateral, err := funding.CreateItem(od.Exchange, od.AssetType, od.Pair.Quote, decimal.NewFromInt(100), decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}
	collat, err := funding.CreateCollateral(contract, collateral)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}
	expectedError = errExchangeUnset
	_, err = p.TrackFuturesOrder(&fill.Fill{
		Order: od,
	}, collat)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}

	cp := currency.NewPair(currency.XRP, currency.DOGE)
	ff := &ftx.FTX{}
	ff.Name = testExchange
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Futures, Pair: cp})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	od.Pair = cp
	od.Exchange = testExchange
	od.Side = gctorder.Short
	od.AssetType = asset.Futures
	od.Amount = 1337
	od.Price = 1337
	od.ID = testExchange
	od.Date = time.Now()
	expectedError = nil

	_, err = p.TrackFuturesOrder(&fill.Fill{
		Order: od,
		Base: event.Base{
			Exchange:     testExchange,
			AssetType:    asset.Futures,
			CurrencyPair: cp,
		},
	}, collat)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v", err, expectedError)
	}
}

func TestGetHoldingsForTime(t *testing.T) {
	t.Parallel()
	s := &Settings{}
	h := s.GetHoldingsForTime(time.Now())
	if !h.Timestamp.IsZero() {
		t.Error("expected unset holdings")
	}
	tt := time.Now()
	s.HoldingsSnapshots = append(s.HoldingsSnapshots, holdings.Holding{
		Timestamp: tt,
		Offset:    1337,
	})
	h = s.GetHoldingsForTime(time.Unix(1337, 0))
	if !h.Timestamp.IsZero() {
		t.Error("expected unset holdings")
	}

	h = s.GetHoldingsForTime(tt)
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
		Base: event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
			AssetType:    asset.Futures,
		},
	}
	ff := &ftx.FTX{}
	ff.Name = testExchange
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: ev.AssetType, Pair: ev.Pair()})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	expectedError = nil
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
		Base: event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
			AssetType:    asset.Futures,
		},
	}
	ff := &ftx.FTX{}
	ff.Name = testExchange
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: ev.AssetType, Pair: ev.Pair()})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	expectedError = gctorder.ErrPositionsNotLoadedForPair
	_, err = p.GetLatestPNLForEvent(ev)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}

	settings, ok := p.exchangeAssetPairSettings[ev.GetExchange()][ev.GetAssetType()][ev.Pair()]
	if !ok {
		t.Fatalf("where did settings go?")
	}
	expectedError = nil
	err = settings.FuturesTracker.TrackNewOrder(&gctorder.Detail{
		Exchange:  ev.GetExchange(),
		AssetType: ev.AssetType,
		Pair:      ev.Pair(),
		Amount:    1,
		Price:     1,
		ID:        "one",
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
	var expectedError = common.ErrNilEvent
	_, err := p.getFuturesSettingsFromEvent(nil)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	expectedError = gctorder.ErrNotFuturesAsset
	_, err = p.getFuturesSettingsFromEvent(&fill.Fill{})
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	ev := &fill.Fill{
		Base: event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
			AssetType:    asset.Futures,
		},
	}
	expectedError = errExchangeUnset
	_, err = p.getFuturesSettingsFromEvent(ev)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}

	ff := &ftx.FTX{}
	ff.Name = testExchange
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: ev.AssetType, Pair: ev.Pair()})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	expectedError = nil
	settings, err := p.getFuturesSettingsFromEvent(ev)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}

	expectedError = errUnsetFuturesTracker
	settings.FuturesTracker = nil
	_, err = p.getFuturesSettingsFromEvent(ev)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
}

func TestGetLatestPNLs(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	latest := p.GetLatestPNLs()
	if len(latest) != 0 {
		t.Error("expected empty")
	}
	ev := &fill.Fill{
		Base: event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
			AssetType:    asset.Futures,
		},
	}
	ff := &ftx.FTX{}
	ff.Name = testExchange
	err := p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: ev.AssetType, Pair: ev.Pair()})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	settings, ok := p.exchangeAssetPairSettings[ev.GetExchange()][ev.GetAssetType()][ev.Pair()]
	if !ok {
		t.Fatalf("where did settings go?")
	}
	err = settings.FuturesTracker.TrackNewOrder(&gctorder.Detail{
		Exchange:  ev.GetExchange(),
		AssetType: ev.AssetType,
		Pair:      ev.Pair(),
		Amount:    1,
		Price:     1,
		ID:        "one",
		Date:      time.Now(),
		Side:      gctorder.Buy,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}
	latest = p.GetLatestPNLs()
	if len(latest) != 1 {
		t.Error("expected 1")
	}
}

func TestGetUnrealisedPNL(t *testing.T) {
	t.Parallel()
	p := PNLSummary{
		Exchange:           testExchange,
		Item:               asset.Futures,
		Pair:               currency.NewPair(currency.BTC, currency.USDT),
		CollateralCurrency: currency.USD,
		Offset:             1,
		Result: gctorder.PNLResult{
			Time:                  time.Now(),
			UnrealisedPNL:         decimal.NewFromInt(1337),
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
		Item:               asset.Futures,
		Pair:               currency.NewPair(currency.BTC, currency.USDT),
		CollateralCurrency: currency.USD,
		Offset:             1,
		Result: gctorder.PNLResult{
			Time:                  time.Now(),
			UnrealisedPNL:         decimal.NewFromInt(1337),
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
		Item:               asset.Futures,
		Pair:               currency.NewPair(currency.BTC, currency.USDT),
		CollateralCurrency: currency.USD,
		Offset:             1,
		Result: gctorder.PNLResult{
			Time:                  time.Now(),
			UnrealisedPNL:         decimal.NewFromInt(1337),
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
		Item:               asset.Futures,
		Pair:               currency.NewPair(currency.BTC, currency.USDT),
		CollateralCurrency: currency.USD,
		Offset:             1,
		Result: gctorder.PNLResult{
			Time:                  time.Now(),
			UnrealisedPNL:         decimal.NewFromInt(1337),
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
		Item:               asset.Futures,
		Pair:               currency.NewPair(currency.BTC, currency.USDT),
		CollateralCurrency: currency.USD,
		Offset:             1,
		Result: gctorder.PNLResult{
			Time:                  time.Now(),
			UnrealisedPNL:         decimal.NewFromInt(1337),
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

	s := &signal.Signal{}
	expectedError = common.ErrNilArguments
	_, err = cannotPurchase(s, nil)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}

	o := &order.Order{}
	s.Direction = gctorder.Buy
	expectedError = nil
	result, err := cannotPurchase(s, o)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	if result.Direction != common.CouldNotBuy {
		t.Errorf("received '%v' expected '%v'", result.Direction, common.CouldNotBuy)
	}

	s.Direction = gctorder.Sell
	expectedError = nil
	result, err = cannotPurchase(s, o)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	if result.Direction != common.CouldNotSell {
		t.Errorf("received '%v' expected '%v'", result.Direction, common.CouldNotSell)
	}

	s.Direction = gctorder.Short
	expectedError = nil
	result, err = cannotPurchase(s, o)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	if result.Direction != common.CouldNotShort {
		t.Errorf("received '%v' expected '%v'", result.Direction, common.CouldNotShort)
	}

	s.Direction = gctorder.Long
	expectedError = nil
	result, err = cannotPurchase(s, o)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	if result.Direction != common.CouldNotLong {
		t.Errorf("received '%v' expected '%v'", result.Direction, common.CouldNotLong)
	}

	s.Direction = gctorder.UnknownSide
	expectedError = nil
	result, err = cannotPurchase(s, o)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	if result.Direction != common.DoNothing {
		t.Errorf("received '%v' expected '%v'", result.Direction, common.DoNothing)
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

	ev := &kline.Kline{}
	expectedError = common.ErrNilArguments
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

	ff := &ftx.FTX{}
	ff.Name = testExchange
	cp := currency.NewPair(currency.BTC, currency.USD)
	expectedError = nil
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Futures, Pair: cp})
	if err != nil {
		t.Error(err)
	}
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: cp})
	if err != nil {
		t.Error(err)
	}
	ev.Exchange = testExchange
	_, err = p.CreateLiquidationOrdersForExchange(ev, funds)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}

	settings, err := p.getSettings(ff.Name, asset.Futures, cp)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}

	err = settings.FuturesTracker.TrackNewOrder(&gctorder.Detail{
		Exchange:  ff.Name,
		AssetType: asset.Futures,
		Pair:      cp,
		Side:      gctorder.Long,
		ID:        "lol",
		Date:      time.Now(),
		Amount:    1337,
		Price:     1337,
	})
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
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
	item.IncreaseAvailable(decimal.NewFromInt(1337))
	orders, err := p.CreateLiquidationOrdersForExchange(ev, funds)
	if !errors.Is(err, expectedError) {
		t.Fatalf("received '%v' expected '%v'", err, expectedError)
	}
	if len(orders) != 2 {
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

	ev := &kline.Kline{}
	expectedError = common.ErrNilArguments
	err = p.CheckLiquidationStatus(ev, nil, nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v', expected '%v'", err, expectedError)
	}

	item := asset.Futures
	pair := currency.NewPair(currency.BTC, currency.USD)
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

	expectedError = common.ErrNilArguments
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

	pnl.Item = asset.Futures
	ev.AssetType = asset.Futures
	ev.Exchange = "ftx"
	ev.CurrencyPair = pair
	exch := &ftx.FTX{}
	exch.Name = testExchange
	expectedError = nil
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: exch, Asset: asset.Futures, Pair: pair})
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v', expected '%v'", err, expectedError)
	}
	settings, err := p.getSettings(testExchange, ev.AssetType, ev.Pair())
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v', expected '%v'", err, expectedError)
	}
	od := &gctorder.Detail{
		Price:     1336,
		Amount:    20,
		Exchange:  exch.Name,
		Side:      gctorder.Short,
		AssetType: ev.AssetType,
		Date:      time.Now(),
		Pair:      pair,
		ID:        "lol",
	}
	err = settings.FuturesTracker.TrackNewOrder(od)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v', expected '%v'", err, expectedError)
	}
	err = p.CheckLiquidationStatus(ev, collat, pnl)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v', expected '%v'", err, expectedError)
	}
}
