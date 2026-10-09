package v18

import (
	"encoding/json" //nolint:depguard // Config versions must retain stable standard-library JSON behaviour
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw string
		exp time.Duration
		ok  bool
	}{
		{raw: "", ok: true},
		{raw: `0`, ok: true},
		{raw: `60000000000`, exp: time.Minute, ok: true},
		{raw: `"1m"`, exp: time.Minute, ok: true},
		{raw: `"60s"`, exp: time.Minute, ok: true},
		{raw: `"1m0s"`, exp: time.Minute, ok: true},
		{raw: `"1MIN"`, exp: time.Minute, ok: true},
		{raw: `"60S"`, exp: time.Minute, ok: true},
		{raw: `"1minute"`, exp: time.Minute, ok: true},
		{raw: `"2w"`, exp: 14 * 24 * time.Hour, ok: true},
		{raw: `"1M"`, exp: 30 * 24 * time.Hour, ok: true},
		{raw: `"1mo"`, exp: 30 * 24 * time.Hour, ok: true},
		{raw: `"0d"`, ok: true},
		{raw: `"raw"`},
		{raw: `null`},
		{raw: `"1 minute"`},
		{raw: `"m"`},
		{raw: `"5x"`},
		{raw: `"99999999999999999999m"`},
		{raw: `"9007199254740993m"`},
	} {
		interval, ok := parseInterval(json.RawMessage(tc.raw))
		assert.Equalf(t, tc.ok, ok, "parseInterval should report whether %q has a duration", tc.raw)
		assert.Equalf(t, tc.exp, interval, "parseInterval should parse %q as the runtime does", tc.raw)
	}
}

func TestSubscriptionMatches(t *testing.T) {
	t.Parallel()
	candles := subscription{channel: "candles", asset: "spot", interval: time.Minute}
	minute := json.RawMessage(`"1m"`)
	for _, tc := range []struct {
		name  string
		saved savedSubscription
		exp   bool
	}{
		{name: "as saved", saved: savedSubscription{Enabled: true, Channel: "candles", Asset: "spot", Interval: minute}, exp: true},
		{name: "with the asset in upper case", saved: savedSubscription{Enabled: true, Channel: "candles", Asset: "SPOT", Interval: minute}, exp: true},
		{name: "with the interval in seconds", saved: savedSubscription{Enabled: true, Channel: "candles", Asset: "spot", Interval: json.RawMessage(`"60s"`)}, exp: true},
		{name: "with empty parameters", saved: savedSubscription{Enabled: true, Channel: "candles", Asset: "spot", Interval: minute, Params: map[string]any{}}, exp: true},
		{name: "disabled", saved: savedSubscription{Channel: "candles", Asset: "spot", Interval: minute}},
		{name: "with a capitalised channel", saved: savedSubscription{Enabled: true, Channel: "Candles", Asset: "spot", Interval: minute}},
		{name: "for another asset", saved: savedSubscription{Enabled: true, Channel: "candles", Asset: "futures", Interval: minute}},
		{name: "with pairs", saved: savedSubscription{Enabled: true, Channel: "candles", Asset: "spot", Interval: minute, Pairs: "XBT/USD"}},
		{name: "with parameters", saved: savedSubscription{Enabled: true, Channel: "candles", Asset: "spot", Interval: minute, Params: map[string]any{"snapshot": false}}},
		{name: "with levels", saved: savedSubscription{Enabled: true, Channel: "candles", Asset: "spot", Interval: minute, Levels: 10}},
		{name: "authenticated", saved: savedSubscription{Enabled: true, Channel: "candles", Asset: "spot", Interval: minute, Authenticated: true}},
		{name: "with another interval", saved: savedSubscription{Enabled: true, Channel: "candles", Asset: "spot", Interval: json.RawMessage(`"5m"`)}},
		{name: "with an interval without a duration", saved: savedSubscription{Enabled: true, Channel: "candles", Asset: "spot", Interval: json.RawMessage(`"raw"`)}},
	} {
		assert.Equalf(t, tc.exp, candles.matches(&tc.saved), "matches should report whether the subscription %s is the default", tc.name)
	}
}

func TestMatchDefaults(t *testing.T) {
	t.Parallel()
	saved := func(defaults ...subscription) []savedSubscription {
		subs := make([]savedSubscription, len(defaults))
		for i, d := range defaults {
			subs[i] = savedSubscription{Enabled: true, Channel: d.channel, Asset: d.asset, Levels: d.levels, Authenticated: d.authenticated}
			if d.interval != 0 {
				subs[i].Interval = json.RawMessage(`"` + d.interval.String() + `"`)
			}
		}
		return subs
	}
	previous := previousSubscriptions
	matched, ok := matchDefaults(saved(previous[5], previous[4], previous[3], previous[2], previous[1], previous[0]), previous)
	require.True(t, ok, "matchDefaults must match the defaults in another order")
	assert.Equal(t, []int{5, 4, 3, 2, 1, 0}, matched, "matchDefaults should return the index of each saved subscription's default")

	current := slices.Concat(previousSubscriptions, futuresSubscriptions)
	matched, ok = matchDefaults(saved(current...), current)
	require.True(t, ok, "matchDefaults must match the current defaults")
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, matched, "matchDefaults should tell the futures defaults from the spot ones")

	for name, subs := range map[string][]savedSubscription{
		"a default missing":    saved(previous[1:]...),
		"a default duplicated": saved(previous[0], previous[0], previous[2], previous[3], previous[4], previous[5]),
		"another subscription": saved(slices.Concat(previous, futuresSubscriptions[:1])...),
		"no subscriptions":     nil,
	} {
		_, ok := matchDefaults(subs, previous)
		assert.Falsef(t, ok, "matchDefaults should not match %s", name)
	}
}

