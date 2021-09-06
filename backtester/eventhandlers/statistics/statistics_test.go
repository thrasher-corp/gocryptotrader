package statistics

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatistics"
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

var (
	eleeg  = decimal.NewFromFloat(1336)
	eleet  = decimal.NewFromInt(1337)
	eleeet = decimal.NewFromFloat(13337)
	eleeb  = decimal.NewFromFloat(1338)
)

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
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
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
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
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
		ClosePrice: eleet,
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
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
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
		ID:        "elite",
		Direction: gctorder.Buy,
		Status:    gctorder.New,
		Price:     eleet,
		Amount:    eleet,
		OrderType: gctorder.Stop,
		Leverage:  eleet,
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
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
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
		Amount:              eleet,
		ClosePrice:          eleet,
		VolumeAdjustedPrice: eleet,
		PurchasePrice:       eleet,
		ExchangeFee:         eleet,
		Slippage:            eleet,
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
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
	})
	if err != nil {
		t.Error(err)
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
		SoldValue:                    eleet,
		BoughtAmount:                 eleet,
		BoughtValue:                  eleet,
		RemainingFunds:               eleet,
		TotalValueDifference:         eleet,
		ChangeInTotalValuePercent:    eleet,
		BoughtValueDifference:        eleet,
		SoldValueDifference:          eleet,
		PositionsValueDifference:     eleet,
		TotalValue:                   eleet,
		TotalFees:                    eleet,
		TotalValueLostToVolumeSizing: eleet,
		TotalValueLostToSlippage:     eleet,
		TotalValueLost:               eleet,
		RiskFreeRate:                 eleet,
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
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
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
				DrawdownPercent: eleet,
			},
		},
	})
	s.BestStrategyResults = s.GetBestStrategyPerformer([]FinalResultsHolder{
		{
			Exchange:         "test",
			Asset:            asset.Spot,
			Pair:             currency.NewPair(currency.BTC, currency.DOGE),
			MaxDrawdown:      currencystatistics.Swing{},
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
			MarketMovement:   eleet,
			StrategyMovement: eleet,
		},
		{
			Exchange:         "test2",
			Asset:            asset.Spot,
			Pair:             currency.NewPair(currency.BTC, currency.DOGE),
			MaxDrawdown:      currencystatistics.Swing{},
			MarketMovement:   eleeb,
			StrategyMovement: eleeb,
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
				DrawdownPercent: eleet,
			},
		},
		{
			Exchange: "test2",
			MaxDrawdown: currencystatistics.Swing{
				DrawdownPercent: eleeb,
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
	s := Statistic{}
	s.PrintAllEventsChronologically()
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
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
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
		Amount:              eleet,
		ClosePrice:          eleet,
		VolumeAdjustedPrice: eleet,
		PurchasePrice:       eleet,
		ExchangeFee:         eleet,
		Slippage:            eleet,
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
		ClosePrice: eleet,
		Direction:  gctorder.Buy,
	})
	if err != nil {
		t.Error(err)
	}

	s.PrintAllEventsChronologically()
}

func TestCalculateTheResults(t *testing.T) {
	s := Statistic{}
	err := s.CalculateAllResults(&funding.FundManager{})
	if err != nil {
		t.Error(err)
	}

	tt := time.Now().Add(-gctkline.OneDay.Duration() * 7)
	tt2 := time.Now().Add(-gctkline.OneDay.Duration() * 6)
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	p2 := currency.NewPair(currency.XRP, currency.DOGE)
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
		Open:   eleet,
		Close:  eleet,
		Low:    eleet,
		High:   eleet,
		Volume: eleet,
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
		OpenPrice:  eleet,
		HighPrice:  eleet,
		LowPrice:   eleet,
		ClosePrice: eleet,
		Volume:     eleet,
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
		Open:   eleeb,
		Close:  eleeb,
		Low:    eleeb,
		High:   eleeb,
		Volume: eleeb,
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
		OpenPrice:  eleet,
		HighPrice:  eleet,
		LowPrice:   eleet,
		ClosePrice: eleet,
		Volume:     eleet,
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
		Open:   eleeb,
		Close:  eleeb,
		Low:    eleeb,
		High:   eleeb,
		Volume: eleeb,
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
		OpenPrice:  eleeb,
		HighPrice:  eleeb,
		LowPrice:   eleeb,
		ClosePrice: eleeb,
		Volume:     eleeb,
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
		Open:   eleeb,
		Close:  eleeb,
		Low:    eleeb,
		High:   eleeb,
		Volume: eleeb,
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
		OpenPrice:  eleeb,
		HighPrice:  eleeb,
		LowPrice:   eleeb,
		ClosePrice: eleeb,
		Volume:     eleeb,
		Direction:  gctorder.Buy,
	})
	if err != nil {
		t.Error(err)
	}

	s.ExchangeAssetPairStatistics[exch][a][p].Events[1].Holdings.QuoteInitialFunds = eleet
	s.ExchangeAssetPairStatistics[exch][a][p].Events[1].Holdings.TotalValue = eleeet
	s.ExchangeAssetPairStatistics[exch][a][p2].Events[1].Holdings.QuoteInitialFunds = eleet
	s.ExchangeAssetPairStatistics[exch][a][p2].Events[1].Holdings.TotalValue = eleeet

	funds := &funding.FundManager{}
	pBase, err := funds.SetupItem(exch, a, p.Base, eleeet, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	pQuote, err := funds.SetupItem(exch, a, p.Quote, eleeet, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = funds.CreatePair(pBase, pQuote)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	pBase2, err := funds.SetupItem(exch, a, p2.Base, eleeet, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	pQuote2, err := funds.SetupItem(exch, a, p2.Quote, eleeet, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = funds.CreatePair(pBase2, pQuote2)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = s.CalculateAllResults(funds)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}
