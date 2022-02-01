package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
)

const (
	address  = "1F1tAaz5x1HUXrCNLbtMDqcw6o5GNn4xqX"
	bitStamp = "BITSTAMP"
)

func TestIsSynced(t *testing.T) {
	t.Parallel()
	var d DepositAddressManager
	if d.IsSynced() {
		t.Error("should be false")
	}
	m := SetupDepositAddressManager()
	err := m.Sync(map[string]map[currency.Code][]deposit.Address{
		bitStamp: {
			currency.BTC: []deposit.Address{
				{
					Address: address,
				},
			},
		},
	})
	if err != nil {
		t.Error(err)
	}
	if !m.IsSynced() {
		t.Error("should be synced")
	}
}

func TestSetupDepositAddressManager(t *testing.T) {
	t.Parallel()
	m := SetupDepositAddressManager()
	if m.store == nil {
		t.Fatal("expected store")
	}
}

func TestSync(t *testing.T) {
	t.Parallel()
	m := SetupDepositAddressManager()
	err := m.Sync(map[string]map[currency.Code][]deposit.Address{
		bitStamp: {
			currency.BTC: []deposit.Address{
				{
					Address: address,
				},
			},
		},
	})
	if err != nil {
		t.Error(err)
	}
	r, err := m.GetDepositAddressByExchangeAndCurrency(bitStamp, "", currency.BTC)
	if err != nil {
		t.Error("unexpected result")
	}
	if r.Address != address {
		t.Error("unexpected result")
	}

	m.store = nil
	err = m.Sync(map[string]map[currency.Code][]deposit.Address{
		bitStamp: {
			currency.BTC: []deposit.Address{
				{
					Address: address,
				},
			},
		},
	})
	if !errors.Is(err, ErrDepositAddressStoreIsNil) {
		t.Errorf("received %v, expected %v", err, ErrDepositAddressStoreIsNil)
	}

	m = nil
	err = m.Sync(map[string]map[currency.Code][]deposit.Address{
		bitStamp: {
			currency.BTC: []deposit.Address{
				{
					Address: address,
				},
			},
		},
	})
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("received %v, expected %v", err, ErrNilSubsystem)
	}
}

func TestGetDepositAddressByExchangeAndCurrency(t *testing.T) {
	t.Parallel()
	m := SetupDepositAddressManager()
	_, err := m.GetDepositAddressByExchangeAndCurrency("", "", currency.BTC)
	if !errors.Is(err, errExchangeNameIsEmpty) {
		t.Errorf("received %v, expected %v", err, errExchangeNameIsEmpty)
	}

	_, err = m.GetDepositAddressByExchangeAndCurrency("asdf", "", currency.Code{})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("received %v, expected %v", err, currency.ErrCurrencyCodeEmpty)
	}

	_, err = m.GetDepositAddressByExchangeAndCurrency("asdf", "", currency.USD)
	if !errors.Is(err, errIsNotCryptocurrency) {
		t.Errorf("received %v, expected %v", err, errIsNotCryptocurrency)
	}

	_, err = m.GetDepositAddressByExchangeAndCurrency("asdf", "", currency.BTC)
	if !errors.Is(err, ErrDepositAddressStoreIsNil) {
		t.Errorf("received %v, expected %v", err, ErrDepositAddressStoreIsNil)
	}

	m.store = map[string]map[*currency.Item][]deposit.Address{
		bitStamp: {
			currency.BTC.Item: []deposit.Address{
				{
					Address: address,
				},
			},
			currency.USDT.Item: []deposit.Address{
				{
					Address: "ABsdZ",
					Chain:   "SOL",
				},
				{
					Address: "0x1b",
					Chain:   "ERC20",
				},
				{
					Address: "1asdad",
					Chain:   "USDT",
				},
			},
			currency.BNB.Item: nil,
		},
	}

	_, err = m.GetDepositAddressByExchangeAndCurrency("asdf", "", currency.BTC)
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Errorf("received %v, expected %v", err, ErrExchangeNotFound)
	}
	_, err = m.GetDepositAddressByExchangeAndCurrency(bitStamp, "", currency.LTC)
	if !errors.Is(err, ErrDepositAddressNotFound) {
		t.Errorf("received %v, expected %v", err, ErrDepositAddressNotFound)
	}
	_, err = m.GetDepositAddressByExchangeAndCurrency(bitStamp, "", currency.BNB)
	if !errors.Is(err, ErrNoDepositAddressesRetrieved) {
		t.Errorf("received %v, expected %v", err, ErrNoDepositAddressesRetrieved)
	}
	_, err = m.GetDepositAddressByExchangeAndCurrency(bitStamp, "NON-EXISTENT-CHAIN", currency.USDT)
	if !errors.Is(err, errDepositAddressChainNotFound) {
		t.Errorf("received %v, expected %v", err, errDepositAddressChainNotFound)
	}

	if r, _ := m.GetDepositAddressByExchangeAndCurrency(bitStamp, "ErC20", currency.USDT); r.Address != "0x1b" && r.Chain != "ERC20" {
		t.Error("unexpected values")
	}
	if r, _ := m.GetDepositAddressByExchangeAndCurrency(bitStamp, "sOl", currency.USDT); r.Address != "ABsdZ" && r.Chain != "SOL" {
		t.Error("unexpected values")
	}
	if r, _ := m.GetDepositAddressByExchangeAndCurrency(bitStamp, "", currency.USDT); r.Address != "1asdad" && r.Chain != "USDT" {
		t.Error("unexpected values")
	}
	_, err = m.GetDepositAddressByExchangeAndCurrency(bitStamp, "", currency.BTC)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
}

