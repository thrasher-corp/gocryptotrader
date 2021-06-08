package account

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const AccountTest = "test"

var one = decimal.NewFromInt(1)
var twenty = decimal.NewFromInt(20)

func TestLoadAccount(t *testing.T) {
	h := Holdings{}
	err := h.LoadAccount("")
	if !errors.Is(err, errAccountNameUnset) {
		t.Fatalf("expected: %v but received: %v", errAccountNameUnset, err)
	}

	err = h.LoadAccount("testAccount")
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	err = h.LoadAccount("testAccOunt")
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if len(h.availableAccounts) != 1 {
		t.Fatal("unexpected account count")
	}
}

func TestGetAccounts(t *testing.T) {
	h := Holdings{}
	_, err := h.GetAccounts()
	if !errors.Is(err, errAccountsNotLoaded) {
		t.Fatalf("expected: %v but received: %v", errAccountsNotLoaded, err)
	}

	err = h.LoadAccount("testAccount")
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	accs, err := h.GetAccounts()
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if len(accs) != 1 {
		t.Fatal("unexpected amount received")
	}

	if accs[0] != "testaccount" {
		t.Fatalf("unexpected value %s received", accs[0])
	}
}

func TestAccountValid(t *testing.T) {
	h := Holdings{}
	err := h.AccountValid("")
	if !errors.Is(err, errAccountNameUnset) {
		t.Fatalf("expected: %v but received: %v", errAccountNameUnset, err)
	}

	err = h.AccountValid("test")
	if !errors.Is(err, errAccountNotFound) {
		t.Fatalf("expected: %v but received: %v", errAccountNotFound, err)
	}

	err = h.LoadAccount("testAccount")
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	err = h.AccountValid("tEsTAccOuNt")
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
}

func TestGetHolding(t *testing.T) {
	h, err := DeployHoldings("getHolding", false)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	_, err = h.GetHolding("", "", currency.Code{})
	if !errors.Is(err, errAccountNameUnset) {
		t.Fatalf("expected: %v but received: %v", errAccountNameUnset, err)
	}

	_, err = h.GetHolding(AccountTest, "", currency.Code{})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("expected: %v but received: %v", asset.ErrNotSupported, err)
	}

	_, err = h.GetHolding(AccountTest, asset.Spot, currency.Code{})
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("expected: %v but received: %v", errCurrencyIsEmpty, err)
	}

	values := HoldingsSnapshot{
		currency.BTC: {Total: 1},
		currency.LTC: {Total: 20},
	}

	err = h.LoadHoldings(AccountTest, asset.Spot, values)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	btcHolding, err := h.GetHolding(AccountTest, asset.Spot, currency.BTC)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if !btcHolding.free.Equal(one) {
		t.Fatalf("expected free holdings: %s, but received %s", one, btcHolding.free)
	}

	if !btcHolding.locked.Equal(decimal.Zero) {
		t.Fatalf("expected free holdings: %s, but received %s", decimal.Zero, btcHolding.locked)
	}

	ltcHolding, err := h.GetHolding(AccountTest, asset.Spot, currency.LTC)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if !ltcHolding.free.Equal(twenty) {
		t.Fatalf("expected free holdings: %s, but received %s", twenty, ltcHolding.free)
	}

	if !ltcHolding.locked.Equal(decimal.Zero) {
		t.Fatalf("expected free holdings: %s, but received %s", decimal.Zero, ltcHolding.locked)
	}

	ethHolding, err := h.GetHolding("subAccount", asset.Spot, currency.ETH)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if !ethHolding.free.Equal(decimal.Zero) {
		t.Fatalf("expected free holdings: %s, but received %s", decimal.Zero, ethHolding.free)
	}

	if !ethHolding.locked.Equal(decimal.Zero) {
		t.Fatalf("expected free holdings: %s, but received %s", decimal.Zero, ethHolding.locked)
	}
}

