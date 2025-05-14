package accounts

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	creds1 = &Credentials{Key: "1"}
	creds2 = &Credentials{Key: "2"}
	creds3 = &Credentials{Key: "3"}
)

func TestNewAccounts(t *testing.T) {
	t.Parallel()
	a, err := NewAccounts(&mockEx{"mocky"}, dispatch.GetNewMux(nil))
	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Equal(t, "mocky", a.Exchange.GetName(), "Exchange name should set correctly")
	assert.NotNil(t, a.subAccounts, "subAccounts should be initialised")
	assert.NotEmpty(t, a.routingID, "routingID should not be empty")
	assert.NotNil(t, a.mux, "mux should be set correctly")
	_, err = NewAccounts(nil, dispatch.GetNewMux(nil))
	assert.ErrorIs(t, err, common.ErrNilPointer)
	_, err = NewAccounts(&mockEx{"mocky"}, nil)
	assert.ErrorContains(t, err, "nil pointer: *dispatch.Mux")
}

func TestMustNewAccounts(t *testing.T) {
	t.Parallel()
	a := MustNewAccounts(&mockEx{"mocky"})
	require.NotNil(t, a)
	require.Panics(t, func() { _ = MustNewAccounts(nil) })
}

func TestNewSubAccount(t *testing.T) {
	t.Parallel()
	a := NewSubAccount(asset.Spot, "")
	require.NotNil(t, a, "must not return nil with no id")
	assert.Equal(t, asset.Spot, a.AssetType, "AssetType should be correct")
	assert.Empty(t, a.ID, "ID should not default to anything")
	a = NewSubAccount(asset.Spot, "42")
	assert.Equal(t, "42", a.ID, "ID should be correct")
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	err := dispatch.EnsureRunning(dispatch.DefaultMaxWorkers, dispatch.DefaultJobsLimit)
	require.NoError(t, err, "dispatch.EnsureRunning must not error")
	p, err := MustNewAccounts(&mockEx{}).Subscribe()
	require.NoError(t, err)
	require.NotNil(t, p, "Subscribe must return a pipe")
	require.Empty(t, p.Channel(), "Pipe must be empty before Saving anything")
}

func TestAccountsCurrencyBalances(t *testing.T) {
	t.Parallel()

	a := accountsFixture(t)

	_, err := (*Accounts)(nil).CurrencyBalances(nil, asset.Spot)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	_, err = a.CurrencyBalances(nil, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = a.CurrencyBalances(creds3, asset.All)
	require.ErrorIs(t, err, ErrNoBalances)

	_, err = a.CurrencyBalances(creds3, asset.All)
	assert.ErrorIs(t, err, ErrNoBalances)
	assert.ErrorContains(t, err, "Key:[3")

	// Add a balance with inconsistent currencies to cover err from currs.Add
	a.subAccounts[*creds3] = map[key.SubAccountAsset]currencyBalances{
		{Asset: asset.Futures}: {currency.DOGE.Item: &balance{internal: Balance{Currency: currency.ETH}}},
	}

	type cMap map[currency.Code]float64
	for _, tc := range []struct {
		c   *Credentials
		aT  asset.Item
		exp cMap
		err error
	}{
		{nil, asset.Spot, cMap{currency.BTC: 6.0, currency.LTC: 10.0}, nil},
		{creds1, asset.All, cMap{currency.BTC: 3.0, currency.LTC: 30.0}, nil},
		{creds1, asset.Spot, cMap{currency.BTC: 3.0, currency.LTC: 10.0}, nil},
		{creds1, asset.Futures, cMap{currency.LTC: 20.0}, nil},
		{creds2, asset.Spot, cMap{currency.BTC: 3.0}, nil},
		{creds3, asset.Futures, cMap{currency.DOGE: 50.0}, errBalanceCurrencyMismatch},
	} {
		t.Run(fmt.Sprintf("%s/%s", tc.c, tc.aT), func(t *testing.T) {
			t.Parallel()
			b, err := a.CurrencyBalances(tc.c, tc.aT)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, len(tc.exp), len(b), "must get correct number of balances")
			for c, expBal := range tc.exp {
				assert.Contains(t, b, c)
				assert.Equalf(t, expBal, b[c].Total, "should get correct total for %s", c)
			}
		})
	}
}

