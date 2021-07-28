package order

import (
	"errors"
	"fmt"
	"sync"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	// ErrExchangeLimitNotLoaded defines if an exchange does not have minmax
	// values
	ErrExchangeLimitNotLoaded = errors.New("exchange limits not loaded")
	// ErrPriceBelowMin is when the price is lower than the minimum price
	// limit accepted by the exchange
	ErrPriceBelowMin = errors.New("price below minimum limit")
	// ErrPriceExceedsMax is when the price is higher than the maximum price
	// limit accepted by the exchange
	ErrPriceExceedsMax = errors.New("price exceeds maximum limit")
	// ErrPriceExceedsStep is when the price is not divisible by its step
	ErrPriceExceedsStep = errors.New("price exceeds step limit")
	// ErrAmountBelowMin is when the amount is lower than the minimum amount
	// limit accepted by the exchange
	ErrAmountBelowMin = errors.New("amount below minimum limit")
	// ErrAmountExceedsMax is when the amount is higher than the maximum amount
	// limit accepted by the exchange
	ErrAmountExceedsMax = errors.New("amount exceeds maximum limit")
	// ErrAmountExceedsStep is when the amount is not divisible by its step
	ErrAmountExceedsStep = errors.New("amount exceeds step limit")
	// ErrNotionalValue is when the notional value does not exceed currency pair
	// requirements
	ErrNotionalValue = errors.New("total notional value is under minimum limit")
	// ErrMarketAmountBelowMin is when the amount is lower than the minimum
	// amount limit accepted by the exchange for a market order
	ErrMarketAmountBelowMin = errors.New("market order amount below minimum limit")
	// ErrMarketAmountExceedsMax is when the amount is higher than the maximum
	// amount limit accepted by the exchange for a market order
	ErrMarketAmountExceedsMax = errors.New("market order amount exceeds maximum limit")
	// ErrMarketAmountExceedsStep is when the amount is not divisible by its
	// step for a market order
	ErrMarketAmountExceedsStep = errors.New("market order amount exceeds step limit")

	errCannotValidateAsset         = errors.New("cannot check limit, asset not loaded")
	errCannotValidateBaseCurrency  = errors.New("cannot check limit, base currency not loaded")
	errCannotValidateQuoteCurrency = errors.New("cannot check limit, quote currency not loaded")
	errExchangeLimitAsset          = errors.New("exchange limits not found for asset")
	errExchangeLimitBase           = errors.New("exchange limits not found for base currency")
	errExchangeLimitQuote          = errors.New("exchange limits not found for quote currency")
	errCannotLoadLimit             = errors.New("cannot load limit, levels not supplied")
	errInvalidPriceLevels          = errors.New("invalid price levels, cannot load limits")
	errInvalidAmountLevels         = errors.New("invalid amount levels, cannot load limits")
)

// ExecutionLimits defines minimum and maximum values in relation to
// order size, order pricing, total notional values, total maximum orders etc
// for execution on an exchange.
type ExecutionLimits struct {
	m   map[asset.Item]map[*currency.Item]map[*currency.Item]*Limits
	mtx sync.RWMutex
}

// MinMaxLevel defines the minimum and maximum parameters for a currency pair
// for outbound exchange execution
type MinMaxLevel struct {
	Pair                currency.Pair
	Asset               asset.Item
	MinPrice            float64
	MaxPrice            float64
	StepPrice           float64
	MultiplierUp        float64
	MultiplierDown      float64
	MultiplierDecimal   float64
	AveragePriceMinutes int64
	MinAmount           float64
	MaxAmount           float64
	StepAmount          float64
	MinNotional         float64
	MaxIcebergParts     int64
	MarketMinQty        float64
	MarketMaxQty        float64
	MarketStepSize      float64
	MaxTotalOrders      int64
	MaxAlgoOrders       int64
}

