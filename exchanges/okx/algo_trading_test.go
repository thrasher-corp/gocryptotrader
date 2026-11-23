package okx

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// algoOrderActionReply is a successful algo order action row for the request pins
var algoOrderActionReply = []map[string]string{{"algoId": "1", "sCode": "0", "sMsg": ""}}

// testPlaceAlgoOrderHelper places an algo order through a helper in both test modes, comparing the mocked reply with
// exp
func testPlaceAlgoOrderHelper(t *testing.T, name string, place func(*PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error), arg *PlaceAlgoOrderRequest, exp *AlgoOrderActionResponse) {
	t.Helper()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := place(arg)
	require.NoErrorf(t, err, "%s must not error", name)
	if mockTests {
		assert.Equalf(t, exp, result, "%s should decode every field", name)
	} else {
		assert.NotEmptyf(t, result.AlgoID, "%s should return the algo ID", name)
	}
}

func TestPlaceAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceAlgoOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.PlaceAlgoOrder(contextGenerate(), &PlaceAlgoOrderRequest{})
	require.ErrorIs(t, err, errMissingInstrumentID)
	arg := &PlaceAlgoOrderRequest{
		ReduceOnly: true,
	}
	arg.OrderType = "conditional"
	_, err = e.PlaceAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.PlaceAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = TradeModeCross
	_, err = e.PlaceAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.Lower()
	arg.OrderType = ""
	_, err = e.PlaceAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = e.PlaceAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *PlaceAlgoOrderRequest
			body string
		}{
			{
				name: "required parameters only",
				arg:  &PlaceAlgoOrderRequest{InstrumentID: perpetualSwapPair.String(), TradeMode: TradeModeCross, Side: order.Buy.Lower(), OrderType: orderConditional, CloseFraction: 1},
				body: `{"instId":"BTC-USDT-SWAP","tdMode":"cross","side":"buy","ordType":"conditional","closeFraction":"1"}`,
			},
			{
				name: "every optional parameter",
				arg: &PlaceAlgoOrderRequest{
					InstrumentID:               perpetualSwapPair.String(),
					TradeMode:                  "CROSS",
					Currency:                   currency.USDT,
					Side:                       "SELL",
					PositionSide:               positionSideShort,
					OrderType:                  orderTrigger,
					Size:                       1,
					OrderTag:                   "tag1",
					TargetCurrency:             targetCurrencyBase,
					AlgoClientOrderID:          "algo1",
					CloseFraction:              1,
					TradeQuoteCurrency:         currency.USDC,
					ReduceOnly:                 true,
					TakeProfitTriggerPrice:     90000,
					TakeProfitTriggerPriceType: "last",
					TakeProfitOrderPrice:       -1,
					TakeProfitOrderKind:        "condition",
					StopLossTriggerPrice:       99000,
					StopLossTriggerPriceType:   "mark",
					StopLossOrderPrice:         -1,
					CancelOnClosePosition:      true,
					ChaseType:                  "distance",
					ChaseValue:                 10,
					MaxChaseType:               "ratio",
					MaxChaseValue:              0.1,
					TriggerPrice:               95000,
					OrderPrice:                 -1,
					AdvanceOrderType:           orderChase,
					AdvancedChaseParameters:    []AdvancedChaseParameters{{ChaseType: "distance", ChaseValue: 10, MaxChaseType: "ratio", MaxChaseValue: 0.1}},
					TriggerPriceType:           "index",
					AttachAlgoOrders: []AttachAlgoOrderRequest{{
						AttachAlgoClientOrderID: "attach1", TakeProfitTriggerPrice: 90000, TakeProfitTriggerRatio: 0.3, TakeProfitTriggerPriceType: "last",
						TakeProfitOrderPrice: -1, StopLossTriggerPrice: 99000, StopLossTriggerRatio: 0.1, StopLossTriggerPriceType: "mark",
						StopLossOrderPrice: -1, CallbackRatio: 0.05, CallbackSpread: 100, ActivePrice: 96000,
					}},
					CallbackRatio:    0.05,
					CallbackSpread:   100,
					ActivePrice:      96000,
					PriceVariance:    0.01,
					PriceSpread:      10,
					SizeLimit:        1,
					PriceLimit:       95000,
					TimeInterval:     10,
					LimitOrderNumber: 5,
					Aggressiveness:   "mid",
					TriggerParameters: []IcebergTriggerRequest{{
						TriggerAction: "start", TriggerStrategy: "rsi", TriggerPrice: 90000, TriggerCondition: "cross_up", Timeframe: "15m", Threshold: 30, TimePeriod: 14,
					}},
				},
				body: `{"instId":"BTC-USDT-SWAP","tdMode":"cross","ccy":"USDT","side":"sell","posSide":"short","ordType":"trigger","sz":"1","tag":"tag1","tgtCcy":"base_ccy","algoClOrdId":"algo1","closeFraction":"1","tradeQuoteCcy":"USDC","reduceOnly":true,"tpTriggerPx":"90000","tpTriggerPxType":"last","tpOrdPx":"-1","tpOrdKind":"condition","slTriggerPx":"99000","slTriggerPxType":"mark","slOrdPx":"-1","cxlOnClosePos":true,"chaseType":"distance","chaseVal":"10","maxChaseType":"ratio","maxChaseVal":"0.1","triggerPx":"95000","orderPx":"-1","advanceOrdType":"chase","advChaseParams":[{"chaseType":"distance","chaseVal":"10","maxChaseType":"ratio","maxChaseVal":"0.1"}],"triggerPxType":"index","attachAlgoOrds":[{"attachAlgoClOrdId":"attach1","tpTriggerPx":"90000","tpTriggerRatio":"0.3","tpOrdPx":"-1","slTriggerPx":"99000","slTriggerRatio":"0.1","slOrdPx":"-1","tpTriggerPxType":"last","slTriggerPxType":"mark","callbackRatio":"0.05","callbackSpread":"100","activePx":"96000"}],"callbackRatio":"0.05","callbackSpread":"100","activePx":"96000","pxVar":"0.01","pxSpread":"10","szLimit":"1","pxLimit":"95000","timeInterval":"10","lmtOrderNumber":"5","aggressiveness":"mid","triggerParams":[{"triggerAction":"start","triggerStrategy":"rsi","triggerPx":"90000","triggerCond":"cross_up","timeframe":"15m","thold":"30","timePeriod":"14"}]}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/trade/order-algo", tc.body, algoOrderActionReply)
				_, err := ex.PlaceAlgoOrder(t.Context(), tc.arg)
				require.NoError(t, err, "PlaceAlgoOrder must not error")
			})
		}
	})

	testPlaceAlgoOrderHelper(t, "PlaceAlgoOrder", func(arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
		return e.PlaceAlgoOrder(contextGenerate(), arg)
	}, &PlaceAlgoOrderRequest{
		InstrumentID:           mainPair.String(),
		TradeMode:              TradeModeCross,
		Side:                   order.Buy.Lower(),
		OrderType:              orderConditional,
		Size:                   2,
		OrderTag:               "tpsl1",
		AlgoClientOrderID:      "order1234",
		TakeProfitTriggerPrice: 15,
		TakeProfitOrderPrice:   18,
	}, &AlgoOrderActionResponse{AlgoID: "1836487817828872192", ClientOrderID: "order1234", AlgoClientOrderID: "order1234", Tag: "tpsl1"})
}

func TestPlaceStopOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceStopOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.PlaceStopOrder(contextGenerate(), &PlaceAlgoOrderRequest{})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
	arg := &PlaceAlgoOrderRequest{
		ReduceOnly: true,
	}
	arg.OrderType = "conditional"
	_, err = e.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.TakeProfitTriggerPrice = 123
	_, err = e.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrUnknownPriceType)

	arg.TakeProfitTriggerPriceType = "last_price"
	arg.AlgoClientOrderID = "12345"
	_, err = e.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = TradeModeCross
	_, err = e.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.Lower()
	arg.OrderType = ""
	_, err = e.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = e.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	testPlaceAlgoOrderHelper(t, "PlaceStopOrder", func(arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
		return e.PlaceStopOrder(contextGenerate(), arg)
	}, &PlaceAlgoOrderRequest{
		AlgoClientOrderID:          "681096944655273984",
		TakeProfitTriggerPriceType: "index",
		InstrumentID:               mainPair.String(),
		OrderType:                  "conditional",
		Side:                       order.Sell.Lower(),
		TradeMode:                  "isolated",
		Size:                       12,
		TakeProfitTriggerPrice:     12335,
		TakeProfitOrderPrice:       1234,
	}, &AlgoOrderActionResponse{AlgoID: "1836487817828872193", AlgoClientOrderID: "681096944655273984"})
}

func TestPlaceIcebergOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceIcebergOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.PlaceIcebergOrder(contextGenerate(), &PlaceAlgoOrderRequest{})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.PlaceIcebergOrder(contextGenerate(), &PlaceAlgoOrderRequest{ReduceOnly: true})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.PlaceIcebergOrder(contextGenerate(), &PlaceAlgoOrderRequest{OrderType: "iceberg", SizeLimit: 123, LimitOrderNumber: 5, Aggressiveness: "mid"})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.PlaceIcebergOrder(contextGenerate(), &PlaceAlgoOrderRequest{OrderType: orderSmartIceberg})
	require.ErrorIs(t, err, errMissingSizeLimit)
	_, err = e.PlaceIcebergOrder(contextGenerate(), &PlaceAlgoOrderRequest{OrderType: orderSmartIceberg, SizeLimit: 123})
	require.ErrorIs(t, err, errLimitOrderNumberRequired)
	_, err = e.PlaceIcebergOrder(contextGenerate(), &PlaceAlgoOrderRequest{OrderType: orderSmartIceberg, SizeLimit: 123, LimitOrderNumber: 5})
	require.ErrorIs(t, err, errAggressivenessRequired)
	_, err = e.PlaceIcebergOrder(contextGenerate(), &PlaceAlgoOrderRequest{OrderType: orderSmartIceberg, SizeLimit: 123, LimitOrderNumber: 5, Aggressiveness: "mid"})
	require.ErrorIs(t, err, errMissingInstrumentID)

	testPlaceAlgoOrderHelper(t, "PlaceIcebergOrder", func(arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
		return e.PlaceIcebergOrder(contextGenerate(), arg)
	}, &PlaceAlgoOrderRequest{
		AlgoClientOrderID: "681096944655273984",
		PriceLimit:        100.22,
		SizeLimit:         9999.9,
		LimitOrderNumber:  5,
		Aggressiveness:    "conservative",
		TriggerParameters: []IcebergTriggerRequest{{TriggerAction: "start", TriggerStrategy: "price", TriggerPrice: 90000, TriggerCondition: "cross_down"}},
		InstrumentID:      mainPair.String(),
		OrderType:         orderSmartIceberg,
		Side:              order.Buy.Lower(),
		TradeMode:         "isolated",
		Size:              6,
	}, &AlgoOrderActionResponse{AlgoID: "1836487817828872194", AlgoClientOrderID: "681096944655273984"})
}

func TestPlaceTWAPOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceTWAPOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.PlaceTWAPOrder(contextGenerate(), &PlaceAlgoOrderRequest{})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	_, err = e.PlaceTWAPOrder(contextGenerate(), &PlaceAlgoOrderRequest{ReduceOnly: true})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	_, err = e.PlaceTWAPOrder(contextGenerate(), &PlaceAlgoOrderRequest{OrderType: "twap"})
	require.ErrorIs(t, err, errMissingSizeLimit)

	_, err = e.PlaceTWAPOrder(contextGenerate(), &PlaceAlgoOrderRequest{SizeLimit: 2, OrderType: "twap"})
	require.ErrorIs(t, err, errInvalidPriceLimit)

	_, err = e.PlaceTWAPOrder(contextGenerate(), &PlaceAlgoOrderRequest{SizeLimit: 2, OrderType: "twap", PriceLimit: 1234.5})
	require.ErrorIs(t, err, errMissingIntervalValue)

	testPlaceAlgoOrderHelper(t, "PlaceTWAPOrder", func(arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
		return e.PlaceTWAPOrder(contextGenerate(), arg)
	}, &PlaceAlgoOrderRequest{
		AlgoClientOrderID: "681096944655273984",
		InstrumentID:      mainPair.String(),
		PriceLimit:        100.22,
		SizeLimit:         9999.9,
		OrderType:         "twap",
		PriceSpread:       0.4,
		TradeMode:         "cross",
		Side:              order.Sell.Lower(),
		Size:              6,
		TimeInterval:      uint64(kline.ThreeDay.Duration().Seconds()),
	}, &AlgoOrderActionResponse{AlgoID: "1836487817828872195", AlgoClientOrderID: "681096944655273984"})
}

func TestPlaceTakeProfitStopLossOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceTakeProfitStopLossOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.PlaceTakeProfitStopLossOrder(contextGenerate(), &PlaceAlgoOrderRequest{})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.PlaceTakeProfitStopLossOrder(contextGenerate(), &PlaceAlgoOrderRequest{ReduceOnly: true})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.PlaceTakeProfitStopLossOrder(contextGenerate(), &PlaceAlgoOrderRequest{OrderType: "conditional"})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
	_, err = e.PlaceTakeProfitStopLossOrder(contextGenerate(), &PlaceAlgoOrderRequest{
		OrderType:                "conditional",
		StopLossTriggerPrice:     1234,
		StopLossTriggerPriceType: "abcd",
	})
	require.ErrorIs(t, err, order.ErrUnknownPriceType)

	testPlaceAlgoOrderHelper(t, "PlaceTakeProfitStopLossOrder", func(arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
		return e.PlaceTakeProfitStopLossOrder(contextGenerate(), arg)
	}, &PlaceAlgoOrderRequest{
		OrderType:                "conditional",
		StopLossTriggerPrice:     1234,
		StopLossTriggerPriceType: "last",
		StopLossOrderPrice:       marketOrderPrice,
		AlgoClientOrderID:        "681096944655273984",
		InstrumentID:             mainPair.String(),
		TradeMode:                "cross",
		Side:                     order.Sell.Lower(),
		Size:                     6,
	}, &AlgoOrderActionResponse{AlgoID: "1836487817828872196", AlgoClientOrderID: "681096944655273984"})
}

func TestPlaceChaseAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceChaseAlgoOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.PlaceChaseAlgoOrder(contextGenerate(), &PlaceAlgoOrderRequest{})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	arg := &PlaceAlgoOrderRequest{
		ReduceOnly: true,
	}
	_, err = e.PlaceChaseAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = orderChase
	arg.MaxChaseType = "percentage"
	_, err = e.PlaceChaseAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errPriceTrackingNotSet)

	arg.MaxChaseType = "percentage"
	arg.MaxChaseValue = .5
	_, err = e.PlaceChaseAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.PlaceChaseAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = "cross"
	_, err = e.PlaceChaseAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.Lower()
	_, err = e.PlaceChaseAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	testPlaceAlgoOrderHelper(t, "PlaceChaseAlgoOrder", func(arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
		return e.PlaceChaseAlgoOrder(contextGenerate(), arg)
	}, &PlaceAlgoOrderRequest{
		AlgoClientOrderID: "681096944655273984",
		InstrumentID:      mainPair.String(),
		OrderType:         orderChase,
		TradeMode:         "cross",
		Side:              order.Sell.Lower(),
		MaxChaseType:      "distance",
		MaxChaseValue:     .5,
		Size:              6,
	}, &AlgoOrderActionResponse{AlgoID: "1836487817828872197", AlgoClientOrderID: "681096944655273984"})
}

func TestPlaceTriggerAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceTriggerAlgoOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.PlaceTriggerAlgoOrder(contextGenerate(), &PlaceAlgoOrderRequest{})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	_, err = e.PlaceTriggerAlgoOrder(contextGenerate(), &PlaceAlgoOrderRequest{AlgoClientOrderID: "1234"})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	_, err = e.PlaceTriggerAlgoOrder(contextGenerate(), &PlaceAlgoOrderRequest{AlgoClientOrderID: "1234", OrderType: "trigger"})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	_, err = e.PlaceTriggerAlgoOrder(contextGenerate(), &PlaceAlgoOrderRequest{AlgoClientOrderID: "1234", OrderType: "trigger", TriggerPrice: 123., TriggerPriceType: "abcd"})
	require.ErrorIs(t, err, order.ErrUnknownPriceType)

	_, err = e.PlaceTriggerAlgoOrder(contextGenerate(), &PlaceAlgoOrderRequest{AlgoClientOrderID: "1234", OrderType: "trigger", TriggerPrice: 123., AdvanceOrderType: orderChase})
	require.ErrorIs(t, err, errPriceTrackingNotSet)

	testPlaceAlgoOrderHelper(t, "PlaceTriggerAlgoOrder", func(arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
		return e.PlaceTriggerAlgoOrder(contextGenerate(), arg)
	}, &PlaceAlgoOrderRequest{
		AlgoClientOrderID: "681096944655273984",
		TriggerPriceType:  "mark",
		TriggerPrice:      1234,
		OrderPrice:        marketOrderPrice,
		InstrumentID:      mainPair.String(),
		OrderType:         "trigger",
		Side:              order.Buy.Lower(),
		TradeMode:         "cross",
		Size:              5,
	}, &AlgoOrderActionResponse{AlgoID: "1836487817828872198", AlgoClientOrderID: "681096944655273984"})
}

func TestPlaceTrailingStopOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceTrailingStopOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.PlaceTrailingStopOrder(contextGenerate(), &PlaceAlgoOrderRequest{})
	assert.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.PlaceTrailingStopOrder(contextGenerate(), &PlaceAlgoOrderRequest{Size: 2})
	assert.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.PlaceTrailingStopOrder(contextGenerate(), &PlaceAlgoOrderRequest{Size: 2, OrderType: orderMoveOrderStop})
	assert.ErrorIs(t, err, errPriceTrackingNotSet)

	testPlaceAlgoOrderHelper(t, "PlaceTrailingStopOrder", func(arg *PlaceAlgoOrderRequest) (*AlgoOrderActionResponse, error) {
		return e.PlaceTrailingStopOrder(contextGenerate(), arg)
	}, &PlaceAlgoOrderRequest{
		AlgoClientOrderID: "681096944655273984", CallbackRatio: 0.01,
		InstrumentID: mainPair.String(), OrderType: orderMoveOrderStop,
		Side: order.Buy.Lower(), TradeMode: "isolated",
		Size: 2, ActivePrice: 1234,
	}, &AlgoOrderActionResponse{AlgoID: "1836487817828872199", AlgoClientOrderID: "681096944655273984"})
}

func TestCancelAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAlgoOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := CancelAlgoOrderRequest{}
	_, err = e.CancelAlgoOrder(contextGenerate(), []CancelAlgoOrderRequest{arg})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.AlgoID = "90994943"
	_, err = e.CancelAlgoOrder(contextGenerate(), []CancelAlgoOrderRequest{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	arg.AlgoID = ""
	_, err = e.CancelAlgoOrder(contextGenerate(), []CancelAlgoOrderRequest{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/trade/cancel-algos",
			`[{"instId":"BTC-USDT","algoId":"90994943"},{"instId":"BTC-USDT-SWAP","algoClOrdId":"algo2"}]`,
			[]map[string]string{{"algoId": "90994943", "sCode": "0"}, {"algoId": "2", "algoClOrdId": "algo2", "sCode": "0"}})
		_, err := ex.CancelAlgoOrder(t.Context(), []CancelAlgoOrderRequest{
			{InstrumentID: mainPair.String(), AlgoID: "90994943"},
			{InstrumentID: perpetualSwapPair.String(), AlgoClientOrderID: "algo2"},
		})
		require.NoError(t, err, "CancelAlgoOrder must not error")
	})

	t.Run("charges each order to OKX's order budget", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			orders int
			exp    map[request.EndpointLimit]float64
		}{
			{"a lone order", 1, map[request.EndpointLimit]float64{cancelAlgoOrderEPL: 1}},
			{"two orders", 2, map[request.EndpointLimit]float64{cancelAlgoOrderEPL: 2}},
			{"a full batch", maxCancelAlgosPerRequest, map[request.EndpointLimit]float64{cancelAlgoOrderEPL: maxCancelAlgosPerRequest}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, orderRowsHandler(t, "/trade/cancel-algos", "algoId", tc.orders))
				pools := chargeToPools(t, ex)
				args := make([]CancelAlgoOrderRequest, tc.orders)
				for i := range args {
					args[i] = CancelAlgoOrderRequest{InstrumentID: mainPair.String(), AlgoID: strconv.Itoa(i + 1)}
				}
				_, err := ex.CancelAlgoOrder(chargeContext(t), args)
				require.NoError(t, err, "CancelAlgoOrder must not error")
				assert.Equal(t, tc.exp, takenTokens(pools), "CancelAlgoOrder should take a token per algo order")
			})
		}
	})

	t.Run("sends nothing its order budget cannot cover", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		pools := chargeToPools(t, ex)
		require.True(t, pools[cancelAlgoOrderEPL].AllowN(time.Now(), orderBudgetPoolSize-1), "the pool must give out all but one of its tokens")
		_, err := ex.CancelAlgoOrder(chargeContext(t), []CancelAlgoOrderRequest{{InstrumentID: mainPair.String(), AlgoID: "1"}, {InstrumentID: mainPair.String(), AlgoID: "2"}})
		assert.ErrorIs(t, err, context.DeadlineExceeded, "CancelAlgoOrder should fail when its budget cannot cover both orders in time")
		assert.Equal(t, 1.0, pools[cancelAlgoOrderEPL].Tokens(), "CancelAlgoOrder should leave the budget it could not use")
	})

	t.Run("charges every attempt of a retried batch", func(t *testing.T) {
		t.Parallel()
		ex, pools, requests := newRetriedBatchExchange(t, "/trade/cancel-algos", "algoId")
		_, err := ex.CancelAlgoOrder(chargeContext(t), []CancelAlgoOrderRequest{{InstrumentID: mainPair.String(), AlgoID: "1"}, {InstrumentID: mainPair.String(), AlgoID: "2"}, {InstrumentID: mainPair.String(), AlgoID: "3"}})
		require.NoError(t, err, "CancelAlgoOrder must not error")
		require.Equal(t, int32(2), requests.Load(), "the batch must be sent again after the venue refused it")
		assert.Equal(t, map[request.EndpointLimit]float64{cancelAlgoOrderEPL: 6}, takenTokens(pools), "CancelAlgoOrder should take a token per algo order on every attempt")
	})

	t.Run("cancels the algo order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.CancelAlgoOrder(contextGenerate(), []CancelAlgoOrderRequest{{InstrumentID: mainPair.String(), AlgoID: "90994943"}})
		require.NoError(t, err, "CancelAlgoOrder must not error")
		if mockTests {
			assert.Equal(t, []AlgoOrderActionResponse{{AlgoID: "90994943"}}, result, "CancelAlgoOrder should decode every field")
		} else {
			assert.Len(t, result, 1, "CancelAlgoOrder should return a result for the algo order")
		}
	})

	t.Run("partial success reports the failed cancellations", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the partially successful batch is a mocked reply")
		}
		result, err := e.CancelAlgoOrder(contextGenerate(), []CancelAlgoOrderRequest{
			{InstrumentID: mainPair.String(), AlgoID: "90994944"},
			{InstrumentID: mainPair.String(), AlgoClientOrderID: "algo2"},
		})
		require.ErrorIs(t, err, errPartialSuccess, "CancelAlgoOrder must report the partial success")
		exp := []AlgoOrderActionResponse{
			{AlgoID: "90994944"},
			{AlgoClientOrderID: "algo2", StatusCode: 51400, StatusMessage: "Order cancellation failed as the order has been filled, canceled or does not exist."},
		}
		assert.Equal(t, exp, result, "CancelAlgoOrder should decode every algo order's result")
	})
}

func TestAmendAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AmendAlgoOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.AmendAlgoOrder(contextGenerate(), &AmendAlgoOrderRequest{NewSize: 2})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.AmendAlgoOrder(contextGenerate(), &AmendAlgoOrderRequest{
		InstrumentID: perpetualSwapPair.String(),
		NewSize:      2,
	})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *AmendAlgoOrderRequest
			body string
		}{
			{
				name: "required parameters only",
				arg:  &AmendAlgoOrderRequest{InstrumentID: perpetualSwapPair.String(), AlgoClientOrderID: "algo_01"},
				body: `{"instId":"BTC-USDT-SWAP","algoClOrdId":"algo_01"}`,
			},
			{
				name: "every optional parameter",
				arg: &AmendAlgoOrderRequest{
					InstrumentID:                  perpetualSwapPair.String(),
					AlgoID:                        "2510789768709120",
					AlgoClientOrderID:             "algo_01",
					CancelOnFail:                  true,
					RequestID:                     "po103ux",
					NewSize:                       2,
					NewTakeProfitTriggerPrice:     63000,
					NewTakeProfitOrderPrice:       -1,
					NewStopLossTriggerPrice:       57000,
					NewStopLossOrderPrice:         -1,
					NewTakeProfitTriggerPriceType: "last",
					NewStopLossTriggerPriceType:   "mark",
					NewTriggerPrice:               60000,
					NewOrderPrice:                 -1,
					NewTriggerPriceType:           "index",
					AttachAlgoOrders: []AmendAttachedAlgoOrderRequest{{
						NewTakeProfitTriggerPrice: 65000, NewTakeProfitTriggerRatio: 0.3, NewTakeProfitTriggerPriceType: "last", NewTakeProfitOrderPrice: -1,
						NewStopLossTriggerPrice: 55000, NewStopLossTriggerRatio: 0.1, NewStopLossTriggerPriceType: "mark", NewStopLossOrderPrice: -1,
						NewCallbackRatio: 0.05, NewCallbackSpread: 100, NewActivePrice: 61000,
					}},
					AdvancedChaseParameters: []AmendAdvancedChaseRequest{{NewChaseValue: 5, NewMaxChaseValue: 0.2}},
				},
				body: `{"instId":"BTC-USDT-SWAP","algoId":"2510789768709120","algoClOrdId":"algo_01","cxlOnFail":true,"reqId":"po103ux","newSz":"2","newTpTriggerPx":"63000","newTpOrdPx":"-1","newSlTriggerPx":"57000","newSlOrdPx":"-1","newTpTriggerPxType":"last","newSlTriggerPxType":"mark","newTriggerPx":"60000","newOrdPx":"-1","newTriggerPxType":"index","attachAlgoOrds":[{"newTpTriggerPx":"65000","newTpTriggerRatio":"0.3","newTpOrdPx":"-1","newSlTriggerPx":"55000","newSlTriggerRatio":"0.1","newSlOrdPx":"-1","newTpTriggerPxType":"last","newSlTriggerPxType":"mark","newCallbackRatio":"0.05","newCallbackSpread":"100","newActivePx":"61000"}],"advChaseParams":[{"newChaseVal":"5","newMaxChaseVal":"0.2"}]}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/trade/amend-algos", tc.body, algoOrderActionReply)
				_, err := ex.AmendAlgoOrder(t.Context(), tc.arg)
				require.NoError(t, err, "AmendAlgoOrder must not error")
			})
		}
	})

	for _, tc := range []struct {
		name string
		arg  *AmendAlgoOrderRequest
		exp  *AmendAlgoOrderResponse
	}{
		{
			name: "new size",
			arg:  &AmendAlgoOrderRequest{AlgoID: "2510789768709120", InstrumentID: perpetualSwapPair.String(), NewSize: 2},
			exp:  &AmendAlgoOrderResponse{AlgoID: "2510789768709120"},
		},
		{
			name: "trigger order by algo client order ID",
			arg: &AmendAlgoOrderRequest{
				AlgoClientOrderID: "algo_01", InstrumentID: perpetualSwapPair.String(), RequestID: "po103ux", NewTriggerPrice: 25920, NewOrderPrice: marketOrderPrice,
			},
			exp: &AmendAlgoOrderResponse{AlgoID: "2510789768709121", AlgoClientOrderID: "algo_01", RequestID: "po103ux"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
			}
			result, err := e.AmendAlgoOrder(contextGenerate(), tc.arg)
			require.NoError(t, err, "AmendAlgoOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "AmendAlgoOrder should decode every field")
			} else {
				assert.NotEmpty(t, result.AlgoID, "AmendAlgoOrder should return the algo ID")
			}
		})
	}

	t.Run("rejected amendment reports its status", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the rejected amendment is a mocked reply")
		}
		result, err := e.AmendAlgoOrder(contextGenerate(), &AmendAlgoOrderRequest{InstrumentID: perpetualSwapPair.String(), AlgoID: "2510789768709122", NewSize: 3})
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "AmendAlgoOrder must report the failed request")
		exp := &AmendAlgoOrderResponse{AlgoID: "2510789768709122", StatusCode: 51503, StatusMessage: "Your order has already been filled or canceled."}
		assert.Equal(t, exp, result, "AmendAlgoOrder should decode the rejected amendment's status")
	})
}

func TestGetAlgoOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetAlgoOrderDetail(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetAlgoOrderDetail(contextGenerate(), &AlgoOrderDetailRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	t.Run("request query", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var query url.Values
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			query = r.URL.Query()
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{{"algoClOrdId": "algodetail"}})
		}))
		_, err := ex.GetAlgoOrderDetail(t.Context(), &AlgoOrderDetailRequest{AlgoClientOrderID: "algodetail"})
		require.NoError(t, err, "GetAlgoOrderDetail must not error")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, url.Values{"algoClOrdId": {"algodetail"}}, query, "GetAlgoOrderDetail should leave the unset algo ID out")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAlgoOrderDetail(contextGenerate(), &AlgoOrderDetailRequest{AlgoID: "1234231231423"})
	require.NoError(t, err, "GetAlgoOrderDetail must not error")
	if mockTests {
		exp := &AlgoOrderResponse{
			InstrumentType:             instTypeSwap,
			InstrumentID:               perpetualSwapPair.String(),
			Currency:                   currency.USDT,
			OrderID:                    "1884789786215137280",
			OrderIDList:                []string{"1884789786215137280", "1884789786215137282"},
			AlgoID:                     "1234231231423",
			ClientOrderID:              "detail1",
			Size:                       100,
			CloseFraction:              1,
			OrderType:                  orderOCO,
			Side:                       order.Sell,
			PositionSide:               positionSideLong,
			TradeMode:                  TradeModeIsolated,
			State:                      "effective",
			Leverage:                   10,
			TakeProfitTriggerPrice:     63000,
			TakeProfitTriggerPriceType: "last",
			TakeProfitOrderPrice:       -1,
			StopLossTriggerPrice:       57000,
			StopLossTriggerPriceType:   "mark",
			StopLossOrderPrice:         -1,
			ActualSize:                 100,
			ActualPrice:                63000.5,
			Tag:                        "ocotag",
			ActualSide:                 "tp",
			TriggerTime:                types.Time(time.UnixMilli(1728673513447)),
			ReduceOnly:                 "true",
			QuickMarginType:            "manual",
			LastPrice:                  60777.5,
			FailCode:                   "0",
			AlgoClientOrderID:          "algodetail",
			AmendPriceOnTriggerType:    "0",
			AttachedAlgoOrders:         []AttachedAlgoOrder{},
			LinkedOrder:                AlgoLinkedOrder{OrderID: "1884789786215137281"},
			CreationTime:               types.Time(time.UnixMilli(1728552255493)),
			UpdateTime:                 types.Time(time.UnixMilli(1728673513447)),
			IsTradeBorrowMode:          "true",
			TradeQuoteCurrency:         currency.USDT,
		}
		assert.Equal(t, exp, result, "GetAlgoOrderDetail should decode every field")
	} else {
		assert.Equal(t, "1234231231423", result.AlgoID, "GetAlgoOrderDetail should return the requested algo order")
	}
}

func TestGetAlgoOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetAlgoOrderList(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetAlgoOrderList(contextGenerate(), &AlgoOrderListRequest{Limit: 1})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	t.Run("request query", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var query url.Values
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/trade/orders-algo-pending", r.URL.Path, "GetAlgoOrderList should call the pending algo order list")
			mu.Lock()
			query = r.URL.Query()
			mu.Unlock()
			writeOKXData(t, w, []any{})
		}))
		_, err := ex.GetAlgoOrderList(t.Context(), &AlgoOrderListRequest{
			OrderType:         "Conditional",
			AlgoID:            "1753184812254216192",
			AlgoClientOrderID: "algo1",
			InstrumentType:    "spot",
			InstrumentID:      mainPair.String(),
			After:             "1753184812254216193",
			Before:            "1753184812254216191",
			Limit:             100,
		})
		require.NoError(t, err, "GetAlgoOrderList must not error")
		mu.Lock()
		defer mu.Unlock()
		exp := url.Values{
			"ordType": {orderConditional}, "algoId": {"1753184812254216192"}, "algoClOrdId": {"algo1"}, "instType": {instTypeSpot},
			"instId": {"BTC-USDT"}, "after": {"1753184812254216193"}, "before": {"1753184812254216191"}, "limit": {"100"},
		}
		assert.Equal(t, exp, query, "GetAlgoOrderList should send every filter, paging by algo ID")
	})

	created := types.Time(time.UnixMilli(1724751378980))
	for _, tc := range []struct {
		name string
		arg  *AlgoOrderListRequest
		exp  []AlgoOrderResponse
	}{
		{
			name: "conditional orders",
			arg:  &AlgoOrderListRequest{OrderType: orderConditional, Limit: 1},
			exp: []AlgoOrderResponse{{
				InstrumentType: instTypeSpot, InstrumentID: mainPair.String(), OrderIDList: []string{}, AlgoID: "1753184812254216192", Size: 10,
				OrderType: orderConditional, Side: order.Buy, PositionSide: positionSideNet, TradeMode: TradeModeCash, TargetCurrency: targetCurrencyQuote,
				State: "live", TakeProfitTriggerPrice: 10000, TakeProfitTriggerPriceType: "last", TakeProfitOrderPrice: -1, ReduceOnly: "false",
				LastPrice: 62916.5, FailCode: "0", AmendPriceOnTriggerType: "0", AttachedAlgoOrders: []AttachedAlgoOrder{}, CreationTime: created,
				UpdateTime: created, TradeQuoteCurrency: currency.USDT,
			}},
		},
		{
			name: "trigger orders",
			arg:  &AlgoOrderListRequest{OrderType: orderTrigger, InstrumentType: instTypeSwap, InstrumentID: perpetualSwapPair.String()},
			exp: []AlgoOrderResponse{
				{
					InstrumentType: instTypeSwap, InstrumentID: perpetualSwapPair.String(), OrderIDList: []string{}, AlgoID: "1753184812254216193", Size: 1,
					OrderType: orderTrigger, Side: order.Buy, PositionSide: positionSideNet, TradeMode: TradeModeCross, State: "live", Leverage: 10,
					TriggerPrice: 25920, TriggerPriceType: "last", OrderPrice: -1, ReduceOnly: "false", LastPrice: 25000.5, AmendPriceOnTriggerType: "0",
					AttachedAlgoOrders: []AttachedAlgoOrder{{
						AttachAlgoClientOrderID: "attachalgo", TakeProfitTriggerPrice: 30000, TakeProfitTriggerPriceType: "last", TakeProfitOrderPrice: -1,
						StopLossTriggerPrice: 20000, StopLossTriggerPriceType: "index", StopLossOrderPrice: -1,
					}},
					CreationTime: created, UpdateTime: created,
				},
				{
					InstrumentType: instTypeSwap, InstrumentID: perpetualSwapPair.String(), OrderIDList: []string{}, SubAlgoIDList: []string{"1753184812254216195"},
					AlgoID: "1753184812254216194", Size: 1, OrderType: orderTrigger, Side: order.Buy, PositionSide: positionSideNet, TradeMode: TradeModeCross,
					State: "effective", Leverage: 10, TriggerPrice: 25900, TriggerPriceType: "mark", AdvanceOrderType: orderChase,
					AdvancedChaseParameters: []AdvancedChaseParameters{{ChaseType: "distance", ChaseValue: 10, MaxChaseType: "ratio", MaxChaseValue: 0.1}},
					TriggerTime:             types.Time(time.UnixMilli(1724751379980)), ReduceOnly: "false", LastPrice: 25000.5, AmendPriceOnTriggerType: "0",
					AttachedAlgoOrders: []AttachedAlgoOrder{}, CreationTime: created, UpdateTime: created,
				},
			},
		},
		{
			name: "trailing stop orders",
			arg:  &AlgoOrderListRequest{OrderType: orderMoveOrderStop, InstrumentType: "swap"},
			exp: []AlgoOrderResponse{
				{
					InstrumentType: instTypeSwap, InstrumentID: perpetualSwapPair.String(), OrderIDList: []string{}, AlgoID: "1753184812254216196", Size: 1,
					OrderType: orderMoveOrderStop, Side: order.Sell, PositionSide: positionSideNet, TradeMode: TradeModeCross, State: "live", Leverage: 10,
					CallbackRatio: 0.05, ActivePrice: 26000, MoveTriggerPrice: 25500, ReduceOnly: "true", LastPrice: 25000.5, AmendPriceOnTriggerType: "0",
					AttachedAlgoOrders: []AttachedAlgoOrder{}, CreationTime: created, UpdateTime: created,
				},
				{
					InstrumentType: instTypeSwap, InstrumentID: perpetualSwapPair.String(), OrderIDList: []string{}, AlgoID: "1753184812254216197", Size: 1,
					OrderType: orderMoveOrderStop, Side: order.Sell, PositionSide: positionSideNet, TradeMode: TradeModeCross, State: "live", Leverage: 10,
					CallbackSpread: 100, MoveTriggerPrice: 25400, ReduceOnly: "true", LastPrice: 25000.5, AmendPriceOnTriggerType: "0",
					AttachedAlgoOrders: []AttachedAlgoOrder{}, CreationTime: created, UpdateTime: created,
				},
			},
		},
		{
			name: "chase orders",
			arg:  &AlgoOrderListRequest{OrderType: orderChase, InstrumentType: instTypeSwap},
			exp: []AlgoOrderResponse{{
				InstrumentType: instTypeSwap, InstrumentID: perpetualSwapPair.String(), OrderIDList: []string{}, AlgoID: "1753184812254216198", Size: 1,
				OrderType: orderChase, Side: order.Buy, PositionSide: positionSideNet, TradeMode: TradeModeCross, State: "live", Leverage: 10,
				ReduceOnly: "false", LastPrice: 25000.5, AmendPriceOnTriggerType: "0", AttachedAlgoOrders: []AttachedAlgoOrder{}, CreationTime: created,
				UpdateTime: created, ChaseType: "distance", ChaseValue: 10, MaxChaseType: "ratio", MaxChaseValue: 0.1,
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			result, err := e.GetAlgoOrderList(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetAlgoOrderList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAlgoOrderList should decode every field")
			} else {
				assert.NotNil(t, result, "GetAlgoOrderList should return a list")
			}
		})
	}
}

func TestGetAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetAlgoOrderHistory(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetAlgoOrderHistory(contextGenerate(), &AlgoOrderHistoryRequest{State: "effective", Limit: 1})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.GetAlgoOrderHistory(contextGenerate(), &AlgoOrderHistoryRequest{OrderType: orderConditional, Limit: 1})
	require.ErrorIs(t, err, errMissingEitherAlgoIDOrState)

	t.Run("request query", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var query url.Values
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/trade/orders-algo-history", r.URL.Path, "GetAlgoOrderHistory should call the algo order history")
			mu.Lock()
			query = r.URL.Query()
			mu.Unlock()
			writeOKXData(t, w, []any{})
		}))
		_, err := ex.GetAlgoOrderHistory(t.Context(), &AlgoOrderHistoryRequest{
			OrderType:      "OCO",
			State:          "canceled",
			AlgoID:         "1880721064716505088",
			InstrumentType: "swap",
			InstrumentID:   perpetualSwapPair.String(),
			After:          "1880721064716505089",
			Before:         "1880721064716505087",
			Limit:          100,
		})
		require.NoError(t, err, "GetAlgoOrderHistory must not error")
		mu.Lock()
		defer mu.Unlock()
		exp := url.Values{
			"ordType": {orderOCO}, "state": {"canceled"}, "algoId": {"1880721064716505088"}, "instType": {instTypeSwap}, "instId": {"BTC-USDT-SWAP"},
			"after": {"1880721064716505089"}, "before": {"1880721064716505087"}, "limit": {"100"},
		}
		assert.Equal(t, exp, query, "GetAlgoOrderHistory should send every filter, paging by algo ID")
	})

	created := types.Time(time.UnixMilli(1724751378980))
	for _, tc := range []struct {
		name string
		arg  *AlgoOrderHistoryRequest
		exp  []AlgoOrderResponse
	}{
		{
			name: "effective conditional orders",
			arg:  &AlgoOrderHistoryRequest{OrderType: orderConditional, State: "effective", Limit: 1},
			exp: []AlgoOrderResponse{{
				InstrumentType: instTypeSwap, InstrumentID: perpetualSwapPair.String(), OrderID: "1884789786215137280",
				OrderIDList: []string{"1884789786215137280"}, AlgoID: "1880721064716505088", Size: 100, CloseFraction: 1, OrderType: orderConditional,
				Side: order.Sell, PositionSide: positionSideLong, TradeMode: TradeModeIsolated, State: "effective", Leverage: 10,
				TakeProfitTriggerPrice: 63000, TakeProfitTriggerPriceType: "last", TakeProfitOrderPrice: -1, StopLossTriggerPrice: 57000,
				StopLossTriggerPriceType: "mark", StopLossOrderPrice: -1, ActualSize: 100, ActualSide: "tp",
				TriggerTime: types.Time(time.UnixMilli(1728673513447)), ReduceOnly: "true", LastPrice: 60777.5, FailCode: "1",
				AmendPriceOnTriggerType: "0", AttachedAlgoOrders: []AttachedAlgoOrder{}, CreationTime: types.Time(time.UnixMilli(1728552255493)),
				UpdateTime: types.Time(time.UnixMilli(1728673513447)),
			}},
		},
		{
			name: "effective smart iceberg orders",
			arg:  &AlgoOrderHistoryRequest{OrderType: orderSmartIceberg, State: "effective"},
			exp: []AlgoOrderResponse{{
				InstrumentType: instTypeSpot, InstrumentID: mainPair.String(), OrderIDList: []string{}, AlgoID: "1880721064716505089", Size: 100,
				OrderType: orderSmartIceberg, Side: order.Buy, PositionSide: positionSideNet, TradeMode: TradeModeCash, State: "effective",
				SizeLimit: 10, PriceLimit: 95000, LimitOrderNumber: 5, Aggressiveness: "conservative",
				TriggerParameters: []AlgoTriggerRequest{{
					TriggerAction: "start", TriggerStrategy: "rsi", TriggerPrice: 90000, TriggerCondition: "cross_up", Timeframe: "15m", Threshold: 30,
					TimePeriod: "14",
				}},
				LastPrice: 94000, AmendPriceOnTriggerType: "0", AttachedAlgoOrders: []AttachedAlgoOrder{}, CreationTime: created, UpdateTime: created,
				TradeQuoteCurrency: currency.USDT,
			}},
		},
		{
			name: "failed TWAP orders of an instrument",
			arg: &AlgoOrderHistoryRequest{
				OrderType: orderTWAP, State: "order_failed", InstrumentType: instTypeSpot, InstrumentID: mainPair.String(),
				After: "1880721064716505092", Before: "1880721064716505080", Limit: 2,
			},
			exp: []AlgoOrderResponse{
				{
					InstrumentType: instTypeSpot, InstrumentID: mainPair.String(), OrderIDList: []string{}, AlgoID: "1880721064716505090", Size: 6,
					OrderType: orderTWAP, Side: order.Sell, PositionSide: positionSideNet, TradeMode: TradeModeCash, State: "order_failed",
					PriceSpread: 10, SizeLimit: 1, PriceLimit: 95000, TimeInterval: 10, ReduceOnly: "false", LastPrice: 94000, FailCode: "51008",
					AmendPriceOnTriggerType: "0", AttachedAlgoOrders: []AttachedAlgoOrder{}, CreationTime: created, UpdateTime: created,
					TradeQuoteCurrency: currency.USDT,
				},
				{
					InstrumentType: instTypeSpot, InstrumentID: mainPair.String(), OrderIDList: []string{}, AlgoID: "1880721064716505091", Size: 6,
					OrderType: orderTWAP, Side: order.Sell, PositionSide: positionSideNet, TradeMode: TradeModeCash, State: "order_failed",
					PriceVariance: 0.01, SizeLimit: 1, PriceLimit: 95000, TimeInterval: 15, ReduceOnly: "false", LastPrice: 94000, FailCode: "51008",
					AmendPriceOnTriggerType: "0", AttachedAlgoOrders: []AttachedAlgoOrder{}, CreationTime: created, UpdateTime: created,
					TradeQuoteCurrency: currency.USDT,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			result, err := e.GetAlgoOrderHistory(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetAlgoOrderHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAlgoOrderHistory should decode every field")
			} else {
				assert.NotNil(t, result, "GetAlgoOrderHistory should return a list")
			}
		})
	}
}