func TestGetDepositAddressesByExchangeAndCurrency(t *testing.T) {
	t.Parallel()
	m := SetupDepositAddressManager()
	_, err := m.GetDepositAddressesByExchangeAndCurrency("", currency.BTC)
	if !errors.Is(err, errExchangeNameIsEmpty) {
		t.Errorf("received %v, expected %v", err, errExchangeNameIsEmpty)
	}

	_, err = m.GetDepositAddressesByExchangeAndCurrency("asdf", currency.Code{})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("received %v, expected %v", err, currency.ErrCurrencyCodeEmpty)
	}

	_, err = m.GetDepositAddressesByExchangeAndCurrency("asdf", currency.USD)
	if !errors.Is(err, errIsNotCryptocurrency) {
		t.Errorf("received %v, expected %v", err, errIsNotCryptocurrency)
	}

	_, err = m.GetDepositAddressesByExchangeAndCurrency("asdf", currency.BTC)
	if !errors.Is(err, ErrDepositAddressStoreIsNil) {
		t.Errorf("received %v, expected %v", err, ErrDepositAddressStoreIsNil)
	}

	m.store = map[string]map[*currency.Item][]deposit.Address{
		bitStamp: {
			currency.BTC.Item: []deposit.Address{
				{
					Address: address,
				},
			},
			currency.USDT.Item: []deposit.Address{
				{
					Address: "ABsdZ",
					Chain:   "SOL",
				},
				{
					Address: "0x1b",
					Chain:   "ERC20",
				},
				{
					Address: "1asdad",
					Chain:   "USDT",
				},
			},
			currency.BNB.Item: nil,
		},
	}

	_, err = m.GetDepositAddressesByExchangeAndCurrency("asdf", currency.BTC)
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Errorf("received %v, expected %v", err, ErrExchangeNotFound)
	}
	_, err = m.GetDepositAddressesByExchangeAndCurrency(bitStamp, currency.LTC)
	if !errors.Is(err, ErrDepositAddressNotFound) {
		t.Errorf("received %v, expected %v", err, ErrDepositAddressNotFound)
	}
	_, err = m.GetDepositAddressByExchangeAndCurrency(bitStamp, "", currency.BNB)
	if !errors.Is(err, ErrNoDepositAddressesRetrieved) {
		t.Errorf("received %v, expected %v", err, ErrNoDepositAddressesRetrieved)
	}
	addresses, err := m.GetDepositAddressesByExchangeAndCurrency(bitStamp, currency.USDT)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}

	if len(addresses) != 3 {
		t.Fatal("unexpected return")
	}
}

func TestGetDepositAddressesByExchange(t *testing.T) {
	t.Parallel()
	m := SetupDepositAddressManager()
	_, err := m.GetDepositAddressesByExchange("")
	if !errors.Is(err, ErrDepositAddressStoreIsNil) {
		t.Errorf("received %v, expected %v", err, ErrDepositAddressStoreIsNil)
	}

	m.store = map[string]map[*currency.Item][]deposit.Address{
		bitStamp: {
			currency.BTC.Item: []deposit.Address{
				{
					Address: address,
				},
			},
		},
	}
	_, err = m.GetDepositAddressesByExchange("non-existent")
	if !errors.Is(err, ErrDepositAddressNotFound) {
		t.Errorf("received %v, expected %v", err, ErrDepositAddressNotFound)
	}

	_, err = m.GetDepositAddressesByExchange(bitStamp)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
}

type FetchDepositAddressTester struct {
	exchange.IBotExchange
	RetryUntilFail bool
	RandomError    bool
}

func (f FetchDepositAddressTester) GetDepositAddress(context.Context, currency.Code, string, string) (*deposit.Address, error) {
	if f.RetryUntilFail {
		return nil, deposit.ErrAddressBeingCreated
	}
	if f.RandomError {
		return nil, errTestError
	}
	return &deposit.Address{Address: "OH WOOOOOOW"}, nil
}

func (f FetchDepositAddressTester) GetName() string {
	return "Super duper exchange"
}

func TestFetchDepositAddressWithRetry(t *testing.T) {
	t.Parallel()
	testExchange := &FetchDepositAddressTester{RetryUntilFail: true}
	_, err := FetchDepositAddressWithRetry(context.Background(),
		testExchange,
		currency.BTC,
		"2CHAINS",
		0,
		time.Nanosecond)
	if !errors.Is(err, errDepositAddressNotGenerated) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDepositAddressNotGenerated)
	}

	testExchange = &FetchDepositAddressTester{RandomError: true}
	_, err = FetchDepositAddressWithRetry(context.Background(),
		testExchange,
		currency.BTC,
		"2CHAINS",
		1,
		time.Nanosecond)
	if !errors.Is(err, errTestError) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errTestError)
	}

	testExchange = &FetchDepositAddressTester{}
	dep, err := FetchDepositAddressWithRetry(context.Background(),
		testExchange,
		currency.BTC,
		"2CHAINS",
		1,
		time.Nanosecond)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if dep.Address != "OH WOOOOOOW" {
		t.Fatal("unexpected address")
	}
}
