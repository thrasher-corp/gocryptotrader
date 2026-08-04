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
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
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

func TestSettlementCurrencyForContract(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		item     asset.Item
		pair     currency.Pair
		expected currency.Code
		err      error
	}{
		{name: "delivery futures", item: asset.Futures, pair: btccwPair, expected: currency.BTC},
		{name: "coin margined futures", item: asset.CoinMarginedFutures, pair: btcusdPair, expected: currency.BTC},
		{name: "USDT margined futures", item: asset.USDTMarginedFutures, pair: btcusdtPair, expected: currency.USDT},
		{name: "empty pair", item: asset.Futures, err: currency.ErrCurrencyPairEmpty},
		{name: "unsupported asset", item: asset.Spot, pair: btcusdtPair, err: asset.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := settlementCurrencyForContract(tc.item, tc.pair)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "settlementCurrencyForContract must return the expected error")
				return
			}
			require.NoError(t, err, "settlementCurrencyForContract must not error")
			assert.Equal(t, tc.expected, got, "settlement currency should match")
		})
	}
}

func TestGetCollateralCurrencyForContract(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	code, item, err := h.GetCollateralCurrencyForContract(asset.CoinMarginedFutures, btcusdPair)
	require.NoError(t, err, "GetCollateralCurrencyForContract must not error")
	assert.Equal(t, currency.BTC, code, "collateral currency should match")
	assert.Equal(t, asset.CoinMarginedFutures, item, "collateral asset should match")
}

func TestGetCurrencyForRealisedPNL(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	code, item, err := h.GetCurrencyForRealisedPNL(asset.USDTMarginedFutures, btcusdtPair)
	require.NoError(t, err, "GetCurrencyForRealisedPNL must not error")
	assert.Equal(t, currency.USDT, code, "realised PNL currency should match")
	assert.Equal(t, asset.USDTMarginedFutures, item, "realised PNL asset should match")
}

func TestSetCollateralMode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		mode         collateral.Mode
		expectedMode uint64
	}{
		{name: "multi asset", mode: collateral.MultiMode, expectedMode: 1},
		{name: "single asset", mode: collateral.SingleMode, expectedMode: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/account/asset_mode", `{"code":200,"data":{"asset_mode":1}}`, func(r *http.Request) {
				var req V5SetAssetModeRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&req), "asset-mode request must decode")
				assert.Equal(t, tc.expectedMode, req.AssetMode, "HTX asset mode should match")
			})
			require.NoError(t, h.SetCollateralMode(t.Context(), asset.USDTMarginedFutures, tc.mode), "SetCollateralMode must not error")
		})
	}
	h := new(Exchange)
	require.ErrorIs(t, h.SetCollateralMode(t.Context(), asset.Spot, collateral.MultiMode), asset.ErrNotSupported, "SetCollateralMode must reject unsupported assets")
	require.ErrorIs(t, h.SetCollateralMode(t.Context(), asset.USDTMarginedFutures, collateral.PortfolioMode), collateral.ErrInvalidCollateralMode, "SetCollateralMode must reject unsupported modes")
}

func TestGetCollateralMode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		assetMode uint64
		expected  collateral.Mode
		err       error
	}{
		{name: "multi asset", assetMode: 1, expected: collateral.MultiMode},
		{name: "single asset", assetMode: 2, expected: collateral.SingleMode},
		{name: "unknown asset mode", assetMode: 3, err: collateral.ErrInvalidCollateralMode},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/asset_mode",
				`{"code":200,"data":{"asset_mode":`+strconv.FormatUint(tc.assetMode, 10)+`}}`, nil)
			got, err := h.GetCollateralMode(t.Context(), asset.USDTMarginedFutures)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "GetCollateralMode must return the expected error")
				return
			}
			require.NoError(t, err, "GetCollateralMode must not error")
			assert.Equal(t, tc.expected, got, "collateral mode should match")
		})
	}
	h := new(Exchange)
	_, err := h.GetCollateralMode(t.Context(), asset.Spot)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCollateralMode must reject unsupported assets")
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		item       asset.Item
		pair       currency.Pair
		marginType margin.Type
		endpoint   exchange.URL
		method     string
		path       string
		response   string
	}{
		{
			name: "delivery futures", item: asset.Futures, pair: btccwPair, marginType: margin.Isolated,
			endpoint: exchange.RestFutures, method: http.MethodPost, path: fAccountData,
			response: `{"status":"ok","data":[{"symbol":"BTC","lever_rate":5}]}`,
		},
		{
			name: "coin margined futures", item: asset.CoinMarginedFutures, pair: btcusdPair, marginType: margin.Isolated,
			endpoint: exchange.RestFutures, method: http.MethodPost, path: "/swap-api/v1/swap_account_info",
			response: `{"status":"ok","data":[{"contract_code":"BTC-USD","lever_rate":5}]}`,
		},
		{
			name: "USDT margined futures", item: asset.USDTMarginedFutures, pair: btcusdtPair, marginType: margin.Multi,
			endpoint: exchange.RestUSDTMargined, method: http.MethodGet, path: "/v5/position/lever",
			response: `{"code":200,"data":[{"contract_code":"BTC-USDT","margin_mode":"cross","position_side":"both","lever_rate":5}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, tc.endpoint, tc.method, tc.path, tc.response, nil)
			got, err := h.GetLeverage(t.Context(), tc.item, tc.pair, tc.marginType, order.UnknownSide)
			require.NoError(t, err, "GetLeverage must not error")
			assert.Equal(t, 5.0, got, "leverage should match")
		})
	}

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	_, err := h.GetLeverage(t.Context(), asset.Spot, btcusdtPair, margin.Unset, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetLeverage must reject unsupported assets")
	_, err = h.GetLeverage(t.Context(), asset.Futures, currency.EMPTYPAIR, margin.Unset, order.UnknownSide)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetLeverage must reject empty pairs")
	_, err = newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fAccountData, `{"status":"ok","data":[]}`, nil).
		GetLeverage(t.Context(), asset.Futures, btccwPair, margin.Isolated, order.UnknownSide)
	require.ErrorIs(t, err, futures.ErrPositionNotFound, "GetLeverage must report missing leverage data")
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	_, err := h.ChangePositionMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangePositionMargin must reject nil requests")
	_, err = h.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "ChangePositionMargin must reject unsupported assets")
	_, err = h.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ChangePositionMargin must reject empty pairs")
	_, err = h.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{
		Asset: asset.USDTMarginedFutures, Pair: btcusdtPair, MarginType: margin.Multi,
	})
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "ChangePositionMargin must reject cross margin")

	resp, err := h.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{
		Asset: asset.USDTMarginedFutures, Pair: btcusdtPair, MarginType: margin.Isolated,
		OriginalAllocatedMargin: 2, NewAllocatedMargin: 2,
	})
	require.NoError(t, err, "ChangePositionMargin must accept a no-op adjustment")
	assert.Equal(t, 2.0, resp.AllocatedMargin, "no-op allocated margin should be unchanged")

	h = newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/position/margin", `{"code":200}`, nil)
	resp, err = h.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{
		Asset: asset.USDTMarginedFutures, Pair: btcusdtPair, MarginType: margin.Isolated,
		OriginalAllocatedMargin: 2, NewAllocatedMargin: 5,
	})
	require.NoError(t, err, "ChangePositionMargin must not error")
	assert.Equal(t, 5.0, resp.AllocatedMargin, "allocated margin should match")
}
