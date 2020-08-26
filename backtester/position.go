package backtest

import (
	"github.com/shopspring/decimal"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (p *Positions) Create(fill FillEvent) {
	p.timestamp = fill.GetTime()
	p.pair = fill.Pair()

	p.update(fill)
}

func (p *Positions) Update(fill FillEvent) {
	p.timestamp = fill.GetTime()

	p.update(fill)
}

func (p *Positions) UpdateValue(data DataEventHandler) {
	p.timestamp = data.GetTime()

	latest := data.LatestPrice()
	p.updateValue(latest)
}

func (p *Positions) update(fill FillEvent) {
	fillAmount := decimal.NewFromFloat(fill.GetAmount())
	fillPrice := decimal.NewFromFloat(fill.GetPrice())
	fillCommission := decimal.NewFromFloat(fill.GetCommission())
	fillExchangeFee := decimal.NewFromFloat(fill.GetExchangeFee())
	fillCost := decimal.NewFromFloat(fill.GetCost())
	fillNetValue := decimal.NewFromFloat(fill.NetValue())

	amount := decimal.NewFromFloat(p.amount)
	amountBought := decimal.NewFromFloat(p.amountBought)
	amountSold := decimal.NewFromFloat(p.amountSold)
	avgPrice := decimal.NewFromFloat(p.avgPrice)
	avgPriceNet := decimal.NewFromFloat(p.avgPriceNet)
	avgPriceBought := decimal.NewFromFloat(p.avgPriceBought)
	avgPriceSold := decimal.NewFromFloat(p.avgPriceSold)
	value := decimal.NewFromFloat(p.value)
	valueBought := decimal.NewFromFloat(p.valueBought)
	valueSold := decimal.NewFromFloat(p.valueSold)
	netValue := decimal.NewFromFloat(p.netValue)
	netValueBought := decimal.NewFromFloat(p.netValueBought)
	netValueSold := decimal.NewFromFloat(p.netValueSold)
	commission := decimal.NewFromFloat(p.commission)
	exchangeFee := decimal.NewFromFloat(p.exchangeFee)
	cost := decimal.NewFromFloat(p.cost)
	costBasis := decimal.NewFromFloat(p.costBasis)
	realProfitLoss := decimal.NewFromFloat(p.realProfitLoss)

	switch fill.GetDirection() {
	case gctorder.Buy, gctorder.Bid:
		if p.amount >= 0 {
			costBasis = costBasis.Add(fillNetValue)
		} else {
			costBasis = costBasis.Add(fillAmount.Abs().Div(amount).Mul(costBasis))
			realProfitLoss = realProfitLoss.Add(fillAmount.Mul(avgPriceNet.Sub(fillPrice))).Sub(fillCost)
		}
		avgPrice = amount.Abs().Mul(avgPrice).Add(fillAmount.Mul(fillPrice)).Div(amount.Abs().Add(fillAmount))
		avgPriceNet = amount.Abs().Mul(avgPriceNet).Add(fillNetValue).Div(amount.Abs().Add(fillAmount))
		avgPriceBought = amountBought.Mul(avgPriceBought).Add(fillAmount.Mul(fillPrice)).Div(amountBought.Add(fillAmount))

		amount = amount.Add(fillAmount)
		amountBought = amountBought.Add(fillAmount)

		valueBought = amountBought.Mul(avgPriceBought)
		netValueBought = netValueBought.Add(fillNetValue)

	case gctorder.Sell, gctorder.Ask:
		if p.amount > 0 {
			costBasis = costBasis.Sub(fillAmount.Abs().Div(amount).Mul(costBasis))
			realProfitLoss = realProfitLoss.Add(fillAmount.Abs().Mul(fillPrice.Sub(avgPriceNet))).Sub(fillCost)
		} else {
			costBasis = costBasis.Sub(fillNetValue)
		}

		avgPrice = amount.Abs().Mul(avgPrice).Add(fillAmount.Mul(fillPrice)).Div(amount.Abs().Add(fillAmount))
		avgPriceNet = amount.Abs().Mul(avgPriceNet).Add(fillNetValue).Div(amount.Abs().Add(fillAmount))
		avgPriceSold = amountSold.Mul(avgPriceSold).Add(fillAmount.Mul(fillPrice)).Div(amountSold.Add(fillAmount))

		amount = amount.Sub(fillAmount)
		amountSold = amountSold.Add(fillAmount)
		valueSold = amountSold.Mul(avgPriceSold)
		netValueSold = netValueSold.Add(fillNetValue)
	}

	commission = commission.Add(fillCommission)
	exchangeFee = exchangeFee.Add(fillExchangeFee)
	cost = cost.Add(fillCost)

	value = valueSold.Sub(valueBought)
	netValue = value.Sub(cost)

	p.amount, _ = amount.Round(DP).Float64()
	p.amountBought, _ = amountBought.Round(DP).Float64()
	p.amountSold, _ = amountSold.Round(DP).Float64()
	p.avgPrice, _ = avgPrice.Round(DP).Float64()
	p.avgPriceBought, _ = avgPriceBought.Round(DP).Float64()
	p.avgPriceSold, _ = avgPriceSold.Round(DP).Float64()
	p.avgPriceNet, _ = avgPriceNet.Round(DP).Float64()
	p.value, _ = value.Round(DP).Float64()
	p.valueBought, _ = valueBought.Round(DP).Float64()
	p.valueSold, _ = valueSold.Round(DP).Float64()
	p.netValue, _ = netValue.Round(DP).Float64()
	p.netValueBought, _ = netValueBought.Round(DP).Float64()
	p.netValueSold, _ = netValueSold.Round(DP).Float64()
	p.commission, _ = commission.Round(DP).Float64()
	p.exchangeFee, _ = exchangeFee.Round(DP).Float64()
	p.cost, _ = cost.Round(DP).Float64()
	p.costBasis, _ = costBasis.Round(DP).Float64()
	p.realProfitLoss, _ = realProfitLoss.Round(DP).Float64()

	p.updateValue(fill.GetPrice())
}

func (p *Positions) updateValue(l float64) {
	latest := decimal.NewFromFloat(l)
	amount := decimal.NewFromFloat(p.amount)
	costBasis := decimal.NewFromFloat(p.costBasis)

	marketPrice := latest
	p.marketPrice, _ = marketPrice.Round(DP).Float64()
	marketValue := amount.Abs().Mul(latest)
	p.marketValue, _ = marketValue.Round(DP).Float64()

	unrealProfitLoss := amount.Mul(latest).Sub(costBasis)
	p.unrealProfitLoss, _ = unrealProfitLoss.Round(DP).Float64()

	realProfitLoss := decimal.NewFromFloat(p.realProfitLoss)
	totalProfitLoss := realProfitLoss.Add(unrealProfitLoss)
	p.totalProfitLoss, _ = totalProfitLoss.Round(DP).Float64()
}
