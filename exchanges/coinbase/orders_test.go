package coinbase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// testSandboxOrderID is the order the sandbox's create, cancel and edit responses name
	testSandboxOrderID = "f898eaf4-6ffc-47be-a159-7ff292e5cdcf"
	// testSandboxBracketedOrderID is the sandbox's order with an attached take profit/stop loss order, and
	// testSandboxAttachedOrderID that attached order
	testSandboxBracketedOrderID = "c6981abf-a8b4-4209-b33f-2aa6ee8cd72f"
	testSandboxAttachedOrderID  = "3daff354-c513-4c9d-869e-6ebf223364f4"
	// testSandboxFuturesProductID is the product the sandbox's close position response names
	testSandboxFuturesProductID = "BIT-30AUG24-CDE"
	// testUnknownOrderID is an order Coinbase does not know, whose fixtures hold the sandbox's failure responses
	testUnknownOrderID = "00000000-0000-0000-0000-000000000000"
	// liveTestOrderSize is the size of the order live tests place, worth more than the minimum order at its price
	liveTestOrderSize = 0.002
	// testOrderNotFoundBody is Coinbase's answer to a request for an order it does not know
	testOrderNotFoundBody = `{"error":"NOT_FOUND","error_details":"order with this orderID was not found","message":"order with this orderID was not found"}`
)

// testCreateOrderRequest is the order the sandbox's create order response confirms
func testCreateOrderRequest() *CreateOrderRequest {
	return &CreateOrderRequest{
		ClientOrderID:      "sandbox_success_order",
		ProductID:          testSpotProductID,
		Side:               OrderSideSell,
		OrderConfiguration: &LimitLimitGTCRequest{BaseSize: 1, LimitPrice: 70000.44},
	}
}

// testInsufficientFundsCreateOrderRequest is an order the sandbox's insufficient funds response refuses
func testInsufficientFundsCreateOrderRequest() *CreateOrderRequest {
	return &CreateOrderRequest{
		ClientOrderID:      "sandbox_insufficient_fund_order",
		ProductID:          testSpotProductID,
		Side:               OrderSideBuy,
		OrderConfiguration: &LimitLimitGTCRequest{BaseSize: 1000, LimitPrice: 70000.44},
	}
}

// testClosePositionRequest is the position the sandbox's close position response closes
func testClosePositionRequest() *ClosePositionRequest {
	return &ClosePositionRequest{ClientOrderID: "sandbox_success_close_position", ProductID: testSandboxFuturesProductID, Size: 100}
}

// testEditOrderRequest is the edit the sandbox's edit and edit preview responses describe, of orderID
func testEditOrderRequest(orderID string) *EditOrderRequest {
	return &EditOrderRequest{OrderID: orderID, Price: 68000.44, Size: 1}
}

// testPreviewOrderRequest is the order the sandbox's preview response describes, for a quote size of quoteSize
func testPreviewOrderRequest(quoteSize float64) *PreviewOrderRequest {
	return &PreviewOrderRequest{ProductID: testSpotProductID, Side: OrderSideBuy, OrderConfiguration: &MarketMarketIOCRequest{QuoteSize: quoteSize}}
}

// testListOrdersRequest selects the sandbox's open order
func testListOrdersRequest() *ListOrdersRequest {
	return &ListOrdersRequest{OrderStatuses: []string{OrderStatusOpen}}
}

// liveTestCreateOrderRequest is a post only limit buy priced far below the market, which live tests place and cancel
func liveTestCreateOrderRequest() *CreateOrderRequest {
	return &CreateOrderRequest{
		ClientOrderID:      "gct-test-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		ProductID:          testSpotProductID,
		Side:               OrderSideBuy,
		OrderConfiguration: &LimitLimitGTCRequest{BaseSize: liveTestOrderSize, LimitPrice: 1000, PostOnly: true},
	}
}

// newLiveTestOrder places the live test order, cancelled when the test ends, and returns its ID
func newLiveTestOrder(t *testing.T) string {
	t.Helper()
	resp, err := e.CreateOrder(t.Context(), liveTestCreateOrderRequest())
	require.NoError(t, err, "CreateOrder must place the live test order")
	require.NotNil(t, resp.SuccessResponse, "CreateOrder must identify the live test order")
	cancelLiveTestOrder(t, resp.SuccessResponse.OrderID)
	return resp.SuccessResponse.OrderID
}

// cancelLiveTestOrder cancels a live test order when the test ends; an order the test cancelled itself cannot be
// cancelled again, which is only logged
func cancelLiveTestOrder(t *testing.T, orderID string) {
	t.Helper()
	t.Cleanup(func() {
		if _, err := e.CancelOrders(context.WithoutCancel(t.Context()), []string{orderID}); err != nil {
			t.Logf("Cancelling live test order %s: %v", orderID, err)
		}
	})
}

// newOrderNotFoundServerExchange returns an exchange whose requests are answered as Coinbase answers a request for an
// order it does not know
func newOrderNotFoundServerExchange(t *testing.T) *Exchange {
	t.Helper()
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte(testOrderNotFoundBody))
		assert.NoError(t, err, "Writing the response should not error")
	}))
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOrders(t.Context(), nil)
	require.ErrorIs(t, err, errOrderIDRequired, "CancelOrders must require an order ID")
	_, err = e.CancelOrders(t.Context(), []string{testSandboxOrderID, ""})
	require.ErrorIs(t, err, errOrderIDRequired, "CancelOrders must refuse an empty order ID")

	skipLiveUnlessCanManipulate(t)
	orderID := testSandboxOrderID
	if !mockTests {
		orderID = newLiveTestOrder(t)
	}
	resp, err := e.CancelOrders(t.Context(), []string{orderID})
	require.NoError(t, err, "CancelOrders must not error")
	if mockTests {
		exp := &CancelOrdersResponse{
			Results: []CancelOrderResult{
				{
					Success:       true,
					FailureReason: "UNKNOWN_CANCEL_FAILURE_REASON",
					OrderID:       "f898eaf4-6ffc-47be-a159-7ff292e5cdcf",
				},
			},
		}
		assert.Equal(t, exp, resp, "CancelOrders should decode every field")
	} else {
		require.Len(t, resp.Results, 1, "CancelOrders must return the order's result")
		assert.Equal(t, orderID, resp.Results[0].OrderID, "CancelOrders should return the order's result")
	}

	resp, err = e.CancelOrders(t.Context(), []string{testUnknownOrderID})
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "CancelOrders should report an order Coinbase does not know")
	if mockTests {
		exp := &CancelOrdersResponse{
			Results: []CancelOrderResult{
				{
					FailureReason: "UNKNOWN_CANCEL_ORDER",
					OrderID:       "00000000-0000-0000-0000-000000000000",
				},
			},
		}
		assert.Equal(t, exp, resp, "CancelOrders should return the results reporting the failure")
	}

	_, err = newResponseServerExchange(t, `{"results":[{"success":false,"failure_reason":"ORDER_IS_FULLY_FILLED","order_id":"`+testSandboxOrderID+`"}]}`).CancelOrders(t.Context(), []string{testSandboxOrderID})
	assert.ErrorIs(t, err, errCancelOrderFailed, "CancelOrders should report a cancel request Coinbase did not submit")
	_, err = newOrderNotFoundServerExchange(t).CancelOrders(t.Context(), []string{testUnknownOrderID})
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "CancelOrders should report a request refused for an order Coinbase does not know")
	_, err = newResponseServerExchange(t, `null`).CancelOrders(t.Context(), []string{testSandboxOrderID})
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelOrders should error for a null response")
	_, err = newUnavailableServerExchange(t).CancelOrders(t.Context(), []string{testSandboxOrderID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelOrders should return the request failure")
}

func TestClosePosition(t *testing.T) {
	t.Parallel()
	_, err := e.ClosePosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ClosePosition must reject a nil request")
	_, err = e.ClosePosition(t.Context(), &ClosePositionRequest{ProductID: testSandboxFuturesProductID})
	require.ErrorIs(t, err, errClientOrderIDRequired, "ClosePosition must require a client order ID")
	_, err = e.ClosePosition(t.Context(), &ClosePositionRequest{ClientOrderID: "close"})
	require.ErrorIs(t, err, errProductIDRequired, "ClosePosition must require a product ID")
	_, err = e.ClosePosition(t.Context(), &ClosePositionRequest{ClientOrderID: "close", ProductID: testSandboxFuturesProductID, Size: -1})
	require.ErrorIs(t, err, errInvalidAmount, "ClosePosition must refuse a negative size")

	if !mockTests {
		t.Skip("closing a position cannot be undone, and needs an open position to close")
	}
	resp, err := e.ClosePosition(t.Context(), testClosePositionRequest())
	require.NoError(t, err, "ClosePosition must not error")
	exp := &CreateOrderResponse{
		Success: true,
		SuccessResponse: &CreateOrderSuccess{
			OrderID:       "f898eaf4-6ffc-47be-a159-7ff292e5cdcf",
			ProductID:     "BIT-30AUG24-CDE",
			Side:          "SELL",
			ClientOrderID: "sandbox_success_close_position",
		},
		OrderConfiguration: &OrderConfiguration{
			MarketMarketIOC: &MarketMarketIOC{
				BaseSize: 100,
			},
		},
	}
	assert.Equal(t, exp, resp, "ClosePosition should decode every field")

	resp, err = newResponseServerExchange(t, `{"success":false,"error_response":{"error":"UNKNOWN_FAILURE_REASON","message":"Position cannot be closed","error_details":"no open position","preview_failure_reason":"UNKNOWN_PREVIEW_FAILURE_REASON","new_order_failure_reason":"CANNOT_CLOSE_ZERO_POSITION"}}`).ClosePosition(t.Context(), testClosePositionRequest())
	assert.ErrorIs(t, err, errOrderNotCreated, "ClosePosition should report an order Coinbase did not create")
	exp = &CreateOrderResponse{
		ErrorResponse: &CreateOrderFailure{
			FailureReason:         "UNKNOWN_FAILURE_REASON",
			Message:               "Position cannot be closed",
			ErrorDetails:          "no open position",
			PreviewFailureReason:  "UNKNOWN_PREVIEW_FAILURE_REASON",
			NewOrderFailureReason: "CANNOT_CLOSE_ZERO_POSITION",
		},
	}
	assert.Equal(t, exp, resp, "ClosePosition should return the response reporting the failure")
	_, err = newResponseServerExchange(t, `null`).ClosePosition(t.Context(), testClosePositionRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "ClosePosition should error for a null response")
	_, err = newUnavailableServerExchange(t).ClosePosition(t.Context(), testClosePositionRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "ClosePosition should return the request failure")
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateOrder must reject a nil request")
	req := testCreateOrderRequest()
	req.ClientOrderID = ""
	_, err = e.CreateOrder(t.Context(), req)
	require.ErrorIs(t, err, errClientOrderIDRequired, "CreateOrder must require a client order ID")
	req = testCreateOrderRequest()
	req.OrderConfiguration = nil
	_, err = e.CreateOrder(t.Context(), req)
	require.ErrorIs(t, err, errOrderConfigurationRequired, "CreateOrder must require an order configuration")

	skipLiveUnlessCanManipulate(t)
	req = testCreateOrderRequest()
	if !mockTests {
		req = liveTestCreateOrderRequest()
	}
	resp, err := e.CreateOrder(t.Context(), req)
	require.NoError(t, err, "CreateOrder must not error")
	if mockTests {
		exp := &CreateOrderResponse{
			Success: true,
			SuccessResponse: &CreateOrderSuccess{
				OrderID:       "f898eaf4-6ffc-47be-a159-7ff292e5cdcf",
				ProductID:     "BTC-USD",
				Side:          "SELL",
				ClientOrderID: "sandbox_success_order",
			},
			OrderConfiguration: &OrderConfiguration{
				LimitLimitGTC: &LimitLimitGTC{
					BaseSize:   1,
					LimitPrice: 70000.44,
				},
			},
		}
		assert.Equal(t, exp, resp, "CreateOrder should decode every field")
	} else {
		require.NotNil(t, resp.SuccessResponse, "CreateOrder must identify the order it placed")
		cancelLiveTestOrder(t, resp.SuccessResponse.OrderID)
		assert.Equal(t, req.ClientOrderID, resp.SuccessResponse.ClientOrderID, "CreateOrder should return the order's client order ID")
	}

	if mockTests {
		resp, err = e.CreateOrder(t.Context(), testInsufficientFundsCreateOrderRequest())
		assert.ErrorIs(t, err, errOrderNotCreated, "CreateOrder should report an order Coinbase did not create")
		exp := &CreateOrderResponse{
			ErrorResponse: &CreateOrderFailure{
				FailureReason:        "INSUFFICIENT_FUND",
				Message:              "Insufficient balance in source account",
				PreviewFailureReason: "PREVIEW_INSUFFICIENT_FUND",
			},
			OrderConfiguration: &OrderConfiguration{
				LimitLimitGTC: &LimitLimitGTC{
					BaseSize:   1,
					LimitPrice: 70000.44,
				},
			},
		}
		assert.Equal(t, exp, resp, "CreateOrder should return the response reporting the failure")
	}

	_, err = newResponseServerExchange(t, `null`).CreateOrder(t.Context(), testCreateOrderRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateOrder should error for a null response")
	_, err = newUnavailableServerExchange(t).CreateOrder(t.Context(), testCreateOrderRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateOrder should return the request failure")
}

func TestEditOrder(t *testing.T) {
	t.Parallel()
	_, err := e.EditOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "EditOrder must reject a nil request")
	_, err = e.EditOrder(t.Context(), testEditOrderRequest(""))
	require.ErrorIs(t, err, errOrderIDRequired, "EditOrder must require an order ID")

	skipLiveUnlessCanManipulate(t)
	req := testEditOrderRequest(testSandboxOrderID)
	if !mockTests {
		req = &EditOrderRequest{OrderID: newLiveTestOrder(t), Price: 1100, Size: liveTestOrderSize}
	}
	resp, err := e.EditOrder(t.Context(), req)
	require.NoError(t, err, "EditOrder must not error")
	if mockTests {
		exp := &EditOrderResponse{
			Success: true,
			Errors:  []EditOrderError{},
		}
		assert.Equal(t, exp, resp, "EditOrder should decode every field")
	} else {
		assert.True(t, resp.Success, "EditOrder should place the edit")
	}

	resp, err = e.EditOrder(t.Context(), testEditOrderRequest(testUnknownOrderID))
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "EditOrder should report an order Coinbase does not know")
	if mockTests {
		assert.ErrorIs(t, err, errOrderNotEdited, "EditOrder should report an edit Coinbase did not place")
		exp := &EditOrderResponse{
			Errors: []EditOrderError{
				{
					EditFailureReason: "ORDER_NOT_FOUND",
				},
			},
		}
		assert.Equal(t, exp, resp, "EditOrder should return the response reporting the failure")
	}

	_, err = newOrderNotFoundServerExchange(t).EditOrder(t.Context(), testEditOrderRequest(testUnknownOrderID))
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "EditOrder should report a request refused for an order Coinbase does not know")
	_, err = newResponseServerExchange(t, `null`).EditOrder(t.Context(), testEditOrderRequest(testSandboxOrderID))
	assert.ErrorIs(t, err, common.ErrNoResponse, "EditOrder should error for a null response")
	_, err = newUnavailableServerExchange(t).EditOrder(t.Context(), testEditOrderRequest(testSandboxOrderID))
	assert.ErrorIs(t, err, request.ErrBadStatus, "EditOrder should return the request failure")
}

