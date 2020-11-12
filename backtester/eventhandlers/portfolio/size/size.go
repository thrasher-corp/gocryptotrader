package size

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/internalordermanager"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (s *Size) SizeOrder(o internalordermanager.OrderEvent, _ interfaces.DataEventHandler, availableFunds float64, cs *exchange.CurrencySettings) (*order.Order, error) {
	retOrder := o.(*order.Order)

	if (s.DefaultBuySize == 0) || (s.DefaultSellSize == 0) {
		return nil, errors.New("no DefaultBuySize or DefaultSellSize set")
	}

	switch retOrder.GetDirection() {
	case gctorder.Buy:
		amount := s.calculateSize(retOrder.Price, availableFunds, cs.ExchangeFee, cs.DefaultBuySize)
		if s.MinimumBuySize > 0 && amount < s.MinimumBuySize {
			return nil, fmt.Errorf("calculated order size '%v' less than the minimum defined amount '%v'", amount, s.MinimumBuySize)
		}
		if s.MaximumBuySize > 0 && amount > s.MaximumBuySize {
			amount = s.MaximumBuySize
		}

		retOrder.SetAmount(amount)
	case gctorder.Sell:
		amount := s.calculateSize(retOrder.Price, availableFunds, cs.ExchangeFee, cs.DefaultSellSize)
		if s.MinimumSellSize > 0 && amount < s.MinimumSellSize {
			return nil, fmt.Errorf("calculated order size '%v' less than the minimum defined amount '%v'", amount, s.MinimumBuySize)
		}
		if s.MaximumSellSize > 0 && amount > s.MaximumSellSize {
			amount = s.MaximumSellSize
		}

		retOrder.SetAmount(amount)
	}

	return retOrder, nil
}

func (s *Size) calculateSize(price, availableFunds, feeRate, defaultSize float64) float64 {
	if availableFunds <= 0 {
		return 0
	}
	var amount float64
	if availableFunds/price > defaultSize {
		amount = defaultSize
	} else {
		amount = availableFunds / price
	}
	fee := amount * feeRate * price
	amountMinusFee := amount * price
	amountMinusFee -= fee
	amountMinusFee /= price
	return amountMinusFee
}
