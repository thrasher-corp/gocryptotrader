package portfolio

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "binance"

var leet = decimal.NewFromInt(1337)

func TestReset(t *testing.T) {
	t.Parallel()
	p := &Portfolio{
		exchangeAssetPairPortfolioSettings: make(map[key.ExchangeAssetPair]*Settings),
	}
	err := p.Reset()
	assert.NoError(t, err)

	if p.exchangeAssetPairPortfolioSettings == nil {
		t.Error("expected a map")
	}

	p = nil
	err = p.Reset()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestSetup(t *testing.T) {
	t.Parallel()
	_, err := Setup(nil, nil, decimal.NewFromInt(-1))
	assert.ErrorIs(t, err, errSizeManagerUnset)

	_, err = Setup(&size.Size{}, nil, decimal.NewFromInt(-1))
	assert.ErrorIs(t, err, errNegativeRiskFreeRate)

	_, err = Setup(&size.Size{}, nil, decimal.NewFromInt(1))
	assert.ErrorIs(t, err, errRiskManagerUnset)

	var p *Portfolio
	p, err = Setup(&size.Size{}, &risk.Risk{}, decimal.NewFromInt(1))
	assert.NoError(t, err)

	if !p.riskFreeRate.Equal(decimal.NewFromInt(1)) {
		t.Error("expected 1")
	}
}

func TestSetupCurrencySettingsMap(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	err := p.SetCurrencySettingsMap(nil)
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	err = p.SetCurrencySettingsMap(&exchange.Settings{})
	assert.ErrorIs(t, err, errExchangeUnset)

	ff := &binance.Exchange{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff})
	assert.ErrorIs(t, err, errAssetUnset)

	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot})
	assert.ErrorIs(t, err, errCurrencyPairUnset)

	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewBTCUSDT()})
	assert.NoError(t, err)
}

func TestSetHoldings(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}

	err := p.SetHoldingsForTimestamp(&holdings.Holding{})
	assert.ErrorIs(t, err, errHoldingsNoTimestamp)

	tt := time.Now()

	err = p.SetHoldingsForTimestamp(&holdings.Holding{Timestamp: tt})
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	ff := &binance.Exchange{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewBTCUSDT()})
	assert.NoError(t, err)

	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSDT(),
		Timestamp: tt,
	})
	assert.NoError(t, err)

	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSDT(),
		Timestamp: tt,
	})
	assert.NoError(t, err)
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
		Pair:      currency.NewBTCUSDT(),
		Timestamp: tt,
	})
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	ff := &binance.Exchange{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewBTCUSDT()})
	assert.NoError(t, err)

	h = p.GetLatestHoldingsForAllCurrencies()
	if len(h) != 0 {
		t.Errorf("received %v, expected %v", len(h), 0)
	}
	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Offset:    1,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSDT(),
		Timestamp: tt,
	})
	assert.NoError(t, err)

	h = p.GetLatestHoldingsForAllCurrencies()
	if len(h) != 1 {
		t.Errorf("received %v, expected %v", len(h), 1)
	}
	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Offset:    1,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSDT(),
		Timestamp: tt,
	})
	assert.NoError(t, err)

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
			CurrencyPair: currency.NewBTCUSDT(),
		},
	}
	_, err := p.ViewHoldingAtTimePeriod(s)
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	ff := &binance.Exchange{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewBTCUSDT()})
	assert.NoError(t, err)

	_, err = p.ViewHoldingAtTimePeriod(s)
	assert.ErrorIs(t, err, errNoHoldings)

	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Offset:    1,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSDT(),
		Timestamp: tt,
	})
	assert.NoError(t, err)

	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Offset:    2,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSDT(),
		Timestamp: tt.Add(time.Hour),
	})
	assert.NoError(t, err)

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
	assert.ErrorIs(t, err, common.ErrNilEvent)

	err = p.UpdateHoldings(&kline.Kline{}, nil)
	assert.ErrorIs(t, err, funding.ErrFundsNotFound)

	bc, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.NewFromInt(1), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	qc, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, decimal.NewFromInt(100), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := funding.CreatePair(bc, qc)
	assert.NoError(t, err)

	b := &event.Base{}
	err = p.UpdateHoldings(&kline.Kline{
		Base: b,
	}, pair)
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	tt := time.Now()
	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Offset:    1,
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSDT(),
		Timestamp: tt,
	})
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	ff := &binance.Exchange{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewBTCUSDT()})
	assert.NoError(t, err)

	b.Time = tt
	b.Exchange = testExchange
	b.CurrencyPair = currency.NewBTCUSDT()
	b.AssetType = asset.Spot
	err = p.UpdateHoldings(&kline.Kline{
		Base: b,
	}, pair)
	assert.NoError(t, err)
}

