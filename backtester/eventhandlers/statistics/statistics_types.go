package statistics

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatstics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Statistic
type Statistic struct {
	StrategyName                string
	ExchangeAssetPairStatistics map[string]map[asset.Item]map[currency.Pair]*currencystatstics.CurrencyStatistic `json:"-"`
	RiskFreeRate                float64                                                                          `json:"risk-free-rate"`
	TotalBuyOrders              int64                                                                            `json:"total-buy-orders"`
	TotalSellOrders             int64                                                                            `json:"total-sell-orders"`
	TotalOrders                 int64                                                                            `json:"total-orders"`
	BiggestDrawdown             *FinalResultsHolder                                                              `json:"biggest-drawdown,omitempty"`
	BestStrategyResults         *FinalResultsHolder                                                              `json:"best-strat-results,omitempty"`
	BestMarketMovement          *FinalResultsHolder                                                              `json:"best-market-movement,omitempty"`
	AllStats                    []currencystatstics.CurrencyStatistic                                            `json:"results"`
}

type FinalResultsHolder struct {
	E                string                  `json:"exchange"`
	A                asset.Item              `json:"asset"`
	P                currency.Pair           `json:"currency"`
	MaxDrawdown      currencystatstics.Swing `json:"max-drawdown"`
	MarketMovement   float64                 `json:"market-movement"`
	StrategyMovement float64                 `json:"strategy-movement"`
}

// Handler interface handles
type Handler interface {
	SetStrategyName(string)
	AddDataEventForTime(interfaces.DataEventHandler)
	AddSignalEventForTime(signal.SignalEvent)
	AddExchangeEventForTime(order.OrderEvent)
	AddFillEventForTime(fill.FillEvent)
	AddHoldingsForTime(holdings.Holding)
	AddComplianceSnapshotForTime(compliance.Snapshot, fill.FillEvent)
	CalculateTheResults() error
	Reset()
	Serialise() string
}

type Results struct {
	Pair              string               `json:"pair"`
	TotalEvents       int                  `json:"totalEvents"`
	TotalTransactions int                  `json:"totalTransactions"`
	Events            []ResultEvent        `json:"events"`
	Transactions      []ResultTransactions `json:"transactions"`
	SharpieRatio      float64              `json:"sharpieRatio"`
	StrategyName      string               `json:"strategyName"`
}

type ResultTransactions struct {
	Time      time.Time     `json:"time"`
	Direction gctorder.Side `json:"direction"`
	Price     float64       `json:"price"`
	Amount    float64       `json:"amount"`
	Why       string        `json:"why,omitempty"`
}

type ResultEvent struct {
	Time time.Time `json:"time"`
}
