package report

import "fmt"

// createUSDTotalsChart used for creating a chart in the HTML report
// to show how much the overall assets are worth over time
func (d *Data) createUSDTotalsChart() *Chart {
	if d.Statistics.FundingStatistics == nil || d.Statistics.FundingStatistics.Report.DisableUSDTracking {
		return nil
	}
	response := &Chart{}
	var usdTotalChartPlot []ChartPlot
	for i := range d.Statistics.FundingStatistics.TotalUSDStatistics.HoldingValues {
		usdTotalChartPlot = append(usdTotalChartPlot, ChartPlot{
			Value:     d.Statistics.FundingStatistics.TotalUSDStatistics.HoldingValues[i].Value.InexactFloat64(),
			UnixMilli: d.Statistics.FundingStatistics.TotalUSDStatistics.HoldingValues[i].Time.UTC().UnixMilli(),
		})
	}
	response.Data = append(response.Data, ChartLine{
		Name:       "Total USD value",
		DataPoints: usdTotalChartPlot,
	})

	for i := range d.Statistics.FundingStatistics.Items {
		var plots []ChartPlot
		for j := range d.Statistics.FundingStatistics.Items[i].ReportItem.Snapshots {
			plots = append(plots, ChartPlot{
				Value:     d.Statistics.FundingStatistics.Items[i].ReportItem.Snapshots[j].USDValue.InexactFloat64(),
				UnixMilli: d.Statistics.FundingStatistics.Items[i].ReportItem.Snapshots[j].Time.UTC().UnixMilli(),
			})
		}
		response.Data = append(response.Data, ChartLine{
			Name:       fmt.Sprintf("%v %v %v USD value", d.Statistics.FundingStatistics.Items[i].ReportItem.Exchange, d.Statistics.FundingStatistics.Items[i].ReportItem.Asset, d.Statistics.FundingStatistics.Items[i].ReportItem.Currency),
			DataPoints: plots,
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
						realisedPNL.DataPoints = append(realisedPNL.DataPoints, ChartPlot{
							Value:     result.Events[i].PNL.GetRealisedPNL().PNL.InexactFloat64(),
							UnixMilli: result.Events[i].Time.UnixMilli(),
						})
						unrealisedPNL.DataPoints = append(unrealisedPNL.DataPoints, ChartPlot{
							Value:     result.Events[i].PNL.GetUnrealisedPNL().PNL.InexactFloat64(),
							UnixMilli: result.Events[i].Time.UnixMilli(),
						})
					}
				}
				if len(unrealisedPNL.DataPoints) == 0 || len(realisedPNL.DataPoints) == 0 {
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
		var plots []ChartPlot
		for j := range d.Statistics.FundingStatistics.Items[i].ReportItem.Snapshots {
			if d.Statistics.FundingStatistics.Items[i].ReportItem.Snapshots[j].Available.IsZero() {
				// highcharts can't render zeroes in logarithmic mode
				response.AxisType = "linear"
			}
			plots = append(plots, ChartPlot{
				Value:     d.Statistics.FundingStatistics.Items[i].ReportItem.Snapshots[j].Available.InexactFloat64(),
				UnixMilli: d.Statistics.FundingStatistics.Items[i].ReportItem.Snapshots[j].Time.UTC().UnixMilli(),
			})
		}
		response.Data = append(response.Data, ChartLine{
			Name:       fmt.Sprintf("%v %v %v holdings", d.Statistics.FundingStatistics.Items[i].ReportItem.Exchange, d.Statistics.FundingStatistics.Items[i].ReportItem.Asset, d.Statistics.FundingStatistics.Items[i].ReportItem.Currency),
			DataPoints: plots,
		})
	}

	return response
}

func (d *Data) createSpotFuturesDiffChart() *Chart {
	response := &Chart{}
	for exch, assetMap := range d.Statistics.ExchangeAssetPairStatistics {
		for item, pairMap := range assetMap {
			for pair, result := range pairMap {
				result.
			}
		}
	}
}