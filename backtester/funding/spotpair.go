package funding

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// ErrNotCollateral is returned when a user requests collateral pair details when it is a funding pair
var ErrNotCollateral = errors.New("not a collateral pair")

// BaseInitialFunds returns the initial funds
// from the base in a currency pair
func (p *SpotPair) BaseInitialFunds() decimal.Decimal {
	return p.base.initialFunds
}

// QuoteInitialFunds returns the initial funds
// from the quote in a currency pair
func (p *SpotPair) QuoteInitialFunds() decimal.Decimal {
	return p.quote.initialFunds
}

// BaseAvailable returns the available funds
// from the base in a currency pair
func (p *SpotPair) BaseAvailable() decimal.Decimal {
	return p.base.available
}

// QuoteAvailable returns the available funds
// from the quote in a currency pair
func (p *SpotPair) QuoteAvailable() decimal.Decimal {
	return p.quote.available
}

// Reserve allocates an amount of funds to be used at a later time
// it prevents multiple events from claiming the same resource
// changes which currency to affect based on the order side
func (p *SpotPair) Reserve(amount decimal.Decimal, side order.Side) error {
	switch side {
	case order.Buy, order.Bid:
		return p.quote.Reserve(amount)
	case order.Sell, order.Ask, order.ClosePosition:
		return p.base.Reserve(amount)
	default:
		return fmt.Errorf("%w for %v %v %v. Unknown side %v",
			errCannotAllocate,
			p.base.exchange,
			p.base.asset,
			p.base.currency,
			side)
	}
}

// Release reduces the amount of funding reserved and adds any difference
// back to the available amount
// changes which currency to affect based on the order side
func (p *SpotPair) Release(amount, diff decimal.Decimal, side order.Side) error {
	switch side {
	case order.Buy, order.Bid:
		return p.quote.Release(amount, diff)
	case order.Sell, order.Ask:
		return p.base.Release(amount, diff)
	}
	return fmt.Errorf("%w for %v %v %v. Unknown side %v",
		errCannotAllocate,
		p.base.exchange,
		p.base.asset,
		p.base.currency,
		side)
}

// IncreaseAvailable adds funding to the available amount
// changes which currency to affect based on the order side
func (p *SpotPair) IncreaseAvailable(amount decimal.Decimal, side order.Side) error {
	switch side {
	case order.Buy, order.Bid:
		return p.base.IncreaseAvailable(amount)
	case order.Sell, order.Ask, order.ClosePosition:
		return p.quote.IncreaseAvailable(amount)
	}
	return fmt.Errorf("%w for %v %v %v. Unknown side %v",
		errCannotAllocate,
		p.base.exchange,
		p.base.asset,
		p.base.currency,
		side)
}

// CanPlaceOrder does a > 0 check to see if there are any funds
// to place an order with
// changes which currency to affect based on the order side
func (p *SpotPair) CanPlaceOrder(side order.Side) bool {
	switch side {
	case order.Buy, order.Bid:
		return p.quote.CanPlaceOrder()
	case order.Sell, order.Ask, order.ClosePosition:
		return p.base.CanPlaceOrder()
	}
	return false
}

// Liquidate basic liquidation response to remove
// all asset value
func (p *SpotPair) Liquidate() {
	p.base.available = decimal.Zero
	p.base.reserved = decimal.Zero
	p.quote.available = decimal.Zero
	p.quote.reserved = decimal.Zero
}

// FundReserver returns a fund reserver interface of the pair
func (p *SpotPair) FundReserver() IFundReserver {
	return p
}

// PairReleaser returns a pair releaser interface of the pair
func (p *SpotPair) PairReleaser() (IPairReleaser, error) {
	if p == nil {
		return nil, ErrNilPair
	}
	return p, nil
}

// CollateralReleaser returns an error because a pair is not collateral
func (p *SpotPair) CollateralReleaser() (ICollateralReleaser, error) {
	return nil, ErrNotCollateral
}

// FundReleaser returns a pair releaser interface of the pair
func (p *SpotPair) FundReleaser() IFundReleaser {
	return p
}

// FundReader returns a fund reader interface of the pair
func (p *SpotPair) FundReader() IFundReader {
	return p
}

// GetPairReader returns an interface of a SpotPair
func (p *SpotPair) GetPairReader() (IPairReader, error) {
	return p, nil
}

// GetCollateralReader returns an error because its not collateral
func (p *SpotPair) GetCollateralReader() (ICollateralReader, error) {
	return nil, ErrNotCollateral
}
