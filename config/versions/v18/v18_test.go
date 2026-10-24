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
	spotOrderbook  = `{"enabled":true,"channel":"orderbook","asset":"spot","levels":20}`
	spotCandles    = `{"enabled":true,"channel":"candles","asset":"spot","interval":"15m"}`
	spotTicker     = `{"enabled":true,"channel":"ticker","asset":"spot","interval":"100ms"}`
	spotMiniTicker = `{"enabled":true,"channel":"public.miniTicker.v3.api.pb","asset":"spot"}`
	spotTrades     = `{"enabled":true,"channel":"allTrades","asset":"spot","interval":"100ms"}`
	spotMyTrades   = `{"enabled":true,"channel":"myTrades","asset":"spot","authenticated":true}`
	spotOrders     = `{"enabled":true,"channel":"myOrders","asset":"spot","authenticated":true}`
	spotAccount    = `{"enabled":true,"channel":"myAccount","asset":"spot","authenticated":true}`

	futuresTicker    = `{"enabled":true,"channel":"ticker","asset":"futures"}`
	futuresOrderbook = `{"enabled":true,"channel":"orderbook","asset":"futures","levels":20}`
	futuresTrades    = `{"enabled":true,"channel":"allTrades","asset":"futures"}`
	futuresCandles   = `{"enabled":true,"channel":"candles","asset":"futures","interval":"15m"}`
	futuresOrders    = `{"enabled":true,"channel":"myOrders","asset":"futures","authenticated":true}`
	futuresMyTrades  = `{"enabled":true,"channel":"myTrades","asset":"futures","authenticated":true}`
	futuresAccount   = `{"enabled":true,"channel":"myAccount","asset":"futures","authenticated":true}`
	futuresPosition  = `{"enabled":true,"channel":"position","asset":"futures","authenticated":true}`
)

var (
	previousDefaults = []string{spotOrderbook, spotCandles, spotTicker, spotMiniTicker, spotTrades, spotMyTrades, spotOrders, spotAccount}
	currentDefaults  = append(append([]string(nil), previousDefaults...),
		futuresTicker, futuresOrderbook, futuresTrades, futuresCandles, futuresOrders, futuresMyTrades, futuresAccount, futuresPosition)
)

// subscriptions returns a MEXC config holding the subscription entries
func subscriptions(entries ...string) string {
	return `{"name":"MEXC","features":{"subscriptions":[` + strings.Join(entries, ",") + `]}}`
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

// reversed returns entries in reverse order
func reversed(entries []string) []string {
	out := make([]string, len(entries))
	for i := range entries {
		out[len(entries)-1-i] = entries[i]
	}
	return out
}

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"MEXC"}, new(v18.Version).Exchanges(), "Exchanges should only list MEXC")
}

