package gate

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestPlaceDeliveryOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceDeliveryOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceDeliveryOrder must reject a nil request")

	_, err = e.PlaceDeliveryOrder(t.Context(), &DeliveryOrderCreateRequest{Size: 1, Price: 1, Settle: currency.USDT})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceDeliveryOrder must reject an empty contract")

	pair := getPair(t, asset.DeliveryFutures)
	_, err = e.PlaceDeliveryOrder(t.Context(), &DeliveryOrderCreateRequest{Contract: pair, Size: 1, Price: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceDeliveryOrder must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceDeliveryOrder(t.Context(), &DeliveryOrderCreateRequest{
		Contract:    pair,
		Size:        6024,
		Iceberg:     0,
		Price:       3765,
		Text:        "t-my-custom-id",
		Settle:      currency.USDT,
		TimeInForce: gtcTIF,
	})
	require.NoError(t, err, "PlaceDeliveryOrder must not error")
	if !mockTests {
		assert.NotZero(t, result.ID, "PlaceDeliveryOrder should return an order ID")
		return
	}
	assert.Equal(t, deliveryOpenOrderFixture(pair, 15675394), result, "PlaceDeliveryOrder should decode every field")

	// The recorded request body only matches a market order sending its zero price as "0"
	result, err = e.PlaceDeliveryOrder(t.Context(), &DeliveryOrderCreateRequest{
		Contract:    pair,
		Size:        -1,
		Text:        "t-market-order",
		Settle:      currency.USDT,
		TimeInForce: iocTIF,
	})
	require.NoError(t, err, "PlaceDeliveryOrder must not error for a market order")
	exp := &FuturesOrderResponse{
		ID:                        15675395,
		Contract:                  pair,
		CreateTime:                types.Time(time.Unix(1546569968, 0)),
		Size:                      -1,
		FillPrice:                 67850,
		MakerFeeRate:              -0.00025,
		TakerFeeRate:              0.00075,
		TimeInForce:               iocTIF,
		Text:                      "t-market-order",
		Status:                    "finished",
		FinishTime:                types.Time(time.Unix(1546569969, 0)),
		FinishAs:                  "filled",
		SelfTradePreventionAction: "-",
		AmendText:                 "-",
	}
	assert.Equal(t, exp, result, "PlaceDeliveryOrder should decode a filled market order")
}

func TestGetDeliveryOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryOrders(t.Context(), currency.EMPTYPAIR, statusOpen, currency.EMPTYCODE, "", 0, 0, true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryOrders must reject an empty settlement currency")

	_, err = e.GetDeliveryOrders(t.Context(), getPair(t, asset.DeliveryFutures), "", currency.USDT, "", 0, 0, true)
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetDeliveryOrders must reject an empty status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	pair := getPair(t, asset.DeliveryFutures)
	result, err := e.GetDeliveryOrders(t.Context(), pair, statusOpen, currency.USDT, "", 0, 0, true)
	require.NoError(t, err, "GetDeliveryOrders must not error")
	if mockTests {
		assert.Equal(t, []*FuturesOrderResponse{deliveryOpenOrderFixture(pair, 15675394)}, result, "GetDeliveryOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryOrders should return a list")
}

func TestCancelMultipleDeliveryOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleDeliveryOrders(t.Context(), currency.EMPTYPAIR, "ask", currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelMultipleDeliveryOrders must reject an empty settlement currency")

	_, err = e.CancelMultipleDeliveryOrders(t.Context(), currency.EMPTYPAIR, "ask", currency.USDT)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelMultipleDeliveryOrders must reject an empty contract")

	pair := getPair(t, asset.DeliveryFutures)
	_, err = e.CancelMultipleDeliveryOrders(t.Context(), pair, "buy", currency.USDT)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CancelMultipleDeliveryOrders must reject a side other than ask or bid")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleDeliveryOrders(t.Context(), pair, "ask", currency.USDT)
	require.NoError(t, err, "CancelMultipleDeliveryOrders must not error")
	if mockTests {
		assert.Equal(t, []*FuturesOrderResponse{deliveryCancelledOrderFixture(pair)}, result, "CancelMultipleDeliveryOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelMultipleDeliveryOrders should return a list")
}

func TestGetSingleDeliveryOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleDeliveryOrder(t.Context(), currency.EMPTYCODE, "123456")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSingleDeliveryOrder must reject an empty settlement currency")

	_, err = e.GetSingleDeliveryOrder(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSingleDeliveryOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSingleDeliveryOrder(t.Context(), currency.USDT, "123456")
	require.NoError(t, err, "GetSingleDeliveryOrder must not error")
	if mockTests {
		assert.Equal(t, deliveryOpenOrderFixture(getPair(t, asset.DeliveryFutures), 123456), result, "GetSingleDeliveryOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleDeliveryOrder should return an order")
}

func TestCancelSingleDeliveryOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSingleDeliveryOrder(t.Context(), currency.EMPTYCODE, "123456")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelSingleDeliveryOrder must reject an empty settlement currency")

	_, err = e.CancelSingleDeliveryOrder(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSingleDeliveryOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelSingleDeliveryOrder(t.Context(), currency.USDT, "123456")
	require.NoError(t, err, "CancelSingleDeliveryOrder must not error")
	if mockTests {
		assert.Equal(t, deliveryCancelledOrderFixture(getPair(t, asset.DeliveryFutures)), result, "CancelSingleDeliveryOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelSingleDeliveryOrder should return an order")
}

// deliveryOpenOrderFixture returns the open delivery order recorded in testdata/http.json
func deliveryOpenOrderFixture(contract currency.Pair, id uint64) *FuturesOrderResponse {
	return &FuturesOrderResponse{
		ID:                        id,
		Contract:                  contract,
		CreateTime:                types.Time(time.Unix(1546569968, 0)),
		Size:                      6024,
		RemainingAmount:           6024,
		Price:                     3765,
		MakerFeeRate:              -0.00025,
		TakerFeeRate:              0.00075,
		TimeInForce:               gtcTIF,
		Text:                      "t-my-custom-id",
		Status:                    statusOpen,
		SelfTradePreventionAction: "-",
		AmendText:                 "-",
	}
}

// deliveryCancelledOrderFixture returns the cancelled delivery order recorded in testdata/http.json
func deliveryCancelledOrderFixture(contract currency.Pair) *FuturesOrderResponse {
	return &FuturesOrderResponse{
		ID:                        123456,
		Contract:                  contract,
		CreateTime:                types.Time(time.Unix(1546569968, 0)),
		Size:                      -100,
		RemainingAmount:           -100,
		Price:                     3800,
		MakerFeeRate:              -0.00025,
		TakerFeeRate:              0.00075,
		TimeInForce:               gtcTIF,
		ReferenceUserID:           1001,
		IsReduceOnly:              true,
		Text:                      "t-my-custom-id",
		Status:                    statusFinished,
		FinishTime:                types.Time(time.Unix(1546570000, 0)),
		FinishAs:                  "cancelled",
		SelfTradePreventionID:     7,
		SelfTradePreventionAction: "cn",
		AmendText:                 "t-amended",
	}
}

func TestGetMyDeliveryTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMyDeliveryTradingHistory(t.Context(), currency.EMPTYCODE, "", getPair(t, asset.DeliveryFutures), 0, 0, 1, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMyDeliveryTradingHistory must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMyDeliveryTradingHistory(t.Context(), currency.USDT, "", getPair(t, asset.DeliveryFutures), 0, 0, 1, "")
	require.NoError(t, err, "GetMyDeliveryTradingHistory must not error")
	if mockTests {
		exp := []*TradingHistoryItemResponse{{
			ID:       121234231,
			Contract: "BTC_USDT_20260626",
			Text:     "t-123456",
			Size:     100,
			Price:    100.123,
			Fee:      0.01,
			PointFee: 0.002,
			Role:     "taker",
		}}
		assert.Equal(t, exp, result, "GetMyDeliveryTradingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMyDeliveryTradingHistory should return a list")
}

func TestGetDeliveryPositionCloseHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryPositionCloseHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures), 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryPositionCloseHistory must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDeliveryPositionCloseHistory(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), 0)
	require.NoError(t, err, "GetDeliveryPositionCloseHistory must not error")
	if mockTests {
		exp := []*PositionCloseHistoryResponse{{
			Time:          types.Time(time.Unix(1546487347, 0)),
			ProfitAndLoss: 0.00013,
			Side:          "long",
			Contract:      "BTC_USDT_20260626",
			Text:          "web",
		}}
		assert.Equal(t, exp, result, "GetDeliveryPositionCloseHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryPositionCloseHistory should return a list")
}

func TestGetDeliveryLiquidationHistory(t *testing.T) {
	t.Parallel()
	at, _ := getTime()
	_, err := e.GetDeliveryLiquidationHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures), 0, at)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryLiquidationHistory must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	pair := getPair(t, asset.DeliveryFutures)
	result, err := e.GetDeliveryLiquidationHistory(t.Context(), currency.USDT, pair, 0, at)
	require.NoError(t, err, "GetDeliveryLiquidationHistory must not error")
	if mockTests {
		exp := []*LiquidationHistoryItemResponse{{
			Time:             types.Time(time.Unix(1744103854, 0)),
			Contract:         pair,
			Size:             600,
			Leverage:         25,
			Margin:           0.006705256878,
			EntryPrice:       3536.123,
			MarkPrice:        3420.27,
			OrderPrice:       3405,
			FillPrice:        3424,
			LiquidationPrice: 3421.54,
			OrderID:          317393847,
			RemainingAmount:  4,
		}}
		assert.Equal(t, exp, result, "GetDeliveryLiquidationHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryLiquidationHistory should return a list")
}

func TestGetDeliverySettlementHistory(t *testing.T) {
	t.Parallel()
	at, _ := getTime()
	_, err := e.GetDeliverySettlementHistory(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures), 0, at)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliverySettlementHistory must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	pair := getPair(t, asset.DeliveryFutures)
	result, err := e.GetDeliverySettlementHistory(t.Context(), currency.USDT, pair, 0, at)
	require.NoError(t, err, "GetDeliverySettlementHistory must not error")
	if mockTests {
		exp := []*SettlementHistoryItemResponse{{
			Time:        types.Time(time.Unix(1744103854, 0)),
			Contract:    pair,
			Size:        600,
			Leverage:    25,
			Margin:      0.006705256878,
			EntryPrice:  3536.123,
			SettlePrice: 3421.54,
			Profit:      -6.87498,
			Fee:         0.03079386,
		}}
		assert.Equal(t, exp, result, "GetDeliverySettlementHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliverySettlementHistory should return a list")
}

func TestCreatePriceTriggeredDeliveryOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePriceTriggeredDeliveryOrder(t.Context(), currency.USDT, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreatePriceTriggeredDeliveryOrder must reject a nil request")

	pair := getPair(t, asset.DeliveryFutures)
	newRequest := func() *FuturesPriceTriggeredOrderRequest {
		return &FuturesPriceTriggeredOrderRequest{
			Initial:   FuturesInitial{Contract: pair, Size: 12, Price: 1234},
			Trigger:   FuturesTrigger{Rule: 1, Price: 123400},
			OrderType: "close-short-position",
		}
	}
	for _, tc := range []struct {
		name   string
		settle currency.Code
		modify func(*FuturesPriceTriggeredOrderRequest)
		err    error
	}{
		{name: "settlement currency", settle: currency.EMPTYCODE, modify: func(*FuturesPriceTriggeredOrderRequest) {}, err: currency.ErrCurrencyCodeEmpty},
		{name: "contract", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.Contract = currency.EMPTYPAIR }, err: currency.ErrCurrencyPairEmpty},
		{name: "price", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.Price = -1 }, err: limits.ErrPriceBelowMin},
		{name: "time in force", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.TimeInForce = pocTIF }, err: order.ErrInvalidTimeInForce},
		{name: "strategy type", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.StrategyType = 2 }, err: errInvalidTriggerStrategyType},
		{name: "price type", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.PriceType = 3 }, err: errInvalidTriggerPriceType},
		{name: "rule", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.Rule = 0 }, err: errInvalidTriggerRule},
		{name: "trigger price", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.Price = 0 }, err: limits.ErrPriceBelowMin},
		{name: "trigger expiry", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.ExpireAfter = fractionalSeconds }, err: errNotWholeSeconds},
		{name: "negative trigger expiry", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.ExpireAfter = negativeSeconds }, err: errNotWholeSeconds},
		{name: "read only order type", settle: currency.USDT, modify: func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "close-long-order" }, err: errInvalidAutoOrderType},
	} {
		arg := newRequest()
		tc.modify(arg)
		_, err = e.CreatePriceTriggeredDeliveryOrder(t.Context(), tc.settle, arg)
		require.ErrorIsf(t, err, tc.err, "CreatePriceTriggeredDeliveryOrder must reject an invalid %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		result, err := e.CreatePriceTriggeredDeliveryOrder(t.Context(), currency.USDT, newRequest())
		require.NoError(t, err, "CreatePriceTriggeredDeliveryOrder must not error")
		assert.NotZero(t, result.ID, "CreatePriceTriggeredDeliveryOrder should return an order ID")
		return
	}

	// The initial order's wire encoding belongs to the shared FuturesInitial type, so the request is checked here
	// rather than against a recorded body
	var requests atomic.Int64
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		assert.Equal(t, http.MethodPost, r.Method, "request method should be POST")
		assert.Equal(t, "/api/v4/delivery/usdt/price_orders", r.URL.Path, "request path should be the delivery price orders endpoint")
		var body struct {
			Initial   map[string]any `json:"initial"`
			Trigger   map[string]any `json:"trigger"`
			OrderType string         `json:"order_type"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "request body should decode")
		assert.Equal(t, "close-short-position", body.OrderType, "order type should be sent at the top level")
		assert.Equal(t, map[string]any{"price": "123400", "rule": float64(1)}, body.Trigger, "trigger should carry the price and rule")
		assert.Len(t, body.Initial, 3, "initial order should carry the contract, size and price")
		assert.Equal(t, "BTC_USDT_20260626", body.Initial["contract"], "initial order contract should match")
		assert.Equal(t, "1234", body.Initial["price"], "initial order price should be a decimal string")
		assert.Equal(t, "12", fmt.Sprint(body.Initial["size"]), "initial order size should match")
		_, err := w.Write([]byte(`{"id":1432329}`))
		assert.NoError(t, err, "writing the price order response should not error")
	}))
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	result, err := ex.CreatePriceTriggeredDeliveryOrder(t.Context(), currency.USDT, newRequest())
	require.NoError(t, err, "CreatePriceTriggeredDeliveryOrder must not error")
	assert.Equal(t, &TriggerOrderIDResponse{ID: 1432329}, result, "CreatePriceTriggeredDeliveryOrder should decode the order ID")
	assert.Equal(t, int64(1), requests.Load(), "CreatePriceTriggeredDeliveryOrder should send one request")
}

func TestGetDeliveryAllAutoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryAllAutoOrder(t.Context(), "", currency.USDT, getPair(t, asset.DeliveryFutures), 0, 1)
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetDeliveryAllAutoOrder must reject an empty status")

	_, err = e.GetDeliveryAllAutoOrder(t.Context(), statusOpen, currency.EMPTYCODE, getPair(t, asset.DeliveryFutures), 0, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryAllAutoOrder must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	pair := getPair(t, asset.DeliveryFutures)
	result, err := e.GetDeliveryAllAutoOrder(t.Context(), statusOpen, currency.USDT, pair, 0, 1)
	require.NoError(t, err, "GetDeliveryAllAutoOrder must not error")
	if mockTests {
		assert.Equal(t, []*PriceTriggeredOrderResponse{deliveryOpenAutoOrderFixture(pair, 1283293)}, result, "GetDeliveryAllAutoOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryAllAutoOrder should return a list")
}

func TestCancelAllDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllDeliveryPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelAllDeliveryPriceTriggeredOrder must reject an empty settlement currency")

	_, err = e.CancelAllDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllDeliveryPriceTriggeredOrder must reject an empty contract")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	pair := getPair(t, asset.DeliveryFutures)
	result, err := e.CancelAllDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, pair)
	require.NoError(t, err, "CancelAllDeliveryPriceTriggeredOrder must not error")
	if mockTests {
		assert.Equal(t, []*PriceTriggeredOrderResponse{deliveryCancelledAutoOrderFixture(pair)}, result, "CancelAllDeliveryPriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelAllDeliveryPriceTriggeredOrder should return a list")
}

func TestGetSingleDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleDeliveryPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSingleDeliveryPriceTriggeredOrder must reject an empty settlement currency")

	_, err = e.GetSingleDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSingleDeliveryPriceTriggeredOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSingleDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, "12345")
	require.NoError(t, err, "GetSingleDeliveryPriceTriggeredOrder must not error")
	if mockTests {
		assert.Equal(t, deliveryOpenAutoOrderFixture(getPair(t, asset.DeliveryFutures), 12345), result, "GetSingleDeliveryPriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleDeliveryPriceTriggeredOrder should return an order")
}

func TestCancelDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelDeliveryPriceTriggeredOrder(t.Context(), currency.EMPTYCODE, "12345")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelDeliveryPriceTriggeredOrder must reject an empty settlement currency")

	_, err = e.CancelDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelDeliveryPriceTriggeredOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, "12345")
	require.NoError(t, err, "CancelDeliveryPriceTriggeredOrder must not error")
	if mockTests {
		assert.Equal(t, deliveryCancelledAutoOrderFixture(getPair(t, asset.DeliveryFutures)), result, "CancelDeliveryPriceTriggeredOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelDeliveryPriceTriggeredOrder should return an order")
}

// deliveryOpenAutoOrderFixture returns the open delivery price-triggered order recorded in testdata/http.json
func deliveryOpenAutoOrderFixture(contract currency.Pair, id uint64) *PriceTriggeredOrderResponse {
	o := &PriceTriggeredOrderResponse{
		ID:         id,
		User:       1234,
		CreateTime: types.Time(time.Unix(1514764800, 0)),
		Status:     statusOpen,
		OrderType:  "close-long-position",
	}
	o.Initial.Contract = contract
	o.Initial.Size = 100
	o.Initial.Price = 5.03
	o.Trigger.PriceType = 1
	o.Trigger.Price = 3000
	o.Trigger.Rule = 1
	o.Trigger.ExpireAfter = types.DurationSeconds(24 * time.Hour)
	return o
}

// deliveryCancelledAutoOrderFixture returns the cancelled delivery price-triggered order recorded in testdata/http.json
func deliveryCancelledAutoOrderFixture(contract currency.Pair) *PriceTriggeredOrderResponse {
	o := deliveryOpenAutoOrderFixture(contract, 12345)
	o.FinishTime = types.Time(time.Unix(1514764900, 0))
	o.TradeID = 13566
	o.Status = statusFinished
	o.FinishAs = "cancelled"
	o.Reason = "cancelled by user"
	return o
}

func TestGetAllDeliveryContracts(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllDeliveryContracts(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAllDeliveryContracts must reject an empty settlement currency")

	r, err := e.GetAllDeliveryContracts(t.Context(), currency.USDT)
	require.NoError(t, err, "GetAllDeliveryContracts must not error")
	if mockTests {
		require.Len(t, r, 6, "GetAllDeliveryContracts must return every recorded contract")
		exp := &DeliveryContractResponse{
			Name:                "BTC_USDT_20260626",
			Underlying:          "BTC_USDT",
			Cycle:               "BI-WEEKLY",
			Type:                "direct",
			QuantoMultiplier:    10,
			MarkType:            "index",
			LastPrice:           0.09043,
			MarkPrice:           0.0904,
			IndexPrice:          0.090395,
			BasisRate:           0.005653,
			BasisValue:          0.00001,
			BasisImpactValue:    1000,
			SettlePriceInterval: types.DurationSeconds(time.Minute),
			SettlePriceDuration: types.DurationSeconds(30 * time.Minute),
			SettleFeeRate:       0.00015,
			OrderPriceRound:     0.00001,
			MarkPriceRound:      0.00001,
			LeverageMin:         1,
			LeverageMax:         50,
			MaintenanceRate:     0.01,
			RiskLimitBase:       500000,
			RiskLimitStep:       500000,
			RiskLimitMax:        5000000,
			MakerFeeRate:        -0.00015,
			TakerFeeRate:        0.00025,
			RefRebateRate:       0.2,
			OrderPriceDeviate:   0.5,
			OrderSizeMin:        1,
			OrderSizeMax:        1000000,
			OrdersLimit:         50,
			OrderbookID:         2981897,
			TradeID:             74916,
			TradeSize:           530433,
			PositionSize:        106,
			ExpireTime:          types.Time(time.Unix(1775808000, 0)),
			ConfigChangeTime:    types.Time(time.Unix(1774597501, 0)),
		}
		assert.Equal(t, exp, r[0], "GetAllDeliveryContracts should decode every field")
	} else {
		assert.NotEmpty(t, r, "GetAllDeliveryContracts should return data")
	}
	r, err = e.GetAllDeliveryContracts(t.Context(), currency.BTC)
	require.NoError(t, err, "GetAllDeliveryContracts must not error")
	// The test below will fail if support for BTC settlement is added. This is intentional, as it ensures we are alerted when it's time to reintroduce support
	if !assert.Empty(t, r, "GetAllDeliveryContracts should not return any data with unsupported settlement currency BTC") {
		t.Error("BTC settlement for delivery futures appears to be supported again by the API. Please raise an issue to reintroduce BTC support for this exchange")
	}
}

func TestGetDeliveryContract(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryContract(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryContract must reject an empty settlement currency")

	_, err = e.GetDeliveryContract(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetDeliveryContract must reject an empty contract")

	result, err := e.GetDeliveryContract(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures))
	require.NoError(t, err, "GetDeliveryContract must not error")
	if mockTests {
		exp := &DeliveryContractResponse{
			Name:                "BTC_USDT_20260626",
			Underlying:          "BTC_USDT",
			Cycle:               "BI-QUARTERLY",
			Type:                "direct",
			QuantoMultiplier:    0.0001,
			MarkType:            "index",
			LastPrice:           67857.5,
			MarkPrice:           67852.21,
			IndexPrice:          67503.92,
			BasisRate:           0.022966,
			BasisValue:          348.29,
			BasisImpactValue:    1000,
			SettlePriceInterval: types.DurationSeconds(time.Minute),
			SettlePriceDuration: types.DurationSeconds(30 * time.Minute),
			SettleFeeRate:       0.00015,
			OrderPriceRound:     0.1,
			MarkPriceRound:      0.01,
			LeverageMin:         1,
			LeverageMax:         100,
			MaintenanceRate:     0.005,
			RiskLimitBase:       1000000,
			RiskLimitStep:       1000000,
			RiskLimitMax:        8000000,
			MakerFeeRate:        -0.00015,
			TakerFeeRate:        0.00025,
			RefRebateRate:       0.2,
			OrderPriceDeviate:   0.5,
			OrderSizeMin:        1,
			OrderSizeMax:        1000000,
			OrdersLimit:         50,
			OrderbookID:         175093732,
			TradeID:             1017842,
			TradeSize:           6032055,
			PositionSize:        1257,
			ExpireTime:          types.Time(time.Unix(1782460800, 0)),
			ConfigChangeTime:    types.Time(time.Unix(1766735467, 0)),
		}
		assert.Equal(t, exp, result, "GetDeliveryContract should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryContract should return a contract")
}

func TestGetDeliveryOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryOrderbook(t.Context(), currency.EMPTYCODE, "0", getPair(t, asset.DeliveryFutures), 0, false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryOrderbook must reject an empty settlement currency")

	_, err = e.GetDeliveryOrderbook(t.Context(), currency.USDT, "0", currency.EMPTYPAIR, 0, false)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetDeliveryOrderbook must reject an empty contract")

	result, err := e.GetDeliveryOrderbook(t.Context(), currency.USDT, "0", getPair(t, asset.DeliveryFutures), 0, false)
	require.NoError(t, err, "GetDeliveryOrderbook must not error")
	if mockTests {
		exp := &OrderbookResponse{
			ResponseTime: types.Time(time.UnixMilli(1775423963401)),
			UpdateTime:   types.Time(time.UnixMilli(1775423963338)),
			Bids:         OrderbookLevels{{Price: 67853.9, Amount: 1}, {Price: 67848.6, Amount: 3}, {Price: 67848.1, Amount: 1007}, {Price: 67837.9, Amount: 2706}, {Price: 67828.2, Amount: 3}},
			Asks:         OrderbookLevels{{Price: 67859.1, Amount: 3}, {Price: 67868.6, Amount: 915}, {Price: 67876.1, Amount: 3}, {Price: 67878.8, Amount: 2714}, {Price: 67890, Amount: 3}},
		}
		assert.Equal(t, exp, result, "GetDeliveryOrderbook should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryOrderbook should return an orderbook")
}

func TestGetDeliveryTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryTradingHistory(t.Context(), currency.EMPTYCODE, "", getPair(t, asset.DeliveryFutures), 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryTradingHistory must reject an empty settlement currency")

	_, err = e.GetDeliveryTradingHistory(t.Context(), currency.USDT, "", currency.EMPTYPAIR, 0, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetDeliveryTradingHistory must reject an empty contract")

	result, err := e.GetDeliveryTradingHistory(t.Context(), currency.USDT, "", getPair(t, asset.DeliveryFutures), 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetDeliveryTradingHistory must not error")
	if mockTests {
		require.Len(t, result, 5, "GetDeliveryTradingHistory must return every recorded trade")
		exp := &TradingHistoryItemResponse{
			ID:                     1017843,
			CreateTime:             types.Time(time.UnixMilli(1775423963713)),
			CreateTimeMilliseconds: types.Time(time.UnixMilli(1775423963713)),
			Contract:               "BTC_USDT_20260626",
			Size:                   -1,
			Price:                  67854,
		}
		assert.Equal(t, exp, result[0], "GetDeliveryTradingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryTradingHistory should return a list")
}

func TestGetDeliveryFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryFuturesCandlesticks(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures), time.Time{}, time.Time{}, 0, kline.OneWeek)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryFuturesCandlesticks must reject an empty settlement currency")

	_, err = e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, kline.OneWeek)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetDeliveryFuturesCandlesticks must reject an empty contract")

	startTime, endTime := getTime()
	_, err = e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), startTime, time.Time{}, 10, kline.OneWeek)
	require.ErrorIs(t, err, errLimitWithTimeRange, "GetDeliveryFuturesCandlesticks must reject a limit with a start time")

	_, err = e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), time.Time{}, endTime, 10, kline.OneWeek)
	require.ErrorIs(t, err, errLimitWithTimeRange, "GetDeliveryFuturesCandlesticks must reject a limit with an end time")

	_, err = e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), endTime, startTime, 0, kline.OneWeek)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDeliveryFuturesCandlesticks must reject a start after the end")

	_, err = e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), startTime, endTime, 0, kline.Interval(5))
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetDeliveryFuturesCandlesticks must reject an unsupported interval")

	deliveryFuturesPair := getPair(t, asset.DeliveryFutures)
	if !mockTests {
		endTime = time.Now().UTC()
		startTime = endTime.Add(-time.Hour)
	}
	result, err := e.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, deliveryFuturesPair, startTime, endTime, 0, kline.OneMin)
	require.NoError(t, err, "GetDeliveryFuturesCandlesticks must not error")
	if mockTests {
		require.Len(t, result, 5, "GetDeliveryFuturesCandlesticks must return every recorded candle")
		exp := &FuturesCandlestickResponse{
			Timestamp:    types.Time(time.Unix(1744103820, 0)),
			Volume:       42,
			ClosePrice:   79897.8,
			HighestPrice: 79915.1,
			LowestPrice:  79897.5,
			OpenPrice:    79915.1,
		}
		assert.Equal(t, exp, result[0], "GetDeliveryFuturesCandlesticks should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryFuturesCandlesticks should return a list")
}

func TestGetDeliveryFutureTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryFutureTickers(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryFutureTickers must reject an empty settlement currency")

	results, err := e.GetDeliveryFutureTickers(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.NoError(t, err, "GetDeliveryFutureTickers must not error")
	if mockTests {
		require.Len(t, results, 5, "GetDeliveryFutureTickers must return every recorded ticker")
		exp := &FuturesTickerResponse{
			Contract:           currency.NewPairWithDelimiter("ARB", "USDT_20260410", currency.UnderscoreDelimiter),
			ChangePercentage:   -2.36,
			Last:               0.091,
			Low24Hour:          0.0893,
			High24Hour:         0.0933,
			TotalSize:          655,
			Volume24Hour:       103971,
			Volume24HourBase:   103971,
			Volume24HourQuote:  9461,
			Volume24HourSettle: 9461,
			MarkPrice:          0.091,
			IndexPrice:         0.0911,
			LowestAsk:          0.0911,
			HighestBid:         0.0909,
		}
		assert.Equal(t, exp, results[0], "GetDeliveryFutureTickers should decode every field")
		return
	}
	assert.NotNil(t, results, "GetDeliveryFutureTickers should return a list")
}

func TestGetDeliveryInsuranceBalanceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryInsuranceBalanceHistory(t.Context(), currency.EMPTYCODE, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryInsuranceBalanceHistory must reject an empty settlement currency")

	result, err := e.GetDeliveryInsuranceBalanceHistory(t.Context(), currency.USDT, 2)
	require.NoError(t, err, "GetDeliveryInsuranceBalanceHistory must not error")
	if mockTests {
		exp := []*InsuranceBalanceResponse{
			{Timestamp: types.Time(time.Unix(1791849600, 0)), Balance: 103102.59316754},
			{Timestamp: types.Time(time.Unix(1791763200, 0)), Balance: 103102.41956754},
		}
		assert.Equal(t, exp, result, "GetDeliveryInsuranceBalanceHistory should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetDeliveryInsuranceBalanceHistory should return balances")
}

func TestGetDeliveryFuturesAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryFuturesAccounts(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryFuturesAccounts must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDeliveryFuturesAccounts(t.Context(), currency.USDT)
	require.NoError(t, err, "GetDeliveryFuturesAccounts must not error")
	if mockTests {
		exp := &FuturesAccountResponse{
			User:                   1666,
			Currency:               currency.USDT,
			Total:                  9707.803567115145,
			UnrealisedPNL:          3371.248828,
			PositionMargin:         38.712189181,
			OrderMargin:            0.5,
			Available:              9669.091377934145,
			Point:                  0.25,
			EnabledEvolvedClassic:  true,
			CrossInitialMargin:     61855.56788525,
			CrossUnrealisedPNL:     1501.178222634128,
			IsolatedPositionMargin: 38.712189181,
		}
		assert.Equal(t, exp, result, "GetDeliveryFuturesAccounts should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryFuturesAccounts should return an account")
}

func TestGetDeliveryAccountBooks(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetDeliveryAccountBooks(t.Context(), currency.EMPTYCODE, 0, startTime, endTime, "dnw")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDeliveryAccountBooks must reject an empty settlement currency")

	_, err = e.GetDeliveryAccountBooks(t.Context(), currency.USDT, 0, endTime, startTime, "dnw")
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDeliveryAccountBooks must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDeliveryAccountBooks(t.Context(), currency.USDT, 0, startTime, endTime, "dnw")
	require.NoError(t, err, "GetDeliveryAccountBooks must not error")
	if mockTests {
		exp := []*AccountBookItemResponse{{
			Time:    types.Time(time.UnixMicro(1744110000123456)),
			Change:  10000,
			Balance: 10004.59316525194,
			Text:    "deposit",
			Type:    "dnw",
		}}
		assert.Equal(t, exp, result, "GetDeliveryAccountBooks should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDeliveryAccountBooks should return a list")
}

func TestGetAllDeliveryPositionsOfUser(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllDeliveryPositionsOfUser(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAllDeliveryPositionsOfUser must reject an empty settlement currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllDeliveryPositionsOfUser(t.Context(), currency.USDT)
	require.NoError(t, err, "GetAllDeliveryPositionsOfUser must not error")
	if mockTests {
		assert.Equal(t, []*PositionResponse{deliveryPositionFixture(getPair(t, asset.DeliveryFutures))}, result, "GetAllDeliveryPositionsOfUser should decode every position")
		return
	}
	assert.NotNil(t, result, "GetAllDeliveryPositionsOfUser should return a list")
}

func TestGetSingleDeliveryPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleDeliveryPosition(t.Context(), currency.EMPTYCODE, getPair(t, asset.DeliveryFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSingleDeliveryPosition must reject an empty settlement currency")

	_, err = e.GetSingleDeliveryPosition(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSingleDeliveryPosition must reject an empty contract")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	pair := getPair(t, asset.DeliveryFutures)
	result, err := e.GetSingleDeliveryPosition(t.Context(), currency.USDT, pair)
	require.NoError(t, err, "GetSingleDeliveryPosition must not error")
	if mockTests {
		assert.Equal(t, deliveryPositionFixture(pair), result, "GetSingleDeliveryPosition should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleDeliveryPosition should return a position")
}

func TestUpdateDeliveryPositionMargin(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateDeliveryPositionMargin(t.Context(), currency.EMPTYCODE, 0.001, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdateDeliveryPositionMargin must reject an empty settlement currency")

	_, err = e.UpdateDeliveryPositionMargin(t.Context(), currency.USDT, 0.001, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateDeliveryPositionMargin must reject an empty contract")

	pair := getPair(t, asset.DeliveryFutures)
	_, err = e.UpdateDeliveryPositionMargin(t.Context(), currency.USDT, 0, pair)
	require.ErrorIs(t, err, errInvalidMarginChange, "UpdateDeliveryPositionMargin must reject a zero change")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	// A negative change removes margin, which Gate documents alongside a positive change adding it
	result, err := e.UpdateDeliveryPositionMargin(t.Context(), currency.USDT, -0.001, pair)
	require.NoError(t, err, "UpdateDeliveryPositionMargin must not error")
	if mockTests {
		exp := deliveryPositionFixture(pair)
		exp.Margin = 4.430548146258
		assert.Equal(t, exp, result, "UpdateDeliveryPositionMargin should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateDeliveryPositionMargin should return a position")
}

func TestUpdateDeliveryPositionLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateDeliveryPositionLeverage(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdateDeliveryPositionLeverage must reject an empty settlement currency")

	_, err = e.UpdateDeliveryPositionLeverage(t.Context(), currency.USDT, currency.EMPTYPAIR, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateDeliveryPositionLeverage must reject an empty contract")

	_, err = e.UpdateDeliveryPositionLeverage(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), -1)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported, "UpdateDeliveryPositionLeverage must reject a negative leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	pair := getPair(t, asset.DeliveryFutures)
	result, err := e.UpdateDeliveryPositionLeverage(t.Context(), currency.USDT, pair, 1)
	require.NoError(t, err, "UpdateDeliveryPositionLeverage must not error")
	if mockTests {
		exp := deliveryPositionFixture(pair)
		exp.Leverage = 1
		assert.Equal(t, exp, result, "UpdateDeliveryPositionLeverage should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateDeliveryPositionLeverage should return a position")
}