func TestGetComplianceManager(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	_, err := p.getComplianceManager("", asset.Empty, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	ff := &binance.Exchange{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewBTCUSDT()})
	assert.NoError(t, err)

	var cm *compliance.Manager
	cm, err = p.getComplianceManager(testExchange, asset.Spot, currency.NewBTCUSDT())
	assert.NoError(t, err)

	if cm == nil {
		t.Error("expected not nil")
	}
}

func TestAddComplianceSnapshot(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	err := p.addComplianceSnapshot(nil)
	assert.ErrorIs(t, err, common.ErrNilEvent)

	err = p.addComplianceSnapshot(&fill.Fill{
		Base: &event.Base{},
	})
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	ff := &binance.Exchange{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewBTCUSDT()})
	assert.NoError(t, err)

	err = p.addComplianceSnapshot(&fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewBTCUSDT(),
			AssetType:    asset.Spot,
		},
		Order: &gctorder.Detail{
			Exchange:  testExchange,
			Pair:      currency.NewBTCUSDT(),
			AssetType: asset.Spot,
		},
	})
	assert.NoError(t, err)
}

func TestOnFill(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	_, err := p.OnFill(nil, nil)
	assert.ErrorIs(t, err, common.ErrNilEvent)

	f := &fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewBTCUSDT(),
			AssetType:    asset.Spot,
		},
		Order: &gctorder.Detail{
			Exchange:  testExchange,
			Pair:      currency.NewBTCUSDT(),
			AssetType: asset.Spot,
		},
	}
	_, err = p.OnFill(f, nil)
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	ff := &binance.Exchange{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewBTCUSDT()})
	assert.NoError(t, err)

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
	assert.ErrorIs(t, err, errHoldingsNoTimestamp)

	f.Time = time.Now()
	_, err = p.OnFill(f, pair)
	assert.NoError(t, err)

	f.Direction = gctorder.Buy
	_, err = p.OnFill(f, pair)
	assert.NoError(t, err)
}

func TestOnSignal(t *testing.T) {
	t.Parallel()
	p := Portfolio{}
	_, err := p.OnSignal(nil, nil, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	b := &event.Base{}
	s := &signal.Signal{
		Base: b,
	}
	_, err = p.OnSignal(s, &exchange.Settings{}, nil)
	assert.ErrorIs(t, err, errSizeManagerUnset)

	p.sizeManager = &size.Size{}

	_, err = p.OnSignal(s, &exchange.Settings{}, nil)
	assert.ErrorIs(t, err, errRiskManagerUnset)

	p.riskManager = &risk.Risk{}

	_, err = p.OnSignal(s, &exchange.Settings{}, nil)
	assert.ErrorIs(t, err, funding.ErrFundsNotFound)

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
	assert.ErrorIs(t, err, errInvalidDirection)

	s.Direction = gctorder.Buy
	_, err = p.OnSignal(s, &exchange.Settings{}, funds)
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	ff := &binance.Exchange{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewBTCUSD()})
	assert.NoError(t, err)

	b.Exchange = testExchange
	b.CurrencyPair = currency.NewBTCUSD()
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
	assert.NoError(t, err)

	if len(resp.Reasons) != 4 {
		t.Error("expected issue")
	}

	s.Direction = gctorder.MissingData
	_, err = p.OnSignal(s, &exchange.Settings{}, funds)
	assert.NoError(t, err)

	s.Direction = gctorder.Buy
	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Exchange:  "lol",
		Asset:     asset.Spot,
		Pair:      currency.NewBTCUSD(),
		Timestamp: time.Now(),
		QuoteSize: leet,
	})
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	cs := &exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewBTCUSD()}
	err = p.SetCurrencySettingsMap(cs)
	assert.NoError(t, err)

	resp, err = p.OnSignal(s, &exchange.Settings{}, funds)
	assert.NoError(t, err)

	if resp.Direction != gctorder.CouldNotBuy {
		t.Errorf("expected common.CouldNotBuy, received %v", resp.Direction)
	}

	s.ClosePrice = decimal.NewFromInt(10)
	s.Direction = gctorder.Buy
	s.Amount = decimal.NewFromInt(1)
	resp, err = p.OnSignal(s, &exchange.Settings{}, funds)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	s.AssetType = asset.Futures
	cs.Asset = asset.Futures

	err = p.SetCurrencySettingsMap(cs)
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)

	s.Direction = gctorder.Long
	_, err = p.OnSignal(s, cs, collateralFunds)
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	cp := currency.NewBTCUSD()
	_, err = p.getSettings(testExchange, asset.Futures, cp)
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	exchangeSettings := &Settings{}
	exchangeSettings.FuturesTracker, err = futures.SetupMultiPositionTracker(&futures.MultiPositionTrackerSetup{
		Exchange:           testExchange,
		Asset:              asset.Futures,
		Pair:               cp,
		Underlying:         currency.USD,
		CollateralCurrency: currency.USD,
		OfflineCalculation: true,
	})
	assert.NoError(t, err)

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
		Pair:          currency.NewBTCUSD(),
	})
	assert.NoError(t, err)

	s.Direction = gctorder.ClosePosition
	_, err = p.OnSignal(s, cs, collateralFunds)
	assert.ErrorIs(t, err, errNoPortfolioSettings)
}

