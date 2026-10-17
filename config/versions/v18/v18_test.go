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
	tickerEntry    = `{"enabled":true,"channel":"ticker","asset":"all"}`
	tradesEntry    = `{"enabled":true,"channel":"allTrades","asset":"all"}`
	spotCandles    = `{"enabled":true,"channel":"candles","asset":"spot","interval":"1m"}`
	marginCandles  = `{"enabled":true,"channel":"candles","asset":"margin","interval":"1m"}`
	fundingCandles = `{"enabled":true,"channel":"candles","asset":"marginfunding","params":{"_period":"p30"},"interval":"1m"}`
	orderbookEntry = `{"enabled":true,"channel":"orderbook","asset":"all","params":{"prec":"R0"},"levels":100}`
	usdtCandles    = `{"enabled":true,"channel":"candles","asset":"usdtmarginedfutures","interval":"1m"}`
	coinCandles    = `{"enabled":true,"channel":"candles","asset":"coinmarginedfutures","interval":"1m"}`
	usdtStatus     = `{"enabled":true,"channel":"status","asset":"usdtmarginedfutures"}`
	coinStatus     = `{"enabled":true,"channel":"status","asset":"coinmarginedfutures"}`
)

var (
	previousDefaults = []string{tickerEntry, tradesEntry, spotCandles, marginCandles, fundingCandles, orderbookEntry}
	currentDefaults  = []string{tickerEntry, tradesEntry, spotCandles, marginCandles, fundingCandles, usdtCandles, coinCandles, orderbookEntry, usdtStatus, coinStatus}
)

// subscriptions returns a Bitfinex config holding the subscription entries
func subscriptions(entries ...string) string {
	return `{"name":"Bitfinex","features":{"subscriptions":[` + strings.Join(entries, ",") + `]}}`
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
	assert.Equal(t, []string{"Bitfinex"}, new(v18.Version).Exchanges(), "Exchanges should only list Bitfinex")
}

