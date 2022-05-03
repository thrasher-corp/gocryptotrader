package size

import (
	"context"
	"fmt"
	"os"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
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
	if fde := o.GetFillDependentEvent(); fde != nil && fde.MatchOrderAmount() {
		hello, err := cs.Exchange.ScaleCollateral(context.TODO(), &gctorder.CollateralCalculator{
			CalculateOffline:   true,
			CollateralCurrency: o.Pair().Base,
			Asset:              fde.GetAssetType(),
			Side:               gctorder.Short,
			USDPrice:           fde.GetClosePrice(),
			IsForNewPosition:   true,
			FreeCollateral:     amountAvailable,
		})
		initialAmount := amountAvailable.Mul(hello.Weighting).Div(fde.GetClosePrice())
		oNotionalPosition := initialAmount.Mul(o.GetClosePrice())
		sizedAmount, err := s.calculateAmount(o.GetDirection(), o.GetClosePrice(), oNotionalPosition, cs, o)
		if err != nil {
			return retOrder, err
		}
		scaledCollateralFromAmount := sizedAmount.Mul(hello.Weighting)
		excess := amountAvailable.Sub(sizedAmount).Add(scaledCollateralFromAmount)
		if excess.IsNegative() {
			os.Exit(-1)
		}
		retOrder.SetAmount(sizedAmount)
		fde.SetAmount(sizedAmount)
		retOrder.FillDependentEvent = fde
		log.Infof(common.Backtester, "%v %v", hello.CollateralContribution, err)
		log.Infof(common.Backtester, "%v %v", hello, err)
		return retOrder, nil
	}

	amount, err := s.calculateAmount(retOrder.Direction, retOrder.ClosePrice, amountAvailable, cs, o)
	if err != nil {
		return retOrder, err
	}
	retOrder.SetAmount(amount)

	return retOrder, nil
}

func (s *Size) calculateAmount(direction gctorder.Side, closePrice, amountAvailable decimal.Decimal, cs *exchange.Settings, o order.Event) (decimal.Decimal, error) {
	var amount decimal.Decimal
	var err error
	switch direction {
	case common.ClosePosition:
		amount = amountAvailable
	case gctorder.Buy, gctorder.Long:
		// check size against currency specific settings
		amount, err = s.calculateBuySize(closePrice, amountAvailable, cs.ExchangeFee, o.GetBuyLimit(), cs.BuySide)
		if err != nil {
			return decimal.Decimal{}, err
		}
		// check size against portfolio specific settings
		var portfolioSize decimal.Decimal
		portfolioSize, err = s.calculateBuySize(closePrice, amountAvailable, cs.ExchangeFee, o.GetBuyLimit(), s.BuySide)
		if err != nil {
			return decimal.Decimal{}, err
		}
		// global settings overrule individual currency settings
		if amount.GreaterThan(portfolioSize) {
			amount = portfolioSize
		}
	case gctorder.Sell, gctorder.Short:
		// check size against currency specific settings
		amount, err = s.calculateSellSize(closePrice, amountAvailable, cs.ExchangeFee, o.GetSellLimit(), cs.SellSide)
		if err != nil {
			return decimal.Decimal{}, err
		}
		// check size against portfolio specific settings
		portfolioSize, err := s.calculateSellSize(closePrice, amountAvailable, cs.ExchangeFee, o.GetSellLimit(), s.SellSide)
		if err != nil {
			return decimal.Decimal{}, err
		}
		// global settings overrule individual currency settings
		if amount.GreaterThan(portfolioSize) {
			amount = portfolioSize
		}
	default:
		return decimal.Decimal{}, fmt.Errorf("%w at %v for %v %v %v", errCannotAllocate, o.GetTime(), o.GetExchange(), o.GetAssetType(), o.Pair())
	}
	if o.GetAmount().IsPositive() {
		setAmountSize := o.GetAmount().Mul(closePrice)
		if setAmountSize.LessThan(amount) {
			amount = setAmountSize
		}
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return decimal.Decimal{}, fmt.Errorf("%w at %v for %v %v %v, no amount sized", errCannotAllocate, o.GetTime(), o.GetExchange(), o.GetAssetType(), o.Pair())
	}
	return amount, nil
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
