package v18_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v18 "github.com/thrasher-corp/gocryptotrader/config/versions/v18"
)

// The v17 and v18 default subscriptions as the config saves them
const (
	previousOrderbook = `{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"}`
	previousTrades    = `{"enabled":true,"channel":"allTrades","asset":"spot"}`
	previousOrders    = `{"enabled":true,"channel":"myOrders","asset":"spot","authenticated":true}`
	previousMyTrades  = `{"enabled":true,"channel":"myTrades","asset":"spot","authenticated":true}`
	currentOrderbook  = `{"enabled":true,"channel":"orderbook","asset":"all","interval":"100ms"}`
	currentTrades     = `{"enabled":true,"channel":"allTrades","asset":"all"}`
	currentFunding    = `{"enabled":true,"channel":"funding_rate","asset":"perpetualcontract"}`
	currentOrders     = `{"enabled":true,"channel":"myOrders","asset":"all","authenticated":true}`
	currentMyTrades   = `{"enabled":true,"channel":"myTrades","asset":"all","authenticated":true}`
)

var (
	previousDefaults = []string{previousOrderbook, previousTrades, previousOrders, previousMyTrades}
	currentDefaults  = []string{currentOrderbook, currentTrades, currentFunding, currentOrders, currentMyTrades}
)

// subscriptions returns a Bitstamp config holding the subscription entries
func subscriptions(entries ...string) string {
	return `{"name":"Bitstamp","features":{"subscriptions":[` + strings.Join(entries, ",") + `]}}`
}

// with returns entries with the entry at index i replaced
func with(entries []string, i int, entry string) []string {
	out := append([]string(nil), entries...)
	out[i] = entry
	return out
}

// field returns an entry with a field added before its closing brace
func field(entry, f string) string {
	return strings.TrimSuffix(entry, "}") + "," + f + "}"
}

// replaced returns entries with each pair of old and new strings replaced
func replaced(entries []string, oldnew ...string) []string {
	r := strings.NewReplacer(oldnew...)
	out := make([]string, len(entries))
	for i := range entries {
		out[i] = r.Replace(entries[i])
	}
	return out
}

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"Bitstamp"}, new(v18.Version).Exchanges(), "Exchanges should only list Bitstamp")
}

func TestUpgradeExchangeSubscriptions(t *testing.T) {
	t.Parallel()
	for name, entries := range map[string][]string{
		"previous defaults":         previousDefaults,
		"reordered":                 {previousMyTrades, previousOrderbook, previousOrders, previousTrades},
		"upper case assets":         replaced(previousDefaults, `"spot"`, `"SPOT"`),
		"title case assets":         replaced(previousDefaults, `"spot"`, `"Spot"`),
		"nanosecond interval":       with(previousDefaults, 0, strings.Replace(previousOrderbook, `"100ms"`, `100000000`, 1)),
		"seconds interval":          with(previousDefaults, 0, strings.Replace(previousOrderbook, `"100ms"`, `"0.1s"`, 1)),
		"zero trades interval":      with(previousDefaults, 1, field(previousTrades, `"interval":"0s"`)),
		"explicit unauthenticated":  with(with(previousDefaults, 0, field(previousOrderbook, `"authenticated":false`)), 1, field(previousTrades, `"authenticated":false`)),
		"null authenticated":        with(previousDefaults, 1, field(previousTrades, `"authenticated":null`)),
		"repeated key last applies": with(previousDefaults, 0, strings.Replace(previousOrderbook, `"asset":"spot"`, `"asset":"margin","asset":"spot"`, 1)),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			version := new(v18.Version)
			got, err := version.UpgradeExchange(t.Context(), []byte(subscriptions(entries...)))
			require.NoError(t, err, "UpgradeExchange must not error")
			assert.JSONEq(t, subscriptions(currentDefaults...), string(got), "UpgradeExchange should replace the previous defaults with the current defaults")
			again, err := version.UpgradeExchange(t.Context(), got)
			require.NoError(t, err, "repeated UpgradeExchange must not error")
			assert.Equal(t, got, again, "UpgradeExchange should be idempotent")
		})
	}
}