func TestAccountsPrivateCurrencyBalances(t *testing.T) {
	t.Parallel()

	a := accountsFixture(t)
	b := a.currencyBalances(creds3, "", asset.Spot)
	r1 := a.subAccounts[*creds3]
	// Using reflect since assert.Same cannot be used on maps to ensure same underlying pointer
	assert.Equal(t,
		reflect.ValueOf(b).UnsafePointer(),
		reflect.ValueOf(r1[key.SubAccountAsset{Asset: asset.Spot}]).UnsafePointer(),
		"should make and return the same map")
	assert.Equal(t,
		reflect.ValueOf(b).UnsafePointer(),
		reflect.ValueOf(a.currencyBalances(creds3, "", asset.Spot)).UnsafePointer(),
		"should return the same map on subsequent calls")
	b = a.currencyBalances(creds3, "", asset.Futures)
	assert.Equal(t,
		reflect.ValueOf(r1).UnsafePointer(),
		reflect.ValueOf(a.subAccounts[*creds3]).UnsafePointer(),
		"should not make a new cred key")
	assert.Equal(t,
		reflect.ValueOf(b).UnsafePointer(),
		reflect.ValueOf(r1[key.SubAccountAsset{Asset: asset.Futures}]).UnsafePointer(),
		"should make and return the same map")
}

type tKey key.SubAccountAsset

