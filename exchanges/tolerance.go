package exchange

import (
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	// ErrExchangeToleranceNotLoaded defines if an exchange does not have minmax
	// values
	ErrExchangeToleranceNotLoaded = errors.New("exchange tolerances not loaded")
	// ErrPriceExceedsMin is when the price is lower than the minimum price
	// tolerance accepted by the exchange
	ErrPriceExceedsMin = errors.New("price exceeds minimum tolerance")
	// ErrPriceExceedsMax is when the price is higher than the maximum price
	// tolerance accepted by the exchange
	ErrPriceExceedsMax = errors.New("price exceeds maximum tolerance")
	// ErrPriceExceedsStep is when the price is not divisable by its step
	ErrPriceExceedsStep = errors.New("price exceeds step tolerance")
	// ErrAmountExceedsMin is when the amount is lower than the minimum amount
	// tolerance accepted by the exchange
	ErrAmountExceedsMin = errors.New("amount exceeds minimum tolerance")
	// ErrAmountExceedsMax is when the amount is highger than the maxiumum amount
	// tolerance accepted by the exchange
	ErrAmountExceedsMax = errors.New("amount exceeds maximum tolerance")
	// ErrAmountExceedsStep is when the amount is not divisable by its step
	ErrAmountExceedsStep = errors.New("amount exceeds step tolerance")
	// ErrNotionalValue is when the notional value does not exceed currency pair
	// requirements
	ErrNotionalValue = errors.New("total notional value is under minimum tolerance")

	errAmountDoesNotConform        = errors.New("amount exceeds min/max parameters")
	errCannotValidateAsset         = errors.New("cannot check tolerance asset not loaded")
	errCannotValidateBaseCurrency  = errors.New("cannot check tolerance base currency not loaded")
	errCannotValidateQuoteCurrency = errors.New("cannot check tolerance quote currency not loaded")
	errExchangeToleranceAsset      = errors.New("exchange tolerances not found for asset")
	errExchangeToleranceBase       = errors.New("exchange tolerances not found for base currency")
	errExchangeToleranceQuote      = errors.New("exchange tolerances not found for quote currency")
	errCannotLoadTolerance         = errors.New("cannot load tolerance levels not supplied")
	errInvalidPriceLevels          = errors.New("invalid price levels cannot load tolerances")
	errInvalidAmountLevels         = errors.New("invalid amount levels cannot load tolerances")
)

// ExecutionTolerance defines minimum and maximum values in relation to
// order size, order pricing, total notional values, total maximum orders etc
// for execution on an exchange.
type ExecutionTolerance struct {
	m map[asset.Item]map[currency.Code]map[currency.Code]*Tolerance
	sync.Mutex
}

// MinMaxLevel defines the minimum and maximum parameters for a currency pair
// for outbound exchange execution
type MinMaxLevel struct {
	Pair             currency.Pair
	Asset            asset.Item
	MinPrice         float64
	MaxPrice         float64
	StepPrice        float64
	MultiplierUp     float64
	MultiplierDown   float64
	AveragePriceMins float64
	MinAmount        float64
	MaxAmount        float64
	StepAmount       float64
	MinNotional      float64
	MaxIcebergeParts int64
	MarketMinimumQty float64
	MarketMaxQty     float64
	MarketStepSize   float64
	MaxTotalOrders   int64
	MaxAlgoOrders    int64
}

// LoadTolerances loads all tolerances levels into memory
func (e *ExecutionTolerance) LoadTolerances(levels []MinMaxLevel) error {
	if len(levels) == 0 {
		return errCannotLoadTolerance
	}
	e.Lock()
	defer e.Unlock()
	if e.m == nil {
		e.m = make(map[asset.Item]map[currency.Code]map[currency.Code]*Tolerance)
	}

	for x := range levels {
		assets, ok := e.m[levels[x].Asset]
		if !ok {
			assets = make(map[currency.Code]map[currency.Code]*Tolerance)
			e.m[levels[x].Asset] = assets
		}

		pairs, ok := assets[levels[x].Pair.Base]
		if !ok {
			pairs = make(map[currency.Code]*Tolerance)
			assets[levels[x].Pair.Base] = pairs
		}

		t, ok := pairs[levels[x].Pair.Quote]
		if !ok {
			t = new(Tolerance)
			pairs[levels[x].Pair.Quote] = t
		}

		if levels[x].MinPrice >= levels[x].MaxPrice {
			return fmt.Errorf("%w for %s %s supplied min: %f max: %f",
				errInvalidPriceLevels,
				levels[x].Asset,
				levels[x].Pair,
				levels[x].MinPrice,
				levels[x].MaxPrice)
		}

		if levels[x].MinAmount >= levels[x].MaxAmount {
			return fmt.Errorf("%w for %s %s supplied min: %f max: %f",
				errInvalidAmountLevels,
				levels[x].Asset,
				levels[x].Pair,
				levels[x].MinAmount,
				levels[x].MaxAmount)
		}

		t.minPrice = levels[x].MinPrice
		t.maxPrice = levels[x].MaxPrice
		t.stepSizePrice = levels[x].StepPrice
		t.minAmount = levels[x].MinAmount
		t.maxAmount = levels[x].MaxAmount
		t.stepSizeAmount = levels[x].StepAmount
		t.minNotional = levels[x].MinNotional
	}
	return nil
}

