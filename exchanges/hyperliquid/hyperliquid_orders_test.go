package hyperliquid

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func newTradingTestExchange(t *testing.T, infoResponses map[string]string, actionResponse func(string, map[string]any) string) *Exchange {
	t.Helper()
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/info":
			var infoPayload infoRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&infoPayload), "Decoding trading info request should not error") {
				return
			}
			response, ok := infoResponses[infoPayload.Type]
			if !ok {
				switch infoPayload.Type {
				case infoTypePerpetualDEXs:
					response = `[null]`
				case infoTypeMetadata:
					response = perpetualMetadataJSON
				case "spotMeta":
					response = spotMetadataJSON
				case testUserRoleInfoType:
					response = testUserRoleResponse
				default:
					http.Error(w, "unexpected info request "+infoPayload.Type, http.StatusBadRequest)
					return
				}
			}
			_, err := w.Write([]byte(response))
			assert.NoError(t, err, "Writing trading info response should not error")
		case "/exchange":
			var actionPayload struct {
				Action map[string]any `json:"action"`
			}
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&actionPayload), "Decoding signed trading request should not error") {
				return
			}
			actionType, _ := actionPayload.Action["type"].(string)
			response := ""
			if actionResponse != nil {
				response = actionResponse(actionType, actionPayload.Action)
			}
			if response == "" {
				http.Error(w, "unexpected exchange action "+actionType, http.StatusBadRequest)
				return
			}
			_, err := w.Write([]byte(response))
			assert.NoError(t, err, "Writing signed trading response should not error")
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey})
	ex.setPairMappings(asset.PerpetualContract, []pairMapping{{
		pair:         testPerpetualPair,
		coin:         "BTC",
		assetID:      0,
		sizeDecimals: 5,
		maxLeverage:  40,
	}})
	ex.setPairMappings(asset.Spot, []pairMapping{{
		pair:         testSpotPair,
		coin:         "@107",
		assetID:      10107,
		sizeDecimals: 2,
	}})
	return ex
}

func mustOpenOrder(t *testing.T, raw string) OpenOrder {
	t.Helper()
	var result OpenOrder
	require.NoError(t, json.Unmarshal([]byte(raw), &result), "Decoding test order must not error")
	return result
}

func TestFormatOrderSize(t *testing.T) {
	for _, tc := range []struct {
		name       string
		size       float64
		decimals   uint64
		expected   string
		expectedIs error
	}{
		{name: "integer", size: 2, decimals: 0, expected: "2"},
		{name: "fraction", size: 1.23, decimals: 2, expected: "1.23"},
		{name: "zero", size: 0, decimals: 2, expectedIs: errSizePrecision},
		{name: "negative", size: -1, decimals: 2, expectedIs: errSizePrecision},
		{name: "metadata precision", size: 1, decimals: 9, expectedIs: errSizePrecision},
		{name: "excess precision", size: 1.234, decimals: 2, expectedIs: errSizePrecision},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := formatOrderSize(tc.size, tc.decimals)
			require.ErrorIs(t, err, tc.expectedIs, "Formatting order size must return the expected error")
			assert.Equal(t, tc.expected, result, "Formatted order size should match")
		})
	}
}

func TestDeriveFilledOrderState(t *testing.T) {
	for _, tc := range []struct {
		name              string
		requested         float64
		filled            float64
		decimals          uint64
		timeInForce       string
		expectedStatus    order.Status
		expectedRemaining float64
		expectedIs        error
	}{
		{name: "filled", requested: 0.1, filled: 0.1, decimals: 5, timeInForce: wireTimeInForceGTC, expectedStatus: order.Filled},
		{name: "partially filled GTC", requested: 0.1, filled: 0.04, decimals: 5, timeInForce: wireTimeInForceGTC, expectedStatus: order.PartiallyFilled, expectedRemaining: 0.06},
		{name: "partially filled IOC", requested: 0.1, filled: 0.04, decimals: 5, timeInForce: wireTimeInForceIOC, expectedStatus: order.PartiallyFilledCancelled, expectedRemaining: 0.06},
		{name: "partially filled ALO", requested: 0.1, filled: 0.04, decimals: 5, timeInForce: wireTimeInForceALO, expectedIs: errActionStatusMalformed},
		{name: "partially filled trigger", requested: 0.1, filled: 0.04, decimals: 5, expectedIs: errActionStatusMalformed},
		{name: "invalid requested size", requested: 0, filled: 0.04, decimals: 5, timeInForce: wireTimeInForceGTC, expectedIs: errInvalidFilledSize},
		{name: "invalid reported precision", requested: 0.1, filled: 0.0400001, decimals: 5, timeInForce: wireTimeInForceGTC, expectedIs: errInvalidFilledSize},
		{name: "unsupported metadata precision", requested: 0.1, filled: 0.04, decimals: 9, timeInForce: wireTimeInForceGTC, expectedIs: errInvalidFilledSize},
		{name: "over-reported fill", requested: 0.1, filled: 0.11, decimals: 5, timeInForce: wireTimeInForceGTC, expectedIs: errInvalidFilledSize},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, remaining, err := deriveFilledOrderState(tc.requested, tc.filled, tc.decimals, tc.timeInForce)
			require.ErrorIs(t, err, tc.expectedIs, "Deriving filled order state must return the expected error")
			assert.Equal(t, tc.expectedStatus, status, "Derived order status should match")
			assert.InDelta(t, tc.expectedRemaining, remaining, 1e-12, "Derived remaining amount should match")
		})
	}
}

func TestValidateLimitPrice(t *testing.T) {
	for _, tc := range []struct {
		name       string
		price      float64
		asset      asset.Item
		decimals   uint64
		expectedIs error
	}{
		{name: "perpetual", price: 1234.5, asset: asset.PerpetualContract, decimals: 5},
		{name: "spot", price: 0.001234, asset: asset.Spot, decimals: 2},
		{name: "large integer", price: 123456789, asset: asset.PerpetualContract, decimals: 5},
		{name: "zero", asset: asset.Spot, expectedIs: errInvalidMarketPrice},
		{name: "nan", price: math.NaN(), asset: asset.Spot, expectedIs: errInvalidMarketPrice},
		{name: "infinity", price: math.Inf(1), asset: asset.Spot, expectedIs: errInvalidMarketPrice},
		{name: "unsupported asset", price: 1, asset: asset.Options, expectedIs: asset.ErrNotSupported},
		{name: "invalid metadata precision", price: 1, asset: asset.PerpetualContract, decimals: 7, expectedIs: errSizePrecision},
		{name: "too many price decimals", price: 100.55, asset: asset.PerpetualContract, decimals: 5, expectedIs: errPricePrecision},
		{name: "too many significant figures", price: 12.3456, asset: asset.Spot, decimals: 2, expectedIs: errPricePrecision},
		{name: "wire precision", price: 0.000000001, asset: asset.Spot, expectedIs: errWireNumberRounding},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateLimitPrice(tc.price, tc.asset, tc.decimals)
			require.ErrorIs(t, err, tc.expectedIs, "Validating a limit price must return the expected error")
		})
	}
}

func TestRoundMarketPrice(t *testing.T) {
	for _, tc := range []struct {
		name       string
		price      float64
		asset      asset.Item
		decimals   uint64
		expected   float64
		expectedIs error
	}{
		{name: "spot", price: 123.456789, asset: asset.Spot, decimals: 2, expected: 123.46},
		{name: "perpetual", price: 123.456789, asset: asset.PerpetualContract, decimals: 5, expected: 123.5},
		{name: "zero", price: 0, asset: asset.Spot, expectedIs: errInvalidMarketPrice},
		{name: "nan", price: math.NaN(), asset: asset.Spot, expectedIs: errInvalidMarketPrice},
		{name: "infinity", price: math.Inf(1), asset: asset.Spot, expectedIs: errInvalidMarketPrice},
		{name: "unsupported asset", price: 1, asset: asset.Options, expectedIs: asset.ErrNotSupported},
		{name: "excess perpetual size precision", price: 1, asset: asset.PerpetualContract, decimals: 7, expectedIs: errSizePrecision},
		{name: "rounds to zero", price: 0.4, asset: asset.PerpetualContract, decimals: 6, expectedIs: errInvalidMarketPrice},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := roundMarketPrice(tc.price, tc.asset, tc.decimals)
			require.ErrorIs(t, err, tc.expectedIs, "Rounding a market price must return the expected error")
			assert.InDelta(t, tc.expected, result, 1e-12, "Rounded market price should match")
		})
	}
}

func TestFormatOrderTimeInForce(t *testing.T) {
	for _, tc := range []struct {
		timeInForce order.TimeInForce
		expected    string
		expectedIs  error
	}{
		{timeInForce: order.UnknownTIF, expected: "Gtc"},
		{timeInForce: order.GoodTillCancel, expected: "Gtc"},
		{timeInForce: order.PostOnly, expected: "Alo"},
		{timeInForce: order.GoodTillCancel | order.PostOnly, expected: "Alo"},
		{timeInForce: order.ImmediateOrCancel, expected: "Ioc"},
		{timeInForce: order.FillOrKill, expectedIs: errUnsupportedTimeInForce},
	} {
		result, err := formatOrderTimeInForce(tc.timeInForce)
		require.ErrorIs(t, err, tc.expectedIs, "Formatting time in force must return the expected error")
		assert.Equal(t, tc.expected, result, "Formatted time in force should match")
	}
}

