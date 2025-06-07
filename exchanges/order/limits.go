package order

import (
	"errors"
	"fmt"
	"sync"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Public errors for order limits
var (
	ErrLoadLimitsFailed            = errors.New("failed to load exchange limits")
	ErrExchangeLimitNotLoaded      = errors.New("exchange limits not loaded")
	ErrPriceBelowMin               = errors.New("price below minimum limit")
	ErrPriceExceedsMax             = errors.New("price exceeds maximum limit")
	ErrPriceExceedsStep            = errors.New("price exceeds step limit") // price is not divisible by its step
	ErrAmountBelowMin              = errors.New("amount below minimum limit")
	ErrAmountExceedsMax            = errors.New("amount exceeds maximum limit")
	ErrAmountExceedsStep           = errors.New("amount exceeds step limit") // amount is not divisible by its step
	ErrNotionalValue               = errors.New("total notional value is under minimum limit")
	ErrMarketAmountBelowMin        = errors.New("market order amount below minimum limit")
	ErrMarketAmountExceedsMax      = errors.New("market order amount exceeds maximum limit")
	ErrMarketAmountExceedsStep     = errors.New("market order amount exceeds step limit") // amount is not divisible by its step for a market order
	ErrCannotValidateAsset         = errors.New("cannot check limit, asset not loaded")
	ErrCannotValidateBaseCurrency  = errors.New("cannot check limit, base currency not loaded")
	ErrCannotValidateQuoteCurrency = errors.New("cannot check limit, quote currency not loaded")
)

var (
	errExchangeLimitBase   = errors.New("exchange limits not found for base currency")
	errExchangeLimitQuote  = errors.New("exchange limits not found for quote currency")
	errCannotLoadLimit     = errors.New("cannot load limit, levels not supplied")
	errInvalidPriceLevels  = errors.New("invalid price levels, cannot load limits")
	errInvalidAmountLevels = errors.New("invalid amount levels, cannot load limits")
	errInvalidQuoteLevels  = errors.New("invalid quote levels, cannot load limits")
)

// ExecutionLimits defines minimum and maximum values in relation to
// order size, order pricing, total notional values, total maximum orders etc
// for execution on an exchange.
type ExecutionLimits struct {
	m   map[asset.Item]map[*currency.Item]map[*currency.Item]MinMaxLevel
	mtx sync.RWMutex
}

// MinMaxLevel defines the minimum and maximum parameters for a currency pair
// for outbound exchange execution
type MinMaxLevel struct {
	Pair                    currency.Pair
	Asset                   asset.Item
	MinPrice                float64
	MaxPrice                float64
	PriceStepIncrementSize  float64
	MultiplierUp            float64
	MultiplierDown          float64
	MultiplierDecimal       float64
	AveragePriceMinutes     int64
	MinimumBaseAmount       float64
	MaximumBaseAmount       float64
	MinimumQuoteAmount      float64
	MaximumQuoteAmount      float64
	AmountStepIncrementSize float64
	QuoteStepIncrementSize  float64
	MinNotional             float64
	MaxIcebergParts         int64
	MarketMinQty            float64
	MarketMaxQty            float64
	MarketStepIncrementSize float64
	MaxTotalOrders          int64
	MaxAlgoOrders           int64
}

// LoadLimits loads all limits levels into memory
func (e *ExecutionLimits) LoadLimits(levels []MinMaxLevel) error {
	if len(levels) == 0 {
		return errCannotLoadLimit
	}
	e.mtx.Lock()
	defer e.mtx.Unlock()
	if e.m == nil {
		e.m = make(map[asset.Item]map[*currency.Item]map[*currency.Item]MinMaxLevel)
	}

	for x := range levels {
		if !levels[x].Asset.IsValid() {
			return fmt.Errorf("cannot load levels for %q: %w", levels[x].Asset, asset.ErrNotSupported)
		}
		m1, ok := e.m[levels[x].Asset]
		if !ok {
			m1 = make(map[*currency.Item]map[*currency.Item]MinMaxLevel)
			e.m[levels[x].Asset] = m1
		}

		if levels[x].Pair.IsEmpty() {
			return currency.ErrCurrencyPairEmpty
		}

		m2, ok := m1[levels[x].Pair.Base.Item]
		if !ok {
			m2 = make(map[*currency.Item]MinMaxLevel)
			m1[levels[x].Pair.Base.Item] = m2
		}

		if levels[x].MinPrice > 0 &&
			levels[x].MaxPrice > 0 &&
			levels[x].MinPrice > levels[x].MaxPrice {
			return fmt.Errorf("%w for %s %s supplied min: %f max: %f",
				errInvalidPriceLevels,
				levels[x].Asset,
				levels[x].Pair,
				levels[x].MinPrice,
				levels[x].MaxPrice)
		}

		if levels[x].MinimumBaseAmount > 0 &&
			levels[x].MaximumBaseAmount > 0 &&
			levels[x].MinimumBaseAmount > levels[x].MaximumBaseAmount {
			return fmt.Errorf("%w for %s %s supplied min: %f max: %f",
				errInvalidAmountLevels,
				levels[x].Asset,
				levels[x].Pair,
				levels[x].MinimumBaseAmount,
				levels[x].MaximumBaseAmount)
		}

		if levels[x].MinimumQuoteAmount > 0 &&
			levels[x].MaximumQuoteAmount > 0 &&
			levels[x].MinimumQuoteAmount > levels[x].MaximumQuoteAmount {
			return fmt.Errorf("%w for %s %s supplied min: %f max: %f",
				errInvalidQuoteLevels,
				levels[x].Asset,
				levels[x].Pair,
				levels[x].MinimumQuoteAmount,
				levels[x].MaximumQuoteAmount)
		}

		m2[levels[x].Pair.Quote.Item] = levels[x]
	}
	return nil
}

// GetOrderExecutionLimits returns the exchange limit parameters for a currency
func (e *ExecutionLimits) GetOrderExecutionLimits(a asset.Item, cp currency.Pair) (MinMaxLevel, error) {
	e.mtx.RLock()
	defer e.mtx.RUnlock()
	if e.m == nil {
		return MinMaxLevel{}, ErrExchangeLimitNotLoaded
	}

	m1, ok := e.m[a]
	if !ok {
		return MinMaxLevel{}, fmt.Errorf("%w %v", ErrCannotValidateAsset, a)
	}

	m2, ok := m1[cp.Base.Item]
	if !ok {
		return MinMaxLevel{}, fmt.Errorf("%w %v", errExchangeLimitBase, cp.Base)
	}

	limit, ok := m2[cp.Quote.Item]
	if !ok {
		return MinMaxLevel{}, fmt.Errorf("%w %v", errExchangeLimitQuote, cp.Quote)
	}

	return limit, nil
}

// CheckOrderExecutionLimits checks to see if the price and amount conforms with
// exchange level order execution limits
func (e *ExecutionLimits) CheckOrderExecutionLimits(a asset.Item, cp currency.Pair, price, amount float64, orderType Type) error {
	e.mtx.RLock()
	defer e.mtx.RUnlock()

	if e.m == nil {
		// No exchange limits loaded so we can nil this
		return nil
	}

	m1, ok := e.m[a]
	if !ok {
		return ErrCannotValidateAsset
	}

	m2, ok := m1[cp.Base.Item]
	if !ok {
		return ErrCannotValidateBaseCurrency
	}

	limit, ok := m2[cp.Quote.Item]
	if !ok {
		return ErrCannotValidateQuoteCurrency
	}

	err := limit.Conforms(price, amount, orderType)
	if err != nil {
		return fmt.Errorf("%w for %s %s", err, a, cp)
	}

	return nil
}

// Conforms checks outbound parameters
func (m *MinMaxLevel) Conforms(price, amount float64, orderType Type) error {
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

	// Multiplier checking not done due to the fact we need coherence with the
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
	if orderType != Market {
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
