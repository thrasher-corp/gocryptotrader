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
	t.Parallel()
	s := NewStore()
	require.IsType(t, &Store{}, s, "Must return a store ref")
	require.NotNil(t, s.m, "storage map must be initialised")
}

// TestNewStoreFromList exercises NewStoreFromList
func TestNewStoreFromList(t *testing.T) {
	t.Parallel()
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

	l = List{nil, &Subscription{Channel: OrderbookChannel}}
	_, err = NewStoreFromList(l)
	assert.ErrorIs(t, err, common.ErrNilPointer, "Should error correctly on nils")
}

// TestAdd exercises Add and add methods
func TestAdd(t *testing.T) {
	t.Parallel()
	err := (*Store)(nil).Add(&Subscription{})
	assert.ErrorIs(t, err, common.ErrNilPointer, "Should error nil pointer correctly")
	assert.ErrorContains(t, err, "called on nil Store", "Should error correctly")

	err = new(Store).Add(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "Should error nil pointer correctly")
	assert.ErrorContains(t, err, "called on an uninitialised Store", "Should error correctly")

	s := NewStore()
	err = s.Add(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "Should error nil pointer correctly")
	assert.ErrorContains(t, err, "Subscription param", "Should error correctly")

	sub := &Subscription{Channel: TickerChannel}
	require.NoError(t, s.Add(sub), "Should not error on a standard add")
	assert.NotNil(t, s.get(sub), "Should have stored the sub")
	assert.ErrorIs(t, s.Add(sub), ErrDuplicate, "Should error on duplicates")
	assert.NotNil(t, sub.Key, "Add should call EnsureKeyed")
}

// TestGet exercises Get and get methods
// Ensures that key's Match is used, but does not exercise subscription.Match; See TestMatch for that coverage
func TestGet(t *testing.T) {
	t.Parallel()
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

	// Tests for a MatchableKey, ensuring that ExactKey works
	assert.Nil(t, s.Get(Subscription{Channel: CandlesChannel}), "Should return nil without pairs")
	assert.Nil(t, s.Get(Subscription{Channel: CandlesChannel, Pairs: currency.Pairs{ltcusdcPair}}), "Should return nil with wrong pair")
	assert.Nil(t, s.Get(Subscription{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair}}), "Should return nil with only one right pair")
	assert.Same(t, exp[3], s.Get(Subscription{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}}), "Should return pointer when all pairs match")
	assert.Nil(t, s.Get(Subscription{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ethusdcPair, ltcusdcPair}}), "Should return nil when key is superset of pairs")
}

// TestRemove exercises the Remove method
func TestRemove(t *testing.T) {
	t.Parallel()
	err := (*Store)(nil).Remove(&Subscription{})
	assert.ErrorIs(t, err, common.ErrNilPointer, "Should error correctly when called on nil")
	assert.ErrorContains(t, err, "Remove called on nil Store", "Should error correctly when called on nil")

	err = new(Store).Remove(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "Should error correctly when called on an uninit store")
	assert.ErrorContains(t, err, "Remove called on an Uninitialised Store", "Should error correctly when called on an uninit store")

	s := NewStore()
	err = s.Remove(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "Should error correctly when called with nil")
	assert.ErrorContains(t, err, "key param", "Should error correctly when called with nil")

	require.NoError(t, s.Add(&Subscription{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}}), "Adding subscription must not error")
	assert.NotNil(t, s.Get(&ExactKey{&Subscription{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}}}), "Should have added the sub")
	assert.ErrorIs(t, s.Remove(&ExactKey{&Subscription{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair}}}), ErrNotFound, "Should error correctly when called with a non-matching key")
	assert.NoError(t, s.Remove(&ExactKey{&Subscription{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}}}), "Should not error when called with a matching key")
	assert.Nil(t, s.Get(&ExactKey{&Subscription{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}}}), "Should have removed the sub")
	assert.ErrorIs(t, s.Remove(&ExactKey{&Subscription{Channel: CandlesChannel, Pairs: currency.Pairs{btcusdtPair, ethusdcPair}}}), ErrNotFound, "Should error correctly when called twice ")
}