func TestDowngradeExchangeSubscriptions(t *testing.T) {
	t.Parallel()
	for name, entries := range map[string][]string{
		"current defaults":  currentDefaults,
		"reordered":         {currentFunding, currentMyTrades, currentOrderbook, currentOrders, currentTrades},
		"upper case assets": replaced(currentDefaults, `"all"`, `"ALL"`, `"perpetualcontract"`, `"PERPETUALCONTRACT"`),
		"mixed case assets": replaced(currentDefaults, `"all"`, `"All"`, `"perpetualcontract"`, `"PerpetualContract"`),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			version := new(v18.Version)
			got, err := version.DowngradeExchange(t.Context(), []byte(subscriptions(entries...)))
			require.NoError(t, err, "DowngradeExchange must not error")
			assert.JSONEq(t, subscriptions(previousDefaults...), string(got), "DowngradeExchange should replace the current defaults with the previous defaults")
			again, err := version.DowngradeExchange(t.Context(), got)
			require.NoError(t, err, "repeated DowngradeExchange must not error")
			assert.Equal(t, got, again, "DowngradeExchange should be idempotent")
		})
	}
}

func TestMigrationPreservesCustomSubscriptions(t *testing.T) {
	t.Parallel()
	customised := map[string][]string{
		"empty":                      {},
		"missing entry":              previousDefaults[:3],
		"extra entry":                append(append([]string(nil), previousDefaults...), `{"enabled":true,"channel":"detail_order_book","asset":"spot"}`),
		"repeated entry":             with(previousDefaults, 3, previousOrders),
		"exchange specific channel":  with(previousDefaults, 0, strings.Replace(previousOrderbook, `"orderbook"`, `"order_book"`, 1)),
		"disabled":                   with(previousDefaults, 0, strings.Replace(previousOrderbook, `"enabled":true`, `"enabled":false`, 1)),
		"enabled omitted":            with(previousDefaults, 1, strings.Replace(previousTrades, `"enabled":true,`, ``, 1)),
		"enabled null":               with(previousDefaults, 2, strings.Replace(previousOrders, `"enabled":true`, `"enabled":null`, 1)),
		"custom interval":            with(previousDefaults, 0, strings.Replace(previousOrderbook, `"100ms"`, `"10ms"`, 1)),
		"null interval":              with(previousDefaults, 0, strings.Replace(previousOrderbook, `"100ms"`, `null`, 1)),
		"interval omitted":           with(previousDefaults, 0, strings.Replace(previousOrderbook, `,"interval":"100ms"`, ``, 1)),
		"trades interval":            with(previousDefaults, 1, field(previousTrades, `"interval":"100ms"`)),
		"quoted nanosecond interval": with(previousDefaults, 0, strings.Replace(previousOrderbook, `"100ms"`, `"100000000"`, 1)),
		"wildcard asset":             with(previousDefaults, 0, strings.Replace(previousOrderbook, `"spot"`, `"all"`, 1)),
		"other asset":                with(previousDefaults, 1, strings.Replace(previousTrades, `"spot"`, `"margin"`, 1)),
		"authenticated public":       with(previousDefaults, 0, field(previousOrderbook, `"authenticated":true`)),
		"unauthenticated private":    with(previousDefaults, 2, strings.Replace(previousOrders, `,"authenticated":true`, ``, 1)),
		"private explicitly public":  with(previousDefaults, 3, strings.Replace(previousMyTrades, `"authenticated":true`, `"authenticated":false`, 1)),
		"pairs":                      with(previousDefaults, 0, field(previousOrderbook, `"pairs":"BTC/USD"`)),
		"empty pairs":                with(previousDefaults, 1, field(previousTrades, `"pairs":""`)),
		"levels":                     with(previousDefaults, 0, field(previousOrderbook, `"levels":10`)),
		"params":                     with(previousDefaults, 2, field(previousOrders, `"params":{}`)),
		"unknown field":              with(previousDefaults, 3, field(previousMyTrades, `"custom":"retained"`)),
		"capitalised field":          with(previousDefaults, 1, strings.Replace(previousTrades, `"enabled"`, `"Enabled"`, 1)),
		"current defaults":           currentDefaults,
	}
	for name, entries := range customised {
		t.Run("upgrade/"+name, func(t *testing.T) {
			t.Parallel()
			input := subscriptions(entries...)
			got, err := new(v18.Version).UpgradeExchange(t.Context(), []byte(input))
			require.NoError(t, err, "UpgradeExchange must accept customised subscriptions")
			assert.Equal(t, input, string(got), "UpgradeExchange should preserve customised subscriptions byte-for-byte")
		})
	}
	for name, entries := range map[string][]string{
		"missing funding rate":    with(currentDefaults, 2, currentOrders),
		"disabled funding rate":   with(currentDefaults, 2, strings.Replace(currentFunding, `"enabled":true`, `"enabled":false`, 1)),
		"wildcard funding rate":   with(currentDefaults, 2, strings.Replace(currentFunding, `"perpetualcontract"`, `"all"`, 1)),
		"spot orderbook":          with(currentDefaults, 0, strings.Replace(currentOrderbook, `"all"`, `"spot"`, 1)),
		"funding rate pairs":      with(currentDefaults, 2, field(currentFunding, `"pairs":"BTC/USD-PERP"`)),
		"custom current interval": with(currentDefaults, 0, strings.Replace(currentOrderbook, `"100ms"`, `"1s"`, 1)),
		"extra current entry":     append(append([]string(nil), currentDefaults...), `{"enabled":true,"channel":"announcements"}`),
		"previous defaults":       previousDefaults,
	} {
		t.Run("downgrade/"+name, func(t *testing.T) {
			t.Parallel()
			input := subscriptions(entries...)
			got, err := new(v18.Version).DowngradeExchange(t.Context(), []byte(input))
			require.NoError(t, err, "DowngradeExchange must accept customised subscriptions")
			assert.Equal(t, input, string(got), "DowngradeExchange should preserve customised subscriptions byte-for-byte")
		})
	}
}

