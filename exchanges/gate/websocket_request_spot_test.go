package gate

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/asset"
	"github.com/thrasher-corp/gocryptotrader/exchange/order"
	"github.com/thrasher-corp/gocryptotrader/exchange/request"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// wsAPISpotOrder returns the spot order the WebSocket API fixtures return, after applying changes
func wsAPISpotOrder(changes func(*WebsocketOrderResponse)) *WebsocketOrderResponse {
	o := &WebsocketOrderResponse{
		ID:                        "1700664330",
		Text:                      "t-my-custom-id",
		AmendText:                 "-",
		CreateTime:                types.Time(time.Unix(1681986204, 0)),
		UpdateTime:                types.Time(time.Unix(1681986204, 0)),
		CreateTimeMs:              types.Time(time.UnixMilli(1681986204832)),
		UpdateTimeMs:              types.Time(time.UnixMilli(1681986204832)),
		Status:                    "open",
		CurrencyPair:              BTCUSDT,
		Type:                      "limit",
		Account:                   "spot",
		Side:                      "buy",
		Amount:                    1,
		Price:                     1,
		TimeInForce:               "gtc",
		Left:                      1,
		FeeCurrency:               currency.BTC,
		GTTakerFee:                0.0015,
		GTDiscount:                true,
		RebatedFeeCurrency:        currency.USDT,
		SelfTradePreventionID:     1,
		SelfTradePreventionAction: "cn",
		FinishAs:                  "open",
	}
	if changes != nil {
		changes(o)
	}
	return o
}

