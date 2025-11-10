package statistics

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/common/key"
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
	StrategyName                string                                           `json:"strategy-name"`
	StrategyDescription         string                                           `json:"strategy-description"`
	StrategyNickname            string                                           `json:"strategy-nickname"`
	StrategyGoal                string                                           `json:"strategy-goal"`
	StartDate                   time.Time                                        `json:"start-date"`
	EndDate                     time.Time                                        `json:"end-date"`
	CandleInterval              gctkline.Interval                                `json:"candle-interval"`
	RiskFreeRate                decimal.Decimal                                  `json:"risk-free-rate"`
	ExchangeAssetPairStatistics map[key.ExchangeAssetPair]*CurrencyPairStatistic `json:"-"`
	CurrencyStatistics          []*CurrencyPairStatistic                         `json:"currency-statistics"`
	TotalBuyOrders              int64                                            `json:"total-buy-orders"`
	TotalLongOrders             int64                                            `json:"total-long-orders"`
	TotalShortOrders            int64                                            `json:"total-short-orders"`
	TotalSellOrders             int64                                            `json:"total-sell-orders"`
	TotalOrders                 int64                                            `json:"total-orders"`
	BiggestDrawdown             *FinalResultsHolder                              `json:"biggest-drawdown,omitempty"`
	BestStrategyResults         *FinalResultsHolder                              `json:"best-start-results,omitempty"`
	BestMarketMovement          *FinalResultsHolder                              `json:"best-market-movement,omitempty"`
	WasAnyDataMissing           bool                                             `json:"was-any-data-missing"`
	FundingStatistics           *FundingStatistics                               `json:"funding-statistics"`
	FundManager                 funding.IFundingManager                          `json:"-"`
	HasCollateral               bool                                             `json:"has-collateral"`
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
	SetEventForOffset(common.Event) error
	AddHoldingsForTime(*holdings.Holding) error
	AddComplianceSnapshotForTime(*compliance.Snapshot, common.Event) error
	CalculateAllResults() error
	Reset() error
	Serialise() (string, error)
	AddPNLForTime(*portfolio.PNLSummary) error
	CreateLog(common.Event) (string, error)
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
	Offset             int64
	ClosePrice         decimal.Decimal
	Time               time.Time
	Holdings           holdings.Holding
	ComplianceSnapshot *compliance.Snapshot
	DataEvent          data.Event
	SignalEvent        signal.Event
	OrderEvent         order.Event
	FillEvent          fill.Event
	PNL                portfolio.IPNL
}

// CurrencyPairStatistic Holds all events and statistics relevant to an exchange, asset type and currency pair
type CurrencyPairStatistic struct {
	Exchange       string
	Asset          asset.Item
	Currency       currency.Pair
	UnderlyingPair currency.Pair `json:"linked-spot-currency"`

	ShowMissingDataWarning       bool `json:"-"`
	IsStrategyProfitable         bool `json:"is-strategy-profitable"`
	DoesPerformanceBeatTheMarket bool `json:"does-performance-beat-the-market"`

	BuyOrders   int64 `json:"buy-orders"`
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
	TotalAssetValue              decimal.Decimal `json:"total-asset-value"`
	TotalFees                    decimal.Decimal `json:"total-fees"`
	TotalValueLostToVolumeSizing decimal.Decimal `json:"total-value-lost-to-volume-sizing"`
	TotalValueLostToSlippage     decimal.Decimal `json:"total-value-lost-to-slippage"`
	TotalValueLost               decimal.Decimal `json:"total-value-lost"`

	Events []DataAtOffset `json:"-"`

	MaxDrawdown           Swing               `json:"max-drawdown"`
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
	IntervalDuration int64           `json:"interval-duration"`
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
	Report             *funding.Report         `json:"-"`
	Items              []FundingItemStatistics `json:"funding-item-statistics"`
	TotalUSDStatistics *TotalFundingStatistics `json:"total-usd-statistics"`
}

// FundingItemStatistics holds statistics for funding items
type FundingItemStatistics struct {
	ReportItem *funding.ReportItem `json:"-"`
	// USD stats
	StartingClosePrice       ValueAtTime     `json:"starting-close-price"`
	EndingClosePrice         ValueAtTime     `json:"ending-close-price"`
	LowestClosePrice         ValueAtTime     `json:"lowest-close-price"`
	HighestClosePrice        ValueAtTime     `json:"highest-close-price"`
	MarketMovement           decimal.Decimal `json:"market-movement"`
	StrategyMovement         decimal.Decimal `json:"strategy-movement"`
	DidStrategyBeatTheMarket bool            `json:"did-strategy-beat-the-market"`
	RiskFreeRate             decimal.Decimal `json:"risk-free-rate"`
	CompoundAnnualGrowthRate decimal.Decimal `json:"compound-annual-growth-rate"`
	BuyOrders                int64           `json:"buy-orders"`
	SellOrders               int64           `json:"sell-orders"`
	TotalOrders              int64           `json:"total-orders"`
	MaxDrawdown              Swing           `json:"max-drawdown"`
	HighestCommittedFunds    ValueAtTime     `json:"highest-committed-funds"`
	// CollateralPair stats
	IsCollateral      bool        `json:"is-collateral"`
	InitialCollateral ValueAtTime `json:"initial-collateral"`
	FinalCollateral   ValueAtTime `json:"final-collateral"`
	HighestCollateral ValueAtTime `json:"highest-collateral"`
	LowestCollateral  ValueAtTime `json:"lowest-collateral"`
	// Contracts
	LowestHoldings  ValueAtTime `json:"lowest-holdings"`
	HighestHoldings ValueAtTime `json:"highest-holdings"`
	InitialHoldings ValueAtTime `json:"initial-holdings"`
	FinalHoldings   ValueAtTime `json:"final-holdings"`
}

// TotalFundingStatistics holds values for overall statistics for funding items
type TotalFundingStatistics struct {
	HoldingValues            []ValueAtTime   `json:"-"`
	HighestHoldingValue      ValueAtTime     `json:"highest-holding-value"`
	LowestHoldingValue       ValueAtTime     `json:"lowest-holding-value"`
	BenchmarkMarketMovement  decimal.Decimal `json:"benchmark-market-movement"`
	RiskFreeRate             decimal.Decimal `json:"risk-free-rate"`
	CompoundAnnualGrowthRate decimal.Decimal `json:"compound-annual-growth-rate"`
	MaxDrawdown              Swing           `json:"max-drawdown"`
	GeometricRatios          *Ratios         `json:"geometric-ratios"`
	ArithmeticRatios         *Ratios         `json:"arithmetic-ratios"`
	DidStrategyBeatTheMarket bool            `json:"did-strategy-beat-the-market"`
	DidStrategyMakeProfit    bool            `json:"did-strategy-make-profit"`
	HoldingValueDifference   decimal.Decimal `json:"holding-value-difference"`
}
