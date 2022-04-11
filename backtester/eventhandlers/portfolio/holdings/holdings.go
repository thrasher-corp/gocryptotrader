package holdings

import (
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Create makes a Holding struct to track total values of strategy holdings over the course of a backtesting run
func Create(ev ClosePriceReader, funding funding.IPairReader) (Holding, error) {
	if ev == nil {
		return Holding{}, common.ErrNilEvent
	}
	if funding.QuoteInitialFunds().LessThan(decimal.Zero) {
		return Holding{}, ErrInitialFundsZero
	}

	return Holding{
		Offset:            ev.GetOffset(),
		Pair:              ev.Pair(),
		Asset:             ev.GetAssetType(),
		Exchange:          ev.GetExchange(),
		Timestamp:         ev.GetTime(),
		QuoteInitialFunds: funding.QuoteInitialFunds(),
		QuoteSize:         funding.QuoteInitialFunds(),
		BaseInitialFunds:  funding.BaseInitialFunds(),
		BaseSize:          funding.BaseInitialFunds(),
		TotalInitialValue: funding.QuoteInitialFunds().Add(funding.BaseInitialFunds().Mul(ev.GetClosePrice())),
	}, nil
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
	latest := d.GetClosePrice()
	h.Offset = d.GetOffset()
	h.updateValue(latest)
}

// HasInvestments determines whether there are any holdings in the base funds
func (h *Holding) HasInvestments() bool {
	return h.BaseSize.GreaterThan(decimal.Zero)
}

// HasFunds determines whether there are any holdings in the quote funds
func (h *Holding) HasFunds() bool {
	return h.QuoteSize.GreaterThan(decimal.Zero)
}

func (h *Holding) update(e fill.Event, f funding.IPairReader) {
	direction := e.GetDirection()
	if o := e.GetOrder(); o != nil {
		amount := decimal.NewFromFloat(o.Amount)
		fee := decimal.NewFromFloat(o.Fee)
		price := decimal.NewFromFloat(o.Price)
		h.BaseSize = f.BaseAvailable()
		h.QuoteSize = f.QuoteAvailable()
		h.BaseValue = h.BaseSize.Mul(price)
		h.TotalFees = h.TotalFees.Add(fee)
		switch direction {
		case order.Buy:
			h.BoughtAmount = h.BoughtAmount.Add(amount)
			h.BoughtValue = h.BoughtAmount.Mul(price)
		case order.Sell:
			h.SoldAmount = h.SoldAmount.Add(amount)
			h.SoldValue = h.SoldAmount.Mul(price)
		case order.DoNothing, order.CouldNotSell, order.CouldNotBuy, order.MissingData, order.TransferredFunds, order.UnknownSide:
		}
	}
	h.TotalValueLostToVolumeSizing = h.TotalValueLostToVolumeSizing.Add(e.GetClosePrice().Sub(e.GetVolumeAdjustedPrice()).Mul(e.GetAmount()))
	h.TotalValueLostToSlippage = h.TotalValueLostToSlippage.Add(e.GetVolumeAdjustedPrice().Sub(e.GetPurchasePrice()).Mul(e.GetAmount()))
	h.updateValue(e.GetClosePrice())
}

func (h *Holding) updateValue(latestPrice decimal.Decimal) {
	origPosValue := h.BaseValue
	origBoughtValue := h.BoughtValue
	origSoldValue := h.SoldValue
	origTotalValue := h.TotalValue
	h.BaseValue = h.BaseSize.Mul(latestPrice)
	h.BoughtValue = h.BoughtAmount.Mul(latestPrice)
	h.SoldValue = h.SoldAmount.Mul(latestPrice)
	h.TotalValue = h.BaseValue.Add(h.QuoteSize)

	h.TotalValueDifference = h.TotalValue.Sub(origTotalValue)
	h.BoughtValueDifference = h.BoughtValue.Sub(origBoughtValue)
	h.PositionsValueDifference = h.BaseValue.Sub(origPosValue)
	h.SoldValueDifference = h.SoldValue.Sub(origSoldValue)

	if !origTotalValue.IsZero() {
		h.ChangeInTotalValuePercent = h.TotalValue.Sub(origTotalValue).Div(origTotalValue)
	}
}
