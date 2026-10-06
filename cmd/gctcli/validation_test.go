package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidPair(t *testing.T) {
	testCases := []struct {
		name  string
		pair  string
		valid bool
	}{
		{
			name:  "dash delimiter",
			pair:  "BTC-USD",
			valid: true,
		},
		{
			name:  "underscore delimiter",
			pair:  "BTC_USD",
			valid: true,
		},
		{
			name:  "slash delimiter",
			pair:  "BTC/USD",
			valid: true,
		},
		{
			name:  "no delimiter",
			pair:  "BTCUSD",
			valid: false,
		},
		{
			name:  "long no delimiter",
			pair:  "DOGEUSDT",
			valid: false,
		},
		{
			name:  "invalid pair",
			pair:  "BT",
			valid: false,
		},
		{
			name:  "empty pair",
			pair:  "",
			valid: false,
		},
	}

	originalDelimiter := pairDelimiter
	t.Cleanup(func() { pairDelimiter = originalDelimiter })
	for _, delimiter := range []string{"|", "+", "x", "::", "—"} {
		t.Run("configured "+delimiter, func(t *testing.T) {
			pairDelimiter = delimiter
			require.True(t, validPair("BTC"+delimiter+"USDT"), "configured delimiter must be accepted")
			require.False(t, validPair("BTC"+delimiter), "missing quote must be rejected")
			require.False(t, validPair(delimiter+"USDT"), "missing base must be rejected")
		})
	}
	pairDelimiter = "-"
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.valid, validPair(tc.pair))
		})
	}
}
