package exchange

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange/slippage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// Reset returns the exchange to initial settings
func (e *Exchange) Reset() {
	*e = Exchange{}
}

var ErrDoNothing = errors.New("received Do Nothing direction")

// ExecuteOrder assesses the portfolio manager's order event and if it passes validation
// will send an order to the exchange/fake order manager to be stored and raise a fill event
func (e *Exchange) ExecuteOrder(o order.Event, data data.Handler, orderManager *engine.OrderManager, funds funding.IFundReleaser) (fill.Event, error) {
	f := &fill.Fill{
		Base: event.Base{
			Offset:       o.GetOffset(),
			Exchange:     o.GetExchange(),
			Time:         o.GetTime(),
			CurrencyPair: o.Pair(),
			AssetType:    o.GetAssetType(),
			Interval:     o.GetInterval(),
			Reason:       o.GetReason(),
		},
		Direction:          o.GetDirection(),
		Amount:             o.GetAmount(),
		ClosePrice:         data.Latest().GetClosePrice(),
		FillDependentEvent: o.GetFillDependentEvent(),
	}
	if o.GetDirection() == common.DoNothing {
		return f, ErrDoNothing
	}
	if o.GetAssetType().IsFutures() && !o.IsClosingPosition() {
		f.Amount = o.GetAllocatedFunds()
	}
	eventFunds := o.GetAllocatedFunds()
	cs, err := e.GetCurrencySettings(o.GetExchange(), o.GetAssetType(), o.Pair())
	if err != nil {
		return f, err
	}
	f.ExchangeFee = cs.ExchangeFee
	f.Direction = o.GetDirection()
	highStr := data.StreamHigh()
	high := highStr[len(highStr)-1]

	lowStr := data.StreamLow()
	low := lowStr[len(lowStr)-1]

	volStr := data.StreamVol()
	volume := volStr[len(volStr)-1]
	var adjustedPrice, amount decimal.Decimal

	if cs.UseRealOrders {
		// get current orderbook
		var ob *orderbook.Base
		ob, err = orderbook.Get(f.Exchange, f.CurrencyPair, f.AssetType)
		if err != nil {
			return f, err
		}
		// calculate an estimated slippage rate
		adjustedPrice, amount = slippage.CalculateSlippageByOrderbook(ob, o.GetDirection(), eventFunds, f.ExchangeFee)
		f.Slippage = adjustedPrice.Sub(f.ClosePrice).Div(f.ClosePrice).Mul(decimal.NewFromInt(100))
	} else {
		slippageRate := slippage.EstimateSlippagePercentage(cs.MinimumSlippageRate, cs.MaximumSlippageRate)
		if cs.SkipCandleVolumeFitting || o.GetAssetType().IsFutures() {
			f.VolumeAdjustedPrice = f.ClosePrice
			amount = f.Amount
		} else {
			f.VolumeAdjustedPrice, amount = ensureOrderFitsWithinHLV(f.ClosePrice, f.Amount, high, low, volume)
			if !amount.Equal(f.GetAmount()) {
				f.AppendReason(fmt.Sprintf("Order size shrunk from %v to %v to fit candle", f.Amount, amount))
			}
		}
		if amount.LessThanOrEqual(decimal.Zero) && f.GetAmount().GreaterThan(decimal.Zero) {
			switch f.GetDirection() {
			case gctorder.Buy:
				f.SetDirection(common.CouldNotBuy)
			case gctorder.Sell:
				f.SetDirection(common.CouldNotSell)
			case gctorder.Short:
				f.SetDirection(common.CouldNotShort)
			case gctorder.Long:
				f.SetDirection(common.CouldNotLong)
			default:
				f.SetDirection(common.DoNothing)
			}
			f.AppendReason(fmt.Sprintf("amount set to 0, %s", errDataMayBeIncorrect))
			return f, err
		}
		adjustedPrice = applySlippageToPrice(f.GetDirection(), f.GetVolumeAdjustedPrice(), slippageRate)
		f.Slippage = slippageRate.Mul(decimal.NewFromInt(100)).Sub(decimal.NewFromInt(100))
		f.ExchangeFee = calculateExchangeFee(adjustedPrice, amount, cs.TakerFee)
	}

	portfolioLimitedAmount := reduceAmountToFitPortfolioLimit(adjustedPrice, amount, eventFunds, f.GetDirection())
	if !portfolioLimitedAmount.Equal(amount) {
		f.AppendReason(fmt.Sprintf("Order size shrunk from %v to %v to remain within portfolio limits", amount, portfolioLimitedAmount))
	}

	limitReducedAmount := portfolioLimitedAmount
	if cs.CanUseExchangeLimits {
		// Conforms the amount to the exchange order defined step amount
		// reducing it when needed
		limitReducedAmount = cs.Limits.ConformToDecimalAmount(portfolioLimitedAmount)
		if !limitReducedAmount.Equal(portfolioLimitedAmount) {
			f.AppendReason(fmt.Sprintf("Order size shrunk from %v to %v to remain within exchange step amount limits",
				portfolioLimitedAmount,
				limitReducedAmount))
		}
	}
	err = verifyOrderWithinLimits(f, limitReducedAmount, &cs)
	if err != nil {
		return f, err
	}
	f.ExchangeFee = calculateExchangeFee(adjustedPrice, limitReducedAmount, cs.ExchangeFee)

	orderID, err := e.placeOrder(context.TODO(), adjustedPrice, limitReducedAmount, cs.UseRealOrders, cs.CanUseExchangeLimits, f, orderManager)
	switch cs.Asset {
	case asset.Spot:
		pr, fundErr := funds.GetPairReleaser()
		if fundErr != nil {
			return f, fundErr
		}
		if err != nil {
			fundErr = pr.Release(eventFunds, eventFunds, f.GetDirection())
			if fundErr != nil {
				f.AppendReason(fundErr.Error())
			}
			if f.GetDirection() == gctorder.Buy {
				f.SetDirection(common.CouldNotBuy)
			} else if f.GetDirection() == gctorder.Sell {
				f.SetDirection(common.CouldNotSell)
			}
			return f, err
		}
		switch f.GetDirection() {
		case gctorder.Buy:
			fundErr = pr.Release(eventFunds, eventFunds.Sub(limitReducedAmount.Mul(adjustedPrice)), f.GetDirection())
			if fundErr != nil {
				return f, fundErr
			}
			pr.IncreaseAvailable(limitReducedAmount, f.GetDirection())
		case gctorder.Sell:
			fundErr = pr.Release(eventFunds, eventFunds.Sub(limitReducedAmount), f.GetDirection())
			if fundErr != nil {
				return f, fundErr
			}
			pr.IncreaseAvailable(limitReducedAmount.Mul(adjustedPrice), f.GetDirection())
		}
	case asset.Futures:
		cr, fundErr := funds.GetCollateralReleaser()
		if fundErr != nil {
			return f, fundErr
		}
		if err != nil {
			fundErr = cr.ReleaseContracts(o.GetAmount())
			if fundErr != nil {
				return f, fundErr
			}
			switch f.GetDirection() {
			case gctorder.Short:
				f.SetDirection(common.CouldNotShort)
			case gctorder.Long:
				f.SetDirection(common.CouldNotLong)
			}
			return f, err
		}
		// realising pnl for a closed futures order occurs in the
		// portfolio OnFill function
	}

	ords := orderManager.GetOrdersSnapshot("")
	for i := range ords {
		if ords[i].ID != orderID {
			continue
		}
		ords[i].Date = o.GetTime()
		ords[i].LastUpdated = o.GetTime()
		ords[i].CloseTime = o.GetTime()
		f.Order = &ords[i]
		f.PurchasePrice = decimal.NewFromFloat(ords[i].Price)
		if ords[i].AssetType.IsFutures() || f.GetDirection() == common.ClosePosition {
			f.Total = limitReducedAmount.Add(f.ExchangeFee)
		} else {
			f.Total = f.PurchasePrice.Mul(limitReducedAmount).Add(f.ExchangeFee)
		}
	}

	if f.Order == nil {
		return nil, fmt.Errorf("placed order %v not found in order manager", orderID)
	}

	return f, nil
}