// LoadLimits loads all limits levels into memory
func (e *ExecutionLimits) LoadLimits(levels []MinMaxLevel) error {
	if len(levels) == 0 {
		return errCannotLoadLimit
	}
	e.mtx.Lock()
	defer e.mtx.Unlock()
	if e.m == nil {
		e.m = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*Limits)
	}

	for x := range levels {
		if !levels[x].Asset.IsValid() {
			return fmt.Errorf("cannot load levels for '%s': %w",
				levels[x].Asset,
				asset.ErrNotSupported)
		}
		m1, ok := e.m[levels[x].Asset]
		if !ok {
			m1 = make(map[*currency.Item]map[*currency.Item]*Limits)
			e.m[levels[x].Asset] = m1
		}

		m2, ok := m1[levels[x].Pair.Base.Item]
		if !ok {
			m2 = make(map[*currency.Item]*Limits)
			m1[levels[x].Pair.Base.Item] = m2
		}

		limit, ok := m2[levels[x].Pair.Quote.Item]
		if !ok {
			limit = new(Limits)
			m2[levels[x].Pair.Quote.Item] = limit
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

		if levels[x].MinAmount > 0 &&
			levels[x].MaxAmount > 0 &&
			levels[x].MinAmount > levels[x].MaxAmount {
			return fmt.Errorf("%w for %s %s supplied min: %f max: %f",
				errInvalidAmountLevels,
				levels[x].Asset,
				levels[x].Pair,
				levels[x].MinAmount,
				levels[x].MaxAmount)
		}
		limit.m.Lock()
		limit.minPrice = levels[x].MinPrice
		limit.maxPrice = levels[x].MaxPrice
		limit.stepIncrementSizePrice = levels[x].StepPrice
		limit.minAmount = levels[x].MinAmount
		limit.maxAmount = levels[x].MaxAmount
		limit.stepIncrementSizeAmount = levels[x].StepAmount
		limit.minNotional = levels[x].MinNotional
		limit.multiplierUp = levels[x].MultiplierUp
		limit.multiplierDown = levels[x].MultiplierDown
		limit.averagePriceMinutes = levels[x].AveragePriceMinutes
		limit.maxIcebergParts = levels[x].MaxIcebergParts
		limit.marketMinQty = levels[x].MarketMinQty
		limit.marketMaxQty = levels[x].MarketMaxQty
		limit.marketStepIncrementSize = levels[x].MarketStepSize
		limit.maxTotalOrders = levels[x].MaxTotalOrders
		limit.maxAlgoOrders = levels[x].MaxAlgoOrders
		limit.m.Unlock()
	}
	return nil
}

// GetOrderExecutionLimits returns the exchange limit parameters for a currency
func (e *ExecutionLimits) GetOrderExecutionLimits(a asset.Item, cp currency.Pair) (*Limits, error) {
	e.mtx.RLock()
	defer e.mtx.RUnlock()

	if e.m == nil {
		return nil, ErrExchangeLimitNotLoaded
	}

	m1, ok := e.m[a]
	if !ok {
		return nil, errExchangeLimitAsset
	}

	m2, ok := m1[cp.Base.Item]
	if !ok {
		return nil, errExchangeLimitBase
	}

	limit, ok := m2[cp.Quote.Item]
	if !ok {
		return nil, errExchangeLimitQuote
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
		return errCannotValidateAsset
	}

	m2, ok := m1[cp.Base.Item]
	if !ok {
		return errCannotValidateBaseCurrency
	}

	limit, ok := m2[cp.Quote.Item]
	if !ok {
		return errCannotValidateQuoteCurrency
	}

	err := limit.Conforms(price, amount, orderType)
	if err != nil {
		return fmt.Errorf("%w for %s %s", err, a, cp)
	}

	return nil
}

// Limits defines total limit values for an associated currency to be checked
// before execution on an exchange
type Limits struct {
	minPrice                float64
	maxPrice                float64
	stepIncrementSizePrice  float64
	minAmount               float64
	maxAmount               float64
	stepIncrementSizeAmount float64
	minNotional             float64
	multiplierUp            float64
	multiplierDown          float64
	averagePriceMinutes     int64
	maxIcebergParts         int64
	marketMinQty            float64
	marketMaxQty            float64
	marketStepIncrementSize float64
	maxTotalOrders          int64
	maxAlgoOrders           int64
	m                       sync.RWMutex
}

