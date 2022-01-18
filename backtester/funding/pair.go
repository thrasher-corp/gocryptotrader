package funding

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	// ErrNotPair is returned when a user requests funding pair details when it is a collateral pair
	ErrNotPair = errors.New("not a funding pair")
)

// BaseInitialFunds returns the initial funds
// from the base in a currency pair
func (p *Pair) BaseInitialFunds() decimal.Decimal {
	return p.Base.initialFunds
}

// QuoteInitialFunds returns the initial funds
// from the quote in a currency pair
func (p *Pair) QuoteInitialFunds() decimal.Decimal {
	return p.Quote.initialFunds
}

// BaseAvailable returns the available funds
// from the base in a currency pair
func (p *Pair) BaseAvailable() decimal.Decimal {
	return p.Base.available
}

// QuoteAvailable returns the available funds
// from the quote in a currency pair
func (p *Pair) QuoteAvailable() decimal.Decimal {
	return p.Quote.available
}

func (p *Pair) GetPairReader() (IPairReader, error) {
	return p, nil
}

func (p *Pair) GetCollateralReader() (ICollateralReader, error) {
	return nil, ErrNotCollateral
}

// Reserve allocates an amount of funds to be used at a later time
// it prevents multiple events from claiming the same resource
// changes which currency to affect based on the order side
func (p *Pair) Reserve(amount decimal.Decimal, side order.Side) error {
	switch side {
	case order.Buy:
		return p.Quote.Reserve(amount)
	case order.Sell:
		return p.Base.Reserve(amount)
	default:
		return fmt.Errorf("%w for %v %v %v. Unknown side %v",
			errCannotAllocate,
			p.Base.exchange,
			p.Base.asset,
			p.Base.currency,
			side)
	}
}

// Release reduces the amount of funding reserved and adds any difference
// back to the available amount
// changes which currency to affect based on the order side
func (p *Pair) Release(amount, diff decimal.Decimal, side order.Side) error {
	switch side {
	case order.Buy:
		return p.Quote.Release(amount, diff)
	case order.Sell:
		return p.Base.Release(amount, diff)
	default:
		return fmt.Errorf("%w for %v %v %v. Unknown side %v",
			errCannotAllocate,
			p.Base.exchange,
			p.Base.asset,
			p.Base.currency,
			side)
	}
}

// IncreaseAvailable adds funding to the available amount
// changes which currency to affect based on the order side
func (p *Pair) IncreaseAvailable(amount decimal.Decimal, side order.Side) {
	switch side {
	case order.Buy:
		p.Base.IncreaseAvailable(amount)
	case order.Sell:
		p.Quote.IncreaseAvailable(amount)
	}
}

// CanPlaceOrder does a > 0 check to see if there are any funds
// to place an order with
// changes which currency to affect based on the order side
func (p *Pair) CanPlaceOrder(side order.Side) bool {
	switch side {
	case order.Buy:
		return p.Quote.CanPlaceOrder()
	case order.Sell:
		return p.Base.CanPlaceOrder()
	}
	return false
}

// FundReader
func (p *Pair) FundReader() IFundReader {
	return p
}

// FundReserver
func (p *Pair) FundReserver() IFundReserver {
	return p
}

// GetPairReleaser
func (p *Pair) GetPairReleaser() (IPairReleaser, error) {
	if p == nil {
		return nil, ErrNilPair
	}
	return p, nil
}

// GetCollateralReleaser
func (p *Pair) GetCollateralReleaser() (ICollateralReleaser, error) {
	return nil, ErrNotCollateral
}

// FundReleaser
func (p *Pair) FundReleaser() IFundReleaser {
	return p
}