func TestBuildOrderWire(t *testing.T) {
	ex := newTradingTestExchange(t, map[string]string{"allMids": `{"BTC":"100","@107":"10"}`}, nil)

	wire, mapping, err := ex.buildOrderWire(t.Context(), testPerpetualPair, asset.PerpetualContract, order.Limit, order.Buy, order.GoodTillCancel, 0.12345, 100.5, 0, 0, true, strings.ToUpper(validClientOrderID))
	require.NoError(t, err, "Building a valid limit order must not error")
	assert.Equal(t, uint64(5), mapping.sizeDecimals, "Built order should retain its market precision")
	assert.Equal(t, uint64(0), wire.AssetID, "Perpetual order should use its universe index")
	assert.True(t, wire.IsBuy, "Buy order should set the buy flag")
	assert.Equal(t, "100.5", wire.Price, "Limit price should be formatted")
	assert.Equal(t, "0.12345", wire.Size, "Limit size should be formatted")
	assert.True(t, wire.ReduceOnly, "Reduce-only flag should be retained")
	assert.Equal(t, "Gtc", wire.Type.Limit.TimeInForce, "Limit time in force should be formatted")
	assert.Equal(t, validClientOrderID, wire.ClientOrderID, "Client order ID should be normalised")

	wire, _, err = ex.buildOrderWire(t.Context(), testPerpetualPair, asset.PerpetualContract, order.Market, order.Buy, order.UnknownTIF, 0.1, 0, 0, 0.01, false, "")
	require.NoError(t, err, "Building a slippage-bounded market buy must not error")
	assert.Equal(t, "101", wire.Price, "Market buy should apply positive slippage to the midpoint")
	assert.Equal(t, "Ioc", wire.Type.Limit.TimeInForce, "Market order should use IOC")

	wire, _, err = ex.buildOrderWire(t.Context(), testPerpetualPair, asset.PerpetualContract, order.Market, order.Sell, order.UnknownTIF, 0.1, 0, 0, 0.01, false, "")
	require.NoError(t, err, "Building a slippage-bounded market sell must not error")
	assert.Equal(t, "99", wire.Price, "Market sell should apply negative slippage to the midpoint")
	assert.False(t, wire.IsBuy, "Sell order should clear the buy flag")

	wire, _, err = ex.buildOrderWire(t.Context(), testPerpetualPair, asset.PerpetualContract, order.StopMarket, order.Sell, order.UnknownTIF, 0.1, 0, 90, 0.1, true, "")
	require.NoError(t, err, "Building a stop-market order must not error")
	require.NotNil(t, wire.Type.Trigger, "Stop-market order must use the trigger wire variant")
	assert.True(t, wire.Type.Trigger.IsMarket, "Stop-market order should set the market flag")
	assert.Equal(t, "90", wire.Type.Trigger.TriggerPrice, "Stop-market trigger price should be formatted")
	assert.Equal(t, "sl", wire.Type.Trigger.TakeProfitStopLoss, "Stop-market order should use stop-loss semantics")
	assert.Equal(t, "81", wire.Price, "Stop-market sell should derive a slippage-bounded execution price")

	wire, _, err = ex.buildOrderWire(t.Context(), testPerpetualPair, asset.PerpetualContract, order.TakeProfit, order.Buy, order.UnknownTIF, 0.1, 111, 110, 0, true, "")
	require.NoError(t, err, "Building a take-profit limit order must not error")
	require.NotNil(t, wire.Type.Trigger, "Take-profit order must use the trigger wire variant")
	assert.False(t, wire.Type.Trigger.IsMarket, "Take-profit limit order should clear the market flag")
	assert.Equal(t, "tp", wire.Type.Trigger.TakeProfitStopLoss, "Take-profit order should use take-profit semantics")
	assert.Equal(t, "111", wire.Price, "Take-profit limit price should be formatted")

	for _, tc := range []struct {
		name         string
		pair         currency.Pair
		asset        asset.Item
		orderType    order.Type
		timeInForce  order.TimeInForce
		amount       float64
		price        float64
		triggerPrice float64
		slippage     float64
		reduceOnly   bool
		clientID     string
		expectedIs   error
	}{
		{name: "missing mapping", pair: currency.NewPair(currency.ETH, currency.USDC), asset: asset.PerpetualContract, orderType: order.Limit, amount: 1, price: 1, expectedIs: errPairMappingNotFound},
		{name: "invalid size", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.Limit, amount: 0.000001, price: 1, expectedIs: errSizePrecision},
		{name: "invalid client ID", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.Limit, amount: 1, price: 1, clientID: "invalid", expectedIs: errClientOrderIDInvalid},
		{name: "invalid limit precision", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.Limit, amount: 1, price: 100.55, expectedIs: errPricePrecision},
		{name: "invalid time in force", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.Limit, timeInForce: order.FillOrKill, amount: 1, price: 1, expectedIs: errUnsupportedTimeInForce},
		{name: "zero slippage", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.Market, amount: 1, slippage: 0, expectedIs: errSlippageTolerance},
		{name: "excess slippage", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.Market, amount: 1, slippage: 1, expectedIs: errSlippageTolerance},
		{name: "unsupported type", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.TrailingStop, amount: 1, price: 1, expectedIs: order.ErrTypeIsInvalid},
		{name: "invalid price", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.Limit, amount: 1, price: math.NaN(), expectedIs: errInvalidMarketPrice},
		{name: "limit with trigger price", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.Limit, amount: 1, price: 1, triggerPrice: 2, expectedIs: errRiskManagementUnsupported},
		{name: "market with trigger price", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.Market, amount: 1, triggerPrice: 2, slippage: 0.1, expectedIs: errRiskManagementUnsupported},
		{name: "spot trigger", pair: testSpotPair, asset: asset.Spot, orderType: order.StopMarket, amount: 1, triggerPrice: 10, slippage: 0.1, reduceOnly: true, expectedIs: errTriggerOrderReduceOnly},
		{name: "non-reduce-only trigger", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.StopMarket, amount: 1, triggerPrice: 10, slippage: 0.1, expectedIs: errTriggerOrderReduceOnly},
		{name: "missing trigger price", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.StopMarket, amount: 1, slippage: 0.1, reduceOnly: true, expectedIs: errTriggerPriceRequired},
		{name: "invalid trigger precision", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.StopMarket, amount: 1, triggerPrice: 100.55, slippage: 0.1, reduceOnly: true, expectedIs: errPricePrecision},
		{name: "trigger market missing slippage", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.StopMarket, amount: 1, triggerPrice: 100, reduceOnly: true, expectedIs: errSlippageTolerance},
		{name: "trigger market derived price overflow", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.StopMarket, amount: 1, triggerPrice: math.MaxFloat64, slippage: 0.9, reduceOnly: true, expectedIs: errInvalidMarketPrice},
		{name: "trigger limit missing price", pair: testPerpetualPair, asset: asset.PerpetualContract, orderType: order.StopLimit, amount: 1, triggerPrice: 100, reduceOnly: true, expectedIs: errInvalidMarketPrice},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ex.buildOrderWire(t.Context(), tc.pair, tc.asset, tc.orderType, order.Buy, tc.timeInForce, tc.amount, tc.price, tc.triggerPrice, tc.slippage, tc.reduceOnly, tc.clientID)
			require.ErrorIs(t, err, tc.expectedIs, "Building invalid order must return the expected error")
		})
	}

	missingMid := newTradingTestExchange(t, map[string]string{"allMids": `{}`}, nil)
	_, _, err = missingMid.buildOrderWire(t.Context(), testPerpetualPair, asset.PerpetualContract, order.Market, order.Buy, order.UnknownTIF, 1, 0, 0, 0.01, false, "")
	require.ErrorIs(t, err, errMarketMidPriceNotFound, "Missing market midpoint must return the expected error")

	zeroMid := newTradingTestExchange(t, map[string]string{"allMids": `{"BTC":"0"}`}, nil)
	_, _, err = zeroMid.buildOrderWire(t.Context(), testPerpetualPair, asset.PerpetualContract, order.Market, order.Buy, order.UnknownTIF, 1, 0, 0, 0.01, false, "")
	require.ErrorIs(t, err, errMarketMidPriceNotFound, "Zero market midpoint must return the expected error")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	errorExchange.setPairMappings(asset.PerpetualContract, []pairMapping{{pair: testPerpetualPair, coin: "BTC", sizeDecimals: 5}})
	_, _, err = errorExchange.buildOrderWire(t.Context(), testPerpetualPair, asset.PerpetualContract, order.Market, order.Buy, order.UnknownTIF, 1, 0, 0, 0.01, false, "")
	require.Error(t, err, "Market midpoint HTTP failure must be returned")

	invalidPricePrecision := newTradingTestExchange(t, map[string]string{"allMids": `{"BTC":"100"}`}, nil)
	invalidPricePrecision.setPairMappings(asset.PerpetualContract, []pairMapping{{pair: testPerpetualPair, coin: "BTC", sizeDecimals: 7}})
	_, _, err = invalidPricePrecision.buildOrderWire(t.Context(), testPerpetualPair, asset.PerpetualContract, order.Market, order.Buy, order.UnknownTIF, 1, 0, 0, 0.01, false, "")
	require.ErrorIs(t, err, errSizePrecision, "Market price precision incompatible with metadata must return the expected error")

	hip3Pair := currency.NewPair(currency.NewCode("xyz:XYZ100"), currency.USDC)
	var hip3Request infoRequest
	hip3 := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&hip3Request), "Decoding HIP-3 midpoint request should not error") {
			return
		}
		_, err := w.Write([]byte(`{"xyz:XYZ100":"50"}`))
		assert.NoError(t, err, "Writing HIP-3 midpoint response should not error")
	}))
	hip3.setPairMappings(asset.PerpetualContract, []pairMapping{{
		pair: hip3Pair, coin: "xyz:XYZ100", dex: testBuilderDEXName, assetID: 110000, sizeDecimals: 2,
	}})
	wire, _, err = hip3.buildOrderWire(t.Context(), hip3Pair, asset.PerpetualContract, order.Market, order.Buy, order.UnknownTIF, 1, 0, 0, 0.01, false, "")
	require.NoError(t, err, "Building a HIP-3 market order must not error")
	assert.Equal(t, testBuilderDEXName, hip3Request.DEX, "HIP-3 market midpoint request should use its DEX")
	assert.Equal(t, uint64(110000), wire.AssetID, "HIP-3 order should use its builder asset ID")
}

func TestBuildOrderWires(t *testing.T) {
	ex := newTradingTestExchange(t, nil, nil)
	submit := &order.Submit{
		Exchange:      "Hyperliquid",
		Type:          order.Limit,
		Side:          order.Buy,
		Pair:          testPerpetualPair,
		AssetType:     asset.PerpetualContract,
		TimeInForce:   order.GoodTillCancel,
		Amount:        0.1,
		Price:         100,
		ClientOrderID: validClientOrderID,
	}

	wires, mapping, grouping, err := ex.buildOrderWires(t.Context(), submit)
	require.NoError(t, err, "Building one ungrouped order must not error")
	require.Len(t, wires, 1, "Ungrouped submission must contain one wire")
	assert.Equal(t, uint64(0), mapping.assetID, "Ungrouped submission should return the parent mapping")
	assert.Equal(t, orderGroupingNone, grouping, "Ungrouped submission should use no grouping")

	standaloneTrigger := *submit
	standaloneTrigger.Type = order.StopMarket
	standaloneTrigger.Side = order.Sell
	standaloneTrigger.TimeInForce = order.UnknownTIF
	standaloneTrigger.Price = 0
	standaloneTrigger.TriggerPrice = 90
	standaloneTrigger.TriggerPriceType = order.LastPrice
	standaloneTrigger.SlippageTolerance = 0.1
	standaloneTrigger.ReduceOnly = true
	wires, mapping, _, err = ex.buildOrderWires(t.Context(), &standaloneTrigger)
	require.ErrorIs(t, err, errRiskManagementUnsupported, "Standalone trigger using last price must fail closed")
	assert.Empty(t, wires, "Rejected standalone trigger should not build a wire")
	assert.Equal(t, pairMapping{}, mapping, "Rejected standalone trigger should not return a mapping")
	standaloneTrigger.TriggerPriceType = order.MarkPrice
	wires, _, grouping, err = ex.buildOrderWires(t.Context(), &standaloneTrigger)
	require.NoError(t, err, "Standalone mark-price trigger must not error")
	require.Len(t, wires, 1, "Standalone trigger must contain one wire")
	assert.Equal(t, orderGroupingNone, grouping, "Standalone trigger should not use grouped semantics")

	grouped := *submit
	grouped.RiskManagementModes = order.RiskManagementModes{
		Mode: orderGroupingNormalTPSL,
		TakeProfit: order.RiskManagement{
			Enabled:          true,
			TriggerPriceType: order.MarkPrice,
			Price:            110,
			OrderType:        order.Market,
		},
		StopLoss: order.RiskManagement{
			Enabled:          true,
			TriggerPriceType: order.MarkPrice,
			Price:            90,
			LimitPrice:       89,
			OrderType:        order.StopLimit,
		},
	}
	wires, _, grouping, err = ex.buildOrderWires(t.Context(), &grouped)
	require.NoError(t, err, "Building grouped parent and TP/SL children must not error")
	require.Len(t, wires, 3, "Grouped submission must contain the parent and both children")
	assert.Equal(t, orderGroupingNormalTPSL, grouping, "Grouped submission should use normal TP/SL grouping")
	assert.Equal(t, validClientOrderID, wires[0].ClientOrderID, "Grouped parent should retain the client order ID")
	assert.Empty(t, wires[1].ClientOrderID, "Grouped take-profit child should not reuse the parent client order ID")
	assert.False(t, wires[1].IsBuy, "Long parent take-profit child should sell")
	assert.Equal(t, "99", wires[1].Price, "Market take-profit child should use Hyperliquid's default ten-percent bound")
	assert.Equal(t, "tp", wires[1].Type.Trigger.TakeProfitStopLoss, "Take-profit child should use TP semantics")
	assert.False(t, wires[2].Type.Trigger.IsMarket, "Stop-limit child should use a limit trigger")
	assert.Equal(t, "89", wires[2].Price, "Stop-limit child should retain its execution price")

	shortParent := grouped
	shortParent.Side = order.Sell
	shortParent.RiskManagementModes.TakeProfit.OrderType = order.TakeProfit
	shortParent.RiskManagementModes.TakeProfit.LimitPrice = 91
	shortParent.RiskManagementModes.StopLoss.OrderType = order.Stop
	shortParent.RiskManagementModes.StopLoss.LimitPrice = 0
	wires, _, _, err = ex.buildOrderWires(t.Context(), &shortParent)
	require.NoError(t, err, "Building explicitly typed short-parent children must not error")
	assert.True(t, wires[1].IsBuy, "Short parent take-profit child should buy")
	assert.False(t, wires[1].Type.Trigger.IsMarket, "Take-profit type should build a limit trigger")
	assert.True(t, wires[2].Type.Trigger.IsMarket, "Stop type should build a market trigger")
	assert.Equal(t, "99", wires[2].Price, "Short-parent stop child should derive an upward buy bound")

	for _, tc := range []struct {
		name       string
		mutate     func(*order.Submit)
		expectedIs error
	}{
		{
			name: "invalid parent",
			mutate: func(s *order.Submit) {
				s.Amount = 0.000001
			},
			expectedIs: errSizePrecision,
		},
		{
			name: "spot grouping",
			mutate: func(s *order.Submit) {
				s.Pair = testSpotPair
				s.AssetType = asset.Spot
				s.RiskManagementModes.TakeProfit.Enabled = true
				s.RiskManagementModes.TakeProfit.Price = 11
			},
			expectedIs: errRiskManagementUnsupported,
		},
		{
			name: "stop entry",
			mutate: func(s *order.Submit) {
				s.RiskManagementModes.StopEntry.Enabled = true
			},
			expectedIs: errRiskManagementUnsupported,
		},
		{
			name: "unsupported mode",
			mutate: func(s *order.Submit) {
				s.RiskManagementModes.Mode = "position"
				s.RiskManagementModes.TakeProfit.Enabled = true
			},
			expectedIs: errRiskManagementUnsupported,
		},
		{
			name: "missing child trigger",
			mutate: func(s *order.Submit) {
				s.RiskManagementModes.TakeProfit.Enabled = true
			},
			expectedIs: errTriggerPriceRequired,
		},
		{
			name: "unsupported trigger source",
			mutate: func(s *order.Submit) {
				s.RiskManagementModes.TakeProfit = order.RiskManagement{Enabled: true, Price: 110, TriggerPriceType: order.IndexPrice}
			},
			expectedIs: errRiskManagementUnsupported,
		},
		{
			name: "unsupported take-profit type",
			mutate: func(s *order.Submit) {
				s.RiskManagementModes.TakeProfit = order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 110, OrderType: order.Stop}
			},
			expectedIs: errRiskManagementUnsupported,
		},
		{
			name: "unsupported stop-loss type",
			mutate: func(s *order.Submit) {
				s.RiskManagementModes.StopLoss = order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 90, OrderType: order.TakeProfit}
			},
			expectedIs: errRiskManagementUnsupported,
		},
		{
			name: "invalid child execution price",
			mutate: func(s *order.Submit) {
				s.RiskManagementModes.TakeProfit = order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 110, LimitPrice: 100.55, OrderType: order.Limit}
			},
			expectedIs: errPricePrecision,
		},
		{
			name: "invalid child slippage",
			mutate: func(s *order.Submit) {
				s.SlippageTolerance = 1
				s.RiskManagementModes.TakeProfit = order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 110}
			},
			expectedIs: errSlippageTolerance,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalid := *submit
			tc.mutate(&invalid)
			_, _, _, err := ex.buildOrderWires(t.Context(), &invalid)
			require.ErrorIs(t, err, tc.expectedIs, "Building invalid grouped order must return the expected error")
		})
	}
}