func TestMigrationWithoutSubscriptions(t *testing.T) {
	t.Parallel()
	for _, input := range []string{
		`{"name":"Bitstamp"}`,
		`{"name":"Bitstamp","features":{}}`,
		`{"name":"Bitstamp","features":{"subscriptions":null}}`,
	} {
		for name, migrate := range map[string]func(context.Context, []byte) ([]byte, error){
			"UpgradeExchange":   new(v18.Version).UpgradeExchange,
			"DowngradeExchange": new(v18.Version).DowngradeExchange,
		} {
			got, err := migrate(t.Context(), []byte(input))
			require.NoErrorf(t, err, "%s must not error for %s", name, input)
			assert.Equalf(t, input, string(got), "%s should leave %s unchanged", name, input)
		}
	}
}

func TestUpgradeExchangeClientIDRequirement(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct{ input, exp string }{
		"required": {
			input: `{"name":"Bitstamp","api":{"credentials":{"key":"Key","secret":"Secret","clientID":"ClientID"},"credentialsValidator":{"requiresKey":true,"requiresSecret":true,"requiresClientID":true}}}`,
			exp:   `{"name":"Bitstamp","api":{"credentials":{"key":"Key","secret":"Secret","clientID":"ClientID"},"credentialsValidator":{"requiresKey":true,"requiresSecret":true}}}`,
		},
		"not required": {
			input: `{"name":"Bitstamp","api":{"credentialsValidator":{"requiresKey":true,"requiresClientID":false}}}`,
			exp:   `{"name":"Bitstamp","api":{"credentialsValidator":{"requiresKey":true}}}`,
		},
		"only requirement": {
			input: `{"name":"Bitstamp","api":{"credentialsValidator":{"requiresClientID":true}}}`,
			exp:   `{"name":"Bitstamp","api":{"credentialsValidator":{}}}`,
		},
		"capitalised key": {
			input: `{"name":"Bitstamp","api":{"credentialsValidator":{"requiresKey":true,"RequiresClientID":true}}}`,
			exp:   `{"name":"Bitstamp","api":{"credentialsValidator":{"requiresKey":true}}}`,
		},
		"repeated key": {
			input: `{"name":"Bitstamp","api":{"credentialsValidator":{"requiresClientID":false,"requiresKey":true,"requiresClientID":true}}}`,
			exp:   `{"name":"Bitstamp","api":{"credentialsValidator":{"requiresKey":true}}}`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			version := new(v18.Version)
			got, err := version.UpgradeExchange(t.Context(), []byte(tc.input))
			require.NoError(t, err, "UpgradeExchange must not error")
			assert.JSONEq(t, tc.exp, string(got), "UpgradeExchange should remove the client ID requirement and keep the other settings")
			again, err := version.UpgradeExchange(t.Context(), got)
			require.NoError(t, err, "repeated UpgradeExchange must not error")
			assert.Equal(t, got, again, "UpgradeExchange should be idempotent")
		})
	}
	for _, input := range []string{
		`{"name":"Bitstamp","api":{}}`,
		`{"name":"Bitstamp","api":{"credentialsValidator":null}}`,
		`{"name":"Bitstamp","api":{"credentialsValidator":{"requiresKey":true,"requiresSecret":true}}}`,
	} {
		got, err := new(v18.Version).UpgradeExchange(t.Context(), []byte(input))
		require.NoErrorf(t, err, "UpgradeExchange must not error for %s", input)
		assert.Equalf(t, input, string(got), "UpgradeExchange should leave %s unchanged", input)
	}
}

