package binanceus

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestTestConnectivity(t *testing.T) {
	t.Parallel()
	assert.NoError(t, e.TestConnectivity(t.Context()), "TestConnectivity should not error")
}

func TestGetExchangeServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetExchangeServerTime(t.Context())
	require.NoError(t, err, "GetExchangeServerTime must not error")
	require.NotNil(t, result, "GetExchangeServerTime must return a result")
	if mockTests {
		assert.Equal(t, &ServerTimeResponse{ServerTime: types.Time(time.UnixMilli(1794459130787))}, result, "GetExchangeServerTime should decode the server time")
		return
	}
	assert.WithinDuration(t, time.Now(), result.ServerTime.Time(), time.Minute, "GetExchangeServerTime should return the current time")
}

func TestGetSystemStatus(t *testing.T) {
	t.Parallel()
	skipIfCredentialsUnset(t)
	result, err := e.GetSystemStatus(t.Context())
	require.NoError(t, err, "GetSystemStatus must not error")
	require.NotNil(t, result, "GetSystemStatus must return a result")
	if mockTests {
		assert.Equal(t, &SystemStatusResponse{Status: 0}, result, "GetSystemStatus should decode the status")
	}
}

func TestGetExchangeInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetExchangeInfo(t.Context(), &ExchangeInfoRequest{Symbols: currency.Pairs{currency.NewBTCUSDT()}, Permissions: []Permission{PermissionSpot}})
	require.ErrorIs(t, err, errInvalidParameterCombination, "GetExchangeInfo must reject permissions with symbols")
	_, err = e.GetExchangeInfo(t.Context(), &ExchangeInfoRequest{Symbols: currency.Pairs{currency.EMPTYPAIR}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetExchangeInfo must reject an empty pair")

	t.Run("sends the request's parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *ExchangeInfoRequest
			exp  url.Values
		}{
			{name: "nil request", exp: url.Values{}},
			{name: "one symbol", req: &ExchangeInfoRequest{Symbols: currency.Pairs{currency.NewBTCUSDT()}}, exp: url.Values{"symbol": {"BTCUSDT"}}},
			{name: "several symbols without permission sets", req: &ExchangeInfoRequest{Symbols: currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USD)}, ShowPermissionSets: new(false)}, exp: url.Values{"symbols": {`["BTCUSDT","ETHUSD"]`}, "showPermissionSets": {"false"}}},
			{name: "one permission", req: &ExchangeInfoRequest{Permissions: []Permission{PermissionSpot}, ShowPermissionSets: new(true)}, exp: url.Values{"permissions": {"SPOT"}, "showPermissionSets": {"true"}}},
			{name: "several permissions", req: &ExchangeInfoRequest{Permissions: []Permission{PermissionSpot, "TRD_GRP_004"}}, exp: url.Values{"permissions": {`["SPOT","TRD_GRP_004"]`}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/api/v3/exchangeInfo", r.URL.Path, "GetExchangeInfo should request the exchange information path")
					assert.Equal(t, tc.exp, r.URL.Query(), "GetExchangeInfo should send the request's parameters")
					writeResponse(t, w, `{}`)
				})
				_, err := ex.GetExchangeInfo(t.Context(), tc.req)
				require.NoError(t, err, "GetExchangeInfo must not error")
			})
		}
	})

	ensureTradablePairs(t)
	result, err := e.GetExchangeInfo(t.Context(), &ExchangeInfoRequest{Symbols: currency.Pairs{spotTradablePair}})
	require.NoError(t, err, "GetExchangeInfo must not error")
	require.NotNil(t, result, "GetExchangeInfo must return a result")
	if !mockTests {
		require.Len(t, result.Symbols, 1, "GetExchangeInfo must return the requested symbol")
		assert.NotEmpty(t, result.RateLimits, "GetExchangeInfo should return the rate limits")
		return
	}
	exp := &ExchangeInfoResponse{
		Timezone:   "UTC",
		ServerTime: types.Time(time.UnixMilli(1794459131446)),
		RateLimits: []RateLimitItemResponse{
			{RateLimitType: "REQUEST_WEIGHT", Interval: "MINUTE", IntervalNumber: 1, Limit: 6000},
			{RateLimitType: "ORDERS", Interval: "SECOND", IntervalNumber: 10, Limit: 100},
			{RateLimitType: "ORDERS", Interval: "DAY", IntervalNumber: 1, Limit: 200000},
			{RateLimitType: "RAW_REQUESTS", Interval: "MINUTE", IntervalNumber: 5, Limit: 300000},
		},
		ExchangeFilters: []ExchangeFilter{},
		Symbols: []SymbolInfo{{
			Symbol:                          "BTCUSDT",
			Status:                          "TRADING",
			BaseAsset:                       currency.BTC,
			BaseAssetPrecision:              8,
			QuoteAsset:                      currency.USDT,
			QuotePrecision:                  8,
			QuoteAssetPrecision:             8,
			BaseCommissionPrecision:         8,
			QuoteCommissionPrecision:        8,
			OrderTypes:                      []OrderType{OrderTypeLimit, OrderTypeLimitMaker, OrderTypeMarket, OrderTypeStopLoss, OrderTypeStopLossLimit, OrderTypeTakeProfit, OrderTypeTakeProfitLimit},
			IcebergAllowed:                  true,
			OneCancelsTheOtherAllowed:       true,
			OneTriggersTheOtherAllowed:      true,
			OnePaysTheOtherAllowed:          true,
			QuoteOrderQuantityMarketAllowed: true,
			AllowTrailingStop:               true,
			CancelReplaceAllowed:            true,
			AmendAllowed:                    true,
			PegInstructionsAllowed:          true,
			IsSpotTradingAllowed:            true,
			Filters: []SymbolFilter{
				{FilterType: "PRICE_FILTER", MinPrice: 0.01, MaxPrice: 1000000, TickSize: 0.01},
				{FilterType: "PERCENT_PRICE", MultiplierUp: 5, MultiplierDown: 0.2, AveragePriceMinutes: 5},
				{FilterType: "LOT_SIZE", MinQuantity: 0.00001, MaxQuantity: 9000, StepSize: 0.00001},
				{FilterType: "MIN_NOTIONAL", MinNotional: 1, ApplyToMarket: true, AveragePriceMinutes: 5},
				{FilterType: "ICEBERG_PARTS", Limit: 10},
				{FilterType: "MARKET_LOT_SIZE", MaxQuantity: 11.92655537},
				{FilterType: "TRAILING_DELTA", MinTrailingAboveDelta: 10, MaxTrailingAboveDelta: 2000, MinTrailingBelowDelta: 10, MaxTrailingBelowDelta: 2000},
				{FilterType: "PERCENT_PRICE_BY_SIDE", BidMultiplierUp: 5, BidMultiplierDown: 0.2, AskMultiplierUp: 5, AskMultiplierDown: 0.2, AveragePriceMinutes: 5},
				{FilterType: "MAX_NUM_ORDERS", MaxNumberOrders: 200},
				{FilterType: "MAX_NUM_ORDER_LISTS", MaxNumberOrderLists: 20},
				{FilterType: "MAX_NUM_ALGO_ORDERS", MaxNumberAlgoOrders: 5},
				{FilterType: "MAX_NUM_ORDER_AMENDS", MaxNumberOrderAmends: 10},
			},
			Permissions:                     []Permission{},
			PermissionSets:                  [][]Permission{{PermissionSpot}},
			DefaultSelfTradePreventionMode:  SelfTradePreventionModeExpireMaker,
			AllowedSelfTradePreventionModes: []SelfTradePreventionMode{SelfTradePreventionModeExpireTaker, SelfTradePreventionModeExpireMaker, SelfTradePreventionModeExpireBoth, SelfTradePreventionModeDecrement, SelfTradePreventionModeTransfer},
		}},
	}
	assert.Equal(t, exp, result, "GetExchangeInfo should decode the exchange information")
}
