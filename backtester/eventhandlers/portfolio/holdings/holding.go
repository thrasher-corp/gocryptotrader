package holdings

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

func Create(fill fill.FillEvent, initialFunds float64) Holding {
	h := Holding{
		Timestamp:    fill.GetTime(),
		InitialFunds: initialFunds,
	}

	h.update(fill)
	return h
}

func (h *Holding) Update(fill fill.FillEvent) {
	h.Timestamp = fill.GetTime()
	h.update(fill)
}

func (h *Holding) UpdateValue(data interfaces.DataEventHandler) {
	h.Timestamp = data.GetTime()
	latest := data.Price()
	h.updateValue(latest)
}

func (h *Holding) update(fill fill.FillEvent) {
	direction := fill.GetDirection()
	o := fill.GetOrder()
	switch direction {
	case order.Buy:
		h.PositionsSize += o.Amount
		h.PositionsValue += o.Amount * o.Price
		h.RemainingFunds -= (o.Amount * o.Price) + o.Fee
		h.TotalFees += o.Fee
		h.BoughtAmount += o.Amount
		h.BoughtValue += o.Amount * o.Price
	case order.Sell:
		h.PositionsSize -= o.Amount
		h.PositionsValue -= o.Amount * o.Price
		h.RemainingFunds += (o.Amount * o.Price) - o.Fee
		h.TotalFees += o.Fee
		h.SoldAmount += o.Amount
		h.SoldValue += o.Amount * o.Price
	case common.DoNothing:
	default:
		log.Error(log.BackTester, "woah nelly, how'd we get here? %v", direction)
	}
	/*
		fillAmount := decimal.NewFromFloat(fill.GetAmount())
		fillPrice := decimal.NewFromFloat(fill.GetPurchasePrice())
		fillExchangeFee := decimal.NewFromFloat(fill.GetExchangeFee())
		fillNetValue := decimal.NewFromFloat(fill.NetValue())

		amount := decimal.NewFromFloat(h.Amount)
		amountBought := decimal.NewFromFloat(h.BoughtAmount)
		amountSold := decimal.NewFromFloat(h.SoldAmount)
		avgPrice := decimal.NewFromFloat(h.AveragePrice)
		avgPriceNet := decimal.NewFromFloat(h.AveragePriceNet)
		avgPriceBought := decimal.NewFromFloat(h.AveragePriceBought)
		avgPriceSold := decimal.NewFromFloat(h.AveragePriceSold)
		value := decimal.NewFromFloat(h.Value)
		valueBought := decimal.NewFromFloat(h.ValueBought)
		valueSold := decimal.NewFromFloat(h.ValueSold)
		netValue := decimal.NewFromFloat(h.NetValue)
		netValueBought := decimal.NewFromFloat(h.NetValueBought)
		netValueSold := decimal.NewFromFloat(h.NetValueSold)
		exchangeFee := decimal.NewFromFloat(h.ExchangeFee)
		cost := decimal.NewFromFloat(h.Cost)
		costBasis := decimal.NewFromFloat(h.CostBasis)
		realProfitLoss := decimal.NewFromFloat(h.RealProfitLoss)

		switch fill.GetDirection() {
		case gctorder.Buy, gctorder.Bid:
			if h.Amount >= 0 {
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
			if h.Amount > 0 {
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

		h.Amount, _ = amount.Round(common.DecimalPlaces).Float64()
		h.BoughtAmount, _ = amountBought.Round(common.DecimalPlaces).Float64()
		h.SoldAmount, _ = amountSold.Round(common.DecimalPlaces).Float64()
		h.AveragePrice, _ = avgPrice.Round(common.DecimalPlaces).Float64()
		h.AveragePriceBought, _ = avgPriceBought.Round(common.DecimalPlaces).Float64()
		h.AveragePriceSold, _ = avgPriceSold.Round(common.DecimalPlaces).Float64()
		h.AveragePriceNet, _ = avgPriceNet.Round(common.DecimalPlaces).Float64()
		h.Value, _ = value.Round(common.DecimalPlaces).Float64()
		h.ValueBought, _ = valueBought.Round(common.DecimalPlaces).Float64()
		h.ValueSold, _ = valueSold.Round(common.DecimalPlaces).Float64()
		h.NetValue, _ = netValue.Round(common.DecimalPlaces).Float64()
		h.NetValueBought, _ = netValueBought.Round(common.DecimalPlaces).Float64()
		h.NetValueSold, _ = netValueSold.Round(common.DecimalPlaces).Float64()
		h.ExchangeFee, _ = exchangeFee.Round(common.DecimalPlaces).Float64()
		h.Cost, _ = cost.Round(common.DecimalPlaces).Float64()
		h.CostBasis, _ = costBasis.Round(common.DecimalPlaces).Float64()
		h.RealProfitLoss, _ = realProfitLoss.Round(common.DecimalPlaces).Float64()

		h.updateValue(fill.GetClosePrice())

	*/
}

func (h *Holding) updateValue(l float64) {
	h.PositionsValue = h.PositionsSize * l
	h.BoughtValue = h.BoughtAmount * l
	h.SoldValue = h.SoldAmount * l
	h.TotalValue = h.PositionsValue + h.RemainingFunds
}
