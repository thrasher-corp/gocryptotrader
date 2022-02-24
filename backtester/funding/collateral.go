package funding

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	// ErrNotCollateral is returned when a user requests collateral from a non-collateral pair
	ErrNotCollateral = errors.New("not a collateral pair")
	ErrNilPair       = errors.New("nil pair")
)

// TODO consider moving futures tracking to funding
// we're already passing around funding items, it can then also have all the lovely tracking attached?

func (c *Collateral) CanPlaceOrder(_ order.Side) bool {
	return c.Collateral.CanPlaceOrder()
}

func (c *Collateral) TakeProfit(contracts, originalPositionSize, positionReturns decimal.Decimal) error {
	err := c.Contract.Release(contracts, decimal.Zero)
	if err != nil {
		return err
	}
	return c.Collateral.Release(originalPositionSize, positionReturns)
}

func (c *Collateral) ContractCurrency() currency.Code {
	return c.Contract.currency
}

func (c *Collateral) UnderlyingAsset() currency.Code {
	// somehow get the underlying
	return c.Contract.currency
}

func (c *Collateral) CollateralCurrency() currency.Code {
	return c.Collateral.currency
}

func (c *Collateral) InitialFunds() decimal.Decimal {
	return c.Collateral.initialFunds
}

func (c *Collateral) AvailableFunds() decimal.Decimal {
	return c.Collateral.available
}

func (c *Collateral) GetPairReader() (IPairReader, error) {
	return nil, fmt.Errorf("could not return pair reader for %v %v %v %v %w", c.Contract.exchange, c.Collateral.asset, c.ContractCurrency(), c.CollateralCurrency(), ErrNotPair)
}

func (c *Collateral) GetCollateralReader() (ICollateralReader, error) {
	return c, nil
}

func (c *Collateral) UpdateCollateral(s order.Side, amount, diff decimal.Decimal) error {
	switch {
	case c.currentDirection == nil:
		c.currentDirection = &s
		return c.Collateral.Reserve(amount)
	case *c.currentDirection == s:
		return c.Collateral.Reserve(amount)
	case *c.currentDirection != s:
		return c.Collateral.Release(amount, diff)
	default:
		return errors.New("woah nelly")
	}
}

func (c *Collateral) UpdateContracts(s order.Side, amount, diff decimal.Decimal) error {
	switch {
	case c.currentDirection == nil:
		c.currentDirection = &s
		return c.Contract.Reserve(amount)
	case *c.currentDirection == s:
		return c.Contract.Reserve(amount)
	case *c.currentDirection != s:
		return c.Contract.Release(amount, diff)
	default:
		return errors.New("woah nelly")
	}
}

func (c *Collateral) ReleaseContracts(amount decimal.Decimal) error {
	// turn this into a protected func
	c.Contract.available = c.Contract.available.Sub(amount)
	return nil
}

// FundReader
func (c *Collateral) FundReader() IFundReader {
	return c
}

// FundReserver
func (c *Collateral) FundReserver() IFundReserver {
	return c
}

// GetPairReleaser
func (c *Collateral) GetPairReleaser() (IPairReleaser, error) {
	return nil, fmt.Errorf("could not get pair releaser for %v %v %v %v %w", c.Contract.exchange, c.Collateral.asset, c.ContractCurrency(), c.CollateralCurrency(), ErrNotPair)
}

// GetCollateralReleaser
func (c *Collateral) GetCollateralReleaser() (ICollateralReleaser, error) {
	return c, nil
}

// FundReleaser
func (c *Collateral) FundReleaser() IFundReleaser {
	return c
}

func (c *Collateral) Liquidate() {
	c.Collateral.available = decimal.Zero
	c.Contract.available = decimal.Zero
}

func (c *Collateral) CurrentHoldings() decimal.Decimal {
	return c.Contract.available
}
