package statistics

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	// ErrAlreadyProcessed occurs when an event has already been processed
	ErrAlreadyProcessed            = errors.New("this event has been processed already")
	errExchangeAssetPairStatsUnset = errors.New("exchangeAssetPairStatistics not setup")
	errCurrencyStatisticsUnset     = errors.New("no data")
	errMissingSnapshots            = errors.New("funding report item missing USD snapshots")
	errNoRelevantStatsFound        = errors.New("no relevant currency pair statistics found")
	errReceivedNoData              = errors.New("received no data")
	errNoDataAtOffset              = errors.New("no data found at offset")
)

// Statistic holds all statistical information for a backtester run, from drawdowns to ratios.
// Any currency specific information is handled in currencystatistics
type Statistic struct {
	StrategyName                string                                                             `json:"strategy-name"`
	StrategyDescription         string                                                             `json:"strategy-description"`
	StrategyNickname            string                                                             `json:"strategy-nickname"`
	StrategyGoal                string                                                             `json:"strategy-goal"`
	StartDate                   time.Time                                                          `json:"start-date"`
	EndDate                     time.Time                                                          `json:"end-date"`
	CandleInterval              gctkline.Interval                                                  `json:"candle-interval"`
	RiskFreeRate                decimal.Decimal                                                    `json:"risk-free-rate"`
	ExchangeAssetPairStatistics map[string]map[asset.Item]map[currency.Pair]*CurrencyPairStatistic `json:"exchange-asset-pair-statistics"`
	TotalBuyOrders              int64                                                              `json:"total-buy-orders"`
	TotalSellOrders             int64                                                              `json:"total-sell-orders"`
	TotalOrders                 int64                                                              `json:"total-orders"`
	BiggestDrawdown             *FinalResultsHolder                                                `json:"biggest-drawdown,omitempty"`
	BestStrategyResults         *FinalResultsHolder                                                `json:"best-start-results,omitempty"`
	BestMarketMovement          *FinalResultsHolder                                                `json:"best-market-movement,omitempty"`
	CurrencyPairStatistics      []CurrencyPairStatistic                                            `json:"currency-pair-statistics"` // as ExchangeAssetPairStatistics cannot be rendered via json.Marshall, we append all result to this slice instead
	WasAnyDataMissing           bool                                                               `json:"was-any-data-missing"`
	FundingStatistics           *FundingStatistics                                                 `json:"funding-statistics"`
	FundManager                 funding.IFundingManager                                            `json:"-"`
}

// FinalResultsHolder holds important stats about a currency's performance
type FinalResultsHolder struct {
	Exchange         string          `json:"exchange"`
	Asset            asset.Item      `json:"asset"`
	Pair             currency.Pair   `json:"currency"`
	MaxDrawdown      Swing           `json:"max-drawdown"`
	MarketMovement   decimal.Decimal `json:"market-movement"`
	StrategyMovement decimal.Decimal `json:"strategy-movement"`
}

// Handler interface details what a statistic is expected to do
type Handler interface {
	SetStrategyName(string)
	SetupEventForTime(common.DataEventHandler) error
	SetEventForOffset(common.EventHandler) error
	AddHoldingsForTime(*holdings.Holding) error
	AddComplianceSnapshotForTime(compliance.Snapshot, fill.Event) error
	CalculateAllResults() error
	Reset()
	Serialise() (string, error)
	AddPNLForTime(*portfolio.PNLSummary) error
}

// Results holds some statistics on results
type Results struct {
	Pair              string               `json:"pair"`
	TotalEvents       int                  `json:"totalEvents"`
	TotalTransactions int                  `json:"totalTransactions"`
	Events            []ResultEvent        `json:"events"`
	Transactions      []ResultTransactions `json:"transactions"`
	StrategyName      string               `json:"strategyName"`
}

// ResultTransactions stores details on a transaction
type ResultTransactions struct {
	Time      time.Time       `json:"time"`
	Direction gctorder.Side   `json:"direction"`
	Price     decimal.Decimal `json:"price"`
	Amount    decimal.Decimal `json:"amount"`
	Reason    string          `json:"reason,omitempty"`
}

// ResultEvent stores the time
type ResultEvent struct {
	Time time.Time `json:"time"`
}

type eventOutputHolder struct {
	Time   time.Time
	Events []string
}

// CurrencyStats defines what is expected in order to
// calculate statistics based on an exchange, asset type and currency pair
type CurrencyStats interface {
	TotalEquityReturn() (decimal.Decimal, error)
	MaxDrawdown() Swing
	LongestDrawdown() Swing
	SharpeRatio(decimal.Decimal) decimal.Decimal
	SortinoRatio(decimal.Decimal) decimal.Decimal
}

// DataAtOffset is used to hold all event information
// at a time interval
type DataAtOffset struct {
	Holdings     holdings.Holding
	Transactions compliance.Snapshot
	DataEvent    common.DataEventHandler
	SignalEvent  signal.Event
	OrderEvent   order.Event
	FillEvent    fill.Event
	// TODO: consider moving this to an interface so we aren't tied to this summary type
	PNL *portfolio.PNLSummary
}