func TestEditOrderPreview(t *testing.T) {
	t.Parallel()
	_, err := e.EditOrderPreview(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "EditOrderPreview must reject a nil request")
	_, err = e.EditOrderPreview(t.Context(), &EditOrderRequest{OrderID: testSandboxOrderID, Size: 1})
	require.ErrorIs(t, err, errInvalidPrice, "EditOrderPreview must require a price")

	// A live preview needs an order to edit, which the test places and cancels
	skipLiveUnlessCanManipulate(t)
	req := testEditOrderRequest(testSandboxOrderID)
	if !mockTests {
		req = &EditOrderRequest{OrderID: newLiveTestOrder(t), Price: 1100, Size: liveTestOrderSize}
	}
	resp, err := e.EditOrderPreview(t.Context(), req)
	require.NoError(t, err, "EditOrderPreview must not error")
	if mockTests {
		exp := &EditOrderPreviewResponse{
			Errors:             []EditOrderError{},
			OrderTotal:         67456.43648,
			CommissionTotal:    544.00352,
			QuoteSize:          68000.44,
			BaseSize:           1,
			BestBid:            62675.96,
			BestAsk:            62683.73,
			AverageFilledPrice: 68000.44,
		}
		assert.Equal(t, exp, resp, "EditOrderPreview should decode every field")
	} else {
		assert.Positive(t, resp.BaseSize.Float64(), "EditOrderPreview should return the edited size")
	}

	if mockTests {
		resp, err = e.EditOrderPreview(t.Context(), testEditOrderRequest(testUnknownOrderID))
		assert.ErrorIs(t, err, errOrderEditPreviewFailed, "EditOrderPreview should report an edit that would fail")
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "EditOrderPreview should report an order Coinbase does not know")
		exp := &EditOrderPreviewResponse{
			Errors: []EditOrderError{
				{
					EditFailureReason: "ORDER_NOT_FOUND",
				},
			},
			OrderTotal:         67456.43648,
			CommissionTotal:    544.00352,
			QuoteSize:          68000.44,
			BaseSize:           1,
			BestBid:            62675.96,
			BestAsk:            62683.73,
			AverageFilledPrice: 68000.44,
		}
		assert.Equal(t, exp, resp, "EditOrderPreview should return the response reporting the failure")
	}

	_, err = newOrderNotFoundServerExchange(t).EditOrderPreview(t.Context(), testEditOrderRequest(testUnknownOrderID))
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "EditOrderPreview should report a request refused for an order Coinbase does not know")
	_, err = newResponseServerExchange(t, `null`).EditOrderPreview(t.Context(), testEditOrderRequest(testSandboxOrderID))
	assert.ErrorIs(t, err, common.ErrNoResponse, "EditOrderPreview should error for a null response")
	_, err = newUnavailableServerExchange(t).EditOrderPreview(t.Context(), testEditOrderRequest(testSandboxOrderID))
	assert.ErrorIs(t, err, request.ErrBadStatus, "EditOrderPreview should return the request failure")
}

func TestGetOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrder(t.Context(), "")
	require.ErrorIs(t, err, errOrderIDRequired, "GetOrder must require an order ID")

	skipLiveWithoutCredentials(t)
	orderID := testSandboxBracketedOrderID
	if !mockTests {
		orders, err := e.ListOrders(t.Context(), &ListOrdersRequest{Limit: 1})
		require.NoError(t, err, "ListOrders must not error")
		if len(orders.Orders) == 0 {
			t.Skip("the account has no orders to get")
		}
		orderID = orders.Orders[0].OrderID
	}
	resp, err := e.GetOrder(t.Context(), orderID)
	require.NoError(t, err, "GetOrder must not error")
	if mockTests {
		exp := &Order{
			OrderID:   "c6981abf-a8b4-4209-b33f-2aa6ee8cd72f",
			ProductID: "BTC-USD",
			UserID:    "eec99928-3d46-5b38-a1ca-122d0e5fb7f0",
			OrderConfiguration: &OrderConfiguration{
				LimitLimitGTC: &LimitLimitGTC{
					BaseSize:   0.05,
					LimitPrice: 99900.83,
				},
			},
			Side:                  "BUY",
			ClientOrderID:         "originating_order",
			Status:                "OPEN",
			TimeInForce:           "GOOD_UNTIL_CANCELLED",
			CreatedTime:           time.Date(2024, 12, 11, 18, 36, 12, 323813000, time.UTC),
			CompletionPercentage:  20,
			FilledSize:            0.01,
			AverageFilledPrice:    99900.83,
			NumberOfFills:         1,
			FilledValue:           999.0083,
			TotalFees:             0.69930581,
			TotalValueAfterFees:   999.70760581,
			TriggerStatus:         "INVALID_ORDER_TYPE",
			OrderType:             "LIMIT",
			RejectReason:          "REJECT_REASON_UNSPECIFIED",
			ProductType:           "SPOT",
			OrderPlacementSource:  "RETAIL_ADVANCED",
			OutstandingHoldAmount: 5003.0335664000001,
			LastFillTime:          time.Date(2024, 12, 11, 18, 36, 14, 410774000, time.UTC),
			EditHistory:           []OrderEdit{},
			MarginType:            "UNKNOWN_MARGIN_TYPE",
			RetailPortfolioID:     "eec99928-3d46-5b38-a1ca-122d0e5fb7f0",
			AttachedOrderID:       "3daff354-c513-4c9d-869e-6ebf223364f4",
			AttachedOrderConfiguration: &OrderConfiguration{
				TriggerBracketGTC: &TriggerBracketGTC{
					LimitPrice:       99990,
					StopTriggerPrice: 99800,
				},
			},
			CostBasisMethod:      "COST_BASIS_METHOD_UNSPECIFIED",
			DisplayedOrderConfig: "UNKNOWN_DISPLAYED_ORDER_CONFIG",
			EquityTradingSession: "UNKNOWN_EQUITY_TRADING_SESSION",
			PredictionSide:       "PREDICTION_SIDE_UNKNOWN",
			OrderDataSource:      "ORDER_DATA_SOURCE_UNKNOWN",
			FundingSource:        "FUNDING_SOURCE_UNSPECIFIED",
			OptionLegs:           json.RawMessage(`[]`),
			Holds:                json.RawMessage(`null`),
			ComboLegs:            json.RawMessage(`[]`),
		}
		assert.Equal(t, exp, resp, "GetOrder should decode every field")

		resp, err = e.GetOrder(t.Context(), testSandboxAttachedOrderID)
		require.NoError(t, err, "GetOrder must not error for an attached order")
		exp = &Order{
			OrderID:   "3daff354-c513-4c9d-869e-6ebf223364f4",
			ProductID: "BTC-USD",
			UserID:    "eec99928-3d46-5b38-a1ca-122d0e5fb7f0",
			OrderConfiguration: &OrderConfiguration{
				TriggerBracketGTC: &TriggerBracketGTC{
					BaseSize:         0.05,
					LimitPrice:       99990,
					StopTriggerPrice: 99800,
				},
			},
			Side:                 "SELL",
			ClientOrderID:        "attached_order",
			Status:               "FILLED",
			TimeInForce:          "GOOD_UNTIL_CANCELLED",
			CreatedTime:          time.Date(2024, 12, 11, 18, 36, 18, 184309000, time.UTC),
			CompletionPercentage: 100,
			FilledSize:           0.05,
			AverageFilledPrice:   99990,
			NumberOfFills:        3,
			FilledValue:          4999.5,
			TotalFees:            3.49965,
			TotalValueAfterFees:  4996.00035,
			TriggerStatus:        "INVALID_ORDER_TYPE",
			OrderType:            "TAKE_PROFIT_STOP_LOSS",
			RejectReason:         "REJECT_REASON_UNSPECIFIED",
			Settled:              true,
			ProductType:          "SPOT",
			OrderPlacementSource: "RETAIL_ADVANCED",
			LastFillTime:         time.Date(2024, 12, 11, 18, 36, 23, 430757000, time.UTC),
			EditHistory:          []OrderEdit{},
			MarginType:           "UNKNOWN_MARGIN_TYPE",
			RetailPortfolioID:    "eec99928-3d46-5b38-a1ca-122d0e5fb7f0",
			OriginatingOrderID:   "c6981abf-a8b4-4209-b33f-2aa6ee8cd72f",
			CostBasisMethod:      "COST_BASIS_METHOD_UNSPECIFIED",
			DisplayedOrderConfig: "UNKNOWN_DISPLAYED_ORDER_CONFIG",
			EquityTradingSession: "UNKNOWN_EQUITY_TRADING_SESSION",
			PredictionSide:       "PREDICTION_SIDE_UNKNOWN",
			OrderDataSource:      "ORDER_DATA_SOURCE_UNKNOWN",
			FundingSource:        "FUNDING_SOURCE_UNSPECIFIED",
			OptionLegs:           json.RawMessage(`[]`),
			Holds:                json.RawMessage(`null`),
			ComboLegs:            json.RawMessage(`[]`),
		}
		assert.Equal(t, exp, resp, "GetOrder should decode every field of an attached take profit/stop loss order")
	} else {
		assert.Equal(t, orderID, resp.OrderID, "GetOrder should return the order")
	}

	_, err = newOrderNotFoundServerExchange(t).GetOrder(t.Context(), testUnknownOrderID)
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetOrder should report an order Coinbase does not know")
	_, err = newResponseServerExchange(t, `{"order":null}`).GetOrder(t.Context(), testSandboxOrderID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOrder should error for a response without an order")
	_, err = newResponseServerExchange(t, `null`).GetOrder(t.Context(), testSandboxOrderID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOrder should error for a null response")
	_, err = newUnavailableServerExchange(t).GetOrder(t.Context(), testSandboxOrderID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrder should return the request failure")
}

func TestListFills(t *testing.T) {
	t.Parallel()
	_, err := e.ListFills(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ListFills must reject a nil request")
	_, err = e.ListFills(t.Context(), &ListFillsRequest{SortBy: OrdersSortByLimitPrice})
	require.ErrorIs(t, err, errInvalidFillsSortBy, "ListFills must refuse an order list sort field")

	skipLiveWithoutCredentials(t)
	req := &ListFillsRequest{}
	if !mockTests {
		req.Limit = 5
	}
	resp, err := e.ListFills(t.Context(), req)
	require.NoError(t, err, "ListFills must not error")
	if mockTests {
		exp := &ListFillsResponse{
			Fills: []Fill{
				{
					EntryID:            "f8b3aa2ba0825457bad20056c52cbe51177af5dd2f66dc55445640f95658f1f3",
					TradeID:            "d006d1e0-cf64-4cdf-8fe1-8441221cd96f",
					OrderID:            "37f17b89-cd9e-41eb-9187-2df6aa72f895",
					TradeTime:          time.Date(2024, 6, 27, 20, 32, 6, 568973000, time.UTC),
					TradeType:          "FILL",
					Price:              70000.44,
					Size:               0.00137199,
					Commission:         0.3841596147024,
					ProductID:          "BTC-USD",
					SequenceTimestamp:  time.Date(2024, 6, 27, 20, 32, 6, 574070000, time.UTC),
					LiquidityIndicator: "MAKER",
					UserID:             "eec99928-3d46-5b38-a1ca-122d0e5fb7f0",
					Side:               "SELL",
					RetailPortfolioID:  "eec99928-3d46-5b38-a1ca-122d0e5fb7f0",
					FillSource:         "FILL_SOURCE_UNKNOWN",
					OrderDataSource:    "ORDER_DATA_SOURCE_UNKNOWN",
					FutureLegs:         []FillFutureLeg{},
					OptionLegs:         json.RawMessage(`[]`),
					ProductType:        "UNKNOWN_PRODUCT_TYPE",
				},
				{
					EntryID:            "49e5397c27030f629a3f02c8a359022fa295b96681e1a630b4dd94eca3837582",
					TradeID:            "0b639999-60d1-4b9c-bf2e-90b3b5f48dfd",
					OrderID:            "439e8aa4-47ab-48f6-bb44-33907960e92f",
					TradeTime:          time.Date(2024, 6, 18, 19, 59, 10, 102220957, time.UTC),
					TradeType:          "FILL",
					Price:              60000.44,
					Size:               1,
					Commission:         480.00352,
					ProductID:          "BTC-USD",
					SequenceTimestamp:  time.Date(2024, 6, 18, 19, 59, 10, 111915000, time.UTC),
					LiquidityIndicator: "UNKNOWN_LIQUIDITY_INDICATOR",
					UserID:             "eec99928-3d46-5b38-a1ca-122d0e5fb7f0",
					Side:               "BUY",
					RetailPortfolioID:  "eec99928-3d46-5b38-a1ca-122d0e5fb7f0",
					FillSource:         "FILL_SOURCE_UNKNOWN",
					OrderDataSource:    "ORDER_DATA_SOURCE_UNKNOWN",
					FutureLegs:         []FillFutureLeg{},
					OptionLegs:         json.RawMessage(`[]`),
					ProductType:        "UNKNOWN_PRODUCT_TYPE",
				},
				{
					EntryID:            "dc91f6cd8c78f47685530484bf34585769b062f8fd1497e0e503388928d69614",
					TradeID:            "2a013f5a-f8e1-4f73-b15b-336fcf4d88d6",
					OrderID:            "48c3142a-3601-40b0-bb42-ac88872fe40a",
					TradeTime:          time.Date(2024, 6, 18, 19, 58, 42, 263681859, time.UTC),
					TradeType:          "FILL",
					Price:              61000.44,
					Size:               1,
					Commission:         488.00352,
					ProductID:          "BTC-USD",
					SequenceTimestamp:  time.Date(2024, 6, 18, 19, 58, 42, 265863000, time.UTC),
					LiquidityIndicator: "UNKNOWN_LIQUIDITY_INDICATOR",
					UserID:             "eec99928-3d46-5b38-a1ca-122d0e5fb7f0",
					Side:               "BUY",
					RetailPortfolioID:  "eec99928-3d46-5b38-a1ca-122d0e5fb7f0",
					FillSource:         "FILL_SOURCE_UNKNOWN",
					OrderDataSource:    "ORDER_DATA_SOURCE_UNKNOWN",
					FutureLegs:         []FillFutureLeg{},
					OptionLegs:         json.RawMessage(`[]`),
					ProductType:        "UNKNOWN_PRODUCT_TYPE",
				},
			},
		}
		assert.Equal(t, exp, resp, "ListFills should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Fills), 5, "ListFills should return at most the limited fills")
	}

	// The mock joins a repeated param's values before matching them, so a test server checks they are repeated
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "ListFills should send a GET request")
		assert.Equal(t, "/api/v3/brokerage/orders/historical/fills", r.URL.Path, "ListFills should request the fills")
		query := r.URL.Query()
		assert.Equal(t, []string{"a", "b"}, query["order_ids"], "ListFills should repeat the order IDs param")
		assert.Equal(t, []string{"BTC", "ETH"}, query["asset_filters"], "ListFills should repeat the asset filters param")
		assert.Equal(t, []string{ProductTypeSpot, ProductTypeFuture}, query["product_types"], "ListFills should repeat the product types param")
		_, err := w.Write([]byte(`{"fills":[],"cursor":"","proof_token_required":false}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	resp, err = ex.ListFills(t.Context(), &ListFillsRequest{OrderIDs: []string{"a", "b"}, AssetFilters: []currency.Code{currency.BTC, currency.ETH}, ProductTypes: []string{ProductTypeSpot, ProductTypeFuture}})
	require.NoError(t, err, "ListFills must not error")
	assert.Equal(t, &ListFillsResponse{Fills: []Fill{}}, resp, "ListFills should decode an empty page")

	_, err = newResponseServerExchange(t, `null`).ListFills(t.Context(), &ListFillsRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "ListFills should error for a null response")
	_, err = newUnavailableServerExchange(t).ListFills(t.Context(), &ListFillsRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListFills should return the request failure")
}

func TestListOrders(t *testing.T) {
	t.Parallel()
	_, err := e.ListOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ListOrders must reject a nil request")
	_, err = e.ListOrders(t.Context(), &ListOrdersRequest{OrderStatuses: []string{"ACTIVE"}})
	require.ErrorIs(t, err, errInvalidOrderStatus, "ListOrders must refuse an unknown order status")

	skipLiveWithoutCredentials(t)
	resp, err := e.ListOrders(t.Context(), testListOrdersRequest())
	require.NoError(t, err, "ListOrders must not error")
	if mockTests {
		exp := &ListOrdersResponse{
			Orders: []Order{
				{
					OrderID:   "ff6e8dad-22c4-422d-bbf2-3c55b70e42d2",
					ProductID: "BTC-USD",
					UserID:    "eec99928-3d46-5b38-a1ca-122d0e5fb7f0",
					OrderConfiguration: &OrderConfiguration{
						LimitLimitGTC: &LimitLimitGTC{
							BaseSize:   1,
							LimitPrice: 70000.44,
						},
					},
					Side:                  "SELL",
					ClientOrderID:         "spot_order_2",
					Status:                "OPEN",
					TimeInForce:           "GOOD_UNTIL_CANCELLED",
					CreatedTime:           time.Date(2024, 6, 28, 15, 16, 47, 696748000, time.UTC),
					TriggerStatus:         "INVALID_ORDER_TYPE",
					OrderType:             "LIMIT",
					RejectReason:          "REJECT_REASON_UNSPECIFIED",
					ProductType:           "SPOT",
					OrderPlacementSource:  "RETAIL_ADVANCED",
					OutstandingHoldAmount: 1,
					EditHistory:           []OrderEdit{},
					MarginType:            "UNKNOWN_MARGIN_TYPE",
					RetailPortfolioID:     "eec99928-3d46-5b38-a1ca-122d0e5fb7f0",
					CostBasisMethod:       "COST_BASIS_METHOD_UNSPECIFIED",
					DisplayedOrderConfig:  "UNKNOWN_DISPLAYED_ORDER_CONFIG",
					EquityTradingSession:  "UNKNOWN_EQUITY_TRADING_SESSION",
					PredictionSide:        "PREDICTION_SIDE_UNKNOWN",
					OrderDataSource:       "ORDER_DATA_SOURCE_UNKNOWN",
					FundingSource:         "FUNDING_SOURCE_UNSPECIFIED",
					OptionLegs:            json.RawMessage(`[]`),
					Holds:                 json.RawMessage(`null`),
					ComboLegs:             json.RawMessage(`[]`),
				},
			},
		}
		assert.Equal(t, exp, resp, "ListOrders should decode every field")
	} else {
		for i := range resp.Orders {
			assert.Equalf(t, OrderStatusOpen, resp.Orders[i].Status, "ListOrders should return only open orders, not order %s", resp.Orders[i].OrderID)
		}
	}

	// The mock joins a repeated param's values before matching them, so a test server checks they are repeated
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "ListOrders should send a GET request")
		assert.Equal(t, "/api/v3/brokerage/orders/historical/batch", r.URL.Path, "ListOrders should request the orders")
		query := r.URL.Query()
		assert.Equal(t, []string{"BTC-USD", "ETH-USD"}, query["product_ids"], "ListOrders should repeat the product IDs param")
		assert.Equal(t, []string{OrderStatusOpen, OrderStatusPending}, query["order_status"], "ListOrders should repeat the order status param")
		assert.Equal(t, []string{TimeInForceGoodUntilCancelled, TimeInForceFillOrKill}, query["time_in_forces"], "ListOrders should repeat the times in force param")
		_, err := w.Write([]byte(`{"orders":[],"sequence":"0","has_next":false,"cursor":"","proof_token_required":false}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	resp, err = ex.ListOrders(t.Context(), &ListOrdersRequest{
		ProductIDs:    []string{"BTC-USD", "ETH-USD"},
		OrderStatuses: []string{OrderStatusOpen, OrderStatusPending},
		TimeInForces:  []string{TimeInForceGoodUntilCancelled, TimeInForceFillOrKill},
	})
	require.NoError(t, err, "ListOrders must not error")
	assert.Equal(t, &ListOrdersResponse{Orders: []Order{}}, resp, "ListOrders should decode an empty page")

	_, err = newResponseServerExchange(t, `null`).ListOrders(t.Context(), &ListOrdersRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "ListOrders should error for a null response")
	_, err = newUnavailableServerExchange(t).ListOrders(t.Context(), &ListOrdersRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListOrders should return the request failure")
}

func TestPreviewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PreviewOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PreviewOrder must reject a nil request")
	req := testPreviewOrderRequest(5)
	req.Side = ""
	_, err = e.PreviewOrder(t.Context(), req)
	require.ErrorIs(t, err, errInvalidOrderSide, "PreviewOrder must require a side")

	// A preview places nothing, so it only needs credentials
	skipLiveWithoutCredentials(t)
	resp, err := e.PreviewOrder(t.Context(), testPreviewOrderRequest(5))
	if mockTests {
		require.NoError(t, err, "PreviewOrder must not error")
		exp := &PreviewOrderResponse{
			OrderTotal:      5.04,
			CommissionTotal: 0.04,
			Errors:          []string{},
			Warnings:        []string{},
			QuoteSize:       5,
			BaseSize:        0.0001,
			BestBid:         65769.37,
			BestAsk:         65769.38,
			PreviewID:       "d5524676-73b5-4616-abe8-3abcf6bf001f",
			FillLikelihood:  "FILL_LIKELIHOOD_UNSPECIFIED",
		}
		assert.Equal(t, exp, resp, "PreviewOrder should decode every field")

		resp, err = e.PreviewOrder(t.Context(), testPreviewOrderRequest(5000000))
		assert.ErrorIs(t, err, errOrderPreviewFailed, "PreviewOrder should report an order that would fail")
		exp.Errors = []string{"PREVIEW_INSUFFICIENT_FUND"}
		assert.Equal(t, exp, resp, "PreviewOrder should return the response reporting the failure")
	} else {
		// The account may lack the funds to place the order, which the preview reports along with its totals
		if err != nil {
			require.ErrorIs(t, err, errOrderPreviewFailed, "PreviewOrder must only report the order's failures")
		}
		require.NotNil(t, resp, "PreviewOrder must return the preview")
		assert.Positive(t, resp.QuoteSize.Float64(), "PreviewOrder should return the order's quote size")
	}

	_, err = newResponseServerExchange(t, `null`).PreviewOrder(t.Context(), testPreviewOrderRequest(5))
	assert.ErrorIs(t, err, common.ErrNoResponse, "PreviewOrder should error for a null response")
	_, err = newUnavailableServerExchange(t).PreviewOrder(t.Context(), testPreviewOrderRequest(5))
	assert.ErrorIs(t, err, request.ErrBadStatus, "PreviewOrder should return the request failure")
}

func TestOrderNotFoundError(t *testing.T) {
	t.Parallel()
	notFound := &APIError{ErrorType: "NOT_FOUND", Message: "order with this orderID was not found"}
	err := orderNotFoundError(notFound)
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "orderNotFoundError should mark an order Coinbase does not know")
	assert.ErrorIs(t, err, notFound, "orderNotFoundError should keep the error Coinbase reported")
	err = orderNotFoundError(fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, notFound))
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "orderNotFoundError should mark a wrapped NOT_FOUND error")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "orderNotFoundError should keep the request failure")
	invalidArgument := &APIError{ErrorType: "INVALID_ARGUMENT", Message: "invalid OrderId {order_id} passed to endpoint"}
	assert.Same(t, invalidArgument, orderNotFoundError(invalidArgument), "orderNotFoundError should not take an invalid argument for a missing order")
	other := errors.New("unrelated")
	assert.Same(t, other, orderNotFoundError(other), "orderNotFoundError should return any other error as it is")
	assert.NoError(t, orderNotFoundError(nil), "orderNotFoundError should not make an error of none")
}

func TestCreateOrderRequestBody(t *testing.T) {
	t.Parallel()
	_, err := (*CreateOrderRequest)(nil).body()
	require.ErrorIs(t, err, common.ErrNilPointer, "body must reject a nil request")
	req := testCreateOrderRequest()
	req.ClientOrderID = ""
	_, err = req.body()
	assert.ErrorIs(t, err, errClientOrderIDRequired, "body should require a client order ID")
	req.ClientOrderID = strings.Repeat("a", maximumClientOrderIDLength+1)
	_, err = req.body()
	assert.ErrorIs(t, err, errClientOrderIDTooLong, "body should refuse a client order ID over 128 characters")
	// The limit counts characters, so 128 two byte characters are accepted
	req.ClientOrderID = strings.Repeat("é", maximumClientOrderIDLength)
	_, err = req.body()
	assert.NoError(t, err, "body should accept a client order ID of 128 characters")
	req = testCreateOrderRequest()
	req.SORPreference = "SOR_ON"
	_, err = req.body()
	assert.ErrorIs(t, err, errInvalidSORPreference, "body should refuse an unknown smart order routing preference")
	req = testCreateOrderRequest()
	req.ProductID = ""
	_, err = req.body()
	assert.ErrorIs(t, err, errProductIDRequired, "body should validate the fields Preview Order shares")

	aedt := time.FixedZone("AEDT", 11*60*60)
	body, err := (&CreateOrderRequest{
		ClientOrderID:              "client-order",
		ProductID:                  "BIT-30OCT26-CDE",
		Side:                       OrderSideBuy,
		OrderConfiguration:         &LimitLimitGTDRequest{BaseSize: 2, LimitPrice: 80000, EndTime: time.Date(2026, 10, 20, 11, 0, 0, 0, aedt), PostOnly: true},
		Leverage:                   2.5,
		MarginType:                 OrderMarginTypeIsolated,
		PreviewID:                  "preview",
		AttachedOrderConfiguration: &TriggerBracketGTCRequest{LimitPrice: 90000, StopTriggerPrice: 70000},
		SORPreference:              SORPreferenceDisabled,
		PredictionMetadata:         &PredictionMetadataRequest{PredictionSide: PredictionSideYes, PreviewOrderEstimatedAverageFilledPrice: 0.42, SupportsFractionalBaseSize: true},
		EquityOrderMetadata:        &EquityOrderMetadata{EquityTradingSession: EquityTradingSessionNormal, DisplayedOrderConfig: DisplayedOrderConfigLimitGFD},
		CostBasisMethod:            CostBasisMethodFIFO,
	}).body()
	require.NoError(t, err, "body must not error")
	raw, err := json.Marshal(body)
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{
		"client_order_id":"client-order",
		"product_id":"BIT-30OCT26-CDE",
		"side":"BUY",
		"order_configuration":{"limit_limit_gtd":{"base_size":"2","limit_price":"80000","end_time":"2026-10-20T00:00:00Z","post_only":true}},
		"leverage":"2.5",
		"margin_type":"ISOLATED",
		"preview_id":"preview",
		"attached_order_configuration":{"trigger_bracket_gtc":{"limit_price":"90000","stop_trigger_price":"70000"}},
		"sor_preference":"SOR_DISABLED",
		"prediction_metadata":{"prediction_side":"PREDICTION_SIDE_YES","preview_order_est_average_filled_price":"0.42","supports_fractional_base_size":true},
		"equity_order_metadata":{"equity_trading_session":"EQUITY_TRADING_SESSION_NORMAL","displayed_order_config":"LIMIT_GFD"},
		"cost_basis_method":"COST_BASIS_METHOD_FIFO"
	}`, string(raw), "body should send every field, times in UTC")
}

func TestPreviewOrderRequestBody(t *testing.T) {
	t.Parallel()
	_, err := (*PreviewOrderRequest)(nil).body()
	require.ErrorIs(t, err, common.ErrNilPointer, "body must reject a nil request")
	for _, tc := range []struct {
		name   string
		modify func(*PreviewOrderRequest)
		err    error
	}{
		{name: "no product ID", modify: func(r *PreviewOrderRequest) { r.ProductID = "" }, err: errProductIDRequired},
		{name: "no side", modify: func(r *PreviewOrderRequest) { r.Side = "" }, err: errInvalidOrderSide},
		{name: "lower case side", modify: func(r *PreviewOrderRequest) { r.Side = "buy" }, err: errInvalidOrderSide},
		{name: "no configuration", modify: func(r *PreviewOrderRequest) { r.OrderConfiguration = nil }, err: errOrderConfigurationRequired},
		{name: "typed nil configuration", modify: func(r *PreviewOrderRequest) { r.OrderConfiguration = (*MarketMarketIOCRequest)(nil) }, err: errOrderConfigurationRequired},
		{name: "invalid configuration", modify: func(r *PreviewOrderRequest) {
			r.OrderConfiguration = &LimitLimitGTCRequest{BaseSize: 1, LimitPrice: -1}
		}, err: errInvalidPrice},
		{name: "ineligible attached configuration", modify: func(r *PreviewOrderRequest) { r.AttachedOrderConfiguration = &LimitLimitGTCRequest{LimitPrice: 1} }, err: errInvalidAttachedOrderConfiguration},
		{name: "negative leverage", modify: func(r *PreviewOrderRequest) { r.Leverage = -1 }, err: errInvalidOrderLeverage},
		{name: "unknown margin type", modify: func(r *PreviewOrderRequest) { r.MarginType = "MARGIN_TYPE_CROSS" }, err: errInvalidOrderMarginType},
		{name: "unknown cost basis method", modify: func(r *PreviewOrderRequest) { r.CostBasisMethod = "FIFO" }, err: errInvalidCostBasisMethod},
		{name: "unknown equity session", modify: func(r *PreviewOrderRequest) {
			r.EquityOrderMetadata = &EquityOrderMetadata{EquityTradingSession: "NORMAL"}
		}, err: errInvalidEquityTradingSession},
		{name: "unknown prediction side", modify: func(r *PreviewOrderRequest) { r.PredictionMetadata = &PredictionMetadataRequest{PredictionSide: "YES"} }, err: errInvalidPredictionSide},
	} {
		req := testPreviewOrderRequest(5)
		tc.modify(req)
		_, err := req.body()
		assert.ErrorIsf(t, err, tc.err, "body should refuse a request with %s", tc.name)
	}

	body, err := (&PreviewOrderRequest{
		ProductID:                  testSpotProductID,
		Side:                       OrderSideSell,
		OrderConfiguration:         &StopLimitStopLimitGTCRequest{BaseSize: 0.5, LimitPrice: 60000, StopPrice: 61000, StopDirection: StopDirectionStopDown},
		Leverage:                   3,
		MarginType:                 OrderMarginTypeCross,
		AttachedOrderConfiguration: &TriggerBracketGTCRequest{LimitPrice: 50000, StopTriggerPrice: 65000},
		EquityOrderMetadata:        &EquityOrderMetadata{EquityTradingSession: EquityTradingSessionAfterHours},
		PredictionMetadata:         &PredictionMetadataRequest{PredictionSide: PredictionSideNo},
		CostBasisMethod:            CostBasisMethodHIFO,
	}).body()
	require.NoError(t, err, "body must not error")
	raw, err := json.Marshal(body)
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{
		"product_id":"BTC-USD",
		"side":"SELL",
		"order_configuration":{"stop_limit_stop_limit_gtc":{"base_size":"0.5","limit_price":"60000","stop_price":"61000","stop_direction":"STOP_DIRECTION_STOP_DOWN"}},
		"leverage":"3",
		"margin_type":"CROSS",
		"attached_order_configuration":{"trigger_bracket_gtc":{"limit_price":"50000","stop_trigger_price":"65000"}},
		"prediction_metadata":{"prediction_side":"PREDICTION_SIDE_NO"},
		"equity_order_metadata":{"equity_trading_session":"EQUITY_TRADING_SESSION_AFTER_HOURS"},
		"cost_basis_method":"COST_BASIS_METHOD_HIFO"
	}`, string(raw), "body should send every field, leaving out Create Order's own")
	raw, err = json.Marshal(&orderBody{ProductID: testSpotProductID, Side: OrderSideBuy, OrderConfiguration: orderConfigurationBody{"market_market_ioc": {QuoteSize: "5"}}})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"product_id":"BTC-USD","side":"BUY","order_configuration":{"market_market_ioc":{"quote_size":"5"}}}`, string(raw), "body should leave out every unset optional field")
}

func TestEditOrderRequestBody(t *testing.T) {
	t.Parallel()
	_, err := (*EditOrderRequest)(nil).body()
	require.ErrorIs(t, err, common.ErrNilPointer, "body must reject a nil request")
	for _, tc := range []struct {
		name   string
		modify func(*EditOrderRequest)
		err    error
	}{
		{name: "no order ID", modify: func(r *EditOrderRequest) { r.OrderID = "" }, err: errOrderIDRequired},
		{name: "no price", modify: func(r *EditOrderRequest) { r.Price = 0 }, err: errInvalidPrice},
		{name: "no size", modify: func(r *EditOrderRequest) { r.Size = 0 }, err: errInvalidAmount},
		{name: "negative stop price", modify: func(r *EditOrderRequest) { r.StopPrice = -1 }, err: errInvalidPrice},
		{name: "negative average entry price", modify: func(r *EditOrderRequest) { r.AverageEntryPrice = -1 }, err: errInvalidPrice},
		{name: "ineligible attached configuration", modify: func(r *EditOrderRequest) { r.AttachedOrderConfiguration = &TriggerBracketGTDRequest{LimitPrice: 1} }, err: errInvalidAttachedOrderConfiguration},
	} {
		req := testEditOrderRequest(testSandboxOrderID)
		tc.modify(req)
		_, err := req.body()
		assert.ErrorIsf(t, err, tc.err, "body should refuse an edit with %s", tc.name)
	}

	body, err := (&EditOrderRequest{
		OrderID:                    testSandboxOrderID,
		Price:                      19000,
		Size:                       0.001,
		AttachedOrderConfiguration: &StopLimitStopLimitGTCRequest{LimitPrice: 16500, StopPrice: 17000, StopDirection: StopDirectionStopDown},
		CancelAttachedOrder:        true,
		StopPrice:                  17000,
		AverageEntryPrice:          18000,
	}).body()
	require.NoError(t, err, "body must not error")
	raw, err := json.Marshal(body)
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{
		"order_id":"f898eaf4-6ffc-47be-a159-7ff292e5cdcf",
		"price":"19000",
		"size":"0.001",
		"attached_order_configuration":{"stop_limit_stop_limit_gtc":{"limit_price":"16500","stop_price":"17000","stop_direction":"STOP_DIRECTION_STOP_DOWN"}},
		"cancel_attached_order":true,
		"stop_price":"17000",
		"average_entry_price":"18000"
	}`, string(raw), "body should send every field")
	body, err = testEditOrderRequest(testSandboxOrderID).body()
	require.NoError(t, err, "body must not error")
	raw, err = json.Marshal(body)
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"order_id":"f898eaf4-6ffc-47be-a159-7ff292e5cdcf","price":"68000.44","size":"1"}`, string(raw), "body should leave out every unset optional field")
}

func TestEquityOrderMetadataValidate(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (*EquityOrderMetadata)(nil).validate(), "validate should leave nil metadata to Coinbase's defaults")
	assert.NoError(t, (&EquityOrderMetadata{EquityTradingSession: EquityTradingSessionPreMarket, DisplayedOrderConfig: DisplayedOrderConfigLimitGTC}).validate(), "validate should accept a documented session and time in force")
	assert.ErrorIs(t, (&EquityOrderMetadata{EquityTradingSession: "PRE_MARKET"}).validate(), errInvalidEquityTradingSession, "validate should refuse an unknown session")
	assert.ErrorIs(t, (&EquityOrderMetadata{DisplayedOrderConfig: "GTC"}).validate(), errInvalidDisplayedOrderConfig, "validate should refuse an unknown time in force")
}

func TestPredictionMetadataRequestBody(t *testing.T) {
	t.Parallel()
	body, err := (*PredictionMetadataRequest)(nil).body()
	require.NoError(t, err, "body must not error for nil metadata")
	assert.Nil(t, body, "body should send no metadata for nil metadata")
	body, err = (&PredictionMetadataRequest{PredictionSide: PredictionSideYes, PreviewOrderEstimatedAverageFilledPrice: 0.0000001, SupportsFractionalBaseSize: true}).body()
	require.NoError(t, err, "body must not error")
	assert.Equal(t, &predictionMetadataBody{PredictionSide: PredictionSideYes, PreviewOrderEstimatedAverageFilledPrice: "0.0000001", SupportsFractionalBaseSize: true}, body, "body should send every field, the price without exponent notation")
	_, err = (&PredictionMetadataRequest{PredictionSide: "NO"}).body()
	assert.ErrorIs(t, err, errInvalidPredictionSide, "body should refuse an unknown prediction side")
	_, err = (&PredictionMetadataRequest{PreviewOrderEstimatedAverageFilledPrice: -1}).body()
	assert.ErrorIs(t, err, errInvalidPrice, "body should refuse a negative price")
}

func TestIsOrderConfigurationSet(t *testing.T) {
	t.Parallel()
	assert.False(t, isOrderConfigurationSet(nil), "isOrderConfigurationSet should report a nil configuration unset")
	assert.False(t, isOrderConfigurationSet((*TriggerBracketGTCRequest)(nil)), "isOrderConfigurationSet should report a typed nil configuration unset")
	assert.True(t, isOrderConfigurationSet(&TriggerBracketGTCRequest{}), "isOrderConfigurationSet should report a configuration set")
}

func TestNewOrderConfigurationBody(t *testing.T) {
	t.Parallel()
	body, err := newOrderConfigurationBody(&MarketMarketIOCRequest{BaseSize: 1})
	require.NoError(t, err, "newOrderConfigurationBody must not error")
	assert.Equal(t, orderConfigurationBody{"market_market_ioc": {BaseSize: "1"}}, body, "newOrderConfigurationBody should key the configuration's body")
	_, err = newOrderConfigurationBody(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "newOrderConfigurationBody should refuse a nil configuration")
	_, err = newOrderConfigurationBody((*LimitLimitGTCRequest)(nil))
	assert.ErrorIs(t, err, common.ErrNilPointer, "newOrderConfigurationBody should refuse a typed nil configuration")
	_, err = newOrderConfigurationBody(&MarketMarketIOCRequest{BaseSize: -1})
	assert.ErrorIs(t, err, errInvalidAmount, "newOrderConfigurationBody should return the configuration's error")
}

func TestAttachedOrderConfigurationBody(t *testing.T) {
	t.Parallel()
	for _, configuration := range []OrderConfigurationRequest{nil, (*TriggerBracketGTCRequest)(nil)} {
		body, err := attachedOrderConfigurationBody(configuration)
		require.NoErrorf(t, err, "attachedOrderConfigurationBody must not error for %T", configuration)
		assert.Nilf(t, body, "attachedOrderConfigurationBody should attach nothing for %T", configuration)
	}
	body, err := attachedOrderConfigurationBody(&TriggerBracketGTCRequest{LimitPrice: 1600, StopTriggerPrice: 1300})
	require.NoError(t, err, "attachedOrderConfigurationBody must not error")
	assert.Equal(t, orderConfigurationBody{"trigger_bracket_gtc": {LimitPrice: "1600", StopTriggerPrice: "1300"}}, body, "attachedOrderConfigurationBody should send the trigger bracket")
	_, err = attachedOrderConfigurationBody(&TriggerBracketGTDRequest{LimitPrice: 1600, StopTriggerPrice: 1300})
	assert.ErrorIs(t, err, errInvalidAttachedOrderConfiguration, "attachedOrderConfigurationBody should refuse any configuration but a GTC trigger bracket")
	_, err = attachedOrderConfigurationBody(&TriggerBracketGTCRequest{BaseSize: 1, LimitPrice: 1600, StopTriggerPrice: 1300})
	assert.ErrorIs(t, err, errAttachedOrderSizeSet, "attachedOrderConfigurationBody should refuse a size, as the attached order takes its parent's")
	_, err = attachedOrderConfigurationBody(&TriggerBracketGTCRequest{LimitPrice: -1})
	assert.ErrorIs(t, err, errInvalidPrice, "attachedOrderConfigurationBody should return the configuration's error")
}

func TestEditAttachedOrderConfigurationBody(t *testing.T) {
	t.Parallel()
	for _, configuration := range []OrderConfigurationRequest{nil, (*LimitLimitGTCRequest)(nil)} {
		body, err := editAttachedOrderConfigurationBody(configuration)
		require.NoErrorf(t, err, "editAttachedOrderConfigurationBody must not error for %T", configuration)
		assert.Nilf(t, body, "editAttachedOrderConfigurationBody should set nothing for %T", configuration)
	}
	for _, configuration := range []OrderConfigurationRequest{
		&TriggerBracketGTCRequest{LimitPrice: 1600, StopTriggerPrice: 1300},
		&LimitLimitGTCRequest{BaseSize: 1, LimitPrice: 1600},
		&StopLimitStopLimitGTCRequest{BaseSize: 1, LimitPrice: 1250, StopPrice: 1300, StopDirection: StopDirectionStopDown},
	} {
		body, err := editAttachedOrderConfigurationBody(configuration)
		require.NoErrorf(t, err, "editAttachedOrderConfigurationBody must not error for %T", configuration)
		assert.Lenf(t, body, 1, "editAttachedOrderConfigurationBody should send %T", configuration)
	}
	for _, configuration := range []OrderConfigurationRequest{&MarketMarketIOCRequest{BaseSize: 1}, &TriggerBracketGTDRequest{LimitPrice: 1}, &ScaledLimitGTCRequest{BaseSize: 1}} {
		_, err := editAttachedOrderConfigurationBody(configuration)
		assert.ErrorIsf(t, err, errInvalidAttachedOrderConfiguration, "editAttachedOrderConfigurationBody should refuse %T", configuration)
	}
	_, err := editAttachedOrderConfigurationBody(&LimitLimitGTCRequest{LimitPrice: -1})
	assert.ErrorIs(t, err, errInvalidPrice, "editAttachedOrderConfigurationBody should return the configuration's error")
}

func TestOrderConfigurationRequests(t *testing.T) {
	t.Parallel()
	end := time.Date(2026, 10, 20, 11, 0, 0, 0, time.FixedZone("AEDT", 11*60*60))
	start := end.Add(-time.Hour)
	for _, tc := range []struct {
		configuration OrderConfigurationRequest
		key           string
		body          string
	}{
		{configuration: &MarketMarketIOCRequest{QuoteSize: 10}, key: "market_market_ioc", body: `{"quote_size":"10"}`},
		{configuration: &MarketMarketFOKRequest{BaseSize: 0.001}, key: "market_market_fok", body: `{"base_size":"0.001"}`},
		{configuration: &SORLimitIOCRequest{QuoteSize: 5, BaseSize: 0.001, LimitPrice: 10000}, key: "sor_limit_ioc", body: `{"quote_size":"5","base_size":"0.001","limit_price":"10000"}`},
		{configuration: &LimitLimitGTCRequest{QuoteSize: 10, LimitPrice: 10000, PostOnly: true}, key: "limit_limit_gtc", body: `{"quote_size":"10","limit_price":"10000","post_only":true}`},
		{configuration: &LimitLimitGTDRequest{BaseSize: 0.001, LimitPrice: 10000, EndTime: end, PostOnly: true}, key: "limit_limit_gtd", body: `{"base_size":"0.001","limit_price":"10000","end_time":"2026-10-20T00:00:00Z","post_only":true}`},
		{configuration: &LimitLimitFOKRequest{QuoteSize: 10, LimitPrice: 10000}, key: "limit_limit_fok", body: `{"quote_size":"10","limit_price":"10000"}`},
		{
			configuration: &TWAPLimitGTDRequest{BaseSize: 10, StartTime: start, EndTime: end, LimitPrice: 10000, NumberOfBuckets: 5, BucketSize: 2, BucketDuration: 300 * time.Second},
			key:           "twap_limit_gtd",
			body:          `{"base_size":"10","start_time":"2026-10-19T23:00:00Z","end_time":"2026-10-20T00:00:00Z","limit_price":"10000","number_buckets":"5","bucket_size":"2","bucket_duration":"300s"}`,
		},
		{
			configuration: &StopLimitStopLimitGTCRequest{BaseSize: 0.001, LimitPrice: 10000, StopPrice: 20000, StopDirection: StopDirectionStopUp},
			key:           "stop_limit_stop_limit_gtc",
			body:          `{"base_size":"0.001","limit_price":"10000","stop_price":"20000","stop_direction":"STOP_DIRECTION_STOP_UP"}`,
		},
		{
			configuration: &StopLimitStopLimitGTDRequest{BaseSize: 0.001, LimitPrice: 10000, StopPrice: 20000, EndTime: end, StopDirection: StopDirectionStopDown},
			key:           "stop_limit_stop_limit_gtd",
			body:          `{"base_size":"0.001","limit_price":"10000","stop_price":"20000","end_time":"2026-10-20T00:00:00Z","stop_direction":"STOP_DIRECTION_STOP_DOWN"}`,
		},
		{
			configuration: &TriggerBracketGTCRequest{BaseSize: 0.001, LimitPrice: 10000, StopTriggerPrice: 20000},
			key:           "trigger_bracket_gtc",
			body:          `{"base_size":"0.001","limit_price":"10000","stop_trigger_price":"20000"}`,
		},
		{
			configuration: &TriggerBracketGTDRequest{BaseSize: 0.001, LimitPrice: 10000, StopTriggerPrice: 20000, EndTime: end},
			key:           "trigger_bracket_gtd",
			body:          `{"base_size":"0.001","limit_price":"10000","stop_trigger_price":"20000","end_time":"2026-10-20T00:00:00Z"}`,
		},
		{
			configuration: &ScaledLimitGTCRequest{
				Orders:            []LimitLimitGTCRequest{{BaseSize: 0.5, LimitPrice: 10000}, {BaseSize: 0.5, LimitPrice: 11000, PostOnly: true}},
				QuoteSize:         10500,
				BaseSize:          1,
				NumberOfOrders:    2,
				MinimumPrice:      10000,
				MaximumPrice:      11000,
				PriceDistribution: ScaledPriceDistributionCustom,
				SizeDistribution:  ScaledSizeDistributionCustom,
				SizeDifference:    0.1,
				SizeRatio:         1.5,
			},
			key:  "scaled_limit_gtc",
			body: `{"orders":[{"base_size":"0.5","limit_price":"10000"},{"base_size":"0.5","limit_price":"11000","post_only":true}],"quote_size":"10500","base_size":"1","num_orders":2,"min_price":"10000","max_price":"11000","price_distribution":"CUSTOM_PRICE_DISTRIBUTION","size_distribution":"CUSTOM_SIZE_DISTRIBUTION","size_diff":"0.1","size_ratio":"1.5"}`,
		},
		{configuration: &ScaledLimitGTCRequest{BaseSize: 1, NumberOfOrders: 3, MinimumPrice: 10000, MaximumPrice: 11000}, key: "scaled_limit_gtc", body: `{"base_size":"1","num_orders":3,"min_price":"10000","max_price":"11000"}`},
	} {
		key, leg, err := tc.configuration.orderConfiguration()
		require.NoErrorf(t, err, "orderConfiguration must not error for %T", tc.configuration)
		assert.Equalf(t, tc.key, key, "orderConfiguration should key %T", tc.configuration)
		raw, err := json.Marshal(leg)
		require.NoError(t, err, "Marshal must not error")
		assert.JSONEqf(t, tc.body, string(raw), "orderConfiguration should send every field of %T, times in UTC", tc.configuration)
	}

	for _, tc := range []struct {
		configuration OrderConfigurationRequest
		err           error
	}{
		{configuration: &MarketMarketIOCRequest{QuoteSize: -1}, err: errInvalidAmount},
		{configuration: &MarketMarketFOKRequest{BaseSize: -1}, err: errInvalidAmount},
		{configuration: &SORLimitIOCRequest{BaseSize: -1}, err: errInvalidAmount},
		{configuration: &SORLimitIOCRequest{LimitPrice: -1}, err: errInvalidPrice},
		{configuration: &LimitLimitGTCRequest{QuoteSize: -1}, err: errInvalidAmount},
		{configuration: &LimitLimitGTCRequest{LimitPrice: -1}, err: errInvalidPrice},
		{configuration: &LimitLimitGTDRequest{BaseSize: -1}, err: errInvalidAmount},
		{configuration: &LimitLimitGTDRequest{LimitPrice: -1}, err: errInvalidPrice},
		{configuration: &LimitLimitFOKRequest{BaseSize: -1}, err: errInvalidAmount},
		{configuration: &LimitLimitFOKRequest{LimitPrice: -1}, err: errInvalidPrice},
		{configuration: &TWAPLimitGTDRequest{BucketSize: -1}, err: errInvalidAmount},
		{configuration: &TWAPLimitGTDRequest{LimitPrice: -1}, err: errInvalidPrice},
		{configuration: &TWAPLimitGTDRequest{StartTime: end, EndTime: start}, err: common.ErrStartAfterEnd},
		{configuration: &TWAPLimitGTDRequest{BucketDuration: -time.Second}, err: errInvalidDuration},
		{configuration: &StopLimitStopLimitGTCRequest{BaseSize: -1}, err: errInvalidAmount},
		{configuration: &StopLimitStopLimitGTCRequest{StopPrice: -1}, err: errInvalidPrice},
		{configuration: &StopLimitStopLimitGTCRequest{StopDirection: "STOP_UP"}, err: errInvalidStopDirection},
		{configuration: &StopLimitStopLimitGTDRequest{BaseSize: -1}, err: errInvalidAmount},
		{configuration: &StopLimitStopLimitGTDRequest{LimitPrice: -1}, err: errInvalidPrice},
		{configuration: &StopLimitStopLimitGTDRequest{StopDirection: "STOP_DOWN"}, err: errInvalidStopDirection},
		{configuration: &TriggerBracketGTCRequest{BaseSize: -1}, err: errInvalidAmount},
		{configuration: &TriggerBracketGTCRequest{StopTriggerPrice: -1}, err: errInvalidPrice},
		{configuration: &TriggerBracketGTDRequest{BaseSize: -1}, err: errInvalidAmount},
		{configuration: &TriggerBracketGTDRequest{LimitPrice: -1}, err: errInvalidPrice},
		{configuration: &ScaledLimitGTCRequest{QuoteSize: -1}, err: errInvalidAmount},
		{configuration: &ScaledLimitGTCRequest{MaximumPrice: -1}, err: errInvalidPrice},
		{configuration: &ScaledLimitGTCRequest{PriceDistribution: "LINEAR"}, err: errInvalidScaledPriceDistribution},
		{configuration: &ScaledLimitGTCRequest{SizeDistribution: "EVEN"}, err: errInvalidScaledSizeDistribution},
		{configuration: &ScaledLimitGTCRequest{Orders: []LimitLimitGTCRequest{{BaseSize: 1, LimitPrice: 1}, {LimitPrice: -1}}}, err: errInvalidPrice},
	} {
		_, _, err := tc.configuration.orderConfiguration()
		assert.ErrorIsf(t, err, tc.err, "orderConfiguration should refuse %+v", tc.configuration)
	}
}

func TestListFillsParams(t *testing.T) {
	t.Parallel()
	_, err := listFillsParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "listFillsParams must reject a nil request")
	for _, tc := range []struct {
		name string
		req  *ListFillsRequest
		err  error
	}{
		{name: "a reversed window", req: &ListFillsRequest{StartSequenceTimestamp: time.Unix(2, 0), EndSequenceTimestamp: time.Unix(1, 0)}, err: common.ErrStartAfterEnd},
		{name: "an unknown sort field", req: &ListFillsRequest{SortBy: "SIZE"}, err: errInvalidFillsSortBy},
		{name: "an unknown order type", req: &ListFillsRequest{OrderTypes: []string{OrderTypeLimit, "ICEBERG"}}, err: errInvalidOrderType},
		{name: "an unknown side", req: &ListFillsRequest{OrderSide: "Buy"}, err: errInvalidOrderSide},
		{name: "an unknown product type", req: &ListFillsRequest{ProductTypes: []string{"FUTURES"}}, err: errInvalidOrderProductType},
	} {
		_, err := listFillsParams(tc.req)
		assert.ErrorIsf(t, err, tc.err, "listFillsParams should refuse %s", tc.name)
	}

	params, err := listFillsParams(&ListFillsRequest{
		OrderIDs:               []string{"o1", "o2"},
		TradeIDs:               []string{"t1"},
		ProductIDs:             []string{"BTC-USD", "ETH-USD"},
		StartSequenceTimestamp: time.Date(2026, 10, 19, 11, 0, 0, 0, time.FixedZone("AEDT", 11*60*60)),
		EndSequenceTimestamp:   time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
		Limit:                  50,
		Cursor:                 "789100",
		SortBy:                 FillsSortByTradeTime,
		AssetFilters:           []currency.Code{currency.BTC, currency.NewCode("eth")},
		OrderTypes:             []string{OrderTypeLimit, OrderTypeMarket},
		OrderSide:              OrderSideBuy,
		ProductTypes:           []string{ProductTypeSpot, "FUTURE_GROUP"},
		ProofToken:             "proof",
	})
	require.NoError(t, err, "listFillsParams must not error")
	exp := url.Values{
		"order_ids":                {"o1", "o2"},
		"trade_ids":                {"t1"},
		"product_ids":              {"BTC-USD", "ETH-USD"},
		"start_sequence_timestamp": {"2026-10-19T00:00:00Z"},
		"end_sequence_timestamp":   {"2026-10-20T00:00:00Z"},
		"limit":                    {"50"},
		"cursor":                   {"789100"},
		"sort_by":                  {"TRADE_TIME"},
		"asset_filters":            {"BTC", "ETH"},
		"order_types":              {"LIMIT", "MARKET"},
		"order_side":               {"BUY"},
		"product_types":            {"SPOT", "FUTURE_GROUP"},
		"proof_token":              {"proof"},
	}
	assert.Equal(t, exp, params, "listFillsParams should send every field, repeating list params and times in UTC")
	params, err = listFillsParams(&ListFillsRequest{})
	require.NoError(t, err, "listFillsParams must not error")
	assert.Empty(t, params, "listFillsParams should send nothing for an empty request")
}

func TestListOrdersParams(t *testing.T) {
	t.Parallel()
	_, err := listOrdersParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "listOrdersParams must reject a nil request")
	for _, tc := range []struct {
		name string
		req  *ListOrdersRequest
		err  error
	}{
		{name: "a reversed window", req: &ListOrdersRequest{StartDate: time.Unix(2, 0), EndDate: time.Unix(1, 0)}, err: common.ErrStartAfterEnd},
		{name: "an unknown product type", req: &ListOrdersRequest{ProductType: "FUTURES"}, err: errInvalidOrderProductType},
		{name: "an unknown order status", req: &ListOrdersRequest{OrderStatuses: []string{OrderStatusOpen, "ACTIVE"}}, err: errInvalidOrderStatus},
		{name: "an empty order status", req: &ListOrdersRequest{OrderStatuses: []string{""}}, err: errInvalidOrderStatus},
		{name: "an unknown time in force", req: &ListOrdersRequest{TimeInForces: []string{"GTC"}}, err: errInvalidOrderTimeInForce},
		{name: "an unknown order type", req: &ListOrdersRequest{OrderTypes: []string{OrderTypeTakeProfitStopLoss}}, err: errInvalidOrderType},
		{name: "an unknown side", req: &ListOrdersRequest{OrderSide: "sell"}, err: errInvalidOrderSide},
		{name: "an unknown placement source", req: &ListOrdersRequest{OrderPlacementSource: "ADVANCED"}, err: errInvalidOrderPlacementSource},
		{name: "an unknown contract expiry type", req: &ListOrdersRequest{ContractExpiryType: "EXPIRED"}, err: errInvalidOrderContractExpiryType},
		{name: "an unknown sort field", req: &ListOrdersRequest{SortBy: FillsSortByTradeTime}, err: errInvalidOrdersSortBy},
	} {
		_, err := listOrdersParams(tc.req)
		assert.ErrorIsf(t, err, tc.err, "listOrdersParams should refuse %s", tc.name)
	}

	simplified := false
	params, err := listOrdersParams(&ListOrdersRequest{
		OrderIDs:                           []string{"o1", "o2"},
		ProductIDs:                         []string{"BIT-30OCT26-CDE"},
		ProductType:                        ProductTypeFuture,
		OrderStatuses:                      []string{OrderStatusOpen, OrderStatusCancelled},
		TimeInForces:                       []string{TimeInForceGoodUntilDateTime, TimeInForceImmediateOrCancel},
		OrderTypes:                         []string{OrderTypeBracket, OrderTypeScaled},
		OrderSide:                          OrderSideSell,
		StartDate:                          time.Date(2026, 10, 19, 11, 0, 0, 0, time.FixedZone("AEDT", 11*60*60)),
		EndDate:                            time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
		OrderPlacementSource:               OrderPlacementSourceRetailSimple,
		ContractExpiryType:                 ContractExpiryTypePerpetual,
		AssetFilters:                       []currency.Code{currency.NewCode("btc")},
		Limit:                              100,
		Cursor:                             "789100",
		SortBy:                             OrdersSortByLastFillTime,
		UseSimplifiedTotalValueCalculation: &simplified,
		ProofToken:                         "proof",
	})
	require.NoError(t, err, "listOrdersParams must not error")
	exp := url.Values{
		"order_ids":                              {"o1", "o2"},
		"product_ids":                            {"BIT-30OCT26-CDE"},
		"product_type":                           {"FUTURE"},
		"order_status":                           {"OPEN", "CANCELLED"},
		"time_in_forces":                         {"GOOD_UNTIL_DATE_TIME", "IMMEDIATE_OR_CANCEL"},
		"order_types":                            {"BRACKET", "SCALED"},
		"order_side":                             {"SELL"},
		"start_date":                             {"2026-10-19T00:00:00Z"},
		"end_date":                               {"2026-10-20T00:00:00Z"},
		"order_placement_source":                 {"RETAIL_SIMPLE"},
		"contract_expiry_type":                   {"PERPETUAL"},
		"asset_filters":                          {"BTC"},
		"limit":                                  {"100"},
		"cursor":                                 {"789100"},
		"sort_by":                                {"LAST_FILL_TIME"},
		"use_simplified_total_value_calculation": {"false"},
		"proof_token":                            {"proof"},
	}
	assert.Equal(t, exp, params, "listOrdersParams should send every field, repeating list params, times in UTC and an explicit false")
	params, err = listOrdersParams(&ListOrdersRequest{})
	require.NoError(t, err, "listOrdersParams must not error")
	assert.Empty(t, params, "listOrdersParams should send nothing for an empty request")
}

func TestAddOrderParams(t *testing.T) {
	t.Parallel()
	params := map[string][]string{}
	addOrderParams(params, "order_ids", []string{"a", "b"})
	addOrderParams(params, "trade_ids", nil)
	assert.Equal(t, map[string][]string{"order_ids": {"a", "b"}}, params, "addOrderParams should add each value and nothing for none")
}

func TestCheckOrderEnum(t *testing.T) {
	t.Parallel()
	assert.NoError(t, checkOrderEnum("", errInvalidOrderSide, orderSides), "checkOrderEnum should leave an unset value to Coinbase's default")
	assert.NoError(t, checkOrderEnum(OrderSideSell, errInvalidOrderSide, orderSides), "checkOrderEnum should accept a documented value")
	assert.ErrorIs(t, checkOrderEnum("sell", errInvalidOrderSide, orderSides), errInvalidOrderSide, "checkOrderEnum should refuse an undocumented value")
}

func TestCheckOrderEnums(t *testing.T) {
	t.Parallel()
	assert.NoError(t, checkOrderEnums(nil, errInvalidOrderStatus, orderStatuses), "checkOrderEnums should accept no values")
	assert.NoError(t, checkOrderEnums([]string{OrderStatusOpen, OrderStatusFilled}, errInvalidOrderStatus, orderStatuses), "checkOrderEnums should accept documented values")
	assert.ErrorIs(t, checkOrderEnums([]string{OrderStatusOpen, "open"}, errInvalidOrderStatus, orderStatuses), errInvalidOrderStatus, "checkOrderEnums should refuse an undocumented value")
	assert.ErrorIs(t, checkOrderEnums([]string{""}, errInvalidOrderStatus, orderStatuses), errInvalidOrderStatus, "checkOrderEnums should refuse an empty value")
}

func TestCheckOrderNumbers(t *testing.T) {
	t.Parallel()
	assert.NoError(t, checkOrderNumbers(errInvalidAmount), "checkOrderNumbers should accept no values")
	assert.NoError(t, checkOrderNumbers(errInvalidAmount, 0, 1.5), "checkOrderNumbers should accept zero and positive values")
	assert.ErrorIs(t, checkOrderNumbers(errInvalidPrice, 1, -0.5), errInvalidPrice, "checkOrderNumbers should refuse a negative value with the sentinel given")
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		assert.ErrorIsf(t, checkOrderNumbers(errInvalidAmount, v), errInvalidAmount, "checkOrderNumbers should refuse %v", v)
	}
}

func TestOrderDecimalParam(t *testing.T) {
	t.Parallel()
	assert.Empty(t, orderDecimalParam(0), "orderDecimalParam should leave zero unset")
	assert.Equal(t, "0.00000001", orderDecimalParam(1e-8), "orderDecimalParam should not use exponent notation")
	assert.Equal(t, "70000.44", orderDecimalParam(70000.44), "orderDecimalParam should format a price")
}

func TestOrderTimeParam(t *testing.T) {
	t.Parallel()
	assert.Empty(t, orderTimeParam(time.Time{}), "orderTimeParam should leave the zero time unset")
	assert.Equal(t, "2026-10-19T01:02:03Z", orderTimeParam(time.Date(2026, 10, 19, 12, 2, 3, 0, time.FixedZone("AEDT", 11*60*60))), "orderTimeParam should format in UTC")
}

func TestOrderDurationParam(t *testing.T) {
	t.Parallel()
	assert.Empty(t, orderDurationParam(0), "orderDurationParam should leave zero unset")
	assert.Equal(t, "300s", orderDurationParam(5*time.Minute), "orderDurationParam should format whole seconds")
	assert.Equal(t, "1.5s", orderDurationParam(1500*time.Millisecond), "orderDurationParam should format fractional seconds")
}

func TestCreateOrderResponseErr(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (&CreateOrderResponse{Success: true}).Err(), "Err should not error for a created order")
	for _, tc := range []struct {
		name    string
		failure *CreateOrderFailure
		exp     string
	}{
		{name: "no failure", exp: "order not created: no reason given"},
		{
			name:    "a new order failure reason",
			failure: &CreateOrderFailure{FailureReason: "UNKNOWN_FAILURE_REASON", Message: "The order configuration was invalid", ErrorDetails: "Market orders cannot be placed with empty order sizes", PreviewFailureReason: "UNKNOWN_PREVIEW_FAILURE_REASON", NewOrderFailureReason: "UNSUPPORTED_ORDER_CONFIGURATION"},
			exp:     "order not created: UNSUPPORTED_ORDER_CONFIGURATION: The order configuration was invalid: Market orders cannot be placed with empty order sizes",
		},
		{
			name:    "the deprecated failure reason",
			failure: &CreateOrderFailure{FailureReason: "INSUFFICIENT_FUND", Message: "Insufficient balance in source account", PreviewFailureReason: "PREVIEW_INSUFFICIENT_FUND"},
			exp:     "order not created: INSUFFICIENT_FUND: Insufficient balance in source account",
		},
		{
			name:    "only a preview failure reason",
			failure: &CreateOrderFailure{NewOrderFailureReason: "UNKNOWN_FAILURE_REASON", PreviewFailureReason: "PREVIEW_INVALID_LIMIT_PRICE", Message: "Invalid limit price", ErrorDetails: "Invalid limit price"},
			exp:     "order not created: PREVIEW_INVALID_LIMIT_PRICE: Invalid limit price",
		},
		{
			name:    "only unknown reasons",
			failure: &CreateOrderFailure{FailureReason: "UNKNOWN_FAILURE_REASON", PreviewFailureReason: "UNKNOWN_PREVIEW_FAILURE_REASON", NewOrderFailureReason: "UNKNOWN_FAILURE_REASON"},
			exp:     "order not created: no reason given",
		},
	} {
		err := (&CreateOrderResponse{ErrorResponse: tc.failure}).Err()
		assert.ErrorIsf(t, err, errOrderNotCreated, "Err should report an order Coinbase did not create, with %s", tc.name)
		assert.EqualErrorf(t, err, tc.exp, "Err should give Coinbase's reasons, with %s", tc.name)
	}
}

func TestCancelOrdersResponseErr(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (&CancelOrdersResponse{}).Err(), "Err should not error for no results")
	resp := &CancelOrdersResponse{Results: []CancelOrderResult{
		{Success: true, FailureReason: CancelFailureReasonUnknown, OrderID: "submitted"},
		{FailureReason: CancelFailureReasonUnknownOrder, OrderID: "unknown"},
		{FailureReason: CancelFailureReasonOrderFullyFilled, OrderID: "filled"},
	}}
	assert.NoError(t, (&CancelOrdersResponse{Results: resp.Results[:1]}).Err(), "Err should not error when every request was submitted")
	err := resp.Err()
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "Err should report the order Coinbase does not know")
	assert.ErrorIs(t, err, errCancelOrderFailed, "Err should report the request Coinbase did not submit")
	assert.EqualError(t, err, "order not found: order \"unknown\": UNKNOWN_CANCEL_ORDER\norder cancel request not submitted: order \"filled\": ORDER_IS_FULLY_FILLED", "Err should name each failed order and its reason")
}

func TestCancelOrderResultErr(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (&CancelOrderResult{Success: true, FailureReason: CancelFailureReasonUnknown}).Err(), "Err should not error for a submitted request")
	err := (&CancelOrderResult{FailureReason: CancelFailureReasonUnknownOrder, OrderID: testUnknownOrderID}).Err()
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "Err should report an order Coinbase does not know")
	assert.NotErrorIs(t, err, errCancelOrderFailed, "Err should report an unknown order as missing alone")
	err = (&CancelOrderResult{FailureReason: CancelFailureReasonDuplicateRequest, OrderID: testSandboxOrderID}).Err()
	assert.ErrorIs(t, err, errCancelOrderFailed, "Err should report any other failure")
	assert.NotErrorIs(t, err, order.ErrOrderNotFound, "Err should not report a known order as missing")
}

func TestEditOrderResponseErr(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (&EditOrderResponse{Success: true, Errors: []EditOrderError{}}).Err(), "Err should not error for a placed edit")
	err := (&EditOrderResponse{Errors: []EditOrderError{{EditFailureReason: "ORDER_NOT_FOUND"}}}).Err()
	assert.ErrorIs(t, err, errOrderNotEdited, "Err should report an edit Coinbase did not place")
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "Err should report an order Coinbase does not know")
}

func TestEditOrderPreviewResponseErr(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (&EditOrderPreviewResponse{Errors: []EditOrderError{}}).Err(), "Err should not error for an edit that would succeed")
	err := (&EditOrderPreviewResponse{Errors: []EditOrderError{{EditFailureReason: "INVALID_EDITED_PRICE"}}}).Err()
	assert.ErrorIs(t, err, errOrderEditPreviewFailed, "Err should report an edit that would fail")
	assert.NotErrorIs(t, err, order.ErrOrderNotFound, "Err should not report a known order as missing")
}

func TestEditOrderFailure(t *testing.T) {
	t.Parallel()
	err := editOrderFailure(errOrderNotEdited, []EditOrderError{
		{EditFailureReason: "INVALID_EDITED_SIZE", PreviewFailureReason: "PREVIEW_INVALID_SIZE_PRECISION"},
		{EditFailureReason: "UNKNOWN_EDIT_ORDER_FAILURE_REASON", PreviewFailureReason: "UNKNOWN_PREVIEW_FAILURE_REASON"},
	})
	assert.ErrorIs(t, err, errOrderNotEdited, "editOrderFailure should wrap the sentinel")
	assert.NotErrorIs(t, err, order.ErrOrderNotFound, "editOrderFailure should not report a known order as missing")
	assert.EqualError(t, err, "order not edited: INVALID_EDITED_SIZE, PREVIEW_INVALID_SIZE_PRECISION", "editOrderFailure should give each reason, leaving out the unknown ones")
	for _, reason := range editOrderNotFoundReasons {
		err = editOrderFailure(errOrderEditPreviewFailed, []EditOrderError{{EditFailureReason: reason}})
		assert.ErrorIsf(t, err, order.ErrOrderNotFound, "editOrderFailure should report %s as a missing order", reason)
		assert.ErrorIsf(t, err, errOrderEditPreviewFailed, "editOrderFailure should wrap the sentinel for %s", reason)
	}
	assert.EqualError(t, editOrderFailure(errOrderNotEdited, nil), "order not edited: no reason given", "editOrderFailure should say when Coinbase gave no reason")
}

func TestPreviewOrderResponseErr(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (&PreviewOrderResponse{Errors: []string{}}).Err(), "Err should not error for an order that would succeed")
	err := (&PreviewOrderResponse{Errors: []string{"PREVIEW_INSUFFICIENT_FUND", "PREVIEW_INVALID_LIMIT_PRICE"}}).Err()
	assert.ErrorIs(t, err, errOrderPreviewFailed, "Err should report an order that would fail")
	assert.EqualError(t, err, "order preview reported failures: PREVIEW_INSUFFICIENT_FUND, PREVIEW_INVALID_LIMIT_PRICE", "Err should give each reason")
}
