package currencystatstics

import (
	"math"
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

func TestSortinoRatio(t *testing.T) {
	rfr := 0.07
	figures := []float64{0.10, 0.04, 0.15, -0.05, 0.20, -0.02, 0.08, -0.06, 0.13, 0.23}
	negativeOnlyFigures := []float64{-0.05, -0.02, -0.06}
	r := calculateSortinoRatio(figures, negativeOnlyFigures, rfr)
	if r != 0.3922322702763678 {
		t.Errorf("received %v instead", r)
	}
}

func TestInformationRatio(t *testing.T) {
	figures := []float64{0.0665, 0.0283, 0.0911, 0.0008, -0.0203, -0.0978, 0.0164, -0.0537, 0.078, 0.0032, 0.0249, 0}
	comparisonFigures := []float64{0.0216, 0.0048, 0.036, 0.0303, 0.0043, -0.0694, 0.0179, -0.0918, 0.0787, 0.0297, 0.003, 0}
	avg := calculateTheAverage(figures)
	if avg != 0.01145 {
		t.Error(avg)
	}
	avgComparison := calculateTheAverage(comparisonFigures)
	if avgComparison != 0.005425 {
		t.Error(avgComparison)
	}

	var eachDiff []float64
	for i := range figures {
		eachDiff = append(eachDiff, figures[i]-comparisonFigures[i])
	}
	stdDev := calculateStandardDeviation(eachDiff)
	if stdDev != 0.028992588851865803 {
		t.Error(stdDev)
	}
	informationRatio := (avg - avgComparison) / stdDev
	if informationRatio != 0.20781172839666107 {
		t.Error(informationRatio)
	}

	information2 := calculateInformationRatio(figures, comparisonFigures)
	if informationRatio != information2 {
		t.Error(information2)
	}
}

func TestCalmarRatio(t *testing.T) {
	drawDown := Swing{Highest: Iteration{Price: 50000}, Lowest: Iteration{Price: 15000}}
	avg := []float64{0.2}
	ratio := calculateCalmarRatio(avg, &drawDown)
	if ratio != 0.28571428571428575 {
		t.Error(ratio)
	}
}

func TestCAGR(t *testing.T) {
	cagr := calculateCompoundAnnualGrowthRate(100, 147, time.Date(2015, 1, 1, 0, 0, 0, 0, time.Local), time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local), gctkline.OneYear)
	if cagr != 0.08009875865888949 {
		t.Error(cagr)
	}
}

func TestCalculateSharpeRatio(t *testing.T) {
	result := calculateSharpeRatio(nil, 0)
	if result != 0 {
		t.Error("expected 0")
	}

	result = calculateSharpeRatio([]float64{0.026}, 0.017)
	if result != 0 {
		t.Error("expected 0")
	}

	returns := []float64{
		0.08,
		0.1,
		0.09,
		0.06,
		0.07,
		0.11,
		0.08,
		0.1,
		0.02,
		0.09,
	}
	result = calculateSharpeRatio(returns, 0.04)
	if result != 1.5491933384829664 {
		t.Error("expected 1.55~")
	}
}

func TestStandardDeviation2(t *testing.T) {
	r := []float64{9, 2, 5, 4, 12, 7}
	mean := calculateTheAverage(r)
	superMean := []float64{}
	for i := range r {
		result := math.Pow(r[i]-mean, 2)
		superMean = append(superMean, result)
	}
	superMeany := (superMean[0] + superMean[1] + superMean[2] + superMean[3] + superMean[4] + superMean[5]) / 5
	manualCalculation := math.Sqrt(superMeany)
	codeCalcu := calculateSampleStandardDeviation(r)
	if manualCalculation != codeCalcu && codeCalcu != 3.619 {
		t.Error("expected 3.619")
	}
}

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
			})
		} else {
			events = append(events, &kline.Kline{
				Event: even,
				Close: 1337 - float64(i),
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
	})

	resp := calculateMaxDrawdown(events)
	if resp.Highest.Price != 1337 && resp.Lowest.Price != 1238 {
		t.Error("unexpected max drawdown")
	}
}
