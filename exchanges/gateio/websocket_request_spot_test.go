package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
	err := e.websocketLogin(t.Context(), nil, "")
	require.ErrorIs(t, err, common.ErrNilPointer)

	ex := newWsAPITestExchange(t, asset.Spot)
	c, err := ex.Websocket.GetConnection(asset.Spot)
	require.NoError(t, err, "GetConnection must not error")

	err = ex.websocketLogin(t.Context(), c, "")
	require.ErrorIs(t, err, errChannelEmpty)

	require.NoError(t, ex.websocketLogin(t.Context(), c, "spot.login"), "websocketLogin must not error")
}

func TestWebsocketSpotSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSpotSubmitOrder(t.Context(), &CreateOrderRequest{})
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
	_, err := e.WebsocketSpotSubmitOrders(t.Context())
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

	ex := newWsAPITestExchange(t, asset.Spot, canManipulateRealOrders)
	got, err := ex.WebsocketSpotCancelAllOrdersByIDs(t.Context(), []WebsocketOrderBatchRequest{{OrderID: "1700664343", Pair: BTCUSDT}})
	require.NoError(t, err, "WebsocketSpotCancelAllOrdersByIDs must not error")
	if mockTests {
		exp := []*WebsocketCancellAllResponse{{Pair: BTCUSDT, Succeeded: true}}
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

	amend := &WebsocketAmendOrder{}
	_, err = e.WebsocketSpotAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	amend.OrderID = "1337"
	_, err = e.WebsocketSpotAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	amend.Pair = BTCUSDT
	_, err = e.WebsocketSpotAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	ex := newWsAPITestExchange(t, asset.Spot, canManipulateRealOrders)
	got, err := ex.WebsocketSpotAmendOrder(t.Context(), &WebsocketAmendOrder{OrderID: "1700664330", Pair: BTCUSDT, Amount: "0.0004"})
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
	e := new(Exchange)
	require.NoError(tb, testexch.Setup(e), "Test instance Setup must not error")
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
