package account

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestGetAmount(t *testing.T) {
	c := Claim{}
	f := c.GetAmount()
	if f != 0 {
		t.Fatal("unexpected number")
	}

	df := c.getAmount()
	if !df.Equal(decimal.Zero) {
		t.Fatal("unexpected decimal")
	}

	c.amount = decimal.NewFromFloat(1337.1337)
	f = c.GetAmount()
	if f != 1337.1337 {
		t.Fatal("unexpected number")
	}

	if !c.amount.Equal(c.getAmount()) {
		t.Fatal("unexpected decimal")
	}
}

func TestGetTime(t *testing.T) {
	c := Claim{}
	tt := c.GetTime()
	if tt != (time.Time{}) {
		t.Fatal("unexpected time")
	}
	tn := time.Now()
	c.t = tn
	tt = c.GetTime()
	if !tt.Equal(tn) {
		t.Fatal("unexpected time")
	}
}

func TestRelease(t *testing.T) {
	holding := &Holding{}
	c := &Claim{
		h:      holding,
		amount: decimal.NewFromFloat(1),
	}
	err := c.Release()
	if !errors.Is(err, errUnableToReleaseClaim) {
		t.Fatalf("expected: %v, but received: %v", errUnableToReleaseClaim, err)
	}

	holding.claims = append(holding.claims, c)
	err = c.Release()
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v, but received: %v", nil, err)
	}
}

func TestReleaseToPending(t *testing.T) {
	holding := &Holding{}
	c := &Claim{
		amount: decimal.NewFromFloat(1),
		h:      holding,
	}
	err := c.ReleaseToPending()
	if !errors.Is(err, errUnableToReleaseClaim) {
		t.Fatalf("expected: %v, but received: %v", errUnableToReleaseClaim, err)
	}

	holding.claims = append(holding.claims, c)
	err = c.ReleaseToPending()
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v, but received: %v", nil, err)
	}

	if !holding.pending.Equal(decimal.NewFromFloat(1)) {
		t.Fatal("unexpected value:", holding.pending)
	}
}

func TestReleaseAndReduce(t *testing.T) {
	holding := &Holding{
		total: decimal.NewFromFloat(1),
	}
	c := &Claim{
		amount: decimal.NewFromFloat(1),
		h:      holding,
	}
	err := c.ReleaseAndReduce()
	if !errors.Is(err, errUnableToReduceClaim) {
		t.Fatalf("expected: %v, but received: %v", errUnableToReduceClaim, err)
	}

	holding.claims = append(holding.claims, c)
	err = c.ReleaseAndReduce()
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v, but received: %v", nil, err)
	}

	if !holding.total.Equal(decimal.Zero) {
		t.Fatal("unexpected value", holding.total)
	}
}

func TestHasClaim(t *testing.T) {
	holding := &Holding{
		total: decimal.NewFromFloat(1),
	}
	c := &Claim{
		amount: decimal.NewFromFloat(1),
		h:      holding,
	}
	if c.HasClaim() {
		t.Fatal("unexpected value")
	}

	holding.claims = append(holding.claims, c)
	if !c.HasClaim() {
		t.Fatal("unexpected value")
	}
}
