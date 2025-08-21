package exchange

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange/slippage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Reset returns the exchange to initial settings
func (e *Exchange) Reset() error {
	if e == nil {
		return gctcommon.ErrNilPointer
	}
	e.CurrencySettings = nil
	return nil
}

// ExecuteOrder assesses the portfolio manager's order event and if it passes validation
// will send an order to the exchange/fake order manager to be stored and raise a fill event
func (e *Exchange) ExecuteOrder(o order.Event, dh data.Handler, om *engine.OrderManager, funds funding.IFundReleaser) (fill.Event, error) {
	f := &fill.Fill{
		Base:               o.GetBase(),
		Direction:          o.GetDirection(),
		Amount:             o.GetAmount(),
		ClosePrice:         o.GetClosePrice(),
		FillDependentEvent: o.GetFillDependentEvent(),
		Liquidated:         o.IsLiquidating(),
	}
	if !common.CanTransact(o.GetDirection()) {
		return f, fmt.Errorf("%w order direction %v", ErrCannotTransact, o.GetDirection())
	}

	allocatedFunds := o.GetAllocatedFunds()
	cs, err := e.GetCurrencySettings(o.GetExchange(), o.GetAssetType(), o.Pair())
	if err != nil {
		return f, err
	}
	f.Direction = o.GetDirection()

	var price, adjustedPrice,
		amount, adjustedAmount,
		fee decimal.Decimal
	amount = o.GetAmount()
	price = o.GetClosePrice()
	if cs.UseRealOrders {
		if o.IsLiquidating() {
			// Liquidation occurs serverside
			if o.GetAssetType().IsFutures() {
				var cr funding.ICollateralReleaser
				cr, err = funds.CollateralReleaser()
				if err != nil {
					return f, err
				}
				// update local records
				cr.Liquidate()
			} else {
				var pr funding.IPairReleaser
				pr, err = funds.PairReleaser()
				if err != nil {
					return f, err
				}
				// update local records
				pr.Liquidate()
			}
			return f, nil
		}
	} else {
		slippageRate := slippage.EstimateSlippagePercentage(cs.MinimumSlippageRate, cs.MaximumSlippageRate)
		if cs.SkipCandleVolumeFitting || o.GetAssetType().IsFutures() || o.GetDirection() == gctorder.ClosePosition {
			f.VolumeAdjustedPrice = f.ClosePrice
			amount = f.Amount
		} else {
			var latest data.Event
			latest, err = dh.Latest()
			if err != nil {
				return nil, err
			}
			adjustedPrice, adjustedAmount = ensureOrderFitsWithinHLV(price, amount, latest.GetHighPrice(), latest.GetLowPrice(), latest.GetVolume())
			if !amount.Equal(adjustedAmount) {
				f.AppendReasonf("Order size shrunk from %v to %v to fit candle", amount, adjustedAmount)
				amount = adjustedAmount
			}
			if !adjustedPrice.Equal(price) {
				f.AppendReasonf("Price adjusted fitting to candle from %v to %v", price, adjustedPrice)
				price = adjustedPrice
				f.VolumeAdjustedPrice = price
			}
		}
		adjustedPrice, err = applySlippageToPrice(f.GetDirection(), price, slippageRate)
		if err != nil {
			return f, err
		}
		if !adjustedPrice.Equal(price) {
			f.AppendReasonf("Price has slipped from %v to %v", price, adjustedPrice)
			price = adjustedPrice
		}
		f.Slippage = slippageRate.Mul(decimal.NewFromInt(100)).Sub(decimal.NewFromInt(100))
	}

	adjustedAmount = reduceAmountToFitPortfolioLimit(adjustedPrice, amount, allocatedFunds, f.GetDirection())
	if !adjustedAmount.Equal(amount) {
		f.AppendReasonf("Order size shrunk from %v to %v to remain within portfolio limits", amount, adjustedAmount)
		amount = adjustedAmount
	}

	if cs.CanUseExchangeLimits || cs.UseRealOrders {
		adjustedAmount = cs.Limits.FloorAmountToStepIncrementDecimal(amount)
		if !adjustedAmount.Equal(amount) && !adjustedAmount.IsZero() {
			f.AppendReasonf("Order size shrunk from %v to %v to remain within exchange step amount limits",
				amount,
				adjustedAmount)
			amount = adjustedAmount
		}
	}
	err = verifyOrderWithinLimits(f, amount, &cs)
	if err != nil {
		return f, err
	}

	fee = calculateExchangeFee(price, amount, cs.TakerFee)

	orderID, err := e.placeOrder(context.TODO(), price, amount, fee, cs.UseRealOrders, cs.CanUseExchangeLimits, f, om)
	if err != nil {
		f.AppendReasonf("could not place order: %v", err)
		setCannotPurchaseDirection(f)
		return f, err
	}
	ords := om.GetOrdersSnapshot(gctorder.UnknownStatus)
	for i := range ords {
		if ords[i].OrderID != orderID {
			continue
		}
		ords[i].Date = o.GetTime()
		ords[i].LastUpdated = o.GetTime()
		ords[i].CloseTime = o.GetTime()
		f.Order = &ords[i]
		f.PurchasePrice = decimal.NewFromFloat(ords[i].Price)
		f.Amount = decimal.NewFromFloat(ords[i].Amount)
		if ords[i].Fee > 0 {
			f.ExchangeFee = decimal.NewFromFloat(ords[i].Fee)
		}
		f.Total = f.PurchasePrice.Mul(f.Amount).Add(f.ExchangeFee)
	}
	if !o.IsLiquidating() {
		err = allocateFundsPostOrder(f, funds, err, o.GetAmount(), allocatedFunds, amount, price, fee)
		if err != nil {
			return f, err
		}
	}
	if f.Order == nil {
		return nil, fmt.Errorf("placed order %v not found in order manager", orderID)
	}
	f.AppendReason(summarisePosition(f.GetDirection(), f.Amount, f.Amount.Mul(f.PurchasePrice), f.ExchangeFee, f.Order.Pair, f.UnderlyingPair))
	return f, nil
}

