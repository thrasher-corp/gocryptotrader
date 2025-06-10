package funding

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestMatchesExchange(t *testing.T) {
	t.Parallel()
	i := Item{}
	if i.MatchesExchange(nil) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
	baseItem, err := CreateItem(exchName, a, pair.Base, decimal.Zero, decimal.Zero)
	assert.NoError(t, err)

	quoteItem, err := CreateItem(exchName, a, pair.Quote, elite, decimal.Zero)
	assert.NoError(t, err)

	if !baseItem.MatchesExchange(quoteItem) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}
	if !baseItem.MatchesExchange(baseItem) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}
}

func TestMatchesItemCurrency(t *testing.T) {
	t.Parallel()
	i := Item{}
	if i.MatchesItemCurrency(nil) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
	baseItem, err := CreateItem(exchName, a, pair.Base, decimal.Zero, decimal.Zero)
	assert.NoError(t, err)

	quoteItem, err := CreateItem(exchName, a, pair.Quote, elite, decimal.Zero)
	assert.NoError(t, err)

	if baseItem.MatchesItemCurrency(quoteItem) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
	if !baseItem.MatchesItemCurrency(baseItem) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}
}

func TestReserve(t *testing.T) {
	t.Parallel()
	i := Item{}
	err := i.Reserve(decimal.Zero)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
	err = i.Reserve(elite)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, errCannotAllocate)
	}

	i.reserved = elite
	err = i.Reserve(elite)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, errCannotAllocate)
	}

	i.available = elite
	err = i.Reserve(elite)
	assert.NoError(t, err)

	err = i.Reserve(elite)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, errCannotAllocate)
	}

	err = i.Reserve(neg)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
}

func TestIncreaseAvailable(t *testing.T) {
	t.Parallel()
	i := Item{}
	err := i.IncreaseAvailable(elite)
	assert.NoError(t, err)

	if !i.available.Equal(elite) {
		t.Errorf("expected %v", elite)
	}
	err = i.IncreaseAvailable(decimal.Zero)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
	err = i.IncreaseAvailable(neg)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
}

func TestRelease(t *testing.T) {
	t.Parallel()
	i := Item{}
	err := i.Release(decimal.Zero, decimal.Zero)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
	err = i.Release(elite, decimal.Zero)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, errCannotAllocate)
	}
	i.reserved = elite
	err = i.Release(elite, decimal.Zero)
	assert.NoError(t, err)

	i.reserved = elite
	err = i.Release(elite, one)
	assert.NoError(t, err)

	err = i.Release(neg, decimal.Zero)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
	err = i.Release(elite, neg)
	if !errors.Is(err, errNegativeAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errNegativeAmountReceived)
	}
}

func TestMatchesCurrency(t *testing.T) {
	t.Parallel()
	i := Item{
		currency: currency.BTC,
	}
	if i.MatchesCurrency(currency.USDT) {
		t.Error("expected false")
	}
	if !i.MatchesCurrency(currency.BTC) {
		t.Error("expected true")
	}
	if i.MatchesCurrency(currency.EMPTYCODE) {
		t.Error("expected false")
	}
	if i.MatchesCurrency(currency.NewCode("")) {
		t.Error("expected false")
	}
}
