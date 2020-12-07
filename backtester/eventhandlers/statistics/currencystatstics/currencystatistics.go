package currencystatstics

import (
	"fmt"
	"sort"

	"gonum.org/v1/gonum/stat"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// SharpeRatio returns sharpe ratio of backtest compared to risk-free
func (c *CurrencyStatistic) CalculateSharpeRatio(riskFreeReturns float64) {
	var equityReturns = make([]float64, len(c.Events))

	for i := range c.Events {
		equityReturns[i] = c.Events[i].Holdings.EquityReturn
	}
	mean, stddev := stat.MeanStdDev(equityReturns, nil)

	c.SharpeRatio = (mean - riskFreeReturns) / stddev
}

func (c *CurrencyStatistic) CalculateResults() {
	last := c.Events[len(c.Events)-1]
	for i := range last.Transactions.Orders {
		if last.Transactions.Orders[i].Side == gctorder.Buy {
			c.BuyOrders++
		} else if last.Transactions.Orders[i].Side == gctorder.Sell {
			c.SellOrders++
		}
	}
	for i := range c.Events {
		price := c.Events[i].DataEvent.Price()
		if c.LowestClosePrice == 0 || price < c.LowestClosePrice {
			c.LowestClosePrice = price
		}
		if c.HighestClosePrice == 0 || price > c.HighestClosePrice {
			c.HighestClosePrice = price
		}
	}
	c.MarketMovement = ((c.HighestClosePrice - c.LowestClosePrice) / c.LowestClosePrice) * 100
	c.StrategyMovement = ((last.Holdings.TotalValue - last.Holdings.InitialFunds) / last.Holdings.InitialFunds) * 100
	var allDataEvents []interfaces.DataEventHandler
	for i := range c.Events {
		allDataEvents = append(allDataEvents, c.Events[i].DataEvent)
	}
	c.DrawDowns = calculateAllDrawDowns(allDataEvents)
}

func (c *CurrencyStatistic) PrintResults(e string, a asset.Item, p currency.Pair) {
	var errs gctcommon.Errors
	sort.Slice(c.Events, func(i, j int) bool {
		return c.Events[i].DataEvent.GetTime().Before(c.Events[j].DataEvent.GetTime())
	})
	currStr := fmt.Sprintf("------------------Events for %v %v %v------------------------", e, a, p)
	log.Infof(log.BackTester, currStr[:61])

	for i := range c.Events {
		if c.Events[i].FillEvent != nil {
			direction := c.Events[i].FillEvent.GetDirection()
			if direction == common.CouldNotBuy || direction == common.CouldNotSell || direction == common.DoNothing {
				log.Infof(log.BackTester, "%v | Direction: %v - Price: %v - Why: %s - Equity Return %.2f",
					c.Events[i].FillEvent.GetTime().Format(gctcommon.SimpleTimeFormat),
					c.Events[i].FillEvent.GetDirection(),
					c.Events[i].FillEvent.GetClosePrice(),
					c.Events[i].FillEvent.GetWhy(),
					c.Events[i].Holdings.EquityReturn)
			} else {
				log.Infof(log.BackTester, "%v | Direction %v - Price: $%v - Amount: %v - Fee: $%v - Why: %s - Equity Return %.2f ",
					c.Events[i].FillEvent.GetTime().Format(gctcommon.SimpleTimeFormat),
					c.Events[i].FillEvent.GetDirection(),
					c.Events[i].FillEvent.GetExchangeFee(),
					c.Events[i].FillEvent.GetAmount(),
					c.Events[i].FillEvent.GetPurchasePrice(),
					c.Events[i].FillEvent.GetWhy(),
					c.Events[i].Holdings.EquityReturn,
				)
			}
		} else if c.Events[i].SignalEvent != nil {
			log.Infof(log.BackTester, "%v | Price: $%v - Why: %v",
				c.Events[i].SignalEvent.GetTime().Format(gctcommon.SimpleTimeFormat),
				c.Events[i].SignalEvent.GetPrice(),
				c.Events[i].SignalEvent.GetWhy())
		} else {
			errs = append(errs, fmt.Errorf("%v %v %v unexpected data received %+v", e, a, p, c.Events[i]))
		}
	}
	last := c.Events[len(c.Events)-1]
	first := c.Events[0]
	currStr = fmt.Sprintf("------------------Stats for %v %v %v-------------------------", e, a, p)
	log.Infof(log.BackTester, currStr[:61])
	log.Infof(log.BackTester, "Initial funds: $%v\n\n", last.Holdings.InitialFunds)

	log.Infof(log.BackTester, "Buy orders: %v", c.BuyOrders)
	log.Infof(log.BackTester, "Buy value: %v", last.Holdings.BoughtValue)
	log.Infof(log.BackTester, "Buy amount: %v", last.Holdings.BoughtAmount)
	log.Infof(log.BackTester, "Sell orders: %v", c.SellOrders)
	log.Infof(log.BackTester, "Sell value: %v", last.Holdings.SoldValue)
	log.Infof(log.BackTester, "Sell amount: %v", last.Holdings.SoldAmount)
	log.Infof(log.BackTester, "Total orders: %v\n\n", c.BuyOrders+c.SellOrders)

	log.Infof(log.BackTester, "Value lost to volume sizing: $%v", last.Holdings.TotalValueLostToVolumeSizing)
	log.Infof(log.BackTester, "Value lost to slippage: $%v", last.Holdings.TotalValueLostToSlippage)
	log.Infof(log.BackTester, "Total Value lost: $%v", last.Holdings.TotalValueLostToSlippage+last.Holdings.TotalValueLostToSlippage)
	log.Infof(log.BackTester, "Total Fees: $%v\n\n", last.Holdings.TotalFees)

	log.Infof(log.BackTester, "Starting Close Price: $%v", first.DataEvent.Price())
	log.Infof(log.BackTester, "Finishing Close Price: $%v", last.DataEvent.Price())

	log.Infof(log.BackTester, "Lowest Close Price: $%v", c.LowestClosePrice)
	log.Infof(log.BackTester, "Highest Close Price: $%v", c.HighestClosePrice)

	log.Infof(log.BackTester, "Market movement: %v%%", c.MarketMovement)
	log.Infof(log.BackTester, "Strategy movement: %v%%", c.StrategyMovement)
	log.Infof(log.BackTester, "Did it beat the market: %v\n\n", c.StrategyMovement > c.MarketMovement)

	log.Info(log.BackTester, "------------------Max Drawdown-------------------------------")
	log.Infof(log.BackTester, "Highest Price: $%.2f", c.DrawDowns.MaxDrawDown.Highest.Price)
	log.Infof(log.BackTester, "Highest Price Time: %v", c.DrawDowns.MaxDrawDown.Highest.Time)
	log.Infof(log.BackTester, "Lowest Price: $%v", c.DrawDowns.MaxDrawDown.Lowest.Price)
	log.Infof(log.BackTester, "Lowest Price Time: %v", c.DrawDowns.MaxDrawDown.Lowest.Time)
	log.Infof(log.BackTester, "Calculated Drawdown: %.2f%%", c.DrawDowns.MaxDrawDown.CalculatedDrawDown)
	log.Infof(log.BackTester, "Difference: $%.2f", c.DrawDowns.MaxDrawDown.Highest.Price-c.DrawDowns.MaxDrawDown.Lowest.Price)
	log.Infof(log.BackTester, "Drawdown length: %v", len(c.DrawDowns.MaxDrawDown.Iterations))

	log.Info(log.BackTester, "------------------Longest Drawdown---------------------------")
	log.Infof(log.BackTester, "Highest Price: $%.2f", c.DrawDowns.LongestDrawDown.Highest.Price)
	log.Infof(log.BackTester, "Highest Price Time: %v", c.DrawDowns.LongestDrawDown.Highest.Time)
	log.Infof(log.BackTester, "Lowest Price: $%.2f", c.DrawDowns.LongestDrawDown.Lowest.Price)
	log.Infof(log.BackTester, "Lowest Price Time: %v", c.DrawDowns.LongestDrawDown.Lowest.Time)
	log.Infof(log.BackTester, "Calculated Drawdown: %.2f%%", c.DrawDowns.LongestDrawDown.CalculatedDrawDown)
	log.Infof(log.BackTester, "Difference: $%.2f", c.DrawDowns.LongestDrawDown.Highest.Price-c.DrawDowns.LongestDrawDown.Lowest.Price)
	log.Infof(log.BackTester, "Drawdown length: %v", len(c.DrawDowns.LongestDrawDown.Iterations))

	log.Info(log.BackTester, "------------------Ratios-------------------------------------")
	log.Infof(log.BackTester, "Sharpe ratio: $%v", last.Holdings.TotalFees)
	log.Infof(log.BackTester, "Sortino ratio: $%v\n\n", last.Holdings.TotalFees)

	log.Infof(log.BackTester, "Final funds: $%v", last.Holdings.RemainingFunds)
	log.Infof(log.BackTester, "Final holdings: %v", last.Holdings.PositionsSize)
	log.Infof(log.BackTester, "Final holdings value: $%v", last.Holdings.PositionsValue)
	log.Infof(log.BackTester, "Final total value: $%v", last.Holdings.TotalValue)
	if len(errs) > 0 {
		log.Info(log.BackTester, "------------------Errors-------------------------------------")
		for i := range errs {
			log.Info(log.BackTester, errs[i].Error())
		}
	}
}

func (c *CurrencyStatistic) MaxDrawdown() DrawDown {
	if len(c.DrawDowns.MaxDrawDown.Iterations) == 0 {
		var allDataEvents []interfaces.DataEventHandler
		for i := range c.Events {
			allDataEvents = append(allDataEvents, c.Events[i].DataEvent)
		}
		c.DrawDowns = calculateAllDrawDowns(allDataEvents)
	}
	return c.DrawDowns.MaxDrawDown
}

func (c *CurrencyStatistic) LongestDrawdown() DrawDown {
	if len(c.DrawDowns.LongestDrawDown.Iterations) == 0 {
		var allDataEvents []interfaces.DataEventHandler
		for i := range c.Events {
			allDataEvents = append(allDataEvents, c.Events[i].DataEvent)
		}
		c.DrawDowns = calculateAllDrawDowns(allDataEvents)
	}
	return c.DrawDowns.LongestDrawDown
}

func calculateAllDrawDowns(closePrices []interfaces.DataEventHandler) DrawDownHolder {
	isDrawingDown := false

	var response DrawDownHolder
	var activeDraw DrawDown
	for i := range closePrices {
		p := closePrices[i].Price()
		t := closePrices[i].GetTime()
		if i == 0 {
			activeDraw.Highest = Iteration{
				Price: p,
				Time:  t,
			}
			activeDraw.Lowest = Iteration{
				Price: p,
				Time:  t,
			}
			continue
		}

		// create
		if !isDrawingDown && activeDraw.Highest.Price > p {
			isDrawingDown = true
			activeDraw = DrawDown{
				Highest: Iteration{
					Price: p,
					Time:  t,
				},
				Lowest: Iteration{
					Price: p,
					Time:  t,
				},
			}
		}

		// close
		if isDrawingDown && activeDraw.Lowest.Price < p {
			activeDraw.Lowest = Iteration{
				Price: activeDraw.Iterations[len(activeDraw.Iterations)-1].Price,
				Time:  activeDraw.Iterations[len(activeDraw.Iterations)-1].Time,
			}
			isDrawingDown = false
			response.DrawDowns = append(response.DrawDowns, activeDraw)
			// reset
			activeDraw = DrawDown{
				Highest: Iteration{
					Price: p,
					Time:  t,
				},
				Lowest: Iteration{
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
			activeDraw.Iterations = append(activeDraw.Iterations, Iteration{
				Time:  t,
				Price: p,
			})
		}
	}

	response.calculateMaxAndLongestDrawDowns()
	response.MaxDrawDown.CalculatedDrawDown = ((response.MaxDrawDown.Lowest.Price - response.MaxDrawDown.Highest.Price) / response.MaxDrawDown.Highest.Price) * 100
	response.LongestDrawDown.CalculatedDrawDown = ((response.LongestDrawDown.Lowest.Price - response.LongestDrawDown.Highest.Price) / response.LongestDrawDown.Highest.Price) * 100

	return response
}

func (s *DrawDownHolder) calculateMaxAndLongestDrawDowns() {
	for i := range s.DrawDowns {
		if s.DrawDowns[i].Highest.Price-s.DrawDowns[i].Lowest.Price > s.MaxDrawDown.Highest.Price-s.MaxDrawDown.Lowest.Price {
			s.MaxDrawDown = s.DrawDowns[i]
		}
		if len(s.DrawDowns[i].Iterations) > len(s.LongestDrawDown.Iterations) {
			s.LongestDrawDown = s.DrawDowns[i]
		}
	}
}