func TestUpdateKeyAndState(t *testing.T) {
	t.Parallel()

	t.Run("validation errors", func(t *testing.T) {
		t.Parallel()

		err := (*Store)(nil).UpdateKeyAndState(&Subscription{}, 42, SubscribedState)
		assert.ErrorIs(t, err, common.ErrNilPointer, "UpdateKeyAndState should error when called on nil Store")
		assert.ErrorContains(t, err, "called on nil Store", "UpdateKeyAndState should include nil Store context")

		err = new(Store).UpdateKeyAndState(&Subscription{}, 42, SubscribedState)
		assert.ErrorIs(t, err, common.ErrNilPointer, "UpdateKeyAndState should error when called on uninitialised Store")
		assert.ErrorContains(t, err, "called on an uninitialised Store", "UpdateKeyAndState should include uninitialised Store context")

		err = NewStore().UpdateKeyAndState(nil, 42, SubscribedState)
		assert.ErrorIs(t, err, common.ErrNilPointer, "UpdateKeyAndState should error when subscription is nil")
		assert.ErrorContains(t, err, "Subscription param", "UpdateKeyAndState should include subscription parameter context")

		err = NewStore().UpdateKeyAndState(&Subscription{}, nil, SubscribedState)
		assert.ErrorIs(t, err, common.ErrNilPointer, "UpdateKeyAndState should error when key is nil")
		assert.ErrorContains(t, err, "key param", "UpdateKeyAndState should include key parameter context")
	})

	t.Run("missing subscription", func(t *testing.T) {
		t.Parallel()

		err := NewStore().UpdateKeyAndState(&Subscription{Channel: TickerChannel}, 42, SubscribedState)
		assert.ErrorIs(t, err, ErrNotFound, "UpdateKeyAndState should error when subscription is missing")
	})

	t.Run("duplicate target key", func(t *testing.T) {
		t.Parallel()

		store := NewStore()
		existing := &Subscription{Key: 42, Channel: TickerChannel}
		other := &Subscription{Key: 1337, Channel: OrderbookChannel}
		require.NoError(t, store.Add(existing), "Add existing subscription must not error")
		require.NoError(t, store.Add(other), "Add other subscription must not error")

		err := store.UpdateKeyAndState(existing, other.Key, SubscribedState)
		assert.ErrorIs(t, err, ErrDuplicate, "UpdateKeyAndState should error when the target key is already used")
		assert.Same(t, existing, store.Get(42), "existing subscription should remain stored at its original key")
		assert.Same(t, other, store.Get(1337), "other subscription should remain stored at the duplicate target key")
	})

	t.Run("updates key and state", func(t *testing.T) {
		t.Parallel()

		store := NewStore()
		sub := &Subscription{Key: "temporary", Channel: TickerChannel}
		require.NoError(t, store.Add(sub), "Add subscription must not error")

		err := store.UpdateKeyAndState(sub, 42, SubscribedState)
		require.NoError(t, err, "UpdateKeyAndState must not error")
		assert.Nil(t, store.Get("temporary"), "old key should no longer resolve")
		assert.Same(t, sub, store.Get(42), "new key should resolve to the same subscription")
		assert.Equal(t, 42, sub.Key, "subscription key should be updated")
		assert.Equal(t, SubscribedState, sub.State(), "subscription state should be updated")
	})

	t.Run("invalid state", func(t *testing.T) {
		t.Parallel()

		store := NewStore()
		sub := &Subscription{Key: "temporary", Channel: TickerChannel}
		require.NoError(t, store.Add(sub), "Add subscription must not error")

		err := store.UpdateKeyAndState(sub, 42, UnsubscribedState+1)
		assert.ErrorIs(t, err, ErrInvalidState, "UpdateKeyAndState should error on invalid state")
		assert.Same(t, sub, store.Get("temporary"), "subscription should remain at its original key after state error")
		assert.Nil(t, store.Get(42), "new key should not resolve after state error")
	})
}

// TestList exercises the List and Len methods
func TestList(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestContained(t *testing.T) {
	t.Parallel()

	var s *Store
	matched := s.Contained(nil)
	assert.Nil(t, matched)

	matched = s.Contained(List{{Channel: TickerChannel}})
	assert.Nil(t, matched)

	s = NewStore()
	matched = s.Contained(nil)
	assert.Nil(t, matched)

	matched = s.Contained(List{})
	assert.Nil(t, matched)

	matched = s.Contained(List{{Channel: TickerChannel}})
	assert.Nil(t, matched)

	require.NoError(t, s.add(&Subscription{Channel: TickerChannel}))

	matched = s.Contained(List{{Channel: TickerChannel}})
	assert.Len(t, matched, 1)
}

func TestMissing(t *testing.T) {
	t.Parallel()

	var s *Store

	unmatched := s.Missing(nil)
	assert.Nil(t, unmatched)

	unmatched = s.Missing(List{{Channel: TickerChannel}})
	assert.Len(t, unmatched, 1)

	s = NewStore()
	unmatched = s.Missing(nil)
	assert.Nil(t, unmatched)

	unmatched = s.Missing(List{})
	assert.Nil(t, unmatched)

	unmatched = s.Missing(List{{Channel: TickerChannel}})
	assert.Len(t, unmatched, 1)

	require.NoError(t, s.add(&Subscription{Channel: TickerChannel}))

	unmatched = s.Missing(List{{Channel: TickerChannel}})
	assert.Nil(t, unmatched)
}