func actionResponse(raw string) *exchangeActionResponse {
	return &exchangeActionResponse{Status: "ok", Response: json.RawMessage(raw)}
}

func TestParseOrderActionStatuses(t *testing.T) {
	_, err := parseOrderActionStatuses(nil, 1)
	require.ErrorIs(t, err, common.ErrNilPointer, "Nil order action response must return the expected error")

	_, err = parseOrderActionStatuses(actionResponse(`invalid`), 1)
	require.Error(t, err, "Invalid order action response must error")

	_, err = parseOrderActionStatuses(actionResponse(`{"data":{"statuses":[]}}`), 1)
	require.ErrorIs(t, err, errActionStatusCount, "Unexpected order status count must return the expected error")

	statuses, err := parseOrderActionStatuses(actionResponse(`{"data":{"statuses":[{"resting":{"oid":7}},{"filled":{"oid":8,"totalSz":"1","avgPx":"2"}},{"error":"bad"},"waitingForFill","waitingForTrigger"]}}`), 5)
	require.NoError(t, err, "Parsing valid order status variants must not error")
	require.Len(t, statuses, 5, "All order status variants must be returned")
	assert.Equal(t, uint64(7), statuses[0].Resting.OrderID, "Resting order ID should be decoded")
	assert.Equal(t, uint64(8), statuses[1].Filled.OrderID, "Filled order ID should be decoded")
	assert.Equal(t, "bad", statuses[2].Error, "Order error should be decoded")
	assert.Equal(t, orderStatusWaitingForFill, statuses[3].Deferred, "Waiting-for-fill status should be decoded")
	assert.Equal(t, orderStatusWaitingForTrigger, statuses[4].Deferred, "Waiting-for-trigger status should be decoded")

	statuses, err = parseOrderActionStatuses(actionResponse(`{"data":{"statuses":[{"error":"batch rejected"}]}}`), 3)
	require.NoError(t, err, "Parsing one deterministic batch error must not error")
	require.Len(t, statuses, 3, "One deterministic batch error must be expanded to every requested order")
	assert.Equal(t, "batch rejected", statuses[2].Error, "Expanded batch error should retain the exchange message")

	for _, raw := range []string{
		`{"data":{"statuses":[invalid]}}`,
		`{"data":{"statuses":[1]}}`,
		`{"data":{"statuses":["invalid"]}}`,
		`{"data":{"statuses":[""]}}`,
		`{"data":{"statuses":[{}]}}`,
		`{"data":{"statuses":[{"resting":{"oid":7},"error":"bad"}]}}`,
		`{"data":{"statuses":[{"resting":{"oid":0}}]}}`,
		`{"data":{"statuses":[{"filled":{"oid":0}}]}}`,
	} {
		_, err = parseOrderActionStatuses(actionResponse(raw), 1)
		require.Error(t, err, "Malformed order status must error")
	}
}

func TestParseCancelActionStatuses(t *testing.T) {
	_, err := parseCancelActionStatuses(nil, []string{"1"})
	require.ErrorIs(t, err, common.ErrNilPointer, "Nil cancel action response must return the expected error")
	_, err = parseCancelActionStatuses(actionResponse(`invalid`), []string{"1"})
	require.Error(t, err, "Invalid cancel action response must error")
	_, err = parseCancelActionStatuses(actionResponse(`{"data":{"statuses":[]}}`), []string{"1"})
	require.ErrorIs(t, err, errActionStatusCount, "Unexpected cancel status count must return the expected error")

	statuses, err := parseCancelActionStatuses(actionResponse(`{"data":{"statuses":["success",{"error":"already closed"}]}}`), []string{"1", validClientOrderID})
	require.Error(t, err, "Per-order cancellation failure must be returned")
	require.ErrorIs(t, err, errActionResponse, "Per-order cancellation failure must retain the action-response classification")
	assert.Equal(t, "success", statuses["1"], "Successful cancellation should be retained")
	assert.Equal(t, "already closed", statuses[validClientOrderID], "Failed cancellation message should be retained")

	_, err = parseCancelActionStatuses(actionResponse(`{"data":{"statuses":[invalid]}}`), []string{"1"})
	require.Error(t, err, "Invalid cancellation status must error")
	_, err = parseCancelActionStatuses(actionResponse(`{"data":{"statuses":[1]}}`), []string{"1"})
	require.Error(t, err, "Cancellation status with an invalid variant type must error")
	_, err = parseCancelActionStatuses(actionResponse(`{"data":{"statuses":[{}]}}`), []string{"1"})
	require.ErrorIs(t, err, errActionStatusMalformed, "Empty cancellation status must return the expected error")
}

func TestClassifyHyperliquidOrderStatus(t *testing.T) {
	for _, tc := range []struct {
		status     string
		expected   order.Status
		expectedIs error
	}{
		{status: "open", expected: order.Open},
		{status: "filled", expected: order.Filled},
		{status: "triggered", expected: order.Closed},
		{status: "canceled", expected: order.Cancelled},
		{status: "scheduledCancel", expected: order.Cancelled},
		{status: "marginCanceled", expected: order.Cancelled},
		{status: "liquidatedCanceled", expected: order.Cancelled},
		{status: "rejected", expected: order.Rejected},
		{status: "perpMaxPositionRejected", expected: order.Rejected},
		{status: "unknown", expected: order.UnknownStatus, expectedIs: errUnsupportedOrderStatus},
	} {
		result, err := classifyHyperliquidOrderStatus(tc.status)
		require.ErrorIs(t, err, tc.expectedIs, "Classifying order status must return the expected error")
		assert.Equal(t, tc.expected, result, "Classified order status should match")
	}
}

func TestClassifyHyperliquidOrderType(t *testing.T) {
	for _, tc := range []struct {
		orderType  string
		isTrigger  bool
		expected   order.Type
		expectedIs error
	}{
		{orderType: "Limit", expected: order.Limit},
		{orderType: "Market", expected: order.Market},
		{orderType: "Stop Limit", isTrigger: true, expected: order.StopLimit},
		{orderType: "Stop Market", isTrigger: true, expected: order.StopMarket},
		{orderType: "Stop", isTrigger: true, expected: order.Stop},
		{orderType: "Take Profit Limit", isTrigger: true, expected: order.TakeProfit},
		{orderType: "Take Profit Market", isTrigger: true, expected: order.TakeProfitMarket},
		{orderType: "unknown", expected: order.UnknownType, expectedIs: order.ErrTypeIsInvalid},
	} {
		result, err := classifyHyperliquidOrderType(tc.orderType, tc.isTrigger)
		require.ErrorIs(t, err, tc.expectedIs, "Classifying order type must return the expected error")
		assert.Equal(t, tc.expected, result, "Classified order type should match")
	}
}

func TestClassifyHyperliquidTimeInForce(t *testing.T) {
	for _, tc := range []struct {
		timeInForce string
		expected    order.TimeInForce
		expectedIs  error
	}{
		{timeInForce: "", expected: order.GoodTillCancel},
		{timeInForce: "GTC", expected: order.GoodTillCancel},
		{timeInForce: "Alo", expected: order.PostOnly},
		{timeInForce: "Ioc", expected: order.ImmediateOrCancel},
		{timeInForce: "FrontendMarket", expected: order.ImmediateOrCancel},
		{timeInForce: "bad", expected: order.UnknownTIF, expectedIs: order.ErrInvalidTimeInForce},
	} {
		result, err := classifyHyperliquidTimeInForce(tc.timeInForce)
		require.ErrorIs(t, err, tc.expectedIs, "Classifying time in force must return the expected error")
		assert.Equal(t, tc.expected, result, "Classified time in force should match")
	}
}

func TestConvertOrder(t *testing.T) {
	ex := newTradingTestExchange(t, nil, nil)
	_, err := ex.convertOrder(t.Context(), nil, "open", time.Time{})
	require.ErrorIs(t, err, common.ErrNilPointer, "Nil source order must return the expected error")
	source := mustOpenOrder(t, `{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"2","oid":7,"timestamp":1700000000000,"triggerPx":"0","isTrigger":false,"reduceOnly":true,"orderType":"Limit","tif":"Gtc","cloid":"`+validClientOrderID+`"}`)
	statusTime := time.UnixMilli(1700000001000)
	detail, err := ex.convertOrder(t.Context(), &source, "open", statusTime)
	require.NoError(t, err, "Converting a valid order must not error")
	assert.Equal(t, order.Buy, detail.Side, "Order side should be converted")
	assert.Equal(t, order.Limit, detail.Type, "Order type should be converted")
	assert.Equal(t, order.Open, detail.Status, "Order status should be converted")
	assert.Equal(t, 2.0, detail.Amount, "Original order size should be used")
	assert.Equal(t, 1.0, detail.ExecutedAmount, "Executed size should be derived")
	assert.Equal(t, validClientOrderID, detail.ClientOrderID, "Client order ID should be retained")
	assert.Equal(t, statusTime.UTC(), detail.LastUpdated, "Status timestamp should be used")
	assert.True(t, detail.ReduceOnly, "Reduce-only flag should be retained")

	source.Side = "A"
	source.OriginalSize = 0
	source.ClientOrderID = nil
	detail, err = ex.convertOrder(t.Context(), &source, "filled", time.Time{})
	require.NoError(t, err, "Converting an order with fallback fields must not error")
	assert.Equal(t, order.Sell, detail.Side, "Sell side should be converted")
	assert.Equal(t, 1.0, detail.Amount, "Remaining size should be used when original size is absent")
	assert.Empty(t, detail.ClientOrderID, "Missing client order ID should remain empty")
	assert.Equal(t, source.Timestamp.Time().UTC(), detail.LastUpdated, "Order timestamp should be the last-updated fallback")

	for _, tc := range []struct {
		name       string
		mutate     func(*OpenOrder)
		status     string
		expectedIs error
	}{
		{name: "missing mapping", mutate: func(o *OpenOrder) { o.Coin = "MISSING" }, status: "open", expectedIs: errPairMappingNotFound},
		{name: "invalid side", mutate: func(o *OpenOrder) { o.Side = "X" }, status: "open", expectedIs: order.ErrSideIsInvalid},
		{name: "invalid type", mutate: func(o *OpenOrder) { o.OrderType = "unknown" }, status: "open", expectedIs: order.ErrTypeIsInvalid},
		{name: "invalid time in force", mutate: func(o *OpenOrder) { o.TimeInForce = "bad" }, status: "open", expectedIs: order.ErrInvalidTimeInForce},
		{name: "invalid status", mutate: func(*OpenOrder) {}, status: "unknown", expectedIs: errUnsupportedOrderStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalid := source
			invalid.Side = "B"
			invalid.OrderType = "Limit"
			invalid.TimeInForce = "Gtc"
			invalid.Coin = "BTC"
			tc.mutate(&invalid)
			_, err := ex.convertOrder(t.Context(), &invalid, tc.status, time.Time{})
			require.ErrorIs(t, err, tc.expectedIs, "Converting invalid order must return the expected error")
		})
	}
}

