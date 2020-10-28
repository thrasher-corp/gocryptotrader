package statistics

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/shopspring/decimal"
	"gonum.org/v1/gonum/stat"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/fill"
	portfolio2 "github.com/thrasher-corp/gocryptotrader/backtester/portfolio"
	results2 "github.com/thrasher-corp/gocryptotrader/backtester/results"
)

// Update Statistic for event
func (s *Statistic) Update(d portfolio.DataEventHandler, p portfolio2.PortfolioHandler) {
	if s.InitialBuy == 0 {
		s.InitialBuy = p.GetInitialFunds() / d.LatestPrice()
	}

	e := EquityPoint{}
	e.Timestamp = d.GetTime()
	e.Equity = p.Value()

	e.BuyAndHoldValue = s.InitialBuy * d.LatestPrice()

	if len(s.Equity) > 0 {
		e = s.calcEquityReturn(e)
	}

	if len(s.Equity) > 0 {
		e = s.calcDrawdown(e)
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
func (s *Statistic) TrackEvent(e portfolio.EventHandler) {
	s.EventHistory = append(s.EventHistory, e)
}

// Events returns list of events
func (s *Statistic) Events() []portfolio.EventHandler {
	return s.EventHistory
}

// TrackTransaction add current transaction (trade) to History for Statistic
func (s *Statistic) TrackTransaction(f fill.FillEvent) {
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
func (s *Statistic) ReturnResults() results2.Results {
	results := results2.Results{
		TotalEvents:       len(s.Events()),
		TotalTransactions: len(s.Transactions()),
		SharpieRatio:      s.SharpeRatio(0),
		StrategyName:      s.StrategyName,
		Pair:              s.Pair,
	}
	for v := range s.Transactions() {
		results.Transactions = append(results.Transactions, results2.ResultTransactions{
			Time:      s.Transactions()[v].GetTime(),
			Direction: s.Transactions()[v].GetDirection(),
			Price:     s.Transactions()[v].GetPrice(),
			Amount:    s.Transactions()[v].GetAmount(),
		})
	}
	for v := range s.Events() {
		results.Events = append(results.Events, results2.ResultEvent{
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
	firstEquity := decimal.NewFromFloat(firstEquityPoint.Equity)

	lastEquityPoint, _ := s.lastEquityPoint()
	lastEquity := decimal.NewFromFloat(lastEquityPoint.Equity)

	totalEquityReturn := lastEquity.Sub(firstEquity).Div(firstEquity)
	total, _ := totalEquityReturn.Round(common.DecimalPlaces).Float64()
	return total, nil
}

func (s *Statistic) MaxDrawdown() float64 {
	_, ep := s.maxDrawdownPoint()
	return ep.DrawnDown
}

func (s *Statistic) MaxDrawdownTime() time.Time {
	_, ep := s.maxDrawdownPoint()
	return ep.Timestamp
}

func (s *Statistic) MaxDrawdownDuration() time.Duration {
	i, ep := s.maxDrawdownPoint()

	if len(s.Equity) == 0 {
		return 0
	}

	maxPoint := EquityPoint{}
	for index := i; index >= 0; index-- {
		if s.Equity[index].Equity > maxPoint.Equity {
			maxPoint = s.Equity[index]
		}
	}

	return ep.Timestamp.Sub(maxPoint.Timestamp)
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

func (s *Statistic) calcDrawdown(e EquityPoint) EquityPoint {
	if s.High.Equity == 0 {
		e.DrawnDown = 0
		return e
	}

	lastHigh := decimal.NewFromFloat(s.High.Equity)
	equity := decimal.NewFromFloat(e.Equity)

	if equity.GreaterThanOrEqual(lastHigh) {
		e.DrawnDown = 0
		return e
	}

	drawdown := equity.Sub(lastHigh).Div(lastHigh)
	e.DrawnDown, _ = drawdown.Round(common.DecimalPlaces).Float64()

	return e
}

func (s *Statistic) maxDrawdownPoint() (i int, ep EquityPoint) {
	if len(s.Equity) == 0 {
		return 0, ep
	}

	var maxDrawdown = 0.0
	var index = 0

	for i, ep := range s.Equity {
		if ep.DrawnDown < maxDrawdown {
			maxDrawdown = ep.DrawnDown
			index = i
		}
	}

	return index, s.Equity[index]
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
