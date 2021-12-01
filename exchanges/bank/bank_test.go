package bank

import (
	"errors"
	"testing"
)

func TestString(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Value  Transfer
		Return string
	}{
		{NotApplicable, "NotApplicable"},
		{WireTransfer, "WireTransfer"},
		{PerfectMoney, "PerfectMoney"},
		{Neteller, "Neteller"},
		{AdvCash, "AdvCash"},
		{Payeer, "Payeer"},
		{Skrill, "Skrill"},
		{Simplex, "Simplex"},
		{SEPA, "SEPA"},
		{Swift, "Swift"},
		{RapidTransfer, "RapidTransfer"},
		{MisterTangoSEPA, "MisterTangoSEPA"},
		{Qiwi, "Qiwi"},
		{VisaMastercard, "VisaMastercard"},
		{WebMoney, "WebMoney"},
		{Capitalist, "Capitalist"},
		{WesternUnion, "WesternUnion"},
		{MoneyGram, "MoneyGram"},
		{Contact, "Contact"},
		{255, ""},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run("", func(t *testing.T) {
			t.Parallel()
			if tt.Value.String() != tt.Return {
				t.Fatalf("expected: %s but received: %s", tt.Value, tt.Return)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	err := Transfer(255).Validate()
	if !errors.Is(err, ErrUnknownTransfer) {
		t.Fatalf("received: %v but expected: %v", err, ErrUnknownTransfer)
	}
	err = NotApplicable.Validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}
