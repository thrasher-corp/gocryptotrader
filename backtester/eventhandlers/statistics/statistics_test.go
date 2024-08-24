package statistics

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "binance"

var (
	eleeg  = decimal.NewFromInt(1336)
	eleet  = decimal.NewFromInt(1337)
	eleeet = decimal.NewFromInt(13337)
	eleeb  = decimal.NewFromInt(1338)
)

func TestReset(t *testing.T) {
	t.Parallel()
	s := &Statistic{
		TotalOrders: 1,
	}
	err := s.Reset()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if s.TotalOrders != 0 {
		t.Error("expected 0")
	}

	s = nil
	err = s.Reset()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}
}

func TestAddDataEventForTime(t *testing.T) {
	t.Parallel()
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}
	err := s.SetEventForOffset(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if s.ExchangeAssetPairStatistics == nil {
		t.Error("expected not nil")
	}
	if len(s.ExchangeAssetPairStatistics[key.ExchangePairAsset{
		Exchange: exch,
		Base:     p.Base.Item,
		Quote:    p.Quote.Item,
		Asset:    a,
	}].Events) != 1 {
		t.Error("expected 1 event")
	}
}

func TestAddSignalEventForTime(t *testing.T) {
	t.Parallel()
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}
	err := s.SetEventForOffset(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	err = s.SetEventForOffset(&signal.Signal{})
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	s.ExchangeAssetPairStatistics = make(map[key.ExchangePairAsset]*CurrencyPairStatistic)
	b := &event.Base{}
	err = s.SetEventForOffset(&signal.Signal{
		Base: b,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	b.Exchange = exch
	b.Time = tt
	b.Interval = gctkline.OneDay
	b.CurrencyPair = p
	b.AssetType = a
	err = s.SetEventForOffset(&kline.Kline{
		Base:   b,
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.SetEventForOffset(&signal.Signal{
		Base:       b,
		ClosePrice: eleet,
		Direction:  gctorder.Buy,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestAddExchangeEventForTime(t *testing.T) {
	t.Parallel()
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}
	err := s.SetEventForOffset(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	err = s.SetEventForOffset(&order.Order{})
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	s.ExchangeAssetPairStatistics = make(map[key.ExchangePairAsset]*CurrencyPairStatistic)
	b := &event.Base{}

	b.Exchange = exch
	b.Time = tt
	b.Interval = gctkline.OneDay
	b.CurrencyPair = p
	b.AssetType = a
	err = s.SetEventForOffset(&kline.Kline{
		Base:   b,
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.SetEventForOffset(&order.Order{
		Base:       b,
		ID:         "elite",
		Direction:  gctorder.Buy,
		Status:     gctorder.New,
		ClosePrice: eleet,
		Amount:     eleet,
		OrderType:  gctorder.Stop,
		Leverage:   eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestAddFillEventForTime(t *testing.T) {
	t.Parallel()
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}
	err := s.SetEventForOffset(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	err = s.SetEventForOffset(&fill.Fill{})
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	s.ExchangeAssetPairStatistics = make(map[key.ExchangePairAsset]*CurrencyPairStatistic)
	b := &event.Base{}
	err = s.SetEventForOffset(&fill.Fill{
		Base: b,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	b.Exchange = exch
	b.Time = tt
	b.Interval = gctkline.OneDay
	b.CurrencyPair = p
	b.AssetType = a

	err = s.SetEventForOffset(&kline.Kline{
		Base:   b,
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.SetEventForOffset(&fill.Fill{
		Base:                b,
		Direction:           gctorder.Buy,
		Amount:              eleet,
		ClosePrice:          eleet,
		VolumeAdjustedPrice: eleet,
		PurchasePrice:       eleet,
		ExchangeFee:         eleet,
		Slippage:            eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestAddHoldingsForTime(t *testing.T) {
	t.Parallel()
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}
	err := s.AddHoldingsForTime(&holdings.Holding{})
	if !errors.Is(err, errExchangeAssetPairStatsUnset) {
		t.Errorf("received: %v, expected: %v", err, errExchangeAssetPairStatsUnset)
	}
	s.ExchangeAssetPairStatistics = make(map[key.ExchangePairAsset]*CurrencyPairStatistic)
	err = s.AddHoldingsForTime(&holdings.Holding{})
	if !errors.Is(err, errCurrencyStatisticsUnset) {
		t.Errorf("received: %v, expected: %v", err, errCurrencyStatisticsUnset)
	}

	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.AddHoldingsForTime(&holdings.Holding{
		Pair:                         p,
		Asset:                        a,
		Exchange:                     exch,
		Timestamp:                    tt,
		QuoteInitialFunds:            eleet,
		BaseSize:                     eleet,
		BaseValue:                    eleet,
		SoldAmount:                   eleet,
		BoughtAmount:                 eleet,
		QuoteSize:                    eleet,
		TotalValueDifference:         eleet,
		ChangeInTotalValuePercent:    eleet,
		PositionsValueDifference:     eleet,
		TotalValue:                   eleet,
		TotalFees:                    eleet,
		TotalValueLostToVolumeSizing: eleet,
		TotalValueLostToSlippage:     eleet,
		TotalValueLost:               eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestAddComplianceSnapshotForTime(t *testing.T) {
	t.Parallel()
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}

	err := s.AddComplianceSnapshotForTime(nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	err = s.AddComplianceSnapshotForTime(nil, &fill.Fill{})
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}

	err = s.AddComplianceSnapshotForTime(&compliance.Snapshot{}, &fill.Fill{})
	if !errors.Is(err, errExchangeAssetPairStatsUnset) {
		t.Errorf("received: %v, expected: %v", err, errExchangeAssetPairStatsUnset)
	}
	s.ExchangeAssetPairStatistics = make(map[key.ExchangePairAsset]*CurrencyPairStatistic)
	b := &event.Base{}
	err = s.AddComplianceSnapshotForTime(&compliance.Snapshot{}, &fill.Fill{Base: b})
	if !errors.Is(err, errCurrencyStatisticsUnset) {
		t.Errorf("received: %v, expected: %v", err, errCurrencyStatisticsUnset)
	}
	b.Exchange = exch
	b.Time = tt
	b.Interval = gctkline.OneDay
	b.CurrencyPair = p
	b.AssetType = a
	err = s.SetEventForOffset(&kline.Kline{
		Base:   b,
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.AddComplianceSnapshotForTime(&compliance.Snapshot{
		Timestamp: tt,
	}, &fill.Fill{
		Base: b,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestSerialise(t *testing.T) {
	t.Parallel()
	s := Statistic{}
	if _, err := s.Serialise(); err != nil {
		t.Error(err)
	}
}

func TestSetStrategyName(t *testing.T) {
	t.Parallel()
	s := Statistic{}
	s.SetStrategyName("test")
	if s.StrategyName != "test" {
		t.Error("expected test")
	}
}

func TestPrintTotalResults(t *testing.T) {
	t.Parallel()
	s := Statistic{
		FundingStatistics: &FundingStatistics{},
	}
	s.BiggestDrawdown = s.GetTheBiggestDrawdownAcrossCurrencies([]FinalResultsHolder{
		{
			Exchange: "test",
			MaxDrawdown: Swing{
				DrawdownPercent: eleet,
			},
		},
	})
	s.BestStrategyResults = s.GetBestStrategyPerformer([]FinalResultsHolder{
		{
			Exchange:         "test",
			Asset:            asset.Spot,
			Pair:             currency.NewPair(currency.BTC, currency.DOGE),
			MaxDrawdown:      Swing{},
			MarketMovement:   eleet,
			StrategyMovement: eleet,
		},
	})
	s.BestMarketMovement = s.GetBestMarketPerformer([]FinalResultsHolder{
		{
			Exchange:       "test",
			MarketMovement: eleet,
		},
	})
	s.PrintTotalResults()
}

func TestGetBestStrategyPerformer(t *testing.T) {
	t.Parallel()
	s := Statistic{}
	resp := s.GetBestStrategyPerformer(nil)
	if resp.Exchange != "" {
		t.Error("expected unset details")
	}

	resp = s.GetBestStrategyPerformer([]FinalResultsHolder{
		{
			Exchange:         "test",
			Asset:            asset.Spot,
			Pair:             currency.NewPair(currency.BTC, currency.DOGE),
			MaxDrawdown:      Swing{},
			MarketMovement:   eleet,
			StrategyMovement: eleet,
		},
		{
			Exchange:         "test2",
			Asset:            asset.Spot,
			Pair:             currency.NewPair(currency.BTC, currency.DOGE),
			MaxDrawdown:      Swing{},
			MarketMovement:   eleeb,
			StrategyMovement: eleeb,
		},
	})

	if resp.Exchange != "test2" {
		t.Error("expected test2")
	}
}

func TestGetTheBiggestDrawdownAcrossCurrencies(t *testing.T) {
	t.Parallel()
	s := Statistic{}
	result := s.GetTheBiggestDrawdownAcrossCurrencies(nil)
	if result.Exchange != "" {
		t.Error("expected empty")
	}

	result = s.GetTheBiggestDrawdownAcrossCurrencies([]FinalResultsHolder{
		{
			Exchange: "test",
			MaxDrawdown: Swing{
				DrawdownPercent: eleet,
			},
		},
		{
			Exchange: "test2",
			MaxDrawdown: Swing{
				DrawdownPercent: eleeb,
			},
		},
	})
	if result.Exchange != "test2" {
		t.Error("expected test2")
	}
}

func TestGetBestMarketPerformer(t *testing.T) {
	t.Parallel()
	s := Statistic{}
	result := s.GetBestMarketPerformer(nil)
	if result.Exchange != "" {
		t.Error("expected empty")
	}

	result = s.GetBestMarketPerformer([]FinalResultsHolder{
		{
			Exchange:       "test",
			MarketMovement: eleet,
		},
		{
			Exchange:       "test2",
			MarketMovement: eleeg,
		},
	})
	if result.Exchange != "test" {
		t.Error("expected test")
	}
}

func TestPrintAllEventsChronologically(t *testing.T) {
	t.Parallel()
	s := Statistic{}
	s.PrintAllEventsChronologically()
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	err := s.SetEventForOffset(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = s.SetEventForOffset(&fill.Fill{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Direction:           gctorder.Buy,
		Amount:              eleet,
		ClosePrice:          eleet,
		VolumeAdjustedPrice: eleet,
		PurchasePrice:       eleet,
		ExchangeFee:         eleet,
		Slippage:            eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = s.SetEventForOffset(&signal.Signal{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		ClosePrice: eleet,
		Direction:  gctorder.Buy,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	s.PrintAllEventsChronologically()
}

func TestCalculateTheResults(t *testing.T) {
	t.Parallel()
	s := Statistic{}
	err := s.CalculateAllResults()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	tt := time.Now().Add(-gctkline.OneDay.Duration() * 7)
	tt2 := time.Now().Add(-gctkline.OneDay.Duration() * 6)
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	p2 := currency.NewPair(currency.XRP, currency.DOGE)
	err = s.SetEventForOffset(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}
	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
			Offset:       1,
		},
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.SetEventForOffset(&signal.Signal{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
			Offset:       1,
		},
		OpenPrice:  eleet,
		HighPrice:  eleet,
		LowPrice:   eleet,
		ClosePrice: eleet,
		Volume:     eleet,
		Direction:  gctorder.Buy,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p2,
			AssetType:    a,
			Offset:       2,
		},
		Open:   eleeb,
		Close:  eleeb,
		Low:    eleeb,
		High:   eleeb,
		Volume: eleeb,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = s.SetEventForOffset(&signal.Signal{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p2,
			AssetType:    a,
			Offset:       2,
		},
		OpenPrice:  eleet,
		HighPrice:  eleet,
		LowPrice:   eleet,
		ClosePrice: eleet,
		Volume:     eleet,
		Direction:  gctorder.Buy,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt2,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
			Offset:       3,
		},
		Open:   eleeb,
		Close:  eleeb,
		Low:    eleeb,
		High:   eleeb,
		Volume: eleeb,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = s.SetEventForOffset(&signal.Signal{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt2,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
			Offset:       3,
		},
		OpenPrice:  eleeb,
		HighPrice:  eleeb,
		LowPrice:   eleeb,
		ClosePrice: eleeb,
		Volume:     eleeb,
		Direction:  gctorder.Buy,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt2,
			Interval:     gctkline.OneDay,
			CurrencyPair: p2,
			AssetType:    a,
			Offset:       4,
		},
		Open:   eleeb,
		Close:  eleeb,
		Low:    eleeb,
		High:   eleeb,
		Volume: eleeb,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	signal4 := &signal.Signal{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt2,
			Interval:     gctkline.OneDay,
			CurrencyPair: p2,
			AssetType:    a,
			Offset:       4,
		},
		OpenPrice:  eleeb,
		HighPrice:  eleeb,
		LowPrice:   eleeb,
		ClosePrice: eleeb,
		Volume:     eleeb,
		Direction:  gctorder.Buy,
	}
	err = s.SetEventForOffset(signal4)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	mapKey1 := key.ExchangePairAsset{
		Exchange: exch,
		Base:     p.Base.Item,
		Quote:    p.Quote.Item,
		Asset:    a,
	}
	mapKey2 := key.ExchangePairAsset{
		Exchange: exch,
		Base:     p2.Base.Item,
		Quote:    p2.Quote.Item,
		Asset:    a,
	}
	s.ExchangeAssetPairStatistics[mapKey1].Events[1].Holdings.QuoteInitialFunds = eleet
	s.ExchangeAssetPairStatistics[mapKey1].Events[1].Holdings.TotalValue = eleeet
	s.ExchangeAssetPairStatistics[mapKey2].Events[1].Holdings.QuoteInitialFunds = eleet
	s.ExchangeAssetPairStatistics[mapKey2].Events[1].Holdings.TotalValue = eleeet

	funds, err := funding.SetupFundingManager(&engine.ExchangeManager{}, false, false, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	pBase, err := funding.CreateItem(exch, a, p.Base, eleeet, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	pQuote, err := funding.CreateItem(exch, a, p.Quote, eleeet, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	pair, err := funding.CreatePair(pBase, pQuote)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = funds.AddPair(pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	pBase2, err := funding.CreateItem(exch, a, p2.Base, eleeet, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	pQuote2, err := funding.CreateItem(exch, a, p2.Quote, eleeet, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	pair2, err := funding.CreatePair(pBase2, pQuote2)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = funds.AddPair(pair2)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	s.FundManager = funds
	err = s.CalculateAllResults()
	if !errors.Is(err, errMissingSnapshots) {
		t.Errorf("received '%v' expected '%v'", err, errMissingSnapshots)
	}
	err = s.CalculateAllResults()
	if !errors.Is(err, errMissingSnapshots) {
		t.Errorf("received '%v' expected '%v'", err, errMissingSnapshots)
	}

	funds, err = funding.SetupFundingManager(&engine.ExchangeManager{}, false, true, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = funds.AddPair(pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = funds.AddPair(pair2)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	s.FundManager = funds
	err = s.CalculateAllResults()
	if !errors.Is(err, errMissingSnapshots) {
		t.Errorf("received '%v' expected '%v'", err, errMissingSnapshots)
	}

	err = s.AddComplianceSnapshotForTime(&compliance.Snapshot{Timestamp: tt2}, signal4)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

func TestCalculateBiggestEventDrawdown(t *testing.T) {
	tt1 := time.Now().Add(-gctkline.OneDay.Duration() * 7).Round(gctkline.OneDay.Duration())
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	var events []data.Event
	for i := range int64(100) {
		tt1 = tt1.Add(gctkline.OneDay.Duration())
		even := &event.Base{
			Exchange:     exch,
			Time:         tt1,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		}
		if i == 50 {
			// throw in a wrench, a spike in price
			events = append(events, &kline.Kline{
				Base:  even,
				Close: decimal.NewFromInt(1336),
				High:  decimal.NewFromInt(1336),
				Low:   decimal.NewFromInt(1336),
			})
		} else {
			events = append(events, &kline.Kline{
				Base:  even,
				Close: decimal.NewFromInt(1337).Sub(decimal.NewFromInt(i)),
				High:  decimal.NewFromInt(1337).Sub(decimal.NewFromInt(i)),
				Low:   decimal.NewFromInt(1337).Sub(decimal.NewFromInt(i)),
			})
		}
	}

	tt1 = tt1.Add(gctkline.OneDay.Duration())
	even := &event.Base{
		Exchange:     exch,
		Time:         tt1,
		Interval:     gctkline.OneDay,
		CurrencyPair: p,
		AssetType:    a,
	}
	events = append(events, &kline.Kline{
		Base:  even,
		Close: decimal.NewFromInt(1338),
		High:  decimal.NewFromInt(1338),
		Low:   decimal.NewFromInt(1338),
	})

	tt1 = tt1.Add(gctkline.OneDay.Duration())
	even = &event.Base{
		Exchange:     exch,
		Time:         tt1,
		Interval:     gctkline.OneDay,
		CurrencyPair: p,
		AssetType:    a,
	}
	events = append(events, &kline.Kline{
		Base:  even,
		Close: decimal.NewFromInt(1337),
		High:  decimal.NewFromInt(1337),
		Low:   decimal.NewFromInt(1337),
	})

	tt1 = tt1.Add(gctkline.OneDay.Duration())
	even = &event.Base{
		Exchange:     exch,
		Time:         tt1,
		Interval:     gctkline.OneDay,
		CurrencyPair: p,
		AssetType:    a,
	}
	events = append(events, &kline.Kline{
		Base:  even,
		Close: decimal.NewFromInt(1339),
		High:  decimal.NewFromInt(1339),
		Low:   decimal.NewFromInt(1339),
	})

	_, err := CalculateBiggestEventDrawdown(nil)
	if !errors.Is(err, errReceivedNoData) {
		t.Errorf("received %v expected %v", err, errReceivedNoData)
	}

	resp, err := CalculateBiggestEventDrawdown(events)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if resp.Highest.Value != decimal.NewFromInt(1337) && !resp.Lowest.Value.Equal(decimal.NewFromInt(1238)) {
		t.Error("unexpected max drawdown")
	}

	// bogus scenario
	bogusEvent := []data.Event{
		&kline.Kline{
			Base: &event.Base{
				Exchange:     exch,
				CurrencyPair: p,
				AssetType:    a,
			},
			Close: decimal.NewFromInt(1339),
			High:  decimal.NewFromInt(1339),
			Low:   decimal.NewFromInt(1339),
		},
	}
	_, err = CalculateBiggestEventDrawdown(bogusEvent)
	if !errors.Is(err, gctcommon.ErrDateUnset) {
		t.Errorf("received %v expected %v", err, gctcommon.ErrDateUnset)
	}
}

func TestCalculateBiggestValueAtTimeDrawdown(t *testing.T) {
	var interval gctkline.Interval
	_, err := CalculateBiggestValueAtTimeDrawdown(nil, interval)
	if !errors.Is(err, errReceivedNoData) {
		t.Errorf("received %v expected %v", err, errReceivedNoData)
	}

	_, err = CalculateBiggestValueAtTimeDrawdown(nil, interval)
	if !errors.Is(err, errReceivedNoData) {
		t.Errorf("received %v expected %v", err, errReceivedNoData)
	}
}

func TestAddPNLForTime(t *testing.T) {
	t.Parallel()
	s := &Statistic{}
	err := s.AddPNLForTime(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received %v expected %v", err, gctcommon.ErrNilPointer)
	}

	sum := &portfolio.PNLSummary{}
	err = s.AddPNLForTime(sum)
	if !errors.Is(err, errExchangeAssetPairStatsUnset) {
		t.Errorf("received %v expected %v", err, errExchangeAssetPairStatsUnset)
	}

	tt := time.Now().Add(-gctkline.OneDay.Duration() * 7)
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	err = s.SetEventForOffset(&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
			Offset:       1,
		},
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = s.AddPNLForTime(sum)
	if !errors.Is(err, errCurrencyStatisticsUnset) {
		t.Errorf("received %v expected %v", err, errCurrencyStatisticsUnset)
	}

	sum.Exchange = exch
	sum.Asset = a
	sum.Pair = p
	err = s.AddPNLForTime(sum)
	if !errors.Is(err, errNoDataAtOffset) {
		t.Errorf("received %v expected %v", err, errNoDataAtOffset)
	}

	sum.Offset = 1
	err = s.AddPNLForTime(sum)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
}