// verifyOrderWithinLimits conforms the amount to fall into the minimum size and maximum size limit after reduced
func verifyOrderWithinLimits(f fill.Event, limitReducedAmount decimal.Decimal, cs *Settings) error {
	if f == nil {
		return common.ErrNilEvent
	}
	if cs == nil {
		return errNilCurrencySettings
	}
	isBeyondLimit := false
	var minMax MinMax
	var direction gctorder.Side
	switch f.GetDirection() {
	case gctorder.Buy:
		minMax = cs.BuySide
		direction = common.CouldNotBuy
	case gctorder.Sell:
		minMax = cs.SellSide
		direction = common.CouldNotSell
	case gctorder.Long:
		minMax = cs.BuySide
		direction = common.CouldNotLong
	case gctorder.Short:
		minMax = cs.SellSide
		direction = common.CouldNotShort
	case common.ClosePosition:
		return nil
	default:
		direction = f.GetDirection()
		f.SetDirection(common.DoNothing)
		return fmt.Errorf("%w: %v", errInvalidDirection, direction)
	}
	var minOrMax, belowExceed string
	var size decimal.Decimal
	if limitReducedAmount.LessThan(minMax.MinimumSize) && minMax.MinimumSize.GreaterThan(decimal.Zero) {
		isBeyondLimit = true
		belowExceed = "below"
		minOrMax = "minimum"
		size = minMax.MinimumSize
	}
	if limitReducedAmount.GreaterThan(minMax.MaximumSize) && minMax.MaximumSize.GreaterThan(decimal.Zero) {
		isBeyondLimit = true
		belowExceed = "exceeded"
		minOrMax = "maximum"
		size = minMax.MaximumSize
	}
	if isBeyondLimit {
		f.SetDirection(direction)
		e := fmt.Sprintf("Order size %v %s %s size %v", limitReducedAmount, belowExceed, minOrMax, size)
		f.AppendReason(e)
		return fmt.Errorf("%w %v", errExceededPortfolioLimit, e)
	}
	return nil
}

