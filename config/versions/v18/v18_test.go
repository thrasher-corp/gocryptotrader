package v18_test

import (
	"bytes"
	"encoding/json" //nolint:depguard // Config versions must retain stable standard-library JSON behaviour
	"strings"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v18 "github.com/thrasher-corp/gocryptotrader/config/versions/v18"
)

// Kraken's default subscriptions as configs save them
const (
	ticker         = `{"enabled":true,"channel":"ticker","asset":"spot"}`
	allTrades      = `{"enabled":true,"channel":"allTrades","asset":"spot"}`
	candles        = `{"enabled":true,"channel":"candles","asset":"spot","interval":"1m"}`
	orderbook      = `{"enabled":true,"channel":"orderbook","asset":"spot","levels":1000}`
	myOrders       = `{"enabled":true,"channel":"myOrders","authenticated":true}`
	myTrades       = `{"enabled":true,"channel":"myTrades","authenticated":true}`
	futuresTicker  = `{"enabled":true,"channel":"ticker","asset":"futures"}`
	futuresTrades  = `{"enabled":true,"channel":"allTrades","asset":"futures"}`
	futuresBook    = `{"enabled":true,"channel":"orderbook","asset":"futures"}`
	futuresOrders  = `{"enabled":true,"channel":"myOrders","asset":"futures","authenticated":true}`
	futuresFills   = `{"enabled":true,"channel":"myTrades","asset":"futures","authenticated":true}`
	futuresDefault = futuresTicker + "," + futuresTrades + "," + futuresBook + "," + futuresOrders + "," + futuresFills
)

// previousDefaults are Kraken's default subscriptions before futures websocket support
var previousDefaults = []string{ticker, allTrades, candles, orderbook, myOrders, myTrades}

// kraken returns a Kraken exchange config with the websocket endpoints given, keeping its other endpoints, and the
// subscriptions given, if any
func kraken(public, private string, subs ...string) string {
	features := ""
	if len(subs) != 0 {
		features = `"features":{"enabled":{"websocketAPI":true},"subscriptions":[` + strings.Join(subs, ",") + `]},`
	}
	return `{"name":"Kraken",` + features + `"api":{"urlEndpoints":{"RestFuturesSupplementaryURL":"https://futures.kraken.com/api/","RestFuturesURL":"https://futures.kraken.com/derivatives","RestSpotURL":"https://api.kraken.com","WebsocketSpotSupplementaryURL":"` + private + `","WebsocketSpotURL":"` + public + `"}},"custom":"retained"}`
}

// subscriptions returns a Kraken exchange config with the current websocket endpoints and the subscriptions given
func subscriptions(subs ...string) string {
	return kraken("wss://ws.kraken.com/v2", "wss://ws-auth.kraken.com/v2", subs...)
}

// downgraded returns a Kraken exchange config with the previous websocket endpoints and the subscriptions given
func downgraded(subs ...string) string {
	return kraken("wss://ws.kraken.com", "wss://ws-auth.kraken.com", subs...)
}

