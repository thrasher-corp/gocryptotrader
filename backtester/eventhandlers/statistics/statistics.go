package statistics

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/shopspring/decimal"
	"gonum.org/v1/gonum/stat"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatstics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// AddDataEventForTime sets up the big map for to store important data at each time interval
func (s *Statistic) AddDataEventForTime(e interfaces.DataEventHandler) {
	ex := e.GetExchange()
	a := e.GetAssetType()
	p := e.Pair()

	if s.EventsByTime[ex] == nil {
		s.EventsByTime[ex] = make(map[asset.Item]map[currency.Pair]currencystatstics.CurrencyStatistic)
	}
	if s.EventsByTime[ex][a] == nil {
		s.EventsByTime[ex][a] = make(map[currency.Pair]currencystatstics.CurrencyStatistic)
	}
	lookup := s.EventsByTime[ex][a][p]
	lookup.Events = append(lookup.Events,
		currencystatstics.EventStore{
			DataEvent: e,
		},
	)
	s.EventsByTime[ex][a][p] = lookup
}

// AddSignalEventForTime adds strategy signal event to the statistics at the time period
func (s *Statistic) AddSignalEventForTime(e signal.SignalEvent) {
	lookup := s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()]
	for i := range lookup.Events {
		if lookup.Events[i].DataEvent.GetTime().Equal(e.GetTime()) {
			lookup.Events[i].SignalEvent = e
			s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()] = lookup
		}
	}
}

// AddExchangeEventForTime adds exchange event to the statistics at the time period
func (s *Statistic) AddExchangeEventForTime(e order.OrderEvent) {
	lookup := s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()]
	for i := range lookup.Events {
		if lookup.Events[i].DataEvent.GetTime().Equal(e.GetTime()) {
			lookup.Events[i].ExchangeEvent = e
			s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()] = lookup
		}
	}
}

// AddFillEventForTime adds fill event to the statistics at the time period
func (s *Statistic) AddFillEventForTime(e fill.FillEvent) {
	lookup := s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()]
	for i := range lookup.Events {
		if lookup.Events[i].DataEvent.GetTime().Equal(e.GetTime()) {
			lookup.Events[i].FillEvent = e
			s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()] = lookup
		}
	}
}

// AddHoldingsForTime adds all holdings to the statistics at the time period
func (s *Statistic) AddHoldingsForTime(h holdings.Holding) {
	lookup := s.EventsByTime[h.Exchange][h.Asset][h.Pair]
	for i := range lookup.Events {
		if lookup.Events[i].DataEvent.GetTime().Equal(h.Timestamp) {
			lookup.Events[i].Holdings = h
			s.EventsByTime[h.Exchange][h.Asset][h.Pair] = lookup
		}
	}
}

// AddComplianceSnapshotForTime adds the compliance snapshot to the statistics at the time period
func (s *Statistic) AddComplianceSnapshotForTime(c compliance.Snapshot, e fill.FillEvent) {
	lookup := s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()]
	for i := range lookup.Events {
		if lookup.Events[i].DataEvent.GetTime().Equal(c.Time) {
			lookup.Events[i].Transactions = c
			s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()] = lookup
		}
	}
}

func (s *Statistic) CalculateTheResults() error {
	log.Info(log.BackTester, "------------------Events-------------------------------------")

	for e, x := range s.EventsByTime {
		for a, y := range x {
			for p, z := range y {
				z.CalculateResults(s.SharpeRatioRiskFreeRate)
				z.PrintResults(e, a, p)
			}
		}
	}
	// todo, do a big final stats output

	return nil
}

// Update Statistic for event
func (s *Statistic) Update(d interfaces.DataEventHandler, p portfolio.Handler) {
	if s.InitialBuy == 0 && d.Price() > 0 {
		s.InitialBuy = p.GetInitialFunds(d.GetExchange(), d.GetAssetType(), d.Pair()) / d.Price()
	}

	e := EquityPoint{}
	e.Timestamp = d.GetTime()
	//e.Equity = p.Value()

	e.BuyAndHoldValue = s.InitialBuy * d.Price()

	if len(s.Equity) > 0 {
		e = s.calcEquityReturn(e)
	}

	if e.Equity >= s.High.Equity {
		s.High = e
	}
	if e.Equity <= s.Low.Equity {
		s.Low = e
	}

	s.Equity = append(s.Equity, e)
}

// TrackEvent event adds current event to History for Statistic calculation
func (s *Statistic) TrackEvent(e interfaces.EventHandler) {
	s.EventHistory = append(s.EventHistory, e)
}

// Events returns list of events
func (s *Statistic) Events() []interfaces.EventHandler {
	return s.EventHistory
}

// TrackTransaction add current transaction (trade) to History for Statistic
func (s *Statistic) TrackTransaction(f fill.FillEvent) {
	if f == nil {
		return
	}
	s.TransactionHistory = append(s.TransactionHistory, f)
}

// Transactions() returns list of transctions
func (s *Statistic) Transactions() []fill.FillEvent {
	return s.TransactionHistory
}

// Reset statistics
func (s *Statistic) Reset() {
	s.EventHistory = nil
	s.TransactionHistory = nil
	s.Equity = nil
	s.High = EquityPoint{}
	s.Low = EquityPoint{}
}

// ReturnResults will return Results for current backtest run
func (s *Statistic) ReturnResults() Results {
	results := Results{
		TotalEvents:       len(s.Events()),
		TotalTransactions: len(s.Transactions()),
		SharpieRatio:      s.SharpeRatio(0),
		StrategyName:      s.StrategyName,
	}

	for v := range s.Transactions() {
		results.Transactions = append(results.Transactions, ResultTransactions{
			Time:      s.Transactions()[v].GetTime(),
			Direction: s.Transactions()[v].GetDirection(),
			Price:     s.Transactions()[v].GetClosePrice(),
			Amount:    s.Transactions()[v].GetAmount(),
			Why:       s.Transactions()[v].GetWhy(),
		})
	}
	for v := range s.Events() {
		results.Events = append(results.Events, ResultEvent{
			Time: s.Events()[v].GetTime(),
		})
	}
	return results
}

