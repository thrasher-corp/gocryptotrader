package size

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (s *Size) SizeOrder(o order.OrderEvent, _ interfaces.DataEventHandler, amountAvailable float64, cs *exchange.CurrencySettings) (*order.Order, error) {
	retOrder := o.(*order.Order)

	switch retOrder.GetDirection() {
	case gctorder.Buy:
		// check size against currency specific settings
		amount, err := s.calculateBuySize(retOrder.Price, amountAvailable, cs.ExchangeFee, cs.BuySide)
		if err != nil {
			return nil, err
		}
		// check size against portfolio specific settings
		portfolioSize, err := s.calculateBuySize(retOrder.Price, amountAvailable, cs.ExchangeFee, s.BuySide)
		if err != nil {
			return nil, err
		}
		// global settings overrule individual currency settings
		if amount > portfolioSize {
			amount = portfolioSize
		}

		retOrder.SetAmount(amount)
	case gctorder.Sell:
		// check size against currency specific settings
		amount, err := s.calculateSellSize(retOrder.Price, amountAvailable, cs.ExchangeFee, cs.SellSide)
		if err != nil {
			return nil, err
		}
		// check size against portfolio specific settings
		portfolioSize, err := s.calculateSellSize(retOrder.Price, amountAvailable, cs.ExchangeFee, s.SellSide)
		if err != nil {
			return nil, err
		}
		// global settings overrule individual currency settings
		if amount > portfolioSize {
			amount = portfolioSize
		}

		retOrder.SetAmount(amount)
	}

	return retOrder, nil
}

// calculateBuySize respects config rules and calculates the amount of money
// that is allowed to be spent/sold for an event.
//
// As fee calculation occurs during the actual ordering process
// this can only attempt to factor the potential fee to remain under the max rules
func (s *Size) calculateBuySize(price, availableFunds, feeRate float64, minMaxSettings config.MinMax) (float64, error) {
	if availableFunds <= 0 {
		return 0, errors.New("no fund available")
	}

	amount := availableFunds * (1 - feeRate) / price
	if minMaxSettings.MaximumSize > 0 && amount > minMaxSettings.MaximumSize {
		amount = minMaxSettings.MaximumSize * (1 - feeRate)
	}
	if minMaxSettings.MaximumTotal > 0 && (amount+feeRate)*price > minMaxSettings.MaximumTotal {
		amount = minMaxSettings.MaximumTotal * (1 - feeRate) / price
	}
	if amount < minMaxSettings.MinimumSize {
		return 0, fmt.Errorf("sized amount '%.8f' less than minimum '%v'", amount, minMaxSettings.MinimumSize)
	}

	return amount, nil
}

// calculateSellSize respects config rules and calculates the amount of money
// that is allowed to be spent/sold for an event.
//
// As fee calculation occurs during the actual ordering process
// this can only attempt to factor the potential fee to remain under the max rules
func (s *Size) calculateSellSize(price, availableFunds, feeRate float64, minMaxSettings config.MinMax) (float64, error) {
	if availableFunds <= 0 {
		return 0, errors.New("no fund available")
	}

	amount := availableFunds * (1 - feeRate)
	if minMaxSettings.MaximumSize > 0 && amount > minMaxSettings.MaximumSize {
		amount = minMaxSettings.MaximumSize * (1 - feeRate)
	}
	if minMaxSettings.MaximumTotal > 0 && amount*price > minMaxSettings.MaximumTotal {
		amount = minMaxSettings.MaximumTotal * (1 - feeRate) / price
	}
	if amount < minMaxSettings.MinimumSize {
		return 0, fmt.Errorf("sized amount '%.8f' less than minimum '%v'", amount, minMaxSettings.MinimumSize)
	}

	return amount, nil
}