func TestConvertOrderFromMapping(t *testing.T) {
	ex := newTradingTestExchange(t, nil, nil)
	mapping, a, err := ex.lookupPairMappingByCoin("BTC")
	require.NoError(t, err, "Getting the mapped-order fixture must not error")
	_, err = ex.convertOrderFromMapping(nil, "open", time.Time{}, &mapping, a)
	require.ErrorIs(t, err, common.ErrNilPointer, "Nil mapped source order must return the expected error")
	_, err = ex.convertOrderFromMapping(&OpenOrder{}, "open", time.Time{}, nil, a)
	require.ErrorIs(t, err, common.ErrNilPointer, "Nil pair mapping must return the expected error")

	source := mustOpenOrder(t, `{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"2","oid":7,"timestamp":1700000000000,"triggerPx":"0","isTrigger":false,"reduceOnly":true,"orderType":"Limit","tif":"Gtc","cloid":"`+validClientOrderID+`"}`)
	statusTime := time.UnixMilli(1700000001000)
	detail, err := ex.convertOrderFromMapping(&source, "open", statusTime, &mapping, a)
	require.NoError(t, err, "Converting a valid mapped order must not error")
	assert.Equal(t, order.Buy, detail.Side, "Mapped order side should be converted")
	assert.Equal(t, testPerpetualPair, detail.Pair, "Mapped order pair should be retained")
	assert.Equal(t, statusTime.UTC(), detail.LastUpdated, "Mapped order status timestamp should be used")

	source.Side = "A"
	source.OriginalSize = 0
	source.ClientOrderID = nil
	detail, err = ex.convertOrderFromMapping(&source, "filled", time.Time{}, &mapping, a)
	require.NoError(t, err, "Converting a mapped order with fallback fields must not error")
	assert.Equal(t, order.Sell, detail.Side, "Mapped sell side should be converted")
	assert.Equal(t, 1.0, detail.Amount, "Mapped order should use remaining size when original size is absent")
	assert.Empty(t, detail.ClientOrderID, "Mapped order without a client ID should remain empty")
	assert.Equal(t, source.Timestamp.Time().UTC(), detail.LastUpdated, "Mapped order timestamp should be the last-updated fallback")

	for _, tc := range []struct {
		name       string
		mutate     func(*OpenOrder)
		status     string
		expectedIs error
	}{
		{name: "invalid side", mutate: func(o *OpenOrder) { o.Side = "X" }, status: "open", expectedIs: order.ErrSideIsInvalid},
		{name: "invalid type", mutate: func(o *OpenOrder) { o.OrderType = "unknown" }, status: "open", expectedIs: order.ErrTypeIsInvalid},
		{name: "invalid time in force", mutate: func(o *OpenOrder) { o.TimeInForce = "bad" }, status: "open", expectedIs: order.ErrInvalidTimeInForce},
		{name: "invalid status", mutate: func(*OpenOrder) {}, status: "unknown", expectedIs: errUnsupportedOrderStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalid := source
			invalid.Side = "B"
			invalid.OrderType = "Limit"
			invalid.TimeInForce = "Gtc"
			tc.mutate(&invalid)
			_, err := ex.convertOrderFromMapping(&invalid, tc.status, time.Time{}, &mapping, a)
			require.ErrorIs(t, err, tc.expectedIs, "Converting an invalid mapped order must return the expected error")
		})
	}
}

func TestExchangeActionEndpointLimit(t *testing.T) {
	for _, tc := range []struct {
		batchLength int
		offset      request.EndpointLimit
	}{
		{batchLength: -1, offset: 0},
		{batchLength: 0, offset: 0},
		{batchLength: 1, offset: 0},
		{batchLength: 39, offset: 0},
		{batchLength: 40, offset: 1},
		{batchLength: 79, offset: 1},
		{batchLength: 80, offset: 2},
		{batchLength: maximumActionBatchSize + 1, offset: maximumActionBatchSize / actionBatchWeightSize},
	} {
		assert.Equal(t, exchangeActionEPLBase+tc.offset, exchangeActionEndpointLimit(tc.batchLength), "Action endpoint limit should match its batch weight")
	}
}

