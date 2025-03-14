package holdings

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Create makes a Holding struct to track total values of strategy holdings over the course of a backtesting run
func Create(ev ClosePriceReader, fundReader funding.IFundReader) (*Holding, error) {
	if ev == nil {
		return nil, common.ErrNilEvent
	}
	a := ev.GetAssetType()
	switch {
	case a.IsFutures():
		funds, err := fundReader.GetCollateralReader()
		if err != nil {
			return nil, err
		}
		return &Holding{
			Offset:            ev.GetOffset(),
			Pair:              ev.Pair(),
			Asset:             ev.GetAssetType(),
			Exchange:          ev.GetExchange(),
			Timestamp:         ev.GetTime(),
			QuoteInitialFunds: funds.InitialFunds(),
			QuoteSize:         funds.InitialFunds(),
			TotalInitialValue: funds.InitialFunds(),
		}, nil
	case a == asset.Spot:
		funds, err := fundReader.GetPairReader()
		if err != nil {
			return nil, err
		}
		if funds.QuoteInitialFunds().LessThan(decimal.Zero) {
			return nil, ErrInitialFundsZero
		}

		return &Holding{
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
	default:
		return nil, fmt.Errorf("%v %w", ev.GetAssetType(), asset.ErrNotSupported)
	}
}

// Update calculates holding statistics for the events time
func (h *Holding) Update(e fill.Event, f funding.IFundReader) error {
	h.Timestamp = e.GetTime()
	h.Offset = e.GetOffset()
	return h.update(e, f)
}

// UpdateValue calculates the holding's value for a data event's time and price
func (h *Holding) UpdateValue(d common.Event) error {
	if d == nil {
		return fmt.Errorf("%w event", gctcommon.ErrNilPointer)
	}
	h.Timestamp = d.GetTime()
	latest := d.GetClosePrice()
	h.Offset = d.GetOffset()
	h.scaleValuesToCurrentPrice(latest)
	return nil
}

func (h *Holding) update(e fill.Event, f funding.IFundReader) error {
	direction := e.GetDirection()
	o := e.GetOrder()
	if o == nil {
		h.scaleValuesToCurrentPrice(e.GetClosePrice())
		return nil
	}
	amount := decimal.NewFromFloat(o.Amount)
	fee := decimal.NewFromFloat(o.Fee)
	price := decimal.NewFromFloat(o.Price)
	a := e.GetAssetType()
	switch {
	case a == asset.Spot:
		spotR, err := f.GetPairReader()
		if err != nil {
			return err
		}
		h.BaseSize = spotR.BaseAvailable()
		h.QuoteSize = spotR.QuoteAvailable()
	case a.IsFutures():
		collat, err := f.GetCollateralReader()
		if err != nil {
			return err
		}
		h.BaseSize = collat.CurrentHoldings()
		h.QuoteSize = collat.AvailableFunds()
	default:
		return fmt.Errorf("%v %w", a, asset.ErrNotSupported)
	}

	h.BaseValue = h.BaseSize.Mul(price)
	h.TotalFees = h.TotalFees.Add(fee)
	if e.GetAssetType().IsFutures() {
		// responsibility of tracking futures orders is
		// with order.PositionTracker
		return nil
	}
	switch direction {
	case order.Buy, order.Bid:
		h.BoughtAmount = h.BoughtAmount.Add(amount)
		h.CommittedFunds = h.BaseSize.Mul(price)
	case order.Sell, order.Ask:
		h.SoldAmount = h.SoldAmount.Add(amount)
		h.CommittedFunds = h.BaseSize.Mul(price)
	}

	if !e.GetVolumeAdjustedPrice().IsZero() {
		h.TotalValueLostToVolumeSizing = h.TotalValueLostToVolumeSizing.Add(e.GetClosePrice().Sub(e.GetVolumeAdjustedPrice()).Mul(e.GetAmount()))
	}
	if !e.GetClosePrice().Equal(e.GetPurchasePrice()) && !e.GetPurchasePrice().IsZero() {
		h.TotalValueLostToSlippage = h.TotalValueLostToSlippage.Add(e.GetClosePrice().Sub(e.GetPurchasePrice()).Mul(e.GetAmount()))
	}
	h.scaleValuesToCurrentPrice(e.GetClosePrice())
	return nil
}

func (h *Holding) scaleValuesToCurrentPrice(currentPrice decimal.Decimal) {
	origPosValue := h.BaseValue
	origTotalValue := h.TotalValue
	h.BaseValue = h.BaseSize.Mul(currentPrice)
	h.TotalValue = h.BaseValue.Add(h.QuoteSize)

	h.TotalValueDifference = h.TotalValue.Sub(origTotalValue)
	h.PositionsValueDifference = h.BaseValue.Sub(origPosValue)

	if !origTotalValue.IsZero() {
		h.ChangeInTotalValuePercent = h.TotalValue.Sub(origTotalValue).Div(origTotalValue)
	}
}
