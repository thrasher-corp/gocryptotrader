package statistics

import (
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctmath "github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
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
		exchangeAssetStats := currStats[report.Items[i].Exchange][report.Items[i].Asset]
		var relevantStats []relatedCurrencyPairStatistics
		for k, v := range exchangeAssetStats {
			if k.Base == report.Items[i].Currency {
				relevantStats = append(relevantStats, relatedCurrencyPairStatistics{isBaseCurrency: true, stat: v})
				continue
			}
			if k.Quote == report.Items[i].Currency {
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
	holdingValues := make(map[time.Time]decimal.Decimal)

	for i := range response.Items {
		usdStats.TotalOrders += response.Items[i].TotalOrders
		usdStats.BuyOrders += response.Items[i].BuyOrders
		usdStats.SellOrders += response.Items[i].SellOrders
		for j := range response.Items[i].ReportItem.Snapshots {
			lookup := holdingValues[response.Items[i].ReportItem.Snapshots[j].Time]
			lookup = lookup.Add(response.Items[i].ReportItem.Snapshots[j].USDValue)
			holdingValues[response.Items[i].ReportItem.Snapshots[j].Time] = lookup
		}
	}
	for k, v := range holdingValues {
		if usdStats.HighestHoldingValue.Value.LessThan(v) {
			usdStats.HighestHoldingValue.Time = k
			usdStats.HighestHoldingValue.Value = v.Round(2)
		}
		if usdStats.LowestHoldingValue.Value.IsZero() {
			usdStats.LowestHoldingValue.Time = k
			usdStats.LowestHoldingValue.Value = v.Round(2)
		}
		if usdStats.LowestHoldingValue.Value.GreaterThan(v) && !usdStats.LowestHoldingValue.Value.IsZero() {
			usdStats.LowestHoldingValue.Time = k
			usdStats.LowestHoldingValue.Value = v.Round(2)
		}
		usdStats.HoldingValues = append(usdStats.HoldingValues, ValueAtTime{Time: k, Value: v})
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
	usdStats.MaxDrawdown = CalculateBiggestValueAtTimeDrawdown(usdStats.HoldingValues, interval)
	var err error
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
		if closePrices[i].USDClosePrice.LessThan(item.LowestClosePrice.Value) || item.LowestClosePrice.Value.IsZero() {
			item.LowestClosePrice.Value = closePrices[i].USDClosePrice
			item.LowestClosePrice.Time = closePrices[i].Time
		}
		if closePrices[i].USDClosePrice.GreaterThan(item.HighestClosePrice.Value) || item.HighestClosePrice.Value.IsZero() {
			item.HighestClosePrice.Value = closePrices[i].USDClosePrice
			item.HighestClosePrice.Time = closePrices[i].Time
		}
	}

	for i := range relatedStats {
		if relatedStats[i].stat == nil {
			return nil, fmt.Errorf("%w related stats", common.ErrNilArguments)
		}
		if relatedStats[i].isBaseCurrency {
			item.BuyOrders += relatedStats[i].stat.BuyOrders
			item.SellOrders += relatedStats[i].stat.SellOrders
		} else {
			item.BuyOrders += relatedStats[i].stat.SellOrders
			item.SellOrders += relatedStats[i].stat.BuyOrders
		}
	}
	item.TotalOrders = item.BuyOrders + item.SellOrders
	if !item.ReportItem.ShowInfinite {
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
	item.DidStrategyBeatTheMarket = item.StrategyMovement.GreaterThan(item.MarketMovement)
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
	var err error
	item.MaxDrawdown, err = CalculateBiggestEventDrawdown(s)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (f *FundingStatistics) PrintResults(wasAnyDataMissing bool) error {
	if f.Report == nil {
		return fmt.Errorf("%w requires report to be generated", common.ErrNilArguments)
	}
	log.Info(log.BackTester, "------------------Funding------------------------------------")
	log.Info(log.BackTester, "------------------Funding Items------------------------------")
	for i := range f.Report.Items {
		log.Infof(log.BackTester, "Exchange: %v", f.Report.Items[i].Exchange)
		log.Infof(log.BackTester, "Asset: %v", f.Report.Items[i].Asset)
		log.Infof(log.BackTester, "Currency: %v", f.Report.Items[i].Currency)
		if !f.Report.Items[i].PairedWith.IsEmpty() {
			log.Infof(log.BackTester, "Paired with: %v", f.Report.Items[i].PairedWith)
		}
		log.Infof(log.BackTester, "Initial funds: %v", f.Report.Items[i].InitialFunds)
		log.Infof(log.BackTester, "Final funds: %v", f.Report.Items[i].FinalFunds)
		if !f.Report.DisableUSDTracking && f.Report.UsingExchangeLevelFunding {
			log.Infof(log.BackTester, "Initial funds in USD: $%v", f.Report.Items[i].USDInitialFunds)
			log.Infof(log.BackTester, "Final funds in USD: $%v", f.Report.Items[i].USDFinalFunds)
		}
		if f.Report.Items[i].ShowInfinite {
			log.Info(log.BackTester, "Difference: ∞%")
		} else {
			log.Infof(log.BackTester, "Difference: %v%%", f.Report.Items[i].Difference)
		}
		if f.Report.Items[i].TransferFee.GreaterThan(decimal.Zero) {
			log.Infof(log.BackTester, "Transfer fee: %v", f.Report.Items[i].TransferFee)
		}
		if f.Report.DisableUSDTracking || !f.Report.UsingExchangeLevelFunding {
			continue
		}
	}
	if f.Report.DisableUSDTracking || !f.Report.UsingExchangeLevelFunding {
		return nil
	}
	log.Info(log.BackTester, "------------------Funding-Totals-----------------------------")
	log.Infof(log.BackTester, "Benchmark Market Movement: %v%%", f.TotalUSDStatistics.BenchmarkMarketMovement)
	log.Infof(log.BackTester, "Strategy Movement: %v%%", f.TotalUSDStatistics.StrategyMovement)
	log.Infof(log.BackTester, "Did strategy make a profit: %v", f.TotalUSDStatistics.DidStrategyMakeProfit)
	log.Infof(log.BackTester, "Did strategy beat the benchmark: %v", f.TotalUSDStatistics.DidStrategyBeatTheMarket)
	log.Infof(log.BackTester, "Buy Orders: %v", f.TotalUSDStatistics.BuyOrders)
	log.Infof(log.BackTester, "Sell Orders: %v", f.TotalUSDStatistics.SellOrders)
	log.Infof(log.BackTester, "Total Orders: %v", f.TotalUSDStatistics.TotalOrders)
	log.Infof(log.BackTester, "Highest funds: %v at %v", f.TotalUSDStatistics.HighestHoldingValue.Value, f.TotalUSDStatistics.HighestHoldingValue.Time)
	log.Infof(log.BackTester, "Lowest funds: %v at %v", f.TotalUSDStatistics.LowestHoldingValue.Value, f.TotalUSDStatistics.LowestHoldingValue.Time)

	log.Info(log.BackTester, "------------------Rates-------------------------------------------------")
	log.Infof(log.BackTester, "Risk free rate: %v%%", f.TotalUSDStatistics.RiskFreeRate.Mul(decimal.NewFromInt(100)).Round(2))
	log.Infof(log.BackTester, "Compound Annual Growth Rate: %v%%", f.TotalUSDStatistics.CompoundAnnualGrowthRate)

	if f.TotalUSDStatistics.ArithmeticRatios == nil || f.TotalUSDStatistics.GeometricRatios == nil {
		return fmt.Errorf("%w missing ratio calculations", common.ErrNilArguments)
	}
	log.Info(log.BackTester, "------------------Ratios------------------------------------------------")
	log.Info(log.BackTester, "------------------Arithmetic--------------------------------------------")
	if wasAnyDataMissing {
		log.Infoln(log.BackTester, "Missing data was detected during this backtesting run")
		log.Infoln(log.BackTester, "Ratio calculations will be skewed")
	}
	log.Infof(log.BackTester, "Sharpe ratio: %v", f.TotalUSDStatistics.ArithmeticRatios.SharpeRatio.Round(4))
	log.Infof(log.BackTester, "Sortino ratio: %v", f.TotalUSDStatistics.ArithmeticRatios.SortinoRatio.Round(4))
	log.Infof(log.BackTester, "Information ratio: %v", f.TotalUSDStatistics.ArithmeticRatios.InformationRatio.Round(4))
	log.Infof(log.BackTester, "Calmar ratio: %v\n\n", f.TotalUSDStatistics.ArithmeticRatios.CalmarRatio.Round(4))

	log.Info(log.BackTester, "------------------Geometric--------------------------------------------")
	if wasAnyDataMissing {
		log.Infoln(log.BackTester, "Missing data was detected during this backtesting run")
		log.Infoln(log.BackTester, "Ratio calculations will be skewed")
	}
	log.Infof(log.BackTester, "Sharpe ratio: %v", f.TotalUSDStatistics.GeometricRatios.SharpeRatio.Round(4))
	log.Infof(log.BackTester, "Sortino ratio: %v", f.TotalUSDStatistics.GeometricRatios.SortinoRatio.Round(4))
	log.Infof(log.BackTester, "Information ratio: %v", f.TotalUSDStatistics.GeometricRatios.InformationRatio.Round(4))
	log.Infof(log.BackTester, "Calmar ratio: %v\n\n", f.TotalUSDStatistics.GeometricRatios.CalmarRatio.Round(4))

	return nil
}
