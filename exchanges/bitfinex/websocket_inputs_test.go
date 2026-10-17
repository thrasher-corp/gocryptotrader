package bitfinex

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// mockOrderRequest is the order the mock websocket places and the mock tests place, an EXCHANGE LIMIT buy
var mockOrderRequest = PlaceOrderRequest{Type: ExchangeOrderTypePrefix + OrderTypeLimit, Symbol: "tBTCUSD", Amount: 0.0002, Price: 40000}

// wsTestOrderRequest returns the order a websocket test places: the mock's, or live a buy far from the market
func wsTestOrderRequest(t *testing.T) *PlaceOrderRequest {
	t.Helper()
	if mockTests {
		req := mockOrderRequest
		return &req
	}
	return liveTestOrderRequest(t)
}

func TestWebsocketInput(t *testing.T) {
	t.Parallel()
	out, err := json.Marshal(&wsInput{Type: wsOrderCancel, Data: &cancelOrderBody{ID: 5}})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `[0,"oc",null,{"id":5}]`, string(out), "an input request should be the channel, type, a placeholder and the data")
}

func TestWebsocketPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketPlaceOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketPlaceOrder must reject a nil request")
	_, err = e.WebsocketPlaceOrder(t.Context(), &PlaceOrderRequest{Type: "NOVEL", Symbol: "tBTCUSD", Amount: 1})
	require.ErrorIs(t, err, errInvalidOrderType, "WebsocketPlaceOrder must validate the order")
	unauthenticated := newTestExchange(t)
	unauthenticated.Websocket.SetCanUseAuthenticatedEndpoints(false)
	_, err = unauthenticated.WebsocketPlaceOrder(t.Context(), &mockOrderRequest)
	require.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketPlaceOrder must need an authenticated websocket")

	skipLiveUnlessCanManipulate(t)
	ex, venue := newWebsocketTestExchange(t, nil)
	req := wsTestOrderRequest(t)
	placed, err := ex.WebsocketPlaceOrder(t.Context(), req)
	require.NoError(t, err, "WebsocketPlaceOrder must not error")
	require.NotNil(t, placed, "WebsocketPlaceOrder must return the order placed")
	if !mockTests {
		cancelLiveTestOrderOnCleanup(t, placed.ID)
		assert.Equal(t, req.Symbol, placed.Symbol, "WebsocketPlaceOrder should return the order placed")
		return
	}
	assert.Zero(t, req.ClientOrderID, "WebsocketPlaceOrder should leave the request unchanged")
	inputs := venue.inputs(wsOrderNew)
	require.Len(t, inputs, 1, "WebsocketPlaceOrder must send one order")
	var body map[string]any
	require.NoError(t, json.Unmarshal(inputs[0], &body), "the order must decode")
	cid, ok := body["cid"].(float64)
	require.True(t, ok, "WebsocketPlaceOrder must give the order a client order ID to match its notification by")
	assert.Equal(t, map[string]any{"type": "EXCHANGE LIMIT", "symbol": "tBTCUSD", "amount": "0.0002", "price": "40000", "cid": cid}, body, "WebsocketPlaceOrder should send the order's fields")
	assert.Equal(t, uint64(cid), placed.ClientOrderID, "WebsocketPlaceOrder should return the order its notification carries")
	assert.Equal(t, uint64(1185815098), placed.ID, "WebsocketPlaceOrder should return the ID Bitfinex gave the order")

	again, err := ex.WebsocketPlaceOrder(t.Context(), req)
	require.NoError(t, err, "WebsocketPlaceOrder must not error for a second order")
	assert.Greater(t, again.ClientOrderID, placed.ClientOrderID, "each order should get its own client order ID")

	_, err = ex.WebsocketPlaceOrder(t.Context(), &PlaceOrderRequest{Type: OrderTypeLimit, Symbol: "tNOPEUSD", Amount: 1, Price: 1, ClientOrderID: 77})
	assert.ErrorIs(t, err, errNotificationFailed, "WebsocketPlaceOrder should return a refused order's notification as an error")
}

func TestWebsocketUpdateOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketUpdateOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketUpdateOrder must reject a nil request")
	_, err = e.WebsocketUpdateOrder(t.Context(), &UpdateOrderRequest{})
	require.ErrorIs(t, err, errOrderIDRequired, "WebsocketUpdateOrder must validate the update")

	skipLiveUnlessCanManipulate(t)
	ex, venue := newWebsocketTestExchange(t, nil)
	id := uint64(1185815100)
	req := &UpdateOrderRequest{ID: id, Amount: 0.0003, Price: 41000}
	if !mockTests {
		placed, err := ex.WebsocketPlaceOrder(t.Context(), liveTestOrderRequest(t))
		require.NoError(t, err, "WebsocketPlaceOrder must not error")
		cancelLiveTestOrderOnCleanup(t, placed.ID)
		req = &UpdateOrderRequest{ID: placed.ID, Amount: placed.Amount, Price: placed.Price - 1000}
	}
	updated, err := ex.WebsocketUpdateOrder(t.Context(), req)
	require.NoError(t, err, "WebsocketUpdateOrder must not error")
	assert.Equal(t, req.ID, updated.ID, "WebsocketUpdateOrder should return the order updated")
	if mockTests {
		inputs := venue.inputs(wsOrderUpdate)
		require.Len(t, inputs, 1, "WebsocketUpdateOrder must send one update")
		assert.JSONEq(t, `{"id":1185815100,"amount":"0.0003","price":"41000"}`, string(inputs[0]), "WebsocketUpdateOrder should send the update's fields")
	}
}

func TestWebsocketCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketCancelExistingOrder(t.Context(), 0)
	require.ErrorIs(t, err, errOrderIDRequired, "WebsocketCancelExistingOrder must need an order ID")

	skipLiveUnlessCanManipulate(t)
	ex, venue := newWebsocketTestExchange(t, nil)
	id := uint64(1185815100)
	if !mockTests {
		placed, err := ex.WebsocketPlaceOrder(t.Context(), liveTestOrderRequest(t))
		require.NoError(t, err, "WebsocketPlaceOrder must not error")
		cancelLiveTestOrderOnCleanup(t, placed.ID)
		id = placed.ID
	}
	cancelled, err := ex.WebsocketCancelExistingOrder(t.Context(), id)
	require.NoError(t, err, "WebsocketCancelExistingOrder must not error")
	assert.Equal(t, id, cancelled.ID, "WebsocketCancelExistingOrder should return the order submitted for cancellation")
	if mockTests {
		inputs := venue.inputs(wsOrderCancel)
		require.Len(t, inputs, 1, "WebsocketCancelExistingOrder must send one cancellation")
		assert.JSONEq(t, `{"id":1185815100}`, string(inputs[0]), "WebsocketCancelExistingOrder should cancel the order by its ID")
	}
}

func TestWebsocketCancelOrders(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.WebsocketCancelOrders(t.Context(), nil), common.ErrNilPointer, "WebsocketCancelOrders must reject a nil request")
	require.ErrorIs(t, e.WebsocketCancelOrders(t.Context(), &CancelOrdersRequest{}), errOrdersToCancelRequired, "WebsocketCancelOrders must need orders to cancel")

	skipLiveUnlessCanManipulate(t)
	ex, venue := newWebsocketTestExchange(t, nil)
	ids := []uint64{1185815100, 1185815101}
	if !mockTests {
		placed, err := ex.WebsocketPlaceOrder(t.Context(), liveTestOrderRequest(t))
		require.NoError(t, err, "WebsocketPlaceOrder must not error")
		cancelLiveTestOrderOnCleanup(t, placed.ID)
		ids = []uint64{placed.ID}
	}
	require.NoError(t, ex.WebsocketCancelOrders(t.Context(), &CancelOrdersRequest{IDs: ids}), "WebsocketCancelOrders must not error")
	if mockTests {
		inputs := venue.inputs(wsOrderCancelMulti)
		require.Len(t, inputs, 1, "WebsocketCancelOrders must send one cancellation")
		assert.JSONEq(t, `{"id":[1185815100,1185815101]}`, string(inputs[0]), "WebsocketCancelOrders should cancel the orders by their IDs")
	}
}

