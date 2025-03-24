package limits

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func LoadLimits(levels []MinMaxLevel) error {
	return executionLimitsManager.LoadLimits(levels)
}

func GetOrderExecutionLimits(k key.ExchangePairAsset) (MinMaxLevel, error) {
	return executionLimitsManager.GetOrderExecutionLimits(k)
}

func CheckOrderExecutionLimits(k key.ExchangePairAsset, price, amount float64, orderType order.Type) error {
	return executionLimitsManager.CheckOrderExecutionLimits(k, price, amount, orderType)
}

// LoadLimits loads all limits levels into memory
func (e *executionLimits) LoadLimits(levels []MinMaxLevel) error {
	if len(levels) == 0 {
		return errCannotLoadLimit
	}
	e.mtx.Lock()
	defer e.mtx.Unlock()
	if e.epa == nil {
		e.epa = make(map[key.ExchangePairAsset]*MinMaxLevel)
	}

	for x := range levels {
		if !levels[x].Key.Asset.IsValid() {
			return fmt.Errorf("cannot load levels for '%s''%s''%s': %w", levels[x].Key.Exchange, levels[x].Key.Asset, levels[x].Key.Pair(), asset.ErrNotSupported)
		}
		levels[x].Key.Exchange = strings.ToLower(levels[x].Key.Exchange)
		if levels[x].MinPrice > 0 &&
			levels[x].MaxPrice > 0 &&
			levels[x].MinPrice > levels[x].MaxPrice {
			return fmt.Errorf("%w for %s %s %s supplied min: %f max: %f",
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
			return fmt.Errorf("%w for %s %s %s supplied min: %f max: %f",
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
			return fmt.Errorf("%w for %s %s %s supplied min: %f max: %f",
				errInvalidQuoteLevels,
				levels[x].Key.Exchange,
				levels[x].Key.Asset,
				levels[x].Key.Pair(),
				levels[x].MinimumQuoteAmount,
				levels[x].MaximumQuoteAmount)
		}
		levels[x].LastUpdated = time.Now()
		e.epa[levels[x].Key] = &levels[x]
	}
	go e.proliferate()
	return nil
}

func (e *executionLimits) proliferate() {
	e.proliferationMTX.Lock()
	defer e.proliferationMTX.Unlock()
	e.mtx.RLock()
	defer e.mtx.RUnlock()
	for x := range e.epa {
		if e.epa[x] == nil {
			continue
		}
		eak := key.ExchangeAsset{
			Exchange: x.Exchange,
			Asset:    x.Asset,
		}
		ea, ok := e.ea[eak]
		if !ok {
			e.ea[eak] = []*MinMaxLevel{e.epa[x]}
		} else if !slices.Contains(ea, e.epa[x]) {
			ea = append(ea, e.epa[x])
		}
		e.ea[eak] = ea

		ex, ok := e.e[x.Exchange]
		if !ok {
			e.e[x.Exchange] = []*MinMaxLevel{e.epa[x]}
		} else if !slices.Contains(ea, e.epa[x]) {
			ex = append(ex, e.epa[x])
		}
		e.e[x.Exchange] = ex
	}
}

// GetOrderExecutionLimits returns the exchange limit parameters for a currency
func (e *executionLimits) GetOrderExecutionLimits(k key.ExchangePairAsset) (MinMaxLevel, error) {
	e.mtx.RLock()
	defer e.mtx.RUnlock()
	if e.epa == nil {
		return MinMaxLevel{}, ErrExchangeLimitNotLoaded
	}
	k.Exchange = strings.ToLower(k.Exchange)
	if el, ok := e.epa[k]; !ok {
		e.proliferationMTX.Lock()
		defer e.proliferationMTX.Unlock()
		if e.ea[key.ExchangeAsset{Exchange: k.Exchange, Asset: k.Asset}] == nil {
			return MinMaxLevel{}, fmt.Errorf("%v %w no list yet", k, ErrNotFound)
		}

		fmt.Println("list", e.ea[key.ExchangeAsset{Exchange: k.Exchange, Asset: k.Asset}])
		return MinMaxLevel{}, fmt.Errorf("%w for %s %s %s", ErrExchangeLimitNotLoaded, k.Exchange, k.Asset, k.Pair())
	} else {
		return *el, nil
	}
}

// CheckOrderExecutionLimits checks to see if the price and amount conforms with
// exchange level order execution limits
func (e *executionLimits) CheckOrderExecutionLimits(k key.ExchangePairAsset, price, amount float64, orderType order.Type) error {
	e.mtx.RLock()
	defer e.mtx.RUnlock()

	if e.epa == nil {
		// No exchange limits loaded so we can nil this
		return nil
	}

	k.Exchange = strings.ToLower(k.Exchange)
	m1, ok := e.epa[k]
	if !ok {
		if e.ea[key.ExchangeAsset{Exchange: k.Exchange, Asset: k.Asset}] == nil {
			return fmt.Errorf("%v %w", k, ErrNotFound)
		}
		fmt.Println("list", e.ea[key.ExchangeAsset{Exchange: k.Exchange, Asset: k.Asset}])
		return fmt.Errorf("%w for %s %s %s", ErrExchangeLimitNotLoaded, k.Exchange, k.Asset, k.Pair())
	}

	err := m1.Conforms(price, amount, orderType)
	if err != nil {
		return fmt.Errorf("%w for %s %s %s", err, k.Exchange, k.Asset, k.Pair())
	}

	return nil
}

// Conforms checks outbound parameters
func (m *MinMaxLevel) Conforms(price, amount float64, orderType order.Type) error {
	// TODO: Update to take in account Quote amounts as well as Base amounts.
	if m == nil {
		return nil
	}

	if m.MinimumBaseAmount != 0 && amount < m.MinimumBaseAmount {
		return fmt.Errorf("%w min: %.8f supplied %.8f",
			ErrAmountBelowMin,
			m.MinimumBaseAmount,
			amount)
	}
	if m.MaximumBaseAmount != 0 && amount > m.MaximumBaseAmount {
		return fmt.Errorf("%w min: %.8f supplied %.8f",
			ErrAmountExceedsMax,
			m.MaximumBaseAmount,
			amount)
	}
	if m.AmountStepIncrementSize != 0 {
		dAmount := decimal.NewFromFloat(amount)
		dStep := decimal.NewFromFloat(m.AmountStepIncrementSize)
		if !dAmount.Mod(dStep).IsZero() {
			return fmt.Errorf("%w stepSize: %.8f supplied %.8f",
				ErrAmountExceedsStep,
				m.AmountStepIncrementSize,
				amount)
		}
	}

	// ContractMultiplier checking not done due to the fact we need coherence with the
	// last average price (TODO)
	// m.multiplierUp will be used to determine how far our price can go up
	// m.multiplierDown will be used to determine how far our price can go down
	// m.averagePriceMinutes will be used to determine mean over this period

	// Max iceberg parts checking not done as we do not have that
	// functionality yet (TODO)
	// m.maxIcebergParts // How many components in an iceberg order

	// Max total orders not done due to order manager limitations (TODO)
	// m.maxTotalOrders

	// Max algo orders not done due to order manager limitations (TODO)
	// m.maxAlgoOrders

	// If order type is Market we do not need to do price checks
	if orderType != order.Market {
		if m.MinPrice != 0 && price < m.MinPrice {
			return fmt.Errorf("%w min: %.8f supplied %.8f",
				ErrPriceBelowMin,
				m.MinPrice,
				price)
		}
		if m.MaxPrice != 0 && price > m.MaxPrice {
			return fmt.Errorf("%w max: %.8f supplied %.8f",
				ErrPriceExceedsMax,
				m.MaxPrice,
				price)
		}
		if m.MinNotional != 0 && (amount*price) < m.MinNotional {
			return fmt.Errorf("%w minimum notional: %.8f value of order %.8f",
				ErrNotionalValue,
				m.MinNotional,
				amount*price)
		}
		if m.PriceStepIncrementSize != 0 {
			dPrice := decimal.NewFromFloat(price)
			dMinPrice := decimal.NewFromFloat(m.MinPrice)
			dStep := decimal.NewFromFloat(m.PriceStepIncrementSize)
			if !dPrice.Sub(dMinPrice).Mod(dStep).IsZero() {
				return fmt.Errorf("%w stepSize: %.8f supplied %.8f",
					ErrPriceExceedsStep,
					m.PriceStepIncrementSize,
					price)
			}
		}
		return nil
	}

	if m.MarketMinQty != 0 &&
		m.MinimumBaseAmount < m.MarketMinQty &&
		amount < m.MarketMinQty {
		return fmt.Errorf("%w min: %.8f supplied %.8f",
			ErrMarketAmountBelowMin,
			m.MarketMinQty,
			amount)
	}
	if m.MarketMaxQty != 0 &&
		m.MaximumBaseAmount > m.MarketMaxQty &&
		amount > m.MarketMaxQty {
		return fmt.Errorf("%w max: %.8f supplied %.8f",
			ErrMarketAmountExceedsMax,
			m.MarketMaxQty,
			amount)
	}
	if m.MarketStepIncrementSize != 0 &&
		m.AmountStepIncrementSize != m.MarketStepIncrementSize {
		dAmount := decimal.NewFromFloat(amount)
		dMinMAmount := decimal.NewFromFloat(m.MarketMinQty)
		dStep := decimal.NewFromFloat(m.MarketStepIncrementSize)
		if !dAmount.Sub(dMinMAmount).Mod(dStep).IsZero() {
			return fmt.Errorf("%w stepSize: %.8f supplied %.8f",
				ErrMarketAmountExceedsStep,
				m.MarketStepIncrementSize,
				amount)
		}
	}
	return nil
}

// ConformToDecimalAmount (POC) conforms amount to its amount interval
func (m *MinMaxLevel) ConformToDecimalAmount(amount decimal.Decimal) decimal.Decimal {
	if m == nil {
		return amount
	}

	dStep := decimal.NewFromFloat(m.AmountStepIncrementSize)
	if dStep.IsZero() || amount.Equal(dStep) {
		return amount
	}

	if amount.LessThan(dStep) {
		return decimal.Zero
	}
	mod := amount.Mod(dStep)
	// subtract modulus to get the floor
	return amount.Sub(mod)
}

// ConformToAmount (POC) conforms amount to its amount interval
func (m *MinMaxLevel) ConformToAmount(amount float64) float64 {
	if m == nil {
		return amount
	}

	if m.AmountStepIncrementSize == 0 || amount == m.AmountStepIncrementSize {
		return amount
	}

	if amount < m.AmountStepIncrementSize {
		return 0
	}

	// Convert floats to decimal types
	dAmount := decimal.NewFromFloat(amount)
	dStep := decimal.NewFromFloat(m.AmountStepIncrementSize)
	// derive modulus
	mod := dAmount.Mod(dStep)
	// subtract modulus to get the floor
	return dAmount.Sub(mod).InexactFloat64()
}

func (m *MinMaxLevel) ConformToFuturesAmountWithFallBack(amount float64) float64 {
	return m.ConformToAmount(amount)
}

// ConformToMarketAmountWithFallback (POC) conforms amount to its amount interval
func (m *MinMaxLevel) ConformToMarketAmountWithFallback(amount float64) float64 {
	if m == nil {
		return amount
	}

	if m.MarketStepIncrementSize == 0 {
		return m.ConformToAmount(amount)
	}
	if amount == m.MarketStepIncrementSize {
		return amount
	}

	if amount < m.MarketStepIncrementSize {
		return 0
	}

	// Convert floats to decimal types
	dAmount := decimal.NewFromFloat(amount)
	dStep := decimal.NewFromFloat(m.MarketStepIncrementSize)
	// derive modulus
	mod := dAmount.Mod(dStep)
	// subtract modulus to get the floor
	return dAmount.Sub(mod).InexactFloat64()
}

// ConformToPrice (POC) conforms amount to its amount interval
func (m *MinMaxLevel) ConformToPrice(price float64) float64 {
	if m == nil {
		return price
	}

	if m.PriceStepIncrementSize == 0 {
		return price
	}

	if price < m.PriceStepIncrementSize {
		return 0
	}

	// Convert floats to decimal types
	dAmount := decimal.NewFromFloat(price)
	dStep := decimal.NewFromFloat(m.PriceStepIncrementSize)
	// derive modulus
	mod := dAmount.Mod(dStep)
	// subtract modulus to get the floor
	return dAmount.Sub(mod).InexactFloat64()
}
