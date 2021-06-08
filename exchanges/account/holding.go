package account

import (
	"errors"
	"sync"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	errAmountExceedsHoldings = errors.New("amount exceeds current free amount")
	errUnableToReleaseClaim  = errors.New("unable to release claim, holding amounts may be locked")
	errNoBalance             = errors.New("no balance found for currency")
	errUnableToReduceClaim   = errors.New("unable to reduce claim, claim not found")
)

// Holding defines the total currency holdings for an account and what is
// currently in use.
type Holding struct {
	// Exchange side levels. Warning: This has some observational dilemma.
	total  decimal.Decimal
	locked decimal.Decimal
	// --------- These will only be altered by the exchange updates --------- //

	// free is the current free amount (total - (locked + claims + pending)).
	free decimal.Decimal

	// claims is the list of current internal claims on current liquidity.
	claims []*Claim

	// pending is a bucket for when we execute an order and liquidity is
	// potentially taken off the exchange but we cannot release the claim amount
	// to free until we can match it by an exchange update. This should reduce
	// when the total amount reduces.
	pending decimal.Decimal

	verbose bool
	m       sync.Mutex
}

// GetTotal returns the current total holdings
func (h *Holding) GetTotal() float64 {
	h.m.Lock()
	total, _ := h.total.Float64()
	h.m.Unlock()
	return total
}

// GetLocked returns the current locked holdings
func (h *Holding) GetLocked() float64 {
	h.m.Lock()
	locked, _ := h.locked.Float64()
	h.m.Unlock()
	return locked
}

// GetPending returns the current pending holdings
func (h *Holding) GetPending() float64 {
	h.m.Lock()
	pending, _ := h.pending.Float64()
	h.m.Unlock()
	return pending
}

// GetFree returns the current free holdings
func (h *Holding) GetFree() float64 {
	h.m.Lock()
	free, _ := h.free.Float64()
	h.m.Unlock()
	return free
}

// GetTotalClaims returns the total claims amount
func (h *Holding) GetTotalClaims() float64 {
	var total decimal.Decimal
	h.m.Lock()
	for x := range h.claims {
		total = total.Add(h.claims[x].getAmount())
	}
	h.m.Unlock()
	claims, _ := total.Float64()
	return claims
}

// setAmounts sets current account amounts in relation to exchange liqudity.
// These totals passed in are exchange items only, we will need to calculate our
// free amounts dependant on current claims, whats currently locked and whats
// pending.
func (h *Holding) setAmounts(total, locked decimal.Decimal) {
	// Determine total free on the exchange
	free := total.Sub(locked)
	h.m.Lock()

	// Determine full claimed amount
	var claimed decimal.Decimal
	for x := range h.claims {
		claimed = claimed.Add(h.claims[x].getAmount())
	}

	if !h.pending.LessThanOrEqual(decimal.Zero) {
		totalDifference := h.total.Sub(total)
		if totalDifference.GreaterThan(decimal.Zero) {
			// Reduce our pending claims which increases our free amount
			h.pending = h.pending.Sub(totalDifference)
		}
		// remove the residual pending amount from the free amount
		remaining := h.pending.Sub(locked)
		if remaining.GreaterThan(decimal.Zero) {
			free = free.Sub(h.pending)
		}
	}
	h.total = total
	h.locked = locked
	h.free = free.Sub(claimed) // Remove any claims on free amounts
	h.m.Unlock()
}

// Claim returns a claim to an amount for the exchange account holding. Allows
// strategies to segregate their own funds from each other while executing in
// parallel. If total amount is required, this will return an error else the
// remaining/free amount will be claimed and a pointer returned.
func (h *Holding) Claim(amount float64, totalRequired bool) (*Claim, error) {
	amt := decimal.NewFromFloat(amount)
	h.m.Lock()
	defer h.m.Unlock()
	if h.free.Equal(decimal.Zero) {
		return nil, errNoBalance
	}
	remainder := h.free.Sub(amt)
	if remainder.LessThan(decimal.Zero) {
		if totalRequired {
			return nil, errAmountExceedsHoldings
		}
		// Claims the total free amount
		freeClaim := &Claim{amount: h.free, h: h}
		// sets free amount to zero
		h.free = decimal.Zero
		// Adds claim for tracking
		h.claims = append(h.claims, freeClaim)
		return freeClaim, nil
	}
	// sets the remainder to the new free amount
	h.free = remainder
	amountClaim := &Claim{amount: amt, h: h}
	h.claims = append(h.claims, amountClaim)
	// return the full requested amount
	return amountClaim, nil
}

var errClaimIsNil = errors.New("claim is nil")

// Release is a protected exported function to release funds that has not
// been successful or is not used
func (h *Holding) Release(c *Claim) error {
	if c == nil {
		return errClaimIsNil
	}
	h.m.Lock()
	defer h.m.Unlock()
	err := h.release(c, false)
	if err != nil {
		return err
	}

	if h.verbose {
		log.Debugf(log.Accounts,
			"Exchange:%s Account:%s Asset:%s Currency:%s Claim of %f, released.",
			c.Exchange,
			c.Account,
			c.Asset,
			c.Currency,
			c.GetAmount())
	}
	return nil
}

