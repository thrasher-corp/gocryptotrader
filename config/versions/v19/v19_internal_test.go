package v19

import (
	"encoding/json" //nolint:depguard // Matches the encoding/json the migration uses
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		raw  json.RawMessage
		exp  time.Duration
		ok   bool
	}{
		{name: "absent", raw: nil, ok: true},
		{name: "null", raw: json.RawMessage(`null`)},
		{name: "nanoseconds", raw: json.RawMessage(`100000000`), exp: 100 * time.Millisecond, ok: true},
		{name: "milliseconds duration", raw: json.RawMessage(`"100ms"`), exp: 100 * time.Millisecond, ok: true},
		{name: "fractional seconds duration", raw: json.RawMessage(`"0.1s"`), exp: 100 * time.Millisecond, ok: true},
		{name: "zero duration", raw: json.RawMessage(`"0s"`), ok: true},
		{name: "quoted nanoseconds", raw: json.RawMessage(`"100000000"`)},
		{name: "unparsable duration", raw: json.RawMessage(`"wibble"`)},
		{name: "boolean", raw: json.RawMessage(`true`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseInterval(tc.raw)
			assert.Equalf(t, tc.ok, ok, "parseInterval should report whether %s parses", tc.raw)
			assert.Equalf(t, tc.exp, got, "parseInterval should return the duration of %s", tc.raw)
		})
	}
}

