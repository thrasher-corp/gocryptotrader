package engine

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

const (
	address  = "1F1tAaz5x1HUXrCNLbtMDqcw6o5GNn4xqX"
	bitStamp = "BITSTAMP"
	btc      = "BTC"
)

func TestSetupDepositAddressManager(t *testing.T) {
	m := SetupDepositAddressManager()
	if m.store == nil {
		t.Fatal("expected store")
	}
}

func TestSync(t *testing.T) {
	m := SetupDepositAddressManager()
	err := m.Sync(map[string]map[string]string{
		bitStamp: {
			btc: address,
		},
	})
	if err != nil {
		t.Error(err)
	}
	r, err := m.GetDepositAddressByExchangeAndCurrency(bitStamp, currency.BTC)
	if err != nil {
		t.Error("unexpected result")
	}
	if r != address {
		t.Error("unexpected result")
	}

	m.store = nil
	err = m.Sync(map[string]map[string]string{
		bitStamp: {
			btc: address,
		},
	})
	if !errors.Is(err, ErrDepositAddressStoreIsNil) {
		t.Errorf("received %v, expected %v", err, ErrDepositAddressStoreIsNil)
	}

	m = nil
	err = m.Sync(map[string]map[string]string{
		bitStamp: {
			btc: address,
		},
	})
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("received %v, expected %v", err, ErrNilSubsystem)
	}
}

func TestGetDepositAddressByExchangeAndCurrency(t *testing.T) {
	m := SetupDepositAddressManager()
	_, err := m.GetDepositAddressByExchangeAndCurrency("", currency.BTC)
	if !errors.Is(err, ErrDepositAddressStoreIsNil) {
		t.Errorf("received %v, expected %v", err, ErrDepositAddressStoreIsNil)
	}

	m.store = map[string]map[string]string{
		bitStamp: {
			btc: address,
		},
	}
	_, err = m.GetDepositAddressByExchangeAndCurrency(bitStamp, currency.BTC)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
}

func TestGetDepositAddressesByExchange(t *testing.T) {
	m := SetupDepositAddressManager()
	_, err := m.GetDepositAddressesByExchange("")
	if !errors.Is(err, ErrDepositAddressStoreIsNil) {
		t.Errorf("received %v, expected %v", err, ErrDepositAddressStoreIsNil)
	}

	m.store = map[string]map[string]string{
		bitStamp: {
			btc: address,
		},
	}
	_, err = m.GetDepositAddressesByExchange(bitStamp)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
}
