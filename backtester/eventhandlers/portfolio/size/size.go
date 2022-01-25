package size

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// SizeOrder is responsible for ensuring that the order size is within config limits
func (s *Size) SizeOrder(o order.Event, amountAvailable decimal.Decimal, cs *exchange.Settings) (*order.Order, error) {
	if o == nil || cs == nil {
		return nil, common.ErrNilArguments
	}
	if amountAvailable.LessThanOrEqual(decimal.Zero) {
		return nil, errNoFunds
	}
	retOrder, ok := o.(*order.Order)
	if !ok {
		return nil, fmt.Errorf("%w expected order event", common.ErrInvalidDataType)
	}
	var amount decimal.Decimal
	var err error
	switch retOrder.GetDirection() {
	case gctorder.Buy, gctorder.Long:
		// check size against currency specific settings
		amount, err = s.calculateBuySize(retOrder.Price, amountAvailable, cs.ExchangeFee, o.GetBuyLimit(), cs.BuySide)
		if err != nil {
			return nil, err
		}
		// check size against portfolio specific settings
		var portfolioSize decimal.Decimal
		portfolioSize, err = s.calculateBuySize(retOrder.Price, amountAvailable, cs.ExchangeFee, o.GetBuyLimit(), s.BuySide)
		if err != nil {
			return nil, err
		}
		// global settings overrule individual currency settings
		if amount.GreaterThan(portfolioSize) {
			amount = portfolioSize
		}
	case gctorder.Sell, gctorder.Short:
		// check size against currency specific settings
		amount, err = s.calculateSellSize(retOrder.Price, amountAvailable, cs.ExchangeFee, o.GetSellLimit(), cs.SellSide)
		if err != nil {
			return nil, err
		}
		// check size against portfolio specific settings
		portfolioSize, err := s.calculateSellSize(retOrder.Price, amountAvailable, cs.ExchangeFee, o.GetSellLimit(), s.SellSide)
		if err != nil {
			return nil, err
		}
		// global settings overrule individual currency settings
		if amount.GreaterThan(portfolioSize) {
			amount = portfolioSize
		}
	}
	amount = amount.Round(8)
	if amount.LessThanOrEqual(decimal.Zero) {
		return retOrder, fmt.Errorf("%w at %v for %v %v %v", errCannotAllocate, o.GetTime(), o.GetExchange(), o.GetAssetType(), o.Pair())
	}
	retOrder.SetAmount(amount)

	return retOrder, nil
}

// calculateBuySize respects config rules and calculates the amount of money
// that is allowed to be spent/sold for an event.
// As fee calculation occurs during the actual ordering process
// this can only attempt to factor the potential fee to remain under the max rules
func (s *Size) calculateBuySize(price, availableFunds, feeRate, buyLimit decimal.Decimal, minMaxSettings exchange.MinMax) (decimal.Decimal, error) {
	if availableFunds.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, errNoFunds
	}
	if price.IsZero() {
		return decimal.Zero, nil
	}
	amount := availableFunds.Mul(decimal.NewFromInt(1).Sub(feeRate)).Div(price)
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
		return decimal.Zero, fmt.Errorf("%w. Sized: '%v' Minimum: '%v'", errLessThanMinimum, amount, minMaxSettings.MinimumSize)
	}
	return amount, nil
}

// calculateSellSize respects config rules and calculates the amount of money
// that is allowed to be spent/sold for an event.
// baseAmount is the base currency quantity that the portfolio currently has that can be sold
// eg BTC-USD baseAmount will be BTC to be sold
// As fee calculation occurs during the actual ordering process
// this can only attempt to factor the potential fee to remain under the max rules
func (s *Size) calculateSellSize(price, baseAmount, feeRate, sellLimit decimal.Decimal, minMaxSettings exchange.MinMax) (decimal.Decimal, error) {
	if baseAmount.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, errNoFunds
	}
	if price.IsZero() {
		return decimal.Zero, nil
	}
	oneMFeeRate := decimal.NewFromInt(1).Sub(feeRate)
	amount := baseAmount.Mul(oneMFeeRate)
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
		return decimal.Zero, fmt.Errorf("%w. Sized: '%v' Minimum: '%v'", errLessThanMinimum, amount, minMaxSettings.MinimumSize)
	}

	return amount, nil
}
