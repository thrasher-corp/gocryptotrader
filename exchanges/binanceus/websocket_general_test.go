package binanceus

import (
	"testing"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestWebsocketAPITestConnectivity(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, false)
	assert.NoError(t, ex.WebsocketAPI().TestConnectivity(t.Context()), "TestConnectivity should not error")

	pinged := newWebsocketTestExchange(t)
	useMockWebsocket(t, pinged, pinged.websocketAPIConnectionSetup(pinged.Config), func(tb testing.TB, msg []byte, w *gws.Conn) error {
		tb.Helper()
		method, err := jsonparser.GetString(msg, "method")
		require.NoError(tb, err, "the request must name its method")
		assert.Equal(tb, "ping", method, "TestConnectivity should send ping")
		return websocketAPIMock(tb, msg, w)
	})
	require.NoError(t, pinged.Websocket.Connect(t.Context()), "Connect must not error")
	assert.NoError(t, pinged.WebsocketAPI().TestConnectivity(t.Context()), "TestConnectivity should not error against the local server")
}

func TestWebsocketAPIGetExchangeServerTime(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, false)
	result, err := ex.WebsocketAPI().GetExchangeServerTime(t.Context())
	require.NoError(t, err, "GetExchangeServerTime must not error")
	require.NotNil(t, result, "GetExchangeServerTime must return a result")
	if mockTests {
		assert.Equal(t, &ServerTimeResponse{ServerTime: types.Time(time.UnixMilli(1794476705398))}, result, "GetExchangeServerTime should decode the server time")
		return
	}
	assert.WithinDuration(t, time.Now(), result.ServerTime.Time(), time.Minute, "GetExchangeServerTime should return the current time")
}

func TestWebsocketAPIGetExchangeInfo(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, false)
	result, err := ex.WebsocketAPI().GetExchangeInfo(t.Context(), &ExchangeInfoRequest{Symbols: currency.Pairs{currency.NewBTCUSDT()}})
	require.NoError(t, err, "GetExchangeInfo must not error")
	require.NotNil(t, result, "GetExchangeInfo must return a result")
	if !mockTests {
		require.Len(t, result.Symbols, 1, "GetExchangeInfo must return the requested symbol")
		assert.Equal(t, "BTCUSDT", result.Symbols[0].Symbol, "GetExchangeInfo should return the requested symbol")
		recorded := websocketAPIFixtureResult[*ExchangeInfoResponse](t, "exchangeInfo", 0)
		assert.Equal(t, recorded.RateLimits, result.RateLimits, "GetExchangeInfo should return the recorded rate limits, which the rate limit pools are budgeted from")
		return
	}
	exp := &ExchangeInfoResponse{
		Timezone:   "UTC",
		ServerTime: types.Time(time.UnixMilli(1794478353820)),
		RateLimits: []RateLimitItemResponse{
			{RateLimitType: "REQUEST_WEIGHT", Interval: "MINUTE", IntervalNumber: 1, Limit: 6000},
			{RateLimitType: "ORDERS", Interval: "SECOND", IntervalNumber: 10, Limit: 100},
			{RateLimitType: "ORDERS", Interval: "DAY", IntervalNumber: 1, Limit: 200000},
			{RateLimitType: "CONNECTIONS", Interval: "MINUTE", IntervalNumber: 5, Limit: 300},
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
				{FilterType: "MARKET_LOT_SIZE", MaxQuantity: 12.00586791},
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