func TestGetLatestHoldings(t *testing.T) {
	t.Parallel()
	s := &Settings{
		HoldingsSnapshots: make(map[int64]*holdings.Holding),
	}
	_, err := s.GetLatestHoldings()
	assert.ErrorIs(t, err, errNoHoldings)

	tt := time.Now()
	s.HoldingsSnapshots[tt.UnixNano()] = &holdings.Holding{Timestamp: tt}

	h, err := s.GetLatestHoldings()
	assert.NoError(t, err)

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
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	cp := currency.NewPair(currency.XRP, currency.DOGE)
	ff := &binance.Exchange{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: cp})
	assert.NoError(t, err)

	tt := time.Now()
	s, ok := p.exchangeAssetPairPortfolioSettings[key.NewExchangeAssetPair(testExchange, asset.Spot, cp)]
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
	assert.NoError(t, err)

	b.Exchange = testExchange
	b.Time = tt
	b.Interval = gctkline.OneDay
	b.CurrencyPair = cp
	b.AssetType = asset.Spot
	e := &kline.Kline{
		Base: b,
	}

	ss, err := p.GetLatestOrderSnapshotForEvent(e)
	assert.NoError(t, err)

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
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	cp := currency.NewPair(currency.XRP, currency.DOGE)
	ff := &binance.Exchange{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: currency.NewPair(currency.XRP, currency.DOGE)})
	require.NoError(t, err, "SetCurrencySettingsMap must not error")
	s, ok := p.exchangeAssetPairPortfolioSettings[key.NewExchangeAssetPair(testExchange, asset.Spot, cp)]
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
	assert.NoError(t, err)

	_, err = p.GetLatestOrderSnapshots()
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	ss, err := p.GetLatestOrderSnapshots()
	assert.NoError(t, err)

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
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	exch := &binance.Exchange{}
	exch.Name = testExchange
	a := asset.Futures
	pair, err := currency.NewPairFromStrings("BTC", "1231")
	assert.NoError(t, err)

	err = p.SetCurrencySettingsMap(&exchange.Settings{
		Exchange:      exch,
		UseRealOrders: false,
		Pair:          pair,
		Asset:         a,
	})
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)

	tt := time.Now().Add(time.Hour)
	tt0 := time.Now().Add(-time.Hour)
	ev.Exchange = exch.Name
	ev.AssetType = a
	ev.CurrencyPair = pair
	ev.Time = tt0

	err = p.UpdatePNL(ev, decimal.Zero)
	assert.ErrorIs(t, err, errNoPortfolioSettings)

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
	mpt, err := futures.SetupMultiPositionTracker(&futures.MultiPositionTrackerSetup{
		Exchange:           testExchange,
		Asset:              ev.AssetType,
		Pair:               ev.Pair(),
		Underlying:         currency.USDT,
		CollateralCurrency: currency.USDT,
		OfflineCalculation: true,
	})
	assert.NoError(t, err)

	s := &Settings{
		FuturesTracker: mpt,
	}

	p.exchangeAssetPairPortfolioSettings = make(map[key.ExchangeAssetPair]*Settings)
	p.exchangeAssetPairPortfolioSettings[key.NewExchangeAssetPair(testExchange, a, pair)] = s
	ev.Close = leet
	err = s.ComplianceManager.AddSnapshot(&compliance.Snapshot{
		Timestamp: tt0,
		Orders: []compliance.SnapshotOrder{
			{
				Order: od,
			},
		},
	}, false)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	err = s.FuturesTracker.TrackNewOrder(od)
	assert.NoError(t, err)

	err = p.UpdatePNL(ev, decimal.NewFromInt(1))
	assert.NoError(t, err)

	pos := s.FuturesTracker.GetPositions()
	if len(pos) != 1 {
		t.Fatalf("expected one position, received '%v'", len(pos))
	}
	if len(pos[0].PNLHistory) == 0 {
		t.Fatal("expected a pnl entry 😎")
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
	assert.ErrorIs(t, err, common.ErrNilEvent)

	_, err = p.TrackFuturesOrder(&fill.Fill{}, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	fundPair := &funding.SpotPair{}
	_, err = p.TrackFuturesOrder(&fill.Fill{}, fundPair)
	assert.ErrorIs(t, err, gctorder.ErrSubmissionIsNil)

	od := &gctorder.Detail{}
	_, err = p.TrackFuturesOrder(&fill.Fill{
		Order: od,
	}, fundPair)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	od.AssetType = asset.Futures
	_, err = p.TrackFuturesOrder(&fill.Fill{
		Order: od,
	}, fundPair)
	assert.ErrorIs(t, err, funding.ErrNotCollateral)

	cp := currency.NewBTCUSD()
	od.Pair = cp
	od.Exchange = testExchange
	od.Side = gctorder.Short
	od.AssetType = asset.Futures
	od.Amount = 1
	od.Price = 0
	od.OrderID = od.Exchange
	od.Date = time.Now()
	contract, err := funding.CreateItem(od.Exchange, od.AssetType, od.Pair.Base, decimal.NewFromInt(9999), decimal.Zero)
	assert.NoError(t, err)

	collateral, err := funding.CreateItem(od.Exchange, od.AssetType, od.Pair.Quote, decimal.NewFromInt(9999), decimal.Zero)
	assert.NoError(t, err)

	err = collateral.IncreaseAvailable(decimal.NewFromInt(9999))
	assert.NoError(t, err)

	err = contract.IncreaseAvailable(decimal.NewFromInt(9999))
	assert.NoError(t, err)

	err = collateral.Reserve(leet)
	assert.NoError(t, err)

	err = contract.Reserve(leet)
	assert.NoError(t, err)

	collat, err := funding.CreateCollateral(contract, collateral)
	assert.NoError(t, err)

	_, err = p.TrackFuturesOrder(&fill.Fill{
		Order: od,
	}, collat)
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	ff := &binance.Exchange{}
	ff.Name = od.Exchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Futures, Pair: cp})
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)

	_, err = p.TrackFuturesOrder(&fill.Fill{
		Order: od,
		Base: &event.Base{
			Exchange:     od.Exchange,
			AssetType:    asset.Futures,
			CurrencyPair: cp,
		},
	}, collat)
	assert.ErrorIs(t, err, errNoPortfolioSettings)

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
	assert.ErrorIs(t, err, errNoPortfolioSettings)

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
	assert.ErrorIs(t, err, errNoPortfolioSettings)
}