// CurrencyPairStatistic Holds all events and statistics relevant to an exchange, asset type and currency pair
type CurrencyPairStatistic struct {
	Exchange string
	Asset    asset.Item
	Currency currency.Pair

	ShowMissingDataWarning       bool `json:"-"`
	IsStrategyProfitable         bool `json:"is-strategy-profitable"`
	DoesPerformanceBeatTheMarket bool `json:"does-performance-beat-the-market"`

	BuyOrders   int64 `json:"buy-orders"`
	LongOrders  int64 `json:"long-orders"`
	ShortOrders int64 `json:"short-orders"`
	SellOrders  int64 `json:"sell-orders"`
	TotalOrders int64 `json:"total-orders"`

	StartingClosePrice   ValueAtTime `json:"starting-close-price"`
	EndingClosePrice     ValueAtTime `json:"ending-close-price"`
	LowestClosePrice     ValueAtTime `json:"lowest-close-price"`
	HighestClosePrice    ValueAtTime `json:"highest-close-price"`
	HighestUnrealisedPNL ValueAtTime `json:"highest-unrealised-pnl"`
	LowestUnrealisedPNL  ValueAtTime `json:"lowest-unrealised-pnl"`
	HighestRealisedPNL   ValueAtTime `json:"highest-realised-pnl"`
	LowestRealisedPNL    ValueAtTime `json:"lowest-realised-pnl"`

	MarketMovement               decimal.Decimal `json:"market-movement"`
	StrategyMovement             decimal.Decimal `json:"strategy-movement"`
	UnrealisedPNL                decimal.Decimal `json:"unrealised-pnl"`
	RealisedPNL                  decimal.Decimal `json:"realised-pnl"`
	CompoundAnnualGrowthRate     decimal.Decimal `json:"compound-annual-growth-rate"`
	TotalAssetValue              decimal.Decimal
	TotalFees                    decimal.Decimal
	TotalValueLostToVolumeSizing decimal.Decimal
	TotalValueLostToSlippage     decimal.Decimal
	TotalValueLost               decimal.Decimal

	Events []DataAtOffset `json:"-"`

	MaxDrawdown           Swing               `json:"max-drawdown,omitempty"`
	HighestCommittedFunds ValueAtTime         `json:"highest-committed-funds"`
	GeometricRatios       *Ratios             `json:"geometric-ratios"`
	ArithmeticRatios      *Ratios             `json:"arithmetic-ratios"`
	InitialHoldings       holdings.Holding    `json:"initial-holdings-holdings"`
	FinalHoldings         holdings.Holding    `json:"final-holdings"`
	FinalOrders           compliance.Snapshot `json:"final-orders"`
}

// Ratios stores all the ratios used for statistics
type Ratios struct {
	SharpeRatio      decimal.Decimal `json:"sharpe-ratio"`
	SortinoRatio     decimal.Decimal `json:"sortino-ratio"`
	InformationRatio decimal.Decimal `json:"information-ratio"`
	CalmarRatio      decimal.Decimal `json:"calmar-ratio"`
}

// Swing holds a drawdown
type Swing struct {
	Highest          ValueAtTime     `json:"highest"`
	Lowest           ValueAtTime     `json:"lowest"`
	DrawdownPercent  decimal.Decimal `json:"drawdown"`
	IntervalDuration int64
}

// ValueAtTime is an individual iteration of price at a time
type ValueAtTime struct {
	Time  time.Time       `json:"time"`
	Value decimal.Decimal `json:"value"`
	Set   bool            `json:"-"`
}

type relatedCurrencyPairStatistics struct {
	isBaseCurrency bool
	stat           *CurrencyPairStatistic
}

// FundingStatistics stores all funding related statistics
type FundingStatistics struct {
	Report             *funding.Report
	Items              []FundingItemStatistics
	TotalUSDStatistics *TotalFundingStatistics
}

// FundingItemStatistics holds statistics for funding items
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
	BuyOrders                int64
	SellOrders               int64
	TotalOrders              int64
	MaxDrawdown              Swing
	HighestCommittedFunds    ValueAtTime
	// Collateral stats
	IsCollateral      bool
	InitialCollateral ValueAtTime
	FinalCollateral   ValueAtTime
	HighestCollateral ValueAtTime
	LowestCollateral  ValueAtTime
	// Contracts
	LowestHoldings  ValueAtTime
	HighestHoldings ValueAtTime
	InitialHoldings ValueAtTime
	FinalHoldings   ValueAtTime
}

// TotalFundingStatistics holds values for overall statistics for funding items
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
	LongOrders               int64
	ShortOrders              int64
	TotalOrders              int64
	MaxDrawdown              Swing
	GeometricRatios          *Ratios
	ArithmeticRatios         *Ratios
	DidStrategyBeatTheMarket bool
	DidStrategyMakeProfit    bool
	HoldingValueDifference   decimal.Decimal
}
