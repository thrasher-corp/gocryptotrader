package backtest

import (
	"errors"
	"math"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (r *Risk) EvaluateOrder(order OrderEvent) (*Order, error) {
	return nil, nil
}

func (s *Size) SizeOrder(orderevent OrderEvent, data DataEvent, pf PortfolioHandler) (*Order, error) {
	if (s.DefaultSize == 0) || (s.DefaultValue == 0) {
		return nil, errors.New("no DefaultSize or DefaultValue set")
	}

	o := orderevent.(*Order)
	switch o.Direction() {
	case order.Buy:
		o.SetAmount(s.setDefaultSize(data.Price()))
	case order.Sell:
		o.SetAmount(s.setDefaultSize(data.Price()))
	default:
		if _, ok := pf.IsInvested(); !ok {
			return o, errors.New("no position in Portfolio")
		}
		if pos, ok := pf.IsLong(); ok {
			o.SetAmount(pos.Amount)
		}
		if pos, ok := pf.IsShort(); ok {
			o.SetAmount(pos.Amount * -1)
		}
	}

	return o, nil
}

func (s *Size) setDefaultSize(price float64) float64 {
	if (s.DefaultSize * price) > s.DefaultValue {
		return math.Floor(s.DefaultValue / price)
	}
	return s.DefaultSize
}

func (p *Position) update(inOrder *Order) {
	fillQty := inOrder.GetAmountFilled()
	fillPrice := inOrder.GetAvgFillPrice()
	fillExchangeFee := inOrder.ExchangeFee()
	fillCost := inOrder.Cost()
	fillNetValue := inOrder.NetValue()


	amount := p.Amount
	amountBought := p.AmountBought
	amountSOld := p.AmountSold
	avgPrice := p.avgPrice
	avgPriceNet := p.avgPriceNet
	avgPriceBot := p.avgPriceBought
	avgPriceSld := p.avgPriceSold
	value := p.value
	valueBot := p.valueBought
	valueSld := p.valueSold
	netValue := p.netValue
	netValueBot := p.netValueBought
	netValueSld := p.netValueSold

	exchangeFee := p.exchangeFee
	cost := p.cost
	costBasis := p.costBasis
	realProfitLoss := p.realProfitLoss

	switch inOrder.Direction() {
	case order.Buy:
		if p.Amount >= 0 {
			costBasis += fillNetValue
		} else {
			costBasis += math.Abs(fillQty) / amount * costBasis
			realProfitLoss += fillQty*(avgPriceNet-fillPrice) - fillCost
		}
		avgPrice = ((math.Abs(amount) * avgPrice) + (fillQty * fillPrice)) / (math.Abs(amount) + fillQty)
		avgPriceNet = (math.Abs(amount)*avgPriceNet + fillNetValue) / (math.Abs(amount) + fillQty)
		avgPriceBot = ((amountBought * avgPriceBot) + (fillQty * fillPrice)) / (amountBought + fillQty)
		amount += fillQty
		amountBought += fillQty

		valueBot = amountBought * avgPriceBot
		netValueBot += fillNetValue

	case order.Sell:
		if p.Amount > 0 {
			costBasis -= math.Abs(fillQty) / amount * costBasis
			realProfitLoss += math.Abs(fillQty)*(fillPrice-avgPriceNet) - fillCost
		} else {
			costBasis -= fillNetValue
		}
		avgPrice = (math.Abs(amount)*avgPrice + fillQty*fillPrice) / (math.Abs(amount) + fillQty)
		avgPriceNet = (math.Abs(amount)*avgPriceNet + fillNetValue) / (math.Abs(amount) + fillQty)
		avgPriceSld = (amountSOld*avgPriceSld + fillQty*fillPrice) / (amountSOld + fillQty)

		amount -= fillQty
		amountSOld += fillQty

		valueSld = amountSOld * avgPriceSld
		netValueSld += fillNetValue
	}
	
	exchangeFee += fillExchangeFee
	cost += fillCost
	value = valueSld - valueBot
	netValue = value - cost

	p.Amount = amount
	p.AmountBought = amountBought
	p.AmountSold = amountSOld
	p.avgPrice = math.Round(avgPrice*math.Pow10(DP)) / math.Pow10(DP)
	p.avgPriceBought = math.Round(avgPriceBot*math.Pow10(DP)) / math.Pow10(DP)
	p.avgPriceSold = math.Round(avgPriceSld*math.Pow10(DP)) / math.Pow10(DP)
	p.avgPriceNet = math.Round(avgPriceNet*math.Pow10(DP)) / math.Pow10(DP)
	p.value = math.Round(value*math.Pow10(DP)) / math.Pow10(DP)
	p.valueBought = math.Round(valueBot*math.Pow10(DP)) / math.Pow10(DP)
	p.valueSold = math.Round(valueSld*math.Pow10(DP)) / math.Pow10(DP)
	p.netValue = math.Round(netValue*math.Pow10(DP)) / math.Pow10(DP)
	p.netValueBought = math.Round(netValueBot*math.Pow10(DP)) / math.Pow10(DP)
	p.netValueSold = math.Round(netValueSld*math.Pow10(DP)) / math.Pow10(DP)
	p.exchangeFee = exchangeFee
	p.cost = cost
	p.costBasis = math.Round(costBasis*math.Pow10(DP)) / math.Pow10(DP)
	p.realProfitLoss = math.Round(realProfitLoss*math.Pow10(DP)) / math.Pow10(DP)

	p.updateValue(inOrder.Price())
}


func (p *Position) updateValue(l float64) {
	latest := l
	qty := p.Amount
	costBasis := p.costBasis
	p.marketValue = math.Abs(qty) * latest
	unrealProfitLoss := qty*latest - costBasis
	p.unrealProfitLoss = math.Round(unrealProfitLoss*math.Pow10(DP)) / math.Pow10(DP)
	realProfitLoss := p.realProfitLoss
	totalProfitLoss := realProfitLoss + unrealProfitLoss
	p.totalProfitLoss = math.Round(totalProfitLoss*math.Pow10(DP)) / math.Pow10(DP)
}
