package report

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatstics"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestGenerateReport(t *testing.T) {
	e := "Binance"
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := Data{
		OriginalCandles: nil,
		Candles: []DetailedKline{
			{
				Exchange:  e,
				Asset:     a,
				Pair:      p,
				Interval:  kline.OneHour,
				Watermark: "Binance - SPOT - BTC-USDT - 1h",
				Candles: []DetailedCandle{
					{
						Time:           time.Now().Add(-time.Hour * 5).Unix(),
						Open:           1337,
						High:           1339,
						Low:            1336,
						Close:          1338,
						Volume:         3,
						MadeOrder:      true,
						OrderDirection: order.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(47, 194, 27, 0.8)",
					},
					{
						Time:           time.Now().Add(-time.Hour * 4).Unix(),
						Open:           1332,
						High:           1332,
						Low:            1330,
						Close:          1331,
						Volume:         2,
						MadeOrder:      true,
						OrderDirection: order.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(252, 3, 3, 0.8)",
					},
					{
						Time:           time.Now().Add(-time.Hour * 3).Unix(),
						Open:           1337,
						High:           1339,
						Low:            1336,
						Close:          1338,
						Volume:         3,
						MadeOrder:      true,
						OrderDirection: order.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(47, 194, 27, 0.8)",
					},
					{
						Time:           time.Now().Add(-time.Hour * 2).Unix(),
						Open:           1337,
						High:           1339,
						Low:            1336,
						Close:          1338,
						Volume:         3,
						MadeOrder:      true,
						OrderDirection: order.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(252, 3, 3, 0.8)",
					},
					{
						Time:         time.Now().Unix(),
						Open:         1337,
						High:         1339,
						Low:          1336,
						Close:        1338,
						Volume:       3,
						VolumeColour: "rgba(47, 194, 27, 0.8)",
					},
				},
			},
			{
				Exchange:  "Bittrex",
				Asset:     a,
				Pair:      currency.NewPair(currency.BTC, currency.USD),
				Interval:  kline.OneDay,
				Watermark: "BITTREX - SPOT - BTC-USD - 1d",
				Candles: []DetailedCandle{
					{
						Time:           time.Now().Add(-time.Hour * 5).Unix(),
						Open:           1337,
						High:           1339,
						Low:            1336,
						Close:          1338,
						Volume:         3,
						MadeOrder:      true,
						OrderDirection: order.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(47, 194, 27, 0.8)",
					},
					{
						Time:           time.Now().Add(-time.Hour * 4).Unix(),
						Open:           1332,
						High:           1332,
						Low:            1330,
						Close:          1331,
						Volume:         2,
						MadeOrder:      true,
						OrderDirection: order.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(252, 3, 3, 0.8)",
					},
					{
						Time:           time.Now().Add(-time.Hour * 3).Unix(),
						Open:           1337,
						High:           1339,
						Low:            1336,
						Close:          1338,
						Volume:         3,
						MadeOrder:      true,
						OrderDirection: order.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(47, 194, 27, 0.8)",
					},
					{
						Time:           time.Now().Add(-time.Hour * 2).Unix(),
						Open:           1337,
						High:           1339,
						Low:            1336,
						Close:          1338,
						Volume:         3,
						MadeOrder:      true,
						OrderDirection: order.Buy,
						OrderAmount:    1337,
						Shape:          "arrowUp",
						Text:           "hi",
						Position:       "aboveBar",
						Colour:         "green",
						PurchasePrice:  50,
						VolumeColour:   "rgba(252, 3, 3, 0.8)",
					},
					{
						Time:         time.Now().Unix(),
						Open:         1337,
						High:         1339,
						Low:          1336,
						Close:        1338,
						Volume:       3,
						VolumeColour: "rgba(47, 194, 27, 0.8)",
					},
				},
			},
		},
		Statistics: &statistics.Statistic{
			StrategyName: "testStrat",
			StartDate:    time.Now().Add(-time.Hour * 24 * 36),
			IntervalSize: kline.OneDay.Duration(),
			EndDate:      time.Now(),
			ExchangeAssetPairStatistics: map[string]map[asset.Item]map[currency.Pair]*currencystatstics.CurrencyStatistic{
				e: {
					a: {
						p: &currencystatstics.CurrencyStatistic{
							Pair:                     p,
							Asset:                    a,
							Exchange:                 e,
							Events:                   nil,
							DrawDowns:                currencystatstics.SwingHolder{},
							Upswings:                 currencystatstics.SwingHolder{},
							LowestClosePrice:         0,
							HighestClosePrice:        0,
							MarketMovement:           0,
							StrategyMovement:         0,
							SharpeRatio:              0,
							SortinoRatio:             0,
							InformationRatio:         0,
							RiskFreeRate:             0,
							CalamariRatio:            0,
							CompoundAnnualGrowthRate: 0,
							BuyOrders:                0,
							SellOrders:               0,
							FinalHoldings:            holdings.Holding{},
							Orders:                   compliance.Snapshot{},
						},
					},
				},
			},
			RiskFreeRate:    0.03,
			TotalBuyOrders:  1337,
			TotalSellOrders: 1330,
			TotalOrders:     200,
			BiggestDrawdown: &statistics.FinalResultsHolder{
				Exchange: e,
				Asset:    a,
				Pair:     p,
				MaxDrawdown: currencystatstics.Swing{
					Highest: currencystatstics.Iteration{
						Time:  time.Now(),
						Price: 1337,
					},
					Lowest: currencystatstics.Iteration{
						Time:  time.Now(),
						Price: 137,
					},
					CalculatedDrawDown: 100,
				},
				MarketMovement:   1377,
				StrategyMovement: 1377,
			},
			BestStrategyResults: &statistics.FinalResultsHolder{
				Exchange: e,
				Asset:    a,
				Pair:     p,
				MaxDrawdown: currencystatstics.Swing{
					Highest: currencystatstics.Iteration{
						Time:  time.Now(),
						Price: 1337,
					},
					Lowest: currencystatstics.Iteration{
						Time:  time.Now(),
						Price: 137,
					},
					CalculatedDrawDown: 100,
				},
				MarketMovement:   1337,
				StrategyMovement: 1337,
			},
			BestMarketMovement: &statistics.FinalResultsHolder{
				Exchange: e,
				Asset:    a,
				Pair:     p,
				MaxDrawdown: currencystatstics.Swing{
					Highest: currencystatstics.Iteration{
						Time:  time.Now(),
						Price: 1337,
					},
					Lowest: currencystatstics.Iteration{
						Time:  time.Now(),
						Price: 137,
					},
					CalculatedDrawDown: 100,
				},
				MarketMovement:   1337,
				StrategyMovement: 1337,
			},
		},
	}
	err := d.GenerateReport()
	if err != nil {
		t.Error(err)
	}
}
