package statistics

import (
	"fmt"
	"sort"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctmath "github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// CalculateFundingStatistics calculates funding statistics for total USD strategy results
// along with individual funding item statistics
func CalculateFundingStatistics(funds funding.IFundingManager, currStats map[string]map[asset.Item]map[currency.Pair]*CurrencyPairStatistic, riskFreeRate decimal.Decimal, interval gctkline.Interval) (*FundingStatistics, error) {
	if currStats == nil {
		return nil, common.ErrNilArguments
	}
	report := funds.GenerateReport()
	response := &FundingStatistics{
		Report: report,
	}
	for i := range report.Items {
		exchangeAssetStats, ok := currStats[report.Items[i].Exchange][report.Items[i].Asset]
		if !ok {
			return nil, fmt.Errorf("%w for %v %v",
				errNoRelevantStatsFound,
				report.Items[i].Exchange,
				report.Items[i].Asset)
		}
		var relevantStats []relatedCurrencyPairStatistics
		for k, v := range exchangeAssetStats {
			if k.Base.Equal(report.Items[i].Currency) {
				relevantStats = append(relevantStats, relatedCurrencyPairStatistics{isBaseCurrency: true, stat: v})
				continue
			}
			if k.Quote.Equal(report.Items[i].Currency) {
				relevantStats = append(relevantStats, relatedCurrencyPairStatistics{stat: v})
			}
		}
		fundingStat, err := CalculateIndividualFundingStatistics(report.DisableUSDTracking, &report.Items[i], relevantStats)
		if err != nil {
			return nil, err
		}
		response.Items = append(response.Items, *fundingStat)
	}
	if report.DisableUSDTracking {
		return response, nil
	}
	usdStats := &TotalFundingStatistics{
		HighestHoldingValue: ValueAtTime{},
		LowestHoldingValue:  ValueAtTime{},
		RiskFreeRate:        riskFreeRate,
	}
	for i := range response.Items {
		if response.Items[i].IsCollateral {
			continue
		}
		usdStats.TotalOrders += response.Items[i].TotalOrders
		usdStats.BuyOrders += response.Items[i].BuyOrders
		usdStats.SellOrders += response.Items[i].SellOrders
	}
	for k, v := range report.USDTotalsOverTime {
		if usdStats.HighestHoldingValue.Value.LessThan(v.USDValue) {
			usdStats.HighestHoldingValue.Time = k
			usdStats.HighestHoldingValue.Value = v.USDValue
		}
		if usdStats.LowestHoldingValue.Value.IsZero() {
			usdStats.LowestHoldingValue.Time = k
			usdStats.LowestHoldingValue.Value = v.USDValue
		}
		if usdStats.LowestHoldingValue.Value.GreaterThan(v.USDValue) && !usdStats.LowestHoldingValue.Value.IsZero() {
			usdStats.LowestHoldingValue.Time = k
			usdStats.LowestHoldingValue.Value = v.USDValue
		}
		usdStats.HoldingValues = append(usdStats.HoldingValues, ValueAtTime{Time: k, Value: v.USDValue})
	}
	sort.Slice(usdStats.HoldingValues, func(i, j int) bool {
		return usdStats.HoldingValues[i].Time.Before(usdStats.HoldingValues[j].Time)
	})

	if len(usdStats.HoldingValues) == 0 {
		return nil, fmt.Errorf("%w and holding values", errMissingSnapshots)
	}

	if !usdStats.HoldingValues[0].Value.IsZero() {
		usdStats.StrategyMovement = usdStats.HoldingValues[len(usdStats.HoldingValues)-1].Value.Sub(
			usdStats.HoldingValues[0].Value).Div(
			usdStats.HoldingValues[0].Value).Mul(
			decimal.NewFromInt(100))
	}
	usdStats.InitialHoldingValue = usdStats.HoldingValues[0]
	usdStats.FinalHoldingValue = usdStats.HoldingValues[len(usdStats.HoldingValues)-1]
	usdStats.HoldingValueDifference = usdStats.FinalHoldingValue.Value.Sub(usdStats.InitialHoldingValue.Value).Div(usdStats.InitialHoldingValue.Value).Mul(decimal.NewFromInt(100))

	riskFreeRatePerCandle := usdStats.RiskFreeRate.Div(decimal.NewFromFloat(interval.IntervalsPerYear()))
	returnsPerCandle := make([]decimal.Decimal, len(usdStats.HoldingValues))
	benchmarkRates := make([]decimal.Decimal, len(usdStats.HoldingValues))
	benchmarkMovement := usdStats.HoldingValues[0].Value
	benchmarkRates[0] = usdStats.HoldingValues[0].Value
	for j := range usdStats.HoldingValues {
		if j != 0 && !usdStats.HoldingValues[j-1].Value.IsZero() {
			benchmarkMovement = benchmarkMovement.Add(benchmarkMovement.Mul(riskFreeRatePerCandle))
			benchmarkRates[j] = riskFreeRatePerCandle
			returnsPerCandle[j] = usdStats.HoldingValues[j].Value.Sub(usdStats.HoldingValues[j-1].Value).Div(usdStats.HoldingValues[j-1].Value)
		}
	}
	benchmarkRates = benchmarkRates[1:]
	returnsPerCandle = returnsPerCandle[1:]
	usdStats.BenchmarkMarketMovement = benchmarkMovement.Sub(usdStats.HoldingValues[0].Value).Div(usdStats.HoldingValues[0].Value).Mul(decimal.NewFromInt(100))
	var err error
	usdStats.MaxDrawdown, err = CalculateBiggestValueAtTimeDrawdown(usdStats.HoldingValues, interval)
	if err != nil {
		return nil, err
	}

	sep := "USD Totals |\t"
	usdStats.ArithmeticRatios, usdStats.GeometricRatios, err = CalculateRatios(benchmarkRates, returnsPerCandle, riskFreeRatePerCandle, &usdStats.MaxDrawdown, sep)
	if err != nil {
		return nil, err
	}

	if !usdStats.HoldingValues[0].Value.IsZero() {
		cagr, err := gctmath.DecimalCompoundAnnualGrowthRate(
			usdStats.HoldingValues[0].Value,
			usdStats.HoldingValues[len(usdStats.HoldingValues)-1].Value,
			decimal.NewFromFloat(interval.IntervalsPerYear()),
			decimal.NewFromInt(int64(len(usdStats.HoldingValues))),
		)
		if err != nil {
			return nil, err
		}
		if !cagr.IsZero() {
			usdStats.CompoundAnnualGrowthRate = cagr
		}
	}
	usdStats.DidStrategyMakeProfit = usdStats.HoldingValues[len(usdStats.HoldingValues)-1].Value.GreaterThan(usdStats.HoldingValues[0].Value)
	usdStats.DidStrategyBeatTheMarket = usdStats.StrategyMovement.GreaterThan(usdStats.BenchmarkMarketMovement)
	response.TotalUSDStatistics = usdStats

	return response, nil
}

// CalculateIndividualFundingStatistics calculates statistics for an individual report item
func CalculateIndividualFundingStatistics(disableUSDTracking bool, reportItem *funding.ReportItem, relatedStats []relatedCurrencyPairStatistics) (*FundingItemStatistics, error) {
	if reportItem == nil {
		return nil, fmt.Errorf("%w - nil report item", common.ErrNilArguments)
	}
	item := &FundingItemStatistics{
		ReportItem: reportItem,
	}
	if disableUSDTracking {
		return item, nil
	}
	closePrices := reportItem.Snapshots
	if len(closePrices) == 0 {
		return nil, errMissingSnapshots
	}
	item.StartingClosePrice = ValueAtTime{
		Time:  closePrices[0].Time,
		Value: closePrices[0].USDClosePrice,
	}
	item.EndingClosePrice = ValueAtTime{
		Time:  closePrices[len(closePrices)-1].Time,
		Value: closePrices[len(closePrices)-1].USDClosePrice,
	}
	for i := range closePrices {
		if closePrices[i].USDClosePrice.LessThan(item.LowestClosePrice.Value) || !item.LowestClosePrice.Set {
			item.LowestClosePrice.Value = closePrices[i].USDClosePrice
			item.LowestClosePrice.Time = closePrices[i].Time
			item.LowestClosePrice.Set = true
		}
		if closePrices[i].USDClosePrice.GreaterThan(item.HighestClosePrice.Value) || !item.HighestClosePrice.Set {
			item.HighestClosePrice.Value = closePrices[i].USDClosePrice
			item.HighestClosePrice.Time = closePrices[i].Time
			item.HighestClosePrice.Set = true
		}
	}
	item.IsCollateral = reportItem.IsCollateral
	if reportItem.Asset.IsFutures() {
		var lowest, highest, initial, final ValueAtTime
		initial.Value = reportItem.Snapshots[0].Available
		initial.Time = reportItem.Snapshots[0].Time
		final.Value = reportItem.Snapshots[len(reportItem.Snapshots)-1].Available
		final.Time = reportItem.Snapshots[len(reportItem.Snapshots)-1].Time
		for i := range reportItem.Snapshots {
			if reportItem.Snapshots[i].Available.LessThan(lowest.Value) || !lowest.Set {
				lowest.Value = reportItem.Snapshots[i].Available
				lowest.Time = reportItem.Snapshots[i].Time
				lowest.Set = true
			}
			if reportItem.Snapshots[i].Available.GreaterThan(highest.Value) || !lowest.Set {
				highest.Value = reportItem.Snapshots[i].Available
				highest.Time = reportItem.Snapshots[i].Time
				highest.Set = true
			}
		}
		if reportItem.IsCollateral {
			item.LowestCollateral = lowest
			item.HighestCollateral = highest
			item.InitialCollateral = initial
			item.FinalCollateral = final
		} else {
			item.LowestHoldings = lowest
			item.HighestHoldings = highest
			item.InitialHoldings = initial
			item.FinalHoldings = final
		}
	}
	if !reportItem.IsCollateral {
		for i := range relatedStats {
			if relatedStats[i].stat == nil {
				return nil, fmt.Errorf("%w related stats", common.ErrNilArguments)
			}
			if relatedStats[i].isBaseCurrency {
				item.BuyOrders += relatedStats[i].stat.BuyOrders
				item.SellOrders += relatedStats[i].stat.SellOrders
			}
		}
	}
	item.TotalOrders = item.BuyOrders + item.SellOrders
	if !item.ReportItem.ShowInfinite && !reportItem.IsCollateral {
		if item.ReportItem.Snapshots[0].USDValue.IsZero() {
			item.ReportItem.ShowInfinite = true
		} else {
			item.StrategyMovement = item.ReportItem.Snapshots[len(item.ReportItem.Snapshots)-1].USDValue.Sub(
				item.ReportItem.Snapshots[0].USDValue).Div(
				item.ReportItem.Snapshots[0].USDValue).Mul(
				decimal.NewFromInt(100))
		}
	}

	if !item.ReportItem.Snapshots[0].USDClosePrice.IsZero() {
		item.MarketMovement = item.ReportItem.Snapshots[len(item.ReportItem.Snapshots)-1].USDClosePrice.Sub(
			item.ReportItem.Snapshots[0].USDClosePrice).Div(
			item.ReportItem.Snapshots[0].USDClosePrice).Mul(
			decimal.NewFromInt(100))
	}
	if !reportItem.IsCollateral {
		item.DidStrategyBeatTheMarket = item.StrategyMovement.GreaterThan(item.MarketMovement)
	}
	item.HighestCommittedFunds = ValueAtTime{}
	for j := range item.ReportItem.Snapshots {
		if item.ReportItem.Snapshots[j].USDValue.GreaterThan(item.HighestCommittedFunds.Value) {
			item.HighestCommittedFunds = ValueAtTime{
				Time:  item.ReportItem.Snapshots[j].Time,
				Value: item.ReportItem.Snapshots[j].USDValue,
			}
		}
	}
	if item.ReportItem.USDPairCandle == nil {
		return nil, fmt.Errorf("%w usd candles missing", errMissingSnapshots)
	}
	s := item.ReportItem.USDPairCandle.GetStream()
	if len(s) == 0 {
		return nil, fmt.Errorf("%w stream missing", errMissingSnapshots)
	}
	if reportItem.IsCollateral {
		return item, nil
	}
	var err error
	item.MaxDrawdown, err = CalculateBiggestEventDrawdown(s)
	return item, err
}