func reduceAmountToFitPortfolioLimit(adjustedPrice, amount, sizedPortfolioTotal decimal.Decimal, side gctorder.Side) decimal.Decimal {
	switch side {
	case gctorder.Buy:
		if adjustedPrice.Mul(amount).GreaterThan(sizedPortfolioTotal) {
			// adjusted amounts exceeds portfolio manager's allowed funds
			// the amount has to be reduced to equal the sizedPortfolioTotal
			amount = sizedPortfolioTotal.Div(adjustedPrice)
		}
	case gctorder.Sell:
		if amount.GreaterThan(sizedPortfolioTotal) {
			amount = sizedPortfolioTotal
		}
	}
	return amount
}

func (e *Exchange) placeOrder(ctx context.Context, price, amount decimal.Decimal, useRealOrders, useExchangeLimits bool, f fill.Event, orderManager *engine.OrderManager) (string, error) {
	if f == nil {
		return "", common.ErrNilEvent
	}
	u, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	var orderID string
	p, _ := price.Float64()
	a, _ := amount.Float64()
	fee, _ := f.GetExchangeFee().Float64()
	o := &gctorder.Submit{
		Price:       p,
		Amount:      a,
		Fee:         fee,
		Exchange:    f.GetExchange(),
		ID:          u.String(),
		Side:        f.GetDirection(),
		AssetType:   f.GetAssetType(),
		Date:        f.GetTime(),
		LastUpdated: f.GetTime(),
		Pair:        f.Pair(),
		Type:        gctorder.Market,
	}

	if useRealOrders {
		resp, err := orderManager.Submit(ctx, o)
		if resp != nil {
			orderID = resp.OrderID
		}
		if err != nil {
			return orderID, err
		}
	} else {
		rate, _ := f.GetAmount().Float64()
		submitResponse := gctorder.SubmitResponse{
			IsOrderPlaced: true,
			OrderID:       u.String(),
			Rate:          rate,
			Fee:           fee,
			Cost:          p,
			FullyMatched:  true,
		}
		resp, err := orderManager.SubmitFakeOrder(o, submitResponse, useExchangeLimits)
		if resp != nil {
			orderID = resp.OrderID
		}
		if err != nil {
			return orderID, err
		}
	}
	return orderID, nil
}

