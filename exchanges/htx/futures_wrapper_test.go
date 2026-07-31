package htx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestGetHistoricalFundingRatesForPair(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/swap-api/v1/swap_historical_funding_rate", r.URL.Path, "coin-margined funding history path should match")
		assert.Equal(t, "25", r.URL.Query().Get("page_size"), "page size should use the pageSize argument")
		assert.Equal(t, "2", r.URL.Query().Get("page_index"), "page index should use the pageIndex argument")
		_, _ = w.Write([]byte(`{"status":"ok","data":{"total_page":1,"current_page":1,"total_size":1,"data":[{"funding_rate":"0.001","funding_time":"1604312615051","contract_code":"BTC-USD"}]}}`))
	}))
	t.Cleanup(server.Close)

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "futures endpoint must be set")
	got, err := h.GetHistoricalFundingRatesForPair(t.Context(), btcusdPair, 25, 2)
	require.NoError(t, err, "GetHistoricalFundingRatesForPair must not error")
	require.Len(t, got.Data.Data, 1, "one funding rate must be returned")
	assert.Equal(t, time.UnixMilli(1604312615051), got.Data.Data[0].FundingTime.Time(), "funding time should match")
}

func TestGetLinearSwapHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, linearSwapFundingHistory, r.URL.Path, "USDT-margined funding history path should match")
		assert.Equal(t, "25", r.URL.Query().Get("page_size"), "page size should use the pageSize argument")
		assert.Equal(t, "2", r.URL.Query().Get("page_index"), "page index should use the pageIndex argument")
		_, _ = w.Write([]byte(`{"status":"ok","data":{"total_page":1,"current_page":1,"total_size":1,"data":[{"funding_rate":"0.001","funding_time":"1604312615051","contract_code":"BTC-USDT"}]}}`))
	}))
	t.Cleanup(server.Close)

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
	got, err := h.GetLinearSwapHistoricalFundingRates(t.Context(), btcusdtPair, 25, 2)
	require.NoError(t, err, "GetLinearSwapHistoricalFundingRates must not error")
	require.Len(t, got.Data.Data, 1, "one funding rate must be returned")
	assert.Equal(t, time.UnixMilli(1604312615051), got.Data.Data[0].FundingTime.Time(), "funding time should match")
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
}

func TestFSwitchLeverage(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, fSwitchLeverage, r.URL.Path, "delivery leverage path should match")
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "request body should be readable")
		assert.JSONEq(t, `{"symbol":"BTC","lever_rate":5}`, string(body), "delivery leverage body should match")
		_, _ = w.Write([]byte(`{"status":"ok","data":{"symbol":"BTC","lever_rate":5}}`))
	}))
	t.Cleanup(server.Close)

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "futures endpoint must be set")
	require.NoError(t, h.FSwitchLeverage(t.Context(), currency.BTC, 5), "FSwitchLeverage must not error")
}

func TestSwitchCoinMarginedLeverage(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/swap-api/v1/swap_switch_lever_rate", r.URL.Path, "coin-margined leverage path should match")
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "request body should be readable")
		assert.JSONEq(t, `{"contract_code":"BTC-USD","lever_rate":5}`, string(body), "coin-margined leverage body should match")
		_, _ = w.Write([]byte(`{"status":"ok","data":{"contract_code":"BTC-USD","lever_rate":5}}`))
	}))
	t.Cleanup(server.Close)

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "futures endpoint must be set")
	require.NoError(t, h.SwitchCoinMarginedLeverage(t.Context(), btcusdPair, 5), "SwitchCoinMarginedLeverage must not error")
}

func TestSwitchLinearSwapLeverage(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v5/position/lever", r.URL.Path, "USDT-margined leverage path should use the current V5 endpoint")
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "request body should be readable")
		assert.Contains(t, string(body), `"margin_mode"`, "V5 leverage request should include margin mode")
		assert.Contains(t, string(body), `"lever_rate":"5"`, "V5 leverage request should encode leverage as documented")
		_, _ = w.Write([]byte(`{"code":200,"message":"Success","data":{"contract_code":"BTC-USDT","lever_rate":"5"}}`))
	}))
	t.Cleanup(server.Close)

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
	require.NoError(t, h.SwitchLinearSwapLeverage(t.Context(), btcusdtPair, 5, false), "isolated SwitchLinearSwapLeverage must not error")
	require.NoError(t, h.SwitchLinearSwapLeverage(t.Context(), btcusdtPair, 5, true), "cross SwitchLinearSwapLeverage must not error")
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

func TestGetHistoricCandlesUSDTMargined(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Minute)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, linearSwapKline, r.URL.Path, "USDT-margined kline path should match")
		_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":` + strconv.FormatInt(now.Add(-time.Minute).Unix(), 10) + `,"open":1,"high":2,"low":0.5,"close":1.5,"vol":10}]}`))
	}))
	t.Cleanup(server.Close)

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.SetPairs(currency.Pairs{btcusdtPair}, asset.USDTMarginedFutures, false), "available pair must be set")
	require.NoError(t, h.SetPairs(currency.Pairs{btcusdtPair}, asset.USDTMarginedFutures, true), "enabled pair must be set")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")

	got, err := h.GetHistoricCandles(t.Context(), btcusdtPair, asset.USDTMarginedFutures, kline.OneMin, now.Add(-2*time.Minute), now)
	require.NoError(t, err, "GetHistoricCandles must not error for USDT-margined futures")
	require.NotEmpty(t, got.Candles, "candles must be returned")

	got, err = h.GetHistoricCandlesExtended(t.Context(), btcusdtPair, asset.USDTMarginedFutures, kline.OneMin, now.Add(-2*time.Minute), now)
	require.NoError(t, err, "GetHistoricCandlesExtended must not error for USDT-margined futures")
	require.NotEmpty(t, got.Candles, "extended candles must be returned")
}
