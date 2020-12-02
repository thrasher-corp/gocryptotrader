package holdings

import (
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (s *Snapshots) GetLatestSnapshot() Holding {
	if len(s.Holdings) == 0 {
		return Holding{}
	}
	return s.Holdings[len(s.Holdings)-1]
}

func (s *Snapshots) GetSnapshotAtTimestamp(t time.Time) Holding {
	for i := range s.Holdings {
		if t.Equal(s.Holdings[i].Timestamp) {
			return s.Holdings[i]
		}
	}
	return Holding{}
}

func (s *Snapshots) GetPreviousSnapshot() Holding {
	if len(s.Holdings) == 0 {
		return Holding{}
	}
	if len(s.Holdings) == 1 {
		return s.Holdings[0]
	}
	return s.Holdings[len(s.Holdings)-2]
}

func Create(f fill.FillEvent, initialFunds float64) (Holding, error) {
	if f == nil {
		return Holding{}, errors.New("nil event received")
	}
	if initialFunds <= 0 {
		return Holding{}, errors.New("initial funds <= 0")
	}
	h := Holding{
		Pair:           f.Pair(),
		Asset:          f.GetAssetType(),
		Exchange:       f.GetExchange(),
		Timestamp:      f.GetTime(),
		InitialFunds:   initialFunds,
		RemainingFunds: initialFunds,
	}

	err := h.update(f)
	if err != nil {
		return h, err
	}
	return h, nil
}

func (h *Holding) Update(f fill.FillEvent) {
	h.Timestamp = f.GetTime()
	h.update(f)
}

func (h *Holding) UpdateValue(d interfaces.DataEventHandler) {
	h.Timestamp = d.GetTime()
	latest := d.Price()
	h.updateValue(latest)
}

func (h *Holding) update(f fill.FillEvent) error {
	direction := f.GetDirection()
	o := f.GetOrder()
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
	case common.DoNothing, common.CouldNotSell, common.CouldNotBuy:
	default:
		return fmt.Errorf("woah nelly, how'd we get here? %v", direction)
	}
	h.updateValue(f.GetClosePrice())
	/*
		fillAmount := decimal.NewFromFloat(f.GetAmount())
		fillPrice := decimal.NewFromFloat(f.GetPurchasePrice())
		fillExchangeFee := decimal.NewFromFloat(f.GetExchangeFee())
		fillNetValue := decimal.NewFromFloat(f.NetValue())

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

		switch f.GetDirection() {
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

		h.updateValue(f.GetClosePrice())

	*/
	return nil
}

func (h *Holding) updateValue(l float64) {
	h.PositionsValue = h.PositionsSize * l
	h.BoughtValue = h.BoughtAmount * l
	h.SoldValue = h.SoldAmount * l
	h.TotalValue = h.PositionsValue + h.RemainingFunds
}