func TestAccountsSubAccounts(t *testing.T) {
	t.Parallel()

	a := accountsFixture(t)

	_, err := (*Accounts)(nil).SubAccounts(nil, asset.Spot)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	_, err = a.SubAccounts(nil, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = a.SubAccounts(creds3, asset.All)
	require.ErrorIs(t, err, ErrNoSubAccounts)
	require.ErrorContains(t, err, "Key:[3")

	for _, tc := range []struct {
		c   *Credentials
		aT  asset.Item
		exp []tKey
	}{
		{nil, asset.All, []tKey{{"1a", asset.Spot}, {"1b", asset.Spot}, {"1b", asset.Futures}, {"2a", asset.Spot}}},
		{creds1, asset.All, []tKey{{"1a", asset.Spot}, {"1b", asset.Spot}, {"1b", asset.Futures}}},
		{creds1, asset.Spot, []tKey{{"1a", asset.Spot}, {"1b", asset.Spot}}},
		{creds1, asset.Futures, []tKey{{"1b", asset.Futures}}},
		{creds2, asset.Spot, []tKey{{"2a", asset.Spot}}},
	} {
		t.Run(fmt.Sprintf("%v/%s", tc.c, tc.aT), func(t *testing.T) {
			t.Parallel()
			b, err := a.SubAccounts(tc.c, tc.aT)
			require.NoError(t, err)
			exp := subAccountsFixture(tc.exp)
			require.Equal(t, len(exp), len(b), "must get correct number of subAccounts")
			require.ElementsMatch(t, exp, b, "must get correct subAccounts")
		})
	}
}

func TestAccountsGetBalance(t *testing.T) {
	t.Parallel()

	a := accountsFixture(t)

	_, err := (*Accounts)(nil).GetBalance("", nil, asset.Empty, currency.EMPTYCODE)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = a.GetBalance("", nil, asset.Empty, currency.EMPTYCODE)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = a.GetBalance("", nil, asset.Spot, currency.EMPTYCODE)
	assert.ErrorIs(t, err, errCredentialsEmpty)

	_, err = a.GetBalance("", creds3, asset.Spot, currency.EMPTYCODE)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = a.GetBalance("", creds3, asset.Spot, currency.DOGE)
	assert.ErrorIs(t, err, ErrNoBalances)
	assert.ErrorContains(t, err, "for Key:[3")

	_, err = a.GetBalance("3a", creds1, asset.Spot, currency.DOGE)
	assert.ErrorIs(t, err, ErrNoBalances)
	assert.ErrorContains(t, err, `for mocky SubAccount "3a" spot`)

	_, err = a.GetBalance("1a", creds1, asset.Spot, currency.DOGE)
	assert.ErrorIs(t, err, ErrNoBalances)
	assert.ErrorContains(t, err, `for mocky SubAccount "1a" spot DOGE`)

	b, err := a.GetBalance("1b", creds1, asset.Spot, currency.BTC)
	require.NoError(t, err)
	assert.Equal(t, 2.0, b.Total, "Total should be correct")
}

func TestAccountsSave(t *testing.T) { //nolint:tparallel // Save's internal tests are sequential
	t.Parallel()

	a := accountsFixture(t)
	relay := subscribeFixture(t, a)
	beforeNow := time.Now()

	ctx := t.Context()
	assert.ErrorContains(t, (*Accounts)(nil).Save(ctx, nil, false), "nil pointer: *accounts.Accounts")
	assert.ErrorContains(t, new(Accounts).Save(ctx, nil, false), "nil pointer: accounts.credSubAccounts")

	for _, tc := range []struct {
		name     string
		creds    *Credentials
		snapshot bool
		accts    SubAccounts
		pre      func(context.Context) context.Context
		post     func(t *testing.T) // Any additional assertions
		err      error
	}{
		{
			name:  "NoCredentials",
			accts: SubAccounts{},
			err:   errCredentialsEmpty,
		},
		{
			name:  "BadCredentials",
			accts: SubAccounts{},
			err:   common.ErrTypeAssertFailure,
			pre:   func(ctx context.Context) context.Context { return context.WithValue(ctx, ContextCredentialsFlag, 42) },
		},
		{
			name:  "BadAsset",
			creds: creds1,
			accts: SubAccounts{{AssetType: asset.All}},
			err:   asset.ErrNotSupported,
		},
		{
			name:  "CurrencyMismatch",
			creds: creds1,
			err:   errBalanceCurrencyMismatch,
			accts: SubAccounts{{
				AssetType: asset.Spot,
				ID:        "1a",
				Balances:  CurrencyBalances{currency.BTC: {Currency: currency.DOGE}},
			}},
		},
		{
			name:  "OutOfSequence",
			creds: creds1,
			err:   errOutOfSequence,
			accts: SubAccounts{{
				AssetType: asset.Spot,
				ID:        "1a",
				Balances:  CurrencyBalances{currency.BTC: {UpdatedAt: skynetDate.Add(-time.Hour)}},
			}},
		},
		{
			name:  "BasicSave",
			creds: creds1,
			accts: SubAccounts{
				{
					AssetType: asset.Spot,
					ID:        "1a",
					Balances:  CurrencyBalances{currency.BTC: {Total: 4, UpdatedAt: skynetDate.Add(time.Minute)}},
				},
				{
					AssetType: asset.Spot,
					ID:        "1c",
					Balances:  CurrencyBalances{currency.ETH: {Total: 6}},
				},
			},
			post: func(t *testing.T) {
				t.Helper()
				_, err := a.GetBalance("1a", creds1, asset.Spot, currency.LTC)
				require.NoError(t, err, "Other balances must not be affected")
			},
		},
		{
			name:  "NewCredsSaveAndPublish",
			creds: creds3,
			accts: SubAccounts{
				{
					AssetType: asset.Futures,
					ID:        "3a",
					Balances:  CurrencyBalances{currency.DOGE: {Total: 6.2}},
				},
			},
			post: func(t *testing.T) {
				t.Helper()
				require.Eventually(t, func() bool { return len(relay) > 0 }, time.Second, time.Millisecond, "Publish must eventually send to Channel")
				pub := <-relay
				assert.Equal(t, "3a", pub.ID, "Publish should have correct ID")
				assert.Contains(t, pub.Balances, currency.DOGE, "Should get DOGE Balance")
				b := pub.Balances[currency.DOGE]
				assert.Equal(t, currency.DOGE, b.Currency, "Currency should default to the Balances map key")
				assert.WithinRange(t, b.UpdatedAt, beforeNow, time.Now(), "UpdatedAt should default to time.Now")
				assert.Equal(t, 6.2, b.Total, "Total should be correct")
			},
		},
		{
			name:  "SnapshotSave",
			creds: creds1,
			accts: SubAccounts{
				{
					AssetType: asset.Spot,
					ID:        "1a",
					Balances:  CurrencyBalances{currency.LTC: {Total: 12}},
				},
			},
			snapshot: true,
			post: func(t *testing.T) {
				t.Helper()
				_, err := a.GetBalance("1a", creds1, asset.Spot, currency.BTC)
				require.ErrorIs(t, err, ErrNoBalances, "BTC balance must be removed")
			},
		},
		{
			name:  "PublishError",
			creds: creds1,
			accts: SubAccounts{
				{
					AssetType: asset.Spot,
					ID:        "1a",
					Balances:  CurrencyBalances{currency.DOGE: {Total: 7.2}},
				},
			},
			pre: func(ctx context.Context) context.Context {
				a.mux = nil
				return ctx
			},
			err: errPublish,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			if tc.creds != nil {
				ctx = DeployCredentialsToContext(ctx, tc.creds)
			}
			expAccts := tc.accts.clone()
			if tc.pre != nil {
				ctx = tc.pre(ctx)
			}
			err := a.Save(ctx, tc.accts, tc.snapshot)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			for i, acct := range tc.accts {
				for curr := range acct.Balances {
					t.Run(fmt.Sprintf("%s/%s/%s", acct.AssetType, acct.ID, curr), func(t *testing.T) {
						exp := expAccts[i].Balances[curr]
						got, err := a.GetBalance(acct.ID, tc.creds, acct.AssetType, curr)
						require.NoError(t, err, "GetBalance must not error")
						if !exp.UpdatedAt.IsZero() {
							assert.Equal(t, exp.UpdatedAt, got.UpdatedAt, "UpdatedAt should match balance")
						} else {
							assert.WithinRange(t, got.UpdatedAt, beforeNow, time.Now(), "UpdatedAt should default to time.Now")
						}
						assert.Equal(t, exp.Total, got.Total, "Total should be correct")
						if tc.post != nil {
							tc.post(t)
						}
					})
				}
			}
		})
	}
}

