package holdings

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Create makes a Holding struct to track total values of strategy holdings over the course of a backtesting run
func Create(ev ClosePriceReader, fundReader funding.IFundReader) (Holding, error) {
	if ev == nil {
		return Holding{}, common.ErrNilEvent
	}

	if ev.GetAssetType().IsFutures() {
		funds, err := fundReader.GetCollateralReader()
		if err != nil {
			return Holding{}, err
		}
		return Holding{
			Offset:            ev.GetOffset(),
			Pair:              ev.Pair(),
			Asset:             ev.GetAssetType(),
			Exchange:          ev.GetExchange(),
			Timestamp:         ev.GetTime(),
			QuoteInitialFunds: funds.InitialFunds(),
			QuoteSize:         funds.InitialFunds(),
			TotalInitialValue: funds.InitialFunds(),
		}, nil
	} else if ev.GetAssetType() == asset.Spot {
		funds, err := fundReader.GetPairReader()
		if err != nil {
			return Holding{}, err
		}
		if funds.QuoteInitialFunds().LessThan(decimal.Zero) {
			return Holding{}, ErrInitialFundsZero
		}

		return Holding{
			Offset:            ev.GetOffset(),
			Pair:              ev.Pair(),
			Asset:             ev.GetAssetType(),
			Exchange:          ev.GetExchange(),
			Timestamp:         ev.GetTime(),
			QuoteInitialFunds: funds.QuoteInitialFunds(),
			QuoteSize:         funds.QuoteInitialFunds(),
			BaseInitialFunds:  funds.BaseInitialFunds(),
			BaseSize:          funds.BaseInitialFunds(),
			TotalInitialValue: funds.QuoteInitialFunds().Add(funds.BaseInitialFunds().Mul(ev.GetClosePrice())),
		}, nil
	}
	return Holding{}, fmt.Errorf("%v %w", ev.GetAssetType(), asset.ErrNotSupported)
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
		case order.Buy, order.Bid:
			h.BoughtAmount = h.BoughtAmount.Add(amount)
			h.BoughtValue = h.BoughtAmount.Mul(price)
		case order.Sell, order.Ask:
			h.SoldAmount = h.SoldAmount.Add(amount)
			h.SoldValue = h.SoldAmount.Mul(price)
		case common.DoNothing, common.CouldNotSell, common.CouldNotBuy, common.MissingData, common.TransferredFunds, "":
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
