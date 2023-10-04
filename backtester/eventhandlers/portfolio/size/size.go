package size

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// SizeOrder is responsible for ensuring that the order size is within config limits
func (s *Size) SizeOrder(o order.Event, amountAvailable decimal.Decimal, cs *exchange.Settings) (*order.Order, decimal.Decimal, error) {
	if o == nil {
		return nil, decimal.Zero, fmt.Errorf("%w order event", gctcommon.ErrNilPointer)
	}
	if cs == nil {
		return nil, decimal.Zero, fmt.Errorf("%w exchange settings", gctcommon.ErrNilPointer)
	}
	if amountAvailable.LessThanOrEqual(decimal.Zero) {
		return nil, decimal.Zero, errNoFunds
	}
	retOrder, ok := o.(*order.Order)
	if !ok {
		return nil, decimal.Zero, fmt.Errorf("%w expected order event", common.ErrInvalidDataType)
	}

	if fde := o.GetFillDependentEvent(); fde != nil && fde.MatchOrderAmount() {
		scalingInfo, err := cs.Exchange.ScaleCollateral(context.TODO(), &futures.CollateralCalculator{
			CalculateOffline:   true,
			CollateralCurrency: o.Pair().Base,
			Asset:              fde.GetAssetType(),
			Side:               gctorder.Short,
			USDPrice:           fde.GetClosePrice(),
			IsForNewPosition:   true,
			FreeCollateral:     amountAvailable,
		})
		if err != nil {
			return nil, decimal.Zero, err
		}

		sizedPrice := o.GetClosePrice()
		if fde.GetClosePrice().GreaterThan(o.GetClosePrice()) {
			// ensure limits are respected by using the largest price
			sizedPrice = fde.GetClosePrice()
		}

		initialAmount := amountAvailable.Mul(scalingInfo.Weighting).Div(fde.GetClosePrice())
		oNotionalPosition := initialAmount.Mul(sizedPrice)
		sizedAmount, estFee, err := s.calculateAmount(o.GetDirection(), sizedPrice, oNotionalPosition, cs, o)
		if err != nil {
			return nil, decimal.Zero, err
		}
		scaledCollateralFromAmount := sizedAmount.Mul(scalingInfo.Weighting)
		excess := amountAvailable.Sub(sizedAmount).Add(scaledCollateralFromAmount)
		if excess.IsNegative() {
			return nil, decimal.Zero, fmt.Errorf("%w not enough funding for position", errCannotAllocate)
		}
		retOrder.SetAmount(sizedAmount)
		fde.SetAmount(sizedAmount)

		return retOrder, estFee, nil
	}

	amount, estFee, err := s.calculateAmount(retOrder.Direction, retOrder.ClosePrice, amountAvailable, cs, o)
	if err != nil {
		return nil, decimal.Zero, err
	}
	retOrder.SetAmount(amount)

	return retOrder, estFee, nil
}

func (s *Size) calculateAmount(direction gctorder.Side, price, amountAvailable decimal.Decimal, cs *exchange.Settings, o order.Event) (amount, fee decimal.Decimal, err error) {
	var portfolioAmount, portfolioFee decimal.Decimal
	switch direction {
	case gctorder.ClosePosition:
		amount = amountAvailable
		fee = amount.Mul(price).Mul(cs.TakerFee)
	case gctorder.Buy, gctorder.Long:
		// check size against currency specific settings
		amount, fee, err = s.calculateBuySize(price, amountAvailable, cs.TakerFee, o.GetBuyLimit(), cs.BuySide)
		if err != nil {
			return decimal.Zero, decimal.Zero, err
		}
		// check size against portfolio specific settings
		portfolioAmount, portfolioFee, err = s.calculateBuySize(price, amountAvailable, cs.TakerFee, o.GetBuyLimit(), s.BuySide)
		if err != nil {
			return decimal.Zero, decimal.Zero, err
		}
		// global settings overrule individual currency settings
		if amount.GreaterThan(portfolioAmount) {
			amount = portfolioAmount
			fee = portfolioFee
		}
	case gctorder.Sell, gctorder.Short:
		// check size against currency specific settings
		amount, fee, err = s.calculateSellSize(price, amountAvailable, cs.TakerFee, o.GetSellLimit(), cs.SellSide)
		if err != nil {
			return decimal.Zero, decimal.Zero, err
		}
		// check size against portfolio specific settings
		portfolioAmount, portfolioFee, err = s.calculateSellSize(price, amountAvailable, cs.TakerFee, o.GetSellLimit(), s.SellSide)
		if err != nil {
			return decimal.Zero, decimal.Zero, err
		}
		// global settings overrule individual currency settings
		if amount.GreaterThan(portfolioAmount) {
			amount = portfolioAmount
			fee = portfolioFee
		}
	default:
		return decimal.Zero, decimal.Zero, fmt.Errorf("%w at %v for %v %v %v", errCannotAllocate, o.GetTime(), o.GetExchange(), o.GetAssetType(), o.Pair())
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, decimal.Zero, fmt.Errorf("%w at %v for %v %v %v, no amount sized", errCannotAllocate, o.GetTime(), o.GetExchange(), o.GetAssetType(), o.Pair())
	}

	if o.GetAmount().IsPositive() {
		// when an order amount is already set and still affordable
		// use the pre-set amount and calculate the fee
		if o.GetAmount().Mul(price).Add(o.GetAmount().Mul(price).Mul(cs.TakerFee)).LessThanOrEqual(amountAvailable) {
			// TODO: introduce option to fail + cancel original order if this order pricing fails
			amount = o.GetAmount()
			fee = o.GetAmount().Mul(price).Mul(cs.TakerFee)
		}
	}
	return amount, fee, nil
}

