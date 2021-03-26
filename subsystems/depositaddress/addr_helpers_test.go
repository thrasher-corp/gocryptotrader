package depositaddress

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"
)

const (
	testBTCAddress = "1F1tAaz5x1HUXrCNLbtMDqcw6o5GNn4xqX"
)

func TestSeed(t *testing.T) {
	var d DepositAddressStore
	u := map[string]map[string]string{
		"BITSTAMP": {
			"BTC": testBTCAddress,
		},
	}

	d.Seed(u)
	r, err := d.GetDepositAddress("BITSTAMP", currency.BTC)
	if err != nil {
		t.Error("unexpected result")
	}

	if r != testBTCAddress {
		t.Error("unexpected result")
	}
}

func TestGetDepositAddress(t *testing.T) {
	var d DepositAddressStore
	_, err := d.GetDepositAddress("", currency.BTC)
	if err != ErrDepositAddressStoreIsNil {
		t.Error("non-error on non-existent exchange")
	}

	d.Store = map[string]map[string]string{
		"BITSTAMP": {
			"BTC": testBTCAddress,
		},
	}

	_, err = d.GetDepositAddress("", currency.BTC)
	if err != exchangemanager.ErrExchangeNotFound {
		t.Error("non-error on non-existent exchange")
	}

	var r string
	r, err = d.GetDepositAddress("BiTStAmP", currency.NewCode("bTC"))
	if err != nil {
		t.Error("unexpected err: ", err)
	}

	if r != testBTCAddress {
		t.Error("unexpected BTC address: ", r)
	}

	_, err = d.GetDepositAddress("BiTStAmP", currency.LTC)
	if err != ErrDepositAddressNotFound {
		t.Error("unexpected err: ", err)
	}
}