// ReleaseToPending is a protected exported function to release funds and shift
// them to pending when an order or a withdrawal opperation has succeeded.
func (h *Holding) ReleaseToPending(c *Claim) error {
	if c == nil {
		return errClaimIsNil
	}

	h.m.Lock()
	defer h.m.Unlock()
	err := h.release(c, true)
	if err != nil {
		return err
	}

	if h.verbose {
		log.Debugf(log.Accounts,
			"Exchange:%s Account:%s Asset:%s Currency:%s Claim of %f, released to pending.",
			c.Exchange,
			c.Account,
			c.Asset,
			c.Currency,
			c.GetAmount())
	}
	return nil
}

var errClaimInvalid = errors.New("claim amount cannot be less than or equal to zero")

// release releases the funds either to pending or free.
func (h *Holding) release(c *Claim, pending bool) error {
	if !c.amount.GreaterThan(decimal.Zero) {
		return errClaimInvalid
	}
	for x := range h.claims {
		if h.claims[x] == c {
			// Remove claim from claims slice
			h.claims[x] = h.claims[len(h.claims)-1]
			h.claims[len(h.claims)-1] = nil
			h.claims = h.claims[:len(h.claims)-1]

			if pending {
				// Change pending amount to be re-adjusted when a new update
				// comes through
				h.pending = h.pending.Add(c.amount)
				return nil
			}
			// Change free amount NOTE: not changing locked amount as this is
			// done by the exchange update
			h.free = h.free.Add(c.amount)
			return nil
		}
	}
	return errUnableToReleaseClaim
}

// CheckClaim determines if a claim is still on a currency holding
func (h *Holding) CheckClaim(c *Claim) bool {
	h.m.Lock()
	defer h.m.Unlock()
	for x := range h.claims {
		if h.claims[x] == c {
			return true
		}
	}
	return false
}

// adjustByBalance defines a way in which the entire holdings can be adjusted by
// a balance change using pending levels as a reference.
func (h *Holding) adjustByBalance(amount float64) error {
	if amount == 0 {
		return errAmountCannotBeZero
	}

	h.m.Lock()
	defer h.m.Unlock()
	amt := decimal.NewFromFloat(amount)
	if amount > 0 {
		switch {
		case h.pending.GreaterThan(decimal.Zero):
			h.free = h.free.Add(amt)
			remaining := h.pending.Sub(amt)
			if remaining.GreaterThanOrEqual(decimal.Zero) {
				amt = amt.Sub(remaining)
				h.pending = remaining
			} else {
				amt = amt.Sub(h.pending)
				h.pending = decimal.Zero
				h.total = h.total.Sub(remaining)
			}

			if amt.Equal(decimal.Zero) || !h.locked.GreaterThan(decimal.Zero) {
				break
			}

			h.free = h.free.Sub(amt)
			h.total = h.total.Sub(amt)
			fallthrough
		case h.locked.GreaterThan(decimal.Zero):
			if h.locked.GreaterThanOrEqual(amt) {
				remaining := h.locked.Sub(amt)
				h.locked = remaining
				h.free = h.free.Add(amt)
			} else {
				remaining := amt.Sub(h.locked)
				h.free = h.free.Add(h.locked.Add(remaining))
				h.total = h.total.Add(remaining)
				h.locked = decimal.Zero
			}
		default:
			h.total = h.total.Add(amt)
			h.free = h.free.Add(amt)
		}
	} else {
		switch {
		case h.pending.GreaterThan(decimal.Zero):
			h.total = h.total.Add(amt)
			remaining := h.pending.Sub(amt.Abs())
			if remaining.GreaterThanOrEqual(decimal.Zero) {
				amt = amt.Add(remaining)
				h.pending = remaining
			} else {
				amt = amt.Add(h.pending)
				h.pending = decimal.Zero
				h.free = h.free.Add(remaining)
			}

			if amt.Equal(decimal.Zero) || !h.locked.GreaterThan(decimal.Zero) {
				break
			}
			h.free = h.free.Sub(amt)
			h.total = h.total.Sub(amt)
			fallthrough
		case h.locked.GreaterThan(decimal.Zero):
			if h.locked.GreaterThanOrEqual(amt.Abs()) {
				remaining := h.locked.Add(amt)
				h.locked = remaining
				h.total = h.total.Add(amt)
			} else {
				remaining := amt.Add(h.locked).Abs()
				h.free = h.free.Sub(remaining)
				h.total = h.total.Add(amt)
				h.locked = decimal.Zero
			}
		default:
			h.total = h.total.Add(amt)
			h.free = h.free.Add(amt)
		}
	}
	return nil
}

// reduce reduces holdings by claim
func (h *Holding) reduce(c *Claim) error {
	h.m.Lock()
	defer h.m.Unlock()
	for x := range h.claims {
		if h.claims[x] == c {
			// Remove claim from claims slice
			h.claims[x] = h.claims[len(h.claims)-1]
			h.claims[len(h.claims)-1] = nil
			h.claims = h.claims[:len(h.claims)-1]

			// Reduce total amount
			h.total = h.total.Sub(c.getAmount())

			if h.verbose {
				log.Debugf(log.Accounts,
					"Exchange:%s Account:%s Asset:%s Currency:%s Claim of %f released, total balance reduced.",
					c.Exchange,
					c.Account,
					c.Asset,
					c.Currency,
					c.GetAmount())
			}
			return nil
		}
	}
	return errUnableToReduceClaim
}

// GetBalance returns a balance on holdings, can pass in
func (h *Holding) GetBalance(all bool) (Balance, error) {
	h.m.Lock()
	defer h.m.Unlock()

	total, _ := h.total.Float64()
	if total == 0 && !all {
		return Balance{}, errNoBalance
	}

	locked, _ := h.locked.Float64()
	return Balance{total, locked}, nil
}
