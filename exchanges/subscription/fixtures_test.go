package subscription

import (
	"maps"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type mockEx struct {
	pairs     currency.Pairs
	assets    asset.Items
	tpl       string
	auth      bool
	errPairs  error
	errFormat error
}

func newMockEx() *mockEx {
	pairs := currency.Pairs{btcusdtPair, ethusdcPair}
	for _, b := range []currency.Code{currency.LTC, currency.XRP, currency.TRX} {
		for _, q := range []currency.Code{currency.USDT, currency.USDC} {
			pairs = append(pairs, currency.NewPair(b, q))
		}
	}

	return &mockEx{
		assets: asset.Items{asset.Spot, asset.Futures},
		pairs:  pairs,
	}
}

func (m *mockEx) GetEnabledPairs(_ asset.Item) (currency.Pairs, error) {
	return m.pairs, m.errPairs
}

func (m *mockEx) GetPairFormat(_ asset.Item, _ bool) (currency.PairFormat, error) {
	return currency.PairFormat{Uppercase: true}, m.errFormat
}

func (m *mockEx) GetSubscriptionTemplate(s *Subscription) (*template.Template, error) {
	if s.Channel == "nil" {
		return nil, nil
	}
	return template.New(m.tpl).
		Funcs(template.FuncMap{
			"assetName": func(a asset.Item) string {
				if a == asset.Futures {
					return "future"
				}
				return a.String()
			},
			"updateAssetPairs": func(ap assetPairs) string {
				ap[asset.Futures] = nil
				ap[asset.Spot] = ap[asset.Spot][0:1]
				return ""
			},
			"batch": common.Batch[currency.Pairs],
		}).
		ParseFiles("testdata/" + m.tpl)
}

func (m *mockEx) GetAssetTypes(_ bool) asset.Items            { return m.assets }
func (m *mockEx) CanUseAuthenticatedWebsocketEndpoints() bool { return m.auth }

// equalLists is a utility function to compare subscription lists and show a pretty failure message
// It overcomes the verbose depth of assert.ElementsMatch spewConfig
// Duplicate of internal/testing/subscriptions/EqualLists
func equalLists(tb testing.TB, a, b List) bool {
	tb.Helper()
	for _, sub := range append(a, b...) {
		sub.Key = &StrictKey{&ExactKey{sub}}
	}
	s, err := NewStoreFromList(a)
	require.NoError(tb, err, "NewStoreFromList must not error")
	added, missing := s.Diff(b)
	if len(added) > 0 || len(missing) > 0 {
		fail := "Differences:"
		if len(added) > 0 {
			fail = fail + "\n + " + strings.Join(added.Strings(), "\n + ")
		}
		if len(missing) > 0 {
			fail = fail + "\n - " + strings.Join(missing.Strings(), "\n - ")
		}
		assert.Fail(tb, fail, "Subscriptions should be equal")
		return false
	}
	return true
}

// StrictKey is key type for subscriptions where all the pairs, QualifiedChannel and Params in a Subscription must match exactly
type StrictKey struct {
	*ExactKey
}

var _ MatchableKey = StrictKey{} // Enforce StrictKey must implement MatchableKey

// Match implements MatchableKey
// Returns true if the key fields exactly matches the subscription, including all Pairs, QualifiedChannel and Params
func (k StrictKey) Match(eachKey MatchableKey) bool {
	if !k.ExactKey.Match(eachKey) {
		return false
	}
	eachSub := eachKey.GetSubscription()
	return eachSub.QualifiedChannel == k.QualifiedChannel &&
		maps.Equal(eachSub.Params, k.Params)
}

// String implements Stringer; returns the Asset, Channel and Pairs
// Does not provide concurrency protection on the subscription it points to
func (k StrictKey) String() string {
	s := k.Subscription
	if s == nil {
		return "Uninitialised StrictKey"
	}
	return s.QualifiedChannel + " " + ExactKey{s}.String()
}