func TestSubmitOrder(t *testing.T) {
	_, err := new(Exchange).SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "Nil order submission must return the expected error")

	submit := &order.Submit{
		Exchange:      "Hyperliquid",
		Type:          order.Limit,
		Side:          order.Buy,
		Pair:          testPerpetualPair,
		AssetType:     asset.PerpetualContract,
		TimeInForce:   order.GoodTillCancel,
		Amount:        0.1,
		Price:         100,
		ClientOrderID: validClientOrderID,
	}
	missingSigner := newTradingTestExchange(t, nil, nil)
	setTestCredentials(missingSigner, &accounts.Credentials{Key: officialSigningAddress})
	_, err = missingSigner.SubmitOrder(t.Context(), submit)
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "Missing signing key must fail before constructing an action")

	invalidBuild := newTradingTestExchange(t, nil, nil)
	invalid := *submit
	invalid.Amount = 0.000001
	_, err = invalidBuild.SubmitOrder(t.Context(), &invalid)
	require.ErrorIs(t, err, errSizePrecision, "Order wire construction failure must be returned")
	trigger := *submit
	trigger.TriggerPrice = 90
	_, err = invalidBuild.SubmitOrder(t.Context(), &trigger)
	require.ErrorIs(t, err, errRiskManagementUnsupported, "Trigger price on a non-trigger order must fail closed")

	resting := newTradingTestExchange(t, nil, func(actionType string, _ map[string]any) string {
		assert.Equal(t, "order", actionType, "Submit should send an order action")
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"resting":{"oid":7}}]}}}`
	})
	result, err := resting.submitOrder(t.Context(), submit)
	require.NoError(t, err, "Submitting a resting limit order must not error")
	assert.Equal(t, "7", result.OrderID, "Submitted order ID should be returned")
	assert.Equal(t, order.New, result.Status, "Resting order should be new")
	assert.Equal(t, 100.0, result.Price, "Submitted wire price should be returned")
	assert.Equal(t, submit.Amount, result.RemainingAmount, "A resting order should retain its full open quantity")

	partiallyFilledGTC := newTradingTestExchange(t, nil, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"filled":{"oid":7,"totalSz":"0.04","avgPx":"100"}}]}}}`
	})
	result, err = partiallyFilledGTC.SubmitOrder(t.Context(), submit)
	require.NoError(t, err, "Submitting a partially filled GTC order must not error")
	assert.Equal(t, order.PartiallyFilled, result.Status, "Partial GTC submission should remain active")
	assert.InDelta(t, 0.06, result.RemainingAmount, 1e-12, "Partial GTC submission should return the open remainder")

	postOnly := *submit
	postOnly.TimeInForce = order.PostOnly
	partiallyFilledPostOnly := newTradingTestExchange(t, nil, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"filled":{"oid":7,"totalSz":"0.04","avgPx":"100"}}]}}}`
	})
	_, err = partiallyFilledPostOnly.SubmitOrder(t.Context(), &postOnly)
	require.ErrorIs(t, err, errActionStatusMalformed, "A partial post-only fill must fail closed")

	stopMarket := *submit
	stopMarket.Type = order.StopMarket
	stopMarket.Side = order.Sell
	stopMarket.Price = 80
	stopMarket.TriggerPrice = 90
	stopMarket.TriggerPriceType = order.MarkPrice
	stopMarket.ReduceOnly = true
	stopMarket.TimeInForce = order.UnknownTIF
	triggerResting := newTradingTestExchange(t, nil, func(actionType string, action map[string]any) string {
		assert.Equal(t, "order", actionType, "Trigger submit should send an order action")
		assert.Equal(t, orderGroupingNone, action["grouping"], "Standalone trigger should not use grouped semantics")
		orders, _ := action["orders"].([]any)
		require.Len(t, orders, 1, "Standalone trigger action must contain one order")
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"resting":{"oid":9}}]}}}`
	})
	result, err = triggerResting.SubmitOrder(t.Context(), &stopMarket)
	require.NoError(t, err, "Submitting a resting stop-market order must not error")
	assert.Equal(t, "9", result.OrderID, "Submitted trigger order ID should be returned")
	assert.Equal(t, order.StopMarket, result.Type, "Submitted trigger type should be retained")
	assert.Equal(t, 90.0, result.TriggerPrice, "Submitted trigger price should be retained")
	assert.Equal(t, order.UnknownTIF, result.TimeInForce, "Trigger submission should not report a limit time in force")

	partiallyFilledTrigger := newTradingTestExchange(t, nil, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"filled":{"oid":9,"totalSz":"0.04","avgPx":"90"}}]}}}`
	})
	_, err = partiallyFilledTrigger.SubmitOrder(t.Context(), &stopMarket)
	require.ErrorIs(t, err, errActionStatusMalformed, "A partial trigger fill must fail closed")

	bracket := *submit
	bracket.RiskManagementModes = order.RiskManagementModes{
		TakeProfit: order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 110},
		StopLoss:   order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 90},
	}
	grouped := newTradingTestExchange(t, nil, func(actionType string, action map[string]any) string {
		assert.Equal(t, "order", actionType, "Bracket submit should send an order action")
		assert.Equal(t, orderGroupingNormalTPSL, action["grouping"], "Bracket submit should use normal TP/SL grouping")
		orders, _ := action["orders"].([]any)
		require.Len(t, orders, 3, "Bracket action must contain parent, take-profit, and stop-loss orders")
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"resting":{"oid":10}},"waitingForFill","waitingForFill"]}}}`
	})
	result, err = grouped.SubmitOrder(t.Context(), &bracket)
	require.NoError(t, err, "Submitting a bracket order with deferred children must not error")
	assert.Equal(t, "10", result.OrderID, "Bracket submission should return the parent order ID")
	assert.NoError(t, result.SubmissionError, "Accepted bracket children should not set a submission error")

	groupedChildFailure := newTradingTestExchange(t, nil, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"resting":{"oid":11}},{"error":"bad TP"},"waitingForFill"]}}}`
	})
	result, err = groupedChildFailure.SubmitOrder(t.Context(), &bracket)
	require.NoError(t, err, "A placed parent with a rejected child must return the parent without encouraging a duplicate retry")
	require.ErrorIs(t, result.SubmissionError, errGroupedOrderChildFailure, "Rejected grouped child must be retained on the parent response")
	assert.Equal(t, "11", result.OrderID, "Partial grouped response should retain the placed parent order ID")

	groupedParentFailure := newTradingTestExchange(t, nil, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"error":"batch rejected"}]}}}`
	})
	_, err = groupedParentFailure.SubmitOrder(t.Context(), &bracket)
	require.ErrorIs(t, err, order.ErrUnableToPlaceOrder, "Deterministic grouped action rejection must fail the parent submission")

	deferredParent := newTradingTestExchange(t, nil, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":["waitingForFill","waitingForFill","waitingForFill"]}}}`
	})
	_, err = deferredParent.SubmitOrder(t.Context(), &bracket)
	require.ErrorIs(t, err, errActionStatusMalformed, "Deferred grouped parent status must fail closed")

	market := *submit
	market.Type = order.Market
	market.Price = 0
	market.SlippageTolerance = 0.01
	filled := newTradingTestExchange(t, map[string]string{"allMids": `{"BTC":"100"}`}, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"filled":{"oid":8,"totalSz":"0.04","avgPx":"101"}}]}}}`
	})
	result, err = filled.SubmitOrder(t.Context(), &market)
	require.NoError(t, err, "Submitting a filled market order must not error")
	assert.Equal(t, order.PartiallyFilledCancelled, result.Status, "Partial IOC execution should be marked partially filled and cancelled")
	assert.Equal(t, 101.0, result.AverageExecutedPrice, "Average execution price should be returned")
	assert.InDelta(t, 0.06, result.RemainingAmount, 1e-12, "Remaining amount should be derived")
	assert.Equal(t, order.ImmediateOrCancel, result.TimeInForce, "Market submission should report its wire time in force")

	overfilled := newTradingTestExchange(t, map[string]string{"allMids": `{"BTC":"100"}`}, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"filled":{"oid":8,"totalSz":"2","avgPx":"101"}}]}}}`
	})
	_, err = overfilled.SubmitOrder(t.Context(), &market)
	require.ErrorIs(t, err, errInvalidFilledSize, "Over-reported fill must fail closed")

	invalidFill := newTradingTestExchange(t, map[string]string{"allMids": `{"BTC":"100"}`}, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"filled":{"oid":8,"totalSz":"0.0400001","avgPx":"101"}}]}}}`
	})
	_, err = invalidFill.SubmitOrder(t.Context(), &market)
	require.ErrorIs(t, err, errInvalidFilledSize, "Invalid reported fill precision must return the expected error")

	rejected := newTradingTestExchange(t, nil, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"error":"bad order"}]}}}`
	})
	_, err = rejected.SubmitOrder(t.Context(), submit)
	require.ErrorIs(t, err, order.ErrUnableToPlaceOrder, "Rejected order must return the expected error")

	malformed := newTradingTestExchange(t, nil, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[]}}}`
	})
	_, err = malformed.SubmitOrder(t.Context(), submit)
	require.ErrorIs(t, err, errActionStatusCount, "Malformed order action response must return the expected error")

	failed := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestCredentials(failed, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey})
	failed.setPairMappings(asset.PerpetualContract, []pairMapping{{pair: testPerpetualPair, coin: "BTC", sizeDecimals: 5}})
	_, err = failed.SubmitOrder(t.Context(), submit)
	require.Error(t, err, "Signed order HTTP failure must be returned")
}

func TestCancelOrders(t *testing.T) {
	ex := newTradingTestExchange(t, nil, func(actionType string, action map[string]any) string {
		cancels, _ := action["cancels"].([]any)
		statuses := make([]string, len(cancels))
		for i := range statuses {
			statuses[i] = `"success"`
		}
		return `{"status":"ok","response":{"type":"` + actionType + `","data":{"statuses":[` + strings.Join(statuses, ",") + `]}}}`
	})

	statuses, err := ex.cancelOrders(t.Context(), nil)
	require.NoError(t, err, "Empty cancellation batch must not error")
	assert.Empty(t, statuses, "Empty cancellation batch should return empty status")

	_, err = ex.cancelOrders(t.Context(), make([]order.Cancel, maximumActionBatchSize+1))
	require.ErrorIs(t, err, errActionBatchTooLarge, "Oversized cancellation batch must return the expected error")

	for _, tc := range []struct {
		name       string
		cancel     order.Cancel
		expectedIs error
	}{
		{name: "missing pair", cancel: order.Cancel{OrderID: "1", AssetType: asset.PerpetualContract}, expectedIs: order.ErrPairIsEmpty},
		{name: "missing asset", cancel: order.Cancel{OrderID: "1", Pair: testPerpetualPair}, expectedIs: order.ErrAssetNotSet},
		{name: "missing mapping", cancel: order.Cancel{OrderID: "1", Pair: currency.NewPair(currency.ETH, currency.USDC), AssetType: asset.PerpetualContract}, expectedIs: errPairMappingNotFound},
		{name: "invalid numeric ID", cancel: order.Cancel{OrderID: "bad", Pair: testPerpetualPair, AssetType: asset.PerpetualContract}, expectedIs: order.ErrOrderIDNotSet},
		{name: "zero numeric ID", cancel: order.Cancel{OrderID: "0", Pair: testPerpetualPair, AssetType: asset.PerpetualContract}, expectedIs: order.ErrOrderIDNotSet},
		{name: "invalid client ID", cancel: order.Cancel{ClientOrderID: "invalid", Pair: testPerpetualPair, AssetType: asset.PerpetualContract}, expectedIs: errClientOrderIDInvalid},
		{name: "missing identifier", cancel: order.Cancel{Pair: testPerpetualPair, AssetType: asset.PerpetualContract}, expectedIs: order.ErrOrderIDNotSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ex.cancelOrders(t.Context(), []order.Cancel{tc.cancel})
			require.ErrorIs(t, err, tc.expectedIs, "Invalid cancellation must return the expected error")
		})
	}

	uppercaseClientOrderID := strings.ToUpper(validClientOrderID)
	statuses, err = ex.cancelOrders(t.Context(), []order.Cancel{
		{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract},
		{ClientOrderID: uppercaseClientOrderID, Pair: testSpotPair, AssetType: asset.Spot},
	})
	require.NoError(t, err, "Mixed numeric and client-ID cancellation must not error")
	assert.Equal(t, "success", statuses["7"], "Numeric cancellation status should be returned")
	assert.Equal(t, "success", statuses[uppercaseClientOrderID], "Client-ID cancellation status should retain the caller's identifier")
	assert.NotContains(t, statuses, validClientOrderID, "Client-ID cancellation status should not silently normalise the caller's key")

	require.ErrorIs(t, ex.CancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "Nil single cancellation must return the expected error")
	require.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract}), "Valid single cancellation must not error")

	batch, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract}})
	require.NoError(t, err, "Valid batch cancellation must not error")
	assert.Equal(t, "success", batch.Status["7"], "Batch cancellation status should be returned")

	failed := newTradingTestExchange(t, nil, func(string, map[string]any) string {
		return `{"status":"err","response":"cancel failed"}`
	})
	statuses, err = failed.cancelOrders(t.Context(), []order.Cancel{
		{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract},
		{ClientOrderID: validClientOrderID, Pair: testPerpetualPair, AssetType: asset.PerpetualContract},
	})
	require.ErrorIs(t, err, errActionResponse, "Action failures from cancellation groups must be returned")
	assert.Empty(t, statuses, "Failed cancellation groups should not report success")

	malformed := newTradingTestExchange(t, nil, func(string, map[string]any) string {
		return `{"status":"ok","response":{"data":{"statuses":[]}}}`
	})
	_, err = malformed.cancelOrders(t.Context(), []order.Cancel{{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract}})
	require.ErrorIs(t, err, errActionStatusCount, "Malformed cancellation response must return the expected error")
}

func TestSetLeverage(t *testing.T) {
	ex := newTradingTestExchange(t, nil, func(actionType string, action map[string]any) string {
		assert.Equal(t, "updateLeverage", actionType, "Leverage change should use the expected action")
		assert.Equal(t, float64(0), action["asset"], "Leverage change should use the perpetual universe index")
		assert.Contains(t, []any{true, false}, action["isCross"], "Leverage change should include the margin mode")
		assert.Contains(t, []any{float64(10), float64(20), float64(30)}, action["leverage"], "Leverage change should include the requested amount")
		return `{"status":"ok","response":{"type":"default"}}`
	})
	require.NoError(t, ex.SetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Unset, 10, order.UnknownSide), "Unset margin must use cross margin")
	require.NoError(t, ex.SetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Multi, 20, order.UnknownSide), "Multi margin must use cross margin")
	require.NoError(t, ex.SetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Isolated, 30, order.UnknownSide), "Isolated margin must use isolated margin")

	for _, tc := range []struct {
		name       string
		asset      asset.Item
		pair       currency.Pair
		marginType margin.Type
		amount     float64
		expectedIs error
	}{
		{name: "unsupported asset", asset: asset.Spot, pair: testSpotPair, marginType: margin.Multi, amount: 1, expectedIs: asset.ErrNotSupported},
		{name: "missing pair", asset: asset.PerpetualContract, pair: currency.NewPair(currency.ETH, currency.USDC), marginType: margin.Multi, amount: 1, expectedIs: errPairMappingNotFound},
		{name: "zero", asset: asset.PerpetualContract, pair: testPerpetualPair, marginType: margin.Multi, amount: 0, expectedIs: errInvalidLeverage},
		{name: "negative", asset: asset.PerpetualContract, pair: testPerpetualPair, marginType: margin.Multi, amount: -1, expectedIs: errInvalidLeverage},
		{name: "fraction", asset: asset.PerpetualContract, pair: testPerpetualPair, marginType: margin.Multi, amount: 1.5, expectedIs: errInvalidLeverage},
		{name: "nan", asset: asset.PerpetualContract, pair: testPerpetualPair, marginType: margin.Multi, amount: math.NaN(), expectedIs: errInvalidLeverage},
		{name: "infinity", asset: asset.PerpetualContract, pair: testPerpetualPair, marginType: margin.Multi, amount: math.Inf(1), expectedIs: errInvalidLeverage},
		{name: "above maximum", asset: asset.PerpetualContract, pair: testPerpetualPair, marginType: margin.Multi, amount: 41, expectedIs: errInvalidLeverage},
		{name: "unsupported margin", asset: asset.PerpetualContract, pair: testPerpetualPair, marginType: margin.NoMargin, amount: 1, expectedIs: margin.ErrMarginTypeUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ex.SetLeverage(t.Context(), tc.asset, tc.pair, tc.marginType, tc.amount, order.UnknownSide)
			require.ErrorIs(t, err, tc.expectedIs, "Setting invalid leverage must return the expected error")
		})
	}

	missingMaximum := newTradingTestExchange(t, nil, nil)
	missingMaximum.setPairMappings(asset.PerpetualContract, []pairMapping{{pair: testPerpetualPair, coin: "BTC"}})
	require.ErrorIs(t, missingMaximum.SetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Multi, 1, order.UnknownSide), errInvalidLeverage, "Missing leverage metadata must fail closed")

	isolatedOnly := newTradingTestExchange(t, nil, func(actionType string, action map[string]any) string {
		assert.Equal(t, "updateLeverage", actionType, "Isolated leverage should use the expected action")
		assert.Equal(t, false, action["isCross"], "An isolated-only market should use isolated margin")
		return `{"status":"ok","response":{"type":"default"}}`
	})
	isolatedOnly.setPairMappings(asset.PerpetualContract, []pairMapping{{
		pair:         testPerpetualPair,
		coin:         "BTC",
		maxLeverage:  40,
		onlyIsolated: true,
	}})
	err := isolatedOnly.SetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Multi, 1, order.UnknownSide)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "Cross leverage on an isolated-only market must return the expected margin error")
	require.ErrorIs(t, err, errCrossMarginUnavailable, "Cross leverage on an isolated-only market must return the specific restriction")
	require.NoError(t, isolatedOnly.SetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Isolated, 1, order.UnknownSide), "Isolated leverage on an isolated-only market must not error")

	hip3Pair := currency.NewPair(currency.NewCode("xyz:XYZ100"), currency.USDC)
	hip3 := newTradingTestExchange(t, nil, func(actionType string, action map[string]any) string {
		assert.Equal(t, "updateLeverage", actionType, "HIP-3 leverage should use the expected action")
		assert.Equal(t, float64(110000), action["asset"], "HIP-3 leverage should use the builder asset ID")
		return `{"status":"ok","response":{"type":"default"}}`
	})
	hip3.setPairMappings(asset.PerpetualContract, []pairMapping{{
		pair: hip3Pair, coin: "xyz:XYZ100", dex: testBuilderDEXName, assetID: 110000, maxLeverage: 20,
	}})
	require.NoError(t, hip3.SetLeverage(t.Context(), asset.PerpetualContract, hip3Pair, margin.Multi, 10, order.UnknownSide), "Setting HIP-3 leverage must not error")

	failed := newTradingTestExchange(t, nil, func(string, map[string]any) string {
		return `{"status":"err","response":"leverage update failed"}`
	})
	require.ErrorIs(t, failed.SetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Multi, 1, order.UnknownSide), errActionResponse, "Exchange leverage failure must be returned")
}

func TestUpdateAccountBalances(t *testing.T) {
	ex := newTradingTestExchange(t, map[string]string{
		"spotClearinghouseState": `{"balances":[{"coin":"USDC","total":"10","hold":"2"}]}`,
		"clearinghouseState":     `{"marginSummary":{"accountValue":"20","totalMarginUsed":"3"},"withdrawable":"16"}`,
		"userAbstraction":        `"default"`,
	}, nil)

	spotAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "Updating spot balances must not error")
	require.Len(t, spotAccounts, 1, "Spot balances must return one subaccount")
	spotBalance := spotAccounts[0].Balances[currency.USDC]
	assert.Equal(t, 10.0, spotBalance.Total, "Spot total should be decoded")
	assert.Equal(t, 2.0, spotBalance.Hold, "Spot hold should be decoded")
	assert.Equal(t, 8.0, spotBalance.Free, "Spot free balance should be derived")

	perpetualAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "Updating perpetual balances must not error")
	require.Len(t, perpetualAccounts, 1, "Perpetual balances must return one subaccount")
	perpetualBalance := perpetualAccounts[0].Balances[currency.USDC]
	assert.Equal(t, 20.0, perpetualBalance.Total, "Perpetual total should be decoded")
	assert.Equal(t, 3.0, perpetualBalance.Hold, "Perpetual hold should be decoded")
	assert.Equal(t, 16.0, perpetualBalance.Free, "Perpetual withdrawable balance should be used")
	defaultBalances, err := ex.Accounts.CurrencyBalances(nil, asset.All)
	require.NoError(t, err, "Aggregating separate default account balances must not error")
	assert.Equal(t, 30.0, defaultBalances[currency.USDC].Total, "Separate spot and perpetual USDC pools should both be counted")

	for _, tc := range []struct {
		name string
		mode AccountAbstraction
	}{
		{name: "unified account", mode: AccountAbstractionUnified},
		{name: "portfolio margin", mode: AccountAbstractionPortfolio},
	} {
		t.Run(tc.name, func(t *testing.T) {
			unified := newTradingTestExchange(t, map[string]string{
				"spotClearinghouseState": `{"balances":[{"coin":"USDC","total":"30","hold":"4"},{"coin":"HYPE","total":"5","hold":"1"}]}`,
				"userAbstraction":        `"` + string(tc.mode) + `"`,
				infoTypePerpetualDEXs:    `[null,{"name":"` + testBuilderDEXName + `"}]`,
			}, nil)
			staleDefault := accounts.NewSubAccount(asset.PerpetualContract, officialSigningAddress)
			staleDefault.Balances.Set(currency.USDC, accounts.Balance{Total: 30})
			staleHIP3 := accounts.NewSubAccount(asset.PerpetualContract, officialSigningAddress+":"+testBuilderDEXName)
			staleHIP3.Balances.Set(currency.NewCode("HYPE"), accounts.Balance{Total: 5})
			require.NoError(t, unified.Accounts.Save(t.Context(), accounts.SubAccounts{staleDefault, staleHIP3}, true), "Saving stale separate perpetual balances must not error")
			spotResult, err := unified.UpdateAccountBalances(t.Context(), asset.Spot)
			require.NoError(t, err, "Updating unified spot balances must not error")
			require.Len(t, spotResult, 1, "Unified spot balances must return one subaccount")
			result, err := unified.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
			require.NoError(t, err, "Updating unified perpetual balances must not error")
			require.Len(t, result, 2, "Unified perpetual refresh must clear every registered DEX subaccount")
			assert.Equal(t, officialSigningAddress, result[0].ID, "Unified default DEX cleanup should use the account address")
			assert.Empty(t, result[0].Balances, "Unified default DEX balances should be cleared")
			assert.Equal(t, officialSigningAddress+":"+testBuilderDEXName, result[1].ID, "Unified HIP-3 cleanup should use the scoped account ID")
			assert.Empty(t, result[1].Balances, "Unified HIP-3 balances should be cleared")
			balances, err := unified.Accounts.CurrencyBalances(nil, asset.All)
			require.NoError(t, err, "Aggregating unified balances must not error")
			assert.Equal(t, 30.0, balances[currency.USDC].Total, "Unified USDC should be counted once")
			assert.Equal(t, 5.0, balances[currency.NewCode("HYPE")].Total, "Unified collateral should be counted once")
		})
	}

	hip3 := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding HIP-3 balance request should not error") {
			return
		}
		var response string
		switch request.Type {
		case "userAbstraction":
			response = `"disabled"`
		case infoTypePerpetualDEXs:
			response = `[null,{"name":"` + testBuilderDEXName + `"}]`
		case "spotMeta":
			response = `{"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150}]}`
		case infoTypeMetadata:
			if request.DEX == testBuilderDEXName {
				response = `{"collateralToken":150}`
			} else {
				response = `{"collateralToken":0}`
			}
		case "clearinghouseState":
			if request.DEX == testBuilderDEXName {
				response = `{"marginSummary":{"accountValue":"7","totalMarginUsed":"2"},"withdrawable":"4"}`
			} else {
				response = `{"marginSummary":{"accountValue":"20","totalMarginUsed":"3"},"withdrawable":"16"}`
			}
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing HIP-3 balance response should not error")
	}))
	setTestCredentials(hip3, &accounts.Credentials{Key: officialSigningAddress})
	hip3Accounts, err := hip3.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "Updating cold standard HIP-3 balances must not error")
	require.Len(t, hip3Accounts, 2, "Standard mode must preserve separate balances even without active market mappings")
	assert.Equal(t, officialSigningAddress, hip3Accounts[0].ID, "Default DEX should use the account address")
	assert.Equal(t, 20.0, hip3Accounts[0].Balances[currency.USDC].Total, "Default DEX collateral should be retained")
	assert.Equal(t, officialSigningAddress+":xyz", hip3Accounts[1].ID, "HIP-3 DEX should have a scoped subaccount ID")
	assert.Equal(t, 7.0, hip3Accounts[1].Balances[currency.NewCode("HYPE")].Total, "HIP-3 collateral token should be retained")

	_, err = ex.UpdateAccountBalances(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Unsupported balance asset must return the expected error")

	missingCredentials := new(Exchange)
	missingCredentials.SetDefaults()
	_, err = missingCredentials.UpdateAccountBalances(t.Context(), asset.Spot)
	require.Error(t, err, "Updating balances without credentials must error")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestCredentials(errorExchange, &accounts.Credentials{Key: officialSigningAddress})
	_, err = errorExchange.UpdateAccountBalances(t.Context(), asset.Spot)
	require.Error(t, err, "Spot balance HTTP failure must be returned")
	_, err = errorExchange.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
	require.Error(t, err, "Perpetual balance HTTP failure must be returned")

	unifiedStateFailure := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding unified balance failure request should not error") {
			return
		}
		switch request.Type {
		case "userAbstraction":
			_, writeErr := w.Write([]byte(`"unifiedAccount"`))
			assert.NoError(t, writeErr, "Writing abstraction mode should not error")
			return
		case infoTypePerpetualDEXs:
			_, writeErr := w.Write([]byte(`[null]`))
			assert.NoError(t, writeErr, "Writing perpetual DEX registry should not error")
			return
		}
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestCredentials(unifiedStateFailure, &accounts.Credentials{Key: officialSigningAddress})
	result, err := unifiedStateFailure.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "Unified perpetual refresh must not query a second balance source")
	require.Len(t, result, 1, "Unified perpetual refresh must return one clearing subaccount")
	assert.Empty(t, result[0].Balances, "Unified perpetual refresh should clear the separate balance bucket")

	for _, tc := range []struct {
		name      string
		responses map[string]string
		expected  error
	}{
		{
			name: "spot metadata failure",
			responses: map[string]string{
				"userAbstraction": `"default"`,
				"spotMeta":        `{`,
			},
		},
		{
			name: "perpetual DEX registry failure",
			responses: map[string]string{
				"userAbstraction":     `"default"`,
				infoTypePerpetualDEXs: `{`,
			},
		},
		{
			name:     "duplicate collateral token",
			expected: errUnexpectedResponseLength,
			responses: map[string]string{
				"userAbstraction": `"default"`,
				"spotMeta":        `{"tokens":[{"name":"USDC","index":0},{"name":"USDC2","index":0}]}`,
			},
		},
		{
			name: "metadata failure",
			responses: map[string]string{
				"userAbstraction": `"default"`,
				"spotMeta":        spotMetadataJSON,
				infoTypeMetadata:  `{`,
			},
		},
		{
			name:     "missing collateral token",
			expected: errSpotTokenNotFound,
			responses: map[string]string{
				"userAbstraction": `"default"`,
				"spotMeta":        `{"tokens":[{"name":"USDC","index":0}]}`,
				infoTypeMetadata:  `{"collateralToken":150}`,
			},
		},
		{
			name:     "blank collateral token",
			expected: errSpotTokenNotFound,
			responses: map[string]string{
				"userAbstraction": `"default"`,
				"spotMeta":        `{"tokens":[{"name":" ","index":0}]}`,
				infoTypeMetadata:  `{"collateralToken":0}`,
			},
		},
		{
			name: "clearinghouse failure",
			responses: map[string]string{
				"userAbstraction":    `"default"`,
				"spotMeta":           spotMetadataJSON,
				"clearinghouseState": `{`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			failed := newTradingTestExchange(t, tc.responses, nil)
			_, err := failed.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
			require.Error(t, err, "Invalid account balance response must return an error")
			if tc.expected != nil {
				require.ErrorIs(t, err, tc.expected, "Invalid account balance response must return the expected error")
			}
		})
	}
}