// calculateBuySize respects config rules and calculates the amount of money
// that is allowed to be spent/sold for an event.
// As fee calculation occurs during the actual ordering process
// this can only attempt to factor the potential fee to remain under the max rules
func (s *Size) calculateBuySize(price, availableFunds, feeRate, buyLimit decimal.Decimal, minMaxSettings exchange.MinMax) (amount, fee decimal.Decimal, err error) {
	if availableFunds.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, decimal.Zero, errNoFunds
	}
	if price.IsZero() {
		return decimal.Zero, decimal.Zero, nil
	}
	amount = availableFunds.Mul(decimal.NewFromInt(1).Sub(feeRate)).Div(price)
	if !buyLimit.IsZero() &&
		buyLimit.GreaterThanOrEqual(minMaxSettings.MinimumSize) &&
		(buyLimit.LessThanOrEqual(minMaxSettings.MaximumSize) || minMaxSettings.MaximumSize.IsZero()) &&
		buyLimit.LessThanOrEqual(amount) {
		amount = buyLimit
	}
	if minMaxSettings.MaximumSize.GreaterThan(decimal.Zero) && amount.GreaterThan(minMaxSettings.MaximumSize) {
		amount = minMaxSettings.MaximumSize.Mul(decimal.NewFromInt(1).Sub(feeRate))
	}
	if minMaxSettings.MaximumTotal.GreaterThan(decimal.Zero) && amount.Add(feeRate).Mul(price).GreaterThan(minMaxSettings.MaximumTotal) {
		amount = minMaxSettings.MaximumTotal.Mul(decimal.NewFromInt(1).Sub(feeRate)).Div(price)
	}
	if amount.LessThan(minMaxSettings.MinimumSize) && minMaxSettings.MinimumSize.GreaterThan(decimal.Zero) {
		return decimal.Zero, decimal.Zero, fmt.Errorf("%w. Sized: '%v' Minimum: '%v'", errLessThanMinimum, amount, minMaxSettings.MinimumSize)
	}
	fee = amount.Mul(price).Mul(feeRate)
	return amount, fee, nil
}

// calculateSellSize respects config rules and calculates the amount of money
// that is allowed to be spent/sold for an event.
// baseAmount is the base currency quantity that the portfolio currently has that can be sold
// eg BTC-USD baseAmount will be BTC to be sold
// As fee calculation occurs during the actual ordering process
// this can only attempt to factor the potential fee to remain under the max rules
func (s *Size) calculateSellSize(price, baseAmount, feeRate, sellLimit decimal.Decimal, minMaxSettings exchange.MinMax) (amount, fee decimal.Decimal, err error) {
	if baseAmount.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, decimal.Zero, errNoFunds
	}
	if price.IsZero() {
		return decimal.Zero, decimal.Zero, nil
	}
	oneMFeeRate := decimal.NewFromInt(1).Sub(feeRate)
	amount = baseAmount.Mul(oneMFeeRate)
	if !sellLimit.IsZero() &&
		sellLimit.GreaterThanOrEqual(minMaxSettings.MinimumSize) &&
		(sellLimit.LessThanOrEqual(minMaxSettings.MaximumSize) || minMaxSettings.MaximumSize.IsZero()) &&
		sellLimit.LessThanOrEqual(amount) {
		amount = sellLimit
	}
	if minMaxSettings.MaximumSize.GreaterThan(decimal.Zero) && amount.GreaterThan(minMaxSettings.MaximumSize) {
		amount = minMaxSettings.MaximumSize.Mul(oneMFeeRate)
	}
	if minMaxSettings.MaximumTotal.GreaterThan(decimal.Zero) && amount.Mul(price).GreaterThan(minMaxSettings.MaximumTotal) {
		amount = minMaxSettings.MaximumTotal.Mul(oneMFeeRate).Div(price)
	}
	if amount.LessThan(minMaxSettings.MinimumSize) && minMaxSettings.MinimumSize.GreaterThan(decimal.Zero) {
		return decimal.Zero, decimal.Zero, fmt.Errorf("%w. Sized: '%v' Minimum: '%v'", errLessThanMinimum, amount, minMaxSettings.MinimumSize)
	}
	fee = amount.Mul(price).Mul(feeRate)
	return amount, fee, nil
}
