package fee

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var identity = decimal.NewFromInt(1)

func TestGetManager(t *testing.T) {
	t.Parallel()
	if GetManager() == nil {
		t.Fatal("manager cannot be nil")
	}
}

func TestRegisterFeeDefinitions(t *testing.T) {
	t.Parallel()
	_, err := RegisterFeeDefinitions("")
	if !errors.Is(err, errExchangeNameIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errExchangeNameIsEmpty)
	}

	d, err := RegisterFeeDefinitions("moo")
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if d == nil {
		t.Fatal("definitions should not be nil")
	}
}

func TestManagerRegister(t *testing.T) {
	t.Parallel()
	man := &Manager{}
	err := man.Register("", nil)
	if !errors.Is(err, errExchangeNameIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errExchangeNameIsEmpty)
	}

	err = man.Register("bruh", nil)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	err = man.Register("bruh", &Definitions{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	err = man.Register("bruh", &Definitions{})
	if !errors.Is(err, errFeeDefinitionsAlreadyLoaded) {
		t.Fatalf("received: %v but expected: %v", err, errFeeDefinitionsAlreadyLoaded)
	}
}

func TestLoadDynamic(t *testing.T) {
	t.Parallel()
	err := (*Definitions)(nil).LoadDynamic(0, 0)
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	err = (&Definitions{}).LoadDynamic(1, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestLoadStatic(t *testing.T) {
	//...
}

func TestGetMakerTotal(t *testing.T) {
	t.Parallel()
	d := &Definitions{online: Global{Maker: identity}}
	_, err := d.GetMakerTotal(0, 0)
	if !errors.Is(err, errPriceIsZero) {
		t.Fatalf("received: %v but expected: %v", err, errPriceIsZero)
	}

	_, err = d.GetMakerTotal(1, 0)
	if !errors.Is(err, errAmountIsZero) {
		t.Fatalf("received: %v but expected: %v", err, errAmountIsZero)
	}

	_, err = d.GetMakerTotal(1, 1)
	if !errors.Is(err, errNotRatio) {
		t.Fatalf("received: %v but expected: %v", err, errNotRatio)
	}

	d = &Definitions{online: Global{Maker: decimal.Zero}}

	val, err := d.GetMakerTotal(50000, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if val != 0 {
		t.Fatalf("received: %v but expected %v", val, 0)
	}

	d = &Definitions{online: Global{Maker: decimal.NewFromFloat(0.01)}}
	val, err = d.GetMakerTotal(50000, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if val != 500 {
		t.Fatalf("received: %v but expected %v", val, 500)
	}
}

func TestGetMaker(t *testing.T) {
	fee, ratio := (&Definitions{online: Global{Maker: identity}}).GetMaker()
	if !ratio {
		t.Fatal("unexpected, should be ratio")
	}
	if fee != 1 {
		t.Fatal("unexpected maker value")
	}
}

func TestGetTakerTotal(t *testing.T) {
	t.Parallel()
	d := &Definitions{online: Global{Taker: identity}}
	_, err := d.GetTakerTotal(0, 0)
	if !errors.Is(err, errPriceIsZero) {
		t.Fatalf("received: %v but expected: %v", err, errPriceIsZero)
	}

	_, err = d.GetTakerTotal(1, 0)
	if !errors.Is(err, errAmountIsZero) {
		t.Fatalf("received: %v but expected: %v", err, errAmountIsZero)
	}

	_, err = d.GetTakerTotal(1, 1)
	if !errors.Is(err, errNotRatio) {
		t.Fatalf("received: %v but expected: %v", err, errNotRatio)
	}

	d = &Definitions{online: Global{Taker: decimal.Zero}}
	val, err := d.GetTakerTotal(50000, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if val != 0 {
		t.Fatalf("received: %v but expected %v", val, 0)
	}

	d = &Definitions{online: Global{Taker: decimal.NewFromFloat(0.01)}}
	val, err = d.GetTakerTotal(50000, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if val != 500 {
		t.Fatalf("received: %v but expected %v", val, 500)
	}
}

func TestGetTaker(t *testing.T) {
	fee, ratio := (&Definitions{online: Global{Taker: identity}}).GetTaker()
	if !ratio {
		t.Fatal("unexpected, should be ratio")
	}
	if fee != 1 {
		t.Fatal("unexpected maker value")
	}
}

func TestGetDeposit(t *testing.T) {
	_, _, err := (&Definitions{}).GetDeposit(currency.Code{}, "")
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}

	_, _, err = (&Definitions{}).GetDeposit(currency.BTC, "")
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v but expected: %v", err, asset.ErrNotSupported)
	}

	_, _, err = (&Definitions{}).GetDeposit(currency.BTC, asset.Spot)
	if !errors.Is(err, errTransferFeeNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errTransferFeeNotFound)
	}

	d := &Definitions{transfer: map[asset.Item]map[*currency.Item]*transfer{
		asset.Spot: {
			currency.BTC.Item: &transfer{Deposit: identity},
		},
	}}

	fee, ratio, err := d.GetDeposit(currency.BTC, asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if !ratio {
		t.Fatal("unexpected ratio value")
	}

	if fee != 1 {
		t.Fatal("unexpected fee value")
	}
}

func TestGetWithdrawal(t *testing.T) {
	_, _, err := (&Definitions{}).GetWithdrawal(currency.Code{}, "")
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}

	_, _, err = (&Definitions{}).GetWithdrawal(currency.BTC, "")
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v but expected: %v", err, asset.ErrNotSupported)
	}

	_, _, err = (&Definitions{}).GetWithdrawal(currency.BTC, asset.Spot)
	if !errors.Is(err, errTransferFeeNotFound) {
		t.Fatalf("received: %v but expected: %v", err, errTransferFeeNotFound)
	}

	d := &Definitions{transfer: map[asset.Item]map[*currency.Item]*transfer{
		asset.Spot: {
			currency.BTC.Item: &transfer{Withdrawal: identity},
		},
	}}

	fee, ratio, err := d.GetWithdrawal(currency.BTC, asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if !ratio {
		t.Fatal("unexpected ratio value")
	}

	if fee != 1 {
		t.Fatal("unexpected fee value")
	}
}

func TestGetAllFees(t *testing.T) {
	_, err := (*Definitions)(nil).GetAllFees()
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}

	d := Definitions{}
	d.LoadStatic(Options{})

	_, err = (*Definitions)(nil).GetAllFees()
	if !errors.Is(err, ErrDefinitionsAreNil) {
		t.Fatalf("received: %v but expected: %v", err, ErrDefinitionsAreNil)
	}
}