func allocateFundsPostOrder(f *fill.Fill, funds funding.IFundReleaser, orderError error, orderAmount, allocatedFunds, limitReducedAmount, adjustedPrice, fee decimal.Decimal) error {
	if f == nil {
		return fmt.Errorf("%w: fill event", common.ErrNilEvent)
	}
	if funds == nil {
		return fmt.Errorf("%w: funding", gctcommon.ErrNilPointer)
	}

	switch f.AssetType {
	case asset.Spot:
		pr, err := funds.PairReleaser()
		if err != nil {
			return err
		}
		if orderError != nil {
			err = pr.Release(allocatedFunds, allocatedFunds, f.GetDirection())
			if err != nil {
				f.AppendReason(err.Error())
			}
			switch f.GetDirection() {
			case gctorder.Buy, gctorder.Bid:
				f.SetDirection(gctorder.CouldNotBuy)
			case gctorder.Sell, gctorder.Ask, gctorder.ClosePosition:
				f.SetDirection(gctorder.CouldNotSell)
			}
			return orderError
		}

		switch f.GetDirection() {
		case gctorder.Buy, gctorder.Bid:
			err = pr.Release(allocatedFunds, allocatedFunds.Sub(limitReducedAmount.Mul(adjustedPrice).Add(fee)), f.GetDirection())
			if err != nil {
				return err
			}
			err = pr.IncreaseAvailable(limitReducedAmount, f.GetDirection())
			if err != nil {
				return err
			}
		case gctorder.Sell, gctorder.Ask:
			err = pr.Release(allocatedFunds, allocatedFunds.Sub(limitReducedAmount), f.GetDirection())
			if err != nil {
				return err
			}
			err = pr.IncreaseAvailable(limitReducedAmount.Mul(adjustedPrice).Sub(fee), f.GetDirection())
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("%w asset type %v", common.ErrInvalidDataType, f.GetDirection())
		}
	case asset.Futures:
		cr, err := funds.CollateralReleaser()
		if err != nil {
			return err
		}
		if orderError != nil {
			err = cr.ReleaseContracts(orderAmount)
			if err != nil {
				return err
			}
			setCannotPurchaseDirection(f)
			return orderError
		}
	default:
		return fmt.Errorf("%w asset type %v", common.ErrInvalidDataType, f.AssetType)
	}
	return nil
}