func TestWebsocketLogin(t *testing.T) {
	t.Parallel()

	t.Run("failed login replies", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.API.AuthenticatedSupport = true
		ex.API.AuthenticatedWebsocketSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})

		// Gate's documented reply to a login with an invalid key
		reply := `{"request_id":"xxxx","header":{"response_time":"1677816784084","status":"401","channel":"spot.login","event":"api","client_id":"::1-0x14002ba2300"},"data":{"errs":{"label":"INVALID_KEY","message":"Invalid key provided"}}}`
		err := ex.websocketLogin(t.Context(), &replyConnection{reply: []byte(reply)}, "spot.login")
		assert.EqualError(t, err, "INVALID_KEY: Invalid key provided", "websocketLogin should return Gate's error label and message")

		err = ex.websocketLogin(t.Context(), &replyConnection{reply: []byte(`null`)}, "spot.login")
		assert.ErrorIs(t, err, common.ErrNoResponse, "websocketLogin should reject a null reply")

		ctx := t.Context()
		// The cases below share the exchange and run in order
		for data, exp := range map[string]string{
			`{"errs":{"label":"INVALID_KEY"}}`:            "INVALID_KEY",
			`{"errs":{"message":"Invalid key provided"}}`: "Invalid key provided",
		} {
			t.Run(exp, func(t *testing.T) {
				reply := `{"request_id":"xxxx","header":{"response_time":"1677816784084","status":"401","channel":"spot.login","event":"api"},"data":` + data + `}`
				err := ex.websocketLogin(ctx, &replyConnection{reply: []byte(reply)}, "spot.login")
				assert.ErrorContainsf(t, err, exp, "websocketLogin should return what Gate sends of its error for data %s", data)
			})
		}
		for _, tc := range []struct {
			name string
			data string
		}{
			{name: "null data", data: `null`},
			{name: "empty data", data: `{}`},
			{name: "null errs", data: `{"errs":null}`},
			{name: "empty errs", data: `{"errs":{}}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				reply := `{"request_id":"xxxx","header":{"response_time":"1677816784084","status":"401","channel":"spot.login","event":"api"},"data":` + tc.data + `}`
				err := ex.websocketLogin(ctx, &replyConnection{reply: []byte(reply)}, "spot.login")
				assert.ErrorIsf(t, err, common.ErrNoResponse, "websocketLogin should reject a failed reply whose data is %s", tc.data)
			})
		}
	})

	err := e.websocketLogin(t.Context(), nil, "")
	require.ErrorIs(t, err, common.ErrNilPointer)

	ex := newWsAPITestExchange(t, asset.Spot)
	c, err := ex.Websocket.GetConnection(asset.Spot)
	require.NoError(t, err, "GetConnection must not error")

	err = ex.websocketLogin(t.Context(), c, "")
	require.ErrorIs(t, err, errChannelEmpty)

	require.NoError(t, ex.websocketLogin(t.Context(), c, "spot.login"), "websocketLogin must not error")
}

// replyConnection answers every request with the reply it holds
type replyConnection struct {
	websocket.Connection
	reply []byte
}

func (c *replyConnection) SendMessageReturnResponse(context.Context, request.EndpointLimit, any, any) ([]byte, error) {
	return c.reply, nil
}

func TestWebsocketSpotSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSpotSubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketSpotSubmitOrder must reject a nil order")
	_, err = e.WebsocketSpotSubmitOrder(t.Context(), &CreateOrderRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	out := &CreateOrderRequest{}
	_, err = e.WebsocketSpotSubmitOrder(t.Context(), out)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	out.CurrencyPair = getPair(t, asset.Spot)
	_, err = e.WebsocketSpotSubmitOrder(t.Context(), out)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	out.Side = order.Sell.Lower()
	_, err = e.WebsocketSpotSubmitOrder(t.Context(), out)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	out.Amount = 1
	out.Type = "limit"
	_, err = e.WebsocketSpotSubmitOrder(t.Context(), out)
	require.ErrorIs(t, err, order.ErrPriceMustBeSetIfLimitOrder)

	ex := newWsAPITestExchange(t, asset.Spot, canManipulateRealOrders)
	got, err := ex.WebsocketSpotSubmitOrder(t.Context(), &CreateOrderRequest{Text: "t-my-custom-id", CurrencyPair: BTCUSDT, Type: "limit", Account: asset.Spot, Side: "buy", Amount: 1, Price: 1})
	require.NoError(t, err, "WebsocketSpotSubmitOrder must not error")
	if mockTests {
		assert.Equal(t, wsAPISpotOrder(nil), got, "WebsocketSpotSubmitOrder should decode every field")

		_, err = ex.WebsocketSpotSubmitOrder(t.Context(), &CreateOrderRequest{Text: "t-rejected", CurrencyPair: BTCUSDT, Type: "limit", Account: asset.Spot, Side: "buy", Amount: 1, Price: 1})
		assert.ErrorContains(t, err, "BALANCE_NOT_ENOUGH", "WebsocketSpotSubmitOrder should surface a rejected order's error")
		return
	}
	assert.NotEmpty(t, got, "WebsocketSpotSubmitOrder should return an order")
}

func TestWebsocketSpotSubmitOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSpotSubmitOrders(t.Context(), &CreateOrderRequest{CurrencyPair: currency.NewBTCUSDT(), Side: "buy", Amount: 1, Type: "market"}, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketSpotSubmitOrders must reject a nil order in a batch")
	_, err = e.WebsocketSpotSubmitOrders(t.Context())
	require.ErrorIs(t, err, errOrdersEmpty)

	out := &CreateOrderRequest{}
	_, err = e.WebsocketSpotSubmitOrders(t.Context(), out)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	out.CurrencyPair = currency.NewBTCUSDT()
	_, err = e.WebsocketSpotSubmitOrders(t.Context(), out)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	out.Side = order.Buy.Lower()
	_, err = e.WebsocketSpotSubmitOrders(t.Context(), out)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	out.Amount = 0.0003
	out.Type = "limit"
	_, err = e.WebsocketSpotSubmitOrders(t.Context(), out)
	require.ErrorIs(t, err, order.ErrPriceMustBeSetIfLimitOrder)

	// Gate takes up to 10 orders for each of up to 4 currency pairs in one batch
	batch := make([]*CreateOrderRequest, 11)
	for i := range batch {
		batch[i] = &CreateOrderRequest{CurrencyPair: BTCUSDT, Type: "limit", Side: "buy", Amount: 1, Price: 1}
	}
	_, err = e.WebsocketSpotSubmitOrders(t.Context(), batch...)
	require.ErrorIs(t, err, errTooManyBatchOrders, "WebsocketSpotSubmitOrders must refuse more than 10 orders for a currency pair")
	for i, base := range []currency.Code{currency.ETH, currency.LTC, currency.XRP, currency.SOL} {
		batch[i] = &CreateOrderRequest{CurrencyPair: currency.NewPairWithDelimiter(base.String(), "USDT", currency.UnderscoreDelimiter), Type: "limit", Side: "buy", Amount: 1, Price: 1}
	}
	_, err = e.WebsocketSpotSubmitOrders(t.Context(), batch[:5]...)
	require.ErrorIs(t, err, errTooManyBatchOrders, "WebsocketSpotSubmitOrders must refuse more than 4 currency pairs")
	// The limits themselves pass the check, so an unconnected instance fails only at sending
	unconnected := new(Exchange)
	require.NoError(t, testexch.Setup(unconnected), "Setup must not error")
	_, err = unconnected.WebsocketSpotSubmitOrders(t.Context(), batch[:4]...)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketSpotSubmitOrders should allow 4 currency pairs and fail only at sending")
	for i := range batch {
		batch[i] = &CreateOrderRequest{CurrencyPair: BTCUSDT, Type: "limit", Side: "buy", Amount: 1, Price: 1}
	}
	_, err = unconnected.WebsocketSpotSubmitOrders(t.Context(), batch[:10]...)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketSpotSubmitOrders should allow 10 orders for a currency pair and fail only at sending")

	ex := newWsAPITestExchange(t, asset.Spot, canManipulateRealOrders)
	first := &CreateOrderRequest{Text: "t-my-custom-id", CurrencyPair: BTCUSDT, Type: "limit", Account: asset.Spot, Side: "buy", Amount: 1, Price: 1}
	second := &CreateOrderRequest{Text: "t-my-other-id", CurrencyPair: BTCUSDT, Type: "limit", Account: asset.Spot, Side: "sell", Amount: 2, Price: 3}
	got, err := ex.WebsocketSpotSubmitOrders(t.Context(), first, second)
	require.NoError(t, err, "WebsocketSpotSubmitOrders must not error")
	if mockTests {
		exp := []*WebsocketOrderResponse{
			wsAPISpotOrder(nil),
			wsAPISpotOrder(func(o *WebsocketOrderResponse) {
				o.ID, o.Text, o.Side, o.Amount, o.Price, o.Left = "1700664331", "t-my-other-id", "sell", 2, 3, 2
			}),
		}
		assert.Equal(t, exp, got, "WebsocketSpotSubmitOrders should decode every order")
		return
	}
	assert.NotEmpty(t, got, "WebsocketSpotSubmitOrders should return orders")
}

func TestWebsocketSpotCancelOrder(t *testing.T) {
	t.Parallel()

	t.Run("unconnected websocket", func(t *testing.T) {
		t.Parallel()
		unconnected := new(Exchange)
		require.NoError(t, testexch.Setup(unconnected), "Setup must not error")
		_, err := unconnected.WebsocketSpotCancelOrder(t.Context(), "1", currency.NewBTCUSDT(), "spot")
		assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketSpotCancelOrder should return the error of a cancellation it cannot send")
	})

	_, err := e.WebsocketSpotCancelOrder(t.Context(), "", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.WebsocketSpotCancelOrder(t.Context(), "1337", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	ex := newWsAPITestExchange(t, asset.Spot, canManipulateRealOrders)
	got, err := ex.WebsocketSpotCancelOrder(t.Context(), "1700664330", BTCUSDT, "")
	require.NoError(t, err, "WebsocketSpotCancelOrder must not error")
	if mockTests {
		exp := wsAPISpotOrder(func(o *WebsocketOrderResponse) {
			o.Status, o.FinishAs = "cancelled", "cancelled"
			o.UpdateTime, o.UpdateTimeMs = types.Time(time.Unix(1681986206, 0)), types.Time(time.UnixMilli(1681986206330))
		})
		assert.Equal(t, exp, got, "WebsocketSpotCancelOrder should decode every field")

		_, err = ex.WebsocketSpotCancelOrder(t.Context(), "1700664331", BTCUSDT, "")
		assert.ErrorIs(t, err, common.ErrNoResponse, "WebsocketSpotCancelOrder should reject a null result")
		return
	}
	assert.NotEmpty(t, got, "WebsocketSpotCancelOrder should return the cancelled order")
}

func TestWebsocketSpotCancelAllOrdersByIDs(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSpotCancelAllOrdersByIDs(t.Context(), []WebsocketOrderBatchRequest{})
	require.ErrorIs(t, err, errNoOrdersToCancel)

	out := WebsocketOrderBatchRequest{}
	_, err = e.WebsocketSpotCancelAllOrdersByIDs(t.Context(), []WebsocketOrderBatchRequest{out})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	out.OrderID = "1337"
	_, err = e.WebsocketSpotCancelAllOrdersByIDs(t.Context(), []WebsocketOrderBatchRequest{out})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	orders := make([]WebsocketOrderBatchRequest, 41)
	for i := range orders {
		orders[i] = WebsocketOrderBatchRequest{OrderID: strconv.Itoa(9200 + i), Pair: BTCUSDT}
	}
	withRepeats := append([]WebsocketOrderBatchRequest{orders[0], orders[1], orders[1]}, orders[2:17]...)
	withRepeats = append(withRepeats, orders[0], orders[17])
	// A list within the limit passes the checks, so an unconnected instance fails only at sending
	unconnected := new(Exchange)
	require.NoError(t, testexch.Setup(unconnected), "Setup must not error")
	for _, tc := range []struct {
		name   string
		orders []WebsocketOrderBatchRequest
		err    error
	}{
		{name: "20 distinct orders", orders: orders[:20], err: websocket.ErrNotConnected},
		{name: "20 orders with repeats", orders: withRepeats, err: websocket.ErrNotConnected},
		{name: "21 orders", orders: orders[:21], err: errTooManyOrderIDs},
		{name: "21 orders ending in a repeat", orders: append(orders[:20:20], orders[19]), err: errTooManyOrderIDs},
		{name: "41 orders", orders: orders, err: errTooManyOrderIDs},
		{name: "21 orders ending in one without an ID", orders: append(orders[:20:20], WebsocketOrderBatchRequest{Pair: BTCUSDT}), err: order.ErrOrderIDNotSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := unconnected.WebsocketSpotCancelAllOrdersByIDs(t.Context(), tc.orders)
			assert.ErrorIs(t, err, tc.err, "WebsocketSpotCancelAllOrdersByIDs should return the expected error")
		})
	}

	t.Run("outcomes carry their order IDs", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the cancelled orders are only recorded for mock tests")
		}
		ethUSDT := currency.NewPairWithDelimiter("ETH", "USDT", "_")
		ex := newWsAPITestExchange(t, asset.Spot)
		got, err := ex.WebsocketSpotCancelAllOrdersByIDs(t.Context(), []WebsocketOrderBatchRequest{
			{OrderID: "1700664344", Pair: BTCUSDT},
			{OrderID: "1700664345", Pair: BTCUSDT},
			{OrderID: "1700664346", Pair: ethUSDT},
		})
		require.NoError(t, err, "WebsocketSpotCancelAllOrdersByIDs must not error")
		exp := []*CancelOrderByIDResponse{
			{CurrencyPair: BTCUSDT, OrderID: "1700664344", Text: "t-first", Succeeded: true},
			{CurrencyPair: BTCUSDT, OrderID: "1700664345", Label: "ORDER_NOT_FOUND", Message: "Order not found"},
			{CurrencyPair: ethUSDT, OrderID: "1700664346", Succeeded: true},
		}
		assert.Equal(t, exp, got, "WebsocketSpotCancelAllOrdersByIDs should return each order's outcome with its ID")
	})

	ex := newWsAPITestExchange(t, asset.Spot, canManipulateRealOrders)
	got, err := ex.WebsocketSpotCancelAllOrdersByIDs(t.Context(), []WebsocketOrderBatchRequest{{OrderID: "1700664343", Pair: BTCUSDT}})
	require.NoError(t, err, "WebsocketSpotCancelAllOrdersByIDs must not error")
	if mockTests {
		exp := []*CancelOrderByIDResponse{{CurrencyPair: BTCUSDT, OrderID: "1700664343", Succeeded: true}}
		assert.Equal(t, exp, got, "WebsocketSpotCancelAllOrdersByIDs should decode every field")
		return
	}
	assert.NotEmpty(t, got, "WebsocketSpotCancelAllOrdersByIDs should return the outcomes")
}

func TestWebsocketSpotCancelAllOrdersByPair(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSpotCancelAllOrdersByPair(t.Context(), currency.NewPairWithDelimiter("LTC", "USDT", "_"), 0, "")
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	ex := newWsAPITestExchange(t, asset.Spot, canManipulateRealOrders)
	if mockTests {
		// The pair-scoped fixture only matches a request carrying the pair as currency_pair, the field Gate scopes the
		// cancellation by; any other key would cancel the side's orders on every pair
		got, err := ex.WebsocketSpotCancelAllOrdersByPair(t.Context(), BTCUSDT, order.Buy, "")
		require.NoError(t, err, "WebsocketSpotCancelAllOrdersByPair must not error")
		exp := []*WebsocketOrderResponse{wsAPISpotOrder(func(o *WebsocketOrderResponse) {
			o.ID, o.Status, o.FinishAs = "1700664337", "cancelled", "cancelled"
		})}
		assert.Equal(t, exp, got, "WebsocketSpotCancelAllOrdersByPair should cancel the pair's orders")
	}
	got, err := ex.WebsocketSpotCancelAllOrdersByPair(t.Context(), currency.EMPTYPAIR, order.Buy, "")
	require.NoError(t, err, "WebsocketSpotCancelAllOrdersByPair must not error")
	if mockTests {
		exp := []*WebsocketOrderResponse{wsAPISpotOrder(func(o *WebsocketOrderResponse) {
			o.ID, o.Status, o.FinishAs = "1700664338", "cancelled", "cancelled"
		})}
		assert.Equal(t, exp, got, "WebsocketSpotCancelAllOrdersByPair should cancel the side's orders on every pair")
		return
	}
	assert.NotNil(t, got, "WebsocketSpotCancelAllOrdersByPair should return the cancelled orders")
}

func TestWebsocketSpotAmendOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSpotAmendOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	amend := &WebsocketAmendOrderRequest{}
	_, err = e.WebsocketSpotAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	amend.OrderID = "1337"
	_, err = e.WebsocketSpotAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	amend.Pair = BTCUSDT
	_, err = e.WebsocketSpotAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	ex := newWsAPITestExchange(t, asset.Spot, canManipulateRealOrders)
	got, err := ex.WebsocketSpotAmendOrder(t.Context(), &WebsocketAmendOrderRequest{OrderID: "1700664330", Pair: BTCUSDT, Amount: "0.0004"})
	require.NoError(t, err, "WebsocketSpotAmendOrder must not error")
	if mockTests {
		exp := wsAPISpotOrder(func(o *WebsocketOrderResponse) {
			o.Amount, o.Left = 0.0004, 0.0004
			o.UpdateTime, o.UpdateTimeMs = types.Time(time.Unix(1681986206, 0)), types.Time(time.UnixMilli(1681986206176))
		})
		assert.Equal(t, exp, got, "WebsocketSpotAmendOrder should decode every field")
		return
	}
	assert.NotEmpty(t, got, "WebsocketSpotAmendOrder should return the amended order")
}

func TestWebsocketSpotListOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSpotListOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.WebsocketSpotListOrders(t.Context(), &WebsocketSpotListOrdersRequest{})
	require.ErrorIs(t, err, errStatusNotSet)

	_, err = e.WebsocketSpotListOrders(t.Context(), &WebsocketSpotListOrdersRequest{Status: "closed"})
	require.ErrorIs(t, err, errInvalidOrderStatus)

	_, err = e.WebsocketSpotListOrders(t.Context(), &WebsocketSpotListOrdersRequest{Status: statusOpen})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	startTime, endTime := getTime()
	_, err = e.WebsocketSpotListOrders(t.Context(), &WebsocketSpotListOrdersRequest{Status: statusFinished, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	ex := newWsAPITestExchange(t, asset.Spot)
	got, err := ex.WebsocketSpotListOrders(t.Context(), &WebsocketSpotListOrdersRequest{
		Pair:      BTCUSDT,
		Status:    statusFinished,
		Page:      1,
		Limit:     3,
		Account:   "spot",
		StartTime: startTime,
		EndTime:   endTime,
		Side:      order.Sell.Lower(),
	})
	require.NoError(t, err, "WebsocketSpotListOrders must not error")
	if mockTests {
		exp := []*WebsocketOrderResponse{wsAPISpotOrder(func(o *WebsocketOrderResponse) {
			o.ID, o.Text, o.Status, o.FinishAs, o.Side, o.Left, o.FilledTotal = "20874890569", "web", "closed", "filled", "sell", 0, 1
		})}
		assert.Equal(t, exp, got, "WebsocketSpotListOrders should decode every field")
		return
	}
	assert.NotNil(t, got, "WebsocketSpotListOrders should return orders")
}

func TestWebsocketSpotGetOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSpotGetOrderStatus(t.Context(), "", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.WebsocketSpotGetOrderStatus(t.Context(), "1337", currency.EMPTYPAIR, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	ex := newWsAPITestExchange(t, asset.Spot)
	got, err := ex.WebsocketSpotGetOrderStatus(t.Context(), "1700664330", BTCUSDT, "")
	require.NoError(t, err, "WebsocketSpotGetOrderStatus must not error")
	if mockTests {
		assert.Equal(t, wsAPISpotOrder(nil), got, "WebsocketSpotGetOrderStatus should decode every field")
		return
	}
	assert.NotEmpty(t, got, "WebsocketSpotGetOrderStatus should return the order")
}

// newExchangeWithWebsocket returns a websocket instance copy for testing.
// This restricts the pairs to a single pair per asset type to reduce test time.
func newExchangeWithWebsocket(tb testing.TB, a asset.Item) *Exchange {
	tb.Helper()
	if apiCredentials.Key == "" || apiCredentials.Secret == "" {
		tb.Skip()
	}
	e := newTestExchange(tb)
	testexch.UpdatePairsOnce(tb, e)
	e.API.AuthenticatedSupport = true
	e.API.AuthenticatedWebsocketSupport = true
	e.SetCredentials(apiCredentials)
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)

	// Disable all other asset types to ensure only the specified asset type is used for websocket tests.
	for _, enabled := range e.GetAssetTypes(true) {
		if enabled != a {
			require.NoError(tb, e.CurrencyPairs.SetAssetEnabled(enabled, false))
		}
	}

	require.NoError(tb, e.Websocket.Connect(tb.Context()))
	return e
}
