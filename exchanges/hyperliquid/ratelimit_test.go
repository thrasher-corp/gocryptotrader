package hyperliquid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestGetRateLimits(t *testing.T) {
	t.Parallel()
	limits := GetRateLimits()
	require.Contains(t, limits, infoStandardEPL, "Standard info endpoint limit must be configured")
	require.Contains(t, limits, infoLightEPL, "Light info endpoint limit must be configured")
	require.Contains(t, limits, infoRecentTradesEPL, "Recent trades endpoint limit must be configured")
	require.Contains(t, limits, infoFundingHistoryEPL, "Funding history endpoint limit must be configured")
	require.Contains(t, limits, infoUserLedgerEPL, "GetRateLimits must configure the user-ledger endpoint")
	require.Contains(t, limits, candleEndpointLimit(maximumCandleCount), "Maximum candle endpoint limit must be configured")
	assert.Equal(t, 21, recentTradesWeight, "Recent trades should reserve one response-size weight bucket")
	assert.Equal(t, 45, fundingHistoryWeight, "Funding history should reserve all 500 response-size buckets")
	assert.Equal(t, 45, userLedgerHistoryWeight, "User ledger history should reserve all 500 response-size buckets")
}

func TestCandleEndpointLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		count uint64
		want  request.EndpointLimit
	}{
		{name: "zero defaults to one", count: 0, want: 1},
		{name: "one", count: 1, want: 1},
		{name: "weight boundary", count: 60, want: 1},
		{name: "next weight", count: 61, want: 2},
		{name: "maximum", count: maximumCandleCount, want: 84},
		{name: "clamped", count: maximumCandleCount + 1, want: 84},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, candleEPLBase+tc.want, candleEndpointLimit(tc.count), "Candle endpoint limit should match its weighted request bucket")
		})
	}
}

func TestFormatInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		interval kline.Interval
		want     string
	}{
		{interval: kline.OneMin, want: "1m"},
		{interval: kline.ThreeMin, want: "3m"},
		{interval: kline.FiveMin, want: "5m"},
		{interval: kline.FifteenMin, want: "15m"},
		{interval: kline.ThirtyMin, want: "30m"},
		{interval: kline.OneHour, want: "1h"},
		{interval: kline.TwoHour, want: "2h"},
		{interval: kline.FourHour, want: "4h"},
		{interval: kline.EightHour, want: "8h"},
		{interval: kline.TwelveHour, want: "12h"},
		{interval: kline.OneDay, want: "1d"},
		{interval: kline.ThreeDay, want: "3d"},
		{interval: kline.OneWeek, want: "1w"},
		{interval: kline.OneMonth, want: "1M"},
	} {
		got, err := formatInterval(tc.interval)
		require.NoError(t, err, "Formatting a supported interval must not error")
		assert.Equal(t, tc.want, got, "Formatted interval should match the Hyperliquid API value")
	}

	_, err := formatInterval(kline.Interval(42))
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "Formatting an unsupported interval must return the expected error")
}
