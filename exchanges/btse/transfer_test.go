package btse

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fee"
)

func TestWithdrawGetFees(t *testing.T) {
	val := getWithdrawal(currency.USD, standardRate, minimumUSDCharge, true)
	_, err := val.GetFee(context.Background(), 99, "", "")
	if !errors.Is(err, errBelowMinimumAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errBelowMinimumAmount)
	}

	feeOnAmount, err := val.GetFee(context.Background(), 102, "", "")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !feeOnAmount.Equal(minimumUSDCharge) {
		t.Fatalf("received: '%v' but expected: '%v'", feeOnAmount, minimumUSDCharge)
	}

	feeOnAmount, err = val.GetFee(context.Background(), 100000, "", "")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !feeOnAmount.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("received: '%v' but expected: '%v'", feeOnAmount, decimal.NewFromInt(100))
	}

	val = getWithdrawal(currency.EUR, standardRate, minimumUSDCharge, true)
	_, err = val.GetFee(context.Background(), 1, "", "")
	if !errors.Is(err, errBelowMinimumAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errBelowMinimumAmount)
	}

	// Test minimum charge using fx, this resultant will change depending on fx
	// rate
	_, err = val.GetFee(context.Background(), 100, "", "")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	feeOnAmount, err = val.GetFee(context.Background(), 100000, "", "")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// This should still be 100 EURO
	if !feeOnAmount.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("received: '%v' but expected: '%v'", feeOnAmount, decimal.NewFromInt(100))
	}

	val = getWithdrawal(currency.EUR, standardRate, decimal.NewFromInt(3), false)
	feeOnAmount, err = val.GetFee(context.Background(), 100, "", "")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !feeOnAmount.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("received: '%v' but expected: '%v'", feeOnAmount, decimal.NewFromInt(3))
	}
}

func TestWithdrawDisplay(t *testing.T) {
	val := getWithdrawal(currency.USD, standardRate, minimumUSDCharge, true)
	out, err := val.Display()
	if err != nil {
		t.Fatal(err)
	}
	var newVal TransferWithdrawalFee
	err = json.Unmarshal([]byte(out), &newVal)
	if err != nil {
		t.Fatal(err)
	}
	if !newVal.Code.Equal(currency.USD) {
		t.Fatal("unexpected currency")
	}
}

func TestWithdrawValidate(t *testing.T) {
	val := getWithdrawal(currency.Code{}, standardRate, minimumUSDCharge, true)
	err := val.Validate()
	if !errors.Is(err, errCurrencyCodeIsEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errCurrencyCodeIsEmpty)
	}

	val = getWithdrawal(currency.USD, decimal.NewFromInt(-1), minimumUSDCharge, true)
	err = val.Validate()
	if !errors.Is(err, errInvalidPercentageRate) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPercentageRate)
	}

	val = getWithdrawal(currency.USD, standardRate, decimal.NewFromInt(-1), true)
	err = val.Validate()
	if !errors.Is(err, errInvalidMinimumCharge) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidMinimumCharge)
	}

	val = getWithdrawal(currency.USD, standardRate, minimumUSDCharge, true)
	err = val.Validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestWithdrawLessThan(t *testing.T) {
	_, err := getDeposit(currency.Code{}).LessThan(nil)
	if !errors.Is(err, fee.ErrCannotCompare) {
		t.Fatalf("received: '%v' but expected: '%v'", err, fee.ErrCannotCompare)
	}
}

func TestDepositGetFee(t *testing.T) {
	val := getDeposit(currency.USD)
	feeOnAmount, err := val.GetFee(context.Background(), 100, "", "")
	if err != nil {
		t.Fatal(err)
	}

	if !feeOnAmount.Equal(decimal.Zero) {
		t.Fatalf("received: '%v' but expected: '%v'", feeOnAmount, decimal.Zero)
	}

	feeOnAmount, err = val.GetFee(context.Background(), 99, "", "")
	if err != nil {
		t.Fatal(err)
	}

	if !feeOnAmount.Equal(minimumDepositCharge) {
		t.Fatalf("received: '%v' but expected: '%v'", feeOnAmount, minimumDepositCharge)
	}

	val = getDeposit(currency.EUR)
	feeOnAmount, err = val.GetFee(context.Background(), 1000, "", "")
	if err != nil {
		t.Fatal(err)
	}

	if !feeOnAmount.Equal(decimal.Zero) {
		t.Fatalf("received: '%v' but expected: '%v'", feeOnAmount, decimal.Zero)
	}

	val = getDeposit(currency.EUR)
	feeOnAmount, err = val.GetFee(context.Background(), 1, "", "")
	if err != nil {
		t.Fatal(err)
	}

	if feeOnAmount.Equal(decimal.Zero) {
		t.Fatalf("received: '%v' but expected: '%v'", feeOnAmount, "a non zero value")
	}
}

func TestDepositDisplay(t *testing.T) {
	val := getDeposit(currency.EUR)
	out, err := val.Display()
	if err != nil {
		t.Fatal(err)
	}
	var newVal TransferDepositFee
	err = json.Unmarshal([]byte(out), &newVal)
	if err != nil {
		t.Fatal(err)
	}

	if !newVal.Code.Equal(currency.EUR) {
		t.Fatal("unexpected currency")
	}
}

func TestDepositValidate(t *testing.T) {
	val := getDeposit(currency.Code{})
	err := val.Validate()
	if !errors.Is(err, errCurrencyCodeIsEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errCurrencyCodeIsEmpty)
	}

	val = getDeposit(currency.DOGE)
	err = val.Validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestDepositLessThan(t *testing.T) {
	_, err := getDeposit(currency.Code{}).LessThan(nil)
	if !errors.Is(err, fee.ErrCannotCompare) {
		t.Fatalf("received: '%v' but expected: '%v'", err, fee.ErrCannotCompare)
	}
}