var skynetDate = time.Unix(872896440, 0)

// accountsFixture returns an Accounts store with SubAccount IDs per credentials, and a subscription channel for updates
func accountsFixture(t *testing.T) *Accounts {
	t.Helper()
	a := MustNewAccounts(&mockEx{})
	for _, f := range []struct {
		c  *Credentials
		sA string
		aT asset.Item
		cC currency.Code
		b  float64
	}{
		{creds1, "1a", asset.Spot, currency.BTC, 1},
		{creds1, "1a", asset.Spot, currency.LTC, 10},
		{creds1, "1b", asset.Spot, currency.BTC, 2},
		{creds1, "1b", asset.Futures, currency.LTC, 20},
		{creds2, "2a", asset.Spot, currency.BTC, 3},
	} {
		// Not using t.Run because this is a helper
		u, err := a.currencyBalances(f.c, f.sA, f.aT).balance(f.cC.Item).update(Balance{Total: f.b, UpdatedAt: skynetDate})
		require.NoErrorf(t, err, "Deploy fixture balance must not error for %s/%s/%s/%s", f.c.Key, f.sA, f.aT, f.cC)
		require.Truef(t, u, "Deploy fixture balance must apply an update for %s/%s/%s/%s", f.c.Key, f.sA, f.aT, f.cC)
	}
	return a
}

