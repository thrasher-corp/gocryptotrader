package v19

import (
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestKeepGeneratable(t *testing.T) {
	t.Parallel()
	exchange := []byte(`{"features":{"subscriptions":[{"enabled":true,"channel":"ticker"},{"enabled":true,"channel":"candles","asset":"spot","interval":"1m"},{"enabled":true,"channel":"allTrades","asset":"spot"}]}}`)
	t.Run("updates the changed entries in place", func(t *testing.T) {
		t.Parallel()
		entries, err := subscriptions(exchange)
		require.NoError(t, err, "subscriptions must not error")
		got, saved, err := keepGeneratable(exchange, entries)
		require.NoError(t, err, "keepGeneratable must not error")
		exp := `{"features":{"subscriptions":[{"enabled":false,"channel":"ticker"},{"enabled":true,"channel":"candles","asset":"spot","interval":"5m"},{"enabled":true,"channel":"allTrades","asset":"spot"}]}}`
		assert.JSONEq(t, exp, string(got), "keepGeneratable should disable the ticker without an asset and give the candles five minutes")
		updated, err := subscriptions(got)
		require.NoError(t, err, "subscriptions must not error for the updated configuration")
		assert.Equal(t, updated, entries, "keepGeneratable should update the changed entries in place")
		decoded, err := decodeEntries(updated)
		require.NoError(t, err, "decodeEntries must not error for the updated entries")
		assert.Equal(t, decoded, saved, "keepGeneratable should return the subscriptions as updated")
	})

	t.Run("an entry it cannot decode", func(t *testing.T) {
		t.Parallel()
		malformed, err := subscriptions([]byte(`{"features":{"subscriptions":[{"enabled":"true","channel":"ticker"}]}}`))
		require.NoError(t, err, "subscriptions must not error for a malformed entry")
		got, saved, err := keepGeneratable(exchange, malformed)
		var typeErr *json.UnmarshalTypeError
		assert.ErrorAs(t, err, &typeErr, "keepGeneratable should wrap the decoding error")
		assert.ErrorContains(t, err, "error decoding Coinbase subscriptions", "keepGeneratable should report a subscription it cannot decode")
		assert.Nil(t, got, "keepGeneratable should return no configuration when it cannot decode a subscription")
		assert.Nil(t, saved, "keepGeneratable should return no subscriptions when it cannot decode one")
	})

	t.Run("an entry it cannot update", func(t *testing.T) {
		t.Parallel()
		// Only the second of three entries changes, so the failed update names index 1
		unwritable, err := subscriptions([]byte(`{"features":{"subscriptions":[{"enabled":true,"channel":"allTrades","asset":"spot"},{"enabled":true,"channel":"ticker"},{"enabled":true,"channel":"orderbook","asset":"spot"}]}}`))
		require.NoError(t, err, "subscriptions must not error")
		got, saved, err := keepGeneratable([]byte(`"Coinbase"`), unwritable)
		assert.ErrorIs(t, err, jsonparser.KeyPathNotFoundError, "keepGeneratable should report a subscription it cannot update")
		assert.ErrorContains(t, err, "error updating Coinbase subscription 1:", "keepGeneratable should name the subscription it cannot update")
		assert.Nil(t, got, "keepGeneratable should return no configuration when it cannot update a subscription")
		assert.Nil(t, saved, "keepGeneratable should return no subscriptions when it cannot update one")
	})
}

func TestParseInterval(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		saved    []byte
		interval time.Duration
		ok       bool
	}{
		"absent":         {nil, 0, true},
		"null":           {[]byte(`null`), 0, false},
		"nanoseconds":    {[]byte(`60000000000`), time.Minute, true},
		"raw":            {[]byte(`"raw"`), -1, true},
		"raw as -1":      {[]byte(`-1`), -1, true},
		"upper case raw": {[]byte(`"RAW"`), 0, false},
		"duration":       {[]byte(`"5m0s"`), 5 * time.Minute, true},
		"minutes":        {[]byte(`"15min"`), 15 * time.Minute, true},
		"month":          {[]byte(`"1M"`), 30 * 24 * time.Hour, true},
		"unknown unit":   {[]byte(`"1y"`), 0, false},
		"seconds":        {[]byte(`"30sec"`), 30 * time.Second, true},
		"days":           {[]byte(`"2d"`), 48 * time.Hour, true},
		"weeks":          {[]byte(`"1wk"`), 7 * 24 * time.Hour, true},
		"overflow":       {[]byte(`"9223372036854775807h"`), 0, false},
		"count overflow": {[]byte(`"99999999999999999999min"`), 0, false},
		"boolean":        {[]byte(`true`), 0, false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			interval, ok := parseInterval(tc.saved)
			assert.Equalf(t, tc.ok, ok, "parseInterval should report whether the loader reads the %s interval", name)
			assert.Equalf(t, tc.interval, interval, "parseInterval should read the %s interval as the loader does", name)
		})
	}
}
