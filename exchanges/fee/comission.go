package fee

import (
	"sync"

	"github.com/shopspring/decimal"
)

// Commission defines a trading fee structure snapshot
type Commission struct {
	// IsSetAmount defines if the value is a set amount (15 USD) rather than a
	// percentage e.g. 0.8% == 0.008.
	IsSetAmount bool
	// Maker defines the fee when you provide liqudity for the orderbooks
	Maker float64
	// Taker defines the fee when you remove liqudity for the orderbooks
	Taker float64
	// WorstCaseMaker defines the worst case fee when you provide liqudity for
	// the orderbooks
	WorstCaseMaker float64
	// WorstCaseTaker defines the worst case fee when you remove liqudity for
	//the orderbooks
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
		setAmount:      c.IsSetAmount,
		maker:          decimal.NewFromFloat(c.Maker),
		taker:          decimal.NewFromFloat(c.Taker),
		worstCaseMaker: wcm,
		worstCaseTaker: wct,
	}
}

// validate validates commission variables
func (c Commission) validate() error {
	if c.Taker < 0 {
		return errTakerInvalid
	}
	if c.Maker > c.Taker {
		return errMakerBiggerThanTaker
	}
	return nil
}

// CommissionInternal defines a trading fee structure for internal tracking
type CommissionInternal struct {
	// SetAmount defines if the value is a set amount (15 USD) rather than a
	// percentage e.g. 0.8% == 0.008.
	setAmount bool
	// Maker defines the fee when you provide liqudity for the orderbooks
	maker decimal.Decimal
	// Taker defines the fee when you remove liqudity for the orderbooks
	taker decimal.Decimal
	// WorstCaseMaker defines the worst case fee when you provide liqudity for
	// the orderbooks
	worstCaseMaker decimal.Decimal
	// WorstCaseTaker defines the worst case fee when you remove liqudity for
	//the orderbooks
	worstCaseTaker decimal.Decimal

	mtx sync.Mutex // protected so this can be exported for external strategies
}

// convert returns a friendly package exportedable type
func (c *CommissionInternal) convert() Commission {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	maker, _ := c.maker.Float64()
	taker, _ := c.taker.Float64()
	worstCaseMaker, _ := c.worstCaseMaker.Float64()
	worstCaseTaker, _ := c.worstCaseTaker.Float64()
	return Commission{
		IsSetAmount:    c.setAmount,
		Maker:          maker,
		Taker:          taker,
		WorstCaseMaker: worstCaseMaker,
		WorstCaseTaker: worstCaseTaker,
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
	defer c.mtx.Unlock()
	return c.calculate(c.maker, price, amount)
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
	defer c.mtx.Unlock()
	return c.calculate(c.taker, price, amount)
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
	defer c.mtx.Unlock()
	return c.calculate(c.worstCaseMaker, price, amount)
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
	defer c.mtx.Unlock()
	return c.calculate(c.worstCaseTaker, price, amount)
}

// GetMaker returns the maker fee and type
func (c *CommissionInternal) GetMaker() (fee float64, isSetAmount bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	rVal, _ := c.maker.Float64()
	return rVal, c.setAmount
}

// GetTaker returns the taker fee and type
func (c *CommissionInternal) GetTaker() (fee float64, isSetAmount bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	rVal, _ := c.taker.Float64()
	return rVal, c.setAmount
}

// GetWorstCaseMaker returns the worst-case maker fee and type
func (c *CommissionInternal) GetWorstCaseMaker() (fee float64, isSetAmount bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	rVal, _ := c.worstCaseMaker.Float64()
	return rVal, c.setAmount
}

// GetWorstCaseTaker returns the worst-case taker fee and type
func (c *CommissionInternal) GetWorstCaseTaker() (fee float64, isSetAmount bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	rVal, _ := c.worstCaseTaker.Float64()
	return rVal, c.setAmount
}

// set sets the commision values for update
func (c *CommissionInternal) set(maker, taker float64, setAmount bool) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	// These should not change, and a package update might need to occur.
	if c.setAmount != setAmount {
		return errFeeTypeMismatch
	}
	c.maker = decimal.NewFromFloat(maker)
	c.taker = decimal.NewFromFloat(taker)
	return nil
}

// calculate returns the commission fee total based on internal loaded values
func (c *CommissionInternal) calculate(fee decimal.Decimal, price, amount float64) (float64, error) {
	// TODO: Add fees based on volume of this asset
	if c.setAmount {
		// Returns the whole number
		setValue, _ := fee.Float64()
		return setValue, nil
	}
	// Return fee derived from percentage, price and amount values
	// TODO: Add rebate for negative values
	var val = decimal.NewFromFloat(price).Mul(decimal.NewFromFloat(amount)).Mul(fee)
	rVal, _ := val.Float64()
	return rVal, nil
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
