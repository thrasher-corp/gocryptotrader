package currencystatstics

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
	tt2 := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	even := event.Event{
		Exchange:     exch,
		Time:         tt1,
		Interval:     gctkline.OneDay,
		CurrencyPair: p,
		AssetType:    a,
	}
	ev := EventStore{
		Holdings: holdings.Holding{},
		Transactions: compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          1337,
					VolumeAdjustedPrice: 1337,
					SlippageRate:        1337,
					CostBasis:           1337,
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
			Event: even,
			Close: 1338,
		},
		SignalEvent: &signal.Signal{
			Event: even,
			Price: 1337,
		},
	}
	even.Time = tt2
	ev2 := EventStore{
		Holdings: holdings.Holding{},
		Transactions: compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          1337,
					VolumeAdjustedPrice: 1337,
					SlippageRate:        1337,
					CostBasis:           1337,
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
			Event: even,
			Close: 1338,
		},
		SignalEvent: &signal.Signal{
			Event: even,
			Price: 1338,
		},
	}

	cs.Events = append(cs.Events, ev, ev2)
	cs.CalculateResults()
	if cs.MarketMovement != 0 {
		t.Error("expected 0")
	}
}

func TestPrintResults(t *testing.T) {
	cs := CurrencyStatistic{}
	tt1 := time.Now()
	tt2 := time.Now()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	even := event.Event{
		Exchange:     exch,
		Time:         tt1,
		Interval:     gctkline.OneDay,
		CurrencyPair: p,
		AssetType:    a,
	}
	ev := EventStore{
		Holdings: holdings.Holding{},
		Transactions: compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          1337,
					VolumeAdjustedPrice: 1337,
					SlippageRate:        1337,
					CostBasis:           1337,
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
			Event: even,
			Close: 1338,
		},
		SignalEvent: &signal.Signal{
			Event: even,
			Price: 1337,
		},
	}
	even.Time = tt2
	ev2 := EventStore{
		Holdings: holdings.Holding{},
		Transactions: compliance.Snapshot{
			Orders: []compliance.SnapshotOrder{
				{
					ClosePrice:          1337,
					VolumeAdjustedPrice: 1337,
					SlippageRate:        1337,
					CostBasis:           1337,
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
			Event: even,
			Close: 1338,
		},
		SignalEvent: &signal.Signal{
			Event: even,
			Price: 1338,
		},
	}

	cs.Events = append(cs.Events, ev, ev2)
	cs.CalculateResults()
	cs.PrintResults(exch, a, p)
}

func TestCalculateMaxDrawdown(t *testing.T) {
	tt1 := time.Now().Round(gctkline.OneDay.Duration())
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	var events []common.DataEventHandler
	for i := 0; i < 100; i++ {
		tt1 = tt1.Add(gctkline.OneDay.Duration())
		even := event.Event{
			Exchange:     exch,
			Time:         tt1,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		}
		if i == 50 {
			// throw in a wrench, a spike in price
			events = append(events, &kline.Kline{
				Event: even,
				Close: 1336,
				High:  1336,
				Low:   1336,
			})
		} else {
			events = append(events, &kline.Kline{
				Event: even,
				Close: 1337 - float64(i),
				High:  1337 - float64(i),
				Low:   1337 - float64(i),
			})
		}
	}

	tt1 = tt1.Add(gctkline.OneDay.Duration())
	even := event.Event{
		Exchange:     exch,
		Time:         tt1,
		Interval:     gctkline.OneDay,
		CurrencyPair: p,
		AssetType:    a,
	}
	events = append(events, &kline.Kline{
		Event: even,
		Close: 1338,
		High:  1338,
		Low:   1338,
	})

	tt1 = tt1.Add(gctkline.OneDay.Duration())
	even = event.Event{
		Exchange:     exch,
		Time:         tt1,
		Interval:     gctkline.OneDay,
		CurrencyPair: p,
		AssetType:    a,
	}
	events = append(events, &kline.Kline{
		Event: even,
		Close: 1337,
		High:  1337,
		Low:   1337,
	})

	tt1 = tt1.Add(gctkline.OneDay.Duration())
	even = event.Event{
		Exchange:     exch,
		Time:         tt1,
		Interval:     gctkline.OneDay,
		CurrencyPair: p,
		AssetType:    a,
	}
	events = append(events, &kline.Kline{
		Event: even,
		Close: 1339,
		High:  1339,
		Low:   1339,
	})

	resp := calculateMaxDrawdown(events)
	if resp.Highest.Price != 1337 && resp.Lowest.Price != 1238 {
		t.Error("unexpected max drawdown")
	}
}