func applySlippageToPrice(direction gctorder.Side, price, slippageRate decimal.Decimal) decimal.Decimal {
	adjustedPrice := price
	if direction == gctorder.Buy {
		adjustedPrice = price.Add(price.Mul(decimal.NewFromInt(1).Sub(slippageRate)))
	} else if direction == gctorder.Sell {
		adjustedPrice = price.Mul(slippageRate)
	}
	return adjustedPrice
}

// SetExchangeAssetCurrencySettings sets the settings for an exchange, asset, currency
func (e *Exchange) SetExchangeAssetCurrencySettings(a asset.Item, cp currency.Pair, c *Settings) {
	if c.Exchange == nil ||
		c.Asset == "" ||
		c.Pair.IsEmpty() {
		return
	}

	for i := range e.CurrencySettings {
		if e.CurrencySettings[i].Pair.Equal(cp) &&
			e.CurrencySettings[i].Asset == a &&
			strings.EqualFold(c.Exchange.GetName(), e.CurrencySettings[i].Exchange.GetName()) {
			e.CurrencySettings[i] = *c
			return
		}
	}
	e.CurrencySettings = append(e.CurrencySettings, *c)
}

// GetCurrencySettings returns the settings for an exchange, asset currency
func (e *Exchange) GetCurrencySettings(exch string, a asset.Item, cp currency.Pair) (Settings, error) {
	for i := range e.CurrencySettings {
		if e.CurrencySettings[i].Pair.Equal(cp) {
			if e.CurrencySettings[i].Asset == a {
				if strings.EqualFold(exch, e.CurrencySettings[i].Exchange.GetName()) {
					return e.CurrencySettings[i], nil
				}
			}
		}
	}
	return Settings{}, fmt.Errorf("%w for %v %v %v", errNoCurrencySettingsFound, exch, a, cp)
}

func ensureOrderFitsWithinHLV(slippagePrice, amount, high, low, volume decimal.Decimal) (adjustedPrice, adjustedAmount decimal.Decimal) {
	adjustedPrice = slippagePrice
	if adjustedPrice.LessThan(low) {
		adjustedPrice = low
	}
	if adjustedPrice.GreaterThan(high) {
		adjustedPrice = high
	}
	if volume.LessThanOrEqual(decimal.Zero) {
		return adjustedPrice, adjustedAmount
	}
	currentVolume := amount.Mul(adjustedPrice)
	if currentVolume.GreaterThan(volume) {
		// reduce the volume to not exceed the total volume of the candle
		// it is slightly less than the total to still allow for the illusion
		// that open high low close values are valid with the remaining volume
		// this is very opinionated
		currentVolume = volume.Mul(decimal.NewFromFloat(0.99999999))
	}
	// extract the amount from the adjusted volume
	adjustedAmount = currentVolume.Div(adjustedPrice)

	return adjustedPrice, adjustedAmount
}

func calculateExchangeFee(price, amount, fee decimal.Decimal) decimal.Decimal {
	return fee.Mul(price).Mul(amount)
}
