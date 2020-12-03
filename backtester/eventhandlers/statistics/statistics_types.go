package statistics

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type EventStore struct {
	Holdings      holdings.Holding
	Transactions  compliance.Snapshot
	DataEvent     interfaces.DataEventHandler
	SignalEvent   signal.SignalEvent
	ExchangeEvent order.OrderEvent
	FillEvent     fill.FillEvent
}

// Statistic
type Statistic struct {
	EventsByTime map[string]map[asset.Item]map[currency.Pair][]EventStore
	///////////////////////////////////////////////
	EventHistory       []interfaces.EventHandler
	TransactionHistory []fill.FillEvent
	Equity             []EquityPoint
	High               EquityPoint
	Low                EquityPoint
	InitialBuy         float64
	InitialFunds       float64

	StrategyName string
}

type EquityPoint struct {
	Timestamp       time.Time
	Equity          float64
	EquityReturn    float64
	DrawnDown       float64
	BuyAndHoldValue float64
}

// Handler interface handles
type Handler interface {
	AddDataEventForTime(interfaces.DataEventHandler)
	AddSignalEventForTime(signal.SignalEvent)
	AddExchangeEventForTime(order.OrderEvent)
	AddFillEventForTime(fill.FillEvent)

	AddHoldingsForTime(holdings.Holding)
	AddComplianceSnapshotForTime(compliance.Snapshot, fill.FillEvent)

	CalculateTheResults() error
	//////////////////////////////////////
	TrackEvent(interfaces.EventHandler)
	Events() []interfaces.EventHandler

	Update(interfaces.DataEventHandler, portfolio.Handler)
	TrackTransaction(fill.FillEvent)
	Transactions() []fill.FillEvent

	TotalEquityReturn() (float64, error)

	MaxDrawdown() float64
	MaxDrawdownTime() time.Time
	MaxDrawdownDuration() time.Duration

	SharpeRatio(float64) float64
	SortinoRatio(float64) float64

	PrintResult()
	ReturnResults() Results
	Reset()

	SetStrategyName(string)
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
