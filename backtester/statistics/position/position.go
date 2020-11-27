package position

import (
	"github.com/shopspring/decimal"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (p *Position) Create(fill fill.FillEvent) {
	p.Timestamp = fill.GetTime()

	p.update(fill)
}

func (p *Position) Update(fill fill.FillEvent) {
	p.Timestamp = fill.GetTime()

	p.update(fill)
}

func (p *Position) UpdateValue(data interfaces.DataEventHandler) {
	p.Timestamp = data.GetTime()

	latest := data.Price()
	p.updateValue(latest)
}

func (p *Position) update(fill fill.FillEvent) {
	fillAmount := decimal.NewFromFloat(fill.GetAmount())
	fillPrice := decimal.NewFromFloat(fill.GetPurchasePrice())
	fillExchangeFee := decimal.NewFromFloat(fill.GetExchangeFee())
	fillNetValue := decimal.NewFromFloat(fill.NetValue())

	amount := decimal.NewFromFloat(p.Amount)
	amountBought := decimal.NewFromFloat(p.AmountBought)
	amountSold := decimal.NewFromFloat(p.AmountSold)
	avgPrice := decimal.NewFromFloat(p.AveragePrice)
	avgPriceNet := decimal.NewFromFloat(p.AveragePriceNet)
	avgPriceBought := decimal.NewFromFloat(p.AveragePriceBought)
	avgPriceSold := decimal.NewFromFloat(p.AveragePriceSold)
	value := decimal.NewFromFloat(p.Value)
	valueBought := decimal.NewFromFloat(p.ValueBought)
	valueSold := decimal.NewFromFloat(p.ValueSold)
	netValue := decimal.NewFromFloat(p.NetValue)
	netValueBought := decimal.NewFromFloat(p.NetValueBought)
	netValueSold := decimal.NewFromFloat(p.NetValueSold)
	exchangeFee := decimal.NewFromFloat(p.ExchangeFee)
	cost := decimal.NewFromFloat(p.Cost)
	costBasis := decimal.NewFromFloat(p.CostBasis)
	realProfitLoss := decimal.NewFromFloat(p.RealProfitLoss)

	switch fill.GetDirection() {
	case gctorder.Buy, gctorder.Bid:
		if p.Amount >= 0 {
			costBasis = costBasis.Add(fillNetValue)
		} else {
			costBasis = costBasis.Add(fillAmount.Abs().Div(amount).Mul(costBasis))
			realProfitLoss = realProfitLoss.Add(fillAmount.Mul(avgPriceNet.Sub(fillPrice))).Sub(exchangeFee)
		}
		avgPrice = amount.Abs().Mul(avgPrice).Add(fillAmount.Mul(fillPrice)).Div(amount.Abs().Add(fillAmount))
		avgPriceNet = amount.Abs().Mul(avgPriceNet).Add(fillNetValue).Div(amount.Abs().Add(fillAmount))
		avgPriceBought = amountBought.Mul(avgPriceBought).Add(fillAmount.Mul(fillPrice)).Div(amountBought.Add(fillAmount))

		amount = amount.Add(fillAmount)
		amountBought = amountBought.Add(fillAmount)

		valueBought = amountBought.Mul(avgPriceBought)
		netValueBought = netValueBought.Add(fillNetValue)

	case gctorder.Sell, gctorder.Ask:
		if p.Amount > 0 {
			costBasis = costBasis.Sub(fillAmount.Abs().Div(amount).Mul(costBasis))
			realProfitLoss = realProfitLoss.Add(fillAmount.Abs().Mul(fillPrice.Sub(avgPriceNet))).Sub(exchangeFee)
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

	exchangeFee = exchangeFee.Add(fillExchangeFee)
	cost = cost.Add(exchangeFee)

	value = valueSold.Sub(valueBought)
	netValue = value.Sub(cost)

	p.Amount, _ = amount.Round(common.DecimalPlaces).Float64()
	p.AmountBought, _ = amountBought.Round(common.DecimalPlaces).Float64()
	p.AmountSold, _ = amountSold.Round(common.DecimalPlaces).Float64()
	p.AveragePrice, _ = avgPrice.Round(common.DecimalPlaces).Float64()
	p.AveragePriceBought, _ = avgPriceBought.Round(common.DecimalPlaces).Float64()
	p.AveragePriceSold, _ = avgPriceSold.Round(common.DecimalPlaces).Float64()
	p.AveragePriceNet, _ = avgPriceNet.Round(common.DecimalPlaces).Float64()
	p.Value, _ = value.Round(common.DecimalPlaces).Float64()
	p.ValueBought, _ = valueBought.Round(common.DecimalPlaces).Float64()
	p.ValueSold, _ = valueSold.Round(common.DecimalPlaces).Float64()
	p.NetValue, _ = netValue.Round(common.DecimalPlaces).Float64()
	p.NetValueBought, _ = netValueBought.Round(common.DecimalPlaces).Float64()
	p.NetValueSold, _ = netValueSold.Round(common.DecimalPlaces).Float64()
	p.ExchangeFee, _ = exchangeFee.Round(common.DecimalPlaces).Float64()
	p.Cost, _ = cost.Round(common.DecimalPlaces).Float64()
	p.CostBasis, _ = costBasis.Round(common.DecimalPlaces).Float64()
	p.RealProfitLoss, _ = realProfitLoss.Round(common.DecimalPlaces).Float64()

	p.updateValue(fill.GetClosePrice())
}

func (p *Position) updateValue(l float64) {
	latest := decimal.NewFromFloat(l)
	amount := decimal.NewFromFloat(p.Amount)
	costBasis := decimal.NewFromFloat(p.CostBasis)

	marketPrice := latest
	p.MarketPrice, _ = marketPrice.Round(common.DecimalPlaces).Float64()
	marketValue := amount.Abs().Mul(latest)
	p.MarketValue, _ = marketValue.Round(common.DecimalPlaces).Float64()

	unrealProfitLoss := amount.Mul(latest).Sub(costBasis)
	p.UnrealProfitLoss, _ = unrealProfitLoss.Round(common.DecimalPlaces).Float64()

	realProfitLoss := decimal.NewFromFloat(p.RealProfitLoss)
	totalProfitLoss := realProfitLoss.Add(unrealProfitLoss)
	p.TotalProfitLoss, _ = totalProfitLoss.Round(common.DecimalPlaces).Float64()
}
