package statistics

import (
	"errors"
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

func CalculateTotalUSDFundingStatistics(report *funding.Report, currStats map[string]map[asset.Item]map[currency.Pair]*CurrencyPairStatistic) (*FundingStatistics, error) {
	if currStats == nil {
		return nil, common.ErrNilArguments
	}
	var interval *gctkline.Interval
	var rfr decimal.Decimal
	response := &FundingStatistics{
		Report: report,
	}
	for i := range report.Items {
		exchangeAssetStats := currStats[report.Items[i].Exchange][report.Items[i].Asset]
		var relevantStats []relatedStat
		for k, v := range exchangeAssetStats {
			if k.Base == report.Items[i].Currency {
				if rfr.IsZero() {
					rfr = v.RiskFreeRate
				}
				if interval == nil {
					dei := v.Events[0].DataEvent.GetInterval()
					interval = &dei
				}
				relevantStats = append(relevantStats, relatedStat{isBaseCurrency: true, stat: v})
				continue
			}
			if k.Quote == report.Items[i].Currency {
				relevantStats = append(relevantStats, relatedStat{stat: v})
			}
		}
		fundingStat, err := CalculateIndividualFundingStatistics(&report.Items[i], relevantStats)
		if err != nil {
			return nil, err
		}
		response.Items = append(response.Items, *fundingStat)
	}

	usdStats := &TotalFundingStatistics{
		HighestHoldingValue: ValueAtTime{},
		LowestHoldingValue:  ValueAtTime{},
		RiskFreeRate:        rfr,
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
			usdStats.HighestHoldingValue.Value = v
		}
		if usdStats.LowestHoldingValue.Value.GreaterThan(v) {
			usdStats.LowestHoldingValue.Time = k
			usdStats.LowestHoldingValue.Value = v
		}
		usdStats.HoldingValues = append(usdStats.HoldingValues, ValueAtTime{Time: k, Value: v})
	}
	sort.Slice(usdStats.HoldingValues, func(i, j int) bool {
		return usdStats.HoldingValues[i].Value.LessThan(usdStats.HoldingValues[j].Value)
	})

	if !usdStats.HoldingValues[0].Value.IsZero() {
		usdStats.StrategyMovement = usdStats.HoldingValues[len(usdStats.HoldingValues)-1].Value.Sub(
			usdStats.HoldingValues[0].Value).Div(
			usdStats.HoldingValues[0].Value).Mul(
			decimal.NewFromInt(100))
	}

	returnsPerCandle := make([]decimal.Decimal, len(usdStats.HoldingValues))
	benchmarkRates := make([]decimal.Decimal, len(usdStats.HoldingValues))
	for j := range usdStats.HoldingValues {
		if j != 0 && !usdStats.HoldingValues[j-1].Value.IsZero() {
			benchmarkRates[j] = usdStats.HoldingValues[j].Value.Sub(usdStats.HoldingValues[j-1].Value).Div(usdStats.HoldingValues[j-1].Value)
			returnsPerCandle[j] = usdStats.HoldingValues[j].Value.Sub(usdStats.HoldingValues[j-1].Value).Div(usdStats.HoldingValues[j-1].Value)
		}
	}
	benchmarkRates = benchmarkRates[1:]
	returnsPerCandle = returnsPerCandle[1:]
	usdStats.BenchmarkMarketMovement = benchmarkRates[len(benchmarkRates)-1].Sub(benchmarkRates[0]).Div(benchmarkRates[0]).Mul(decimal.NewFromInt(100))
	var err error
	var arithmeticBenchmarkAverage, geometricBenchmarkAverage decimal.Decimal
	arithmeticBenchmarkAverage, err = gctmath.DecimalArithmeticMean(benchmarkRates)
	if err != nil {
		return nil, err
	}
	geometricBenchmarkAverage, err = gctmath.DecimalFinancialGeometricMean(benchmarkRates)
	if err != nil {
		return nil, err
	}

	riskFreeRatePerCandle := usdStats.RiskFreeRate.Div(decimal.NewFromFloat(interval.IntervalsPerYear()))
	riskFreeRateForPeriod := riskFreeRatePerCandle.Mul(decimal.NewFromInt(int64(len(benchmarkRates))))

	var arithmeticReturnsPerCandle, geometricReturnsPerCandle, arithmeticSharpe, arithmeticSortino,
		arithmeticInformation, arithmeticCalmar, geomSharpe, geomSortino, geomInformation, geomCalmar decimal.Decimal

	arithmeticReturnsPerCandle, err = gctmath.DecimalArithmeticMean(returnsPerCandle)
	if err != nil {
		return nil, err
	}
	geometricReturnsPerCandle, err = gctmath.DecimalFinancialGeometricMean(returnsPerCandle)
	if err != nil {
		return nil, err
	}

	arithmeticSharpe, err = gctmath.DecimalSharpeRatio(returnsPerCandle, riskFreeRatePerCandle, arithmeticReturnsPerCandle)
	if err != nil {
		return nil, err
	}
	arithmeticSortino, err = gctmath.DecimalSortinoRatio(returnsPerCandle, riskFreeRatePerCandle, arithmeticReturnsPerCandle)
	if err != nil && !errors.Is(err, gctmath.ErrNoNegativeResults) {
		if errors.Is(err, gctmath.ErrInexactConversion) {
			log.Warnf(log.BackTester, "USD Totals |\t funding arithmetic sortino ratio %v", err)
		} else {
			return nil, err
		}
	}
	arithmeticInformation, err = gctmath.DecimalInformationRatio(returnsPerCandle, benchmarkRates, arithmeticReturnsPerCandle, arithmeticBenchmarkAverage)
	if err != nil {
		return nil, err
	}
	usdStats.MaxDrawdown = CalculateBiggestValueAtTimeDrawdown(usdStats.HoldingValues, *interval)
	mxhp := usdStats.MaxDrawdown.Highest.Value
	mdlp := usdStats.MaxDrawdown.Lowest.Value
	arithmeticCalmar, err = gctmath.DecimalCalmarRatio(mxhp, mdlp, arithmeticReturnsPerCandle, riskFreeRateForPeriod)
	if err != nil {
		return nil, err
	}

	usdStats.ArithmeticRatios = Ratios{}
	if !arithmeticSharpe.IsZero() {
		usdStats.ArithmeticRatios.SharpeRatio = arithmeticSharpe
	}
	if !arithmeticSortino.IsZero() {
		usdStats.ArithmeticRatios.SortinoRatio = arithmeticSortino
	}
	if !arithmeticInformation.IsZero() {
		usdStats.ArithmeticRatios.InformationRatio = arithmeticInformation
	}
	if !arithmeticCalmar.IsZero() {
		usdStats.ArithmeticRatios.CalmarRatio = arithmeticCalmar
	}

	geomSharpe, err = gctmath.DecimalSharpeRatio(returnsPerCandle, riskFreeRatePerCandle, geometricReturnsPerCandle)
	if err != nil {
		return nil, err
	}
	geomSortino, err = gctmath.DecimalSortinoRatio(returnsPerCandle, riskFreeRatePerCandle, geometricReturnsPerCandle)
	if err != nil && !errors.Is(err, gctmath.ErrNoNegativeResults) {
		if errors.Is(err, gctmath.ErrInexactConversion) {
			log.Warnf(log.BackTester, "USD Totals |\t geometric sortino ratio %v", err)
		} else {
			return nil, err
		}
	}
	geomInformation, err = gctmath.DecimalInformationRatio(returnsPerCandle, benchmarkRates, geometricReturnsPerCandle, geometricBenchmarkAverage)
	if err != nil {
		return nil, err
	}
	geomCalmar, err = gctmath.DecimalCalmarRatio(mxhp, mdlp, geometricReturnsPerCandle, riskFreeRateForPeriod)
	if err != nil {
		return nil, err
	}
	usdStats.GeometricRatios = Ratios{}
	if !arithmeticSharpe.IsZero() {
		usdStats.GeometricRatios.SharpeRatio = geomSharpe
	}
	if !arithmeticSortino.IsZero() {
		usdStats.GeometricRatios.SortinoRatio = geomSortino
	}
	if !arithmeticInformation.IsZero() {
		usdStats.GeometricRatios.InformationRatio = geomInformation
	}
	if !arithmeticCalmar.IsZero() {
		usdStats.GeometricRatios.CalmarRatio = geomCalmar
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

func CalculateIndividualFundingStatistics(reportItem *funding.ReportItem, relevantStats []relatedStat) (*FundingItemStatistics, error) {
	if len(relevantStats) == 0 {
		// continue or error for being unrelated
		return nil, fmt.Errorf("somehow this has happened")
	}
	interval := relevantStats[0].stat.Events[0].DataEvent.GetInterval()
	item := &FundingItemStatistics{
		ReportItem:   reportItem,
		RiskFreeRate: relevantStats[0].stat.RiskFreeRate,
	}
	closePrices := reportItem.Snapshots
	sep := fmt.Sprintf("%v %v %v |\t", item.ReportItem.Exchange, item.ReportItem.Asset, item.ReportItem.Currency)
	item.StartingClosePrice = closePrices[0].USDClosePrice
	item.EndingClosePrice = closePrices[len(closePrices)-1].USDClosePrice
	for j := range closePrices {
		if closePrices[j].USDClosePrice.LessThan(item.LowestClosePrice) || item.LowestClosePrice.IsZero() {
			item.LowestClosePrice = closePrices[j].USDClosePrice
		}
		if closePrices[j].USDClosePrice.GreaterThan(item.HighestClosePrice) || item.HighestClosePrice.IsZero() {
			item.HighestClosePrice = closePrices[j].USDClosePrice
		}
	}
	item.MarketMovement = item.EndingClosePrice.Sub(item.StartingClosePrice).Div(item.StartingClosePrice).Mul(decimal.NewFromInt(100))
	for j := range relevantStats {
		if relevantStats[j].isBaseCurrency {
			item.BuyOrders += relevantStats[j].stat.BuyOrders
			item.SellOrders += relevantStats[j].stat.SellOrders
		} else {
			item.BuyOrders += relevantStats[j].stat.SellOrders
			item.SellOrders += relevantStats[j].stat.BuyOrders
		}
	}
	item.TotalOrders = item.BuyOrders + item.SellOrders
	if !item.ReportItem.ShowInfinite {
		item.StrategyMovement = item.ReportItem.Snapshots[len(item.ReportItem.Snapshots)-1].USDValue.Sub(
			item.ReportItem.Snapshots[0].USDValue).Div(
			item.ReportItem.Snapshots[0].USDValue).Mul(
			decimal.NewFromInt(100))
	}
	item.MarketMovement = item.ReportItem.Snapshots[len(item.ReportItem.Snapshots)-1].USDClosePrice.Sub(
		item.ReportItem.Snapshots[0].USDClosePrice).Div(
		item.ReportItem.Snapshots[0].USDClosePrice).Mul(
		decimal.NewFromInt(100))
	item.DidStrategyBeatTheMarket = item.StrategyMovement.GreaterThan(item.MarketMovement)
	item.HighestCommittedFunds = ValueAtTime{}
	returnsPerCandle := make([]decimal.Decimal, len(item.ReportItem.Snapshots))
	benchmarkRates := make([]decimal.Decimal, len(item.ReportItem.Snapshots))
	for j := range item.ReportItem.Snapshots {
		if item.ReportItem.Snapshots[j].USDValue.GreaterThan(item.HighestCommittedFunds.Value) {
			item.HighestCommittedFunds = ValueAtTime{
				Time:  item.ReportItem.Snapshots[j].Time,
				Value: item.ReportItem.Snapshots[j].USDValue,
			}
		}
		if j != 0 && !item.ReportItem.Snapshots[j-1].USDValue.IsZero() {
			returnsPerCandle[j] = item.ReportItem.Snapshots[j].USDValue.Sub(item.ReportItem.Snapshots[j-1].USDValue).Div(item.ReportItem.Snapshots[j-1].USDValue)
			benchmarkRates[j] = item.ReportItem.Snapshots[j].USDClosePrice.Sub(
				item.ReportItem.Snapshots[j-1].USDClosePrice).Div(
				item.ReportItem.Snapshots[j-1].USDClosePrice)
		}
	}
	var err error
	// remove the first entry as its zero and impacts
	// ratio calculations as no movement has been made
	benchmarkRates = benchmarkRates[1:]
	returnsPerCandle = returnsPerCandle[1:]
	var arithmeticBenchmarkAverage, geometricBenchmarkAverage decimal.Decimal
	arithmeticBenchmarkAverage, err = gctmath.DecimalArithmeticMean(benchmarkRates)
	if err != nil {
		return nil, err
	}
	geometricBenchmarkAverage, err = gctmath.DecimalFinancialGeometricMean(benchmarkRates)
	if err != nil {
		return nil, err
	}

	intervalsPerYear := interval.IntervalsPerYear()
	riskFreeRatePerCandle := item.RiskFreeRate.Div(decimal.NewFromFloat(intervalsPerYear))
	riskFreeRateForPeriod := riskFreeRatePerCandle.Mul(decimal.NewFromInt(int64(len(benchmarkRates))))

	var arithmeticReturnsPerCandle, geometricReturnsPerCandle, arithmeticSharpe, arithmeticSortino,
		arithmeticInformation, arithmeticCalmar, geomSharpe, geomSortino, geomInformation, geomCalmar decimal.Decimal

	arithmeticReturnsPerCandle, err = gctmath.DecimalArithmeticMean(returnsPerCandle)
	if err != nil {
		return nil, err
	}
	geometricReturnsPerCandle, err = gctmath.DecimalFinancialGeometricMean(returnsPerCandle)
	if err != nil {
		return nil, err
	}

	arithmeticSharpe, err = gctmath.DecimalSharpeRatio(returnsPerCandle, riskFreeRatePerCandle, arithmeticReturnsPerCandle)
	if err != nil {
		return nil, err
	}
	arithmeticSortino, err = gctmath.DecimalSortinoRatio(returnsPerCandle, riskFreeRatePerCandle, arithmeticReturnsPerCandle)
	if err != nil && !errors.Is(err, gctmath.ErrNoNegativeResults) {
		if errors.Is(err, gctmath.ErrInexactConversion) {
			log.Warnf(log.BackTester, "%v funding arithmetic sortino ratio %v", sep, err)
		} else {
			return nil, err
		}
	}
	arithmeticInformation, err = gctmath.DecimalInformationRatio(returnsPerCandle, benchmarkRates, arithmeticReturnsPerCandle, arithmeticBenchmarkAverage)
	if err != nil {
		return nil, err
	}
	item.MaxDrawdown = CalculateBiggestEventDrawdown(item.ReportItem.USDPairCandle.GetStream())
	mxhp := item.MaxDrawdown.Highest.Value
	mdlp := item.MaxDrawdown.Lowest.Value
	arithmeticCalmar, err = gctmath.DecimalCalmarRatio(mxhp, mdlp, arithmeticReturnsPerCandle, riskFreeRateForPeriod)
	if err != nil {
		return nil, err
	}

	item.ArithmeticRatios = Ratios{}
	if !arithmeticSharpe.IsZero() {
		item.ArithmeticRatios.SharpeRatio = arithmeticSharpe
	}
	if !arithmeticSortino.IsZero() {
		item.ArithmeticRatios.SortinoRatio = arithmeticSortino
	}
	if !arithmeticInformation.IsZero() {
		item.ArithmeticRatios.InformationRatio = arithmeticInformation
	}
	if !arithmeticCalmar.IsZero() {
		item.ArithmeticRatios.CalmarRatio = arithmeticCalmar
	}

	geomSharpe, err = gctmath.DecimalSharpeRatio(returnsPerCandle, riskFreeRatePerCandle, geometricReturnsPerCandle)
	if err != nil {
		return nil, err
	}
	geomSortino, err = gctmath.DecimalSortinoRatio(returnsPerCandle, riskFreeRatePerCandle, geometricReturnsPerCandle)
	if err != nil && !errors.Is(err, gctmath.ErrNoNegativeResults) {
		if errors.Is(err, gctmath.ErrInexactConversion) {
			log.Warnf(log.BackTester, "%v geometric sortino ratio %v", sep, err)
		} else {
			return nil, err
		}
	}
	geomInformation, err = gctmath.DecimalInformationRatio(returnsPerCandle, benchmarkRates, geometricReturnsPerCandle, geometricBenchmarkAverage)
	if err != nil {
		return nil, err
	}
	geomCalmar, err = gctmath.DecimalCalmarRatio(mxhp, mdlp, geometricReturnsPerCandle, riskFreeRateForPeriod)
	if err != nil {
		return nil, err
	}
	item.GeometricRatios = Ratios{}
	if !arithmeticSharpe.IsZero() {
		item.GeometricRatios.SharpeRatio = geomSharpe
	}
	if !arithmeticSortino.IsZero() {
		item.GeometricRatios.SortinoRatio = geomSortino
	}
	if !arithmeticInformation.IsZero() {
		item.GeometricRatios.InformationRatio = geomInformation
	}
	if !arithmeticCalmar.IsZero() {
		item.GeometricRatios.CalmarRatio = geomCalmar
	}

	if !item.ReportItem.InitialFunds.IsZero() {
		cagr, err := gctmath.DecimalCompoundAnnualGrowthRate(
			item.ReportItem.USDInitialFunds,
			item.ReportItem.USDFinalFunds,
			decimal.NewFromFloat(intervalsPerYear),
			decimal.NewFromInt(int64(len(item.ReportItem.Snapshots))),
		)
		if err != nil {
			return nil, err
		}
		if !cagr.IsZero() {
			item.CompoundAnnualGrowthRate = cagr
		}
	}
	return item, nil
}
