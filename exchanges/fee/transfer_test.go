package fee

import (
	"errors"
	"testing"
)

func TestTransferCalculate(t *testing.T) {
	_, err := (&transfer{}).calculate(one, 0)
	if !errors.Is(err, errAmountIsZero) {
		t.Fatalf("received: %v but expected: %v", err, errAmountIsZero)
	}

	v, err := (&transfer{Percentage: true}).calculate(one, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if v != 1 {
		t.Fatal("unexpected value")
	}
}
