package exchange

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
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

// ExecuteOrder assesses the portfolio manager's order event and if it passes validation
// will send an order to the exchange/fake order manager to be stored and raise a fill event
func (e *Exchange) ExecuteOrder(o order.Event, data data.Handler, bot *engine.Engine, funds funding.IPairReleaser) (*fill.Fill, error) {
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
		Direction:  o.GetDirection(),
		Amount:     o.GetAmount(),
		ClosePrice: data.Latest().ClosePrice(),
	}
	eventFunds := o.GetAllocatedFunds()
	cs, err := e.GetCurrencySettings(o.GetExchange(), o.GetAssetType(), o.Pair())
	if err != nil {
		return f, err
	}
	f.ExchangeFee = cs.ExchangeFee // defaulting to just using taker fee right now without orderbook
	f.Direction = o.GetDirection()
	if o.GetDirection() != gctorder.Buy && o.GetDirection() != gctorder.Sell {
		return f, nil
	}
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
		adjustedPrice, amount, err = e.sizeOfflineOrder(high, low, volume, &cs, f)
		if err != nil {
			switch f.GetDirection() {
			case gctorder.Buy:
				f.SetDirection(common.CouldNotBuy)
			case gctorder.Sell:
				f.SetDirection(common.CouldNotSell)
			default:
				f.SetDirection(common.DoNothing)
			}
			f.AppendReason(err.Error())
			return f, err
		}
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

	orderID, err := e.placeOrder(context.TODO(), adjustedPrice, limitReducedAmount, cs.UseRealOrders, cs.CanUseExchangeLimits, f, bot)
	if err != nil {
		fundErr := funds.Release(eventFunds, eventFunds, f.GetDirection())
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
		err = funds.Release(eventFunds, eventFunds.Sub(limitReducedAmount.Mul(adjustedPrice)), f.GetDirection())
		if err != nil {
			return f, err
		}
		funds.IncreaseAvailable(limitReducedAmount, f.GetDirection())
	case gctorder.Sell:
		err = funds.Release(eventFunds, eventFunds.Sub(limitReducedAmount), f.GetDirection())
		if err != nil {
			return f, err
		}
		funds.IncreaseAvailable(limitReducedAmount.Mul(adjustedPrice), f.GetDirection())
	}

	ords, _ := bot.OrderManager.GetOrdersSnapshot("")
	for i := range ords {
		if ords[i].ID != orderID {
			continue
		}
		ords[i].Date = o.GetTime()
		ords[i].LastUpdated = o.GetTime()
		ords[i].CloseTime = o.GetTime()
		f.Order = &ords[i]
		f.PurchasePrice = decimal.NewFromFloat(ords[i].Price)
		f.Total = f.PurchasePrice.Mul(limitReducedAmount).Add(f.ExchangeFee)
	}

	if f.Order == nil {
		return nil, fmt.Errorf("placed order %v not found in order manager", orderID)
	}

	return f, nil
}

