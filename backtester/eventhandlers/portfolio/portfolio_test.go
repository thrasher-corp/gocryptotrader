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
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "binance"

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
	err := p.SetupCurrencySettingsMap(nil, nil)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	err = p.SetupCurrencySettingsMap(&exchange.Settings{}, nil)
	if !errors.Is(err, errExchangeUnset) {
		t.Errorf("received: %v, expected: %v", err, errExchangeUnset)
	}

	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: "hi"}, nil)
	if !errors.Is(err, errAssetUnset) {
		t.Errorf("received: %v, expected: %v", err, errAssetUnset)
	}

	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: "hi", Asset: asset.Spot}, nil)
	if !errors.Is(err, errCurrencyPairUnset) {
		t.Errorf("received: %v, expected: %v", err, errCurrencyPairUnset)
	}

	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: "hi", Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)}, nil)
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

	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: testExchange, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)}, nil)
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

	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: testExchange, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)}, nil)
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

	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: testExchange, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)}, nil)
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
	if err != nil {
		t.Fatal(err)
	}
	err = p.UpdateHoldings(&kline.Kline{}, pair)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received '%v' expected '%v'", err, errNoPortfolioSettings)
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

	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: testExchange, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)}, nil)
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

	err := p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: "hi", Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)}, nil)
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
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: "hi", Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)}, nil)
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
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}

	err = p.addComplianceSnapshot(&fill.Fill{})
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: "hi", Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)}, nil)
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
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
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
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: "hi", Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)}, nil)
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
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: "hi", Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)}, nil)
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
	err = p.setHoldingsForOffset(&holdings.Holding{
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		Timestamp: time.Now(),
		QuoteSize: decimal.NewFromInt(1337)}, false)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: testExchange, Asset: asset.Spot, Pair: currency.NewPair(currency.BTC, currency.USD)}, nil)
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
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: "exch", Asset: asset.Spot, Pair: cp}, nil)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	tt := time.Now()
	s, ok := p.exchangeAssetPairSettings["exch"][asset.Spot][cp]
	if !ok {
		t.Fatal("couldn't get settings")
	}
	err = s.ComplianceManager.AddSnapshot([]compliance.SnapshotOrder{
		{
			SpotOrder: &gctorder.Detail{
				Exchange:  "exch",
				AssetType: asset.Spot,
				Pair:      cp,
				Amount:    1337,
			},
		},
	}, tt, 0, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	e := &kline.Kline{
		Base: event.Base{
			Exchange:     "exch",
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
	if ss.Orders[0].SpotOrder.Amount != 1337 {
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
	err = p.SetupCurrencySettingsMap(&exchange.Settings{Exchange: "exch", Asset: asset.Spot, Pair: currency.NewPair(currency.XRP, currency.DOGE)}, nil)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	tt := time.Now()
	s, ok := p.exchangeAssetPairSettings["exch"][asset.Spot][cp]
	if !ok {
		t.Fatal("couldn't get settings")
	}
	err = s.ComplianceManager.AddSnapshot([]compliance.SnapshotOrder{
		{
			SpotOrder: &gctorder.Detail{
				Exchange:  "exch",
				AssetType: asset.Spot,
				Pair:      cp,
				Amount:    1337,
			},
		},
	}, tt, 0, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	ss, err := p.GetLatestOrderSnapshots()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = s.ComplianceManager.AddSnapshot([]compliance.SnapshotOrder{
		ss[0].Orders[0],
		{
			SpotOrder: &gctorder.Detail{
				Exchange:  "exch",
				AssetType: asset.Spot,
				Pair:      cp,
				Amount:    1338,
			},
		},
	}, tt, 1, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	ss, err = p.GetLatestOrderSnapshots()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(ss) != 1 {
		t.Fatal("expected 1")
	}
	if len(ss[0].Orders) != 2 {
		t.Error("expected 2")
	}
}

func TestCalculatePNL(t *testing.T) {
	p := &Portfolio{
		riskFreeRate:              decimal.Decimal{},
		sizeManager:               nil,
		riskManager:               nil,
		exchangeAssetPairSettings: nil,
	}

	ev := &kline.Kline{}
	err := p.CalculatePNL(ev, nil)
	if !errors.Is(err, errNoPortfolioSettings) {
		t.Errorf("received: %v, expected: %v", err, errNoPortfolioSettings)
	}

	exch := "binance"
	a := asset.Futures
	pair, _ := currency.NewPairFromStrings("BTC", "1231")
	err = p.SetupCurrencySettingsMap(&exchange.Settings{
		Exchange:      exch,
		UseRealOrders: false,
		Pair:          pair,
		Asset:         a,
	}, nil)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	tt := time.Now()
	tt0 := time.Now().Add(-time.Hour)
	ev.Exchange = exch
	ev.AssetType = a
	ev.CurrencyPair = pair
	ev.Time = tt0

	err = p.CalculatePNL(ev, nil)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	futuresOrder := &gctorder.FuturesTracker{
		CurrentDirection: gctorder.Short,
		ShortPositions: &gctorder.Detail{
			Price:     1336,
			Amount:    20,
			Exchange:  exch,
			Side:      gctorder.Short,
			AssetType: asset.Futures,
			Date:      tt0,
			Pair:      pair,
		},
	}
	s, ok := p.exchangeAssetPairSettings["exch"][asset.Spot][pair]
	if !ok {
		t.Fatal("couldn't get settings")
	}
	ev.Close = decimal.NewFromInt(1337)
	err = s.ComplianceManager.AddSnapshot([]compliance.SnapshotOrder{
		{
			ClosePrice:   decimal.NewFromInt(1336),
			FuturesOrder: futuresOrder,
		},
	}, tt0, 0, false)
	err = p.CalculatePNL(ev, nil)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	if len(futuresOrder.PNLHistory) == 0 {
		t.Error("expected a pnl entry ( ͡° ͜ʖ ͡°)")
	}

	if !futuresOrder.UnrealisedPNL.Equal(decimal.NewFromInt(20)) {
		// 20 orders * $1 difference * 1x leverage
		t.Error("expected 20")
	}

	err = s.ComplianceManager.AddSnapshot([]compliance.SnapshotOrder{
		{
			ClosePrice: decimal.NewFromInt(1336),
			SpotOrder:  futuresOrder.ShortPositions,
		},
	}, tt, 1, false)
	err = p.CalculatePNL(ev, nil)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	// coverage of logic
	futuresOrder.LongPositions = futuresOrder.ShortPositions

	err = s.ComplianceManager.AddSnapshot([]compliance.SnapshotOrder{
		{
			ClosePrice:   decimal.NewFromInt(1336),
			FuturesOrder: futuresOrder,
		},
	}, tt.Add(time.Hour), 2, false)
	err = p.CalculatePNL(ev, nil)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}
