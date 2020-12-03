package statistics

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gonum.org/v1/gonum/stat"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// AddDataEventForTime sets up the big map for to store important data at each time interval
func (s *Statistic) AddDataEventForTime(e interfaces.DataEventHandler) {
	ex := e.GetExchange()
	a := e.GetAssetType()
	p := e.Pair()

	if s.EventsByTime[ex] == nil {
		s.EventsByTime[ex] = make(map[asset.Item]map[currency.Pair][]EventStore)
	}
	if s.EventsByTime[ex][a] == nil {
		s.EventsByTime[ex][a] = make(map[currency.Pair][]EventStore)
	}
	lookup := s.EventsByTime[ex][a][p]
	lookup = append(lookup, EventStore{DataEvent: e})
	s.EventsByTime[ex][a][p] = lookup
}

// AddSignalEventForTime adds strategy signal event to the statistics at the time period
func (s *Statistic) AddSignalEventForTime(e signal.SignalEvent) {
	lookup := s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()]
	for i := range lookup {
		if lookup[i].DataEvent.GetTime().Equal(e.GetTime()) {
			lookup[i].SignalEvent = e
			s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()] = lookup
		}
	}
}

// AddExchangeEventForTime adds exchange event to the statistics at the time period
func (s *Statistic) AddExchangeEventForTime(e order.OrderEvent) {
	lookup := s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()]
	for i := range lookup {
		if lookup[i].DataEvent.GetTime().Equal(e.GetTime()) {
			lookup[i].ExchangeEvent = e
			s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()] = lookup
		}
	}
}

// AddFillEventForTime adds fill event to the statistics at the time period
func (s *Statistic) AddFillEventForTime(e fill.FillEvent) {
	lookup := s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()]
	for i := range lookup {
		if lookup[i].DataEvent.GetTime().Equal(e.GetTime()) {
			lookup[i].FillEvent = e
			s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()] = lookup
		}
	}
}

// AddHoldingsForTime adds all holdings to the statistics at the time period
func (s *Statistic) AddHoldingsForTime(h holdings.Holding) {
	lookup := s.EventsByTime[h.Exchange][h.Asset][h.Pair]
	for i := range lookup {
		if lookup[i].DataEvent.GetTime().Equal(h.Timestamp) {
			lookup[i].Holdings = h
			s.EventsByTime[h.Exchange][h.Asset][h.Pair] = lookup
		}
	}
}

// AddComplianceSnapshotForTime adds the compliance snapshot to the statistics at the time period
func (s *Statistic) AddComplianceSnapshotForTime(c compliance.Snapshot, e fill.FillEvent) {
	lookup := s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()]
	for i := range lookup {
		if lookup[i].DataEvent.GetTime().Equal(c.Time) {
			lookup[i].Transactions = c
			s.EventsByTime[e.GetExchange()][e.GetAssetType()][e.Pair()] = lookup
		}
	}
}

