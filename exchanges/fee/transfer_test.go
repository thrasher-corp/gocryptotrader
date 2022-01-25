package fee

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestConvert(t *testing.T) {
	t.Parallel()
	tr := Transfer{
		Deposit:        Convert(1),
		MinimumDeposit: Convert(2),
		MaximumDeposit: Convert(3),

		Withdrawal:        Convert(4),
		MinimumWithdrawal: Convert(5),
		MaximumWithdrawal: Convert(6),
	}

	internal := tr.convert()

	if !internal.DepositEnabled {
		t.Fatal("should be enabled")
	}

	fee, err := internal.Deposit.GetFee(context.Background(), 1, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !fee.Equal(one) {
		t.Fatal("unexpected value")
	}

	fee, err = internal.MinimumDeposit.GetFee(context.Background(), 1, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !fee.Equal(two) {
		t.Fatal("unexpected value")
	}

	fee, err = internal.MaximumDeposit.GetFee(context.Background(), 1, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !fee.Equal(decimal.NewFromInt(3)) {
		t.Fatal("unexpected value")
	}

	if !internal.WithdrawalEnabled {
		t.Fatal("should be enabled")
	}

	fee, err = internal.Withdrawal.GetFee(context.Background(), 1, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !fee.Equal(decimal.NewFromInt(4)) {
		t.Fatal("unexpected value")
	}

	fee, err = internal.MinimumWithdrawal.GetFee(context.Background(), 1, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !fee.Equal(decimal.NewFromInt(5)) {
		t.Fatal("unexpected value")
	}

	fee, err = internal.MaximumWithdrawal.GetFee(context.Background(), 1, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !fee.Equal(decimal.NewFromInt(6)) {
		t.Fatal("unexpected value")
	}
}

var errTest = errors.New("error test")

type validateError struct{}

func (g *validateError) GetFee(ctx context.Context, amount float64, destinationAddress, tag string) (decimal.Decimal, error) {
	return decimal.Zero, nil
}
func (g *validateError) Display() (string, error)         { return "", nil }
func (g *validateError) Validate() error                  { return errTest }
func (g *validateError) LessThan(val Value) (bool, error) { return false, nil }

type lessThanError struct {
	Wow bool
}

func (g *lessThanError) GetFee(ctx context.Context, amount float64, destinationAddress, tag string) (decimal.Decimal, error) {
	return decimal.Zero, nil
}
func (g *lessThanError) Display() (string, error) { return "", nil }
func (g *lessThanError) Validate() error          { return nil }
func (g *lessThanError) LessThan(val Value) (bool, error) {
	if !g.Wow {
		return false, errTest
	}
	return g.Wow, nil
}

func TestValidate(t *testing.T) {
	t.Parallel()
	tr := Transfer{}
	err := tr.validate()
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCurrencyIsEmpty)
	}

	tr.Currency = currency.BTC3L
	tr.Deposit = &validateError{}
	err = tr.validate()
	if !errors.Is(err, errTest) {
		t.Fatalf("received: %v but expected: %v", err, errTest)
	}

	tr.MaximumDeposit = &validateError{}
	tr.Deposit = Convert(1)
	err = tr.validate()
	if !errors.Is(err, errTest) {
		t.Fatalf("received: %v but expected: %v", err, errTest)
	}

	tr.MaximumDeposit = Convert(1)
	tr.MinimumDeposit = &validateError{}
	err = tr.validate()
	if !errors.Is(err, errTest) {
		t.Fatalf("received: %v but expected: %v", err, errTest)
	}

	tr.MinimumDeposit = Convert(1)
	tr.MaximumDeposit = &lessThanError{}
	err = tr.validate()
	if !errors.Is(err, errTest) {
		t.Fatalf("received: %v but expected: %v", err, errTest)
	}

	tr.MaximumDeposit = &lessThanError{Wow: true}
	err = tr.validate()
	if !errors.Is(err, errMaxLessThanMin) {
		t.Fatalf("received: %v but expected: %v", err, errMaxLessThanMin)
	}

	tr.MaximumDeposit = Convert(1)
	tr.Withdrawal = &validateError{}
	err = tr.validate()
	if !errors.Is(err, errTest) {
		t.Fatalf("received: %v but expected: %v", err, errTest)
	}

	tr.MaximumWithdrawal = &validateError{}
	tr.Withdrawal = Convert(1)
	err = tr.validate()
	if !errors.Is(err, errTest) {
		t.Fatalf("received: %v but expected: %v", err, errTest)
	}

	tr.MaximumWithdrawal = Convert(1)
	tr.MinimumWithdrawal = &validateError{}
	err = tr.validate()
	if !errors.Is(err, errTest) {
		t.Fatalf("received: %v but expected: %v", err, errTest)
	}

	tr.MinimumWithdrawal = Convert(1)
	tr.MaximumWithdrawal = &lessThanError{}
	err = tr.validate()
	if !errors.Is(err, errTest) {
		t.Fatalf("received: %v but expected: %v", err, errTest)
	}

	tr.MaximumWithdrawal = &lessThanError{Wow: true}
	err = tr.validate()
	if !errors.Is(err, errMaxLessThanMin) {
		t.Fatalf("received: %v but expected: %v", err, errMaxLessThanMin)
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	var tr *transfer
	err := tr.update(&Transfer{})
	if !errors.Is(err, errTransferIsNil) {
		t.Fatalf("received: %v but expected: %v", err, errTransferIsNil)
	}

	tr = new(transfer)
	err = tr.update(nil)
	if !errors.Is(err, errTransferIsNil) {
		t.Fatalf("received: %v but expected: %v", err, errTransferIsNil)
	}
	incoming := &Transfer{
		Deposit:        Convert(1),
		MinimumDeposit: Convert(2),
		MaximumDeposit: Convert(3),

		Withdrawal:        Convert(4),
		MinimumWithdrawal: Convert(5),
		MaximumWithdrawal: Convert(6),
	}

	err = tr.update(incoming)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if !tr.DepositEnabled {
		t.Fatal("should be enabled")
	}

	fee, err := tr.Deposit.GetFee(context.Background(), 1, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !fee.Equal(one) {
		t.Fatal("unexpected value")
	}

	fee, err = tr.MinimumDeposit.GetFee(context.Background(), 1, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !fee.Equal(two) {
		t.Fatal("unexpected value")
	}

	fee, err = tr.MaximumDeposit.GetFee(context.Background(), 1, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !fee.Equal(decimal.NewFromInt(3)) {
		t.Fatal("unexpected value")
	}

	if !tr.WithdrawalEnabled {
		t.Fatal("should be enabled")
	}

	fee, err = tr.Withdrawal.GetFee(context.Background(), 1, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !fee.Equal(decimal.NewFromInt(4)) {
		t.Fatal("unexpected value")
	}

	fee, err = tr.MinimumWithdrawal.GetFee(context.Background(), 1, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !fee.Equal(decimal.NewFromInt(5)) {
		t.Fatal("unexpected value")
	}

	fee, err = tr.MaximumWithdrawal.GetFee(context.Background(), 1, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !fee.Equal(decimal.NewFromInt(6)) {
		t.Fatal("unexpected value")
	}

	err = tr.update(&Transfer{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if tr.DepositEnabled {
		t.Fatal("should not be operational")
	}

	if tr.Deposit != nil {
		t.Fatal("unexpected value")
	}

	if tr.MaximumDeposit != nil {
		t.Fatal("unexpected value")
	}

	if tr.MinimumDeposit != nil {
		t.Fatal("unexpected value")
	}

	if tr.WithdrawalEnabled {
		t.Fatal("should not be operational")
	}

	if tr.Withdrawal != nil {
		t.Fatal("unexpected value")
	}

	if tr.MaximumWithdrawal != nil {
		t.Fatal("unexpected value")
	}

	if tr.MinimumWithdrawal != nil {
		t.Fatal("unexpected value")
	}
}

type getFeeError struct{}

func (g *getFeeError) GetFee(ctx context.Context, amount float64, destinationAddress, tag string) (decimal.Decimal, error) {
	return decimal.Zero, errTest
}
func (g *getFeeError) Display() (string, error)         { return "", nil }
func (g *getFeeError) Validate() error                  { return nil }
func (g *getFeeError) LessThan(val Value) (bool, error) { return false, nil }

func TestTransferCalculate(t *testing.T) {
	t.Parallel()
	_, err := (&transfer{}).calculate(nil, 0)
	if !errors.Is(err, errAmountIsZero) {
		t.Fatalf("received: %v but expected: %v", err, errAmountIsZero)
	}

	_, err = (&transfer{}).calculate(&getFeeError{}, 1)
	if !errors.Is(err, errTest) {
		t.Fatalf("received: %v but expected: %v", err, errTest)
	}

	v, err := (&transfer{Percentage: true}).calculate(Standard{Decimal: decimal.NewFromInt(1)}, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if v != 1 {
		t.Fatal("unexpected value")
	}
}