func TestDowngradeExchangeClientIDRequirement(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct{ input, exp string }{
		"absent": {
			input: `{"name":"Bitstamp","api":{"credentialsValidator":{"requiresKey":true,"requiresSecret":true}}}`,
			exp:   `{"name":"Bitstamp","api":{"credentialsValidator":{"requiresKey":true,"requiresSecret":true,"requiresClientID":true}}}`,
		},
		"not required": {
			input: `{"name":"Bitstamp","api":{"credentialsValidator":{"requiresClientID":false}}}`,
			exp:   `{"name":"Bitstamp","api":{"credentialsValidator":{"requiresClientID":true}}}`,
		},
		"no validator": {
			input: `{"name":"Bitstamp","api":{}}`,
			exp:   `{"name":"Bitstamp","api":{}}`,
		},
		"null validator": {
			input: `{"name":"Bitstamp","api":{"credentialsValidator":null}}`,
			exp:   `{"name":"Bitstamp","api":{"credentialsValidator":null}}`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			version := new(v18.Version)
			got, err := version.DowngradeExchange(t.Context(), []byte(tc.input))
			require.NoError(t, err, "DowngradeExchange must not error")
			assert.JSONEq(t, tc.exp, string(got), "DowngradeExchange should restore the client ID requirement of a saved validator")
			again, err := version.DowngradeExchange(t.Context(), got)
			require.NoError(t, err, "repeated DowngradeExchange must not error")
			assert.Equal(t, got, again, "DowngradeExchange should be idempotent")
		})
	}
}