func roundIt(r float64) float64 {
	return math.Round(r*100000000) / 100000000

}

func (s *Statistic) PrintResult() {
	fmt.Printf("Counted %d total events.\n", len(s.Events()))

	fmt.Printf("Counted %d total transactions:\n", len(s.Transactions()))
	sb := strings.Builder{}

	transactions := s.Transactions()
	for k, v := range transactions {
		sb.WriteString(fmt.Sprintf("%v.\t", k+1))
		sb.WriteString(fmt.Sprintf("%v\t", v.GetTime().Format(gctcommon.SimpleTimeFormat)))
		sb.WriteString(fmt.Sprintf("%v\t", v.GetDirection()))
		if v.GetDirection() != common.DoNothing {
			sb.WriteString(fmt.Sprintf("Amount: %f, Price: ", roundIt(v.GetAmount())))
			sb.WriteString(fmt.Sprintf("$%f\t", roundIt(v.GetPurchasePrice())))
			sb.WriteString(fmt.Sprintf("Fee: $%f\t", roundIt(v.GetExchangeFee())))
		} else {
			sb.WriteString("\t\t\t")
		}
		if v.GetWhy() != "" {
			sb.WriteString(fmt.Sprintf("Why: %v\t", v.GetWhy()))
		}
		sb.WriteString("\n")
	}

	fmt.Print(sb.String())
	fmt.Printf("Initial funds: $%f\nValue at enddate %v:\t$%f\n",
		roundIt(s.InitialFunds),
		s.Equity[len(s.Equity)-1].Timestamp.Format(gctcommon.SimpleTimeFormat),
		roundIt(s.Equity[len(s.Equity)-1].BuyAndHoldValue))
	fmt.Printf("Difference: $%f\n", roundIt(s.Equity[len(s.Equity)-1].BuyAndHoldValue-s.InitialFunds))
}

func (s *Statistic) TotalEquityReturn() (r float64, err error) {
	firstEquityPoint, ok := s.firstEquityPoint()
	if !ok {
		return r, errors.New("could not calculate totalEquityReturn, no equity points found")
	}
	if firstEquityPoint.Equity == 0 {
		return 0, errors.New("equity zero")
	}
	firstEquity := decimal.NewFromFloat(firstEquityPoint.Equity)

	lastEquityPoint, _ := s.lastEquityPoint()
	lastEquity := decimal.NewFromFloat(lastEquityPoint.Equity)

	totalEquityReturn := lastEquity.Sub(firstEquity).Div(firstEquity)
	total, _ := totalEquityReturn.Round(common.DecimalPlaces).Float64()
	return total, nil
}

// SharpeRatio returns sharpe ratio of backtest compared to risk-free
func (s *Statistic) SharpeRatio(riskfree float64) float64 {
	var equityReturns = make([]float64, len(s.Equity))

	for i := range s.Equity {
		equityReturns[i] = s.Equity[i].EquityReturn
	}
	mean, stddev := stat.MeanStdDev(equityReturns, nil)

	return (mean - riskfree) / stddev
}

func (s *Statistic) SortinoRatio(riskfree float64) float64 {
	var equityReturns = make([]float64, len(s.Equity))

	for i, v := range s.Equity {
		equityReturns[i] = v.EquityReturn
	}
	mean := stat.Mean(equityReturns, nil)

	var negReturns []float64
	for _, v := range equityReturns {
		if v < 0 {
			negReturns = append(negReturns, v)
		}
	}
	stdDev := stat.StdDev(negReturns, nil)
	return (mean - riskfree) / stdDev
}

// ViewEquityHistory returns a equity History list
func (s *Statistic) ViewEquityHistory() []EquityPoint {
	return s.Equity
}

func (s *Statistic) firstEquityPoint() (ep EquityPoint, ok bool) {
	if len(s.Equity) == 0 {
		return ep, false
	}
	ep = s.Equity[0]

	return ep, true
}

func (s *Statistic) lastEquityPoint() (ep EquityPoint, ok bool) {
	if len(s.Equity) == 0 {
		return ep, false
	}
	ep = s.Equity[len(s.Equity)-1]

	return ep, true
}

func (s *Statistic) calcEquityReturn(e EquityPoint) EquityPoint {
	last, ok := s.lastEquityPoint()
	if !ok {
		e.EquityReturn = 0
		return e
	}

	lastEquity := decimal.NewFromFloat(last.Equity)
	currentEquity := decimal.NewFromFloat(e.Equity)

	if lastEquity.Equal(decimal.Zero) {
		e.EquityReturn = 1
		return e
	}

	equityReturn := currentEquity.Sub(lastEquity).Div(lastEquity)
	e.EquityReturn, _ = equityReturn.Round(common.DecimalPlaces).Float64()

	return e
}

func (s *Statistic) JSON(writeFile bool) ([]byte, error) {
	output, err := json.MarshalIndent(s.ReturnResults(), "", " ")
	if err != nil {
		return []byte{}, err
	}

	if writeFile {
		f, err := os.Create(s.StrategyName + ".json")
		if err != nil {
			return []byte{}, nil
		}
		_, err = f.Write(output)
		if err != nil {
			return []byte{}, nil
		}
		err = f.Close()
		if err != nil {
			fmt.Println(err)
		}
	}
	return output, nil
}

func (s *Statistic) SetStrategyName(name string) {
	s.StrategyName = name
}