func TestWebsocketSubmitFundingOffer(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSubmitFundingOffer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketSubmitFundingOffer must reject a nil request")
	_, err = e.WebsocketSubmitFundingOffer(t.Context(), &SubmitFundingOfferRequest{Type: "NOVEL"})
	require.ErrorIs(t, err, errInvalidFundingOfferType, "WebsocketSubmitFundingOffer must validate the offer")

	skipLiveUnlessCanManipulate(t)
	ex, venue := newWebsocketTestExchange(t, nil)
	req := &SubmitFundingOfferRequest{Type: FundingOfferTypeLimit, Symbol: "fUST", Amount: 500, Rate: 0.02, Period: 2}
	if !mockTests {
		// Offering funding at 2% a day is far above the market, so the offer rests until cancelled
		req.Amount = 150
	}
	offer, err := ex.WebsocketSubmitFundingOffer(t.Context(), req)
	require.NoError(t, err, "WebsocketSubmitFundingOffer must not error")
	require.NotZero(t, offer.ID, "WebsocketSubmitFundingOffer must return the offer's ID")
	if !mockTests {
		t.Cleanup(func() { _, _ = e.CancelFundingOffer(context.WithoutCancel(t.Context()), offer.ID) })
		return
	}
	assert.Equal(t, &FundingOffer{ID: 41238934, Amount: 500, Rate: 0.02, Period: 2}, offer, "WebsocketSubmitFundingOffer should return the offer its notification carries")
	inputs := venue.inputs(wsFundingOfferNew)
	require.Len(t, inputs, 1, "WebsocketSubmitFundingOffer must send one offer")
	assert.JSONEq(t, `{"type":"LIMIT","symbol":"fUST","amount":"500","rate":"0.02","period":2}`, string(inputs[0]), "WebsocketSubmitFundingOffer should send the offer's fields")
}

func TestWebsocketCancelFundingOffer(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketCancelFundingOffer(t.Context(), 0)
	require.ErrorIs(t, err, errFundingOfferIDRequired, "WebsocketCancelFundingOffer must need an offer ID")

	skipLiveUnlessCanManipulate(t)
	ex, venue := newWebsocketTestExchange(t, nil)
	id := uint64(41238934)
	if !mockTests {
		offer, err := ex.WebsocketSubmitFundingOffer(t.Context(), &SubmitFundingOfferRequest{Type: FundingOfferTypeLimit, Symbol: "fUST", Amount: 150, Rate: 0.02, Period: 2})
		require.NoError(t, err, "WebsocketSubmitFundingOffer must not error")
		id = offer.ID
	}
	offer, err := ex.WebsocketCancelFundingOffer(t.Context(), id)
	require.NoError(t, err, "WebsocketCancelFundingOffer must not error")
	assert.Equal(t, id, offer.ID, "WebsocketCancelFundingOffer should return the offer submitted for cancellation")
	if mockTests {
		inputs := venue.inputs(wsFundingOfferCancel)
		require.Len(t, inputs, 1, "WebsocketCancelFundingOffer must send one cancellation")
		assert.JSONEq(t, `{"id":`+strconv.FormatUint(id, 10)+`}`, string(inputs[0]), "WebsocketCancelFundingOffer should cancel the offer by its ID")
	}
}

func TestWebsocketCalc(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.WebsocketCalc(t.Context()), errCalcRequestsRequired, "WebsocketCalc must need a calculation")
	tooMany := make([]string, wsMaxCalcRequests+1)
	require.ErrorIs(t, e.WebsocketCalc(t.Context(), tooMany...), errTooManyCalcRequests, "WebsocketCalc must refuse more calculations than a request takes")

	skipLiveWithoutCredentials(t)
	ex, venue := newWebsocketTestExchange(t, nil)
	require.NoError(t, ex.WebsocketCalc(t.Context(), "margin_base", "wallet_exchange_BTC"), "WebsocketCalc must not error")
	if mockTests {
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			inputs := venue.inputs(wsCalc)
			if assert.Len(c, inputs, 1, "WebsocketCalc should send one request") {
				assert.JSONEq(c, `[["margin_base"],["wallet_exchange_BTC"]]`, string(inputs[0]), "WebsocketCalc should send each calculation in its own array")
			}
		}, 5*time.Second, 10*time.Millisecond, "the calc request must reach the mock")
	}
}