func TestGetHoldingsForTime(t *testing.T) {
	t.Parallel()
	s := &Settings{
		HoldingsSnapshots: make(map[int64]*holdings.Holding),
	}
	_, err := s.GetHoldingsForTime(time.Now())
	assert.ErrorIs(t, err, errNoHoldings)

	tt := time.Now()
	s.HoldingsSnapshots[tt.UnixNano()] = &holdings.Holding{
		Timestamp: tt,
		Offset:    1337,
	}
	_, err = s.GetHoldingsForTime(time.Unix(1337, 0))
	assert.ErrorIs(t, err, errNoHoldings)

	h, err := s.GetHoldingsForTime(tt)
	assert.NoError(t, err)

	if h.Timestamp.IsZero() && h.Offset != 1337 {
		t.Error("expected set holdings")
	}
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	_, err := p.GetPositions(nil)
	assert.ErrorIs(t, err, common.ErrNilEvent)

	ev := &fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewBTCUSD(),
			AssetType:    asset.Futures,
		},
	}
	ff := &binance.Exchange{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: ev.AssetType, Pair: ev.Pair()})
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)

	_, err = p.GetPositions(ev)
	assert.ErrorIs(t, err, errNoPortfolioSettings)
}

func TestGetLatestPNLForEvent(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	_, err := p.GetLatestPNLForEvent(nil)
	assert.ErrorIs(t, err, common.ErrNilEvent)

	ev := &fill.Fill{
		Base: &event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewBTCUSD(),
			AssetType:    asset.Futures,
		},
	}
	ff := &binance.Exchange{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: ev.AssetType, Pair: ev.Pair()})
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)

	_, err = p.GetLatestPNLForEvent(ev)
	assert.ErrorIs(t, err, errNoPortfolioSettings)

	mpt, err := futures.SetupMultiPositionTracker(&futures.MultiPositionTrackerSetup{
		Exchange:           testExchange,
		Asset:              ev.AssetType,
		Pair:               ev.Pair(),
		Underlying:         currency.USDT,
		CollateralCurrency: currency.USDT,
		OfflineCalculation: true,
	})
	require.NoError(t, err, "SetupMultiPositionTracker must not error")
	s := &Settings{
		FuturesTracker: mpt,
	}

	p.exchangeAssetPairPortfolioSettings = make(map[key.ExchangeAssetPair]*Settings)
	p.exchangeAssetPairPortfolioSettings[key.NewExchangeAssetPair(testExchange, asset.Futures, ev.Pair())] = s
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
	require.NoError(t, err, "TrackNewOrder must not error")

	latest, err := p.GetLatestPNLForEvent(ev)
	require.NoError(t, err, "GetLatestPNLForEvent must not error")
	assert.NotNil(t, latest, "GetLatestPNLForEvent should return a non-nil result")
}

