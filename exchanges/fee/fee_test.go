package fee

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestLoad(t *testing.T) {
	t.Parallel()
	err := Load("", currency.EMPTYPAIR, 0, -100, -100)
	if !errors.Is(err, ErrExchangeNameEmpty) {
		t.Fatalf("received: %v, expected: %v", err, ErrExchangeNameEmpty)
	}

	err = Load("test", currency.EMPTYPAIR, 0, -100, -100)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: %v, expected: %v", err, currency.ErrCurrencyPairEmpty)
	}

	err = Load("test", currency.NewPair(currency.BTC, currency.USDT), 0, -100, -100)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v, expected: %v", err, asset.ErrNotSupported)
	}

	err = Load("test", currency.NewPair(currency.BTC, currency.USDT), asset.Spot, 100, -100)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRetrievePercentageRates(t *testing.T) {
	t.Parallel()
	_, err := RetrievePercentageRates("", currency.EMPTYPAIR, 0)
	if !errors.Is(err, ErrExchangeNameEmpty) {
		t.Fatalf("received: %v, expected: %v", err, ErrExchangeNameEmpty)
	}

	_, err = RetrievePercentageRates("test", currency.EMPTYPAIR, 0)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: %v, expected: %v", err, currency.ErrCurrencyPairEmpty)
	}

	_, err = RetrievePercentageRates("test", currency.NewPair(currency.BTC, currency.USD), 0)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v, expected: %v", err, asset.ErrNotSupported)
	}

	_, err = RetrievePercentageRates("test", currency.NewPair(currency.BTC, currency.USD), asset.Spot)
	if !errors.Is(err, ErrFeeRateNotFound) {
		t.Fatalf("received: %v, expected: %v", err, ErrFeeRateNotFound)
	}

	err = Load("test", currency.NewPair(currency.BTC, currency.USD), asset.Spot, 100, -100)
	if err != nil {
		t.Fatal(err)
	}

	rates, err := RetrievePercentageRates("test", currency.NewPair(currency.BTC, currency.USD), asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}

	if rates.Maker != 100 {
		t.Fatalf("received: %v, expected: %v", rates.Maker, 100)
	}

	if rates.Taker != -100 {
		t.Fatalf("received: %v, expected: %v", rates.Taker, -100)
	}
}