func summarisePosition(direction gctorder.Side, orderAmount, orderTotal, orderFee decimal.Decimal, pair, underlying currency.Pair) string {
	baseCurr := pair.Base.String()
	quoteCurr := pair.Quote
	if !underlying.IsEmpty() {
		baseCurr = pair.String()
		quoteCurr = underlying.Quote
	}
	return fmt.Sprintf("Placed %s order of %v %v for %v %v, with %v %v in fees, totalling %v %v",
		direction,
		orderAmount.Round(8),
		baseCurr,
		orderTotal.Round(8),
		quoteCurr,
		orderFee.Round(8),
		quoteCurr,
		orderTotal.Add(orderFee).Round(8),
		quoteCurr,
	)
}

func setCannotPurchaseDirection(f fill.Event) {
	switch f.GetDirection() {
	case gctorder.Buy, gctorder.Bid:
		f.SetDirection(gctorder.CouldNotBuy)
	case gctorder.Sell, gctorder.Ask:
		f.SetDirection(gctorder.CouldNotSell)
	case gctorder.Long:
		f.SetDirection(gctorder.CouldNotLong)
	case gctorder.Short:
		f.SetDirection(gctorder.CouldNotShort)
	}
}

// verifyOrderWithinLimits conforms the amount to fall into the minimum size and maximum size limit after reduced
func verifyOrderWithinLimits(f fill.Event, amount decimal.Decimal, cs *Settings) error {
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
	case gctorder.Buy, gctorder.Bid, gctorder.Long:
		minMax = cs.BuySide
		direction = gctorder.CouldNotBuy
	case gctorder.Sell, gctorder.Ask, gctorder.Short:
		direction = gctorder.CouldNotSell
		minMax = cs.SellSide
	case gctorder.ClosePosition:
		return nil
	default:
		direction = f.GetDirection()
		f.SetDirection(gctorder.DoNothing)
		return fmt.Errorf("%w: %v", errInvalidDirection, direction)
	}
	var minOrMax, belowExceed string
	var size decimal.Decimal
	if amount.LessThan(minMax.MinimumSize) && minMax.MinimumSize.GreaterThan(decimal.Zero) {
		isBeyondLimit = true
		belowExceed = "below"
		minOrMax = "minimum"
		size = minMax.MinimumSize
	}
	if amount.GreaterThan(minMax.MaximumSize) && minMax.MaximumSize.GreaterThan(decimal.Zero) {
		isBeyondLimit = true
		belowExceed = "exceeded"
		minOrMax = "maximum"
		size = minMax.MaximumSize
	}
	if isBeyondLimit {
		f.SetDirection(direction)
		e := fmt.Sprintf("Order size %v %s %s size %v", amount, belowExceed, minOrMax, size)
		f.AppendReason(e)
		return errExceededPortfolioLimit
	}
	return nil
}

func reduceAmountToFitPortfolioLimit(adjustedPrice, amount, sizedPortfolioTotal decimal.Decimal, side gctorder.Side) decimal.Decimal {
	switch side {
	case gctorder.Buy, gctorder.Bid:
		if adjustedPrice.Mul(amount).GreaterThan(sizedPortfolioTotal) {
			// adjusted amounts exceeds portfolio manager's allowed funds
			// the amount has to be reduced to equal the sizedPortfolioTotal
			amount = sizedPortfolioTotal.Div(adjustedPrice)
		}
	case gctorder.Sell, gctorder.Ask:
		if amount.GreaterThan(sizedPortfolioTotal) {
			amount = sizedPortfolioTotal
		}
	}
	return amount
}

