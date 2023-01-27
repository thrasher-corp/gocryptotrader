package funding

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
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
		contract: &Item{asset: asset.Futures,
			available: decimal.NewFromInt(1),
		},
	}
	var expectedError error
	err := c.TakeProfit(decimal.NewFromInt(1), decimal.NewFromInt(1))
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
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
	if _, err := c.GetPairReader(); !errors.Is(err, ErrNotPair) {
		t.Errorf("received '%v' expected '%v'", err, ErrNotPair)
	}
}

func TestCollateralGetCollateralReader(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		collateral: &Item{available: decimal.NewFromInt(1337)},
	}
	var expectedError error
	cr, err := c.GetCollateralReader()
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	if cr != c {
		t.Error("expected the same thing")
	}
}

func TestCollateralUpdateContracts(t *testing.T) {
	t.Parallel()
	b := gctorder.Buy
	var expectedError error
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
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	if !c.contract.available.Equal(leet) {
		t.Errorf("received '%v' expected '%v'", c.contract.available, leet)
	}
	b = gctorder.Sell
	err = c.UpdateContracts(gctorder.Buy, leet)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	if !c.contract.available.Equal(decimal.Zero) {
		t.Errorf("received '%v' expected '%v'", c.contract.available, decimal.Zero)
	}

	c.currentDirection = nil
	err = c.UpdateContracts(gctorder.Buy, leet)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
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

	expectedError := errPositiveOnly
	err := c.ReleaseContracts(decimal.Zero)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	expectedError = errCannotAllocate
	err = c.ReleaseContracts(decimal.NewFromInt(1337))
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	expectedError = nil
	c.contract.available = decimal.NewFromInt(1337)
	err = c.ReleaseContracts(decimal.NewFromInt(1337))
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
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
	if _, err := c.PairReleaser(); !errors.Is(err, ErrNotPair) {
		t.Errorf("received '%v' expected '%v'", err, ErrNotPair)
	}
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
	var expectedError error
	if _, err := c.CollateralReleaser(); !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
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
	var expectedError error
	err := c.Reserve(decimal.NewFromInt(1), gctorder.Long)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	if !c.collateral.reserved.Equal(decimal.NewFromInt(1)) {
		t.Errorf("received '%v' expected '%v'", c.collateral.reserved, decimal.NewFromInt(1))
	}
	if !c.collateral.available.Equal(decimal.NewFromInt(1336)) {
		t.Errorf("received '%v' expected '%v'", c.collateral.available, decimal.NewFromInt(1336))
	}

	err = c.Reserve(decimal.NewFromInt(1), gctorder.Short)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	if !c.collateral.reserved.Equal(decimal.NewFromInt(2)) {
		t.Errorf("received '%v' expected '%v'", c.collateral.reserved, decimal.NewFromInt(2))
	}
	if !c.collateral.available.Equal(decimal.NewFromInt(1335)) {
		t.Errorf("received '%v' expected '%v'", c.collateral.available, decimal.NewFromInt(1335))
	}

	err = c.Reserve(decimal.NewFromInt(2), gctorder.ClosePosition)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	if !c.collateral.reserved.Equal(decimal.NewFromInt(4)) {
		t.Errorf("received '%v' expected '%v'", c.collateral.reserved, decimal.Zero)
	}
	if !c.collateral.available.Equal(decimal.NewFromInt(1333)) {
		t.Errorf("received '%v' expected '%v'", c.collateral.available, decimal.NewFromInt(1333))
	}

	expectedError = errCannotAllocate
	err = c.Reserve(decimal.NewFromInt(2), gctorder.Buy)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
}

func TestCollateralLiquidate(t *testing.T) {
	t.Parallel()
	c := &CollateralPair{
		collateral: &Item{
			asset:        asset.Futures,
			isCollateral: true,
			available:    decimal.NewFromInt(1337),
		},
		contract: &Item{asset: asset.Futures,
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
