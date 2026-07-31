package htx

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestAppendFuturesCandles(t *testing.T) {
	t.Parallel()
	start := time.Unix(100, 0)
	end := time.Unix(200, 0)
	for _, tc := range []struct {
		name     string
		candles  []FuturesKline
		expected int
	}{
		{
			name: "inclusive range and field mapping",
			candles: []FuturesKline{
				{IDTimestamp: types.Time(start), Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 3},
				{IDTimestamp: types.Time(end), Open: 4, High: 5, Low: 3.5, Close: 4.5, Volume: 6},
			},
			expected: 2,
		},
		{
			name: "outside range",
			candles: []FuturesKline{
				{IDTimestamp: types.Time(start.Add(-time.Second))},
				{IDTimestamp: types.Time(end.Add(time.Second))},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := appendFuturesCandles(nil, tc.candles, start, end)
			require.Len(t, result, tc.expected, "appendFuturesCandles must return the expected candles")
			if tc.expected == 0 {
				return
			}
			assert.Equal(t, start, result[0].Time, "candle time should map")
			assert.Equal(t, float64(1), result[0].Open, "open should map")
			assert.Equal(t, float64(2), result[0].High, "high should map")
			assert.Equal(t, float64(0.5), result[0].Low, "low should map")
			assert.Equal(t, float64(1.5), result[0].Close, "close should map")
			assert.Equal(t, float64(3), result[0].Volume, "volume should map")
		})
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")

	_, err := h.GetHistoricalFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetHistoricalFundingRates must reject a nil request")
	_, err = h.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricalFundingRates must reject unsupported assets")
	_, err = h.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.CoinMarginedFutures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetHistoricalFundingRates must require a pair")

	const fundingTimestamp = int64(1604312615051)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","data":{"total_page":1,"current_page":1,"total_size":1,"data":[{"funding_rate":"0.001","funding_time":"1604312615051","contract_code":"BTC-USD"}]}}`))
	}))
	t.Cleanup(server.Close)
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "futures endpoint must be set")

	at := time.UnixMilli(fundingTimestamp)
	got, err := h.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:     asset.CoinMarginedFutures,
		Pair:      btcusdPair,
		StartDate: at.Add(-time.Minute),
		EndDate:   at.Add(time.Minute),
	})
	require.NoError(t, err, "GetHistoricalFundingRates must not error")
	require.Len(t, got.FundingRates, 1, "one funding rate must be returned")
	assert.Equal(t, at, got.FundingRates[0].Time, "funding time should match")
	assert.Equal(t, "0.001", got.FundingRates[0].Rate.String(), "funding rate should match")

	usdtServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v5/market/funding_rate_history", r.URL.Path, "USDT-margined funding history should use V5")
		assert.Equal(t, strconv.FormatInt(at.Add(-time.Minute).UnixMilli(), 10), r.URL.Query().Get("start_time"), "start time should be sent")
		assert.Equal(t, strconv.FormatInt(at.Add(time.Minute).UnixMilli(), 10), r.URL.Query().Get("end_time"), "end time should be sent")
		_, _ = w.Write([]byte(`{"code":200,"data":[{"id":"1","funding_rate":"0.001","funding_time":"1604312615051","contract_code":"BTC-USDT"}]}`))
	}))
	t.Cleanup(usdtServer.Close)
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), usdtServer.URL), "USDT-margined endpoint must be set")
	got, err = h.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:     asset.USDTMarginedFutures,
		Pair:      btcusdtPair,
		StartDate: at.Add(-time.Minute),
		EndDate:   at.Add(time.Minute),
	})
	require.NoError(t, err, "USDT-margined GetHistoricalFundingRates must not error")
	require.Len(t, got.FundingRates, 1, "one USDT-margined funding rate must be returned")
	assert.Equal(t, at, got.FundingRates[0].Time, "USDT-margined funding time should match")
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")

	for _, tc := range []struct {
		name       string
		item       asset.Item
		pair       currency.Pair
		marginType margin.Type
		amount     float64
		expected   error
	}{
		{name: "empty pair", item: asset.Futures, marginType: margin.Isolated, amount: 5, expected: currency.ErrCurrencyPairEmpty},
		{name: "zero leverage", item: asset.Futures, pair: btcusdPair, marginType: margin.Isolated, expected: errInvalidLeverage},
		{name: "fractional leverage", item: asset.Futures, pair: btcusdPair, marginType: margin.Isolated, amount: 1.5, expected: errInvalidLeverage},
		{name: "unsupported asset", item: asset.Spot, pair: btcusdPair, marginType: margin.Isolated, amount: 5, expected: asset.ErrNotSupported},
		{name: "delivery cross margin", item: asset.Futures, pair: btcusdPair, marginType: margin.Multi, amount: 5, expected: margin.ErrMarginTypeUnsupported},
		{name: "coin cross margin", item: asset.CoinMarginedFutures, pair: btcusdPair, marginType: margin.Multi, amount: 5, expected: margin.ErrMarginTypeUnsupported},
		{name: "USDT unsupported margin", item: asset.USDTMarginedFutures, pair: btcusdtPair, marginType: margin.NoMargin, amount: 5, expected: margin.ErrMarginTypeUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := h.SetLeverage(t.Context(), tc.item, tc.pair, tc.marginType, tc.amount, order.UnknownSide)
			require.ErrorIs(t, err, tc.expected, "SetLeverage must return the expected validation error")
		})
	}
}
