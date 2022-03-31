package report

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// createUSDTotalsChart used for creating a chart in the HTML report
// to show how much the overall assets are worth over time
func (d *Data) createUSDTotalsChart() *Chart {
	if d.Statistics.FundingStatistics == nil || d.Statistics.FundingStatistics.Report.DisableUSDTracking {
		return nil
	}
	response := &Chart{}
	var usdTotalChartPlot []LinePlot
	for i := range d.Statistics.FundingStatistics.TotalUSDStatistics.HoldingValues {
		usdTotalChartPlot = append(usdTotalChartPlot, LinePlot{
			Value:     d.Statistics.FundingStatistics.TotalUSDStatistics.HoldingValues[i].Value.InexactFloat64(),
			UnixMilli: d.Statistics.FundingStatistics.TotalUSDStatistics.HoldingValues[i].Time.UTC().UnixMilli(),
		})
	}
	response.Data = append(response.Data, ChartLine{
		Name:      "Total USD value",
		LinePlots: usdTotalChartPlot,
	})

	for i := range d.Statistics.FundingStatistics.Items {
		var plots []LinePlot
		for j := range d.Statistics.FundingStatistics.Items[i].ReportItem.Snapshots {
			plots = append(plots, LinePlot{
				Value:     d.Statistics.FundingStatistics.Items[i].ReportItem.Snapshots[j].USDValue.InexactFloat64(),
				UnixMilli: d.Statistics.FundingStatistics.Items[i].ReportItem.Snapshots[j].Time.UTC().UnixMilli(),
			})
		}
		response.Data = append(response.Data, ChartLine{
			Name:      fmt.Sprintf("%v %v %v USD value", d.Statistics.FundingStatistics.Items[i].ReportItem.Exchange, d.Statistics.FundingStatistics.Items[i].ReportItem.Asset, d.Statistics.FundingStatistics.Items[i].ReportItem.Currency),
			LinePlots: plots,
		})
	}

	return response
}

func (d *Data) createPNLCharts() *Chart {
	response := &Chart{}
	for exch, assetMap := range d.Statistics.ExchangeAssetPairStatistics {
		for item, pairMap := range assetMap {
			for pair, result := range pairMap {
				id := fmt.Sprintf("%v %v %v",
					exch,
					item,
					pair)
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
		}

	}
	return response
}

// createHoldingsOverTimeChart used for creating a chart in the HTML report
// to show how many holdings of each type was held over the time of backtesting
func (d *Data) createHoldingsOverTimeChart() *Chart {
	if d.Statistics.FundingStatistics == nil {
		return nil
	}
	response := &Chart{}
	response.AxisType = "logarithmic"
	for i := range d.Statistics.FundingStatistics.Items {
		var plots []LinePlot
		for j := range d.Statistics.FundingStatistics.Items[i].ReportItem.Snapshots {
			if d.Statistics.FundingStatistics.Items[i].ReportItem.Snapshots[j].Available.IsZero() {
				// highcharts can't render zeroes in logarithmic mode
				response.AxisType = "linear"
			}
			plots = append(plots, LinePlot{
				Value:     d.Statistics.FundingStatistics.Items[i].ReportItem.Snapshots[j].Available.InexactFloat64(),
				UnixMilli: d.Statistics.FundingStatistics.Items[i].ReportItem.Snapshots[j].Time.UTC().UnixMilli(),
			})
		}
		response.Data = append(response.Data, ChartLine{
			Name:      fmt.Sprintf("%v %v %v holdings", d.Statistics.FundingStatistics.Items[i].ReportItem.Exchange, d.Statistics.FundingStatistics.Items[i].ReportItem.Asset, d.Statistics.FundingStatistics.Items[i].ReportItem.Currency),
			LinePlots: plots,
		})
	}

	return response
}

type linkCurrencyDiff struct {
	FuturesPair   currency.Pair
	SpotPair      currency.Pair
	FuturesEvents []statistics.DataAtOffset
	SpotEvents    []statistics.DataAtOffset
	DiffPercent   []decimal.Decimal
}

func (d *Data) createFuturesSpotDiffChart() *Chart {
	var currs []linkCurrencyDiff
	response := &Chart{}
	for _, assetMap := range d.Statistics.ExchangeAssetPairStatistics {
		for item, pairMap := range assetMap {
			if !item.IsFutures() {
				continue
			}
			for pair, result := range pairMap {
				currs = append(currs, linkCurrencyDiff{
					FuturesPair:   pair,
					SpotPair:      result.UnderlyingPair,
					FuturesEvents: result.Events,
				})
			}
		}
	}
	for _, assetMap := range d.Statistics.ExchangeAssetPairStatistics {
		for item, pairMap := range assetMap {
			if item.IsFutures() {
				continue
			}
			for pair, result := range pairMap {
				for i := range currs {
					if pair.Equal(currs[i].SpotPair) {
						currs[i].SpotEvents = result.Events
					}
				}
			}
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
	return response
}