// GetTolerance returns the exchange tolerance parameters for a currency
func (e *ExecutionTolerance) GetTolerance(a asset.Item, cp currency.Pair) (*Tolerance, error) {
	e.Lock()
	defer e.Unlock()

	if e.m == nil {
		return nil, ErrExchangeToleranceNotLoaded
	}

	assets, ok := e.m[a]
	if !ok {
		return nil, errExchangeToleranceAsset
	}

	pairs, ok := assets[cp.Base]
	if !ok {
		return nil, errExchangeToleranceBase
	}

	t, ok := pairs[cp.Quote]
	if !ok {
		return nil, errExchangeToleranceQuote
	}

	return t, nil
}

// CheckTolerance checks to see if the price and amount conforms with exchange
// level order execution tolerances
func (e *ExecutionTolerance) CheckTolerance(a asset.Item, cp currency.Pair, price, amount float64) error {
	e.Lock()
	defer e.Unlock()

	if e.m == nil {
		// No exchange tolerances loaded so we can nil this
		return nil
	}

	assets, ok := e.m[a]
	if !ok {
		return errCannotValidateAsset
	}

	pairs, ok := assets[cp.Base]
	if !ok {
		return errCannotValidateBaseCurrency
	}

	t, ok := pairs[cp.Quote]
	if !ok {
		return errCannotValidateQuoteCurrency
	}

	err := t.Conforms(price, amount)
	if err != nil {
		return fmt.Errorf("%w for %s %s", err, a, cp)
	}

	return nil
}

// Tolerance defines total limit values for an associated currency to be checked
// before execution on an exchange
type Tolerance struct {
	minPrice       float64
	maxPrice       float64
	stepSizePrice  float64
	minAmount      float64
	maxAmount      float64
	stepSizeAmount float64
	minNotional    float64
	sync.Mutex
}

// Conforms checks outbound parameters
func (t *Tolerance) Conforms(price, amount float64) error {
	if t == nil {
		// For when we return a nil pointer we can assume there's nothing to
		// check
		return nil
	}

	t.Lock()
	defer t.Unlock()
	if t.minPrice != 0 && price < t.minPrice {
		return fmt.Errorf("%w min: %f suppplied %f",
			ErrPriceExceedsMin,
			t.minPrice,
			price)
	}
	if t.maxPrice != 0 && price > t.maxPrice {
		return fmt.Errorf("%w max: %f suppplied %f",
			ErrPriceExceedsMax,
			t.maxPrice,
			price)
	}

	if t.stepSizePrice != 0 {
		increase := 1 / t.stepSizePrice
		if math.Mod(price*increase, t.stepSizePrice*increase) != 0 {
			return fmt.Errorf("%w stepSize: %f suppplied %f",
				ErrPriceExceedsStep,
				t.stepSizePrice,
				price)
		}
	}

	if t.minAmount != 0 && amount < t.minAmount {
		return fmt.Errorf("%w min: %f suppplied %f",
			ErrAmountExceedsMin,
			t.minAmount,
			price)
	}

	if t.maxAmount != 0 && amount > t.maxAmount {
		return fmt.Errorf("%w min: %f suppplied %f",
			ErrAmountExceedsMax,
			t.maxAmount,
			price)
	}

	if t.stepSizeAmount != 0 {
		increase := 1 / t.stepSizeAmount
		if math.Mod(amount*increase, t.stepSizeAmount*increase) != 0 {
			return fmt.Errorf("%w stepSize: %f suppplied %f",
				ErrAmountExceedsStep,
				t.stepSizeAmount,
				amount)
		}
	}

	if t.minNotional != 0 && (amount*price) < t.minNotional {
		return fmt.Errorf("%w minimum notional: %f value of order %f",
			ErrNotionalValue,
			t.minNotional,
			amount*price)
	}

	return nil
}

// ConformToAmount (POC) conforms amount to its amount interval (Warning: this
// has a chance to increase position sizing to conform to step size amount)
// TODO: Add in decimal package
func (t *Tolerance) ConformToAmount(amount float64) float64 {
	t.Lock()
	defer t.Unlock()
	if t.stepSizeAmount == 0 {
		return amount
	}
	increase := 1 / t.stepSizeAmount
	// math round used because we don't want miss precision the downside to this
	// is that it will increase position size due to rounding issues.
	return math.Round(amount*increase) / increase
}
