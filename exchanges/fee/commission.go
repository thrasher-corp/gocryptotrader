package fee

import (
	"fmt"
	"math"
	"sync"

	"github.com/shopspring/decimal"
)

// defaultPercentageRateThreshold defines an upper bounds on current percentage
// rates to filter out any abnormal or incorrectly inputted percentage rates.
// This is currently set to 15% which is astronomically high compared to
// the general exchange mean commissions.
var defaultPercentageRateThreshold = 0.15

// Commission defines a trading fee structure snapshot
type Commission struct {
	// isFixedAmount defines if the value is a set amount (15 USD) rather than a
	// percentage e.g. 0.8% == 0.008.
	IsFixedAmount bool
	// Maker defines the fee when you provide liquidity for the orderbooks
	Maker float64
	// Taker defines the fee when you remove liquidity for the orderbooks
	Taker float64
	// WorstCaseMaker defines the worst case fee when you provide liquidity for
	// the orderbooks
	WorstCaseMaker float64
	// WorstCaseTaker defines the worst case fee when you remove liquidity for
	// the orderbooks
	WorstCaseTaker float64
}

// convert returns an internal commission rate type
func (c Commission) convert() *CommissionInternal {
	// If worst case scenario variables have not be assigned this defaults them
	// to maker and taker. Reduces specific loading code on the exchange wrapper
	// side.
	var wcm = decimal.NewFromFloat(c.WorstCaseMaker)
	if wcm.IsZero() {
		wcm = decimal.NewFromFloat(c.Maker)
	}
	var wct = decimal.NewFromFloat(c.WorstCaseTaker)
	if wct.IsZero() {
		wct = decimal.NewFromFloat(c.Taker)
	}
	return &CommissionInternal{
		isFixedAmount:  c.IsFixedAmount,
		maker:          decimal.NewFromFloat(c.Maker),
		taker:          decimal.NewFromFloat(c.Taker),
		worstCaseMaker: wcm,
		worstCaseTaker: wct,
	}
}

// validate validates commission variables
func (c Commission) validate() error {
	// In all instances providing liquidity (maker) has a lower fees compared to
	// taking liquidity (taker).
	if c.Maker > c.Taker {
		return errMakerBiggerThanTaker
	}

	if !c.IsFixedAmount {
		// Abs so we check threshold levels in positive and negative direction.
		if math.Abs(c.Maker) >= defaultPercentageRateThreshold {
			return fmt.Errorf("%w exceeds percentage rate threshold %f",
				errMakerInvalid,
				defaultPercentageRateThreshold)
		}
		if math.Abs(c.Taker) >= defaultPercentageRateThreshold {
			return fmt.Errorf("%w exceeds percentage rate threshold %f",
				errTakerInvalid,
				defaultPercentageRateThreshold)
		}
	}

	return nil
}

// CommissionInternal defines a trading fee structure for internal tracking
type CommissionInternal struct {
	// isFixedAmount defines if the value is a fixed amount (15 USD) rather than
	// a percentage e.g. 0.8% == 0.008.
	isFixedAmount bool
	// Maker defines the fee when you provide liquidity for the orderbooks
	maker decimal.Decimal
	// Taker defines the fee when you remove liquidity for the orderbooks
	taker decimal.Decimal
	// WorstCaseMaker defines the worst case fee when you provide liquidity for
	// the orderbooks
	worstCaseMaker decimal.Decimal
	// WorstCaseTaker defines the worst case fee when you remove liquidity for
	// the orderbooks
	worstCaseTaker decimal.Decimal

	mtx sync.Mutex // protected so this can be exported for external strategies
}

// convert returns a friendly package exportable type
func (c *CommissionInternal) convert() Commission {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return Commission{
		IsFixedAmount:  c.isFixedAmount,
		Maker:          c.maker.InexactFloat64(),
		Taker:          c.taker.InexactFloat64(),
		WorstCaseMaker: c.worstCaseMaker.InexactFloat64(),
		WorstCaseTaker: c.worstCaseTaker.InexactFloat64(),
	}
}

