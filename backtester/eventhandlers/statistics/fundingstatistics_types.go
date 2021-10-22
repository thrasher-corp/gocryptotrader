package statistics

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
)

type relatedStat struct {
	isBaseCurrency bool
	stat           *CurrencyPairStatistic
}

type FundingStatistics struct {
	UsingExchangeLevelFunding bool
	Report                    *funding.Report
	Items                     []FundingItemStatistics
	TotalUSDStatistics        *TotalFundingStatistics
}

type FundingItemStatistics struct {
	ReportItem *funding.ReportItem
	// USD stats
	StartingClosePrice       ValueAtTime
	EndingClosePrice         ValueAtTime
	LowestClosePrice         ValueAtTime
	HighestClosePrice        ValueAtTime
	MarketMovement           decimal.Decimal
	StrategyMovement         decimal.Decimal
	DidStrategyBeatTheMarket bool
	RiskFreeRate             decimal.Decimal
	CompoundAnnualGrowthRate decimal.Decimal
	// Extra stats
	BuyOrders             int64
	SellOrders            int64
	TotalOrders           int64
	MaxDrawdown           Swing
	HighestCommittedFunds ValueAtTime
}

type TotalFundingStatistics struct {
	HoldingValues            []ValueAtTime
	InitialHoldingValue      ValueAtTime
	FinalHoldingValue        ValueAtTime
	HighestHoldingValue      ValueAtTime
	LowestHoldingValue       ValueAtTime
	BenchmarkMarketMovement  decimal.Decimal
	StrategyMovement         decimal.Decimal
	RiskFreeRate             decimal.Decimal
	CompoundAnnualGrowthRate decimal.Decimal
	BuyOrders                int64
	SellOrders               int64
	TotalOrders              int64
	MaxDrawdown              Swing
	GeometricRatios          *Ratios
	ArithmeticRatios         *Ratios
	DidStrategyBeatTheMarket bool
	DidStrategyMakeProfit    bool
	// Used in report template

	HoldingValueDifference decimal.Decimal
}