func (s *Statistic) CalculateTheResults() error {
	var errs gctcommon.Errors
	log.Info(log.BackTester, "------------------Events-------------------------------------")

	for e, x := range s.EventsByTime {
		for a, y := range x {
			for p, z := range y {
				sort.Slice(z, func(i, j int) bool {
					return z[i].DataEvent.GetTime().Before(z[j].DataEvent.GetTime())
				})
				currStr := fmt.Sprintf("------------------Events for %v %v %v------------------------", e, a, p)
				log.Infof(log.BackTester, currStr[:61])

				for i := range z {
					if z[i].FillEvent != nil {
						direction := z[i].FillEvent.GetDirection()
						if direction == common.CouldNotBuy || direction == common.CouldNotSell || direction == common.DoNothing {
							log.Infof(log.BackTester, "%v | Direction: %v - Price: %v - Why: %s",
								z[i].FillEvent.GetTime().Format(gctcommon.SimpleTimeFormat),
								z[i].FillEvent.GetDirection(),
								z[i].FillEvent.GetClosePrice(),
								z[i].FillEvent.GetWhy())
						} else {
							log.Infof(log.BackTester, "%v | Direction %v - Price: $%v - Amount: %v - Fee: $%v - Why: %s",
								z[i].FillEvent.GetTime().Format(gctcommon.SimpleTimeFormat),
								z[i].FillEvent.GetDirection(),
								z[i].FillEvent.GetExchangeFee(),
								z[i].FillEvent.GetAmount(),
								z[i].FillEvent.GetPurchasePrice(),
								z[i].FillEvent.GetWhy())
						}
					} else if z[i].SignalEvent != nil {
						log.Infof(log.BackTester, "%v | Price: $%v - Why: %v",
							z[i].SignalEvent.GetTime().Format(gctcommon.SimpleTimeFormat),
							z[i].SignalEvent.GetPrice(),
							z[i].SignalEvent.GetWhy())
					} else {
						errs = append(errs, fmt.Errorf("%v %v %v unexpected data received %+v", e, a, p, z[i]))
					}
				}
				last := z[len(z)-1]
				first := z[0]
				currStr = fmt.Sprintf("------------------Stats for %v %v %v-------------------------", e, a, p)
				log.Infof(log.BackTester, currStr[:61])
				log.Infof(log.BackTester, "Initial funds: $%v\n\n", last.Holdings.InitialFunds)

				var buyOrders, sellOrders int64
				for i := range last.Transactions.Orders {
					if last.Transactions.Orders[i].Side == gctorder.Buy {
						buyOrders++
					} else if last.Transactions.Orders[i].Side == gctorder.Sell {
						sellOrders++
					}
				}
				log.Infof(log.BackTester, "Buy orders: %v", buyOrders)
				log.Infof(log.BackTester, "Buy value: %v", last.Holdings.BoughtValue)
				log.Infof(log.BackTester, "Buy amount: %v", last.Holdings.BoughtAmount)
				log.Infof(log.BackTester, "Sell orders: %v", sellOrders)
				log.Infof(log.BackTester, "Sell value: %v", last.Holdings.SoldValue)
				log.Infof(log.BackTester, "Sell amount: %v", last.Holdings.SoldAmount)
				log.Infof(log.BackTester, "Total orders: %v\n\n", buyOrders+sellOrders)

				log.Infof(log.BackTester, "Value lost to volume sizing: $%v", last.Holdings.TotalValueLostToVolumeSizing)
				log.Infof(log.BackTester, "Value lost to slippage: $%v", last.Holdings.TotalValueLostToSlippage)
				log.Infof(log.BackTester, "Total Value lost: $%v", last.Holdings.TotalValueLostToSlippage+last.Holdings.TotalValueLostToSlippage)
				log.Infof(log.BackTester, "Total Fees: $%v\n\n", last.Holdings.TotalFees)

				log.Infof(log.BackTester, "Starting Close Price: $%v", first.DataEvent.Price())
				log.Infof(log.BackTester, "Finishing Close Price: $%v", last.DataEvent.Price())
				var lowest, highest float64
				for i := range z {
					price := z[i].DataEvent.Price()
					if lowest == 0 || price < lowest {
						lowest = price
					}
					if highest == 0 || price > highest {
						highest = price
					}
				}
				log.Infof(log.BackTester, "Lowest Close Price: $%v", lowest)
				log.Infof(log.BackTester, "Highest Close Price: $%v", highest)
				marketMove := ((highest - lowest) / lowest) * 100
				strategyMove := ((last.Holdings.TotalValue - last.Holdings.InitialFunds) / last.Holdings.InitialFunds) * 100
				log.Infof(log.BackTester, "Market movement: %v%%", marketMove)
				log.Infof(log.BackTester, "Strategy movement: %v%%", strategyMove)
				log.Infof(log.BackTester, "Did it beat the market: %v\n\n", strategyMove > marketMove)

				var allDataEvents []interfaces.DataEventHandler
				for i := range z {
					allDataEvents = append(allDataEvents, z[i].DataEvent)
				}
				draws := calculateAllDrawDowns(allDataEvents)
				log.Info(log.BackTester, "------------------Max Drawdown-------------------------------")
				log.Infof(log.BackTester, "Highest Price: $%.2f", draws.MaxDrawDown.Highest.Price)
				log.Infof(log.BackTester, "Highest Price Time: %v", draws.MaxDrawDown.Highest.Time)
				log.Infof(log.BackTester, "Lowest Price: $%v", draws.MaxDrawDown.Lowest.Price)
				log.Infof(log.BackTester, "Lowest Price Time: %v", draws.MaxDrawDown.Lowest.Time)
				log.Infof(log.BackTester, "Calculated Drawdown: %.2f%%", ((draws.MaxDrawDown.Lowest.Price-draws.MaxDrawDown.Highest.Price)/draws.MaxDrawDown.Highest.Price)*100)
				log.Infof(log.BackTester, "Difference: $%.2f", draws.MaxDrawDown.Highest.Price-draws.MaxDrawDown.Lowest.Price)
				log.Infof(log.BackTester, "Drawdown length: %v", len(draws.MaxDrawDown.Iterations))

				log.Info(log.BackTester, "------------------Longest Drawdown---------------------------")
				log.Infof(log.BackTester, "Highest Price: $%.2f", draws.LongestDrawDown.Highest.Price)
				log.Infof(log.BackTester, "Highest Price Time: %v", draws.LongestDrawDown.Highest.Time)
				log.Infof(log.BackTester, "Lowest Price: $%.2f", draws.LongestDrawDown.Lowest.Price)
				log.Infof(log.BackTester, "Lowest Price Time: %v", draws.LongestDrawDown.Lowest.Time)
				log.Infof(log.BackTester, "Calculated Drawdown: %.2f%%", ((draws.LongestDrawDown.Lowest.Price-draws.LongestDrawDown.Highest.Price)/draws.LongestDrawDown.Highest.Price)*100)
				log.Infof(log.BackTester, "Difference: $%.2f", draws.LongestDrawDown.Highest.Price-draws.LongestDrawDown.Lowest.Price)
				log.Infof(log.BackTester, "Drawdown length: %v", len(draws.LongestDrawDown.Iterations))

				log.Info(log.BackTester, "------------------Ratios-------------------------------------")
				log.Infof(log.BackTester, "Shape ratio: $%v", last.Holdings.TotalFees)
				log.Infof(log.BackTester, "Sortino ratio: $%v\n\n", last.Holdings.TotalFees)

				log.Infof(log.BackTester, "Final funds: $%v", last.Holdings.RemainingFunds)
				log.Infof(log.BackTester, "Final holdings: %v", last.Holdings.PositionsSize)
				log.Infof(log.BackTester, "Final holdings value: $%v", last.Holdings.PositionsValue)
				log.Infof(log.BackTester, "Final total value: $%v", last.Holdings.TotalValue)
			}
		}
	}

	if len(errs) > 0 {
		log.Info(log.BackTester, "------------------Errors-------------------------------------")
		for i := range errs {
			log.Info(log.BackTester, errs[i].Error())
		}
	}

	return nil
}