// CalculateMaker returns the calculated maker fees
func (c *CommissionInternal) CalculateMaker(price, amount float64) (float64, error) {
	if price == 0 {
		return 0, errPriceIsZero
	}
	if amount == 0 {
		return 0, errAmountIsZero
	}

	c.mtx.Lock()
	fee := c.maker
	isFixedAmount := c.isFixedAmount
	c.mtx.Unlock()
	return calculate(isFixedAmount, fee, price, amount)
}

// CalculateTaker returns the calculated taker fees
func (c *CommissionInternal) CalculateTaker(price, amount float64) (float64, error) {
	if price == 0 {
		return 0, errPriceIsZero
	}
	if amount == 0 {
		return 0, errAmountIsZero
	}

	c.mtx.Lock()
	fee := c.taker
	isFixedAmount := c.isFixedAmount
	c.mtx.Unlock()
	return calculate(isFixedAmount, fee, price, amount)
}

// CalculateWorstCaseMaker returns the worst-case calculated maker fees
func (c *CommissionInternal) CalculateWorstCaseMaker(price, amount float64) (float64, error) {
	if price == 0 {
		return 0, errPriceIsZero
	}
	if amount == 0 {
		return 0, errAmountIsZero
	}

	c.mtx.Lock()
	fee := c.worstCaseMaker
	isFixedAmount := c.isFixedAmount
	c.mtx.Unlock()
	return calculate(isFixedAmount, fee, price, amount)
}

// CalculateWorstCaseTaker returns the worst-case calculated taker fees
func (c *CommissionInternal) CalculateWorstCaseTaker(price, amount float64) (float64, error) {
	if price == 0 {
		return 0, errPriceIsZero
	}
	if amount == 0 {
		return 0, errAmountIsZero
	}

	c.mtx.Lock()
	fee := c.worstCaseTaker
	isFixedAmount := c.isFixedAmount
	c.mtx.Unlock()
	return calculate(isFixedAmount, fee, price, amount)
}

// GetMaker returns the maker fee and type
func (c *CommissionInternal) GetMaker() (fee float64, isFixedAmount bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return c.maker.InexactFloat64(), c.isFixedAmount
}

// GetTaker returns the taker fee and type
func (c *CommissionInternal) GetTaker() (fee float64, isFixedAmount bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return c.taker.InexactFloat64(), c.isFixedAmount
}

// GetWorstCaseMaker returns the worst-case maker fee and type
func (c *CommissionInternal) GetWorstCaseMaker() (fee float64, isFixedAmount bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return c.worstCaseMaker.InexactFloat64(), c.isFixedAmount
}

// GetWorstCaseTaker returns the worst-case taker fee and type
func (c *CommissionInternal) GetWorstCaseTaker() (fee float64, isFixedAmount bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return c.worstCaseTaker.InexactFloat64(), c.isFixedAmount
}

// set sets the commission values for update
func (c *CommissionInternal) set(maker, taker float64, isFixedAmount bool) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.isFixedAmount != isFixedAmount {
		// These should not change, and a package update might need to occur.
		return errFeeTypeMismatch
	}
	c.maker = decimal.NewFromFloat(maker)
	c.taker = decimal.NewFromFloat(taker)
	return nil
}

// calculate returns the commission fee total based on internal loaded values
func calculate(isFixedAmount bool, fee decimal.Decimal, price, amount float64) (float64, error) {
	// TODO: Add fees based on volume of this asset
	if isFixedAmount {
		// Returns the whole number
		return fee.InexactFloat64(), nil
	}
	// TODO: Add rebate for negative values
	var val = decimal.NewFromFloat(price).Mul(decimal.NewFromFloat(amount)).Mul(fee)
	return val.InexactFloat64(), nil
}

// load protected loader for maker and taker fee updates
func (c *CommissionInternal) load(maker, taker float64) {
	c.mtx.Lock()
	c.maker = decimal.NewFromFloat(maker)
	if c.worstCaseMaker.Equal(decimal.Zero) {
		c.worstCaseMaker = c.maker
	}
	c.taker = decimal.NewFromFloat(taker)
	if c.worstCaseTaker.Equal(decimal.Zero) {
		c.worstCaseTaker = c.maker
	}
	c.mtx.Unlock()
}