func TestLoad(t *testing.T) {
	h, err := DeployHoldings("load", false)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	err = h.LoadHoldings("", "", nil)
	if !errors.Is(err, errAccountNameUnset) {
		t.Fatalf("expected: %v but received: %v", errAccountNameUnset, err)
	}

	err = h.LoadHoldings(AccountTest, "", nil)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("expected: %v but received: %v", asset.ErrNotSupported, err)
	}

	err = h.LoadHoldings(AccountTest, asset.Spot, nil)
	if !errors.Is(err, errSnapshotIsNil) {
		t.Fatalf("expected: %v but received: %v", errSnapshotIsNil, err)
	}

	values := HoldingsSnapshot{
		currency.BTC: {Total: 1},
		currency.LTC: {Total: 20},
	}

	err = h.LoadHoldings(AccountTest, asset.Spot, values)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	values = HoldingsSnapshot{
		currency.BTC: {Total: 2, Locked: 0.5},
		currency.XRP: {Total: 60000},
	}

	err = h.LoadHoldings(AccountTest, asset.Spot, values)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	btcHolding, err := h.GetHolding(AccountTest, asset.Spot, currency.BTC)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if btcHolding.GetFree() != 1.5 {
		t.Fatal("unexpected amounts received")
	}
}

func TestGetFullSnapshot(t *testing.T) {
	h := Holdings{}
	_, err := h.GetFullSnapshot()
	if !errors.Is(err, errAccountBalancesNotLoaded) {
		t.Fatalf("expected: %v but received: %v", errAccountBalancesNotLoaded, err)
	}

	h.funds = make(map[string]map[asset.Item]map[*currency.Item]*Holding)

	_, err = h.GetFullSnapshot()
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
}

func TestPublish(t *testing.T) {
	h := Holdings{}
	h.publish()
}

func TestGetHoldingsSnapshot(t *testing.T) {
	h := Holdings{}
	h.Verbose = true
	_, err := h.GetHoldingsSnapshot("", "")
	if !errors.Is(err, errAccountNameUnset) {
		t.Fatalf("expected: %v but received: %v", errAccountNameUnset, err)
	}

	_, err = h.GetHoldingsSnapshot(Default, "")
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("expected: %v but received: %v", asset.ErrNotSupported, err)
	}

	_, err = h.GetHoldingsSnapshot(Default, asset.Spot)
	if !errors.Is(err, errAccountBalancesNotLoaded) {
		t.Fatalf("expected: %v but received: %v", errAccountBalancesNotLoaded, err)
	}

	err = h.LoadHoldings(Default, asset.Spot, HoldingsSnapshot{
		currency.BTC: Balance{
			Total:  1337,
			Locked: 1,
		},
	})
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	_, err = h.GetHoldingsSnapshot("exchange", asset.Spot)
	if !errors.Is(err, errAccountNotFound) {
		t.Fatalf("expected: %v but received: %v", errAccountNotFound, err)
	}

	_, err = h.GetHoldingsSnapshot(Default, asset.Futures)
	if !errors.Is(err, errAssetTypeNotFound) {
		t.Fatalf("expected: %v but received: %v", errAssetTypeNotFound, err)
	}

	m, err := h.GetHoldingsSnapshot(Default, asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	for code, holdings := range m {
		if code.Item != currency.BTC.Item {
			t.Fatal("invalid code")
		}
		if holdings.Locked != 1 {
			t.Fatal("unexpected amount")
		}

		if holdings.Total != 1337 {
			t.Fatal("unexpected amount")
		}
	}
}