func TestModifyOrder(t *testing.T) {
	openOrderResponse := `{"status":"order","order":{"order":{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"2","oid":7,"timestamp":1700000000000,"isTrigger":false,"reduceOnly":true,"orderType":"Limit","tif":"Gtc","cloid":"` + validClientOrderID + `"},"status":"open","statusTimestamp":1700000001000}}`
	ex := newTradingTestExchange(t, map[string]string{"orderStatus": openOrderResponse}, func(actionType string, action map[string]any) string {
		assert.Equal(t, "batchModify", actionType, "Modification should send a batchModify action")
		assert.NotEmpty(t, action["modifies"], "Modification action should include one order")
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"resting":{"oid":7}}]}}}`
	})

	_, err := ex.ModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil, "Nil modification must return the expected error")

	missingSigner := newTradingTestExchange(t, nil, nil)
	setTestCredentials(missingSigner, &accounts.Credentials{Key: officialSigningAddress})
	_, err = missingSigner.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract})
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "Missing signing key must fail before querying the existing order")

	_, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "bad", Pair: testPerpetualPair, AssetType: asset.PerpetualContract})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "Non-numeric order ID must return the expected error")
	_, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "0", Pair: testPerpetualPair, AssetType: asset.PerpetualContract})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "Zero order ID must return the expected error")
	_, err = ex.ModifyOrder(t.Context(), &order.Modify{ClientOrderID: "invalid", Pair: testPerpetualPair, AssetType: asset.PerpetualContract})
	require.ErrorIs(t, err, errClientOrderIDInvalid, "Invalid client order ID must return the expected error")
	_, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract, TriggerPrice: 90})
	require.ErrorIs(t, err, errRiskManagementUnsupported, "Trigger price on a non-trigger modification must fail closed")

	result, err := ex.ModifyOrder(t.Context(), &order.Modify{
		OrderID:          "7",
		Pair:             testPerpetualPair,
		AssetType:        asset.PerpetualContract,
		Price:            101,
		Amount:           1.5,
		Side:             order.Sell,
		Type:             order.Limit,
		TimeInForce:      order.PostOnly,
		NewClientOrderID: "0x00000000000000000000000000000002",
	})
	require.NoError(t, err, "Modifying an order with explicit fields must not error")
	assert.Equal(t, order.Sell, result.Side, "Modified side should be returned")
	assert.Equal(t, 101.0, result.Price, "Modified price should be returned")
	assert.Equal(t, 1.5, result.Amount, "Modified amount should be returned")
	assert.Equal(t, 1.5, result.RemainingAmount, "Resting modified amount should remain open")
	assert.Equal(t, order.PostOnly, result.TimeInForce, "Modified time in force should be returned")
	assert.Equal(t, "0x00000000000000000000000000000002", result.ClientOrderID, "New client order ID should be returned")

	result, err = ex.ModifyOrder(t.Context(), &order.Modify{ClientOrderID: strings.ToUpper(validClientOrderID), Pair: testPerpetualPair, AssetType: asset.PerpetualContract})
	require.NoError(t, err, "Modifying by client ID with inherited fields must not error")
	assert.Equal(t, order.Buy, result.Side, "Existing side should be inherited")
	assert.Equal(t, order.Limit, result.Type, "Existing type should be inherited")
	assert.Equal(t, order.GoodTillCancel, result.TimeInForce, "Existing time in force should be inherited")
	assert.Equal(t, 1.0, result.Amount, "Only the existing remaining amount should be inherited")
	assert.Equal(t, 100.0, result.Price, "Existing price should be inherited")
	assert.Equal(t, "7", result.OrderID, "Modification by client ID should return the resolved numeric order ID")

	filled := newTradingTestExchange(t, map[string]string{"orderStatus": openOrderResponse}, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"filled":{"oid":7,"totalSz":"1","avgPx":"100"}}]}}}`
	})
	result, err = filled.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract})
	require.NoError(t, err, "Immediately filled modification must not error")
	assert.Equal(t, order.Filled, result.Status, "Immediately filled modification should be marked filled")
	assert.Zero(t, result.RemainingAmount, "Filled modification should have no remaining amount")

	partiallyFilled := newTradingTestExchange(t, map[string]string{"orderStatus": openOrderResponse}, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"filled":{"oid":7,"totalSz":"0.4","avgPx":"100"}}]}}}`
	})
	result, err = partiallyFilled.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract})
	require.NoError(t, err, "Partially filled GTC modification must not error")
	assert.Equal(t, order.PartiallyFilled, result.Status, "Partial GTC modification should remain active")
	assert.Equal(t, 0.6, result.RemainingAmount, "Partial GTC modification should return the open remainder")

	marketModification := newTradingTestExchange(t, map[string]string{
		"orderStatus": openOrderResponse,
		"allMids":     `{"BTC":"100"}`,
	}, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"filled":{"oid":7,"totalSz":"1","avgPx":"101"}}]}}}`
	})
	result, err = marketModification.ModifyOrder(t.Context(), &order.Modify{
		OrderID:           "7",
		Pair:              testPerpetualPair,
		AssetType:         asset.PerpetualContract,
		Type:              order.Market,
		SlippageTolerance: 0.01,
	})
	require.NoError(t, err, "Converting a limit order to market must not error")
	assert.Equal(t, 101.0, result.Price, "Market modification should report the submitted wire price")
	assert.Equal(t, order.ImmediateOrCancel, result.TimeInForce, "Market modification should report its wire time in force")

	partiallyFilledIOC := newTradingTestExchange(t, map[string]string{
		"orderStatus": openOrderResponse,
		"allMids":     `{"BTC":"100"}`,
	}, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"filled":{"oid":7,"totalSz":"0.4","avgPx":"101"}}]}}}`
	})
	result, err = partiallyFilledIOC.ModifyOrder(t.Context(), &order.Modify{
		OrderID:           "7",
		Pair:              testPerpetualPair,
		AssetType:         asset.PerpetualContract,
		Type:              order.Market,
		SlippageTolerance: 0.01,
	})
	require.NoError(t, err, "Partially filled IOC modification must not error")
	assert.Equal(t, order.PartiallyFilledCancelled, result.Status, "Partial IOC modification should be marked partially filled and cancelled")
	assert.Equal(t, 0.6, result.RemainingAmount, "Partial IOC modification should return the unexecuted amount")

	triggerOrderResponse := `{"status":"order","order":{"order":{"coin":"BTC","side":"A","limitPx":"80","sz":"1","origSz":"1","oid":12,"timestamp":1700000000000,"isTrigger":true,"triggerPx":"90","reduceOnly":true,"orderType":"Stop Market","tif":""},"status":"open","statusTimestamp":1700000001000}}`
	triggerModification := newTradingTestExchange(t, map[string]string{"orderStatus": triggerOrderResponse}, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":["waitingForTrigger"]}}}`
	})
	_, err = triggerModification.ModifyOrder(t.Context(), &order.Modify{
		OrderID:   "12",
		Pair:      testPerpetualPair,
		AssetType: asset.PerpetualContract,
	})
	require.ErrorIs(t, err, errRiskManagementUnsupported, "Trigger modification without an explicit mark-price source must fail closed")
	result, err = triggerModification.ModifyOrder(t.Context(), &order.Modify{
		OrderID:          "12",
		Pair:             testPerpetualPair,
		AssetType:        asset.PerpetualContract,
		TriggerPriceType: order.MarkPrice,
	})
	require.NoError(t, err, "Modifying a deferred trigger order with inherited fields must not error")
	assert.Equal(t, "12", result.OrderID, "Deferred trigger modification should retain the existing order ID")
	assert.Equal(t, order.StopMarket, result.Type, "Deferred trigger modification should retain the existing type")
	assert.Equal(t, 90.0, result.TriggerPrice, "Deferred trigger modification should inherit the trigger price")
	assert.Equal(t, order.UnknownTIF, result.TimeInForce, "Trigger modification should not report a limit time in force")

	overfilled := newTradingTestExchange(t, map[string]string{"orderStatus": openOrderResponse}, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"filled":{"oid":7,"totalSz":"2","avgPx":"100"}}]}}}`
	})
	_, err = overfilled.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract})
	require.ErrorIs(t, err, errInvalidFilledSize, "Over-reported modification fill must fail closed")

	invalidFill := newTradingTestExchange(t, map[string]string{"orderStatus": openOrderResponse}, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"filled":{"oid":7,"totalSz":"0.400001","avgPx":"100"}}]}}}`
	})
	_, err = invalidFill.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract})
	require.ErrorIs(t, err, errInvalidFilledSize, "Invalid modification fill precision must return the expected error")

	rejected := newTradingTestExchange(t, map[string]string{"orderStatus": openOrderResponse}, func(string, map[string]any) string {
		return `{"status":"ok","response":{"type":"order","data":{"statuses":[{"error":"bad modify"}]}}}`
	})
	_, err = rejected.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract})
	require.ErrorIs(t, err, order.ErrUnableToPlaceOrder, "Rejected modification must return the expected error")

	notFound := newTradingTestExchange(t, map[string]string{"orderStatus": `{"status":"unknownOid"}`}, nil)
	_, err = notFound.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract})
	require.ErrorIs(t, err, order.ErrOrderNotFound, "Missing existing order must return the expected error")

	filledOrderResponse := `{"status":"order","order":{"order":{"coin":"BTC","side":"B","limitPx":"100","sz":"0","origSz":"2","oid":7,"timestamp":1700000000000,"isTrigger":false,"reduceOnly":true,"orderType":"Limit","tif":"Gtc"},"status":"filled","statusTimestamp":1700000001000}}`
	terminal := newTradingTestExchange(t, map[string]string{"orderStatus": filledOrderResponse}, nil)
	_, err = terminal.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract})
	require.ErrorIs(t, err, errOrderNotModifiable, "Terminal existing order must not be submitted for modification")

	invalidBuild := newTradingTestExchange(t, map[string]string{"orderStatus": openOrderResponse}, nil)
	_, err = invalidBuild.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract, Amount: 0.000001})
	require.ErrorIs(t, err, errSizePrecision, "Invalid modified size must return the expected error")

	malformed := newTradingTestExchange(t, map[string]string{"orderStatus": openOrderResponse}, func(string, map[string]any) string {
		return `{"status":"ok","response":{"data":{"statuses":[]}}}`
	})
	_, err = malformed.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract})
	require.ErrorIs(t, err, errActionStatusCount, "Malformed modification response must return the expected error")

	actionFailure := newTradingTestExchange(t, map[string]string{"orderStatus": openOrderResponse}, func(string, map[string]any) string {
		return `{"status":"err","response":"modify failed"}`
	})
	_, err = actionFailure.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: testPerpetualPair, AssetType: asset.PerpetualContract})
	require.ErrorIs(t, err, errActionResponse, "Signed modification failure must be returned")
}