func TestMigrateEndpoints(t *testing.T) {
	t.Parallel()
	var offered []string
	migrate := func(endpoint string) (string, bool) {
		offered = append(offered, endpoint)
		return endpoint + "/next", endpoint == "wss://a"
	}
	got, err := migrateEndpoints([]byte(`{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://a","WebsocketSpotSupplementaryURL":"wss://b","RestSpotURL":"wss://a"}}}`), migrate)
	require.NoError(t, err, "migrateEndpoints must not error")
	assert.JSONEq(t, `{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://a/next","WebsocketSpotSupplementaryURL":"wss://b","RestSpotURL":"wss://a"}}}`, string(got), "migrateEndpoints should replace only the websocket endpoints migrate replaces")
	assert.Equal(t, []string{"wss://a", "wss://b"}, offered, "migrateEndpoints should offer migrate each websocket endpoint")

	offered = nil
	got, err = migrateEndpoints([]byte(`{"api":{"urlEndpoints":{"WebsocketSpotURL":5}}}`), migrate)
	require.NoError(t, err, "migrateEndpoints must not error for an endpoint that is not a string")
	assert.JSONEq(t, `{"api":{"urlEndpoints":{"WebsocketSpotURL":5}}}`, string(got), "migrateEndpoints should keep an endpoint that is not a string")
	assert.Empty(t, offered, "migrateEndpoints should not offer migrate an endpoint that is not a string")

	_, err = migrateEndpoints([]byte(`{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss://a`), migrate)
	assert.ErrorIs(t, err, jsonparser.MalformedStringError, "migrateEndpoints should reject an unterminated endpoint")
	_, err = migrateEndpoints([]byte(`{"api":{"urlEndpoints":{"WebsocketSpotURL":"wss:\q"}}}`), migrate)
	assert.ErrorIs(t, err, jsonparser.MalformedValueError, "migrateEndpoints should reject an endpoint with an invalid escape")
}

func TestMigrateSubscriptions(t *testing.T) {
	t.Parallel()
	previous := `{"enabled":true,"channel":"ticker","asset":"spot"},{"enabled":true,"channel":"allTrades","asset":"spot"},` +
		`{"enabled":true,"channel":"candles","asset":"spot","interval":"1m"},{"enabled":true,"channel":"orderbook","asset":"spot","levels":1000},` +
		`{"enabled":true,"channel":"myOrders","authenticated":true},{"enabled":true,"channel":"myTrades","authenticated":true}`
	futures := `{"enabled":true,"channel":"ticker","asset":"futures"},{"enabled":true,"channel":"allTrades","asset":"futures"},` +
		`{"enabled":true,"channel":"orderbook","asset":"futures"},{"enabled":true,"channel":"myOrders","asset":"futures","authenticated":true},` +
		`{"enabled":true,"channel":"myTrades","asset":"futures","authenticated":true}`
	features := func(subs ...string) string {
		return `{"features":{"subscriptions":[` + strings.Join(subs, ",") + `]}}`
	}
	for _, tc := range []struct {
		name, in, exp string
		upgrade       bool
	}{
		{name: "upgrading the previous defaults", in: features(previous), exp: features(previous, futures), upgrade: true},
		{name: "upgrading the current defaults", in: features(previous, futures), exp: features(previous, futures), upgrade: true},
		{name: "downgrading the current defaults", in: features(previous, futures), exp: features(previous)},
		{name: "downgrading the previous defaults", in: features(previous), exp: features(previous)},
		{name: "upgrading without subscriptions", in: `{"features":{}}`, exp: `{"features":{}}`, upgrade: true},
	} {
		got, err := migrateSubscriptions([]byte(tc.in), tc.upgrade)
		require.NoErrorf(t, err, "migrateSubscriptions must not error %s", tc.name)
		assert.JSONEqf(t, tc.exp, string(got), "migrateSubscriptions should give the expected subscriptions %s", tc.name)
	}

	_, err := migrateSubscriptions([]byte(`{"features":{"subscriptions":[}}`), true)
	assert.ErrorIs(t, err, jsonparser.MalformedArrayError, "migrateSubscriptions should reject unterminated subscriptions")
	var typeErr *json.UnmarshalTypeError
	_, err = migrateSubscriptions([]byte(`{"features":{"subscriptions":{}}}`), true)
	assert.ErrorAs(t, err, &typeErr, "migrateSubscriptions should reject subscriptions that are not an array")
	_, err = migrateSubscriptions([]byte(`{"features":{"subscriptions":[1]}}`), false)
	assert.ErrorAs(t, err, &typeErr, "migrateSubscriptions should reject a subscription that is not an object")
}
