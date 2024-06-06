package subscription

import (
	"maps"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// TestNewStore exercises NewStore
func TestNewStore(t *testing.T) {
	s := NewStore()
	require.IsType(t, &Store{}, s, "Must return a store ref")
	require.NotNil(t, s.m, "storage map must be initialised")
}

// TestNewStoreFromList exercises NewStoreFromList
func TestNewStoreFromList(t *testing.T) {
	s, err := NewStoreFromList(List{})
	assert.NoError(t, err, "Should not error on empty list")
	require.IsType(t, &Store{}, s, "Must return a store ref")
	l := List{
		{Channel: OrderbookChannel},
		{Channel: TickerChannel},
	}
	s, err = NewStoreFromList(l)
	assert.NoError(t, err, "Should not error on empty list")
	assert.Len(t, s.m, 2, "Map should have 2 values")
	assert.NotNil(t, s.get(l[0]), "Should be able to get a list element")

	l = append(l, &Subscription{Channel: OrderbookChannel})
	_, err = NewStoreFromList(l)
	assert.ErrorIs(t, err, ErrDuplicate, "Should error correctly on duplicates")
}

// TestAdd exercises Add and add methods
func TestAdd(t *testing.T) {
	assert.ErrorIs(t, (*Store)(nil).Add(&Subscription{}), common.ErrNilPointer, "Should error nil pointer correctly")
	assert.ErrorIs(t, (&Store{}).Add(nil), common.ErrNilPointer, "Should error nil pointer correctly")
	assert.ErrorIs(t, (&Store{}).Add(&Subscription{}), common.ErrNilPointer, "Should error nil pointer correctly")

	s := NewStore()
	sub := &Subscription{Channel: TickerChannel}
	require.NoError(t, s.Add(sub), "Should not error on a standard add")
	assert.NotNil(t, s.get(sub), "Should have stored the sub")
	assert.ErrorIs(t, s.Add(sub), ErrDuplicate, "Should error on duplicates")
	assert.Same(t, sub.Key, sub, "Add should call EnsureKeyed")
}

// TestGet exercises Get and get methods
// Ensures that key's Match is used, but does not exercise subscription.Match; See TestMatch for that coverage
func TestGet(t *testing.T) {
	assert.Nil(t, (*Store)(nil).Get(&Subscription{}), "Should return nil when called on nil")
	assert.Nil(t, (&Store{}).Get(&Subscription{}), "Should return nil when called with no subscription map")
	s := NewStore()
	exp := List{
		{Channel: AllOrdersChannel},
		{Channel: TickerChannel, Pairs: currency.Pairs{btcusdtPair}},
		{Key: 42, Channel: OrderbookChannel},
		{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}},
	}
	for _, sub := range exp {
		require.NoError(t, s.Add(sub), "Adding subscription must not error)")
	}

	// Tests for the default Subscription key
	assert.Nil(t, s.Get(Subscription{Channel: OrderbookChannel}), "Should return nil for a sub with a different key type")
	assert.Same(t, exp[0], s.Get(Subscription{Channel: AllOrdersChannel}), "Should return pointer for known sub passed-by-value")
	assert.Same(t, exp[1], s.Get(exp[1]), "Should return same pointer for known sub")
	assert.Same(t, exp[2], s.Get(42), "Should return pointer for simple key lookup")
	assert.Same(t, exp[3], s.Get(Subscription{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair}}), "Should return pointer for single pair lookup")
	assert.Nil(t, s.Get(Subscription{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ltcusdcPair}}), "Should return nil for a bad lookup")

	// Tests for a MatchableKey, ensuring that AllPairsKey works
	assert.Nil(t, s.Get(&AllPairsKey{Channel: CandlesChannel}), "Should return nil without pairs")
	assert.Nil(t, s.Get(&AllPairsKey{Channel: CandlesChannel, Pairs: currency.Pairs{ltcusdcPair}}), "Should return nil with wrong pair")
	assert.Nil(t, s.Get(&AllPairsKey{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair}}), "Should return nil with only one right pair")
	assert.Same(t, exp[3], s.Get(&AllPairsKey{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}}), "Should return pointer when all pairs match")
	assert.Nil(t, s.Get(&AllPairsKey{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ethusdcPair, ltcusdcPair}}), "Should return nil when key is superset of pairs")
}

