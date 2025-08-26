package limits

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Load loads all limits into private limit holder
func Load(levels []MinMaxLevel) error {
	return manager.load(levels)
}

// GetOrderExecutionLimits returns the order limit matching the key
func GetOrderExecutionLimits(k key.ExchangeAssetPair) (MinMaxLevel, error) {
	return manager.getOrderExecutionLimits(k)
}

// CheckOrderExecutionLimits is a convenience method to check if the price and amount conforms
// to the exchange order limits
func CheckOrderExecutionLimits(k key.ExchangeAssetPair, price, amount float64, orderType order.Type) error {
	return manager.checkOrderExecutionLimits(k, price, amount, orderType)
}

func (e *store) load(levels []MinMaxLevel) error {
	if len(levels) == 0 {
		return ErrEmptyLevels
	}
	e.mtx.Lock()
	defer e.mtx.Unlock()
	if e.epaLimits == nil {
		e.epaLimits = make(map[key.ExchangeAssetPair]*MinMaxLevel)
	}

	for x := range levels {
		if levels[x].Key.Exchange == "" {
			return fmt.Errorf("cannot load levels for %q %q: %w", levels[x].Key.Asset, levels[x].Key.Pair(), errExchangeNameEmpty)
		}
		if !levels[x].Key.Asset.IsValid() {
			return fmt.Errorf("cannot load levels for %q %q: %w", levels[x].Key.Exchange, levels[x].Key.Pair(), errAssetInvalid)
		}
		if levels[x].Key.Pair().IsEmpty() {
			return fmt.Errorf("cannot load levels for %q %q: %w", levels[x].Key.Exchange, levels[x].Key.Asset, errPairNotSet)
		}
		if levels[x].MinPrice > 0 &&
			levels[x].MaxPrice > 0 &&
			levels[x].MinPrice > levels[x].MaxPrice {
			return fmt.Errorf("%w for %q %q %q supplied min: %f max: %f",
				errInvalidPriceLevels,
				levels[x].Key.Exchange,
				levels[x].Key.Asset,
				levels[x].Key.Pair(),
				levels[x].MinPrice,
				levels[x].MaxPrice)
		}

		if levels[x].MinimumBaseAmount > 0 &&
			levels[x].MaximumBaseAmount > 0 &&
			levels[x].MinimumBaseAmount > levels[x].MaximumBaseAmount {
			return fmt.Errorf("%w for %q %q %q supplied min: %f max: %f",
				errInvalidAmountLevels,
				levels[x].Key.Exchange,
				levels[x].Key.Asset,
				levels[x].Key.Pair(),
				levels[x].MinimumBaseAmount,
				levels[x].MaximumBaseAmount)
		}

		if levels[x].MinimumQuoteAmount > 0 &&
			levels[x].MaximumQuoteAmount > 0 &&
			levels[x].MinimumQuoteAmount > levels[x].MaximumQuoteAmount {
			return fmt.Errorf("%w for %q %q %q supplied min: %f max: %f",
				errInvalidQuoteLevels,
				levels[x].Key.Exchange,
				levels[x].Key.Asset,
				levels[x].Key.Pair(),
				levels[x].MinimumQuoteAmount,
				levels[x].MaximumQuoteAmount)
		}
		levels[x].UpdatedAt = time.Now()
		e.epaLimits[levels[x].Key] = &levels[x]
	}
	return nil
}

func (e *store) getOrderExecutionLimits(k key.ExchangeAssetPair) (MinMaxLevel, error) {
	e.mtx.RLock()
	defer e.mtx.RUnlock()
	if e.epaLimits == nil {
		return MinMaxLevel{}, ErrExchangeLimitNotLoaded
	}
	el, ok := e.epaLimits[k]
	if !ok {
		return MinMaxLevel{}, fmt.Errorf("%w for %q %q %q", ErrOrderLimitNotFound, k.Exchange, k.Asset, k.Pair())
	}
	return *el, nil
}

func (e *store) checkOrderExecutionLimits(k key.ExchangeAssetPair, price, amount float64, orderType order.Type) error {
	e.mtx.RLock()
	defer e.mtx.RUnlock()
	if e.epaLimits == nil {
		return ErrExchangeLimitNotLoaded
	}
	m1, ok := e.epaLimits[k]
	if !ok {
		return fmt.Errorf("%w for %q %q %q", ErrOrderLimitNotFound, k.Exchange, k.Asset, k.Pair())
	}

	err := m1.Validate(price, amount, orderType)
	if err != nil {
		return fmt.Errorf("%w for %q %q %q", err, k.Exchange, k.Asset, k.Pair())
	}

	return nil
}