// replaced returns the previous default subscriptions with the subscription at i replaced by sub
func replaced(i int, sub string) []string {
	subs := append([]string(nil), previousDefaults...)
	subs[i] = sub
	return subs
}

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"Kraken"}, new(v18.Version).Exchanges(), "Exchanges should return just Kraken")
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, in, exp string
	}{
		{"previous defaults", kraken("wss://ws.kraken.com", "wss://ws-auth.kraken.com"), kraken("wss://ws.kraken.com/v2", "wss://ws-auth.kraken.com/v2")},
		{"beta endpoints", kraken("wss://beta-ws.kraken.com", "wss://beta-ws-auth.kraken.com"), kraken("wss://beta-ws.kraken.com/v2", "wss://beta-ws-auth.kraken.com/v2")},
		{"trailing slashes", kraken("wss://ws.kraken.com/", "wss://ws-auth.kraken.com/"), kraken("wss://ws.kraken.com/v2", "wss://ws-auth.kraken.com/v2")},
		{"current endpoints", kraken("wss://ws.kraken.com/v2", "wss://ws-auth.kraken.com/v2"), kraken("wss://ws.kraken.com/v2", "wss://ws-auth.kraken.com/v2")},
		{"custom endpoints", kraken("wss://proxy.example.com", "wss://ws-auth.example.com"), kraken("wss://proxy.example.com", "wss://ws-auth.example.com")},
		{"one custom endpoint", kraken("wss://ws.kraken.com", "wss://proxy.example.com"), kraken("wss://ws.kraken.com/v2", "wss://proxy.example.com")},
		{"public endpoint only", `{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.kraken.com"}}}`, `{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.kraken.com/v2"}}}`},
		{"private endpoint only", `{"api":{"urlEndpoints":{"WebsocketSpotSupplementaryURL":"wss://ws-auth.kraken.com"}}}`, `{"api":{"urlEndpoints":{"WebsocketSpotSupplementaryURL":"wss://ws-auth.kraken.com/v2"}}}`},
		{"empty endpoint", kraken("", ""), kraken("", "")},
		{"null endpoint", `{"api":{"urlEndpoints":{"WebsocketSpotURL":null}}}`, `{"api":{"urlEndpoints":{"WebsocketSpotURL":null}}}`},
		{"v1 endpoint under another key", `{"api":{"urlEndpoints":{"RestSpotURL":"wss://ws.kraken.com"}}}`, `{"api":{"urlEndpoints":{"RestSpotURL":"wss://ws.kraken.com"}}}`},
		{"no endpoints", `{"name":"Kraken","api":{}}`, `{"name":"Kraken","api":{}}`},
		{"no api", `{"name":"Kraken"}`, `{"name":"Kraken"}`},
		{"endpoints and subscriptions", kraken("wss://ws.kraken.com", "wss://ws-auth.kraken.com", previousDefaults...), kraken("wss://ws.kraken.com/v2", "wss://ws-auth.kraken.com/v2", append(previousDefaults, futuresDefault)...)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := new(v18.Version).UpgradeExchange(t.Context(), []byte(tc.in))
			require.NoError(t, err, "UpgradeExchange must not error")
			assert.JSONEq(t, tc.exp, string(got), "UpgradeExchange should move only Kraken's websocket v1 endpoints")
			again, err := new(v18.Version).UpgradeExchange(t.Context(), bytes.Clone(got))
			require.NoError(t, err, "repeated UpgradeExchange must not error")
			assert.Equal(t, got, again, "UpgradeExchange should be idempotent")
		})
	}
}

