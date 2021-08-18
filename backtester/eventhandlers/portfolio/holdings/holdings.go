package holdings

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Create takes a fill event and creates a new holding for the exchange, asset, pair
func Create(f fill.Event, initialFunds, riskFreeRate decimal.Decimal) (Holding, error) {
	if f == nil {
		return Holding{}, common.ErrNilEvent
	}
	if initialFunds.LessThan(decimal.Zero) {
		return Holding{}, ErrInitialFundsZero
	}
	h := Holding{
		Offset:         f.GetOffset(),
		Pair:           f.Pair(),
		Asset:          f.GetAssetType(),
		Exchange:       f.GetExchange(),
		Timestamp:      f.GetTime(),
		InitialFunds:   initialFunds,
		RemainingFunds: initialFunds,
		RiskFreeRate:   riskFreeRate,
	}
	h.update(f)

	return h, nil
}

// Update calculates holding statistics for the events time
func (h *Holding) Update(f fill.Event) {
	h.Timestamp = f.GetTime()
	h.Offset = f.GetOffset()
	h.update(f)
}

// UpdateValue calculates the holding's value for a data event's time and price
func (h *Holding) UpdateValue(d common.DataEventHandler) {
	h.Timestamp = d.GetTime()
	latest := d.ClosePrice()
	h.Offset = d.GetOffset()
	h.updateValue(latest)
}

func (h *Holding) update(f fill.Event) {
	direction := f.GetDirection()

	o := f.GetOrder()
	if o != nil {
		amount := decimal.NewFromFloat(o.Amount)
		fee := decimal.NewFromFloat(o.Fee)
		price := decimal.NewFromFloat(o.Price)
		switch direction {
		case order.Buy:
			h.CommittedFunds = h.CommittedFunds.Add(amount.Mul(price).Add(fee))
			h.PositionsSize = h.PositionsSize.Add(amount)
			h.PositionsValue = h.PositionsValue.Add(amount.Mul(price))
			h.RemainingFunds = h.RemainingFunds.Sub(amount.Mul(price).Add(fee))
			h.TotalFees = h.TotalFees.Add(fee)
			h.BoughtAmount = h.BoughtAmount.Add(amount)
			h.BoughtValue = h.BoughtValue.Add(amount.Mul(price))
		case order.Sell:
			h.CommittedFunds = h.CommittedFunds.Sub(amount.Mul(price).Add(fee))
			h.PositionsSize = h.PositionsSize.Sub(amount)
			h.PositionsValue = h.PositionsValue.Sub(amount.Mul(price))
			h.RemainingFunds = h.RemainingFunds.Add(amount.Mul(price).Sub(fee))
			h.TotalFees = h.TotalFees.Add(fee)
			h.SoldAmount = h.SoldAmount.Add(amount)
			h.SoldValue = h.SoldValue.Add(amount.Mul(price))
		case common.DoNothing, common.CouldNotSell, common.CouldNotBuy, common.MissingData, "":
		}
	}
	h.TotalValueLostToVolumeSizing = h.TotalValueLostToVolumeSizing.Add(f.GetClosePrice().Sub(f.GetVolumeAdjustedPrice()).Mul(f.GetAmount()))
	h.TotalValueLostToSlippage = h.TotalValueLostToSlippage.Add(f.GetVolumeAdjustedPrice().Sub(f.GetPurchasePrice()).Mul(f.GetAmount()))
	h.updateValue(f.GetClosePrice())
}

func (h *Holding) updateValue(l decimal.Decimal) {
	origPosValue := h.PositionsValue
	origBoughtValue := h.BoughtValue
	origSoldValue := h.SoldValue
	origTotalValue := h.TotalValue
	h.PositionsValue = h.PositionsSize.Mul(l)
	h.BoughtValue = h.BoughtAmount.Mul(l)
	h.SoldValue = h.SoldAmount.Mul(l)
	h.TotalValue = h.PositionsValue.Add(h.RemainingFunds)

	h.TotalValueDifference = h.TotalValue.Sub(origTotalValue)
	h.BoughtValueDifference = h.BoughtValue.Sub(origBoughtValue)
	h.PositionsValueDifference = h.PositionsValue.Sub(origPosValue)
	h.SoldValueDifference = h.SoldValue.Sub(origSoldValue)

	if !origTotalValue.IsZero() {
		h.ChangeInTotalValuePercent = h.TotalValue.Sub(origTotalValue).Div(origTotalValue)
	}
}
