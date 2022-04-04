package funding

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// collateral related errors
var (
	ErrNotCollateral = errors.New("not a collateral pair")
	ErrIsCollateral  = errors.New("is collateral pair")
	ErrNilPair       = errors.New("nil pair")
	errPositiveOnly  = errors.New("reduces the amount by subtraction, positive numbers only")
)

// CanPlaceOrder checks if there is any collateral to spare
func (c *Collateral) CanPlaceOrder(_ gctorder.Side) bool {
	return c.collateral.CanPlaceOrder()
}

// TakeProfit handles both the reduction of contracts and the change in collateral
func (c *Collateral) TakeProfit(contracts, positionReturns decimal.Decimal) error {
	err := c.contract.ReduceContracts(contracts)
	if err != nil {
		return err
	}
	return c.collateral.TakeProfit(positionReturns)
}

// ContractCurrency returns the contract currency
func (c *Collateral) ContractCurrency() currency.Code {
	return c.contract.currency
}

// CollateralCurrency returns collateral currency
func (c *Collateral) CollateralCurrency() currency.Code {
	return c.collateral.currency
}

// InitialFunds returns initial funds of collateral
func (c *Collateral) InitialFunds() decimal.Decimal {
	return c.collateral.initialFunds
}

// AvailableFunds returns available funds of collateral
func (c *Collateral) AvailableFunds() decimal.Decimal {
	return c.collateral.available
}

// GetPairReader returns an error because collateral isn't a pair
func (c *Collateral) GetPairReader() (IPairReader, error) {
	return nil, fmt.Errorf("could not return pair reader for %v %v %v %v %w", c.contract.exchange, c.collateral.asset, c.ContractCurrency(), c.CollateralCurrency(), ErrNotPair)
}

// GetCollateralReader returns a collateral reader interface of Collateral
func (c *Collateral) GetCollateralReader() (ICollateralReader, error) {
	return c, nil
}

// UpdateContracts adds or subtracts contracts based on order direction
func (c *Collateral) UpdateContracts(s gctorder.Side, amount decimal.Decimal) error {
	switch {
	case c.currentDirection == nil:
		c.currentDirection = &s
		return c.contract.AddContracts(amount)
	case *c.currentDirection == s:
		return c.contract.AddContracts(amount)
	case *c.currentDirection != s:
		return c.contract.ReduceContracts(amount)
	default:
		return errors.New("woah nelly")
	}
}

// ReleaseContracts lowers the amount of available contracts
func (c *Collateral) ReleaseContracts(amount decimal.Decimal) error {
	if amount.LessThan(decimal.Zero) {
		return fmt.Errorf("release %w", errPositiveOnly)
	}
	if c.contract.available.LessThan(amount) {
		return fmt.Errorf("%w amount '%v' larger than available '%v'", errCannotAllocate, amount, c.contract.available)
	}
	c.contract.available = c.contract.available.Sub(amount)
	return nil
}

// FundReader returns a fund reader interface of collateral
func (c *Collateral) FundReader() IFundReader {
	return c
}

// FundReserver returns a fund reserver interface of Collateral
func (c *Collateral) FundReserver() IFundReserver {
	return c
}

// PairReleaser returns an error as there is no such thing for collateral
func (c *Collateral) PairReleaser() (IPairReleaser, error) {
	return nil, fmt.Errorf("could not get pair releaser for %v %v %v %v %w", c.contract.exchange, c.collateral.asset, c.ContractCurrency(), c.CollateralCurrency(), ErrNotPair)
}

// Reserve reserves or releases collateral based on order side
func (c *Collateral) Reserve(amount decimal.Decimal, side gctorder.Side) error {
	switch side {
	case gctorder.Long, gctorder.Short:
		return c.collateral.Reserve(amount)
	case common.ClosePosition:
		return c.collateral.Release(amount, amount)
	default:
		return fmt.Errorf("%w for %v %v %v. Unknown side %v",
			errCannotAllocate,
			c.collateral.exchange,
			c.collateral.asset,
			c.collateral.currency,
			side)
	}
}

// CollateralReleaser returns an ICollateralReleaser to interact with
// collateral
func (c *Collateral) CollateralReleaser() (ICollateralReleaser, error) {
	return c, nil
}

// FundReleaser returns an IFundReleaser to interact with
// collateral
func (c *Collateral) FundReleaser() IFundReleaser {
	return c
}

// Liquidate kills your funds and future
func (c *Collateral) Liquidate() {
	c.collateral.available = decimal.Zero
	c.collateral.reserved = decimal.Zero
	c.contract.available = decimal.Zero
	c.contract.reserved = decimal.Zero
	c.currentDirection = nil
}

// CurrentHoldings returns available contract holdings
func (c *Collateral) CurrentHoldings() decimal.Decimal {
	return c.contract.available
}
