package bybit

import (
	"fmt"
	"slices"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	validSides         = []string{sideBuy, sideSell}
	validOrderTypes    = []string{"Market", "Limit"}
	validOrderFilters  = []string{"Order", "tpslOrder", "StopOrder"}
	validTriggerPrices = []string{"", "LastPrice", "IndexPrice", "MarkPrice"}
)

// Validate checks the input parameters and returns an error if they are invalid.
func (r *PlaceOrderRequest) Validate() error {
	if err := isValidCategory(r.Category); err != nil {
		return err
	}
	if r.Symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if r.EnableBorrow {
		r.IsLeverage = 1
	}
	if !slices.Contains(validSides, r.Side) {
		return fmt.Errorf("%w: %q", order.ErrSideIsInvalid, r.Side)
	}
	if !slices.Contains(validOrderTypes, r.OrderType) {
		return fmt.Errorf("%w: %q", order.ErrTypeIsInvalid, r.OrderType)
	}
	if r.OrderQuantity <= 0 {
		return limits.ErrAmountBelowMin
	}
	switch r.TriggerDirection {
	case 0, 1, 2: // 0: None, 1: triggered when market price rises to triggerPrice, 2: triggered when market price falls to triggerPrice
	default:
		return fmt.Errorf("%w, triggerDirection: %d", errInvalidTriggerDirection, r.TriggerDirection)
	}
	if r.OrderFilter != "" {
		if r.Category != cSpot {
			return fmt.Errorf("%w, orderFilter is valid for 'spot' only", errInvalidCategory)
		}
		if !slices.Contains(validOrderFilters, r.OrderFilter) {
			return fmt.Errorf("%w, orderFilter=%s", errInvalidOrderFilter, r.OrderFilter)
		}
	}
	if !slices.Contains(validTriggerPrices, r.TriggerPriceType) {
		return errInvalidTriggerPriceType
	}

	return nil
}

// Validate checks the input parameters and returns an error if they are invalid
func (r *AmendOrderRequest) Validate() error {
	if err := isValidCategory(r.Category); err != nil {
		return err
	}
	if r.Symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if r.OrderID == "" && r.OrderLinkID == "" {
		return errEitherOrderIDOROrderLinkIDRequired
	}
	return nil
}

// Validate checks the input parameters and returns an error if they are invalid
func (r *CancelOrderRequest) Validate() error {
	if err := isValidCategory(r.Category); err != nil {
		return err
	}
	if r.Symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if r.OrderID == "" && r.OrderLinkID == "" {
		return errEitherOrderIDOROrderLinkIDRequired
	}
	if r.OrderFilter != "" {
		if r.Category != cSpot {
			return fmt.Errorf("%w, orderFilter is valid for 'spot' only", errInvalidCategory)
		}
		if !slices.Contains(validOrderFilters, r.OrderFilter) {
			return fmt.Errorf("%w, orderFilter=%s", errInvalidOrderFilter, r.OrderFilter)
		}
	}
	return nil
}
