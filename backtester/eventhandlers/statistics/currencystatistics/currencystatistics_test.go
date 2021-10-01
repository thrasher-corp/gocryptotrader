package currencystatistics

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "binance"

func TestCalculateResults(t *testing.T) {
	t.Parallel()
	cs := CurrencyPairStatistic{}
	tt1 := time.Now()
	tt2 := time.Now().Add(gctkline.OneDay.Duration())
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	even := event.Base{
		Exchange:     exch,
		Time:         tt1,
		Interval:     gctkline.OneDay,
		CurrencyPair: p,
		AssetType:    a,
	}
	ev := EventStore{
		Holdings: holdings.Holding{
			ChangeInTotalValuePercent: decimal.NewFromFloat(0.1333),
			Timestamp:                 tt1,
			QuoteInitialFunds:         decimal.NewFromInt(1337),
			RiskFreeRate:              decimal.NewFromInt(1),
		},
		Transactions: compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          decimal.NewFromInt(1338),
					VolumeAdjustedPrice: decimal.NewFromInt(1338),
					SlippageRate:        decimal.NewFromInt(1338),
					CostBasis:           decimal.NewFromInt(1338),
					Detail:              &order.Detail{Side: order.Buy},
				},
				{
					ClosePrice:          decimal.NewFromInt(1337),
					VolumeAdjustedPrice: decimal.NewFromInt(1337),
					SlippageRate:        decimal.NewFromInt(1337),
					CostBasis:           decimal.NewFromInt(1337),
					Detail:              &order.Detail{Side: order.Sell},
				},
			},
		},
		DataEvent: &kline.Kline{
			Base:   even,
			Open:   decimal.NewFromInt(2000),
			Close:  decimal.NewFromInt(2000),
			Low:    decimal.NewFromInt(2000),
			High:   decimal.NewFromInt(2000),
			Volume: decimal.NewFromInt(2000),
		},
		SignalEvent: &signal.Signal{
			Base:       even,
			ClosePrice: decimal.NewFromInt(2000),
		},
	}
	even2 := even
	even2.Time = tt2
	ev2 := EventStore{
		Holdings: holdings.Holding{
			ChangeInTotalValuePercent: decimal.NewFromFloat(0.1337),
			Timestamp:                 tt2,
			QuoteInitialFunds:         decimal.NewFromInt(1337),
			RiskFreeRate:              decimal.NewFromInt(1),
		},
		Transactions: compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          decimal.NewFromInt(1338),
					VolumeAdjustedPrice: decimal.NewFromInt(1338),
					SlippageRate:        decimal.NewFromInt(1338),
					CostBasis:           decimal.NewFromInt(1338),
					Detail:              &order.Detail{Side: order.Buy},
				},
				{
					ClosePrice:          decimal.NewFromInt(1337),
					VolumeAdjustedPrice: decimal.NewFromInt(1337),
					SlippageRate:        decimal.NewFromInt(1337),
					CostBasis:           decimal.NewFromInt(1337),
					Detail:              &order.Detail{Side: order.Sell},
				},
			},
		},
		DataEvent: &kline.Kline{
			Base:   even2,
			Open:   decimal.NewFromInt(1337),
			Close:  decimal.NewFromInt(1337),
			Low:    decimal.NewFromInt(1337),
			High:   decimal.NewFromInt(1337),
			Volume: decimal.NewFromInt(1337),
		},
		SignalEvent: &signal.Signal{
			Base:       even2,
			ClosePrice: decimal.NewFromInt(1337),
		},
	}

	cs.Events = append(cs.Events, ev, ev2)
	b, err := funding.CreateItem(testExchange, asset.Spot, currency.BTC, decimal.NewFromInt(13337), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	q, err := funding.CreateItem(testExchange, asset.Spot, currency.USDT, decimal.NewFromInt(13337), decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := funding.CreatePair(b, q)
	if err != nil {
		t.Fatal(err)
	}
	err = cs.CalculateResults(pair)
	if err != nil {
		t.Error(err)
	}
	if !cs.MarketMovement.Equal(decimal.NewFromFloat(-33.15)) {
		t.Error("expected -33.15")
	}
}

func TestPrintResults(t *testing.T) {
	cs := CurrencyPairStatistic{}
	tt1 := time.Now()
	tt2 := time.Now().Add(gctkline.OneDay.Duration())
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	even := event.Base{
		Exchange:     exch,
		Time:         tt1,
		Interval:     gctkline.OneDay,
		CurrencyPair: p,
		AssetType:    a,
	}
	ev := EventStore{
		Holdings: holdings.Holding{
			ChangeInTotalValuePercent: decimal.NewFromFloat(0.1333),
			Timestamp:                 tt1,
			QuoteInitialFunds:         decimal.NewFromInt(1337),
		},
		Transactions: compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          decimal.NewFromInt(1338),
					VolumeAdjustedPrice: decimal.NewFromInt(1338),
					SlippageRate:        decimal.NewFromInt(1338),
					CostBasis:           decimal.NewFromInt(1338),
					Detail:              &order.Detail{Side: order.Buy},
				},
				{
					ClosePrice:          decimal.NewFromInt(1337),
					VolumeAdjustedPrice: decimal.NewFromInt(1337),
					SlippageRate:        decimal.NewFromInt(1337),
					CostBasis:           decimal.NewFromInt(1337),
					Detail:              &order.Detail{Side: order.Sell},
				},
			},
		},
		DataEvent: &kline.Kline{
			Base:   even,
			Open:   decimal.NewFromInt(2000),
			Close:  decimal.NewFromInt(2000),
			Low:    decimal.NewFromInt(2000),
			High:   decimal.NewFromInt(2000),
			Volume: decimal.NewFromInt(2000),
		},
		SignalEvent: &signal.Signal{
			Base:       even,
			ClosePrice: decimal.NewFromInt(2000),
		},
	}
	even2 := even
	even2.Time = tt2
	ev2 := EventStore{
		Holdings: holdings.Holding{
			ChangeInTotalValuePercent: decimal.NewFromFloat(0.1337),
			Timestamp:                 tt2,
			QuoteInitialFunds:         decimal.NewFromInt(1337),
		},
		Transactions: compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          decimal.NewFromInt(1338),
					VolumeAdjustedPrice: decimal.NewFromInt(1338),
					SlippageRate:        decimal.NewFromInt(1338),
					CostBasis:           decimal.NewFromInt(1338),
					Detail:              &order.Detail{Side: order.Buy},
				},
				{
					ClosePrice:          decimal.NewFromInt(1337),
					VolumeAdjustedPrice: decimal.NewFromInt(1337),
					SlippageRate:        decimal.NewFromInt(1337),
					CostBasis:           decimal.NewFromInt(1337),
					Detail:              &order.Detail{Side: order.Sell},
				},
			},
		},
		DataEvent: &kline.Kline{
			Base:   even2,
			Open:   decimal.NewFromInt(1337),
			Close:  decimal.NewFromInt(1337),
			Low:    decimal.NewFromInt(1337),
			High:   decimal.NewFromInt(1337),
			Volume: decimal.NewFromInt(1337),
		},
		SignalEvent: &signal.Signal{
			Base:       even2,
			ClosePrice: decimal.NewFromInt(1337),
		},
	}

	cs.Events = append(cs.Events, ev, ev2)
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
	err = cs.CalculateResults(pair)
	if err != nil {
		t.Error(err)
	}
	cs.PrintResults(exch, a, p, pair, true)
}