// Conforms checks outbound parameters
func (l *Limits) Conforms(price, amount float64, orderType Type) error {
	if l == nil {
		// For when we return a nil pointer we can assume there's nothing to
		// check
		return nil
	}

	l.m.RLock()
	defer l.m.RUnlock()
	if l.minAmount != 0 && amount < l.minAmount {
		return fmt.Errorf("%w min: %.8f supplied %.8f",
			ErrAmountBelowMin,
			l.minAmount,
			amount)
	}
	if l.maxAmount != 0 && amount > l.maxAmount {
		return fmt.Errorf("%w min: %.8f supplied %.8f",
			ErrAmountExceedsMax,
			l.maxAmount,
			amount)
	}
	if l.stepIncrementSizeAmount != 0 {
		dAmount := decimal.NewFromFloat(amount)
		dMinAmount := decimal.NewFromFloat(l.minAmount)
		dStep := decimal.NewFromFloat(l.stepIncrementSizeAmount)
		if !dAmount.Sub(dMinAmount).Mod(dStep).IsZero() {
			return fmt.Errorf("%w stepSize: %.8f supplied %.8f",
				ErrAmountExceedsStep,
				l.stepIncrementSizeAmount,
				amount)
		}
	}

	// Multiplier checking not done due to the fact we need coherence with the
	// last average price (TODO)
	// l.multiplierUp will be used to determine how far our price can go up
	// l.multiplierDown will be used to determine how far our price can go down
	// l.averagePriceMinutes will be used to determine mean over this period

	// Max iceberg parts checking not done as we do not have that
	// functionality yet (TODO)
	// l.maxIcebergParts // How many components in an iceberg order

	// Max total orders not done due to order manager limitations (TODO)
	// l.maxTotalOrders

	// Max algo orders not done due to order manager limitations (TODO)
	// l.maxAlgoOrders

	// If order type is Market we do not need to do price checks
	if orderType != Market {
		if l.minPrice != 0 && price < l.minPrice {
			return fmt.Errorf("%w min: %.8f supplied %.8f",
				ErrPriceBelowMin,
				l.minPrice,
				price)
		}
		if l.maxPrice != 0 && price > l.maxPrice {
			return fmt.Errorf("%w max: %.8f supplied %.8f",
				ErrPriceExceedsMax,
				l.maxPrice,
				price)
		}
		if l.minNotional != 0 && (amount*price) < l.minNotional {
			return fmt.Errorf("%w minimum notional: %.8f value of order %.8f",
				ErrNotionalValue,
				l.minNotional,
				amount*price)
		}
		if l.stepIncrementSizePrice != 0 {
			dPrice := decimal.NewFromFloat(price)
			dMinPrice := decimal.NewFromFloat(l.minPrice)
			dStep := decimal.NewFromFloat(l.stepIncrementSizePrice)
			if !dPrice.Sub(dMinPrice).Mod(dStep).IsZero() {
				return fmt.Errorf("%w stepSize: %.8f supplied %.8f",
					ErrPriceExceedsStep,
					l.stepIncrementSizePrice,
					price)
			}
		}
		return nil
	}

	if l.marketMinQty != 0 &&
		l.minAmount < l.marketMinQty &&
		amount < l.marketMinQty {
		return fmt.Errorf("%w min: %.8f supplied %.8f",
			ErrMarketAmountBelowMin,
			l.marketMinQty,
			amount)
	}
	if l.marketMaxQty != 0 &&
		l.maxAmount > l.marketMaxQty &&
		amount > l.marketMaxQty {
		return fmt.Errorf("%w max: %.8f supplied %.8f",
			ErrMarketAmountExceedsMax,
			l.marketMaxQty,
			amount)
	}
	if l.marketStepIncrementSize != 0 && l.stepIncrementSizeAmount != l.marketStepIncrementSize {
		dAmount := decimal.NewFromFloat(amount)
		dMinMAmount := decimal.NewFromFloat(l.marketMinQty)
		dStep := decimal.NewFromFloat(l.marketStepIncrementSize)
		if !dAmount.Sub(dMinMAmount).Mod(dStep).IsZero() {
			return fmt.Errorf("%w stepSize: %.8f supplied %.8f",
				ErrMarketAmountExceedsStep,
				l.marketStepIncrementSize,
				amount)
		}
	}
	return nil
}

// ConformToAmount (POC) conforms amount to its amount interval
func (l *Limits) ConformToAmount(amount float64) float64 {
	if l == nil {
		// For when we return a nil pointer we can assume there's nothing to
		// check
		return amount
	}
	l.m.Lock()
	defer l.m.Unlock()
	if l.stepIncrementSizeAmount == 0 || amount == l.stepIncrementSizeAmount {
		return amount
	}

	if amount < l.stepIncrementSizeAmount {
		return 0
	}

	// Convert floats to decimal types
	dAmount := decimal.NewFromFloat(amount)
	dStep := decimal.NewFromFloat(l.stepIncrementSizeAmount)
	// derive modulus
	mod := dAmount.Mod(dStep)
	// subtract modulus to get the floor
	rVal := dAmount.Sub(mod)
	fVal, _ := rVal.Float64()
	return fVal
}
