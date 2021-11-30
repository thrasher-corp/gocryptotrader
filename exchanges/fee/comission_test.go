package fee

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
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

func TestValidateCommission(t *testing.T) {
	c := Commission{}
	err := c.validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	c.Maker = 1
	err = c.validate()
	if !errors.Is(err, errMakerBiggerThanTaker) {
		t.Fatalf("received: %v but expected: %v", err, errMakerBiggerThanTaker)
	}

	c.Maker = 1
	c.Taker = 1
	err = c.validate()
	if !errors.Is(err, errMakerInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errMakerInvalid)
	}

	c.Maker = 0.002
	err = c.validate()
	if !errors.Is(err, errTakerInvalid) {
		t.Fatalf("received: %v but expected: %v", err, errTakerInvalid)
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

	com := &CommissionInternal{maker: decimal.NewFromFloat(0.02)}
	fee, err := com.CalculateMaker(100, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if fee != 2 {
		t.Fatal("unexpected value")
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

	com := &CommissionInternal{taker: decimal.NewFromFloat(0.05)}
	fee, err := com.CalculateTaker(100, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if fee != 5 {
		t.Fatal("unexpected value")
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

	com := &CommissionInternal{worstCaseMaker: decimal.NewFromFloat(0.02)}
	fee, err := com.CalculateWorstCaseMaker(100, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if fee != 2 {
		t.Fatal("unexpected value")
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

	com := &CommissionInternal{worstCaseTaker: decimal.NewFromFloat(0.05)}
	fee, err := com.CalculateWorstCaseTaker(100, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if fee != 5 {
		t.Fatal("unexpected value")
	}
}

func TestInternalGetMaker(t *testing.T) {
	fee, isSetAmount := (&CommissionInternal{maker: one}).GetMaker()
	if fee != 1 {
		t.Fatalf("received: %v but expected: %v", fee, 1)
	}

	if isSetAmount {
		t.Fatal("unexpected value")
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

func TestInternalGetTaker(t *testing.T) {
	fee, isSetAmount := (&CommissionInternal{taker: one}).GetTaker()
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

func TestLoad(t *testing.T) {
	c := &CommissionInternal{}
	c.load(1, 2)

	if !c.maker.Equal(decimal.NewFromInt(1)) {
		t.Fatal("unexpected result")
	}

	if !c.taker.Equal(decimal.NewFromInt(2)) {
		t.Fatal("unexpected result")
	}
}