func TestCalculateMaxDrawdown(t *testing.T) {
	tt1 := time.Now().Add(-gctkline.OneDay.Duration() * 7).Round(gctkline.OneDay.Duration())
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	var events []common.DataEventHandler
	for i := int64(0); i < 100; i++ {
		tt1 = tt1.Add(gctkline.OneDay.Duration())
		even := event.Base{
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
	even := event.Base{
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
	even = event.Base{
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
	even = event.Base{
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

	resp := calculateMaxDrawdown(events)
	if resp.Highest.Price != decimal.NewFromInt(1337) && !resp.Lowest.Price.Equal(decimal.NewFromInt(1238)) {
		t.Error("unexpected max drawdown")
	}
}

func TestCalculateHighestCommittedFunds(t *testing.T) {
	t.Parallel()
	c := CurrencyPairStatistic{}
	c.calculateHighestCommittedFunds()
	if !c.HighestCommittedFunds.Time.IsZero() {
		t.Error("expected no time with not committed funds")
	}
	tt1 := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	tt2 := time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC)
	tt3 := time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)
	c.Events = append(c.Events,
		EventStore{DataEvent: &kline.Kline{Close: decimal.NewFromInt(1337)}, Holdings: holdings.Holding{Timestamp: tt1, BaseSize: decimal.NewFromInt(10)}},
		EventStore{DataEvent: &kline.Kline{Close: decimal.NewFromInt(1338)}, Holdings: holdings.Holding{Timestamp: tt2, BaseSize: decimal.NewFromInt(1337)}},
		EventStore{DataEvent: &kline.Kline{Close: decimal.NewFromInt(1339)}, Holdings: holdings.Holding{Timestamp: tt3, BaseSize: decimal.NewFromInt(11)}},
	)
	c.calculateHighestCommittedFunds()
	if c.HighestCommittedFunds.Time != tt2 {
		t.Errorf("expected %v, received %v", tt2, c.HighestCommittedFunds.Time)
	}
}
