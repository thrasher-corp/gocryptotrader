package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidPair(t *testing.T) {
	t.Parallel()

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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.valid, validPair(tc.pair))
		})
	}
}