func TestUpgradeExchangeSubscriptions(t *testing.T) {
	t.Parallel()
	for name, entries := range map[string][]string{
		"previous defaults":         previousDefaults,
		"reordered":                 {orderbookEntry, fundingCandles, tickerEntry, marginCandles, tradesEntry, spotCandles},
		"upper case assets":         replaced(previousDefaults, `"all"`, `"ALL"`, `"spot"`, `"SPOT"`, `"margin"`, `"MARGIN"`, `"marginfunding"`, `"MARGINFUNDING"`),
		"title case assets":         replaced(previousDefaults, `"all"`, `"All"`, `"spot"`, `"Spot"`),
		"nanosecond interval":       with(previousDefaults, 2, strings.Replace(spotCandles, `"1m"`, `60000000000`, 1)),
		"seconds interval":          with(previousDefaults, 3, strings.Replace(marginCandles, `"1m"`, `"60s"`, 1)),
		"unit interval":             with(previousDefaults, 4, strings.Replace(fundingCandles, `"1m"`, `"1min"`, 1)),
		"duration interval":         with(previousDefaults, 2, strings.Replace(spotCandles, `"1m"`, `"1m0s"`, 1)),
		"zero ticker interval":      with(previousDefaults, 0, field(tickerEntry, `"interval":"0s"`)),
		"zero trades levels":        with(previousDefaults, 1, field(tradesEntry, `"levels":0`)),
		"null trades levels":        with(previousDefaults, 1, field(tradesEntry, `"levels":null`)),
		"empty ticker params":       with(previousDefaults, 0, field(tickerEntry, `"params":{}`)),
		"null trades params":        with(previousDefaults, 1, field(tradesEntry, `"params":null`)),
		"explicit unauthenticated":  with(with(previousDefaults, 0, field(tickerEntry, `"authenticated":false`)), 5, field(orderbookEntry, `"authenticated":false`)),
		"null authenticated":        with(previousDefaults, 1, field(tradesEntry, `"authenticated":null`)),
		"repeated key last applies": with(previousDefaults, 2, strings.Replace(spotCandles, `"asset":"spot"`, `"asset":"margin","asset":"spot"`, 1)),
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
		"reordered":         {coinStatus, orderbookEntry, usdtCandles, tickerEntry, fundingCandles, usdtStatus, marginCandles, coinCandles, tradesEntry, spotCandles},
		"upper case assets": replaced(currentDefaults, `"usdtmarginedfutures"`, `"USDTMARGINEDFUTURES"`, `"coinmarginedfutures"`, `"COINMARGINEDFUTURES"`),
		"mixed case assets": replaced(currentDefaults, `"usdtmarginedfutures"`, `"USDTMarginedFutures"`, `"all"`, `"All"`),
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
		"empty":                    {},
		"missing entry":            previousDefaults[:5],
		"extra entry":              append(append([]string(nil), previousDefaults...), `{"enabled":true,"channel":"candles","asset":"spot","interval":"5m"}`),
		"repeated entry":           with(previousDefaults, 3, spotCandles),
		"exchange specific name":   with(previousDefaults, 5, strings.Replace(orderbookEntry, `"orderbook"`, `"book"`, 1)),
		"capitalised channel":      with(previousDefaults, 0, strings.Replace(tickerEntry, `"ticker"`, `"Ticker"`, 1)),
		"disabled":                 with(previousDefaults, 0, strings.Replace(tickerEntry, `"enabled":true`, `"enabled":false`, 1)),
		"enabled omitted":          with(previousDefaults, 1, strings.Replace(tradesEntry, `"enabled":true,`, ``, 1)),
		"enabled null":             with(previousDefaults, 2, strings.Replace(spotCandles, `"enabled":true`, `"enabled":null`, 1)),
		"custom interval":          with(previousDefaults, 2, strings.Replace(spotCandles, `"1m"`, `"5m"`, 1)),
		"null interval":            with(previousDefaults, 3, strings.Replace(marginCandles, `"1m"`, `null`, 1)),
		"interval omitted":         with(previousDefaults, 2, strings.Replace(spotCandles, `,"interval":"1m"`, ``, 1)),
		"ticker interval":          with(previousDefaults, 0, field(tickerEntry, `"interval":"1m"`)),
		"quoted interval number":   with(previousDefaults, 2, strings.Replace(spotCandles, `"1m"`, `"60000000000"`, 1)),
		"wildcard candles":         with(previousDefaults, 2, strings.Replace(spotCandles, `"spot"`, `"all"`, 1)),
		"other asset":              with(previousDefaults, 0, strings.Replace(tickerEntry, `"all"`, `"spot"`, 1)),
		"authenticated public":     with(previousDefaults, 0, field(tickerEntry, `"authenticated":true`)),
		"pairs":                    with(previousDefaults, 0, field(tickerEntry, `"pairs":"BTC-USD"`)),
		"custom levels":            with(previousDefaults, 5, strings.Replace(orderbookEntry, `100`, `25`, 1)),
		"null levels":              with(previousDefaults, 5, strings.Replace(orderbookEntry, `100`, `null`, 1)),
		"levels omitted":           with(previousDefaults, 5, strings.Replace(orderbookEntry, `,"levels":100`, ``, 1)),
		"fractional levels":        with(previousDefaults, 5, strings.Replace(orderbookEntry, `100`, `100.5`, 1)),
		"custom precision":         with(previousDefaults, 5, strings.Replace(orderbookEntry, `"R0"`, `"P0"`, 1)),
		"lower case precision":     with(previousDefaults, 5, strings.Replace(orderbookEntry, `"R0"`, `"r0"`, 1)),
		"extra parameter":          with(previousDefaults, 5, strings.Replace(orderbookEntry, `"prec":"R0"`, `"prec":"R0","freq":"F1"`, 1)),
		"funding period omitted":   with(previousDefaults, 4, strings.Replace(fundingCandles, `"params":{"_period":"p30"},`, ``, 1)),
		"custom funding period":    with(previousDefaults, 4, strings.Replace(fundingCandles, `"p30"`, `"p2"`, 1)),
		"ticker parameter":         with(previousDefaults, 0, field(tickerEntry, `"params":{"freq":"F1"}`)),
		"unknown field":            with(previousDefaults, 1, field(tradesEntry, `"custom":"retained"`)),
		"capitalised field":        with(previousDefaults, 1, strings.Replace(tradesEntry, `"enabled"`, `"Enabled"`, 1)),
		"current defaults":         currentDefaults,
		"previous and status only": append(append([]string(nil), previousDefaults...), usdtStatus),
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
		"missing coin margined status":  currentDefaults[:9],
		"disabled usdt margined status": with(currentDefaults, 8, strings.Replace(usdtStatus, `"enabled":true`, `"enabled":false`, 1)),
		"custom perpetual candles":      with(currentDefaults, 5, strings.Replace(usdtCandles, `"1m"`, `"5m"`, 1)),
		"status pairs":                  with(currentDefaults, 9, field(coinStatus, `"pairs":"ETHF0-BTCF0"`)),
		"previous defaults":             previousDefaults,
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

func TestDowngradeDisablesStatusSubscriptions(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct{ entries, exp []string }{
		"wildcard status": {
			entries: with(currentDefaults, 8, strings.Replace(usdtStatus, `"usdtmarginedfutures"`, `"all"`, 1)),
			exp:     with(currentDefaults, 8, `{"enabled":false,"channel":"status","asset":"all"}`),
		},
		"spot status": {
			entries: append(append([]string(nil), previousDefaults...), `{"enabled":true,"channel":"status","asset":"spot","pairs":"BTC-USD"}`),
			exp:     append(append([]string(nil), previousDefaults...), `{"enabled":false,"channel":"status","asset":"spot","pairs":"BTC-USD"}`),
		},
		"perpetual status kept": {
			entries: append(append([]string(nil), previousDefaults...), strings.Replace(usdtStatus, `"usdtmarginedfutures"`, `"USDTMarginedFutures"`, 1)),
			exp:     append(append([]string(nil), previousDefaults...), strings.Replace(usdtStatus, `"usdtmarginedfutures"`, `"USDTMarginedFutures"`, 1)),
		},
		"disabled status kept": {
			entries: append(append([]string(nil), previousDefaults...), `{"enabled":false,"channel":"status","asset":"all"}`),
			exp:     append(append([]string(nil), previousDefaults...), `{"enabled":false,"channel":"status","asset":"all"}`),
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := new(v18.Version).DowngradeExchange(t.Context(), []byte(subscriptions(tc.entries...)))
			require.NoError(t, err, "DowngradeExchange must not error")
			assert.JSONEq(t, subscriptions(tc.exp...), string(got), "DowngradeExchange should disable the status subscriptions the previous version cannot subscribe to")
		})
	}
}

func TestMigrationWithoutSubscriptions(t *testing.T) {
	t.Parallel()
	for _, input := range []string{
		`{"name":"Bitfinex"}`,
		`{"name":"Bitfinex","features":{}}`,
		`{"name":"Bitfinex","features":{"subscriptions":null}}`,
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
		"subscriptions object":  {input: `{"features":{"subscriptions":{}}}`, errContains: "error decoding Bitfinex subscriptions"},
		"subscription string":   {input: `{"features":{"subscriptions":["ticker","allTrades"]}}`, errContains: "error decoding Bitfinex subscriptions"},
		"params string":         {input: subscriptions(with(previousDefaults, 5, strings.Replace(orderbookEntry, `{"prec":"R0"}`, `"R0"`, 1))...), errContains: "error decoding Bitfinex subscriptions"},
		"subscriptions missing": {input: `{"features":{"subscriptions":}}`, errContains: "error getting Bitfinex subscriptions"},
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
	input := []byte(`{"version":17,"exchanges":[{"name":"Bitfinex","features":{"subscriptions":[` + previous + `]}},{"name":"Kraken","features":{"subscriptions":[` + previous + `]}}]}`)
	unchanged, err := versions.Manager.Deploy(t.Context(), input, 17)
	require.NoError(t, err, "Deploy must leave an existing v17 configuration valid")
	assert.Equal(t, input, unchanged, "Deploy should retain v17 Bitfinex subscriptions until the v18 upgrade")

	out, err := versions.Manager.Deploy(t.Context(), input, 18)
	require.NoError(t, err, "Deploy must apply the registered v18 upgrade")
	expected := `{"version":18,"exchanges":[{"name":"Bitfinex","features":{"subscriptions":[` + current + `]}},{"name":"Kraken","features":{"subscriptions":[` + previous + `]}}]}`
	assert.JSONEq(t, expected, string(out), "Deploy should only migrate Bitfinex and advance to v18")

	downgraded, err := versions.Manager.Deploy(t.Context(), out, 17)
	require.NoError(t, err, "Deploy must downgrade from v18")
	assert.JSONEq(t, string(input), string(downgraded), "Downgrade should restore the v17 Bitfinex subscriptions")

	reupgraded, err := versions.Manager.Deploy(t.Context(), downgraded, 18)
	require.NoError(t, err, "Deploy must reapply v18")
	assert.JSONEq(t, expected, string(reupgraded), "Reapplying v18 should reproduce the upgraded configuration")
}
