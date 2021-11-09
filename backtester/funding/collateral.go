package funding

import (
	"errors"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	// ErrNotCollateral is returned when a user requests collateral from a non-collateral pair
	ErrNotCollateral = errors.New("not a collateral pair")
)

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
	return nil, ErrNotPair
}

func (c *Collateral) GetCollateralReader() (ICollateralReader, error) {
	return c, nil
}