func TestGetOrderInfo(t *testing.T) {
	response := `{"status":"order","order":{"order":{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"2","oid":7,"timestamp":1700000000000,"isTrigger":false,"reduceOnly":false,"orderType":"Limit","tif":"Gtc"},"status":"open","statusTimestamp":1700000001000}}`
	ex := newTradingTestExchange(t, map[string]string{"orderStatus": response}, nil)

	_, err := ex.GetOrderInfo(t.Context(), "", testPerpetualPair, asset.PerpetualContract)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "Empty order ID must return the expected error")
	_, err = ex.GetOrderInfo(t.Context(), "7", testPerpetualPair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Unsupported order asset must return the expected error")
	_, err = ex.GetOrderInfo(t.Context(), "invalid", testPerpetualPair, asset.PerpetualContract)
	require.ErrorIs(t, err, errClientOrderIDInvalid, "Invalid client order ID must return the expected error")

	detail, err := ex.GetOrderInfo(t.Context(), "7", testPerpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "Getting order by numeric ID must not error")
	assert.Equal(t, "7", detail.OrderID, "Order detail should be returned")
	_, err = ex.GetOrderInfo(t.Context(), validClientOrderID, currency.EMPTYPAIR, asset.Empty)
	require.NoError(t, err, "Getting order by client ID without filters must not error")

	_, err = ex.GetOrderInfo(t.Context(), "7", testSpotPair, asset.PerpetualContract)
	require.ErrorIs(t, err, order.ErrOrderNotFound, "Mismatched pair filter must return not found")
	_, err = ex.GetOrderInfo(t.Context(), "7", testPerpetualPair, asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderNotFound, "Mismatched asset filter must return not found")

	unknown := newTradingTestExchange(t, map[string]string{"orderStatus": `{"status":"unknownOid"}`}, nil)
	_, err = unknown.GetOrderInfo(t.Context(), "7", testPerpetualPair, asset.PerpetualContract)
	require.ErrorIs(t, err, order.ErrOrderNotFound, "Unknown order response must return not found")

	nilOrder := newTradingTestExchange(t, map[string]string{"orderStatus": `{"status":"order","order":null}`}, nil)
	_, err = nilOrder.GetOrderInfo(t.Context(), "7", testPerpetualPair, asset.PerpetualContract)
	require.ErrorIs(t, err, order.ErrOrderNotFound, "Order response without order data must return not found")

	missingCredentials := new(Exchange)
	missingCredentials.SetDefaults()
	_, err = missingCredentials.GetOrderInfo(t.Context(), "7", testPerpetualPair, asset.PerpetualContract)
	require.Error(t, err, "Getting order without credentials must error")

	badOrder := newTradingTestExchange(t, map[string]string{"orderStatus": `{"status":"order","order":{"order":{"coin":"MISSING","side":"B","limitPx":"1","sz":"1","origSz":"1","oid":7,"timestamp":1700000000000,"orderType":"Limit","tif":"Gtc"},"status":"open","statusTimestamp":1700000001000}}`}, nil)
	_, err = badOrder.GetOrderInfo(t.Context(), "7", testPerpetualPair, asset.PerpetualContract)
	require.ErrorIs(t, err, errPairMappingNotFound, "Invalid returned order must return its conversion error")

	failed := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestCredentials(failed, &accounts.Credentials{Key: officialSigningAddress})
	failed.setPairMappings(asset.PerpetualContract, []pairMapping{{pair: testPerpetualPair, coin: "BTC"}})
	_, err = failed.GetOrderInfo(t.Context(), "7", testPerpetualPair, asset.PerpetualContract)
	require.Error(t, err, "Order-status HTTP failure must be returned")
}

