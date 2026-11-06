package kraken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetFuturesFills(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *FuturesFillsRequest
		exp  *FuturesFillsResponse
	}{
		{
			name: "latest fills",
			exp: &FuturesFillsResponse{
				Fills: []FuturesFill{
					{
						FillID:        "3d57ed09-fbd6-44f1-8e8b-b10e551c5e73",
						OrderID:       futuresTradingBatchLimitID,
						ClientOrderID: "gct-batch-1",
						Symbol:        "PF_XBTUSD",
						Side:          "buy",
						Size:          0.0002,
						Price:         81449,
						FillTime:      futuresTradingTestTime(8, 26, 31, 77),
						FillType:      "maker",
						RealisedPNL:   0.0098,
						SequenceID:    42,
					},
					{
						FillID:      "56b86ada-73b0-454d-a95a-e29e3e85b349",
						OrderID:     futuresTradingRestingOrderID,
						Symbol:      "PF_XBTUSD",
						Side:        "sell",
						Size:        0.0001,
						Price:       82750,
						FillTime:    futuresTradingTestTime(8, 1, 40, 270),
						FillType:    "takerAfterEdit",
						RealisedPNL: -0.0126,
						// Beyond the integers a float64 holds exactly
						SequenceID: 9007199254740993,
					},
				},
				ServerTime: futuresTradingTestTime(8, 36, 0, 311),
			},
		},
		{
			name: "fills before a fill time",
			// Sent in milliseconds, as 08:01:40.270Z
			req: &FuturesFillsRequest{LastFillTime: time.Date(2026, 10, 9, 8, 1, 40, 270000000, time.UTC)},
			exp: &FuturesFillsResponse{
				Fills: []FuturesFill{
					{
						FillID:        "6a1d4e2f-8b3c-4d5e-9f6a-7b8c9d0e1f2a",
						OrderID:       "7c8d9e0f-1a2b-4c3d-8e4f-5a6b7c8d9e0f",
						ClientOrderID: "gct-inverse-1",
						Symbol:        "PI_XBTUSD",
						Side:          "sell",
						Size:          5000,
						Price:         81802.5,
						FillTime:      time.Date(2026, 10, 8, 22, 14, 5, 390000000, time.UTC),
						FillType:      "taker",
					},
				},
				ServerTime: futuresTradingTestTime(8, 36, 1, 462),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesFills(t.Context(), tc.req)
			require.NoError(t, err, "GetFuturesFills must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesFills should decode every field")
				return
			}
			assert.NotZero(t, result.ServerTime, "GetFuturesFills should return the server time")
		})
	}
}
