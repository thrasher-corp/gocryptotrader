package fundingstatistics

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/common"
	gctmath "github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

type relatedStat struct {
	isBaseCurrency bool
	stat           *currencystatistics.CurrencyPairStatistic
}

func CalculateResults(f funding.IFundingManager, currStats map[string]map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyPairStatistic) *FundingStatistics {
	report := f.GenerateReport()
	var errs common.Errors
	response := &FundingStatistics{
		Report: report,
	}
	for i := range report.Items {
		exchangeAssetStats := currStats[report.Items[i].Exchange][report.Items[i].Asset]
		var relevantStats []relatedStat
		for k, v := range exchangeAssetStats {
			if k.Base == report.Items[i].Currency {
				relevantStats = append(relevantStats, relatedStat{isBaseCurrency: true, stat: v})
				continue
			}
			if k.Quote == report.Items[i].Currency {
				relevantStats = append(relevantStats, relatedStat{stat: v})
			}
		}
		if len(relevantStats) == 0 {
			// continue or error for being unrelated
			return nil
		}
		interval := relevantStats[0].stat.Events[0].DataEvent.GetInterval()
		item := FundingItemStatistics{
			ReportItem:   &report.Items[i],
			RiskFreeRate: relevantStats[0].stat.RiskFreeRate,
		}
		closePrices := report.Items[i].Snapshots
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
		item.HighestCommittedFunds = currencystatistics.HighestCommittedFunds{}
		returnsPerCandle := make([]decimal.Decimal, len(item.ReportItem.Snapshots))
		benchmarkRates := make([]decimal.Decimal, len(item.ReportItem.Snapshots))
		for j := range item.ReportItem.Snapshots {
			if item.ReportItem.Snapshots[j].USDValue.GreaterThan(item.HighestCommittedFunds.Value) {
				item.HighestCommittedFunds = currencystatistics.HighestCommittedFunds{
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
			errs = append(errs, err)
		}
		geometricBenchmarkAverage, err = gctmath.DecimalFinancialGeometricMean(benchmarkRates)
		if err != nil {
			errs = append(errs, err)
		}

		intervalsPerYear := interval.IntervalsPerYear()
		riskFreeRatePerCandle := item.RiskFreeRate.Div(decimal.NewFromFloat(intervalsPerYear))
		riskFreeRateForPeriod := riskFreeRatePerCandle.Mul(decimal.NewFromInt(int64(len(benchmarkRates))))

		var arithmeticReturnsPerCandle, geometricReturnsPerCandle, arithmeticSharpe, arithmeticSortino,
			arithmeticInformation, arithmeticCalmar, geomSharpe, geomSortino, geomInformation, geomCalmar decimal.Decimal

		arithmeticReturnsPerCandle, err = gctmath.DecimalArithmeticMean(returnsPerCandle)
		if err != nil {
			errs = append(errs, err)
		}
		geometricReturnsPerCandle, err = gctmath.DecimalFinancialGeometricMean(returnsPerCandle)
		if err != nil {
			errs = append(errs, err)
		}

		arithmeticSharpe, err = gctmath.DecimalSharpeRatio(returnsPerCandle, riskFreeRatePerCandle, arithmeticReturnsPerCandle)
		if err != nil {
			errs = append(errs, err)
		}
		arithmeticSortino, err = gctmath.DecimalSortinoRatio(returnsPerCandle, riskFreeRatePerCandle, arithmeticReturnsPerCandle)
		if err != nil && !errors.Is(err, gctmath.ErrNoNegativeResults) {
			if errors.Is(err, gctmath.ErrInexactConversion) {
				log.Warnf(log.BackTester, "%v funding arithmetic sortino ratio %v", sep, err)
			} else {
				errs = append(errs, err)
			}
		}
		arithmeticInformation, err = gctmath.DecimalInformationRatio(returnsPerCandle, benchmarkRates, arithmeticReturnsPerCandle, arithmeticBenchmarkAverage)
		if err != nil {
			errs = append(errs, err)
		}
		item.MaxDrawdown = currencystatistics.CalculateMaxDrawdown(item.ReportItem.USDPairCandle.GetStream())
		mxhp := item.MaxDrawdown.Highest.Price
		mdlp := item.MaxDrawdown.Lowest.Price
		arithmeticCalmar, err = gctmath.DecimalCalmarRatio(mxhp, mdlp, arithmeticReturnsPerCandle, riskFreeRateForPeriod)
		if err != nil {
			errs = append(errs, err)
		}

		item.ArithmeticRatios = currencystatistics.Ratios{}
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
			errs = append(errs, err)
		}
		geomSortino, err = gctmath.DecimalSortinoRatio(returnsPerCandle, riskFreeRatePerCandle, geometricReturnsPerCandle)
		if err != nil && !errors.Is(err, gctmath.ErrNoNegativeResults) {
			if errors.Is(err, gctmath.ErrInexactConversion) {
				log.Warnf(log.BackTester, "%v geometric sortino ratio %v", sep, err)
			} else {
				errs = append(errs, err)
			}
		}
		geomInformation, err = gctmath.DecimalInformationRatio(returnsPerCandle, benchmarkRates, geometricReturnsPerCandle, geometricBenchmarkAverage)
		if err != nil {
			errs = append(errs, err)
		}
		geomCalmar, err = gctmath.DecimalCalmarRatio(mxhp, mdlp, geometricReturnsPerCandle, riskFreeRateForPeriod)
		if err != nil {
			errs = append(errs, err)
		}
		item.GeometricRatios = currencystatistics.Ratios{}
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
				errs = append(errs, err)
			}
			if !cagr.IsZero() {
				item.CompoundAnnualGrowthRate = cagr
			}
		}
		response.Items = append(response.Items, item)
	}
	if len(errs) > 0 {
		log.Error(log.BackTester, errs)
	}
	return response
}

type FundingStatistics struct {
	Report *funding.Report
	Items  []FundingItemStatistics
}

type FundingItemStatistics struct {
	ReportItem *funding.ReportItem
	// USD stats
	StartingClosePrice       decimal.Decimal
	EndingClosePrice         decimal.Decimal
	LowestClosePrice         decimal.Decimal
	HighestClosePrice        decimal.Decimal
	MarketMovement           decimal.Decimal
	StrategyMovement         decimal.Decimal
	DidStrategyBeatTheMarket bool
	RiskFreeRate             decimal.Decimal
	CompoundAnnualGrowthRate decimal.Decimal
	// Extra stats
	BuyOrders             int64                                    `json:"buy-orders"`
	SellOrders            int64                                    `json:"sell-orders"`
	TotalOrders           int64                                    `json:"total-orders"`
	MaxDrawdown           currencystatistics.Swing                 `json:"max-drawdown,omitempty"`
	HighestCommittedFunds currencystatistics.HighestCommittedFunds `json:"highest-committed-funds"`
	GeometricRatios       currencystatistics.Ratios                `json:"geometric-ratios"`
	ArithmeticRatios      currencystatistics.Ratios                `json:"arithmetic-ratios"`
}
