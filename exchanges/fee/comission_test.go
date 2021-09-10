package fee

import (
	"errors"
	"testing"
)

func TestExternalInternalConvert(t *testing.T) {
	c := Commission{
		IsSetAmount: true,
		Maker:       1,
		Taker:       2,
	}

	cInternal := c.convert()
	if cInternal == nil {
		t.Fatal("should not be nil")
	}
	if !c.IsSetAmount {
		t.Fatal("unexpected value")
	}
	if !cInternal.maker.Equal(one) {
		t.Fatal("unexpected value")
	}
	if !cInternal.taker.Equal(two) {
		t.Fatal("unexpected value")
	}
	if !cInternal.worstCaseMaker.Equal(one) {
		t.Fatal("unexpected value")
	}
	if !cInternal.worstCaseTaker.Equal(two) {
		t.Fatal("unexpected value")
	}

	c = cInternal.convert()

	if !c.IsSetAmount {
		t.Fatal("unexpected value")
	}
	if c.Maker != 1 {
		t.Fatal("unexpected value")
	}
	if c.Taker != 2 {
		t.Fatal("unexpected value")
	}
	if c.WorstCaseMaker != 1 {
		t.Fatal("unexpected value")
	}
	if c.WorstCaseTaker != 2 {
		t.Fatal("unexpected value")
	}
}

func TestInternalCalculateMaker(t *testing.T) {
	_, err := (*CommissionInternal)(nil).CalculateMaker(0, 0)
	if !errors.Is(err, errPriceIsZero) {
		t.Fatalf("received: %v but expected: %v", err, errPriceIsZero)
	}

	_, err = (*CommissionInternal)(nil).CalculateMaker(1, 0)
	if !errors.Is(err, errAmountIsZero) {
		t.Fatalf("received: %v but expected: %v", err, errAmountIsZero)
	}
}

func TestInternalCalculateTaker(t *testing.T) {
	_, err := (*CommissionInternal)(nil).CalculateTaker(0, 0)
	if !errors.Is(err, errPriceIsZero) {
		t.Fatalf("received: %v but expected: %v", err, errPriceIsZero)
	}

	_, err = (*CommissionInternal)(nil).CalculateTaker(1, 0)
	if !errors.Is(err, errAmountIsZero) {
		t.Fatalf("received: %v but expected: %v", err, errAmountIsZero)
	}
}

func TestInternalCalculateWorstCaseMaker(t *testing.T) {
	_, err := (*CommissionInternal)(nil).CalculateWorstCaseMaker(0, 0)
	if !errors.Is(err, errPriceIsZero) {
		t.Fatalf("received: %v but expected: %v", err, errPriceIsZero)
	}

	_, err = (*CommissionInternal)(nil).CalculateWorstCaseMaker(1, 0)
	if !errors.Is(err, errAmountIsZero) {
		t.Fatalf("received: %v but expected: %v", err, errAmountIsZero)
	}
}

func TestInternalCalculateWorstCaseTaker(t *testing.T) {
	_, err := (*CommissionInternal)(nil).CalculateWorstCaseTaker(0, 0)
	if !errors.Is(err, errPriceIsZero) {
		t.Fatalf("received: %v but expected: %v", err, errPriceIsZero)
	}

	_, err = (*CommissionInternal)(nil).CalculateWorstCaseTaker(1, 0)
	if !errors.Is(err, errAmountIsZero) {
		t.Fatalf("received: %v but expected: %v", err, errAmountIsZero)
	}
}

func TestInternalGetWorstCaseMaker(t *testing.T) {
	fee, isSetAmount := (&CommissionInternal{worstCaseMaker: one}).GetWorstCaseMaker()
	if fee != 1 {
		t.Fatalf("received: %v but expected: %v", fee, 1)
	}

	if isSetAmount {
		t.Fatal("unexpected value")
	}
}

func TestInternalGetWorstCaseTaker(t *testing.T) {
	fee, isSetAmount := (&CommissionInternal{worstCaseTaker: one}).GetWorstCaseTaker()
	if fee != 1 {
		t.Fatalf("received: %v but expected: %v", fee, 1)
	}

	if isSetAmount {
		t.Fatal("unexpected value")
	}
}

func TestInternalSet(t *testing.T) {
	err := (&CommissionInternal{}).set(0, 0, true)
	if !errors.Is(err, errFeeTypeMismatch) {
		t.Fatalf("received: %v but expected: %v", err, errFeeTypeMismatch)
	}

	err = (&CommissionInternal{}).set(0, 0, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestInternalCalculate(t *testing.T) {
	v, err := (&CommissionInternal{setAmount: true}).calculate(two, 50000, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if v != 2 {
		t.Fatal("unexpected value")
	}

	v, err = (&CommissionInternal{}).calculate(one, 50000, 0.01)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if v != 500 {
		t.Fatal("unexpected value")
	}
}
