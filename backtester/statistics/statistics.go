package statistics

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gonum.org/v1/gonum/stat"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	portfolio2 "github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	common2 "github.com/thrasher-corp/gocryptotrader/common"
)

// Update Statistic for event
func (s *Statistic) Update(d interfaces.DataEventHandler, p portfolio2.PortfolioHandler) {
	if s.InitialBuy == 0 {
		s.InitialBuy = p.GetInitialFunds() / d.Price()
	}

	e := EquityPoint{}
	e.Timestamp = d.GetTime()
	e.Equity = p.Value()

	e.BuyAndHoldValue = s.InitialBuy * d.Price()

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
			Price:     s.Transactions()[v].GetPrice(),
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

func (s *Statistic) PrintResult() {
	fmt.Printf("Counted %d total events.\n", len(s.Events()))

	fmt.Printf("Counted %d total transactions:\n", len(s.Transactions()))
	sb := strings.Builder{}

	butts := s.Transactions()
	for k, v := range butts {
		sb.WriteString(fmt.Sprintf("%v. ", k+1))
		sb.WriteString(fmt.Sprintf("%v\t", v.GetTime().Format(common2.SimpleTimeFormat)))
		sb.WriteString(fmt.Sprintf("%v\t", v.GetDirection()))
		if v.GetDirection() != common.DoNothing {
			sb.WriteString(fmt.Sprintf("%v @ ", v.GetAmount()))
			sb.WriteString(fmt.Sprintf("$%v\t", v.GetPrice()))
			sb.WriteString(fmt.Sprintf("Fee: $%v\t", v.GetExchangeFee()))
			sb.WriteString(fmt.Sprintf("Cost Basis: %v\t", v.GetPrice()*v.GetAmount()+v.GetExchangeFee()))
		} else {
			sb.WriteString("\t\t\t")
		}
		if v.GetWhy() != "" {
			sb.WriteString(fmt.Sprintf("Why: %v\t", v.GetWhy()))
		}
		sb.WriteString("\n")
	}

	fmt.Print(sb.String())
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
