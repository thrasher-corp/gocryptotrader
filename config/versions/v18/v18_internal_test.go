package v18

import (
	"encoding/json" //nolint:depguard // Matches the encoding/json the migration uses
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
		raw json.RawMessage
		exp time.Duration
		ok  bool
	}{
		{raw: nil, ok: true},
		{raw: json.RawMessage(`null`)},
		{raw: json.RawMessage(`900000000000`), exp: 15 * time.Minute, ok: true},
		{raw: json.RawMessage(`"15m"`), exp: 15 * time.Minute, ok: true},
		{raw: json.RawMessage(`"0.25h"`), exp: 15 * time.Minute, ok: true},
		{raw: json.RawMessage(`"100ms"`), exp: 100 * time.Millisecond, ok: true},
		{raw: json.RawMessage(`"0s"`), ok: true},
		{raw: json.RawMessage(`"900000000000"`)},
		{raw: json.RawMessage(`"1d"`)},
		{raw: json.RawMessage(`true`)},
	} {
		got, ok := parseInterval(tc.raw)
		assert.Equalf(t, tc.ok, ok, "parseInterval should report whether %s parses", tc.raw)
		assert.Equalf(t, tc.exp, got, "parseInterval should return the duration of %s", tc.raw)
	}
}

func TestShortInterval(t *testing.T) {
	t.Parallel()
	for d, exp := range map[time.Duration]string{
		15 * time.Minute:        "15m",
		100 * time.Millisecond:  "100ms",
		4 * time.Hour:           "4h",
		90 * time.Second:        "1m30s",
		time.Hour + time.Minute: "1h1m",
	} {
		assert.Equalf(t, exp, shortInterval(d), "%s should be saved as the config saves it", d)
	}
}

var savedPrevious = []string{
	`{"enabled":true,"channel":"orderbook","asset":"spot","levels":20}`,
	`{"enabled":true,"channel":"candles","asset":"spot","interval":"15m"}`,
	`{"enabled":true,"channel":"ticker","asset":"spot","interval":"100ms"}`,
	`{"enabled":true,"channel":"public.miniTicker.v3.api.pb","asset":"spot"}`,
	`{"enabled":true,"channel":"allTrades","asset":"spot","interval":"100ms"}`,
	`{"enabled":true,"channel":"myTrades","asset":"spot","authenticated":true}`,
	`{"enabled":true,"channel":"myOrders","asset":"spot","authenticated":true}`,
	`{"enabled":true,"channel":"myAccount","asset":"spot","authenticated":true}`,
}

func rawEntries(entries ...string) []json.RawMessage {
	raw := make([]json.RawMessage, len(entries))
	for i := range entries {
		raw[i] = json.RawMessage(entries[i])
	}
	return raw
}

func TestNamesDefault(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		s   savedSubscription
		exp bool
	}{
		{savedSubscription{Channel: "ticker", Asset: "spot"}, true},
		{savedSubscription{Channel: "ticker", Asset: "FUTURES"}, true},
		{savedSubscription{Channel: "position", Asset: "futures"}, true},
		{savedSubscription{Channel: "public.miniTicker.v3.api.pb", Asset: "all"}, true},
		{savedSubscription{Channel: "public.aggre.bookTicker.v3.api.pb", Asset: "spot"}, true},
		{savedSubscription{Channel: "public.limit.depth.v3.api.pb", Asset: "Spot"}, true},
		{savedSubscription{Channel: "public.aggre.deals.v3.api.pb", Asset: "spot"}, true},
		{savedSubscription{Channel: "public.kline.v3.api.pb", Asset: "SPOT"}, true},
		{savedSubscription{Channel: "private.deals.v3.api.pb", Asset: "spot"}, true},
		{savedSubscription{Channel: "private.orders.v3.api.pb", Asset: "All"}, true},
		{savedSubscription{Channel: "private.account.v3.api.pb", Asset: "all"}, true},
		{savedSubscription{Channel: "candles", Asset: "ALL"}, true},
		{savedSubscription{Channel: "public.aggre.deals.v3.api.pb"}, false},
		{savedSubscription{Channel: "myOrders"}, false},
		{savedSubscription{Channel: " ticker ", Asset: "spot"}, false},
		{savedSubscription{Channel: "position", Asset: "spot"}, false},
		{savedSubscription{Channel: "public.aggre.deals.v3.api.pb", Asset: "futures"}, false},
		{savedSubscription{Channel: "public.bookTicker.batch.v3.api.pb", Asset: "spot"}, false},
		{savedSubscription{Channel: "Ticker", Asset: "spot"}, false},
		{savedSubscription{Channel: "ticker", Asset: "margin"}, false},
		{savedSubscription{Asset: "spot"}, false},
	} {
		assert.Equalf(t, tc.exp, namesDefault(&tc.s), "namesDefault should report whether channel %q under asset %q names a default", tc.s.Channel, tc.s.Asset)
	}
}

func TestSavedAssetUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw string
		exp savedAsset
	}{
		{`"futures"`, "futures"},
		{`null`, ""},
	} {
		a := savedAsset("spot")
		require.NoErrorf(t, json.Unmarshal([]byte(tc.raw), &a), "Unmarshal must decode %s", tc.raw)
		assert.Equalf(t, tc.exp, a, "%s should decode as the loader decodes it", tc.raw)
	}
	var a savedAsset
	var typeErr *json.UnmarshalTypeError
	assert.ErrorAs(t, json.Unmarshal([]byte(`1`), &a), &typeErr, "a number should be refused as the loader refuses it")
}

