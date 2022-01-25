package ftx

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestGetLoadableTransferFees(t *testing.T) {
	look, err := f.GetLoadableTransferFees()
	if err != nil {
		t.Fatal(err)
	}

	if len(look) == 0 {
		t.Fatal("unexpected amount")
	}
}

func TestUpdateCommissionFees(t *testing.T) {
	t.Parallel()
	err := f.UpdateCommissionFees(context.Background(), asset.CoinMarginedFutures)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	if !areTestAPIKeysSet() {
		t.Skip("credentials not set")
	}

	err = f.UpdateCommissionFees(context.Background(), asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestUpdateTransferFees(t *testing.T) {
	t.Parallel()
	err := f.UpdateTransferFees(context.Background())
	if !errors.Is(err, common.ErrNotYetImplemented) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNotYetImplemented)
	}
}

func TestGetAdhocFees(t *testing.T) {
	adhoc := GetAdhocFees(currency.Code{}, nil)
	err := adhoc.Validate()
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, currency.ErrCurrencyCodeEmpty)
	}

	adhoc = GetAdhocFees(currency.BTC, nil)
	err = adhoc.Validate()
	if !errors.Is(err, errExchangeNotSet) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeNotSet)
	}

	adhoc = GetAdhocFees(currency.BTC, &f)
	err = adhoc.Validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, err = (Adhoc{}).Display()
	if !errors.Is(err, errExchangeNotSet) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeNotSet)
	}

	returned, err := adhoc.Display()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	expected := "Currency: BTC using FTX method func(context.Context, currency.Code, float64, string, string) (ftx.WithdrawalFee, error)"
	if returned != expected {
		t.Fatalf("received: '%v' but expected: '%v'", returned, expected)
	}

	_, err = (Adhoc{}).GetFee(context.Background(), 10, "1F1tAaz5x1HUXrCNLbtMDqcw6o5GNn4xqX", "")
	if !errors.Is(err, errExchangeNotSet) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeNotSet)
	}

	if !areTestAPIKeysSet() {
		t.Skip("credentials not set")
	}

	withdrawal, err := adhoc.GetFee(context.Background(), 10, "1F1tAaz5x1HUXrCNLbtMDqcw6o5GNn4xqX", "")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if withdrawal.LessThanOrEqual(decimal.Zero) {
		t.Fatal("expected value higher than zero")
	}
}