func TestGetOpenOrdersForAsset(t *testing.T) {
	var requestedDEXes []string
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding scoped open-orders request should not error") {
			return
		}
		if request.Type == infoTypePerpetualDEXs {
			_, err := w.Write([]byte(`[null,{"name":"` + testBuilderDEXName + `"}]`))
			assert.NoError(t, err, "Writing the perpetual DEX registry should not error")
			return
		}
		requestedDEXes = append(requestedDEXes, request.DEX)
		coin := "BTC"
		orderID := 7
		if request.DEX == testBuilderDEXName {
			coin = "xyz:XYZ100"
			orderID = 8
		}
		_, err := fmt.Fprintf(w, `[{"coin":%q,"oid":%d}]`, coin, orderID)
		assert.NoError(t, err, "Writing scoped open-orders response should not error")
	}))
	orders, err := ex.getOpenOrdersForAsset(t.Context(), officialSigningAddress, asset.PerpetualContract)
	require.NoError(t, err, "Getting cold open orders across registered perpetual DEXes must not error")
	require.Len(t, orders, 2, "getOpenOrdersForAsset must combine one response per DEX")
	assert.Equal(t, []string{"", testBuilderDEXName}, requestedDEXes, "Perpetual open orders should query each DEX once")

	requestedDEXes = nil
	orders, err = ex.getOpenOrdersForAsset(t.Context(), officialSigningAddress, asset.Spot)
	require.NoError(t, err, "Getting spot open orders must not error")
	require.Len(t, orders, 1, "getOpenOrdersForAsset must use the default DEX response for spot")
	assert.Equal(t, []string{""}, requestedDEXes, "Spot open orders should query only the default DEX")
	_, err = ex.getOpenOrdersForAsset(t.Context(), officialSigningAddress, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Unsupported open-order asset must return the expected error")

	noMappings := newStaticInfoExchange(t, map[string]string{"frontendOpenOrders": `[]`})
	orders, err = noMappings.getOpenOrdersForAsset(t.Context(), officialSigningAddress, asset.PerpetualContract)
	require.NoError(t, err, "Getting perpetual open orders without cached mappings must not error")
	assert.Empty(t, orders, "Default DEX open-order fallback should retain an empty response")

	failed := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding failed open-orders request should not error") {
			return
		}
		if request.Type == infoTypePerpetualDEXs {
			_, err := w.Write([]byte(`[null]`))
			assert.NoError(t, err, "Writing the failed open-orders registry should not error")
			return
		}
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = failed.getOpenOrdersForAsset(t.Context(), officialSigningAddress, asset.PerpetualContract)
	require.Error(t, err, "Scoped open-order HTTP failure must be returned")
}

func TestGetOrders(t *testing.T) {
	openOrders := `[{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"2","oid":7,"timestamp":1700000000000,"isTrigger":false,"reduceOnly":false,"orderType":"Limit","tif":"Gtc"},{"coin":"@107","side":"A","limitPx":"10","sz":"2","origSz":"2","oid":8,"timestamp":1700000000000,"isTrigger":false,"reduceOnly":false,"orderType":"Limit","tif":"Gtc"}]`
	history := `[{"order":{"coin":"BTC","side":"B","limitPx":"100","sz":"0","origSz":"2","oid":7,"timestamp":1700000000000,"isTrigger":false,"reduceOnly":false,"orderType":"Limit","tif":"Gtc"},"status":"filled","statusTimestamp":1700000001000},{"order":{"coin":"@107","side":"A","limitPx":"10","sz":"0","origSz":"2","oid":8,"timestamp":1700000000000,"isTrigger":false,"reduceOnly":false,"orderType":"Limit","tif":"Gtc"},"status":"filled","statusTimestamp":1700000001000}]`
	ex := newTradingTestExchange(t, map[string]string{"frontendOpenOrders": openOrders, "historicalOrders": history}, nil)

	_, err := ex.GetActiveOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "Nil active-order request must return the expected error")
	_, err = ex.GetOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "Nil order-history request must return the expected error")

	unsupported := &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType}
	_, err = ex.GetActiveOrders(t.Context(), unsupported)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Unsupported active-order asset must return the expected error")
	_, err = ex.GetOrderHistory(t.Context(), unsupported)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Unsupported history asset must return the expected error")

	orderRequest := &order.MultiOrderRequest{AssetType: asset.PerpetualContract, Side: order.AnySide, Type: order.AnyType}
	missingCredentials := new(Exchange)
	missingCredentials.SetDefaults()
	_, err = missingCredentials.GetActiveOrders(t.Context(), orderRequest)
	require.Error(t, err, "Active orders without credentials must error")
	_, err = missingCredentials.GetOrderHistory(t.Context(), orderRequest)
	require.Error(t, err, "Order history without credentials must error")

	active, err := ex.GetActiveOrders(t.Context(), orderRequest)
	require.NoError(t, err, "Getting active perpetual orders must not error")
	require.Len(t, active, 1, "Active orders must filter out other assets")
	assert.Equal(t, "7", active[0].OrderID, "Active perpetual order should be returned")

	historical, err := ex.GetOrderHistory(t.Context(), orderRequest)
	require.NoError(t, err, "Getting perpetual order history must not error")
	require.Len(t, historical, 1, "Order history must filter out other assets")
	assert.Equal(t, order.Filled, historical[0].Status, "Historical order status should be converted")

	failed := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestCredentials(failed, &accounts.Credentials{Key: officialSigningAddress})
	_, err = failed.GetActiveOrders(t.Context(), orderRequest)
	require.Error(t, err, "Active-order HTTP failure must be returned")
	_, err = failed.GetOrderHistory(t.Context(), orderRequest)
	require.Error(t, err, "Order-history HTTP failure must be returned")

	badOrders := newTradingTestExchange(t, map[string]string{
		"frontendOpenOrders": `[{"coin":"MISSING","side":"B","limitPx":"1","sz":"1","origSz":"1","oid":7,"timestamp":1700000000000,"orderType":"Limit","tif":"Gtc"}]`,
		"historicalOrders":   `[{"order":{"coin":"MISSING","side":"B","limitPx":"1","sz":"1","origSz":"1","oid":7,"timestamp":1700000000000,"orderType":"Limit","tif":"Gtc"},"status":"open","statusTimestamp":1700000001000}]`,
	}, nil)
	_, err = badOrders.GetActiveOrders(t.Context(), orderRequest)
	require.ErrorIs(t, err, errPairMappingNotFound, "Active-order conversion failure must be returned")
	_, err = badOrders.GetOrderHistory(t.Context(), orderRequest)
	require.ErrorIs(t, err, errPairMappingNotFound, "Order-history conversion failure must be returned")

	mixedOrders := newTradingTestExchange(t, map[string]string{
		"frontendOpenOrders": `[{"coin":"MISSING","side":"B","limitPx":"1","sz":"1","origSz":"1","oid":7,"timestamp":1700000000000,"orderType":"Limit","tif":"Gtc"},{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"2","oid":8,"timestamp":1700000000000,"orderType":"Limit","tif":"Gtc"}]`,
		"historicalOrders":   `[{"order":{"coin":"MISSING","side":"B","limitPx":"1","sz":"1","origSz":"1","oid":7,"timestamp":1700000000000,"orderType":"Limit","tif":"Gtc"},"status":"open","statusTimestamp":1700000001000},{"order":{"coin":"BTC","side":"B","limitPx":"100","sz":"0","origSz":"2","oid":8,"timestamp":1700000000000,"orderType":"Limit","tif":"Gtc"},"status":"filled","statusTimestamp":1700000001000}]`,
	}, nil)
	active, err = mixedOrders.GetActiveOrders(t.Context(), orderRequest)
	require.ErrorIs(t, err, errPairMappingNotFound, "Mixed active orders must report the skipped conversion")
	require.Len(t, active, 1, "Mixed active orders must retain convertible orders")
	assert.Equal(t, "8", active[0].OrderID, "Convertible active order should be returned")
	historical, err = mixedOrders.GetOrderHistory(t.Context(), orderRequest)
	require.ErrorIs(t, err, errPairMappingNotFound, "Mixed order history must report the skipped conversion")
	require.Len(t, historical, 1, "Mixed order history must retain convertible orders")
	assert.Equal(t, "8", historical[0].OrderID, "Convertible historical order should be returned")
}

func TestCancelAllOrders(t *testing.T) {
	openOrders := `[{"coin":"BTC","oid":7},{"coin":"@107","oid":8}]`
	ex := newTradingTestExchange(t, map[string]string{"frontendOpenOrders": openOrders}, func(actionType string, action map[string]any) string {
		cancels, _ := action["cancels"].([]any)
		statuses := make([]string, len(cancels))
		for i := range statuses {
			statuses[i] = `"success"`
		}
		return `{"status":"ok","response":{"type":"` + actionType + `","data":{"statuses":[` + strings.Join(statuses, ",") + `]}}}`
	})

	_, err := ex.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "Nil cancel-all request must return the expected error")
	_, err = ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Options})
	require.ErrorIs(t, err, asset.ErrNotSupported, "Unsupported cancel-all asset must return the expected error")

	missingCredentials := new(Exchange)
	missingCredentials.SetDefaults()
	_, err = missingCredentials.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualContract})
	require.Error(t, err, "Cancel-all without credentials must error")

	result, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualContract})
	require.NoError(t, err, "Canceling all perpetual orders must not error")
	assert.Equal(t, "success", result.Status["7"], "Matching perpetual order should be canceled")
	assert.NotContains(t, result.Status, "8", "Other asset order should be excluded")

	result, err = ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualContract, Pair: currency.NewPair(currency.ETH, currency.USDC)})
	require.NoError(t, err, "Canceling unmatched pair must not error")
	assert.Empty(t, result.Status, "Unmatched pair should produce no cancellations")

	badOrders := newTradingTestExchange(t, map[string]string{"frontendOpenOrders": `[{"coin":"MISSING","oid":7}]`}, nil)
	_, err = badOrders.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualContract})
	require.ErrorIs(t, err, errPairMappingNotFound, "Cancel-all mapping failure must be returned")

	mixedOrders := newTradingTestExchange(t, map[string]string{"frontendOpenOrders": `[{"coin":"MISSING","oid":6},{"coin":"BTC","oid":7}]`}, func(actionType string, _ map[string]any) string {
		return `{"status":"ok","response":{"type":"` + actionType + `","data":{"statuses":["success"]}}}`
	})
	result, err = mixedOrders.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualContract})
	require.ErrorIs(t, err, errPairMappingNotFound, "Cancel-all must report an unmappable skipped order")
	assert.Equal(t, "success", result.Status["7"], "Cancel-all should still cancel mappable orders")

	failed := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestCredentials(failed, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey})
	_, err = failed.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualContract})
	require.Error(t, err, "Cancel-all order lookup failure must be returned")

	orders := make([]string, maximumActionBatchSize+1)
	for i := range orders {
		orders[i] = `{"coin":"BTC","oid":` + strconv.Itoa(i+1) + `}`
	}
	chunked := newTradingTestExchange(t, map[string]string{"frontendOpenOrders": `[` + strings.Join(orders, ",") + `]`}, func(actionType string, action map[string]any) string {
		cancels, _ := action["cancels"].([]any)
		statuses := make([]string, len(cancels))
		for i := range statuses {
			statuses[i] = `"success"`
		}
		return `{"status":"ok","response":{"type":"` + actionType + `","data":{"statuses":[` + strings.Join(statuses, ",") + `]}}}`
	})
	result, err = chunked.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualContract})
	require.NoError(t, err, "Cancel-all must chunk orders beyond one action batch")
	assert.Len(t, result.Status, maximumActionBatchSize+1, "Every chunked order should be canceled")

	partial := newTradingTestExchange(t, map[string]string{"frontendOpenOrders": `[{"coin":"BTC","oid":7}]`}, func(string, map[string]any) string {
		return `{"status":"ok","response":{"data":{"statuses":[{"error":"already closed"}]}}}`
	})
	result, err = partial.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualContract})
	require.Error(t, err, "Cancel-all per-order failure must be returned")
	assert.Equal(t, "already closed", result.Status["7"], "Cancel-all failure status should be retained")
}