func TestDefaultSubscriptionMatches(t *testing.T) {
	t.Parallel()
	orderbook := &defaultSubscription{channel: "orderbook", asset: spotAsset, levels: 20}
	candles := &defaultSubscription{channel: "candles", asset: spotAsset, interval: 15 * time.Minute}
	orders := &defaultSubscription{channel: "myOrders", asset: spotAsset, authenticated: true}
	for _, tc := range []struct {
		d   *defaultSubscription
		s   savedSubscription
		exp bool
	}{
		{orderbook, savedSubscription{Enabled: true, Levels: 20}, true},
		{orderbook, savedSubscription{Levels: 20}, false},
		{orderbook, savedSubscription{Enabled: true, Levels: 5}, false},
		{orderbook, savedSubscription{Enabled: true, Levels: 20, Authenticated: true}, false},
		{orderbook, savedSubscription{Enabled: true, Levels: 20, Interval: json.RawMessage(`"1m"`)}, false},
		{candles, savedSubscription{Enabled: true, Interval: json.RawMessage(`"15m"`)}, true},
		{candles, savedSubscription{Enabled: true, Interval: json.RawMessage(`null`)}, false},
		{orders, savedSubscription{Enabled: true, Authenticated: true}, true},
		{orders, savedSubscription{Enabled: true}, false},
	} {
		assert.Equalf(t, tc.exp, tc.d.matches(&tc.s), "matches should report whether %+v resolves to the %s default", tc.s, tc.d.channel)
	}
}

func TestMatchDefaults(t *testing.T) {
	t.Parallel()
	const disabled = `{"enabled":false,"channel":"public.bookTicker.batch.v3.api.pb","asset":"spot"}`
	for name, tc := range map[string]struct {
		entries []string
		matched bool
		carried []json.RawMessage
	}{
		"defaults":                    {entries: savedPrevious, matched: true},
		"defaults and disabled entry": {entries: append(append([]string(nil), savedPrevious...), disabled), matched: true, carried: rawEntries(disabled)},
		"defaults and enabled entry":  {entries: append(append([]string(nil), savedPrevious...), `{"enabled":true,"channel":"public.bookTicker.batch.v3.api.pb","asset":"spot"}`)},
		"disabled default":            {entries: append(append([]string(nil), savedPrevious...), `{"enabled":false,"channel":"ticker","asset":"spot"}`)},
	} {
		matched, carried, err := matchDefaults(rawEntries(tc.entries...), previousDefaults)
		require.NoErrorf(t, err, "matchDefaults must not error for %s", name)
		assert.Equalf(t, tc.matched, matched, "matchDefaults should report whether %s match the defaults", name)
		assert.Equalf(t, tc.carried, carried, "matchDefaults should carry the disabled entries naming no default for %s", name)
	}
	for _, entry := range []string{`"orderbook"`, `{"enabled":true,"channel":"orderbook","asset":"spot","levels":20.5}`} {
		_, _, err := matchDefaults(rawEntries(entry), previousDefaults)
		var typeErr *json.UnmarshalTypeError
		assert.ErrorAsf(t, err, &typeErr, "matchDefaults should refuse the malformed entry %s", entry)
	}
}

func TestReplaceSubscriptions(t *testing.T) {
	t.Parallel()
	const disabled = `{"enabled":false,"channel":"public.bookTicker.batch.v3.api.pb","asset":"spot"}`
	saved := `{"features":{"subscriptions":[` + strings.Join(savedPrevious, ",") + `,` + disabled + `]}}`
	got, err := replaceSubscriptions([]byte(saved), previousDefaults, previousDefaults)
	require.NoError(t, err, "replaceSubscriptions must not error")
	assert.JSONEq(t, saved, string(got), "replacing the defaults with themselves should keep the list and its disabled entry")

	got, err = replaceSubscriptions([]byte(saved), currentDefaults, previousDefaults)
	require.NoError(t, err, "replaceSubscriptions must not error when nothing matches")
	assert.Equal(t, saved, string(got), "subscriptions which match no defaults should be left unchanged")

	for _, malformed := range []string{`{"features":{"subscriptions":{}}}`, `{"features":{"subscriptions":["orderbook"]}}`} {
		got, err = replaceSubscriptions([]byte(malformed), previousDefaults, currentDefaults)
		var typeErr *json.UnmarshalTypeError
		assert.ErrorAsf(t, err, &typeErr, "replaceSubscriptions should refuse %s", malformed)
		assert.Equalf(t, malformed, string(got), "replaceSubscriptions should leave %s unchanged", malformed)
	}
	missing := []byte(`{"features":{"subscriptions":}}`)
	got, err = replaceSubscriptions(missing, previousDefaults, currentDefaults)
	assert.ErrorIs(t, err, jsonparser.UnknownValueTypeError, "replaceSubscriptions should refuse subscriptions without a value")
	assert.Equal(t, missing, got, "replaceSubscriptions should leave subscriptions without a value unchanged")
	unsubscribed := []byte(`{"name":"MEXC"}`)
	got, err = replaceSubscriptions(unsubscribed, previousDefaults, currentDefaults)
	require.NoError(t, err, "replaceSubscriptions must not error without subscriptions")
	assert.Equal(t, unsubscribed, got, "a config without subscriptions should be left unchanged")
}