func TestDisableUnsupportedChannels(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input, exp string
	}{
		{name: "no subscriptions", input: `{"name":"Bitstamp"}`, exp: `{"name":"Bitstamp"}`},
		{name: "subscriptions object", input: `{"features":{"subscriptions":{}}}`, exp: `{"features":{"subscriptions":{}}}`},
		{
			name:  "previous channels",
			input: `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"all"},{"enabled":true,"channel":"allTrades"},{"enabled":true,"channel":"myOrders","asset":"spot"},{"enabled":true,"channel":"myTrades"}]}}`,
			exp:   `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"all"},{"enabled":true,"channel":"allTrades"},{"enabled":true,"channel":"myOrders","asset":"spot"},{"enabled":true,"channel":"myTrades"}]}}`,
		},
		{
			name:  "perpetual contract assets",
			input: `{"features":{"subscriptions":[{"enabled":true,"channel":"funding_rate","asset":"perpetualcontract"},{"enabled":true,"channel":"funding_rate","asset":"PerpetualContract"}]}}`,
			exp:   `{"features":{"subscriptions":[{"enabled":true,"channel":"funding_rate","asset":"perpetualcontract"},{"enabled":true,"channel":"funding_rate","asset":"PerpetualContract"}]}}`,
		},
		{
			name:  "unsupported channels",
			input: `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook"},{"enabled":true,"channel":"funding_rate","asset":"all"},{"enabled":true,"channel":"announcements"}]}}`,
			exp:   `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook"},{"enabled":false,"channel":"funding_rate","asset":"all"},{"enabled":false,"channel":"announcements"}]}}`,
		},
		{
			name:  "unsupported channels not enabled",
			input: `{"features":{"subscriptions":[{"enabled":false,"channel":"announcements"},{"channel":"announcements"},{"enabled":null,"channel":"announcements"}]}}`,
			exp:   `{"features":{"subscriptions":[{"enabled":false,"channel":"announcements"},{"channel":"announcements"},{"enabled":null,"channel":"announcements"}]}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := disableUnsupportedChannels([]byte(tc.input))
			require.NoError(t, err, "disableUnsupportedChannels must not error")
			assert.JSONEq(t, tc.exp, string(got), "disableUnsupportedChannels should disable only the enabled channels the previous version cannot subscribe to")
		})
	}
}

func TestClientIDRequirementKeys(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input, errContains string
		exp                      []string
	}{
		{name: "no validator", input: `{"name":"Bitstamp","api":{}}`},
		{name: "null validator", input: `{"api":{"credentialsValidator":null}}`},
		{name: "without the requirement", input: `{"api":{"credentialsValidator":{"requiresKey":true}}}`},
		{name: "requirement", input: `{"api":{"credentialsValidator":{"requiresKey":true,"requiresClientID":false}}}`, exp: []string{"requiresClientID"}},
		{name: "any case", input: `{"api":{"credentialsValidator":{"RequiresClientID":true,"requiresclientid":false}}}`, exp: []string{"RequiresClientID", "requiresclientid"}},
		{name: "repeated key", input: `{"api":{"credentialsValidator":{"requiresClientID":false,"requiresClientID":true}}}`, exp: []string{"requiresClientID", "requiresClientID"}},
		{name: "malformed validator", input: `{"api":{"credentialsValidator":}}`, errContains: "error getting Bitstamp credentials validator"},
		{name: "malformed requirement", input: `{"api":{"credentialsValidator":{"requiresClientID":}}}`, errContains: "error decoding Bitstamp credentials validator"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			keys, err := clientIDRequirementKeys([]byte(tc.input))
			if tc.errContains != "" {
				assert.ErrorContains(t, err, tc.errContains, "clientIDRequirementKeys should report the malformed validator")
				return
			}
			require.NoError(t, err, "clientIDRequirementKeys must not error")
			assert.Equal(t, tc.exp, keys, "clientIDRequirementKeys should return each key the config loader reads as the requirement")
		})
	}
}

func TestReplaceSubscriptions(t *testing.T) {
	t.Parallel()
	from := []defaultSubscription{{channel: "orderbook", asset: "spot", interval: 100 * time.Millisecond}, {channel: "myOrders", asset: "spot", authenticated: true}}
	to := []defaultSubscription{{channel: "allTrades", asset: "all", interval: time.Second, authenticated: true}, {channel: "funding_rate", asset: "perpetualcontract"}}
	for _, tc := range []struct {
		name, input, exp, errContains string
	}{
		{name: "no subscriptions", input: `{"name":"Bitstamp"}`, exp: `{"name":"Bitstamp"}`},
		{name: "null subscriptions", input: `{"features":{"subscriptions":null}}`, exp: `{"features":{"subscriptions":null}}`},
		{
			name:  "matching subscriptions",
			input: `{"features":{"subscriptions":[{"enabled":true,"channel":"myOrders","asset":"SPOT","authenticated":true},{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"}]}}`,
			exp:   `{"features":{"subscriptions":[{"enabled":true,"channel":"allTrades","asset":"all","interval":"1s","authenticated":true},{"enabled":true,"channel":"funding_rate","asset":"perpetualcontract"}]}}`,
		},
		{
			name:  "customised subscriptions",
			input: `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"10ms"},{"enabled":true,"channel":"myOrders","asset":"spot","authenticated":true}]}}`,
			exp:   `{"features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"10ms"},{"enabled":true,"channel":"myOrders","asset":"spot","authenticated":true}]}}`,
		},
		{name: "malformed subscriptions", input: `{"features":{"subscriptions":}}`, errContains: "error getting Bitstamp subscriptions"},
		{name: "subscriptions object", input: `{"features":{"subscriptions":{}}}`, errContains: "error decoding Bitstamp subscriptions"},
		{name: "malformed entry", input: `{"features":{"subscriptions":["orderbook"]}}`, errContains: "error decoding Bitstamp subscriptions"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := replaceSubscriptions([]byte(tc.input), from, to)
			if tc.errContains != "" {
				require.ErrorContains(t, err, tc.errContains, "replaceSubscriptions must report the failing operation")
				assert.Equal(t, tc.input, string(got), "replaceSubscriptions should leave a malformed config unchanged")
				return
			}
			require.NoError(t, err, "replaceSubscriptions must not error")
			assert.JSONEq(t, tc.exp, string(got), "replaceSubscriptions should replace only subscriptions matching the defaults")
		})
	}
}

func TestMatchDefaults(t *testing.T) {
	t.Parallel()
	defaults := []defaultSubscription{{channel: "orderbook", asset: "spot", interval: 100 * time.Millisecond}, {channel: "myOrders", asset: "spot", authenticated: true}}
	const (
		orderbook = `{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"}`
		orders    = `{"enabled":true,"channel":"myOrders","asset":"spot","authenticated":true}`
	)
	for _, tc := range []struct {
		name    string
		entries []string
		exp     bool
		err     bool
	}{
		{name: "in order", entries: []string{orderbook, orders}, exp: true},
		{name: "any order", entries: []string{orders, orderbook}, exp: true},
		{name: "fewer entries", entries: []string{orderbook}},
		{name: "more entries", entries: []string{orderbook, orders, orders}},
		{name: "repeated channel", entries: []string{orderbook, orderbook}},
		{name: "other channel", entries: []string{orderbook, `{"enabled":true,"channel":"myTrades","asset":"spot","authenticated":true}`}},
		{name: "different setting", entries: []string{orderbook, `{"enabled":true,"channel":"myOrders","asset":"spot"}`}},
		{name: "other field", entries: []string{orderbook, `{"enabled":true,"channel":"myOrders","asset":"spot","authenticated":true,"pairs":"BTC/USD"}`}},
		{name: "entry not an object", entries: []string{orderbook, `"myOrders"`}, err: true},
		{name: "mistyped field", entries: []string{orderbook, `{"enabled":"true","channel":"myOrders","asset":"spot","authenticated":true}`}, err: true},
		{name: "mistyped field after another field", entries: []string{`{"enabled":true,"channel":"orderbook","levels":10}`, `{"enabled":"true","channel":"myOrders"}`}, err: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			entries := make([]json.RawMessage, len(tc.entries))
			for i := range tc.entries {
				entries[i] = json.RawMessage(tc.entries[i])
			}
			matched, err := matchDefaults(entries, defaults)
			if tc.err {
				assert.Error(t, err, "matchDefaults should report a malformed entry")
				return
			}
			require.NoError(t, err, "matchDefaults must not error")
			assert.Equal(t, tc.exp, matched, "matchDefaults should report whether each entry resolves to a different default")
		})
	}
}

func TestDefaultSubscriptionMatches(t *testing.T) {
	t.Parallel()
	public := &defaultSubscription{channel: "orderbook", asset: "spot", interval: 100 * time.Millisecond}
	private := &defaultSubscription{channel: "myOrders", asset: "spot", authenticated: true}
	for _, tc := range []struct {
		name  string
		d     *defaultSubscription
		saved savedSubscription
		exp   bool
	}{
		{name: "public", d: public, saved: savedSubscription{Enabled: new(true), Asset: "spot", Interval: json.RawMessage(`"100ms"`)}, exp: true},
		{name: "asset in another case", d: public, saved: savedSubscription{Enabled: new(true), Asset: "SPOT", Interval: json.RawMessage(`"100ms"`)}, exp: true},
		{name: "interval in nanoseconds", d: public, saved: savedSubscription{Enabled: new(true), Asset: "spot", Interval: json.RawMessage(`100000000`)}, exp: true},
		{name: "explicitly unauthenticated", d: public, saved: savedSubscription{Enabled: new(true), Asset: "spot", Interval: json.RawMessage(`"100ms"`), Authenticated: new(false)}, exp: true},
		{name: "enabled absent", d: public, saved: savedSubscription{Asset: "spot", Interval: json.RawMessage(`"100ms"`)}},
		{name: "disabled", d: public, saved: savedSubscription{Enabled: new(false), Asset: "spot", Interval: json.RawMessage(`"100ms"`)}},
		{name: "other asset", d: public, saved: savedSubscription{Enabled: new(true), Asset: "margin", Interval: json.RawMessage(`"100ms"`)}},
		{name: "other interval", d: public, saved: savedSubscription{Enabled: new(true), Asset: "spot", Interval: json.RawMessage(`"10ms"`)}},
		{name: "interval absent", d: public, saved: savedSubscription{Enabled: new(true), Asset: "spot"}},
		{name: "null interval", d: public, saved: savedSubscription{Enabled: new(true), Asset: "spot", Interval: json.RawMessage(`null`)}},
		{name: "authenticated public", d: public, saved: savedSubscription{Enabled: new(true), Asset: "spot", Interval: json.RawMessage(`"100ms"`), Authenticated: new(true)}},
		{name: "private", d: private, saved: savedSubscription{Enabled: new(true), Asset: "spot", Authenticated: new(true)}, exp: true},
		{name: "private without authentication", d: private, saved: savedSubscription{Enabled: new(true), Asset: "spot"}},
		{name: "private explicitly unauthenticated", d: private, saved: savedSubscription{Enabled: new(true), Asset: "spot", Authenticated: new(false)}},
		{name: "private with an interval", d: private, saved: savedSubscription{Enabled: new(true), Asset: "spot", Interval: json.RawMessage(`"1s"`), Authenticated: new(true)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, tc.d.matches(&tc.saved), "matches should report whether the saved subscription resolves to the default")
		})
	}
}
