package account

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// Claim is a type representing the claim on current amount in holdings, this
// will be utilised for withdrawals and trading activity between multiple
// strategies. This allows us to make sure, we have the funds available and we
// cannot execute a double spend which will result in exchange errors.
type Claim struct {
	// amount is the successfully claimed amount requested
	amount decimal.Decimal
	// h is the pointer to the holding for releasing this claim when finished
	h *Holding
	// t is the time at which the claim was successfully called
	t time.Time

	ident
	m sync.Mutex
}

// GetAmount returns the amount that has been claimed as a float64
func (c *Claim) GetAmount() float64 {
	c.m.Lock()
	defer c.m.Unlock()
	amt, _ := c.amount.Float64()
	return amt
}

// getAmount returns the amount as a decimal for internal use
// (Warning not protected)
func (c *Claim) getAmount() decimal.Decimal {
	return c.amount
}

// GetTime returns the time at which the claim was successfully called
func (c *Claim) GetTime() time.Time {
	c.m.Lock()
	defer c.m.Unlock()
	return c.t
}

// Release is when an order fails to execute or funds cannot be withdrawn, this
// will releases the funds back to holdings for further use.
func (c *Claim) Release() error {
	c.m.Lock()
	defer c.m.Unlock()
	return c.h.Release(c)
}

// ReleaseToPending is used when an order or withdrawal has been been submitted,
// this hands over funds to a pending bucket for account settlement, change of
// state will release these from pending.
func (c *Claim) ReleaseToPending() error {
	c.m.Lock()
	defer c.m.Unlock()
	return c.h.ReleaseToPending(c)
}

// ReleaseAndReduce this pending claim and reduce this amount and total holdings
// manually.
func (c *Claim) ReleaseAndReduce() error {
	c.m.Lock()
	defer c.m.Unlock()
	return c.h.reduce(c)
}

// HasClaim determines if a claim is still on an amount on a holding
func (c *Claim) HasClaim() bool {
	c.m.Lock()
	defer c.m.Unlock()
	return c.h.CheckClaim(c)
}