// verifyOrderWithinLimits conforms the amount to fall into the minimum size and maximum size limit after reduced
func verifyOrderWithinLimits(f *fill.Fill, limitReducedAmount decimal.Decimal, cs *Settings) error {
	if f == nil {
		return common.ErrNilEvent
	}
	if cs == nil {
		return errNilCurrencySettings
	}
	isBeyondLimit := false
	var minMax config.MinMax
	var direction gctorder.Side
	switch f.GetDirection() {
	case gctorder.Buy:
		minMax = cs.BuySide
		direction = common.CouldNotBuy
	case gctorder.Sell:
		minMax = cs.SellSide
		direction = common.CouldNotSell
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

func (e *Exchange) placeOrder(ctx context.Context, price, amount decimal.Decimal, useRealOrders, useExchangeLimits bool, f *fill.Fill, bot *engine.Engine) (string, error) {
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
	fee, _ := f.ExchangeFee.Float64()
	o := &gctorder.Submit{
		Price:       p,
		Amount:      a,
		Fee:         fee,
		Exchange:    f.Exchange,
		ID:          u.String(),
		Side:        f.Direction,
		AssetType:   f.AssetType,
		Date:        f.GetTime(),
		LastUpdated: f.GetTime(),
		Pair:        f.Pair(),
		Type:        gctorder.Market,
	}

	if useRealOrders {
		resp, err := bot.OrderManager.Submit(ctx, o)
		if resp != nil {
			orderID = resp.OrderID
		}
		if err != nil {
			return orderID, err
		}
	} else {
		rate, _ := f.Amount.Float64()
		submitResponse := gctorder.SubmitResponse{
			IsOrderPlaced: true,
			OrderID:       u.String(),
			Rate:          rate,
			Fee:           fee,
			Cost:          p,
			FullyMatched:  true,
		}
		resp, err := bot.OrderManager.SubmitFakeOrder(o, submitResponse, useExchangeLimits)
		if resp != nil {
			orderID = resp.OrderID
		}
		if err != nil {
			return orderID, err
		}
	}
	return orderID, nil
}

func (e *Exchange) sizeOfflineOrder(high, low, volume decimal.Decimal, cs *Settings, f *fill.Fill) (adjustedPrice, adjustedAmount decimal.Decimal, err error) {
	if cs == nil || f == nil {
		return decimal.Zero, decimal.Zero, common.ErrNilArguments
	}
	// provide history and estimate volatility
	slippageRate := slippage.EstimateSlippagePercentage(cs.MinimumSlippageRate, cs.MaximumSlippageRate)
	if cs.SkipCandleVolumeFitting {
		f.VolumeAdjustedPrice = f.ClosePrice
		adjustedAmount = f.Amount
	} else {
		f.VolumeAdjustedPrice, adjustedAmount = ensureOrderFitsWithinHLV(f.ClosePrice, f.Amount, high, low, volume)
		if !adjustedAmount.Equal(f.Amount) {
			f.AppendReason(fmt.Sprintf("Order size shrunk from %v to %v to fit candle", f.Amount, adjustedAmount))
		}
	}

	if adjustedAmount.LessThanOrEqual(decimal.Zero) && f.Amount.GreaterThan(decimal.Zero) {
		return decimal.Zero, decimal.Zero, fmt.Errorf("amount set to 0, %w", errDataMayBeIncorrect)
	}
	adjustedPrice = applySlippageToPrice(f.GetDirection(), f.GetVolumeAdjustedPrice(), slippageRate)

	f.Slippage = slippageRate.Mul(decimal.NewFromInt(100)).Sub(decimal.NewFromInt(100))
	f.ExchangeFee = calculateExchangeFee(adjustedPrice, adjustedAmount, cs.ExchangeFee)
	return adjustedPrice, adjustedAmount, nil
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
func (e *Exchange) SetExchangeAssetCurrencySettings(exch string, a asset.Item, cp currency.Pair, c *Settings) {
	if c.ExchangeName == "" ||
		c.AssetType == "" ||
		c.CurrencyPair.IsEmpty() {
		return
	}

	for i := range e.CurrencySettings {
		if e.CurrencySettings[i].CurrencyPair == cp &&
			e.CurrencySettings[i].AssetType == a &&
			exch == e.CurrencySettings[i].ExchangeName {
			e.CurrencySettings[i] = *c
			return
		}
	}
	e.CurrencySettings = append(e.CurrencySettings, *c)
}

// GetCurrencySettings returns the settings for an exchange, asset currency
func (e *Exchange) GetCurrencySettings(exch string, a asset.Item, cp currency.Pair) (Settings, error) {
	for i := range e.CurrencySettings {
		if e.CurrencySettings[i].CurrencyPair.Equal(cp) {
			if e.CurrencySettings[i].AssetType == a {
				if exch == e.CurrencySettings[i].ExchangeName {
					return e.CurrencySettings[i], nil
				}
			}
		}
	}
	return Settings{}, fmt.Errorf("no currency settings found for %v %v %v", exch, a, cp)
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
