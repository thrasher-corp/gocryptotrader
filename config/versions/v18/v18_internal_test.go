package v18

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBitfinex(t *testing.T) {
	t.Parallel()
	for input, exp := range map[string]bool{
		`{"name":"Bitfinex"}`:                 true,
		`{"name":"bitfinex"}`:                 true,
		`{"name":"BITFINEX"}`:                 true,
		`{"Name":"Bitfinex"}`:                 true,
		`{"name":"Kraken","name":"Bitfinex"}`: true,
		`{"name":"bitfinex","name":"Kraken"}`: false,
		`{"name":"bitfinex","Name":"Kraken"}`: false,
		`{"name":"Bitfinexx"}`:                false,
		`{"name":"xBitfinex"}`:                false,
		`{"name":"Bitfinzz"}`:                 false,
		`{"name":" Bitfinex "}`:               false,
		`{"name":"Kraken"}`:                   false,
		`{}`:                                  false,
	} {
		got, err := isBitfinex([]byte(input))
		require.NoErrorf(t, err, "isBitfinex must not error for %s", input)
		assert.Equalf(t, exp, got, "isBitfinex should read the name of %s as the config loader does", input)
	}
	_, err := isBitfinex([]byte(`{"name":`))
	assert.ErrorContains(t, err, "error decoding exchange name", "isBitfinex should report a configuration it cannot decode")
}

func TestReplaceSubscriptions(t *testing.T) {
	t.Parallel()
	// UpgradeExchange and DowngradeExchange refuse a configuration the loader cannot decode before reaching this
	input := []byte(`{"name":"Bitfinex","features":{"subscriptions":}}`)
	got, err := replaceSubscriptions(input, previousDefaults, currentDefaults)
	assert.ErrorContains(t, err, "error getting Bitfinex subscriptions", "replaceSubscriptions should report subscriptions it cannot get")
	assert.Equal(t, input, got, "replaceSubscriptions should leave a configuration it cannot read unchanged")
}
