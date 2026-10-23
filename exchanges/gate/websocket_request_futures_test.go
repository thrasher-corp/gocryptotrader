package gate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	BTCUSDT = currency.NewPairWithDelimiter("BTC", "USDT", "_")
	BTCUSD  = currency.NewPairWithDelimiter("BTC", "USD", "_")
)

// wsAPIFuturesOrder returns the futures order the WebSocket API fixtures return, after applying changes
func wsAPIFuturesOrder(changes func(*WebsocketFuturesOrderResponse)) *WebsocketFuturesOrderResponse {
	o := &WebsocketFuturesOrderResponse{
		ID:                        74046543,
		User:                      6790020,
		CreateTime:                types.Time(time.UnixMilli(1681196535010)),
		FinishTime:                types.Time(time.UnixMilli(1681196536343)),
		FinishAs:                  "cancelled",
		Status:                    "finished",
		Contract:                  BTCUSDT,
		Size:                      10,
		Price:                     31303.2,
		TimeInForce:               "gtc",
		Left:                      10,
		Text:                      "t-my-custom-id",
		TakerFee:                  0.0005,
		MakerFee:                  0.0002,
		SelfTradePreventionAction: "-",
		AmendText:                 "-",
		UpdateTime:                types.Time(time.UnixMilli(1681196536343)),
	}
	if changes != nil {
		changes(o)
	}
	return o
}

// wsAPIPlacedFuturesOrder returns the futures order the WebSocket API order placement fixtures return, after applying
// changes
func wsAPIPlacedFuturesOrder(changes func(*WebsocketFuturesOrderResponse)) *WebsocketFuturesOrderResponse {
	o := wsAPIFuturesOrder(func(o *WebsocketFuturesOrderResponse) {
		wsAPIOpenFuturesOrder(o)
		o.ID, o.Price = 74046511, 31503.3
		o.CreateTime = types.Time(time.UnixMilli(1681195121754))
		o.UpdateTime = types.Time(time.UnixMilli(1681195121754))
	})
	if changes != nil {
		changes(o)
	}
	return o
}

// wsAPIOpenFuturesOrder returns the open futures order the WebSocket API fixtures return
func wsAPIOpenFuturesOrder(o *WebsocketFuturesOrderResponse) {
	o.Status, o.FinishAs, o.FinishTime = "open", "", types.Time{}
}

func TestWebsocketFuturesSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesSubmitOrder(t.Context(), asset.USDTMarginedFutures, &FuturesOrderCreateRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	out := &FuturesOrderCreateRequest{Contract: BTCUSDT}
	_, err = e.WebsocketFuturesSubmitOrder(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, errInvalidOrderSize)

	out.Size = 2
	_, err = e.WebsocketFuturesSubmitOrder(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)

	out.TimeInForce = "gtc"
	_, err = e.WebsocketFuturesSubmitOrder(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)

	out.TimeInForce = iocTIF
	out.Price = 40000
	out.AutoSize = "silly_billies"
	_, err = e.WebsocketFuturesSubmitOrder(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, errInvalidAutoSize)

	_, err = e.WebsocketFuturesSubmitOrder(t.Context(), asset.Spot, &FuturesOrderCreateRequest{Contract: BTCUSDT, Size: 1, Price: 40000, TimeInForce: gtcTIF})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	ex := newWsAPITestExchange(t, asset.USDTMarginedFutures, canManipulateRealOrders)
	got, err := ex.WebsocketFuturesSubmitOrder(t.Context(), asset.USDTMarginedFutures, &FuturesOrderCreateRequest{Contract: BTCUSDT, Size: 10, Price: 31503.3, TimeInForce: gtcTIF, Text: "t-my-custom-id"})
	require.NoError(t, err, "WebsocketFuturesSubmitOrder must not error")
	if mockTests {
		assert.Equal(t, wsAPIPlacedFuturesOrder(nil), got, "WebsocketFuturesSubmitOrder should decode every field")
		return
	}
	assert.NotEmpty(t, got, "WebsocketFuturesSubmitOrder should return the placed order")
}

func TestWebsocketFuturesSubmitOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesSubmitOrders(t.Context(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, errOrdersEmpty)

	out := &FuturesOrderCreateRequest{}
	_, err = e.WebsocketFuturesSubmitOrders(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	out.Contract = BTCUSDT
	_, err = e.WebsocketFuturesSubmitOrders(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, errInvalidOrderSize)

	out.Size = 1
	out.Price = 40000
	out.TimeInForce = "gtc"
	out.AutoSize = "silly_billies"
	_, err = e.WebsocketFuturesSubmitOrders(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, errInvalidAutoSize)

	out.AutoSize = "close_long"
	_, err = e.WebsocketFuturesSubmitOrders(t.Context(), asset.USDTMarginedFutures, out)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	out.AutoSize = ""
	_, err = e.WebsocketFuturesSubmitOrders(t.Context(), asset.Binary, out)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = e.WebsocketFuturesSubmitOrders(t.Context(), asset.Options, out)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	batch := make([]*FuturesOrderCreateRequest, 11)
	for i := range batch {
		batch[i] = out
	}
	_, err = e.WebsocketFuturesSubmitOrders(t.Context(), asset.USDTMarginedFutures, batch...)
	require.ErrorIs(t, err, errTooManyOrderRequest, "WebsocketFuturesSubmitOrders must refuse more than 10 orders")
	// The limit itself passes the check, so an unconnected instance fails only at sending
	unconnected := new(Exchange)
	require.NoError(t, testexch.Setup(unconnected), "Setup must not error")
	_, err = unconnected.WebsocketFuturesSubmitOrders(t.Context(), asset.USDTMarginedFutures, batch[:10]...)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketFuturesSubmitOrders should allow 10 orders and fail only at sending")

	ex := newWsAPITestExchange(t, asset.USDTMarginedFutures, canManipulateRealOrders)
	first := &FuturesOrderCreateRequest{Contract: BTCUSDT, Size: 10, Price: 31503.3, TimeInForce: gtcTIF, Text: "t-my-custom-id"}
	second := &FuturesOrderCreateRequest{Contract: BTCUSDT, Size: -5, Price: 31600, TimeInForce: gtcTIF, Text: "t-my-other-id"}
	got, err := ex.WebsocketFuturesSubmitOrders(t.Context(), asset.USDTMarginedFutures, first, second)
	require.NoError(t, err, "WebsocketFuturesSubmitOrders must not error")
	if mockTests {
		succeeded := true
		exp := []*WebsocketFuturesOrderResponse{
			wsAPIPlacedFuturesOrder(func(o *WebsocketFuturesOrderResponse) { o.Succeeded = &succeeded }),
			wsAPIPlacedFuturesOrder(func(o *WebsocketFuturesOrderResponse) {
				o.ID, o.Size, o.Left, o.Price, o.Text, o.Succeeded = 74046512, -5, -5, 31600, "t-my-other-id", &succeeded
			}),
		}
		assert.Equal(t, exp, got, "WebsocketFuturesSubmitOrders should decode every order")
		return
	}
	assert.NotEmpty(t, got, "WebsocketFuturesSubmitOrders should return the placed orders")
}

func TestWebsocketFuturesCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesCancelOrder(t.Context(), "", currency.EMPTYPAIR, asset.Empty)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.WebsocketFuturesCancelOrder(t.Context(), "42069", currency.EMPTYPAIR, asset.Empty)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.WebsocketFuturesCancelOrder(t.Context(), "42069", BTCUSDT, asset.Empty)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	ex := newWsAPITestExchange(t, asset.USDTMarginedFutures, canManipulateRealOrders)
	got, err := ex.WebsocketFuturesCancelOrder(t.Context(), "74046543", BTCUSDT, asset.USDTMarginedFutures)
	require.NoError(t, err, "WebsocketFuturesCancelOrder must not error")
	if mockTests {
		assert.Equal(t, wsAPIFuturesOrder(nil), got, "WebsocketFuturesCancelOrder should decode every field")

		_, err = ex.WebsocketFuturesCancelOrder(t.Context(), "74046544", BTCUSDT, asset.USDTMarginedFutures)
		assert.ErrorIs(t, err, common.ErrNoResponse, "WebsocketFuturesCancelOrder should reject a null result")
		return
	}
	assert.NotEmpty(t, got, "WebsocketFuturesCancelOrder should return the cancelled order")
}

func TestWebsocketFuturesCancelOrdersByIDs(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesCancelOrdersByIDs(t.Context(), asset.Spot, []string{"1"})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.WebsocketFuturesCancelOrdersByIDs(t.Context(), asset.USDTMarginedFutures, nil)
	require.ErrorIs(t, err, errNoOrdersToCancel)

	_, err = e.WebsocketFuturesCancelOrdersByIDs(t.Context(), asset.USDTMarginedFutures, make([]string, 21))
	require.ErrorIs(t, err, errTooManyOrderIDs)

	_, err = e.WebsocketFuturesCancelOrdersByIDs(t.Context(), asset.USDTMarginedFutures, []string{"1", ""})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	ex := newWsAPITestExchange(t, asset.USDTMarginedFutures, canManipulateRealOrders)
	got, err := ex.WebsocketFuturesCancelOrdersByIDs(t.Context(), asset.USDTMarginedFutures, []string{"1694883366", "123"})
	require.NoError(t, err, "WebsocketFuturesCancelOrdersByIDs must not error")
	if mockTests {
		exp := []*WebsocketFuturesCancelOrderByIDResponse{
			{OrderID: "1694883366", UserID: 111, Succeeded: true},
			{OrderID: "123", UserID: 111, Message: "ORDER_NOT_FOUND"},
		}
		assert.Equal(t, exp, got, "WebsocketFuturesCancelOrdersByIDs should decode each order's outcome")
		return
	}
	assert.NotEmpty(t, got, "WebsocketFuturesCancelOrdersByIDs should return the outcomes")
}

func TestWebsocketFuturesCancelAllOpenFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesCancelAllOpenFuturesOrders(t.Context(), currency.EMPTYPAIR, asset.Empty, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.WebsocketFuturesCancelAllOpenFuturesOrders(t.Context(), BTCUSDT, asset.Empty, "bruh")
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.WebsocketFuturesCancelAllOpenFuturesOrders(t.Context(), BTCUSDT, asset.USDTMarginedFutures, "bruh")
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	ex := newWsAPITestExchange(t, asset.USDTMarginedFutures, canManipulateRealOrders)
	got, err := ex.WebsocketFuturesCancelAllOpenFuturesOrders(t.Context(), BTCUSDT, asset.USDTMarginedFutures, order.Bid.Lower())
	require.NoError(t, err, "WebsocketFuturesCancelAllOpenFuturesOrders must not error")
	if mockTests {
		exp := []*WebsocketFuturesOrderResponse{wsAPIFuturesOrder(func(o *WebsocketFuturesOrderResponse) { o.ID = 74046545 })}
		assert.Equal(t, exp, got, "WebsocketFuturesCancelAllOpenFuturesOrders should decode every field")
		return
	}
	assert.NotNil(t, got, "WebsocketFuturesCancelAllOpenFuturesOrders should return the cancelled orders")
}

func TestWebsocketFuturesAmendOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesAmendOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	amend := &WebsocketFuturesAmendOrderRequest{}
	_, err = e.WebsocketFuturesAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	amend.OrderID = "1337"
	_, err = e.WebsocketFuturesAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	amend.Contract = BTCUSDT
	_, err = e.WebsocketFuturesAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	amend.Asset = asset.USDTMarginedFutures
	_, err = e.WebsocketFuturesAmendOrder(t.Context(), amend)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	amend.Size = 2

	ex := newWsAPITestExchange(t, asset.USDTMarginedFutures, canManipulateRealOrders)
	amend.OrderID = "74046543"
	got, err := ex.WebsocketFuturesAmendOrder(t.Context(), amend)
	require.NoError(t, err, "WebsocketFuturesAmendOrder must not error")
	if mockTests {
		exp := wsAPIFuturesOrder(func(o *WebsocketFuturesOrderResponse) {
			wsAPIOpenFuturesOrder(o)
			o.Size, o.Left = 2, 2
		})
		assert.Equal(t, exp, got, "WebsocketFuturesAmendOrder should decode every field")
		return
	}
	assert.NotEmpty(t, got, "WebsocketFuturesAmendOrder should return the amended order")
}

func TestWebsocketFuturesOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	list := &WebsocketFutureOrdersListRequest{}
	_, err = e.WebsocketFuturesOrderList(t.Context(), list)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	list.Contract = BTCUSDT
	_, err = e.WebsocketFuturesOrderList(t.Context(), list)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	list.Asset = asset.USDTMarginedFutures
	_, err = e.WebsocketFuturesOrderList(t.Context(), list)
	require.ErrorIs(t, err, errStatusNotSet)

	ex := newWsAPITestExchange(t, asset.USDTMarginedFutures)
	list.Status = statusOpen
	got, err := ex.WebsocketFuturesOrderList(t.Context(), list)
	require.NoError(t, err, "WebsocketFuturesOrderList must not error")
	if mockTests {
		exp := []*WebsocketFuturesOrderResponse{wsAPIFuturesOrder(wsAPIOpenFuturesOrder)}
		assert.Equal(t, exp, got, "WebsocketFuturesOrderList should decode every field")
		return
	}
	assert.NotNil(t, got, "WebsocketFuturesOrderList should return orders")
}

func TestWebsocketFuturesGetOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketFuturesGetOrderStatus(t.Context(), currency.EMPTYPAIR, asset.Empty, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.WebsocketFuturesGetOrderStatus(t.Context(), BTCUSDT, asset.Empty, "")
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.WebsocketFuturesGetOrderStatus(t.Context(), BTCUSDT, asset.USDTMarginedFutures, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	ex := newWsAPITestExchange(t, asset.USDTMarginedFutures)
	got, err := ex.WebsocketFuturesGetOrderStatus(t.Context(), BTCUSDT, asset.USDTMarginedFutures, "74046543")
	require.NoError(t, err, "WebsocketFuturesGetOrderStatus must not error")
	if mockTests {
		assert.Equal(t, wsAPIFuturesOrder(wsAPIOpenFuturesOrder), got, "WebsocketFuturesGetOrderStatus should decode every field")
		return
	}
	assert.NotEmpty(t, got, "WebsocketFuturesGetOrderStatus should return the order")
}