func (e *Exchange) placeOrder(ctx context.Context, price, amount, fee decimal.Decimal, useRealOrders, useExchangeLimits bool, f fill.Event, orderManager *engine.OrderManager) (string, error) {
	if f == nil {
		return "", common.ErrNilEvent
	}
	orderID, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	submit := &gctorder.Submit{
		Price:            price.InexactFloat64(),
		Amount:           amount.InexactFloat64(),
		Exchange:         f.GetExchange(),
		Side:             f.GetDirection(),
		AssetType:        f.GetAssetType(),
		Pair:             f.Pair(),
		Type:             gctorder.Market,
		RetrieveFees:     true,
		RetrieveFeeDelay: time.Millisecond * 500,
	}

	var resp *engine.OrderSubmitResponse
	if useRealOrders {
		resp, err = orderManager.Submit(ctx, submit)
	} else {
		var submitResponse *gctorder.SubmitResponse
		submitResponse, err = submit.DeriveSubmitResponse(orderID.String())
		if err != nil {
			return orderID.String(), err
		}
		submitResponse.Status = gctorder.Filled
		submitResponse.OrderID = orderID.String()
		submitResponse.Fee = fee.InexactFloat64()
		submitResponse.Cost = submit.Price
		submitResponse.LastUpdated = f.GetTime()
		submitResponse.Date = f.GetTime()
		resp, err = orderManager.SubmitFakeOrder(submit, submitResponse, useExchangeLimits)
	}
	if err != nil {
		return orderID.String(), err
	}
	return resp.OrderID, nil
}

func applySlippageToPrice(direction gctorder.Side, price, slippageRate decimal.Decimal) (decimal.Decimal, error) {
	var adjustedPrice decimal.Decimal
	switch direction {
	case gctorder.Buy, gctorder.Bid, gctorder.Long:
		adjustedPrice = price.Add(price.Mul(decimal.NewFromInt(1).Sub(slippageRate)))
	case gctorder.Sell, gctorder.Ask, gctorder.Short:
		adjustedPrice = price.Mul(slippageRate)
	default:
		return decimal.Zero, fmt.Errorf("%v %w", direction, gctorder.ErrSideIsInvalid)
	}
	if adjustedPrice.IsZero() {
		adjustedPrice = price
	}

	return adjustedPrice, nil
}

// SetExchangeAssetCurrencySettings sets the settings for an exchange, asset, currency
func (e *Exchange) SetExchangeAssetCurrencySettings(a asset.Item, cp currency.Pair, c *Settings) {
	if c.Exchange == nil ||
		c.Asset == asset.Empty ||
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

func ensureOrderFitsWithinHLV(price, amount, high, low, volume decimal.Decimal) (adjustedPrice, adjustedAmount decimal.Decimal) {
	adjustedPrice = price
	if adjustedPrice.LessThan(low) {
		adjustedPrice = low
	}
	if adjustedPrice.GreaterThan(high) {
		adjustedPrice = high
	}
	if volume.LessThanOrEqual(decimal.Zero) || amount.LessThanOrEqual(volume) {
		return adjustedPrice, amount
	}
	if amount.GreaterThan(volume) {
		// reduce the volume to not exceed the total volume of the candle
		// it is slightly less than the total to still allow for the illusion
		// that open high low close values are valid with the remaining volume
		// this is very opinionated
		adjustedAmount = volume.Mul(decimal.NewFromFloat(0.99999999))
	}
	return adjustedPrice, adjustedAmount
}

func calculateExchangeFee(price, amount, fee decimal.Decimal) decimal.Decimal {
	return fee.Mul(price).Mul(amount)
}