func TestBatchCancelEndpointsRefuseOversizedLists(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	args := make([]CancelOrderRequest, 21)
	for i := range args {
		args[i] = CancelOrderRequest{InstrumentID: "BTC-USDT", OrderID: strconv.Itoa(i)}
	}
	_, err := e.CancelMultipleOrders(t.Context(), args)
	assert.ErrorIs(t, err, errExceedLimit, "CancelMultipleOrders should refuse more than 20 orders")
	_, err = e.WebsocketAPI().CancelMultipleOrders(t.Context(), args)
	assert.ErrorIs(t, err, errExceedLimit, "WebsocketAPI CancelMultipleOrders should refuse more than 20 orders")
	algoArgs := make([]CancelAlgoOrderRequest, maxCancelAlgosPerRequest+1)
	for i := range algoArgs {
		algoArgs[i] = CancelAlgoOrderRequest{InstrumentID: "BTC-USDT", AlgoID: strconv.Itoa(i)}
	}
	_, err = e.CancelAlgoOrder(t.Context(), algoArgs)
	assert.ErrorIs(t, err, errExceedLimit, "CancelAlgoOrder should refuse more than 10 algo orders")
}

func TestAmendAlgoOrderResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		exp  *AmendAlgoOrderResponse
	}{
		{
			name: "amended",
			data: `{"algoClOrdId":"algo_01","algoId":"2510789768709120","reqId":"po103ux","sCode":"0","sMsg":""}`,
			exp:  &AmendAlgoOrderResponse{AlgoID: "2510789768709120", AlgoClientOrderID: "algo_01", RequestID: "po103ux"},
		},
		{
			name: "rejected",
			data: `{"algoClOrdId":"","algoId":"2510789768709122","reqId":"","sCode":"51503","sMsg":"Your order has already been filled or canceled."}`,
			exp:  &AmendAlgoOrderResponse{AlgoID: "2510789768709122", StatusCode: 51503, StatusMessage: "Your order has already been filled or canceled."},
		},
		{name: "no status code", data: `{"algoId":"2510789768709120"}`, exp: &AmendAlgoOrderResponse{AlgoID: "2510789768709120", statusCodeMissing: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := new(AmendAlgoOrderResponse)
			require.NoError(t, json.Unmarshal([]byte(tc.data), got), "Unmarshal must decode the row")
			assert.Equal(t, tc.exp, got, "Unmarshal should decode every field and whether the row states its status code")
		})
	}
	t.Run("status code which is no integer", func(t *testing.T) {
		t.Parallel()
		assert.ErrorIs(t, json.Unmarshal([]byte(`{"algoId":"1","sCode":"5XXXX"}`), new(AmendAlgoOrderResponse)), strconv.ErrSyntax, "Unmarshal should refuse a status code which is no integer")
	})
}
