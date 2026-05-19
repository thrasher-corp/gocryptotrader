package okx

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestMessageID(t *testing.T) {
	t.Parallel()
	id := new(Exchange).MessageID()
	require.Len(t, id, 32, "Must return the correct length of message id")
	u, err := uuid.FromString(id)
	require.NoError(t, err, "MessageID must return a valid UUID")
	require.Equal(t, uuid.V7, u.Version(), "MessageID must return a V7 uuid")
	require.Len(t, u.String(), 36, "UUID v7 string representation must be 36 characters long")
}

// 7696807	       153.1 ns/op	      48 B/op	       2 allocs/op
func BenchmarkMessageID(b *testing.B) {
	e := new(Exchange)
	for b.Loop() {
		_ = e.MessageID()
	}
}

func TestLookupInstrumentIDCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		instruments  []Instrument
		instrumentID string
		expected     int64
	}{
		{
			name: "matching instrument code",
			instruments: []Instrument{
				{
					InstrumentID:     currency.NewPairWithDelimiter("BTC", "USDT", "-"),
					InstrumentIDCode: types.Number(123),
				},
			},
			instrumentID: "BTC-USDT",
			expected:     123,
		},
		{
			name: "single non-matching entry returns zero",
			instruments: []Instrument{
				{
					InstrumentID:     currency.NewPairWithDelimiter("ETH", "USDT", "-"),
					InstrumentIDCode: types.Number(987),
				},
			},
			instrumentID: "BTC-USDT",
			expected:     0,
		},
		{
			name: "no match in multiple entries",
			instruments: []Instrument{
				{
					InstrumentID:     currency.NewPairWithDelimiter("ETH", "USDT", "-"),
					InstrumentIDCode: types.Number(456),
				},
				{
					InstrumentID:     currency.NewPairWithDelimiter("SOL", "USDT", "-"),
					InstrumentIDCode: types.Number(789),
				},
			},
			instrumentID: "BTC-USDT",
			expected:     0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, lookupInstrumentIDCode(testCase.instruments, testCase.instrumentID), "lookup result should match expected code")
		})
	}
}

func TestOptionInstrumentSelectors(t *testing.T) {
	t.Parallel()

	underlying, family := optionInstrumentSelectors("BTC-USD-240329-70000-C")
	require.Equal(t, "BTC-USD", underlying, "underlying selector must parse option instrument ID")
	require.Equal(t, "BTC-USD", family, "family selector must parse option instrument ID")

	underlying, family = optionInstrumentSelectors("ETH_USD_240329_3500_P")
	require.Equal(t, "ETH_USD", underlying, "underlying selector must parse underscore instrument ID")
	require.Equal(t, "ETH_USD", family, "family selector must parse underscore instrument ID")

	underlying, family = optionInstrumentSelectors("INVALID")
	require.Equal(t, "INVALID", underlying, "fallback underlying must return raw instrument ID")
	require.Equal(t, "INVALID", family, "fallback family must return raw instrument ID")
}

func TestResolveInstrumentIDCode(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	_, err := ex.resolveInstrumentIDCode(context.Background(), asset.Spot, "")
	require.ErrorIs(t, err, errMissingInstrumentID, "resolveInstrumentIDCode must return missing instrument ID error")

	_, err = ex.resolveInstrumentIDCode(context.Background(), asset.Empty, "BTC-USDT")
	require.ErrorIs(t, err, errInvalidInstrumentType, "resolveInstrumentIDCode must return invalid instrument type error")
}
