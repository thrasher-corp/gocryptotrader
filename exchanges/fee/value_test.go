package fee

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

func TestValueConvert(t *testing.T) {
	val := Convert(-1)
	fee, err := val.GetFee(1)
	if err != nil {
		t.Fatal(err)
	}

	if !fee.Equal(decimal.NewFromInt(-1)) {
		t.Fatal("unexpected result")
	}

	display, err := val.Display()
	if err != nil {
		t.Fatal(err)
	}

	if display != "-1" {
		t.Fatal("unexpected value:", display)
	}

	err = val.Validate()
	if !errors.Is(err, errInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errInvalid)
	}

	_, err = val.LessThan(&getFeeError{})
	if !errors.Is(err, errCannotCompare) {
		t.Fatalf("received: %v but expected: %v", err, errCannotCompare)
	}
}

func TestValueConvertWithAmount(t *testing.T) {
	val := ConvertWithAmount(0.005, 0.002, 1)
	fee, err := val.GetFee(.9)
	if err != nil {
		t.Fatal(err)
	}

	if !fee.Equal(decimal.NewFromFloat(0.005)) {
		t.Fatal("unexpected result:", fee)
	}

	fee, err = val.GetFee(1.5)
	if err != nil {
		t.Fatal(err)
	}

	if !fee.Equal(decimal.NewFromFloat(0.002)) {
		t.Fatal("unexpected result:", fee)
	}

	display, err := val.Display()
	if err != nil {
		t.Fatal(err)
	}

	if display != `{"feeWhenLower":"0.005","feeWhenHigherOrEqual":"0.002","amount":"1"}` {
		t.Fatal("unexpected value:", display)
	}

	err = val.Validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	val = ConvertWithAmount(-1, -1, -1)
	err = val.Validate()
	if !errors.Is(err, errInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errInvalid)
	}

	val = ConvertWithAmount(1, -1, -1)
	err = val.Validate()
	if !errors.Is(err, errInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errInvalid)
	}

	val = ConvertWithAmount(1, .5, -1)
	err = val.Validate()
	if !errors.Is(err, errInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errInvalid)
	}

	_, err = val.LessThan(&getFeeError{})
	if !errors.Is(err, errCannotCompare) {
		t.Fatalf("received: %v but expected: %v", err, errCannotCompare)
	}
}

func TestValueConvertBlockchain(t *testing.T) {
	val := ConvertBlockchain("BTC")
	_, err := val.GetFee(1)
	if err != nil {
		t.Fatal(err)
	}

	display, err := val.Display()
	if err != nil {
		t.Fatal(err)
	}

	if display != "current fees are BTC blockchain transaction fees" {
		t.Fatal("unexpected value:", display)
	}

	err = val.Validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	_, err = val.LessThan(&getFeeError{})
	if !errors.Is(err, errCannotCompare) {
		t.Fatalf("received: %v but expected: %v", err, errCannotCompare)
	}

	val = ConvertBlockchain("")
	err = val.Validate()
	if !errors.Is(err, errBlockchainEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errBlockchainEmpty)
	}
}

func TestValueConvertWithMaxAndMin(t *testing.T) {
	val := ConvertWithMaxAndMin(1, 100, 20)
	fee, err := val.GetFee(.5)
	if err != nil {
		t.Fatal(err)
	}

	if !fee.Equal(decimal.NewFromInt(20)) {
		t.Fatal("unexpected result")
	}

	fee, err = val.GetFee(120)
	if err != nil {
		t.Fatal(err)
	}

	if !fee.Equal(decimal.NewFromInt(100)) {
		t.Fatal("unexpected result")
	}

	fee, err = val.GetFee(60)
	if err != nil {
		t.Fatal(err)
	}

	if !fee.Equal(decimal.NewFromInt(60)) {
		t.Fatal("unexpected result")
	}

	display, err := val.Display()
	if err != nil {
		t.Fatal(err)
	}

	if display != `{"minimumFee":"20","maximumFee":"100","fee":"1"}` {
		t.Fatal("unexpected value:", display)
	}

	err = val.Validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	_, err = val.LessThan(&getFeeError{})
	if !errors.Is(err, errCannotCompare) {
		t.Fatalf("received: %v but expected: %v", err, errCannotCompare)
	}

	val = ConvertWithMaxAndMin(-1, 100, 20)
	err = val.Validate()
	if !errors.Is(err, errInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errInvalid)
	}

	val = ConvertWithMaxAndMin(1, -100, 20)
	err = val.Validate()
	if !errors.Is(err, errInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errInvalid)
	}

	val = ConvertWithMaxAndMin(1, 100, -20)
	err = val.Validate()
	if !errors.Is(err, errInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errInvalid)
	}
}

func TestValueConvertConvertWithMinimumAmount(t *testing.T) {
	val := ConvertWithMinimumAmount(1, 5)
	fee, err := val.GetFee(5)
	if err != nil {
		t.Fatal(err)
	}

	if !fee.Equal(decimal.NewFromInt(1)) {
		t.Fatal("unexpected result")
	}

	_, err = val.GetFee(4.9)
	if !errors.Is(err, errAmountIsLessThanMinimumRequired) {
		t.Fatalf("received: %v but expected: %v", err, errAmountIsLessThanMinimumRequired)
	}

	display, err := val.Display()
	if err != nil {
		t.Fatal(err)
	}

	if display != `{"withMinimumAmount":"5","fee":"1"}` {
		t.Fatal("unexpected value:", display)
	}

	err = val.Validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	_, err = val.LessThan(&getFeeError{})
	if !errors.Is(err, errCannotCompare) {
		t.Fatalf("received: %v but expected: %v", err, errCannotCompare)
	}

	val = ConvertWithMinimumAmount(-1, 5)
	err = val.Validate()
	if !errors.Is(err, errInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errInvalid)
	}

	val = ConvertWithMinimumAmount(1, -5)
	err = val.Validate()
	if !errors.Is(err, errInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errInvalid)
	}
}
