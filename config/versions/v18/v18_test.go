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
	heartbeatEntry  = `{"enabled":true,"channel":"heartbeat"}`
	statusEntry     = `{"enabled":true,"channel":"status","asset":"all"}`
	spotTicker      = `{"enabled":true,"channel":"ticker","asset":"spot"}`
	spotCandles     = `{"enabled":true,"channel":"candles","asset":"spot"}`
	spotTrades      = `{"enabled":true,"channel":"allTrades","asset":"spot"}`
	spotOrderbook   = `{"enabled":true,"channel":"orderbook","asset":"spot"}`
	accountEntry    = `{"enabled":true,"channel":"myAccount","asset":"all","authenticated":true}`
	tickerBatch     = `{"enabled":true,"channel":"ticker_batch","asset":"spot"}`
	tickerEntry     = `{"enabled":true,"channel":"ticker","asset":"all"}`
	candlesEntry    = `{"enabled":true,"channel":"candles","asset":"all","interval":"5m"}`
	tradesEntry     = `{"enabled":true,"channel":"allTrades","asset":"all"}`
	orderbookEntry  = `{"enabled":true,"channel":"orderbook","asset":"all"}`
	balanceSummary  = `{"enabled":true,"channel":"futures_balance_summary","authenticated":true}`
	derivativesBook = `{"enabled":true,"channel":"book","asset":"usdcmarginedfutures"}`
)

var (
	previousDefaults = []string{heartbeatEntry, statusEntry, spotTicker, spotCandles, spotTrades, spotOrderbook, accountEntry, tickerBatch}
	currentDefaults  = []string{tickerEntry, candlesEntry, tradesEntry, orderbookEntry, statusEntry, accountEntry}
)

// subscriptions returns a Coinbase config holding the subscription entries
func subscriptions(entries ...string) string {
	return `{"name":"Coinbase","features":{"subscriptions":[` + strings.Join(entries, ",") + `]}}`
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
	assert.Equal(t, []string{"Coinbase"}, new(v18.Version).Exchanges(), "Exchanges should only list Coinbase")
}

