package bybit

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestFetchTradablePairsFiltersNonTradingOptions(t *testing.T) {
	t.Parallel()

	tradingPair := currency.NewPairWithDelimiter("BTC", "14MAR26-95000-C", "-")
	settlingPair := currency.NewPairWithDelimiter("BTC", "14MAR26-90000-C", "-")
	preLaunchPair := currency.NewPairWithDelimiter("BTC", "14MAR26-85000-C", "-")
	tradingInstrument := newOptionInstrumentInfo("BTC14MAR26-95000-C", tradingStatus)
	settlingInstrument := newOptionInstrumentInfo("BTC14MAR26-90000-C", "Settling")
	preLaunchInstrument := newOptionInstrumentInfo("BTC14MAR26-85000-C", "PreLaunch")

	var queries []url.Values
	ex := newInstrumentInfoTestExchange(t, "BybitOptionsFetchTradablePairsTest", cOption, map[string][]InstrumentInfo{
		"BTC": {tradingInstrument, settlingInstrument, preLaunchInstrument},
	}, &queries)

	pairs, err := ex.FetchTradablePairs(t.Context(), asset.Options)
	require.NoError(t, err, "FetchTradablePairs must not error")
	assert.Equal(t, currency.Pairs{tradingPair}, pairs, "FetchTradablePairs should only include trading option pairs")

	require.NoError(t, ex.UpdatePairs(pairs, asset.Options, false), "UpdatePairs must not error for available option pairs")
	available, err := ex.GetAvailablePairs(asset.Options)
	require.NoError(t, err, "GetAvailablePairs must not error")
	assert.Equal(t, currency.Pairs{tradingPair}, available, "Available option pairs should only include trading instruments")

	enabled, err := ex.GetEnabledPairs(asset.Options)
	require.NoError(t, err, "GetEnabledPairs must not error")
	assert.Equal(t, currency.Pairs{tradingPair}, enabled, "Enabled option pairs should only include trading instruments")

	err = ex.CurrencyPairs.EnablePair(asset.Options, settlingPair)
	require.ErrorIs(t, err, currency.ErrPairNotFound, "EnablePair must error for a settling option pair")

	err = ex.CurrencyPairs.EnablePair(asset.Options, preLaunchPair)
	require.ErrorIs(t, err, currency.ErrPairNotFound, "EnablePair must error for a pre-launch option pair")

	assertOptionsInstrumentQueries(t, queries, "FetchTradablePairs")
}

func TestUpdateOrderExecutionLimitsFiltersNonTradingOptions(t *testing.T) {
	t.Parallel()

	tradingPair := currency.NewPairWithDelimiter("BTC", "14MAR26-95000-C", "-")
	settlingPair := currency.NewPairWithDelimiter("BTC", "14MAR26-90000-C", "-")
	closedPair := currency.NewPairWithDelimiter("BTC", "14MAR26-85000-C", "-")
	tradingInstrument := newOptionInstrumentInfo("BTC14MAR26-95000-C", tradingStatus)
	settlingInstrument := newOptionInstrumentInfo("BTC14MAR26-90000-C", "Settling")
	closedInstrument := newOptionInstrumentInfo("BTC14MAR26-85000-C", "Closed")

	var queries []url.Values
	ex := newInstrumentInfoTestExchange(t, "BybitOptionsUpdateLimitsTest", cOption, map[string][]InstrumentInfo{
		"BTC": {tradingInstrument, settlingInstrument, closedInstrument},
	}, &queries)

	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Options, currency.Pairs{tradingPair}, false), "StorePairs must not error for available option pairs")
	require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Options), "UpdateOrderExecutionLimits must not error")

	loadedLimit, err := ex.GetOrderExecutionLimits(asset.Options, tradingPair)
	require.NoError(t, err, "GetOrderExecutionLimits must not error for a trading option pair")
	require.True(t, loadedLimit.Key.Pair().Equal(tradingPair), "Loaded limit pair must match the trading option pair")
	assert.Equal(t, tradingInstrument.LotSizeFilter.MinOrderQuantity, loadedLimit.MinimumBaseAmount, "MinimumBaseAmount should match the trading option instrument")

	_, err = ex.GetOrderExecutionLimits(asset.Options, settlingPair)
	require.ErrorIs(t, err, limits.ErrOrderLimitNotFound, "GetOrderExecutionLimits must error for a settling option pair")

	_, err = ex.GetOrderExecutionLimits(asset.Options, closedPair)
	require.ErrorIs(t, err, limits.ErrOrderLimitNotFound, "GetOrderExecutionLimits must error for a closed option pair")

	assertOptionsInstrumentQueries(t, queries, "UpdateOrderExecutionLimits")
}