func TestUpgradeExchangeSubscriptions(t *testing.T) {
	t.Parallel()
	explicitZeros := `{"enabled":true,"channel":"ticker","pairs":"","asset":"spot","params":{},"interval":0,"levels":0,"authenticated":false,"qualifiedChannel":"ignored"}`
	for _, tc := range []struct {
		name string
		subs []string
	}{
		{"previous defaults", previousDefaults},
		{"another order", []string{myTrades, orderbook, ticker, candles, myOrders, allTrades}},
		{"capitalised asset", replaced(0, `{"enabled":true,"channel":"ticker","asset":"Spot"}`)},
		{"upper case asset", replaced(0, `{"enabled":true,"channel":"ticker","asset":"SPOT"}`)},
		{"capitalised field names", replaced(0, `{"Enabled":true,"Channel":"ticker","Asset":"spot"}`)},
		{"explicit zero values", replaced(0, explicitZeros)},
		{"interval in seconds", replaced(2, `{"enabled":true,"channel":"candles","asset":"spot","interval":"60s"}`)},
		{"interval in minutes and seconds", replaced(2, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1m0s"}`)},
		{"interval in nanoseconds", replaced(2, `{"enabled":true,"channel":"candles","asset":"spot","interval":60000000000}`)},
		{"interval unit spelt out", replaced(2, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1minute"}`)},
		{"interval unit in upper case", replaced(2, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1MIN"}`)},
		{"interval in upper case seconds", replaced(2, `{"enabled":true,"channel":"candles","asset":"spot","interval":"60S"}`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := new(v18.Version).UpgradeExchange(t.Context(), []byte(subscriptions(tc.subs...)))
			require.NoError(t, err, "UpgradeExchange must not error")
			exp := subscriptions(append(tc.subs, futuresDefault)...)
			assert.JSONEq(t, exp, string(got), "UpgradeExchange should add the futures subscriptions, keeping the saved ones")
			again, err := new(v18.Version).UpgradeExchange(t.Context(), bytes.Clone(got))
			require.NoError(t, err, "repeated UpgradeExchange must not error")
			assert.Equal(t, got, again, "UpgradeExchange should be idempotent")
		})
	}
}

func TestUpgradeExchangeCustomSubscriptions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, in string
	}{
		{"current defaults", subscriptions(append(previousDefaults, futuresDefault)...)},
		{"futures ticker added", subscriptions(append(previousDefaults, futuresTicker)...)},
		{"level 3 book added", subscriptions(append(previousDefaults, `{"enabled":true,"channel":"allOrders","asset":"spot","levels":10}`)...)},
		{"default removed", subscriptions(previousDefaults[1:]...)},
		{"default duplicated", subscriptions(replaced(1, ticker)...)},
		{"disabled", subscriptions(replaced(0, `{"enabled":false,"channel":"ticker","asset":"spot"}`)...)},
		{"enabled omitted", subscriptions(replaced(0, `{"channel":"ticker","asset":"spot"}`)...)},
		{"capitalised channel", subscriptions(replaced(0, `{"enabled":true,"channel":"Ticker","asset":"spot"}`)...)},
		{"another asset", subscriptions(replaced(0, `{"enabled":true,"channel":"ticker","asset":"all"}`)...)},
		{"pairs", subscriptions(replaced(0, `{"enabled":true,"channel":"ticker","asset":"spot","pairs":"XBT/USD"}`)...)},
		{"params", subscriptions(replaced(0, `{"enabled":true,"channel":"ticker","asset":"spot","params":{"event_trigger":"bbo"}}`)...)},
		{"another depth", subscriptions(replaced(3, `{"enabled":true,"channel":"orderbook","asset":"spot","levels":10}`)...)},
		{"another interval", subscriptions(replaced(2, `{"enabled":true,"channel":"candles","asset":"spot","interval":"5m"}`)...)},
		{"monthly interval", subscriptions(replaced(2, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1M"}`)...)},
		{"invalid interval", subscriptions(replaced(2, `{"enabled":true,"channel":"candles","asset":"spot","interval":"1 minute"}`)...)},
		{"raw interval", subscriptions(replaced(2, `{"enabled":true,"channel":"candles","asset":"spot","interval":"raw"}`)...)},
		// Scaling 2^53+1 minutes wraps to one minute, but the runtime rejects intervals that overflow
		{"overflowing interval", subscriptions(replaced(2, `{"enabled":true,"channel":"candles","asset":"spot","interval":"9007199254740993m"}`)...)},
		{"unauthenticated", subscriptions(replaced(4, `{"enabled":true,"channel":"myOrders"}`)...)},
		{"no subscriptions", subscriptions()},
		{"empty subscriptions", `{"name":"Kraken","features":{"subscriptions":[]}}`},
		{"null subscriptions", `{"name":"Kraken","features":{"subscriptions":null}}`},
		{"no features", `{"name":"Kraken"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := new(v18.Version).UpgradeExchange(t.Context(), []byte(tc.in))
			require.NoError(t, err, "UpgradeExchange must not error")
			assert.JSONEq(t, tc.in, string(got), "UpgradeExchange should keep subscriptions other than the previous defaults")
		})
	}
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, in, exp string
	}{
		{"current defaults", kraken("wss://ws.kraken.com/v2", "wss://ws-auth.kraken.com/v2"), kraken("wss://ws.kraken.com", "wss://ws-auth.kraken.com")},
		{"beta endpoints", kraken("wss://beta-ws.kraken.com/v2", "wss://beta-ws-auth.kraken.com/v2"), kraken("wss://beta-ws.kraken.com", "wss://beta-ws-auth.kraken.com")},
		{"trailing slashes", kraken("wss://ws.kraken.com/v2/", "wss://ws-auth.kraken.com/v2/"), kraken("wss://ws.kraken.com", "wss://ws-auth.kraken.com")},
		{"previous endpoints", kraken("wss://ws.kraken.com", "wss://ws-auth.kraken.com"), kraken("wss://ws.kraken.com", "wss://ws-auth.kraken.com")},
		{"custom endpoints", kraken("wss://proxy.example.com/v2", "wss://ws-auth.example.com/v2"), kraken("wss://proxy.example.com/v2", "wss://ws-auth.example.com/v2")},
		{"null endpoint", `{"api":{"urlEndpoints":{"WebsocketSpotSupplementaryURL":null}}}`, `{"api":{"urlEndpoints":{"WebsocketSpotSupplementaryURL":null}}}`},
		{"no api", `{"name":"Kraken"}`, `{"name":"Kraken"}`},
		{"endpoints and subscriptions", kraken("wss://ws.kraken.com/v2", "wss://ws-auth.kraken.com/v2", append(previousDefaults, futuresDefault)...), kraken("wss://ws.kraken.com", "wss://ws-auth.kraken.com", previousDefaults...)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := new(v18.Version).DowngradeExchange(t.Context(), []byte(tc.in))
			require.NoError(t, err, "DowngradeExchange must not error")
			assert.JSONEq(t, tc.exp, string(got), "DowngradeExchange should move only Kraken's websocket v2 endpoints")
		})
	}
}

func TestDowngradeExchangeSubscriptions(t *testing.T) {
	t.Parallel()
	disabledTicker := `{"enabled":false,"channel":"ticker","asset":"futures"}`
	level3 := `{"enabled":true,"channel":"allOrders","asset":"spot","levels":10}`
	for _, tc := range []struct {
		name, in, exp string
	}{
		{"current defaults", subscriptions(append(previousDefaults, futuresDefault)...), downgraded(previousDefaults...)},
		{"another order", subscriptions(futuresFills, myTrades, futuresTicker, orderbook, ticker, futuresBook, candles, futuresOrders, myOrders, futuresTrades, allTrades), downgraded(myTrades, orderbook, ticker, candles, myOrders, allTrades)},
		{"upper case asset", subscriptions(append(previousDefaults, `{"enabled":true,"channel":"ticker","asset":"FUTURES"}`, futuresTrades, futuresBook, futuresOrders, futuresFills)...), downgraded(previousDefaults...)},
		{"previous defaults", subscriptions(previousDefaults...), downgraded(previousDefaults...)},
		{"futures subscription disabled", subscriptions(append(previousDefaults, disabledTicker, futuresTrades, futuresBook, futuresOrders, futuresFills)...), downgraded(append(previousDefaults, disabledTicker, futuresTrades, futuresBook, futuresOrders, futuresFills)...)},
		{"futures subscriptions removed", subscriptions(append(previousDefaults, futuresTicker)...), downgraded(append(previousDefaults, futuresTicker)...)},
		{"level 3 book added", subscriptions(append(previousDefaults, futuresDefault, level3)...), downgraded(append(previousDefaults, futuresDefault, level3)...)},
		{"no features", `{"name":"Kraken"}`, `{"name":"Kraken"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := new(v18.Version).DowngradeExchange(t.Context(), []byte(tc.in))
			require.NoError(t, err, "DowngradeExchange must not error")
			assert.JSONEq(t, tc.exp, string(got), "DowngradeExchange should remove the futures subscriptions only from the current defaults")
		})
	}
}

func TestMigrationMalformed(t *testing.T) {
	t.Parallel()
	_, err := new(v18.Version).UpgradeExchange(t.Context(), []byte(`{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.kraken.com`))
	assert.ErrorIs(t, err, jsonparser.MalformedStringError, "UpgradeExchange should reject a malformed exchange config")
	_, err = new(v18.Version).DowngradeExchange(t.Context(), []byte(`{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.kraken.com/v2`))
	assert.ErrorIs(t, err, jsonparser.MalformedStringError, "DowngradeExchange should reject a malformed exchange config")

	for _, subs := range []string{`{}`, `1`, `[1]`, `[{"levels":"1000"}]`} {
		var typeErr *json.UnmarshalTypeError
		_, err = new(v18.Version).UpgradeExchange(t.Context(), []byte(`{"features":{"subscriptions":`+subs+`}}`))
		assert.ErrorAsf(t, err, &typeErr, "UpgradeExchange should reject subscriptions %s", subs)
		_, err = new(v18.Version).DowngradeExchange(t.Context(), []byte(`{"features":{"subscriptions":`+subs+`}}`))
		assert.ErrorAsf(t, err, &typeErr, "DowngradeExchange should reject subscriptions %s", subs)
	}
}

func TestRegisteredMigration(t *testing.T) {
	t.Parallel()
	binance := `{"name":"Binance","api":{"urlEndpoints":{"WebsocketSpotURL":"wss://ws.kraken.com"}},"features":{"subscriptions":[` + strings.Join(previousDefaults, ",") + `]}}`
	v17 := `{"version":17,"exchanges":[` + binance + `,` + kraken("wss://ws.kraken.com", "wss://ws-auth.kraken.com", previousDefaults...) + `]}`
	upgraded := `{"version":18,"exchanges":[` + binance + `,` + kraken("wss://ws.kraken.com/v2", "wss://ws-auth.kraken.com/v2", append(previousDefaults, futuresDefault)...) + `]}`
	got, err := versions.Manager.Deploy(t.Context(), []byte(v17), 18)
	require.NoError(t, err, "Deploy must apply the registered v18 upgrade")
	assert.JSONEq(t, upgraded, string(got), "Deploy should migrate only Kraken's websocket endpoints and subscriptions")

	got, err = versions.Manager.Deploy(t.Context(), got, 17)
	require.NoError(t, err, "Deploy must downgrade v18 to v17")
	assert.JSONEq(t, v17, string(got), "Deploy should restore the websocket v1 endpoints and the previous subscriptions")
}
