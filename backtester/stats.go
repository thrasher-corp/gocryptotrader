package backtest

import (
	"math"
	"time"

	"gonum.org/v1/gonum/stat"
)

func (s *Statistic) Update(d DataEvent, p PortfolioHandler) {
	e := equityPoint{}
	e.timestamp = d.Time()
	e.equity = p.Value()

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

func (s *Statistic) TrackEvent(e EventHandler) {
	s.eventHistory = append(s.eventHistory, e)
}

func (s Statistic) Events() []EventHandler {
	return s.eventHistory
}

func (s *Statistic) TrackTransaction(f OrderEvent) {
	s.transactionHistory = append(s.transactionHistory, f)
}

func (s Statistic) Transactions() []OrderEvent {
	return s.transactionHistory
}

func (s *Statistic) Reset() {
	s.eventHistory = nil
	s.transactionHistory = nil
	s.equity = nil
	s.high = equityPoint{}
	s.low = equityPoint{}
}

func (s Statistic) PrintResult() Results {
	results := Results{
		TotalEvents:       len(s.Events()),
		TotalTransactions: len(s.Transactions()),
		SharpieRatio:      s.SharpRatio(0),
	}
	for v := range s.Transactions() {
		results.Transactions = append(results.Transactions, resultTransactions{
			time:      s.Transactions()[v].Time(),
			direction: s.Transactions()[v].Direction(),
			price:     s.Transactions()[v].Price(),
			amount:    s.Transactions()[v].Amount(),
		})
	}
	return results
}

func (s Statistic) TotalEquityReturn() (r float64, err error) {
	firstEquityPoint, _ := s.firstEquityPoint()
	firstEquity := firstEquityPoint.equity

	lastEquityPoint, _ := s.lastEquityPoint()
	lastEquity := lastEquityPoint.equity

	totalEquityReturn := (lastEquity - firstEquity) / firstEquity
	total := math.Round(totalEquityReturn*math.Pow10(DP)) / math.Pow10(DP)
	return total, nil
}

func (s Statistic) MaxDrawdown() float64 {
	_, ep := s.maxDrawdownPoint()
	return ep.drawdown
}

func (s Statistic) MaxDrawdownTime() time.Time {
	_, ep := s.maxDrawdownPoint()
	return ep.timestamp
}

func (s Statistic) MaxDrawdownDuration() (d time.Duration) {
	i, ep := s.maxDrawdownPoint()

	if len(s.equity) == 0 {
		return d
	}

	maxPoint := equityPoint{}
	for index := i; index >= 0; index-- {
		if s.equity[index].equity > maxPoint.equity {
			maxPoint = s.equity[index]
		}
	}

	d = ep.timestamp.Sub(maxPoint.timestamp)
	return d
}

func (s *Statistic) SharpRatio(riskfree float64) float64 {
	var equityReturns = make([]float64, len(s.equity))

	for i := range s.equity {
		equityReturns[i] = s.equity[i].equityReturn
	}
	mean, stddev := stat.MeanStdDev(equityReturns, nil)

	return (mean - riskfree) / stddev
}

func (s *Statistic) SortinoRatio(riskfree float64) float64 {
	var equityReturns = make([]float64, len(s.equity))

	for i := range s.equity {
		equityReturns[i] = s.equity[i].equityReturn
	}
	mean := stat.Mean(equityReturns, nil)

	var negReturns []float64
	for x := range equityReturns {
		if equityReturns[x] < 0 {
			negReturns = append(negReturns, equityReturns[x])
		}
	}
	return (mean - riskfree) / stat.StdDev(negReturns, nil)
}

func (s Statistic) firstEquityPoint() (ep equityPoint, ok bool) {
	if len(s.equity) <= 0 {
		return ep, false
	}
	ep = s.equity[0]

	return ep, true
}

func (s Statistic) lastEquityPoint() (ep equityPoint, ok bool) {
	if len(s.equity) <= 0 {
		return ep, false
	}
	ep = s.equity[len(s.equity)-1]

	return ep, true
}

func (s Statistic) calcEquityReturn(e equityPoint) equityPoint {
	last, ok := s.lastEquityPoint()
	if !ok {
		e.equityReturn = 0
		return e
	}

	lastEquity := last.equity
	currentEquity := e.equity

	if lastEquity == 0 {
		e.equityReturn = 1
		return e
	}

	equityReturn := (currentEquity - lastEquity) / lastEquity
	e.equityReturn = math.Round(equityReturn*math.Pow10(DP)) / math.Pow10(DP)

	return e
}

func (s Statistic) calcDrawdown(e equityPoint) equityPoint {
	if s.high.equity == 0 {
		e.drawdown = 0
		return e
	}

	lastHigh := s.high.equity
	equity := e.equity

	if equity >= lastHigh {
		e.drawdown = 0
		return e
	}

	drawdown := (equity - lastHigh) / lastHigh
	e.drawdown = math.Round(drawdown*math.Pow10(DP)) / math.Pow10(DP)

	return e
}

func (s Statistic) maxDrawdownPoint() (i int, ep equityPoint) {
	if len(s.equity) == 0 {
		return 0, ep
	}

	var maxDrawdown = 0.0
	var index = 0

	for i, ep := range s.equity {
		if ep.drawdown < maxDrawdown {
			maxDrawdown = ep.drawdown
			index = i
		}
	}

	return index, s.equity[index]
}

func (s *Statistic) GetEquity() *[]equityPoint {
	return &s.equity
}
