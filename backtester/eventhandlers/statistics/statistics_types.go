package statistics

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errExchangeAssetPairStatsUnset = errors.New("exchangeAssetPairStatistics not setup")
	errCurrencyStatisticsUnset     = errors.New("no data")
)

// Statistic holds all statistical information for a backtester run, from drawdowns to ratios.
// Any currency specific information is handled in currencystatistics
type Statistic struct {
	StrategyName                string                                                                            `json:"strategy-name"`
	StrategyDescription         string                                                                            `json:"strategy-description"`
	StrategyNickname            string                                                                            `json:"strategy-nickname"`
	StrategyGoal                string                                                                            `json:"strategy-goal"`
	ExchangeAssetPairStatistics map[string]map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyStatistic `json:"-"`
	RiskFreeRate                decimal.Decimal                                                                   `json:"risk-free-rate"`
	TotalBuyOrders              int64                                                                             `json:"total-buy-orders"`
	TotalSellOrders             int64                                                                             `json:"total-sell-orders"`
	TotalOrders                 int64                                                                             `json:"total-orders"`
	BiggestDrawdown             *FinalResultsHolder                                                               `json:"biggest-drawdown,omitempty"`
	BestStrategyResults         *FinalResultsHolder                                                               `json:"best-start-results,omitempty"`
	BestMarketMovement          *FinalResultsHolder                                                               `json:"best-market-movement,omitempty"`
	AllStats                    []currencystatistics.CurrencyStatistic                                            `json:"results"` // as ExchangeAssetPairStatistics cannot be rendered via json.Marshall, we append all result to this slice instead
	WasAnyDataMissing           bool                                                                              `json:"was-any-data-missing"`
	Funding                     *funding.Report                                                                   `json:"funding"`
}

// FinalResultsHolder holds important stats about a currency's performance
type FinalResultsHolder struct {
	Exchange         string                   `json:"exchange"`
	Asset            asset.Item               `json:"asset"`
	Pair             currency.Pair            `json:"currency"`
	MaxDrawdown      currencystatistics.Swing `json:"max-drawdown"`
	MarketMovement   decimal.Decimal          `json:"market-movement"`
	StrategyMovement decimal.Decimal          `json:"strategy-movement"`
}

// Handler interface details what a statistic is expected to do
type Handler interface {
	SetStrategyName(string)
	SetupEventForTime(common.DataEventHandler) error
	SetEventForOffset(common.EventHandler) error
	AddHoldingsForTime(*holdings.Holding) error
	AddComplianceSnapshotForTime(compliance.Snapshot, fill.Event) error
	CalculateAllResults(funding.IFundingManager) error
	Reset()
	Serialise() (string, error)
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
