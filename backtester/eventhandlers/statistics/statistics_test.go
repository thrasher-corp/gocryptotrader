package statistics

import (
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "binance"

func TestReset(t *testing.T) {
	s := Statistic{
		TotalOrders: 1,
	}
	s.Reset()
	if s.TotalOrders != 0 {
		t.Error("expected 0")
	}
}

func TestAddDataEventForTime(t *testing.T) {
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}
	err := s.SetupEventForTime(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, received %v", common.ErrNilEvent, err)
	}
	err = s.SetupEventForTime(&kline.Kline{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   1337,
		Close:  1337,
		Low:    1337,
		High:   1337,
		Volume: 1337,
	})
	if err != nil {
		t.Error(err)
	}
	if s.ExchangeAssetPairStatistics == nil {
		t.Error("expected not nil")
	}
	if len(s.ExchangeAssetPairStatistics[exch][a][p].Events) != 1 {
		t.Error("expected 1 event")
	}
}

func TestAddSignalEventForTime(t *testing.T) {
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}
	err := s.SetEventForOffset(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, received %v", common.ErrNilEvent, err)
	}
	err = s.SetEventForOffset(&signal.Signal{})
	if !errors.Is(err, errExchangeAssetPairStatsUnset) {
		t.Errorf("expected: %v, received %v", errExchangeAssetPairStatsUnset, err)
	}
	s.setupMap(exch, a)
	s.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyStatistic)
	err = s.SetEventForOffset(&signal.Signal{})
	if !errors.Is(err, errCurrencyStatisticsUnset) {
		t.Errorf("expected: %v, received %v", errCurrencyStatisticsUnset, err)
	}

	err = s.SetupEventForTime(&kline.Kline{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   1337,
		Close:  1337,
		Low:    1337,
		High:   1337,
		Volume: 1337,
	})
	if err != nil {
		t.Error(err)
	}
	err = s.SetEventForOffset(&signal.Signal{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		ClosePrice: 1337,
		Direction:  gctorder.Buy,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestAddExchangeEventForTime(t *testing.T) {
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}
	err := s.SetEventForOffset(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, received %v", common.ErrNilEvent, err)
	}
	err = s.SetEventForOffset(&order.Order{})
	if !errors.Is(err, errExchangeAssetPairStatsUnset) {
		t.Errorf("expected: %v, received %v", errExchangeAssetPairStatsUnset, err)
	}
	s.setupMap(exch, a)
	s.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyStatistic)
	err = s.SetEventForOffset(&order.Order{})
	if !errors.Is(err, errCurrencyStatisticsUnset) {
		t.Errorf("expected: %v, received %v", errCurrencyStatisticsUnset, err)
	}

	err = s.SetupEventForTime(&kline.Kline{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   1337,
		Close:  1337,
		Low:    1337,
		High:   1337,
		Volume: 1337,
	})
	if err != nil {
		t.Error(err)
	}
	err = s.SetEventForOffset(&order.Order{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		ID:        "1337",
		Direction: gctorder.Buy,
		Status:    gctorder.New,
		Price:     1337,
		Amount:    1337,
		OrderType: gctorder.Stop,
		Leverage:  1337,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestAddFillEventForTime(t *testing.T) {
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}
	err := s.SetEventForOffset(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, received %v", common.ErrNilEvent, err)
	}
	err = s.SetEventForOffset(&fill.Fill{})
	if err != nil && err.Error() != "exchangeAssetPairStatistics not setup" {
		t.Error(err)
	}
	s.setupMap(exch, a)
	s.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyStatistic)
	err = s.SetEventForOffset(&fill.Fill{})
	if !errors.Is(err, errCurrencyStatisticsUnset) {
		t.Errorf("expected: %v, received %v", errCurrencyStatisticsUnset, err)
	}

	err = s.SetupEventForTime(&kline.Kline{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   1337,
		Close:  1337,
		Low:    1337,
		High:   1337,
		Volume: 1337,
	})
	if err != nil {
		t.Error(err)
	}
	err = s.SetEventForOffset(&fill.Fill{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Direction:           gctorder.Buy,
		Amount:              1337,
		ClosePrice:          1337,
		VolumeAdjustedPrice: 1337,
		PurchasePrice:       1337,
		ExchangeFee:         1337,
		Slippage:            1337,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestAddHoldingsForTime(t *testing.T) {
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}
	err := s.AddHoldingsForTime(&holdings.Holding{})
	if !errors.Is(err, errExchangeAssetPairStatsUnset) {
		t.Errorf("expected: %v, received %v", errExchangeAssetPairStatsUnset, err)
	}
	s.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyStatistic)
	err = s.AddHoldingsForTime(&holdings.Holding{})
	if !errors.Is(err, errCurrencyStatisticsUnset) {
		t.Errorf("expected: %v, received %v", errCurrencyStatisticsUnset, err)
	}

	err = s.SetupEventForTime(&kline.Kline{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   1337,
		Close:  1337,
		Low:    1337,
		High:   1337,
		Volume: 1337,
	})
	if err != nil {
		t.Error(err)
	}
	err = s.AddHoldingsForTime(&holdings.Holding{
		Pair:                         p,
		Asset:                        a,
		Exchange:                     exch,
		Timestamp:                    tt,
		InitialFunds:                 1337,
		PositionsSize:                1337,
		PositionsValue:               1337,
		SoldAmount:                   1337,
		SoldValue:                    1337,
		BoughtAmount:                 1337,
		BoughtValue:                  1337,
		RemainingFunds:               1337,
		TotalValueDifference:         1337,
		ChangeInTotalValuePercent:    1337,
		BoughtValueDifference:        1337,
		SoldValueDifference:          1337,
		PositionsValueDifference:     1337,
		TotalValue:                   1337,
		TotalFees:                    1337,
		TotalValueLostToVolumeSizing: 1337,
		TotalValueLostToSlippage:     1337,
		TotalValueLost:               1337,
		RiskFreeRate:                 1337,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestAddComplianceSnapshotForTime(t *testing.T) {
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	s := Statistic{}

	err := s.AddComplianceSnapshotForTime(compliance.Snapshot{}, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, received %v", common.ErrNilEvent, err)
	}
	err = s.AddComplianceSnapshotForTime(compliance.Snapshot{}, &fill.Fill{})
	if !errors.Is(err, errExchangeAssetPairStatsUnset) {
		t.Errorf("expected: %v, received %v", errExchangeAssetPairStatsUnset, err)
	}
	s.setupMap(exch, a)
	s.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyStatistic)
	err = s.AddComplianceSnapshotForTime(compliance.Snapshot{}, &fill.Fill{})
	if !errors.Is(err, errCurrencyStatisticsUnset) {
		t.Errorf("expected: %v, received %v", errCurrencyStatisticsUnset, err)
	}

	err = s.SetupEventForTime(&kline.Kline{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   1337,
		Close:  1337,
		Low:    1337,
		High:   1337,
		Volume: 1337,
	})
	if err != nil {
		t.Error(err)
	}
	err = s.AddComplianceSnapshotForTime(compliance.Snapshot{
		Timestamp: tt,
	}, &fill.Fill{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestSerialise(t *testing.T) {
	s := Statistic{}
	_, err := s.Serialise()
	if err != nil {
		t.Error(err)
	}
}

func TestSetStrategyName(t *testing.T) {
	s := Statistic{}
	s.SetStrategyName("test")
	if s.StrategyName != "test" {
		t.Error("expected test")
	}
}

func TestPrintTotalResults(t *testing.T) {
	s := Statistic{}
	s.BiggestDrawdown = s.GetTheBiggestDrawdownAcrossCurrencies([]FinalResultsHolder{
		{
			Exchange: "test",
			MaxDrawdown: currencystatistics.Swing{
				DrawdownPercent: 1337,
			},
		},
	})
	s.BestStrategyResults = s.GetBestStrategyPerformer([]FinalResultsHolder{
		{
			Exchange:         "test",
			Asset:            asset.Spot,
			Pair:             currency.NewPair(currency.BTC, currency.DOGE),
			MaxDrawdown:      currencystatistics.Swing{},
			MarketMovement:   1337,
			StrategyMovement: 1337,
		},
	})
	s.BestMarketMovement = s.GetBestMarketPerformer([]FinalResultsHolder{
		{
			Exchange:       "test",
			MarketMovement: 1337,
		},
	})
	s.PrintTotalResults()
}

func TestGetBestStrategyPerformer(t *testing.T) {
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
			MaxDrawdown:      currencystatistics.Swing{},
			MarketMovement:   1337,
			StrategyMovement: 1337,
		},
		{
			Exchange:         "test2",
			Asset:            asset.Spot,
			Pair:             currency.NewPair(currency.BTC, currency.DOGE),
			MaxDrawdown:      currencystatistics.Swing{},
			MarketMovement:   1338,
			StrategyMovement: 1338,
		},
	})

	if resp.Exchange != "test2" {
		t.Error("expected test2")
	}
}

func TestGetTheBiggestDrawdownAcrossCurrencies(t *testing.T) {
	s := Statistic{}
	result := s.GetTheBiggestDrawdownAcrossCurrencies(nil)
	if result.Exchange != "" {
		t.Error("expected empty")
	}

	result = s.GetTheBiggestDrawdownAcrossCurrencies([]FinalResultsHolder{
		{
			Exchange: "test",
			MaxDrawdown: currencystatistics.Swing{
				DrawdownPercent: 1337,
			},
		},
		{
			Exchange: "test2",
			MaxDrawdown: currencystatistics.Swing{
				DrawdownPercent: 1338,
			},
		},
	})
	if result.Exchange != "test2" {
		t.Error("expected test2")
	}
}

func TestGetBestMarketPerformer(t *testing.T) {
	s := Statistic{}
	result := s.GetBestMarketPerformer(nil)
	if result.Exchange != "" {
		t.Error("expected empty")
	}

	result = s.GetBestMarketPerformer([]FinalResultsHolder{
		{
			Exchange:       "test",
			MarketMovement: 1337,
		},
		{
			Exchange:       "test2",
			MarketMovement: 1336,
		},
	})
	if result.Exchange != "test" {
		t.Error("expected test")
	}
}

func TestPrintAllEvents(t *testing.T) {
	s := Statistic{}
	s.PrintAllEvents()
	tt := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	err := s.SetupEventForTime(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, received %v", common.ErrNilEvent, err)
	}
	err = s.SetupEventForTime(&kline.Kline{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   1337,
		Close:  1337,
		Low:    1337,
		High:   1337,
		Volume: 1337,
	})
	if err != nil {
		t.Error(err)
	}

	err = s.SetEventForOffset(&fill.Fill{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Direction:           gctorder.Buy,
		Amount:              1337,
		ClosePrice:          1337,
		VolumeAdjustedPrice: 1337,
		PurchasePrice:       1337,
		ExchangeFee:         1337,
		Slippage:            1337,
	})
	if err != nil {
		t.Error(err)
	}

	err = s.SetEventForOffset(&signal.Signal{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		ClosePrice: 1337,
		Direction:  gctorder.Buy,
	})
	if err != nil {
		t.Error(err)
	}

	s.PrintAllEvents()
}

func TestCalculateTheResults(t *testing.T) {
	s := Statistic{}
	err := s.CalculateAllResults()
	if err != nil {
		t.Error(err)
	}

	tt := time.Now().Add(-gctkline.OneDay.Duration() * 7)
	tt2 := time.Now().Add(-gctkline.OneDay.Duration() * 6)
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	p2 := currency.NewPair(currency.DOGE, currency.DOGE)
	err = s.SetupEventForTime(nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("expected: %v, received %v", common.ErrNilEvent, err)
	}
	err = s.SetupEventForTime(&kline.Kline{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   1337,
		Close:  1337,
		Low:    1337,
		High:   1337,
		Volume: 1337,
	})
	if err != nil {
		t.Error(err)
	}
	err = s.SetEventForOffset(&signal.Signal{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		OpenPrice:  1337,
		HighPrice:  1337,
		LowPrice:   1337,
		ClosePrice: 1337,
		Volume:     1337,
		Direction:  gctorder.Buy,
	})
	if err != nil {
		t.Error(err)
	}
	err = s.SetupEventForTime(&kline.Kline{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p2,
			AssetType:    a,
		},
		Open:   1338,
		Close:  1338,
		Low:    1338,
		High:   1338,
		Volume: 1338,
	})
	if err != nil {
		t.Error(err)
	}

	err = s.SetEventForOffset(&signal.Signal{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p2,
			AssetType:    a,
		},
		OpenPrice:  1337,
		HighPrice:  1337,
		LowPrice:   1337,
		ClosePrice: 1337,
		Volume:     1337,
		Direction:  gctorder.Buy,
	})
	if err != nil {
		t.Error(err)
	}

	err = s.SetupEventForTime(&kline.Kline{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt2,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   1338,
		Close:  1338,
		Low:    1338,
		High:   1338,
		Volume: 1338,
	})
	if err != nil {
		t.Error(err)
	}
	err = s.SetEventForOffset(&signal.Signal{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt2,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		OpenPrice:  1338,
		HighPrice:  1338,
		LowPrice:   1338,
		ClosePrice: 1338,
		Volume:     1338,
		Direction:  gctorder.Buy,
	})
	if err != nil {
		t.Error(err)
	}

	err = s.SetupEventForTime(&kline.Kline{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt2,
			Interval:     gctkline.OneDay,
			CurrencyPair: p2,
			AssetType:    a,
		},
		Open:   1338,
		Close:  1338,
		Low:    1338,
		High:   1338,
		Volume: 1338,
	})
	if err != nil {
		t.Error(err)
	}
	err = s.SetEventForOffset(&signal.Signal{
		Base: event.Base{
			Exchange:     exch,
			Time:         tt2,
			Interval:     gctkline.OneDay,
			CurrencyPair: p2,
			AssetType:    a,
		},
		OpenPrice:  1338,
		HighPrice:  1338,
		LowPrice:   1338,
		ClosePrice: 1338,
		Volume:     1338,
		Direction:  gctorder.Buy,
	})
	if err != nil {
		t.Error(err)
	}

	s.ExchangeAssetPairStatistics[exch][a][p].Events[1].Holdings.InitialFunds = 1337
	s.ExchangeAssetPairStatistics[exch][a][p].Events[1].Holdings.TotalValue = 13337
	s.ExchangeAssetPairStatistics[exch][a][p2].Events[1].Holdings.InitialFunds = 1337
	s.ExchangeAssetPairStatistics[exch][a][p2].Events[1].Holdings.TotalValue = 13337

	err = s.CalculateAllResults()
	if err != nil {
		t.Error(err)
	}
}
