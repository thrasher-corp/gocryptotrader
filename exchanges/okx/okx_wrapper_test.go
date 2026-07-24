package okx

import (
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

func TestCachedInstrumentIDCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		assetType     asset.Item
		instrumentID  string
		expected      int64
		expectedError error
	}{
		{name: "empty instrument ID", assetType: asset.Spot, expectedError: errMissingInstrumentID},
		{name: "unsupported asset", assetType: asset.Empty, instrumentID: "BTC-USDT", expectedError: asset.ErrNotSupported},
		{name: "cache miss", assetType: asset.Spot, instrumentID: "ETH-USDT", expectedError: errMissingInstrumentIDCode},
		{name: "cache hit", assetType: asset.Spot, instrumentID: "BTC-USDT", expected: 123},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ex := &Exchange{
				instrumentsInfoMap: map[string][]Instrument{
					instTypeSpot: {
						{
							InstrumentID:     currency.NewPairWithDelimiter("BTC", "USDT", "-"),
							InstrumentIDCode: types.Number(123),
						},
					},
				},
			}
			instrumentIDCode, err := ex.cachedInstrumentIDCode(tc.assetType, tc.instrumentID)
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError, "cachedInstrumentIDCode must return expected error")
				return
			}
			require.NoError(t, err, "cachedInstrumentIDCode must not error for a cached instrument")
			assert.Equal(t, tc.expected, instrumentIDCode, "cached instrument ID code should be returned")
		})
	}
}