type SuperDrawDown struct {
	DrawDowns       []DrawDown
	MaxDrawDown     DrawDown
	LongestDrawDown DrawDown
}

type DrawDown struct {
	Highest    Draw
	Lowest     Draw
	Iterations []Iterations
}

type Draw struct {
	Price float64
	Time  time.Time
}

type Iterations struct {
	Time  time.Time
	Price float64
}

func (s *SuperDrawDown) calculateMaxAndLongestDrawDowns() {
	for i := range s.DrawDowns {
		if s.DrawDowns[i].Highest.Price-s.DrawDowns[i].Lowest.Price > s.MaxDrawDown.Highest.Price-s.MaxDrawDown.Lowest.Price {
			s.MaxDrawDown = s.DrawDowns[i]
		}
		if len(s.DrawDowns[i].Iterations) > len(s.LongestDrawDown.Iterations) {
			s.LongestDrawDown = s.DrawDowns[i]
		}
	}
}

func calculateAllDrawDowns(closePrices []interfaces.DataEventHandler) SuperDrawDown {
	isDrawingDown := false

	var response SuperDrawDown
	var activeDraw DrawDown
	for i := range closePrices {
		p := closePrices[i].Price()
		t := closePrices[i].GetTime()
		if i == 0 {
			activeDraw.Highest = Draw{
				Price: p,
				Time:  t,
			}
			activeDraw.Lowest = Draw{
				Price: p,
				Time:  t,
			}
		}

		// create
		if !isDrawingDown && activeDraw.Highest.Price > p {
			isDrawingDown = true
			activeDraw = DrawDown{
				Highest: Draw{
					Price: p,
					Time:  t,
				},
				Lowest: Draw{
					Price: p,
					Time:  t,
				},
			}
		}

		// close
		if isDrawingDown && activeDraw.Lowest.Price < p {
			activeDraw.Lowest = Draw{
				Price: activeDraw.Iterations[len(activeDraw.Iterations)-1].Price,
				Time:  activeDraw.Iterations[len(activeDraw.Iterations)-1].Time,
			}
			isDrawingDown = false
			response.DrawDowns = append(response.DrawDowns, activeDraw)
			// reset
			activeDraw = DrawDown{
				Highest: Draw{
					Price: p,
					Time:  t,
				},
				Lowest: Draw{
					Price: p,
					Time:  t,
				},
			}
		}

		// append
		if isDrawingDown {
			if p < activeDraw.Lowest.Price {
				activeDraw.Lowest.Price = p
				activeDraw.Lowest.Time = t
			}
			activeDraw.Iterations = append(activeDraw.Iterations, Iterations{
				Time:  t,
				Price: p,
			})
		}
	}

	response.calculateMaxAndLongestDrawDowns()

	return response
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
	result, _ := s.TotalEquityReturn()
	fmt.Printf("Initial funds: $%f\nValue at enddate %v:\t$%f\n",
		roundIt(s.InitialFunds),
		s.Equity[len(s.Equity)-1].Timestamp.Format(gctcommon.SimpleTimeFormat),
		roundIt(s.Equity[len(s.Equity)-1].BuyAndHoldValue))
	fmt.Printf("Difference: $%f\n", roundIt(s.Equity[len(s.Equity)-1].BuyAndHoldValue-s.InitialFunds))
	fmt.Printf("TotalEquity: %f\nMaxDrawdown: %f", roundIt(result), roundIt(s.MaxDrawdown()))
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
