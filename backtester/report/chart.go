package report

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// createUSDTotalsChart used for creating a chart in the HTML report
// to show how much the overall assets are worth over time
func createUSDTotalsChart(items []statistics.ValueAtTime, stats []statistics.FundingItemStatistics) (*Chart, error) {
	if items == nil {
		return nil, fmt.Errorf("%w missing values at time", gctcommon.ErrNilPointer)
	}
	if stats == nil {
		return nil, fmt.Errorf("%w missing funding item statistics", gctcommon.ErrNilPointer)
	}
	response := &Chart{
		AxisType: "logarithmic",
	}
	usdTotalChartPlot := make([]LinePlot, len(items))
	for i := range items {
		usdTotalChartPlot[i] = LinePlot{
			Value:     items[i].Value.InexactFloat64(),
			UnixMilli: items[i].Time.UnixMilli(),
		}
	}
	response.Data = append(response.Data, ChartLine{
		Name:      "Total USD value",
		LinePlots: usdTotalChartPlot,
	})

	for i := range stats {
		var plots []LinePlot
		if stats[i].ReportItem.AppendedViaAPI {
			continue
		}
		for j := range stats[i].ReportItem.Snapshots {
			if stats[i].ReportItem.Snapshots[j].Available.IsZero() {
				response.ShowZeroDisclaimer = true
			}
			plots = append(plots, LinePlot{
				Value:     stats[i].ReportItem.Snapshots[j].USDValue.InexactFloat64(),
				UnixMilli: stats[i].ReportItem.Snapshots[j].Time.UnixMilli(),
			})
		}
		response.Data = append(response.Data, ChartLine{
			Name:      fmt.Sprintf("%v %v %v USD value", stats[i].ReportItem.Exchange, stats[i].ReportItem.Asset, stats[i].ReportItem.Currency),
			LinePlots: plots,
		})
	}

	return response, nil
}

// createHoldingsOverTimeChart used for creating a chart in the HTML report
// to show how many holdings of each type was held over the time of backtesting
func createHoldingsOverTimeChart(stats []statistics.FundingItemStatistics) (*Chart, error) {
	if stats == nil {
		return nil, fmt.Errorf("%w missing funding item statistics", gctcommon.ErrNilPointer)
	}
	response := &Chart{
		AxisType: "logarithmic",
	}
	for i := range stats {
		var plots []LinePlot
		if stats[i].ReportItem.AppendedViaAPI {
			continue
		}
		for j := range stats[i].ReportItem.Snapshots {
			if stats[i].ReportItem.Snapshots[j].Available.IsZero() {
				response.ShowZeroDisclaimer = true
			}
			plots = append(plots, LinePlot{
				UnixMilli: stats[i].ReportItem.Snapshots[j].Time.UnixMilli(),
				Value:     stats[i].ReportItem.Snapshots[j].Available.InexactFloat64(),
			})
		}
		response.Data = append(response.Data, ChartLine{
			Name:      fmt.Sprintf("%v %v %v holdings", stats[i].ReportItem.Exchange, stats[i].ReportItem.Asset, stats[i].ReportItem.Currency),
			LinePlots: plots,
		})
	}

	return response, nil
}

// createPNLCharts shows a running history of all realised and unrealised PNL values
// over time
func createPNLCharts(items map[key.ExchangeAssetPair]*statistics.CurrencyPairStatistic) (*Chart, error) {
	if items == nil {
		return nil, fmt.Errorf("%w missing currency pair statistics", gctcommon.ErrNilPointer)
	}
	response := &Chart{
		AxisType: "linear",
	}
	for mapKey, result := range items {
		id := fmt.Sprintf("%v %v %v%v",
			mapKey.Exchange,
			mapKey.Asset,
			mapKey.Base,
			mapKey.Quote)
		uPNLName := fmt.Sprintf("%v Unrealised PNL", id)
		rPNLName := fmt.Sprintf("%v Realised PNL", id)

		unrealisedPNL := ChartLine{Name: uPNLName}
		realisedPNL := ChartLine{Name: rPNLName}
		for i := range result.Events {
			if result.Events[i].PNL != nil {
				realisedPNL.LinePlots = append(realisedPNL.LinePlots, LinePlot{
					Value:     result.Events[i].PNL.GetRealisedPNL().PNL.InexactFloat64(),
					UnixMilli: result.Events[i].Time.UnixMilli(),
				})
				unrealisedPNL.LinePlots = append(unrealisedPNL.LinePlots, LinePlot{
					Value:     result.Events[i].PNL.GetUnrealisedPNL().PNL.InexactFloat64(),
					UnixMilli: result.Events[i].Time.UnixMilli(),
				})
			}
		}
		if len(unrealisedPNL.LinePlots) == 0 || len(realisedPNL.LinePlots) == 0 {
			continue
		}
		response.Data = append(response.Data, unrealisedPNL, realisedPNL)
	}
	return response, nil
}

// createFuturesSpotDiffChart highlights the difference in futures and spot prices
// over time
func createFuturesSpotDiffChart(items map[key.ExchangeAssetPair]*statistics.CurrencyPairStatistic) (*Chart, error) {
	if items == nil {
		return nil, fmt.Errorf("%w missing currency pair statistics", gctcommon.ErrNilPointer)
	}
	currs := make(map[currency.Pair]linkCurrencyDiff)
	response := &Chart{
		AxisType: "linear",
	}

	for mapKey, result := range items {
		cp := currency.NewPair(mapKey.Base.Currency(), mapKey.Quote.Currency())
		if mapKey.Asset.IsFutures() {
			p := result.UnderlyingPair.Format(currency.EMPTYFORMAT)
			diff, ok := currs[p]
			if !ok {
				diff = linkCurrencyDiff{}
			}
			diff.FuturesPair = cp
			diff.SpotPair = p
			diff.FuturesEvents = result.Events
			currs[p] = diff
		} else {
			p := cp.Format(currency.EMPTYFORMAT)
			diff, ok := currs[p]
			if !ok {
				diff = linkCurrencyDiff{}
			}
			diff.SpotEvents = result.Events
			currs[p] = diff
		}
	}

	for i := range currs {
		if currs[i].FuturesEvents == nil || currs[i].SpotEvents == nil {
			continue
		}
		if len(currs[i].SpotEvents) != len(currs[i].FuturesEvents) {
			continue
		}
		line := ChartLine{
			Name: fmt.Sprintf("%v %v diff %%", currs[i].FuturesPair, currs[i].SpotPair),
		}
		for j := range currs[i].SpotEvents {
			spotPrice := currs[i].SpotEvents[j].DataEvent.GetClosePrice()
			futuresPrice := currs[i].FuturesEvents[j].DataEvent.GetClosePrice()
			diff := futuresPrice.Sub(spotPrice).Div(spotPrice).Mul(decimal.NewFromInt(100))
			line.LinePlots = append(line.LinePlots, LinePlot{
				Value:     diff.InexactFloat64(),
				UnixMilli: currs[i].SpotEvents[j].Time.UnixMilli(),
			})
		}
		response.Data = append(response.Data, line)
	}
	return response, nil
}
