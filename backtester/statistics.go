package backtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/shopspring/decimal"
	"gonum.org/v1/gonum/stat"
)

// Update statistic for event
func (s *Statistic) Update(d DataEventHandler, p PortfolioHandler) {
	if s.initialBuy == 0 {
		s.initialBuy = p.InitialFunds() / d.LatestPrice()
	}

	e := EquityPoint{}
	e.timestamp = d.GetTime()
	e.equity = p.Value()

	e.buyAndHoldValue = s.initialBuy * d.LatestPrice()

	if len(s.equity) > 0 {
		e = s.calcEquityReturn(e)
	}

	if len(s.equity) > 0 {
		e = s.calcDrawdown(e)
	}

	if e.equity >= s.high.equity {
		s.high = e
	}
	if e.equity <= s.low.equity {
		s.low = e
	}

	s.equity = append(s.equity, e)
}

// TrackEvent event adds current event to history for statistic calculation
func (s *Statistic) TrackEvent(e EventHandler) {
	s.eventHistory = append(s.eventHistory, e)
}

// Events returns list of events
func (s *Statistic) Events() []EventHandler {
	return s.eventHistory
}

// TrackTransaction add current transaction (trade) to history for statistic
func (s *Statistic) TrackTransaction(f FillEvent) {
	s.transactionHistory = append(s.transactionHistory, f)
}

// Transactions() returns list of transctions
func (s *Statistic) Transactions() []FillEvent {
	return s.transactionHistory
}

// Reset statistics
func (s *Statistic) Reset() {
	s.eventHistory = nil
	s.transactionHistory = nil
	s.equity = nil
	s.high = EquityPoint{}
	s.low = EquityPoint{}
}

// ReturnResults will return Results for current backtest run
func (s *Statistic) ReturnResults() Results {
	results := Results{
		TotalEvents:       len(s.Events()),
		TotalTransactions: len(s.Transactions()),
		SharpieRatio:      s.SharpeRatio(0),
		StrategyName:      s.strategyName,
		Pair:              s.pair,
	}
	for v := range s.Transactions() {
		results.Transactions = append(results.Transactions, resultTransactions{
			Time:      s.Transactions()[v].GetTime(),
			Direction: s.Transactions()[v].GetDirection(),
			Price:     s.Transactions()[v].GetPrice(),
			Amount:    s.Transactions()[v].GetAmount(),
		})
	}
	for v := range s.Events() {
		results.Events = append(results.Events, resultEvent{
			Time: s.Events()[v].GetTime(),
		})
	}
	return results
}

func (s *Statistic) PrintResult() {
	fmt.Printf("Counted %d total events.\n", len(s.Events()))

	fmt.Printf("Counted %d total transactions:\n", len(s.Transactions()))
	for k, v := range s.Transactions() {
		fmt.Printf("%d. Transaction: %v Action: %v Price: %f Amount %f\n", k+1, v.GetTime().Format("2006-01-02 15:04:05.999999999"), v.GetDirection(), v.GetPrice(), v.GetAmount())
	}

	result, _ := s.TotalEquityReturn()

	fmt.Println("TotalEquity:", result, "MaxDrawdown:", s.MaxDrawdown())
}

func (s *Statistic) TotalEquityReturn() (r float64, err error) {
	firstEquityPoint, ok := s.firstEquityPoint()
	if !ok {
		return r, errors.New("could not calculate totalEquityReturn, no equity points found")
	}
	firstEquity := decimal.NewFromFloat(firstEquityPoint.equity)

	lastEquityPoint, _ := s.lastEquityPoint()
	lastEquity := decimal.NewFromFloat(lastEquityPoint.equity)

	totalEquityReturn := lastEquity.Sub(firstEquity).Div(firstEquity)
	total, _ := totalEquityReturn.Round(DP).Float64()
	return total, nil
}

func (s *Statistic) MaxDrawdown() float64 {
	_, ep := s.maxDrawdownPoint()
	return ep.drawnDown
}

func (s *Statistic) MaxDrawdownTime() time.Time {
	_, ep := s.maxDrawdownPoint()
	return ep.timestamp
}

func (s *Statistic) MaxDrawdownDuration() time.Duration {
	i, ep := s.maxDrawdownPoint()

	if len(s.equity) == 0 {
		return 0
	}

	maxPoint := EquityPoint{}
	for index := i; index >= 0; index-- {
		if s.equity[index].equity > maxPoint.equity {
			maxPoint = s.equity[index]
		}
	}

	return ep.timestamp.Sub(maxPoint.timestamp)
}

// SharpeRatio returns sharpe ratio of backtest compared to risk-free
func (s *Statistic) SharpeRatio(riskfree float64) float64 {
	var equityReturns = make([]float64, len(s.equity))

	for i := range s.equity {
		equityReturns[i] = s.equity[i].equityReturn
	}
	mean, stddev := stat.MeanStdDev(equityReturns, nil)

	return (mean - riskfree) / stddev
}

func (s *Statistic) SortinoRatio(riskfree float64) float64 {
	var equityReturns = make([]float64, len(s.equity))

	for i, v := range s.equity {
		equityReturns[i] = v.equityReturn
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

// ViewEquityHistory returns a equity history list
func (s *Statistic) ViewEquityHistory() []EquityPoint {
	return s.equity
}

func (s *Statistic) firstEquityPoint() (ep EquityPoint, ok bool) {
	if len(s.equity) == 0 {
		return ep, false
	}
	ep = s.equity[0]

	return ep, true
}

func (s *Statistic) lastEquityPoint() (ep EquityPoint, ok bool) {
	if len(s.equity) == 0 {
		return ep, false
	}
	ep = s.equity[len(s.equity)-1]

	return ep, true
}

func (s *Statistic) calcEquityReturn(e EquityPoint) EquityPoint {
	last, ok := s.lastEquityPoint()
	if !ok {
		e.equityReturn = 0
		return e
	}

	lastEquity := decimal.NewFromFloat(last.equity)
	currentEquity := decimal.NewFromFloat(e.equity)

	if lastEquity.Equal(decimal.Zero) {
		e.equityReturn = 1
		return e
	}

	equityReturn := currentEquity.Sub(lastEquity).Div(lastEquity)
	e.equityReturn, _ = equityReturn.Round(DP).Float64()

	return e
}

func (s *Statistic) calcDrawdown(e EquityPoint) EquityPoint {
	if s.high.equity == 0 {
		e.drawnDown = 0
		return e
	}

	lastHigh := decimal.NewFromFloat(s.high.equity)
	equity := decimal.NewFromFloat(e.equity)

	if equity.GreaterThanOrEqual(lastHigh) {
		e.drawnDown = 0
		return e
	}

	drawdown := equity.Sub(lastHigh).Div(lastHigh)
	e.drawnDown, _ = drawdown.Round(DP).Float64()

	return e
}

func (s *Statistic) maxDrawdownPoint() (i int, ep EquityPoint) {
	if len(s.equity) == 0 {
		return 0, ep
	}

	var maxDrawdown = 0.0
	var index = 0

	for i, ep := range s.equity {
		if ep.drawnDown < maxDrawdown {
			maxDrawdown = ep.drawnDown
			index = i
		}
	}

	return index, s.equity[index]
}

func (s *Statistic) JSON(writeFile bool) ([]byte, error) {
	output, err := json.MarshalIndent(s.ReturnResults(), "", " ")
	if err != nil {
		return []byte{}, err
	}

	if writeFile {
		f, err := os.Create(s.strategyName+".json")
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
	s.strategyName = name
}