func TestMigrationRejectsMalformedConfigs(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct{ input, errContains string }{
		"subscriptions object":  {input: `{"features":{"subscriptions":{}}}`, errContains: "error decoding Bitstamp subscriptions"},
		"subscription string":   {input: `{"features":{"subscriptions":["orderbook","allTrades","myOrders","myTrades"]}}`, errContains: "error decoding Bitstamp subscriptions"},
		"enabled string":        {input: subscriptions(with(previousDefaults, 0, strings.Replace(previousOrderbook, `"enabled":true`, `"enabled":"true"`, 1))...), errContains: "error decoding Bitstamp subscriptions"},
		"subscriptions missing": {input: `{"features":{"subscriptions":}}`, errContains: "error getting Bitstamp subscriptions"},
	} {
		for direction, migrate := range map[string]func(context.Context, []byte) ([]byte, error){
			"upgrade":   new(v18.Version).UpgradeExchange,
			"downgrade": new(v18.Version).DowngradeExchange,
		} {
			t.Run(direction+"/"+name, func(t *testing.T) {
				t.Parallel()
				input := tc.input
				if direction == "downgrade" && name == "enabled string" {
					input = subscriptions(with(currentDefaults, 0, strings.Replace(currentOrderbook, `"enabled":true`, `"enabled":"true"`, 1))...)
				}
				got, err := migrate(t.Context(), []byte(input))
				require.ErrorContains(t, err, tc.errContains, "migration must report the failing operation")
				assert.Equal(t, input, string(got), "failed migration should preserve the input")
			})
		}
	}
	for _, input := range []string{`{"api":{"credentialsValidator":}}`, `{"api":{"credentialsValidator":{"requiresClientID":}}}`} {
		_, err := new(v18.Version).UpgradeExchange(t.Context(), []byte(input))
		assert.ErrorContainsf(t, err, "Bitstamp credentials validator", "UpgradeExchange should report a malformed validator in %s", input)
	}
	_, err := new(v18.Version).DowngradeExchange(t.Context(), []byte(`{"api":{"credentialsValidator":}}`))
	assert.ErrorContains(t, err, "Bitstamp credentials validator", "DowngradeExchange should report a malformed validator")
}

func TestRegisteredMigration(t *testing.T) {
	t.Parallel()
	previous := `{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":true,"channel":"allTrades","asset":"spot"},{"enabled":true,"channel":"myOrders","asset":"spot","authenticated":true},{"enabled":true,"channel":"myTrades","asset":"spot","authenticated":true}`
	current := strings.Join(currentDefaults, ",")
	validator := `{"requiresKey":true,"requiresSecret":true,"requiresClientID":true}`
	// Other exchanges with the same settings are not migrated
	input := []byte(`{"version":17,"exchanges":[{"name":"Bitstamp","api":{"credentialsValidator":` + validator + `},"features":{"subscriptions":[` + previous + `]}},{"name":"Kraken","api":{"credentialsValidator":` + validator + `},"features":{"subscriptions":[` + previous + `]}}]}`)
	unchanged, err := versions.Manager.Deploy(t.Context(), input, 17)
	require.NoError(t, err, "Deploy must leave an existing v17 configuration valid")
	assert.Equal(t, input, unchanged, "Deploy should retain v17 Bitstamp settings until the v18 upgrade")

	out, err := versions.Manager.Deploy(t.Context(), input, 18)
	require.NoError(t, err, "Deploy must apply the registered v18 upgrade")
	expected := `{"version":18,"exchanges":[{"name":"Bitstamp","api":{"credentialsValidator":{"requiresKey":true,"requiresSecret":true}},"features":{"subscriptions":[` + current + `]}},{"name":"Kraken","api":{"credentialsValidator":` + validator + `},"features":{"subscriptions":[` + previous + `]}}]}`
	assert.JSONEq(t, expected, string(out), "Deploy should only migrate Bitstamp and advance to v18")

	downgraded, err := versions.Manager.Deploy(t.Context(), out, 17)
	require.NoError(t, err, "Deploy must downgrade from v18")
	assert.JSONEq(t, string(input), string(downgraded), "Downgrade should restore the v17 Bitstamp settings")

	reupgraded, err := versions.Manager.Deploy(t.Context(), downgraded, 18)
	require.NoError(t, err, "Deploy must reapply v18")
	assert.JSONEq(t, expected, string(reupgraded), "Reapplying v18 should reproduce the upgraded configuration")
}