func TestUpgradeExchangeSubscriptions(t *testing.T) {
	t.Parallel()
	for name, entries := range map[string][]string{
		"previous defaults":         previousDefaults,
		"reordered":                 {tickerBatch, accountEntry, spotOrderbook, spotTrades, spotCandles, spotTicker, statusEntry, heartbeatEntry},
		"upper case assets":         replaced(previousDefaults, `"all"`, `"ALL"`, `"spot"`, `"SPOT"`),
		"title case assets":         replaced(previousDefaults, `"all"`, `"All"`, `"spot"`, `"Spot"`),
		"empty heartbeat asset":     with(previousDefaults, 0, field(heartbeatEntry, `"asset":""`)),
		"zero candles interval":     with(previousDefaults, 3, field(spotCandles, `"interval":"0s"`)),
		"nanosecond zero interval":  with(previousDefaults, 3, field(spotCandles, `"interval":0`)),
		"zero levels":               with(previousDefaults, 5, field(spotOrderbook, `"levels":0`)),
		"null levels":               with(previousDefaults, 5, field(spotOrderbook, `"levels":null`)),
		"empty params":              with(previousDefaults, 2, field(spotTicker, `"params":{}`)),
		"null params":               with(previousDefaults, 4, field(spotTrades, `"params":null`)),
		"explicit unauthenticated":  with(previousDefaults, 7, field(tickerBatch, `"authenticated":false`)),
		"null authenticated":        with(previousDefaults, 1, field(statusEntry, `"authenticated":null`)),
		"repeated key last applies": with(previousDefaults, 2, strings.Replace(spotTicker, `"asset":"spot"`, `"asset":"futures","asset":"spot"`, 1)),
		"legacy level2_batch":       {`{"enabled":true,"channel":"level2_batch","asset":"spot"}`, spotTicker},
		"legacy matches":            {`{"enabled":false,"channel":"matches"}`, `{"enabled":true,"channel":"custom","asset":"spot","pairs":"BTC-USD"}`},
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
		"current defaults":     currentDefaults,
		"reordered":            {accountEntry, statusEntry, orderbookEntry, tradesEntry, candlesEntry, tickerEntry},
		"upper case assets":    replaced(currentDefaults, `"all"`, `"ALL"`),
		"duration interval":    with(currentDefaults, 1, strings.Replace(candlesEntry, `"5m"`, `"5m0s"`, 1)),
		"nanosecond interval":  with(currentDefaults, 1, strings.Replace(candlesEntry, `"5m"`, `300000000000`, 1)),
		"unit interval":        with(currentDefaults, 1, strings.Replace(candlesEntry, `"5m"`, `"5min"`, 1)),
		"seconds interval":     with(currentDefaults, 1, strings.Replace(candlesEntry, `"5m"`, `"300s"`, 1)),
		"explicit unauthentic": with(currentDefaults, 0, field(tickerEntry, `"authenticated":false`)),
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
	for name, entries := range map[string][]string{
		"empty":                    {},
		"missing entry":            previousDefaults[:7],
		"extra entry":              append(append([]string(nil), previousDefaults...), `{"enabled":true,"channel":"candles","asset":"futures"}`),
		"repeated entry":           with(previousDefaults, 3, spotTicker),
		"exchange specific name":   with(previousDefaults, 5, strings.Replace(spotOrderbook, `"orderbook"`, `"level2"`, 1)),
		"capitalised channel":      with(previousDefaults, 2, strings.Replace(spotTicker, `"ticker"`, `"Ticker"`, 1)),
		"disabled":                 with(previousDefaults, 2, strings.Replace(spotTicker, `"enabled":true`, `"enabled":false`, 1)),
		"enabled omitted":          with(previousDefaults, 4, strings.Replace(spotTrades, `"enabled":true,`, ``, 1)),
		"enabled null":             with(previousDefaults, 3, strings.Replace(spotCandles, `"enabled":true`, `"enabled":null`, 1)),
		"custom interval":          with(previousDefaults, 3, field(spotCandles, `"interval":"5m"`)),
		"null interval":            with(previousDefaults, 3, field(spotCandles, `"interval":null`)),
		"heartbeat asset":          with(previousDefaults, 0, field(heartbeatEntry, `"asset":"spot"`)),
		"wildcard ticker":          with(previousDefaults, 2, strings.Replace(spotTicker, `"spot"`, `"all"`, 1)),
		"spot status":              with(previousDefaults, 1, strings.Replace(statusEntry, `"all"`, `"spot"`, 1)),
		"unauthenticated account":  with(previousDefaults, 6, strings.Replace(accountEntry, `,"authenticated":true`, ``, 1)),
		"authenticated ticker":     with(previousDefaults, 2, field(spotTicker, `"authenticated":true`)),
		"pairs":                    with(previousDefaults, 2, field(spotTicker, `"pairs":"BTC-USD"`)),
		"levels":                   with(previousDefaults, 5, field(spotOrderbook, `"levels":25`)),
		"fractional levels":        with(previousDefaults, 5, field(spotOrderbook, `"levels":1.5`)),
		"parameter":                with(previousDefaults, 2, field(spotTicker, `"params":{"product_ids":"BTC-USD"}`)),
		"unknown field":            with(previousDefaults, 4, field(spotTrades, `"custom":"retained"`)),
		"capitalised field":        with(previousDefaults, 4, strings.Replace(spotTrades, `"enabled"`, `"Enabled"`, 1)),
		"current defaults":         currentDefaults,
		"master's example config":  {heartbeatEntry, `{"enabled":false,"channel":"status","authenticated":true}`, spotTicker, spotCandles, spotTrades, spotOrderbook, `{"enabled":true,"channel":"account","authenticated":true}`, `{"enabled":false,"channel":"ticker_batch","asset":"spot"}`},
		"previous and one more":    append(append([]string(nil), previousDefaults...), balanceSummary),
		"legacy name as parameter": with(previousDefaults, 2, field(spotTicker, `"params":{"channel":"matches"}`)),
	} {
		t.Run("upgrade/"+name, func(t *testing.T) {
			t.Parallel()
			input := subscriptions(entries...)
			got, err := new(v18.Version).UpgradeExchange(t.Context(), []byte(input))
			require.NoError(t, err, "UpgradeExchange must accept customised subscriptions")
			assert.Equal(t, input, string(got), "UpgradeExchange should preserve customised subscriptions byte-for-byte")
		})
	}
	for name, entries := range map[string][]string{
		"missing account":      currentDefaults[:5],
		"disabled candles":     with(currentDefaults, 1, strings.Replace(candlesEntry, `"enabled":true`, `"enabled":false`, 1)),
		"one minute candles":   with(currentDefaults, 1, strings.Replace(candlesEntry, `"5m"`, `"1m"`, 1)),
		"candles without time": with(currentDefaults, 1, strings.Replace(candlesEntry, `,"interval":"5m"`, ``, 1)),
		"spot ticker":          with(currentDefaults, 0, strings.Replace(tickerEntry, `"all"`, `"spot"`, 1)),
		"ticker pairs":         with(currentDefaults, 0, field(tickerEntry, `"pairs":"BTC-USD"`)),
		"previous defaults":    previousDefaults,
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

func TestDowngradeDisablesUnsupportedSubscriptions(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct{ entries, exp []string }{
		"balance summary": {
			entries: append(append([]string(nil), previousDefaults...), balanceSummary),
			exp:     append(append([]string(nil), previousDefaults...), strings.Replace(balanceSummary, `"enabled":true`, `"enabled":false`, 1)),
		},
		"wildcard orders": {
			entries: append(append([]string(nil), previousDefaults...), `{"enabled":true,"channel":"myOrders","asset":"all","authenticated":true}`),
			exp:     append(append([]string(nil), previousDefaults...), `{"enabled":false,"channel":"myOrders","asset":"all","authenticated":true}`),
		},
		"futures gateway channel": {
			entries: append(append([]string(nil), previousDefaults...), `{"enabled":true,"channel":"book","asset":"Futures"}`),
			exp:     append(append([]string(nil), previousDefaults...), `{"enabled":false,"channel":"book","asset":"Futures"}`),
		},
		"perpetual channel kept": {
			entries: append(append([]string(nil), previousDefaults...), derivativesBook),
			exp:     append(append([]string(nil), previousDefaults...), derivativesBook),
		},
		"coin margined channel kept": {
			entries: append(append([]string(nil), previousDefaults...), `{"enabled":true,"channel":"myTrades","asset":"COINMARGINEDFUTURES","authenticated":true}`),
			exp:     append(append([]string(nil), previousDefaults...), `{"enabled":true,"channel":"myTrades","asset":"COINMARGINEDFUTURES","authenticated":true}`),
		},
		"disabled kept": {
			entries: append(append([]string(nil), previousDefaults...), strings.Replace(balanceSummary, `"enabled":true`, `"enabled":false`, 1)),
			exp:     append(append([]string(nil), previousDefaults...), strings.Replace(balanceSummary, `"enabled":true`, `"enabled":false`, 1)),
		},
		"customised current defaults": {
			entries: append(append([]string(nil), currentDefaults...), balanceSummary),
			exp:     append(append([]string(nil), currentDefaults...), strings.Replace(balanceSummary, `"enabled":true`, `"enabled":false`, 1)),
		},
		// The previous version replaces every subscription with its defaults for a legacy channel, enabled or not
		"legacy channels removed": {
			entries: []string{
				`{"enabled":false,"channel":"matches","asset":"spot"}`,
				`{"enabled":true,"channel":"ticker","asset":"spot"}`,
				`{"enabled":true,"channel":"level2_batch","asset":"spot"}`,
				`{"enabled":false,"channel":"matches"}`,
				`{"enabled":true,"channel":"candles","asset":"spot"}`,
			},
			exp: []string{
				`{"enabled":true,"channel":"ticker","asset":"spot"}`,
				`{"enabled":true,"channel":"candles","asset":"spot"}`,
			},
		},
		"legacy channel names matched exactly": {
			entries: []string{`{"enabled":false,"channel":"Matches","asset":"spot"}`, `{"enabled":true,"channel":"ticker","asset":"spot"}`},
			exp:     []string{`{"enabled":false,"channel":"Matches","asset":"spot"}`, `{"enabled":true,"channel":"ticker","asset":"spot"}`},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := new(v18.Version).DowngradeExchange(t.Context(), []byte(subscriptions(tc.entries...)))
			require.NoError(t, err, "DowngradeExchange must not error")
			assert.JSONEq(t, subscriptions(tc.exp...), string(got), "DowngradeExchange should disable the subscriptions the previous version cannot generate")
		})
	}
}

func TestMigrationWithoutSubscriptions(t *testing.T) {
	t.Parallel()
	for _, input := range []string{
		`{"name":"Coinbase"}`,
		`{"name":"Coinbase","features":{}}`,
		`{"name":"Coinbase","features":{"subscriptions":null}}`,
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

func TestMigrationRejectsMalformedConfigs(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct{ input, errContains string }{
		"subscriptions object":  {input: `{"features":{"subscriptions":{}}}`, errContains: "error decoding Coinbase subscriptions"},
		"subscription string":   {input: `{"features":{"subscriptions":["ticker","allTrades"]}}`, errContains: "error decoding Coinbase subscriptions"},
		"params string":         {input: subscriptions(with(previousDefaults, 2, field(spotTicker, `"params":"BTC-USD"`))...), errContains: "error decoding Coinbase subscriptions"},
		"subscriptions missing": {input: `{"features":{"subscriptions":}}`, errContains: "error getting Coinbase subscriptions"},
	} {
		for direction, migrate := range map[string]func(context.Context, []byte) ([]byte, error){
			"upgrade":   new(v18.Version).UpgradeExchange,
			"downgrade": new(v18.Version).DowngradeExchange,
		} {
			t.Run(direction+"/"+name, func(t *testing.T) {
				t.Parallel()
				got, err := migrate(t.Context(), []byte(tc.input))
				require.ErrorContains(t, err, tc.errContains, "migration must report the failing operation")
				assert.Equal(t, tc.input, string(got), "failed migration should preserve the input")
			})
		}
	}
}

func TestRegisteredMigration(t *testing.T) {
	t.Parallel()
	previous := strings.Join(previousDefaults, ",")
	current := strings.Join(currentDefaults, ",")
	// Other exchanges with the same subscriptions are not migrated
	input := []byte(`{"version":17,"exchanges":[{"name":"Coinbase","features":{"subscriptions":[` + previous + `]}},{"name":"Kraken","features":{"subscriptions":[` + previous + `]}}]}`)
	unchanged, err := versions.Manager.Deploy(t.Context(), input, 17)
	require.NoError(t, err, "Deploy must leave an existing v17 configuration valid")
	assert.Equal(t, input, unchanged, "Deploy should retain v17 Coinbase subscriptions until the v18 upgrade")

	out, err := versions.Manager.Deploy(t.Context(), input, 18)
	require.NoError(t, err, "Deploy must apply the registered v18 upgrade")
	expected := `{"version":18,"exchanges":[{"name":"Coinbase","features":{"subscriptions":[` + current + `]}},{"name":"Kraken","features":{"subscriptions":[` + previous + `]}}]}`
	assert.JSONEq(t, expected, string(out), "Deploy should only migrate Coinbase and advance to v18")

	downgraded, err := versions.Manager.Deploy(t.Context(), out, 17)
	require.NoError(t, err, "Deploy must downgrade from v18")
	assert.JSONEq(t, string(input), string(downgraded), "Downgrade should restore the v17 Coinbase subscriptions")

	reupgraded, err := versions.Manager.Deploy(t.Context(), downgraded, 18)
	require.NoError(t, err, "Deploy must reapply v18")
	assert.JSONEq(t, expected, string(reupgraded), "Reapplying v18 should reproduce the upgraded configuration")
}