// TestRemove exercises the Remove method
func TestRemove(t *testing.T) {
	assert.ErrorIs(t, (*Store)(nil).Remove(&Subscription{}), common.ErrNilPointer, "Should error correctly when called on nil")
	assert.ErrorIs(t, (&Store{}).Remove(nil), common.ErrNilPointer, "Should error correctly when called passing nil")
	assert.ErrorIs(t, (&Store{}).Remove(&Subscription{}), common.ErrNilPointer, "Should error correctly when called with no subscription map")

	s := NewStore()
	require.NoError(t, s.Add(&Subscription{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}}), "Adding subscription must not error")
	assert.NotNil(t, s.Get(&AllPairsKey{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}}), "Should have added the sub")
	assert.ErrorIs(t, s.Remove(&AllPairsKey{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair}}), ErrNotFound, "Should error correctly when called with a non-matching key")
	assert.NoError(t, s.Remove(&AllPairsKey{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}}), "Should not error when called with a matching key")
	assert.Nil(t, s.Get(&AllPairsKey{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}}), "Should have removed the sub")
	assert.ErrorIs(t, s.Remove(&AllPairsKey{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}}), ErrNotFound, "Should error correctly when called twice ")
}

// TestList exercises the List and Len methods
func TestList(t *testing.T) {
	assert.Empty(t, (*Store)(nil).List(), "Should return an empty List when called on nil")
	assert.Empty(t, (&Store{}).List(), "Should return an empty List when called on Store without map")
	s := NewStore()
	exp := List{
		{Channel: OrderbookChannel},
		{Channel: TickerChannel},
		{Key: 42, Channel: CandlesChannel},
	}
	for _, sub := range exp {
		require.NoError(t, s.Add(sub), "Adding subscription must not error)")
	}
	l := s.List()
	require.Len(t, l, 3, "Must have 3 elements in the list")
	assert.ElementsMatch(t, exp, l, "List Should have the same subscriptions")

	require.Equal(t, 3, s.Len(), "Len must return 3")
	require.Equal(t, 0, (*Store)(nil).Len(), "Len must return 0 on a nil store")
	require.Equal(t, 0, (&Store{}).Len(), "Len must return 0 on an uninitialized store")
}

// TestStoreClear exercises the Clear method
func TestStoreClear(t *testing.T) {
	assert.NotPanics(t, func() { (*Store)(nil).Clear() }, "Should not panic when called on nil")
	s := &Store{}
	assert.NotPanics(t, func() { s.Clear() }, "Should not panic when called with no subscription map")
	assert.NotNil(t, s.m, "Should create a map when called on an empty Store")
	require.NoError(t, s.Add(&Subscription{Channel: CandlesChannel}), "Adding subscription must not error")
	require.Len(t, s.m, 1, "Must have a subscription")
	s.Clear()
	require.Empty(t, s.m, "Map must be empty after clearing")
	assert.NotPanics(t, func() { s.Clear() }, "Should not panic when called on an empty map")
}

// TestStoreDiff exercises the Diff method
func TestStoreDiff(t *testing.T) {
	s := NewStore()
	assert.NotPanics(t, func() { (*Store)(nil).Diff(List{}) }, "Should not panic when called on nil")
	assert.NotPanics(t, func() { (&Store{}).Diff(List{}) }, "Should not panic when called with no subscription map")
	subs, unsubs := s.Diff(List{{Channel: TickerChannel}, {Channel: CandlesChannel}, {Channel: OrderbookChannel}})
	assert.Equal(t, 3, len(subs), "Should get the correct number of subs")
	assert.Empty(t, unsubs, "Should get no unsubs")
	for _, sub := range subs {
		require.NoError(t, s.add(sub), "add must not error")
	}
	assert.NotPanics(t, func() { s.Diff(nil) }, "Should not panic when called with nil list")

	subs, unsubs = s.Diff(List{{Channel: CandlesChannel}})
	assert.Empty(t, subs, "Should get no subs")
	assert.Equal(t, 2, len(unsubs), "Should get the correct number of unsubs")
	subs, unsubs = s.Diff(List{{Channel: TickerChannel}, {Channel: MyTradesChannel}})
	require.Equal(t, 1, len(subs), "Should get the correct number of subs")
	assert.Equal(t, MyTradesChannel, subs[0].Channel, "Should get correct channels in sub")
	require.Equal(t, 2, len(unsubs), "Should get the correct number of unsubs")
	EqualLists(t, unsubs, List{{Channel: OrderbookChannel}, {Channel: CandlesChannel}})
}

func EqualLists(tb testing.TB, a, b List) {
	tb.Helper()
	// Must not use store.Diff directly
	s, err := NewStoreFromList(a)
	require.NoError(tb, err, "NewStoreFromList must not error")
	missingMap := maps.Clone(s.m)
	var added, missing List
	for _, sub := range b {
		if found := s.get(sub); found != nil {
			delete(missingMap, found.Key)
		} else {
			added = append(added, sub)
		}
	}
	for _, c := range missingMap {
		missing = append(missing, c)
	}
	if len(added) > 0 || len(missing) > 0 {
		fail := "Differences:"
		if len(added) > 0 {
			fail = fail + "\n + " + strings.Join(added.Strings(), "\n + ")
		}
		if len(missing) > 0 {
			fail = fail + "\n - " + strings.Join(missing.Strings(), "\n - ")
		}
		assert.Fail(tb, fail, "Subscriptions should be equal")
	}
}