func TestAdjustHolding(t *testing.T) {
	h, err := DeployHoldings("someexchange", false)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	// Initial start with limit orders already present on exchange
	values := HoldingsSnapshot{
		currency.XRP: {Total: 40.5, Locked: 6},
	}

	h.Verbose = true

	err = h.LoadHoldings("someaccount", asset.Spot, values)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	err = h.AdjustByBalance("", "", currency.Code{}, 0)
	if !errors.Is(err, errAccountNameUnset) {
		t.Fatalf("expected: %v but received: %v", errAccountNameUnset, err)
	}

	err = h.AdjustByBalance("someaccount", "", currency.Code{}, 0)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("expected: %v but received: %v", asset.ErrNotSupported, err)
	}

	err = h.AdjustByBalance("someaccount", asset.Spot, currency.Code{}, 0)
	if !errors.Is(err, errCurrencyCodeEmpty) {
		t.Fatalf("expected: %v but received: %v", errCurrencyCodeEmpty, err)
	}

	err = h.AdjustByBalance("someaccount", asset.Spot, currency.XRP, 0)
	if !errors.Is(err, errAmountCannotBeLessOrEqualToZero) {
		t.Fatalf("expected: %v but received: %v",
			errAmountCannotBeLessOrEqualToZero,
			err)
	}

	err = h.AdjustByBalance("dummy", asset.Spot, currency.XRP, 1)
	if !errors.Is(err, errAccountNotFound) {
		t.Fatalf("expected: %v but received: %v", errAccountNotFound, err)
	}

	err = h.AdjustByBalance("someaccount", asset.Futures, currency.XRP, 1)
	if !errors.Is(err, errAssetTypeNotFound) {
		t.Fatalf("expected: %v but received: %v", errAssetTypeNotFound, err)
	}

	err = h.AdjustByBalance("someaccount", asset.Spot, currency.BTC, 1)
	if !errors.Is(err, errCurrencyItemNotFound) {
		t.Fatalf("expected: %v but received: %v", errCurrencyItemNotFound, err)
	}

	holding, err := h.GetHolding("someaccount", asset.Spot, currency.XRP)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(holding, 40.5, 6, 34.5, 0, 0, t)

	// Balance increased by one - limit order cancelled
	err = h.AdjustByBalance("someaccount", asset.Spot, currency.XRP, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(holding, 40.5, 5, 35.5, 0, 0, t)

	// limit/market order executed claim by algo or rpc
	claim, err := holding.Claim(1, true)
	if err != nil {
		t.Fatal(err)
	}
	checkValues(holding, 40.5, 5, 34.5, 0, 1, t)

	// limit/market order accepted by exchange
	err = claim.ReleaseToPending()
	if err != nil {
		t.Fatal(err)
	}
	checkValues(holding, 40.5, 5, 34.5, 1, 0, t)

	// simulate balance change on pending - does not mean it was matched
	// this demonstrates Poloniex balance flow
	err = h.AdjustByBalance("someaccount", asset.Spot, currency.XRP, -1)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(holding, 39.5, 4, 35.5, 0, 0, t)
}

func TestHoldingsClaim(t *testing.T) {
	h := Holdings{}
	_, err := h.Claim("", "", currency.Code{}, 0, false)
	if !errors.Is(err, errAccountNameUnset) {
		t.Fatalf("expected: %v but received: %v", errAccountNameUnset, err)
	}
	_, err = h.Claim("someaccount", "", currency.Code{}, 0, false)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("expected: %v but received: %v", asset.ErrNotSupported, err)
	}
	_, err = h.Claim("someaccount", asset.Spot, currency.Code{}, 0, false)
	if !errors.Is(err, errCurrencyCodeEmpty) {
		t.Fatalf("expected: %v but received: %v", errCurrencyCodeEmpty, err)
	}
	_, err = h.Claim("someaccount", asset.Spot, currency.BTC, 0, false)
	if !errors.Is(err, errAmountCannotBeLessOrEqualToZero) {
		t.Fatalf("expected: %v but received: %v", errAmountCannotBeLessOrEqualToZero, err)
	}
	_, err = h.Claim("someaccount", asset.Spot, currency.BTC, 1, false)
	if !errors.Is(err, errAccountNotFound) {
		t.Fatalf("expected: %v but received: %v", errAccountNotFound, err)
	}

	err = h.LoadHoldings("someaccount", asset.Spot, HoldingsSnapshot{
		currency.BTC: Balance{
			Total:  1,
			Locked: .2,
		},
	})

	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	h.Verbose = true

	_, err = h.Claim("someaccount", asset.Spot, currency.BTC, 1, true)
	if !errors.Is(err, errAmountExceedsHoldings) {
		t.Fatalf("expected: %v but received: %v", errAmountExceedsHoldings, err)
	}

	c, err := h.Claim("someaccount", asset.Spot, currency.BTC, 1, false)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if c.GetAmount() != .8 {
		t.Fatal("unexpected amount")
	}
}
