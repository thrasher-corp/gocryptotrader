package holdings

import (
	"errors"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func Create(e fill.Event, f funding.IPairReader, riskFreeRate decimal.Decimal) error {
	if e == nil {
		return common.ErrNilEvent
	}
	if f == nil {
		return errors.New("woah nelly")
	}
	if f.QuoteInitialFunds().LessThan(decimal.Zero) {
		return ErrInitialFundsZero
	}
	holding := Holding{
		Offset:         e.GetOffset(),
		Pair:           e.Pair(),
		Asset:          e.GetAssetType(),
		Exchange:       e.GetExchange(),
		Timestamp:      e.GetTime(),
		InitialFunds:   f.QuoteInitialFunds(),
		RemainingFunds: f.QuoteInitialFunds(),
		RiskFreeRate:   riskFreeRate,
	}
	holding.update(e, f)

	return nil
}

// Update calculates holding statistics for the events time
func (h *Holding) Update(e fill.Event, f funding.IPairReader) {
	h.Timestamp = e.GetTime()
	h.Offset = e.GetOffset()
	h.update(e, f)
}

// UpdateValue calculates the holding's value for a data event's time and price
func (h *Holding) UpdateValue(d common.DataEventHandler) {
	h.Timestamp = d.GetTime()
	latest := d.ClosePrice()
	h.Offset = d.GetOffset()
	h.updateValue(latest)
}

func (h *Holding) update(e fill.Event, f funding.IPairReader) {
	direction := e.GetDirection()
	o := e.GetOrder()
	if o != nil {
		amount := decimal.NewFromFloat(o.Amount)
		fee := decimal.NewFromFloat(o.Fee)
		price := decimal.NewFromFloat(o.Price)
		switch direction {
		case order.Buy:
			h.CommittedFunds = h.CommittedFunds.Add(amount.Mul(price).Add(fee))
			h.PositionsSize = h.PositionsSize.Add(amount)
			h.PositionsValue = h.PositionsValue.Add(amount.Mul(price))
			h.RemainingFunds = f.QuoteAvailable()
			h.TotalFees = h.TotalFees.Add(fee)
			h.BoughtAmount = h.BoughtAmount.Add(amount)
			h.BoughtValue = h.BoughtValue.Add(amount.Mul(price))
		case order.Sell:
			h.CommittedFunds = h.CommittedFunds.Sub(amount.Mul(price).Add(fee))
			h.PositionsSize = h.PositionsSize.Sub(amount)
			h.PositionsValue = h.PositionsValue.Sub(amount.Mul(price))
			h.RemainingFunds = f.BaseAvailable()
			h.TotalFees = h.TotalFees.Add(fee)
			h.SoldAmount = h.SoldAmount.Add(amount)
			h.SoldValue = h.SoldValue.Add(amount.Mul(price))
		case common.DoNothing, common.CouldNotSell, common.CouldNotBuy, common.MissingData, "":
		}
	}
	h.TotalValueLostToVolumeSizing = h.TotalValueLostToVolumeSizing.Add(e.GetClosePrice().Sub(e.GetVolumeAdjustedPrice()).Mul(e.GetAmount()))
	h.TotalValueLostToSlippage = h.TotalValueLostToSlippage.Add(e.GetVolumeAdjustedPrice().Sub(e.GetPurchasePrice()).Mul(e.GetAmount()))
	h.updateValue(e.GetClosePrice())
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
