package fundingstatistics

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type relatedStat struct {
	isBaseCurrency bool
	stat           *currencystatistics.CurrencyPairStatistic
}

func CalculateResults(f funding.IFundingManager, currStats map[string]map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyPairStatistic) *FundingStatistics {
	report := f.GenerateReport()
	response := &FundingStatistics{}
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
		item := FundingItemStatistics{
			ReportItem:   &report.Items[i],
			RiskFreeRate: relevantStats[0].stat.RiskFreeRate,
		}
		closePrices := report.Items[i].USDPairCandle.StreamClose()
		item.StartingClosePrice = closePrices[0]
		item.EndingClosePrice = closePrices[len(closePrices)-1]
		for j := range closePrices {
			if closePrices[j].LessThan(item.LowestClosePrice) || item.LowestClosePrice.IsZero() {
				item.LowestClosePrice = closePrices[j]
			}
			if closePrices[j].GreaterThan(item.HighestClosePrice) || item.HighestClosePrice.IsZero() {
				item.HighestClosePrice = closePrices[j]
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
		//  StrategyMovement:         decimal.Decimal{},
		//	RiskFreeRate:             decimal.Decimal{},
		//	CompoundAnnualGrowthRate: decimal.Decimal{},
		//	MaxDrawdown:              currencystatistics.Swing{},
		//	HighestCommittedFunds:    currencystatistics.HighestCommittedFunds{},
		//	GeometricRatios:          currencystatistics.Ratios{},
		//	ArithmeticRatios:         currencystatistics.Ratios{},
		response.Items = append(response.Items)
	}
	return response
}

type FundingStatistics struct {
	USDInitialTotal decimal.Decimal
	USDFinalTotal   decimal.Decimal
	Difference      decimal.Decimal
	Items           []FundingItemStatistics
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