func TestUpgradeExchangeSubscriptions(t *testing.T) {
	t.Parallel()
	for name, entries := range map[string][]string{
		"previous defaults":        previousDefaults,
		"reordered":                reversed(previousDefaults),
		"upper case assets":        replaced(previousDefaults, `"spot"`, `"SPOT"`),
		"title case assets":        replaced(previousDefaults, `"spot"`, `"Spot"`),
		"nanosecond interval":      with(previousDefaults, 1, strings.Replace(spotCandles, `"15m"`, `900000000000`, 1)),
		"seconds interval":         with(previousDefaults, 2, strings.Replace(spotTicker, `"100ms"`, `"0.1s"`, 1)),
		"long form interval":       with(previousDefaults, 1, strings.Replace(spotCandles, `"15m"`, `"15m0s"`, 1)),
		"zero interval":            with(previousDefaults, 3, field(spotMiniTicker, `"interval":"0s"`)),
		"zero levels":              with(previousDefaults, 3, field(spotMiniTicker, `"levels":0`)),
		"null levels":              with(previousDefaults, 3, field(spotMiniTicker, `"levels":null`)),
		"explicit unauthenticated": with(previousDefaults, 0, field(spotOrderbook, `"authenticated":false`)),
		"null authenticated":       with(previousDefaults, 4, field(spotTrades, `"authenticated":null`)),
		"repeated key last applies": with(previousDefaults, 0,
			strings.Replace(spotOrderbook, `"asset":"spot"`, `"asset":"futures","asset":"spot"`, 1)),
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
		"reordered":         reversed(currentDefaults),
		"upper case assets": replaced(currentDefaults, `"spot"`, `"SPOT"`, `"futures"`, `"FUTURES"`),
		"mixed case assets": replaced(currentDefaults, `"spot"`, `"Spot"`, `"futures"`, `"Futures"`),
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
		"empty":                     {},
		"missing entry":             previousDefaults[:7],
		"extra entry":               append(append([]string(nil), previousDefaults...), `{"enabled":true,"channel":"public.bookTicker.batch.v3.api.pb","asset":"spot"}`),
		"repeated entry":            with(previousDefaults, 7, spotOrders),
		"futures subscribed":        append(append([]string(nil), previousDefaults...), futuresTicker),
		"disabled":                  with(previousDefaults, 0, strings.Replace(spotOrderbook, `"enabled":true`, `"enabled":false`, 1)),
		"enabled omitted":           with(previousDefaults, 1, strings.Replace(spotCandles, `"enabled":true,`, ``, 1)),
		"enabled null":              with(previousDefaults, 2, strings.Replace(spotTicker, `"enabled":true`, `"enabled":null`, 1)),
		"custom levels":             with(previousDefaults, 0, strings.Replace(spotOrderbook, `20`, `5`, 1)),
		"levels omitted":            with(previousDefaults, 0, strings.Replace(spotOrderbook, `,"levels":20`, ``, 1)),
		"ticker levels":             with(previousDefaults, 2, field(spotTicker, `"levels":20`)),
		"custom interval":           with(previousDefaults, 1, strings.Replace(spotCandles, `"15m"`, `"1h"`, 1)),
		"day interval":              with(previousDefaults, 1, strings.Replace(spotCandles, `"15m"`, `"1d"`, 1)),
		"null interval":             with(previousDefaults, 1, strings.Replace(spotCandles, `"15m"`, `null`, 1)),
		"interval omitted":          with(previousDefaults, 2, strings.Replace(spotTicker, `,"interval":"100ms"`, ``, 1)),
		"mini ticker interval":      with(previousDefaults, 3, field(spotMiniTicker, `"interval":"100ms"`)),
		"quoted nanosecond":         with(previousDefaults, 1, strings.Replace(spotCandles, `"15m"`, `"900000000000"`, 1)),
		"futures asset":             with(previousDefaults, 1, strings.Replace(spotCandles, `"spot"`, `"futures"`, 1)),
		"wildcard asset":            with(previousDefaults, 0, strings.Replace(spotOrderbook, `"spot"`, `"all"`, 1)),
		"channel case":              with(previousDefaults, 4, strings.Replace(spotTrades, `"allTrades"`, `"alltrades"`, 1)),
		"authenticated public":      with(previousDefaults, 0, field(spotOrderbook, `"authenticated":true`)),
		"unauthenticated private":   with(previousDefaults, 5, strings.Replace(spotMyTrades, `,"authenticated":true`, ``, 1)),
		"private explicitly public": with(previousDefaults, 6, strings.Replace(spotOrders, `"authenticated":true`, `"authenticated":false`, 1)),
		"pairs":                     with(previousDefaults, 0, field(spotOrderbook, `"pairs":"BTC-USDT"`)),
		"params":                    with(previousDefaults, 1, field(spotCandles, `"params":{}`)),
		"unknown field":             with(previousDefaults, 7, field(spotAccount, `"custom":"retained"`)),
		"capitalised field":         with(previousDefaults, 4, strings.Replace(spotTrades, `"enabled"`, `"Enabled"`, 1)),
		"current defaults":          currentDefaults,
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
		"missing position":         currentDefaults[:15],
		"disabled futures ticker":  with(currentDefaults, 8, strings.Replace(futuresTicker, `"enabled":true`, `"enabled":false`, 1)),
		"custom futures levels":    with(currentDefaults, 9, strings.Replace(futuresOrderbook, `20`, `5`, 1)),
		"futures candles interval": with(currentDefaults, 11, strings.Replace(futuresCandles, `"15m"`, `"4h"`, 1)),
		"spot position":            with(currentDefaults, 15, strings.Replace(futuresPosition, `"futures"`, `"spot"`, 1)),
		"previous defaults":        previousDefaults,
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
		`{"name":"MEXC"}`,
		`{"name":"MEXC","features":{}}`,
		`{"name":"MEXC","features":{"subscriptions":null}}`,
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
	for name, tc := range map[string]struct{ upgrade, downgrade, errContains string }{
		"subscriptions object": {
			upgrade:     `{"features":{"subscriptions":{}}}`,
			downgrade:   `{"features":{"subscriptions":{}}}`,
			errContains: "error decoding MEXC subscriptions",
		},
		"subscription string": {
			upgrade:     `{"features":{"subscriptions":["orderbook","candles"]}}`,
			downgrade:   `{"features":{"subscriptions":["orderbook","candles"]}}`,
			errContains: "error decoding MEXC subscriptions",
		},
		"enabled string": {
			upgrade:     subscriptions(with(previousDefaults, 0, strings.Replace(spotOrderbook, `"enabled":true`, `"enabled":"true"`, 1))...),
			downgrade:   subscriptions(with(currentDefaults, 8, strings.Replace(futuresTicker, `"enabled":true`, `"enabled":"true"`, 1))...),
			errContains: "error decoding MEXC subscriptions",
		},
		"fractional levels": {
			upgrade:     subscriptions(with(previousDefaults, 0, strings.Replace(spotOrderbook, `20`, `20.5`, 1))...),
			downgrade:   subscriptions(with(currentDefaults, 9, strings.Replace(futuresOrderbook, `20`, `20.5`, 1))...),
			errContains: "error decoding MEXC subscriptions",
		},
		"subscriptions missing": {
			upgrade:     `{"features":{"subscriptions":}}`,
			downgrade:   `{"features":{"subscriptions":}}`,
			errContains: "error getting MEXC subscriptions",
		},
	} {
		for direction, migrate := range map[string]struct {
			input string
			run   func(context.Context, []byte) ([]byte, error)
		}{
			"upgrade":   {tc.upgrade, new(v18.Version).UpgradeExchange},
			"downgrade": {tc.downgrade, new(v18.Version).DowngradeExchange},
		} {
			t.Run(direction+"/"+name, func(t *testing.T) {
				t.Parallel()
				got, err := migrate.run(t.Context(), []byte(migrate.input))
				require.ErrorContains(t, err, tc.errContains, "migration must report the failing operation")
				assert.Equal(t, migrate.input, string(got), "failed migration should preserve the input")
			})
		}
	}
}

func TestRegisteredMigration(t *testing.T) {
	t.Parallel()
	previous := strings.Join(previousDefaults, ",")
	current := strings.Join(currentDefaults, ",")
	// Other exchanges with the same subscriptions are not migrated
	input := []byte(`{"version":17,"exchanges":[{"name":"MEXC","features":{"subscriptions":[` + previous + `]}},{"name":"Kraken","features":{"subscriptions":[` + previous + `]}}]}`)
	unchanged, err := versions.Manager.Deploy(t.Context(), input, 17)
	require.NoError(t, err, "Deploy must leave an existing v17 configuration valid")
	assert.Equal(t, input, unchanged, "Deploy should retain v17 MEXC subscriptions until the v18 upgrade")

	out, err := versions.Manager.Deploy(t.Context(), input, 18)
	require.NoError(t, err, "Deploy must apply the registered v18 upgrade")
	expected := `{"version":18,"exchanges":[{"name":"MEXC","features":{"subscriptions":[` + current + `]}},{"name":"Kraken","features":{"subscriptions":[` + previous + `]}}]}`
	assert.JSONEq(t, expected, string(out), "Deploy should only migrate MEXC and advance to v18")

	downgraded, err := versions.Manager.Deploy(t.Context(), out, 17)
	require.NoError(t, err, "Deploy must downgrade from v18")
	assert.JSONEq(t, string(input), string(downgraded), "Downgrade should restore the v17 MEXC subscriptions")

	reupgraded, err := versions.Manager.Deploy(t.Context(), downgraded, 18)
	require.NoError(t, err, "Deploy must reapply v18")
	assert.JSONEq(t, expected, string(reupgraded), "Reapplying v18 should reproduce the upgraded configuration")
}
