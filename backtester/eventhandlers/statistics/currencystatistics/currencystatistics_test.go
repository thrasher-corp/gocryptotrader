package currencystatistics

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "binance"

func TestCalculateResults(t *testing.T) {
	cs := CurrencyStatistic{}
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
			ChangeInTotalValuePercent: 0.1333,
			Timestamp:                 tt1,
			InitialFunds:              1337,
		},
		Transactions: compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          1338,
					VolumeAdjustedPrice: 1338,
					SlippageRate:        1338,
					CostBasis:           1338,
					Detail:              &order.Detail{Side: order.Buy},
				},
				{
					ClosePrice:          1337,
					VolumeAdjustedPrice: 1337,
					SlippageRate:        1337,
					CostBasis:           1337,
					Detail:              &order.Detail{Side: order.Sell},
				},
			},
		},
		DataEvent: &kline.Kline{
			Base:   even,
			Open:   2000,
			Close:  2000,
			Low:    2000,
			High:   2000,
			Volume: 2000,
		},
		SignalEvent: &signal.Signal{
			Base:       even,
			ClosePrice: 2000,
		},
	}
	even2 := even
	even2.Time = tt2
	ev2 := EventStore{
		Holdings: holdings.Holding{
			ChangeInTotalValuePercent: 0.1337,
			Timestamp:                 tt2,
			InitialFunds:              1337,
		},
		Transactions: compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          1338,
					VolumeAdjustedPrice: 1338,
					SlippageRate:        1338,
					CostBasis:           1338,
					Detail:              &order.Detail{Side: order.Buy},
				},
				{
					ClosePrice:          1337,
					VolumeAdjustedPrice: 1337,
					SlippageRate:        1337,
					CostBasis:           1337,
					Detail:              &order.Detail{Side: order.Sell},
				},
			},
		},
		DataEvent: &kline.Kline{
			Base:   even2,
			Open:   1337,
			Close:  1337,
			Low:    1337,
			High:   1337,
			Volume: 1337,
		},
		SignalEvent: &signal.Signal{
			Base:       even2,
			ClosePrice: 1337,
		},
	}

	cs.Events = append(cs.Events, ev, ev2)
	err := cs.CalculateResults()
	if err != nil {
		t.Error(err)
	}
	if cs.MarketMovement != -33.15 {
		t.Error("expected -33.15")
	}
}

func TestPrintResults(t *testing.T) {
	cs := CurrencyStatistic{}
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
			ChangeInTotalValuePercent: 0.1333,
			Timestamp:                 tt1,
			InitialFunds:              1337,
		},
		Transactions: compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          1338,
					VolumeAdjustedPrice: 1338,
					SlippageRate:        1338,
					CostBasis:           1338,
					Detail:              &order.Detail{Side: order.Buy},
				},
				{
					ClosePrice:          1337,
					VolumeAdjustedPrice: 1337,
					SlippageRate:        1337,
					CostBasis:           1337,
					Detail:              &order.Detail{Side: order.Sell},
				},
			},
		},
		DataEvent: &kline.Kline{
			Base:   even,
			Open:   2000,
			Close:  2000,
			Low:    2000,
			High:   2000,
			Volume: 2000,
		},
		SignalEvent: &signal.Signal{
			Base:       even,
			ClosePrice: 2000,
		},
	}
	even2 := even
	even2.Time = tt2
	ev2 := EventStore{
		Holdings: holdings.Holding{
			ChangeInTotalValuePercent: 0.1337,
			Timestamp:                 tt2,
			InitialFunds:              1337,
		},
		Transactions: compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          1338,
					VolumeAdjustedPrice: 1338,
					SlippageRate:        1338,
					CostBasis:           1338,
					Detail:              &order.Detail{Side: order.Buy},
				},
				{
					ClosePrice:          1337,
					VolumeAdjustedPrice: 1337,
					SlippageRate:        1337,
					CostBasis:           1337,
					Detail:              &order.Detail{Side: order.Sell},
				},
			},
		},
		DataEvent: &kline.Kline{
			Base:   even2,
			Open:   1337,
			Close:  1337,
			Low:    1337,
			High:   1337,
			Volume: 1337,
		},
		SignalEvent: &signal.Signal{
			Base:       even2,
			ClosePrice: 1337,
		},
	}

	cs.Events = append(cs.Events, ev, ev2)
	err := cs.CalculateResults()
	if err != nil {
		t.Error(err)
	}
	cs.PrintResults(exch, a, p)
}

func TestCalculateMaxDrawdown(t *testing.T) {
	tt1 := time.Now().Add(-gctkline.OneDay.Duration() * 7).Round(gctkline.OneDay.Duration())
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	var events []common.DataEventHandler
	for i := 0; i < 100; i++ {
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
				Close: 1336,
				High:  1336,
				Low:   1336,
			})
		} else {
			events = append(events, &kline.Kline{
				Base:  even,
				Close: 1337 - float64(i),
				High:  1337 - float64(i),
				Low:   1337 - float64(i),
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
		Close: 1338,
		High:  1338,
		Low:   1338,
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
		Close: 1337,
		High:  1337,
		Low:   1337,
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
		Close: 1339,
		High:  1339,
		Low:   1339,
	})

	resp := calculateMaxDrawdown(events)
	if resp.Highest.Price != 1337 && resp.Lowest.Price != 1238 {
		t.Error("unexpected max drawdown")
	}
}

func TestCalculateHighestCommittedFunds(t *testing.T) {
	c := CurrencyStatistic{}
	c.calculateHighestCommittedFunds()
	if !c.HighestCommittedFunds.Time.IsZero() {
		t.Error("expected no time with not committed funds")
	}
	tt1 := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	tt2 := time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC)
	tt3 := time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)
	c.Events = append(c.Events,
		EventStore{Holdings: holdings.Holding{Timestamp: tt1, CommittedFunds: 10}},
		EventStore{Holdings: holdings.Holding{Timestamp: tt2, CommittedFunds: 1337}},
		EventStore{Holdings: holdings.Holding{Timestamp: tt3, CommittedFunds: 11}},
	)
	c.calculateHighestCommittedFunds()
	if c.HighestCommittedFunds.Time != tt2 {
		t.Errorf("expected %v, received %v", tt2, c.HighestCommittedFunds.Time)
	}
}