func TestUpdateOrderExecutionLimitsLeavesNonOptionsStatusHandlingUnchanged(t *testing.T) {
	t.Parallel()

	spotPair := currency.NewPairWithDelimiter("BTC", "USDT", "_")
	closedInstrument := newOptionInstrumentInfo("BTCUSDT", "Closed")

	var queries []url.Values
	ex := newInstrumentInfoTestExchange(t, "BybitSpotUpdateLimitsTest", cSpot, map[string][]InstrumentInfo{
		"": {closedInstrument},
	}, &queries)

	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{spotPair}, false), "StorePairs must not error for available spot pairs")
	require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Spot), "UpdateOrderExecutionLimits must not error for spot pairs")

	loadedLimit, err := ex.GetOrderExecutionLimits(asset.Spot, spotPair)
	require.NoError(t, err, "GetOrderExecutionLimits must not error for a closed spot pair")
	assert.Equal(t, closedInstrument.LotSizeFilter.MinOrderQuantity, loadedLimit.MinimumBaseAmount, "MinimumBaseAmount should match the closed spot instrument")

	require.Len(t, queries, 1, "UpdateOrderExecutionLimits must query spot instruments once")
	assert.Equal(t, cSpot, queries[0].Get("category"), "UpdateOrderExecutionLimits should request the spot category")
	assert.Empty(t, queries[0].Get("status"), "UpdateOrderExecutionLimits should not filter non-options instrument status")
	assert.Equal(t, "1000", queries[0].Get("limit"), "UpdateOrderExecutionLimits should request the expected page size")
}

func newInstrumentInfoTestExchange(t *testing.T, name, category string, responses map[string][]InstrumentInfo, queries *[]url.Values) *Exchange {
	t.Helper()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = name

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "Request method should be GET")
		assert.Equal(t, bybitAPIVersion+"market/instruments-info", r.URL.Path, "Request path should be the instruments info endpoint")

		query := r.URL.Query()
		*queries = append(*queries, query)

		payload := struct {
			RetCode int64  `json:"retCode"`
			RetMsg  string `json:"retMsg"`
			Result  *struct {
				Category       string           `json:"category"`
				List           []InstrumentInfo `json:"list"`
				NextPageCursor string           `json:"nextPageCursor"`
			} `json:"result"`
			Time int64 `json:"time"`
		}{
			RetCode: 0,
			RetMsg:  "OK",
			Result: &struct {
				Category       string           `json:"category"`
				List           []InstrumentInfo `json:"list"`
				NextPageCursor string           `json:"nextPageCursor"`
			}{
				Category:       category,
				List:           responses[query.Get("baseCoin")],
				NextPageCursor: "",
			},
			Time: time.Now().UnixMilli(),
		}

		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(payload)
		assert.NoError(t, err, "Encoding the instruments info response should not error")
	}))
	t.Cleanup(server.Close)

	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	return ex
}

type optionInstrumentInfoResponse struct {
	Symbol       string `json:"symbol"`
	OptionsType  string `json:"optionsType"`
	Status       string `json:"status"`
	BaseCoin     string `json:"baseCoin"`
	LaunchTime   int64  `json:"launchTime"`
	DeliveryTime int64  `json:"deliveryTime"`
	PriceFilter  struct {
		MinPrice float64 `json:"minPrice"`
		MaxPrice float64 `json:"maxPrice"`
		TickSize float64 `json:"tickSize"`
	} `json:"priceFilter"`
	LotSizeFilter struct {
		MinOrderQuantity float64 `json:"minOrderQty"`
		MaxOrderQuantity float64 `json:"maxOrderQty"`
		QuantityStep     float64 `json:"qtyStep"`
		QuotePrecision   float64 `json:"quotePrecision"`
		MaxOrderAmount   float64 `json:"maxOrderAmt"`
		MinNotionalValue float64 `json:"minNotionalValue"`
	} `json:"lotSizeFilter"`
}

func newOptionInstrumentInfo(symbol, status string) optionInstrumentInfoResponse {
	info := optionInstrumentInfoResponse{
		Symbol:      symbol,
		BaseCoin:    "BTC",
		Status:      status,
		OptionsType: "Call",
	}
	info.PriceFilter.MinPrice = 1
	info.PriceFilter.MaxPrice = 1_000_000
	info.PriceFilter.TickSize = 0.1
	info.LotSizeFilter.MinOrderQuantity = 0.1
	info.LotSizeFilter.MaxOrderQuantity = 10
	info.LotSizeFilter.QuantityStep = 0.1
	info.LotSizeFilter.QuotePrecision = 0.1
	info.LotSizeFilter.MaxOrderAmount = 1_000_000
	info.LotSizeFilter.MinNotionalValue = 1
	return info
}

func assertOptionsInstrumentQueries(t *testing.T, queries []url.Values, caller string) {
	t.Helper()

	require.Len(t, queries, len(supportedOptionsTypes), caller+" must query every supported option base coin")

	requestedBaseCoins := make([]string, 0, len(queries))
	for _, query := range queries {
		requestedBaseCoins = append(requestedBaseCoins, query.Get("baseCoin"))
		assert.Equal(t, cOption, query.Get("category"), caller+" should request the option category")
		assert.Equal(t, tradingStatus, query.Get("status"), caller+" should request trading option instruments")
		assert.Equal(t, "1000", query.Get("limit"), caller+" should request the expected page size")
	}

	assert.ElementsMatch(t, supportedOptionsTypes, requestedBaseCoins, caller+" should query every supported option base coin exactly once")
}
