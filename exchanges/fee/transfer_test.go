package fee

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

func TestTransferCalculate(t *testing.T) {
	_, err := (&transfer{}).calculate(nil, 0)
	if !errors.Is(err, errAmountIsZero) {
		t.Fatalf("received: %v but expected: %v", err, errAmountIsZero)
	}

	v, err := (&transfer{Percentage: true}).calculate(Standard{Decimal: decimal.NewFromInt(1)}, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if v != 1 {
		t.Fatal("unexpected value")
	}
}