func TestGetFuturesSettingsFromEvent(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	_, err := p.getFuturesSettingsFromEvent(nil)
	require.ErrorIs(t, err, common.ErrNilEvent)

	b := &event.Base{}
	_, err = p.getFuturesSettingsFromEvent(&fill.Fill{
		Base: b,
	})
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	b.Exchange = testExchange
	b.CurrencyPair = currency.NewBTCUSDT()
	b.AssetType = asset.Futures
	ev := &fill.Fill{
		Base: b,
	}
	_, err = p.getFuturesSettingsFromEvent(ev)
	require.ErrorIs(t, err, errNoPortfolioSettings)

	ff := &binance.Exchange{}
	ff.Name = testExchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: ev.AssetType, Pair: ev.Pair()})
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)

	_, err = p.getFuturesSettingsFromEvent(ev)
	require.ErrorIs(t, err, errNoPortfolioSettings)

	_, err = p.getFuturesSettingsFromEvent(ev)
	require.ErrorIs(t, err, errNoPortfolioSettings)
}

func TestGetUnrealisedPNL(t *testing.T) {
	t.Parallel()
	p := PNLSummary{
		Exchange:           testExchange,
		Asset:              asset.Futures,
		Pair:               currency.NewBTCUSDT(),
		CollateralCurrency: currency.USDT,
		Offset:             1,
		Result: futures.PNLResult{
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
		Pair:               currency.NewBTCUSDT(),
		CollateralCurrency: currency.USDT,
		Offset:             1,
		Result: futures.PNLResult{
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
		Pair:               currency.NewBTCUSDT(),
		CollateralCurrency: currency.USDT,
		Offset:             1,
		Result: futures.PNLResult{
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
		Pair:               currency.NewBTCUSDT(),
		CollateralCurrency: currency.USDT,
		Offset:             1,
		Result: futures.PNLResult{
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
		Pair:               currency.NewBTCUSDT(),
		CollateralCurrency: currency.USDT,
		Offset:             1,
		Result: futures.PNLResult{
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
	_, err := cannotPurchase(nil, nil)
	assert.ErrorIs(t, err, common.ErrNilEvent)

	s := &signal.Signal{
		Base: &event.Base{},
	}
	_, err = cannotPurchase(s, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	o := &order.Order{
		Base: &event.Base{},
	}
	s.Direction = gctorder.Buy
	result, err := cannotPurchase(s, o)
	require.NoError(t, err, "cannotPurchase must not error")
	assert.Equal(t, gctorder.CouldNotBuy, result.Direction)

	s.Direction = gctorder.Sell
	result, err = cannotPurchase(s, o)
	require.NoError(t, err, "cannotPurchase must not error")
	assert.Equal(t, gctorder.CouldNotSell, result.Direction)

	s.Direction = gctorder.Short
	result, err = cannotPurchase(s, o)
	require.NoError(t, err, "cannotPurchase must not error")
	assert.Equal(t, gctorder.CouldNotShort, result.Direction)

	s.Direction = gctorder.Long
	result, err = cannotPurchase(s, o)
	require.NoError(t, err, "cannotPurchase must not error")
	assert.Equal(t, gctorder.CouldNotLong, result.Direction)

	s.Direction = gctorder.UnknownSide
	result, err = cannotPurchase(s, o)
	require.NoError(t, err, "cannotPurchase must not error")
	assert.Equal(t, gctorder.DoNothing, result.Direction)
}

func TestCreateLiquidationOrdersForExchange(t *testing.T) {
	t.Parallel()

	p := &Portfolio{}
	_, err := p.CreateLiquidationOrdersForExchange(nil, nil)
	require.ErrorIs(t, err, common.ErrNilEvent)

	b := &event.Base{}

	ev := &kline.Kline{
		Base: b,
	}
	_, err = p.CreateLiquidationOrdersForExchange(ev, nil)
	require.ErrorIs(t, err, gctcommon.ErrNilPointer)

	funds := &funding.FundManager{}
	_, err = p.CreateLiquidationOrdersForExchange(ev, funds)
	require.NoError(t, err)

	ff := &binance.Exchange{}
	ff.Name = testExchange
	cp := currency.NewBTCUSDT()
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Futures, Pair: cp})
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)

	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: ff, Asset: asset.Spot, Pair: cp})
	assert.NoError(t, err)

	ev.Exchange = ff.Name
	_, err = p.CreateLiquidationOrdersForExchange(ev, funds)
	require.NoError(t, err)

	_, err = p.getSettings(ff.Name, asset.Futures, cp)
	require.ErrorIs(t, err, errNoPortfolioSettings)

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

	mpt, err := futures.SetupMultiPositionTracker(&futures.MultiPositionTrackerSetup{
		Exchange:           testExchange,
		Asset:              od.AssetType,
		Pair:               cp,
		Underlying:         currency.USDT,
		CollateralCurrency: currency.USDT,
		OfflineCalculation: true,
	})
	assert.NoError(t, err)

	settings := &Settings{
		FuturesTracker: mpt,
	}

	err = settings.FuturesTracker.TrackNewOrder(od)
	assert.NoError(t, err)

	p.exchangeAssetPairPortfolioSettings = make(map[key.ExchangeAssetPair]*Settings)
	p.exchangeAssetPairPortfolioSettings[key.NewExchangeAssetPair(testExchange, asset.Spot, ev.Pair())] = settings

	ev.Exchange = ff.Name
	ev.AssetType = asset.Futures
	ev.CurrencyPair = cp
	_, err = p.CreateLiquidationOrdersForExchange(ev, funds)
	require.NoError(t, err)

	// spot order
	item, err := funding.CreateItem(ff.Name, asset.Spot, currency.BTC, decimal.Zero, decimal.Zero)
	require.NoError(t, err)

	err = funds.AddItem(item)
	require.NoError(t, err)

	err = item.IncreaseAvailable(leet)
	require.NoError(t, err)

	orders, err := p.CreateLiquidationOrdersForExchange(ev, funds)
	require.NoError(t, err)

	if len(orders) != 1 {
		t.Errorf("expected one order generated, received '%v'", len(orders))
	}
}

func TestGetPositionStatus(t *testing.T) {
	t.Parallel()
	p := PNLSummary{
		Result: futures.PNLResult{
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
	err := p.CheckLiquidationStatus(nil, nil, nil)
	assert.ErrorIs(t, err, common.ErrNilEvent)

	ev := &kline.Kline{
		Base: &event.Base{},
	}
	err = p.CheckLiquidationStatus(ev, nil, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	item := asset.Futures
	pair := currency.NewBTCUSDT()
	contract, err := funding.CreateItem(testExchange, item, pair.Base, decimal.NewFromInt(100), decimal.Zero)
	assert.NoError(t, err)

	collateral, err := funding.CreateItem(testExchange, item, pair.Quote, decimal.NewFromInt(100), decimal.Zero)
	assert.NoError(t, err)

	collat, err := funding.CreateCollateral(contract, collateral)
	assert.NoError(t, err)

	err = p.CheckLiquidationStatus(ev, collat, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	pnl := &PNLSummary{}
	err = p.CheckLiquidationStatus(ev, collat, pnl)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	pnl.Asset = asset.Futures
	ev.AssetType = asset.Futures
	ev.Exchange = testExchange
	ev.CurrencyPair = pair
	exch := &binance.Exchange{}
	exch.Name = ev.Exchange
	err = p.SetCurrencySettingsMap(&exchange.Settings{Exchange: exch, Asset: asset.Futures, Pair: pair})
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)

	_, err = p.getSettings(ev.Exchange, ev.AssetType, ev.Pair())
	assert.ErrorIs(t, err, errNoPortfolioSettings)

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
	mpt, err := futures.SetupMultiPositionTracker(&futures.MultiPositionTrackerSetup{
		Exchange:           testExchange,
		Asset:              ev.AssetType,
		Pair:               ev.Pair(),
		Underlying:         currency.USDT,
		CollateralCurrency: currency.USDT,
		OfflineCalculation: true,
	})
	assert.NoError(t, err)

	settings := &Settings{
		FuturesTracker: mpt,
	}

	err = settings.FuturesTracker.TrackNewOrder(od)
	assert.NoError(t, err)

	p.exchangeAssetPairPortfolioSettings = make(map[key.ExchangeAssetPair]*Settings)
	p.exchangeAssetPairPortfolioSettings[key.NewExchangeAssetPair(testExchange, asset.Futures, ev.Pair())] = settings
	err = p.CheckLiquidationStatus(ev, collat, pnl)
	assert.NoError(t, err)
}

func TestSetHoldingsForEvent(t *testing.T) {
	t.Parallel()
	p := &Portfolio{}
	err := p.SetHoldingsForEvent(nil, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	item, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.Zero, decimal.Zero)
	assert.NoError(t, err)

	cp, err := funding.CreatePair(item, item)
	assert.NoError(t, err)

	err = p.SetHoldingsForEvent(cp.FundReader(), nil)
	assert.ErrorIs(t, err, common.ErrNilEvent)

	err = p.SetHoldingsForEvent(cp.FundReader(), nil)
	assert.ErrorIs(t, err, common.ErrNilEvent)

	tt := time.Now()
	ev := &signal.Signal{
		Base: &event.Base{
			Exchange:     testExchange,
			CurrencyPair: currency.NewPair(currency.BTC, currency.BTC),
			AssetType:    asset.Spot,
			Time:         tt,
		},
	}
	f := &binance.Exchange{}
	f.SetDefaults()
	err = p.SetCurrencySettingsMap(&exchange.Settings{
		Exchange: f,
		Pair:     ev.Pair(),
		Asset:    ev.AssetType,
	})
	assert.NoError(t, err)

	err = p.SetHoldingsForEvent(cp.FundReader(), ev)
	assert.NoError(t, err)

	err = p.SetHoldingsForTimestamp(&holdings.Holding{
		Item:      currency.BTC,
		Pair:      ev.Pair(),
		Asset:     ev.AssetType,
		Exchange:  ev.Exchange,
		Timestamp: tt,
	})
	assert.NoError(t, err)

	err = p.SetHoldingsForEvent(cp.FundReader(), ev)
	assert.NoError(t, err)
}
