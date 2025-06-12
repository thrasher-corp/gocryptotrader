package funding

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestCollateralCanPlaceOrder(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		collateral: &Item{available: decimal.NewFromInt(1337)},
	}
	if !c.CanPlaceOrder(gctorder.UnknownSide) {
		t.Error("expected true")
	}
}

func TestCollateralTakeProfit(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		collateral: &Item{
			asset:        asset.Futures,
			isCollateral: true,
		},
		contract: &Item{
			asset:     asset.Futures,
			available: decimal.NewFromInt(1),
		},
	}
	err := c.TakeProfit(decimal.NewFromInt(1), decimal.NewFromInt(1))
	assert.NoError(t, err, "TakeProfit should not error")
}

func TestCollateralCollateralCurrency(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		collateral: &Item{currency: currency.DOGE},
	}
	if !c.CollateralCurrency().Equal(currency.DOGE) {
		t.Errorf("received '%v' expected '%v'", c.CollateralCurrency(), currency.DOGE)
	}
}

func TestCollateralContractCurrency(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		contract: &Item{currency: currency.DOGE},
	}
	if !c.ContractCurrency().Equal(currency.DOGE) {
		t.Errorf("received '%v' expected '%v'", c.ContractCurrency(), currency.DOGE)
	}
}

func TestCollateralInitialFunds(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		collateral: &Item{initialFunds: decimal.NewFromInt(1337)},
	}
	if !c.InitialFunds().Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '%v'", c.InitialFunds(), decimal.NewFromInt(1337))
	}
}

func TestCollateralAvailableFunds(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		collateral: &Item{available: decimal.NewFromInt(1337)},
	}
	if !c.AvailableFunds().Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '%v'", c.AvailableFunds(), decimal.NewFromInt(1337))
	}
}

func TestCollateralGetPairReader(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		contract:   &Item{},
		collateral: &Item{},
	}
	_, err := c.GetPairReader()
	assert.ErrorIs(t, err, ErrNotPair)
}

func TestCollateralGetCollateralReader(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		collateral: &Item{available: decimal.NewFromInt(1337)},
	}
	cr, err := c.GetCollateralReader()
	require.NoError(t, err, "GetCollateralReader must not error")
	assert.Equal(t, cr, c)
}

func TestCollateralUpdateContracts(t *testing.T) {
	t.Parallel()
	b := gctorder.Buy
	c := &CollateralPair{
		collateral: &Item{
			asset:        asset.Futures,
			isCollateral: true,
		},
		contract:         &Item{asset: asset.Futures},
		currentDirection: &b,
	}
	leet := decimal.NewFromInt(1337)
	err := c.UpdateContracts(gctorder.Buy, leet)
	assert.NoError(t, err, "UpdateContracts should not error")

	if !c.contract.available.Equal(leet) {
		t.Errorf("received '%v' expected '%v'", c.contract.available, leet)
	}
	b = gctorder.Sell
	err = c.UpdateContracts(gctorder.Buy, leet)
	assert.NoError(t, err, "UpdateContracts should not error")

	if !c.contract.available.Equal(decimal.Zero) {
		t.Errorf("received '%v' expected '%v'", c.contract.available, decimal.Zero)
	}

	c.currentDirection = nil
	err = c.UpdateContracts(gctorder.Buy, leet)
	assert.NoError(t, err, "UpdateContracts should not error")

	if !c.contract.available.Equal(leet) {
		t.Errorf("received '%v' expected '%v'", c.contract.available, leet)
	}
}

func TestCollateralReleaseContracts(t *testing.T) {
	t.Parallel()
	b := gctorder.Buy
	c := &CollateralPair{
		collateral: &Item{
			asset:        asset.Futures,
			isCollateral: true,
		},
		contract:         &Item{asset: asset.Futures},
		currentDirection: &b,
	}

	err := c.ReleaseContracts(decimal.Zero)
	assert.ErrorIs(t, err, errPositiveOnly)

	err = c.ReleaseContracts(decimal.NewFromInt(1337))
	assert.ErrorIs(t, err, errCannotAllocate)

	c.contract.available = decimal.NewFromInt(1337)
	err = c.ReleaseContracts(decimal.NewFromInt(1337))
	assert.NoError(t, err, "ReleaseContracts should not error")
}

func TestCollateralFundReader(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		collateral: &Item{available: decimal.NewFromInt(1337)},
	}
	if c.FundReader() != c {
		t.Error("expected the same thing")
	}
}

func TestCollateralPairReleaser(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		collateral: &Item{},
		contract:   &Item{},
	}
	_, err := c.PairReleaser()
	assert.ErrorIs(t, err, ErrNotPair)
}

func TestCollateralFundReserver(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		collateral: &Item{available: decimal.NewFromInt(1337)},
	}
	if c.FundReserver() != c {
		t.Error("expected the same thing")
	}
}

func TestCollateralCollateralReleaser(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		collateral: &Item{},
		contract:   &Item{},
	}
	_, err := c.CollateralReleaser()
	assert.NoError(t, err, "CollateralReleaser should not error")
}

func TestCollateralFundReleaser(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		collateral: &Item{available: decimal.NewFromInt(1337)},
	}
	if c.FundReleaser() != c {
		t.Error("expected the same thing")
	}
}

func TestCollateralReserve(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		collateral: &Item{
			asset:        asset.Futures,
			isCollateral: true,
			available:    decimal.NewFromInt(1337),
		},
		contract: &Item{asset: asset.Futures},
	}
	err := c.Reserve(decimal.NewFromInt(1), gctorder.Long)
	require.NoError(t, err, "Reserve must not error")
	assert.Equal(t, decimal.NewFromInt(1), c.collateral.reserved)
	assert.Equal(t, decimal.NewFromInt(1336), c.collateral.available)

	err = c.Reserve(decimal.NewFromInt(1), gctorder.Short)
	require.NoError(t, err, "Reserve must not error")
	assert.Equal(t, decimal.NewFromInt(2), c.collateral.reserved)
	assert.Equal(t, decimal.NewFromInt(1335), c.collateral.available)

	err = c.Reserve(decimal.NewFromInt(2), gctorder.ClosePosition)
	require.NoError(t, err, "Reserve must not error")
	assert.Equal(t, decimal.NewFromInt(4), c.collateral.reserved)
	assert.Equal(t, decimal.NewFromInt(1333), c.collateral.available)
	err = c.Reserve(decimal.NewFromInt(2), gctorder.Buy)
	assert.ErrorIs(t, err, errCannotAllocate)
}

func TestCollateralLiquidate(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		collateral: &Item{
			asset:        asset.Futures,
			isCollateral: true,
			available:    decimal.NewFromInt(1337),
		},
		contract: &Item{
			asset:     asset.Futures,
			available: decimal.NewFromInt(1337),
		},
	}
	c.Liquidate()
	if !c.collateral.available.Equal(decimal.Zero) {
		t.Errorf("received '%v' expected '%v'", c.collateral.available, decimal.Zero)
	}
	if !c.contract.available.Equal(decimal.Zero) {
		t.Errorf("received '%v' expected '%v'", c.contract.available, decimal.Zero)
	}
}

func TestCollateralCurrentHoldings(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		contract: &Item{available: decimal.NewFromInt(1337)},
	}
	if !c.CurrentHoldings().Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '%v'", c.CurrentHoldings(), decimal.NewFromInt(1337))
	}
}