var subAccts = SubAccounts{
	{
		ID:        "1a",
		AssetType: asset.Spot,
		Balances: CurrencyBalances{
			currency.LTC: Balance{Currency: currency.LTC, Total: 10, UpdatedAt: skynetDate},
			currency.BTC: Balance{Currency: currency.BTC, Total: 1, UpdatedAt: skynetDate},
		},
	},
	{
		ID:        "1b",
		AssetType: asset.Spot,
		Balances:  CurrencyBalances{currency.BTC: Balance{Currency: currency.BTC, Total: 2.0, UpdatedAt: skynetDate}},
	},
	{
		ID:        "1b",
		AssetType: asset.Futures,
		Balances:  CurrencyBalances{currency.LTC: Balance{Currency: currency.LTC, Total: 20.0, UpdatedAt: skynetDate}},
	},
	{
		ID:        "2a",
		AssetType: asset.Spot,
		Balances:  CurrencyBalances{currency.BTC: Balance{Currency: currency.BTC, Total: 3.0, UpdatedAt: skynetDate}},
	},
}

func subAccountsFixture(keys []tKey) (a SubAccounts) {
	if keys == nil {
		return subAccts.clone()
	}
	for _, k := range keys {
		i := slices.IndexFunc(subAccts, func(s *SubAccount) bool {
			return k.SubAccount == s.ID && k.Asset == s.AssetType
		})
		if i == -1 {
			panic(fmt.Sprintf("subAccountsFixture called with unknown subAccount key: %v", k))
		}
		a = append(a, subAccts[i])
	}
	return a
}

func subscribeFixture(t *testing.T, a *Accounts) chan *SubAccount {
	t.Helper()
	err := dispatch.EnsureRunning(dispatch.DefaultMaxWorkers, dispatch.DefaultJobsLimit)
	require.NoError(t, err, "dispatch.EnsureRunning must not error")
	p, err := a.Subscribe()
	require.NoError(t, err, "Subscribe must not error")
	require.NotNil(t, p, "Subscribe must return a pipe")
	relay := make(chan *SubAccount, 64)
	go func() {
		for v := range p.Channel() {
			if s, ok := v.(*SubAccount); ok && s.ID == "3a" { // Only interested in relaying events for a single test account
				relay <- s
			}
		}
	}()
	runtime.Gosched()
	return relay
}

func TestMerge(t *testing.T) {
	t.Parallel()
	s := subAccountsFixture(nil)
	assert.Nil(t, s.Merge(nil), "Should return nil for a merge of nil SubAccounts")
	exp := len(s)
	a := &SubAccount{
		ID:        "1a",
		AssetType: asset.Spot,
		Balances:  CurrencyBalances{currency.BTC: Balance{Total: 1}},
	}
	s = s.Merge(a)
	require.Equal(t, exp, len(s), "Must contain correct number of accounts after merging")

	for _, acct := range s {
		if acct.ID == "1a" && acct.AssetType == asset.Spot {
			assert.Equal(t, 2.0, acct.Balances[currency.BTC].Total)
		}
	}

	a = &SubAccount{
		ID:        "new",
		AssetType: asset.Spot,
		Balances:  CurrencyBalances{currency.BTC: Balance{Total: 1}},
	}
	s = s.Merge(a)
	assert.Contains(t, s, a, "Should contain the new subaccount")
}

func TestSubAccountsClone(t *testing.T) {
	t.Parallel()
	s := SubAccounts{
		{ID: "1", AssetType: asset.Spot, Balances: CurrencyBalances{currency.BTC: {Total: 1}}},
		{ID: "2", AssetType: asset.Futures, Balances: CurrencyBalances{currency.LTC: {Total: 2}}},
	}
	c := s.clone()
	require.Equal(t, s, c, "Clone must match original")
	c[0].ID = "3"
	assert.NotEqual(t, s, c, "Should not be equal after modification")
}

func (l SubAccounts) clone() (c SubAccounts) {
	for _, s := range l {
		bals := make(CurrencyBalances, len(s.Balances))
		maps.Copy(bals, s.Balances)
		c = append(c, &SubAccount{
			ID:        s.ID,
			AssetType: s.AssetType,
			Balances:  bals,
		})
	}
	return c
}
