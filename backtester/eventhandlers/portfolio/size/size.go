package size

import (
	"fmt"
	"math"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
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
	retOrder := o.(*order.Order)
	var amount decimal.Decimal
	var err error
	switch retOrder.GetDirection() {
	case gctorder.Buy:
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
		if amount > portfolioSize {
			amount = portfolioSize
		}

	case gctorder.Sell:
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
		if amount > portfolioSize {
			amount = portfolioSize
		}
	}
	amount = math.Floor(amount*100000000) / 100000000
	if amount <= 0 {
		return retOrder, fmt.Errorf("%w at %v for %v %v %v", errCannotAllocate, o.GetTime(), o.GetExchange(), o.GetAssetType(), o.Pair())
	}
	retOrder.SetAmount(amount)

	return retOrder, nil
}

// calculateBuySize respects config rules and calculates the amount of money
// that is allowed to be spent/sold for an event.
// As fee calculation occurs during the actual ordering process
// this can only attempt to factor the potential fee to remain under the max rules
func (s *Size) calculateBuySize(price, availableFunds, feeRate, buyLimit decimal.Decimal, minMaxSettings config.MinMax) (decimal.Decimal, error) {
	if availableFunds <= 0 {
		return 0, errNoFunds
	}
	if price == 0 {
		return 0, nil
	}
	amount := availableFunds * (1 - feeRate) / price
	if buyLimit != 0 && buyLimit >= minMaxSettings.MinimumSize && (buyLimit <= minMaxSettings.MaximumSize || minMaxSettings.MaximumSize == 0) && buyLimit <= amount {
		amount = buyLimit
	}
	if minMaxSettings.MaximumSize > 0 && amount > minMaxSettings.MaximumSize {
		amount = minMaxSettings.MaximumSize * (1 - feeRate)
	}
	if minMaxSettings.MaximumTotal > 0 && (amount+feeRate)*price > minMaxSettings.MaximumTotal {
		amount = minMaxSettings.MaximumTotal * (1 - feeRate) / price
	}
	if amount < minMaxSettings.MinimumSize && minMaxSettings.MinimumSize > 0 {
		return 0, fmt.Errorf("%w. Sized: '%.8f' Minimum: '%f'", errLessThanMinimum, amount, minMaxSettings.MinimumSize)
	}
	return amount, nil
}

// calculateSellSize respects config rules and calculates the amount of money
// that is allowed to be spent/sold for an event.
// baseAmount is the base currency quantity that the portfolio currently has that can be sold
// eg BTC-USD baseAmount will be BTC to be sold
// As fee calculation occurs during the actual ordering process
// this can only attempt to factor the potential fee to remain under the max rules
func (s *Size) calculateSellSize(price, baseAmount, feeRate, sellLimit decimal.Decimal, minMaxSettings config.MinMax) (decimal.Decimal, error) {
	if baseAmount <= 0 {
		return 0, errNoFunds
	}
	if price == 0 {
		return 0, nil
	}
	amount := baseAmount * (1 - feeRate)
	if sellLimit != 0 && sellLimit >= minMaxSettings.MinimumSize && (sellLimit <= minMaxSettings.MaximumSize || minMaxSettings.MaximumSize == 0) && sellLimit <= amount {
		amount = sellLimit
	}
	if minMaxSettings.MaximumSize > 0 && amount > minMaxSettings.MaximumSize {
		amount = minMaxSettings.MaximumSize * (1 - feeRate)
	}
	if minMaxSettings.MaximumTotal > 0 && amount*price > minMaxSettings.MaximumTotal {
		amount = minMaxSettings.MaximumTotal * (1 - feeRate) / price
	}
	if amount < minMaxSettings.MinimumSize && minMaxSettings.MinimumSize > 0 {
		return 0, fmt.Errorf("%w. Sized: '%.8f' Minimum: '%f'", errLessThanMinimum, amount, minMaxSettings.MinimumSize)
	}

	return amount, nil
}
