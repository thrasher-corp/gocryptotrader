package coinbase

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	// derivativesTradingLiveInstrument is the instrument live trading tests place orders in and query, a linear USDC
	// perpetual, whose amounts are in BTC
	derivativesTradingLiveInstrument = "BTC_USDC-PERPETUAL"
	// derivativesTradingLiveAmount is the live instrument's smallest order, in BTC
	derivativesTradingLiveAmount = 0.0001
	// derivativesTradingLiveLabelLength is the longest label the gateway accepts
	derivativesTradingLiveLabelLength = 64

	// The reasons live tests skip methods no live test can safely or possibly exercise
	derivativesTradingSkipCancelAll     = "it would cancel every open order in its scope, not only the test's"
	derivativesTradingSkipClosePosition = "it needs an open position, which a live test would have to open at market"

	// derivativesTradingNullResult is a gateway response holding no result
	derivativesTradingNullResult = `{"jsonrpc":"2.0","result":null}`
	// derivativesTradingOrderResult is a gateway order response holding an empty order
	derivativesTradingOrderResult = `{"jsonrpc":"2.0","result":{"order":{},"trades":[]}}`
)

// derivativesTradingLiveCurrency settles the live instrument
var derivativesTradingLiveCurrency = currency.USDC

// The mock requests send the documented requests of Coinbase's Global Derivatives guides where they show one, and
// testdata/http.json answers each with the documented response of its method

// mockDerivativesBuyRequest is the technical guide's take profit and stop loss order, whose take limit leg gains the
// trigger price the method reference requires of it
func mockDerivativesBuyRequest() *DerivativesOrderRequest {
	return &DerivativesOrderRequest{
		InstrumentName:  "BTC_USDC-PERPETUAL",
		Amount:          0.1,
		Type:            DerivativesOrderTypeLimit,
		Price:           60000,
		LinkedOrderType: DerivativesLinkedOrderTypeOneTriggersOneCancelsOther,
		OTOCOConfig: []DerivativesSecondaryOrder{
			{Type: DerivativesOrderTypeTakeLimit, Direction: DerivativesDirectionSell, Amount: 0.1, Price: 65000, TriggerPrice: 65000, Trigger: DerivativesTriggerLastPrice},
			{Type: DerivativesOrderTypeStopMarket, Direction: DerivativesDirectionSell, Amount: 0.1, TriggerPrice: 55000, Trigger: DerivativesTriggerMarkPrice},
		},
	}
}

// mockDerivativesSellRequest is the isolated positions guide's reduce only sell, which closes an isolated position
func mockDerivativesSellRequest() *DerivativesOrderRequest {
	return &DerivativesOrderRequest{InstrumentName: "SOL_USDC-PERPETUAL", Amount: 10, Type: DerivativesOrderTypeMarket, ReduceOnly: true, Isolated: true}
}

// mockDerivativesCancelOrderByIDRequest is the isolated positions guide's cancel, for the documented response's order
func mockDerivativesCancelOrderByIDRequest() *DerivativesCancelOrderByIDRequest {
	return &DerivativesCancelOrderByIDRequest{OrderID: "ETH-SLIS-12", Isolated: true}
}

func mockDerivativesCancelAllRequest() *DerivativesCancelAllRequest {
	return &DerivativesCancelAllRequest{IncludeIsolated: true}
}

func mockDerivativesCancelAllByCurrencyRequest() *DerivativesCancelAllByCurrencyRequest {
	return &DerivativesCancelAllByCurrencyRequest{Currency: currency.USDC, Kind: DerivativesKindFuture, Type: DerivativesCancelOrderTypeLimit}
}

func mockDerivativesCancelAllByCurrencyPairRequest() *DerivativesCancelAllByCurrencyPairRequest {
	return &DerivativesCancelAllByCurrencyPairRequest{CurrencyPair: "btc_usdc", Kind: DerivativesKindFuture}
}

// mockDerivativesCancelAllByInstrumentRequest is the isolated positions guide's mass cancel
func mockDerivativesCancelAllByInstrumentRequest() *DerivativesCancelAllByInstrumentRequest {
	return &DerivativesCancelAllByInstrumentRequest{InstrumentName: "SOL_USDC-PERPETUAL", IncludeIsolated: true}
}

func mockDerivativesCancelAllByKindOrTypeRequest() *DerivativesCancelAllByKindOrTypeRequest {
	return &DerivativesCancelAllByKindOrTypeRequest{Currencies: []currency.Code{currency.USDC, currency.BTC}, Kind: DerivativesKindFuture}
}

func mockDerivativesCancelByLabelRequest() *DerivativesCancelByLabelRequest {
	return &DerivativesCancelByLabelRequest{Label: "fooBar", Currency: currency.USDC, IncludeIsolated: true}
}

// mockDerivativesClosePositionRequest is the isolated positions guide's close
func mockDerivativesClosePositionRequest() *DerivativesClosePositionRequest {
	return &DerivativesClosePositionRequest{InstrumentName: "SOL_USDC-PERPETUAL", Type: DerivativesOrderTypeMarket, Isolated: true}
}

// mockDerivativesEditOrderRequest edits the documented response's order as the isolated positions guide's edit does
func mockDerivativesEditOrderRequest() *DerivativesEditOrderRequest {
	return &DerivativesEditOrderRequest{OrderID: "94166", Amount: 150, Price: 50111, Isolated: true}
}

func mockDerivativesEditOrderByLabelRequest() *DerivativesEditOrderByLabelRequest {
	return &DerivativesEditOrderByLabelRequest{Label: "i_love_deribit", InstrumentName: "BTC-PERPETUAL", Amount: 150, Price: 50111}
}

// mockDerivativesMarginsRequest previews the margin of the isolated positions guide's isolated order
func mockDerivativesMarginsRequest() *DerivativesMarginsRequest {
	return &DerivativesMarginsRequest{InstrumentName: "SOL_USDC-PERPETUAL", Amount: 10, Price: 100, Isolated: true}
}

func mockDerivativesMaxOrderSizeRequest() *DerivativesMaxOrderSizeRequest {
	return &DerivativesMaxOrderSizeRequest{InstrumentName: "BTC_USDC-PERPETUAL", Leverage: 10, Price: 65000}
}

// mockDerivativesOpenOrdersRequest is the isolated positions guide's open orders request
func mockDerivativesOpenOrdersRequest() *DerivativesOpenOrdersRequest {
	return &DerivativesOpenOrdersRequest{IncludeIsolated: true}
}

func mockDerivativesOpenOrdersByCurrencyRequest() *DerivativesOpenOrdersByCurrencyRequest {
	return &DerivativesOpenOrdersByCurrencyRequest{Currency: currency.BTC, Kind: DerivativesKindOption, Type: DerivativesOpenOrderTypeLimit}
}

func mockDerivativesOpenOrdersByInstrumentRequest() *DerivativesOpenOrdersByInstrumentRequest {
	return &DerivativesOpenOrdersByInstrumentRequest{InstrumentName: "BTC-15FEB19-3250-P", Type: DerivativesOpenOrderTypeAll}
}

func mockDerivativesOpenOrdersByLabelRequest() *DerivativesOpenOrdersByLabelRequest {
	return &DerivativesOpenOrdersByLabelRequest{Currency: currency.BTC, Label: "fooBar"}
}

// mockDerivativesOrderHistoryByCurrencyRequest reads an isolated margin subaccount's history, as the isolated positions
// guide's subaccount ID does
func mockDerivativesOrderHistoryByCurrencyRequest() *DerivativesOrderHistoryByCurrencyRequest {
	return &DerivativesOrderHistoryByCurrencyRequest{Currency: currency.BTC, Kind: DerivativesKindFuture, Count: 1, SubaccountID: 123456}
}

func mockDerivativesOrderHistoryByInstrumentRequest() *DerivativesOrderHistoryByInstrumentRequest {
	return &DerivativesOrderHistoryByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", Count: 1, IncludeOld: true}
}

func mockDerivativesOrderMarginByIDsRequest() *DerivativesOrderMarginByIDsRequest {
	return &DerivativesOrderMarginByIDsRequest{OrderIDs: []string{"ETH-349278", "ETH-349279", "ETH-349280"}}
}

func mockDerivativesOrderStateRequest() *DerivativesOrderStateRequest {
	return &DerivativesOrderStateRequest{OrderID: "ETH-331562"}
}

func mockDerivativesOrderStateByLabelRequest() *DerivativesOrderStateByLabelRequest {
	return &DerivativesOrderStateByLabelRequest{Currency: currency.ETH, Label: "fooBar", IncludeIsolated: true}
}

// mockDerivativesSettlementHistoryByCurrencyRequest is the isolated positions guide's settlement history request
func mockDerivativesSettlementHistoryByCurrencyRequest() *DerivativesSettlementHistoryByCurrencyRequest {
	return &DerivativesSettlementHistoryByCurrencyRequest{Currency: currency.USDC, Count: 100, SubaccountID: 123456}
}

func mockDerivativesSettlementHistoryByInstrumentRequest() *DerivativesSettlementHistoryByInstrumentRequest {
	return &DerivativesSettlementHistoryByInstrumentRequest{InstrumentName: "ETH-22FEB19", Type: "settlement", Count: 1}
}

func mockDerivativesTriggerOrderHistoryRequest() *DerivativesTriggerOrderHistoryRequest {
	return &DerivativesTriggerOrderHistoryRequest{Currency: currency.BTC, InstrumentName: "BTC-PERPETUAL", Count: 10}
}

func mockDerivativesUserTradesByCurrencyRequest() *DerivativesUserTradesByCurrencyRequest {
	return &DerivativesUserTradesByCurrencyRequest{Currency: currency.BTC, Kind: DerivativesKindAny, Count: 2, Sorting: DerivativesSortingDescending}
}

func mockDerivativesUserTradesByCurrencyAndTimeRequest() *DerivativesUserTradesByCurrencyAndTimeRequest {
	return &DerivativesUserTradesByCurrencyAndTimeRequest{Currency: currency.BTC, StartTime: time.UnixMilli(1589900000000), EndTime: time.UnixMilli(1590500000000), Count: 2}
}

func mockDerivativesUserTradesByInstrumentRequest() *DerivativesUserTradesByInstrumentRequest {
	return &DerivativesUserTradesByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", Count: 2, Sorting: DerivativesSortingAscending}
}

func mockDerivativesUserTradesByInstrumentAndTimeRequest() *DerivativesUserTradesByInstrumentAndTimeRequest {
	return &DerivativesUserTradesByInstrumentAndTimeRequest{InstrumentName: "BTC-PERPETUAL", StartTime: time.UnixMilli(1589900000000), EndTime: time.UnixMilli(1590500000000), Count: 2}
}

func mockDerivativesUserTradesByOrderRequest() *DerivativesUserTradesByOrderRequest {
	return &DerivativesUserTradesByOrderRequest{OrderID: "ETH-584830574", Sorting: DerivativesSortingAscending}
}

// mockDerivativesSetLeverageRequest sets the documented response's leverage as the isolated positions guide's request
// does
func mockDerivativesSetLeverageRequest() *DerivativesSetLeverageRequest {
	return &DerivativesSetLeverageRequest{InstrumentName: "SOL_USDC-PERPETUAL", Leverage: new(50.0), Isolated: true}
}

// mockDerivativesRemoveLeverageRequest removes the custom leverage, which the gateway answers with null
func mockDerivativesRemoveLeverageRequest() *DerivativesSetLeverageRequest {
	return &DerivativesSetLeverageRequest{InstrumentName: "SOL_USDC-PERPETUAL"}
}

// The expected values are the documented responses testdata/http.json answers the mock requests with, which the HTTP
// and websocket tests both decode

// testDerivativesPlacedOrderResponse is the documented buy, sell and close position response
func testDerivativesPlacedOrderResponse() *DerivativesOrderTradesResponse {
	return &DerivativesOrderTradesResponse{
		Order: DerivativesOrderResponse{
			OrderID:             "ETH-584864807",
			OrderState:          "filled",
			OrderType:           "limit",
			TimeInForce:         "good_til_cancelled",
			InstrumentName:      "ETH-PERPETUAL",
			CreationTimestamp:   types.Time(time.UnixMilli(1590486335742)),
			LastUpdateTimestamp: types.Time(time.UnixMilli(1590486335742)),
			Direction:           "sell",
			Price:               DerivativesOrderPrice{Value: 198.75},
			ReduceOnly:          true,
			API:                 true,
			Amount:              21,
			FilledAmount:        21,
			AveragePrice:        202.8,
		},
		Trades: []DerivativesUserTradeResponse{
			{
				TradeID:        "ETH-2696097",
				TradeSequence:  1966068,
				InstrumentName: "ETH-PERPETUAL",
				Timestamp:      types.Time(time.UnixMilli(1590486335742)),
				OrderType:      "limit",
				OrderID:        "ETH-584864807",
				Direction:      "sell",
				IndexPrice:     202.86,
				Price:          202.8,
				MarkPrice:      202.79,
				Amount:         21,
				Liquidity:      "T",
				Fee:            0.00007766,
				FeeCurrency:    currency.ETH,
				State:          "filled",
				ReduceOnly:     true,
			},
		},
	}
}

// testDerivativesEditedOrderResponse is the documented edit and edit by label response
func testDerivativesEditedOrderResponse() *DerivativesOrderTradesResponse {
	return &DerivativesOrderTradesResponse{
		Order: DerivativesOrderResponse{
			OrderID:             "94166",
			OrderState:          "open",
			OrderType:           "limit",
			TimeInForce:         "good_til_cancelled",
			InstrumentName:      "BTC-PERPETUAL",
			CreationTimestamp:   types.Time(time.UnixMilli(1616155547764)),
			LastUpdateTimestamp: types.Time(time.UnixMilli(1616155550773)),
			Direction:           "buy",
			Price:               DerivativesOrderPrice{Value: 50111},
			Label:               "i_love_deribit",
			API:                 true,
			Amount:              150,
			Replaced:            true,
		},
		Trades: []DerivativesUserTradeResponse{},
	}
}

// testDerivativesCancelledOrder is the documented cancel response, an untriggered stop market order
func testDerivativesCancelledOrder() *DerivativesOrderResponse {
	return &DerivativesOrderResponse{
		OrderID:             "ETH-SLIS-12",
		OrderState:          "untriggered",
		OrderType:           "stop_market",
		TimeInForce:         "good_til_cancelled",
		InstrumentName:      "ETH-PERPETUAL",
		CreationTimestamp:   types.Time(time.UnixMilli(1550575961291)),
		LastUpdateTimestamp: types.Time(time.UnixMilli(1550575961291)),
		Direction:           "sell",
		Price:               DerivativesOrderPrice{Market: true},
		Amount:              5,
		Trigger:             "index_price",
		TriggerPrice:        144.73,
	}
}

// testDerivativesOpenOrders is the documented open orders response
func testDerivativesOpenOrders() []DerivativesOrderResponse {
	return []DerivativesOrderResponse{
		{
			OrderID:             "146062",
			OrderState:          "open",
			OrderType:           "limit",
			TimeInForce:         "good_til_cancelled",
			InstrumentName:      "BTC-15FEB19-3250-P",
			CreationTimestamp:   types.Time(time.UnixMilli(1550050597036)),
			LastUpdateTimestamp: types.Time(time.UnixMilli(1550050597036)),
			Direction:           "buy",
			Price:               DerivativesOrderPrice{Value: 0.0028},
			Label:               "fooBar",
			API:                 true,
			Amount:              10,
		},
	}
}

// testDerivativesOrderHistoryResponse is the documented order history response
func testDerivativesOrderHistoryResponse() *DerivativesOrderHistoryResponse {
	return &DerivativesOrderHistoryResponse{
		Orders: []DerivativesOrderResponse{
			{
				OrderID:             "146475",
				OrderState:          "filled",
				OrderType:           "limit",
				TimeInForce:         "good_til_cancelled",
				InstrumentName:      "BTC-PERPETUAL",
				CreationTimestamp:   types.Time(time.UnixMilli(1550661808761)),
				LastUpdateTimestamp: types.Time(time.UnixMilli(1550661808761)),
				Direction:           "buy",
				Price:               DerivativesOrderPrice{Value: 3886.5},
				API:                 true,
				Amount:              40,
				FilledAmount:        40,
				AveragePrice:        3659.8,
			},
		},
	}
}

// testDerivativesOrderMargins is the documented order margins response
func testDerivativesOrderMargins() []DerivativesOrderMarginResponse {
	return []DerivativesOrderMarginResponse{
		{OrderID: "ETH-349278", InitialMargin: 0.00091156, InitialMarginCurrency: currency.ETH},
		{OrderID: "ETH-349279", InitialMarginCurrency: currency.ETH},
		{OrderID: "ETH-349280", InitialMarginCurrency: currency.ETH},
	}
}

// testDerivativesOrderState is the documented order state response, with the label the by label response carries
func testDerivativesOrderState(label string) *DerivativesOrderResponse {
	return &DerivativesOrderResponse{
		OrderID:             "ETH-331562",
		OrderState:          "filled",
		OrderType:           "limit",
		TimeInForce:         "good_til_cancelled",
		InstrumentName:      "ETH-PERPETUAL",
		CreationTimestamp:   types.Time(time.UnixMilli(1550219749176)),
		LastUpdateTimestamp: types.Time(time.UnixMilli(1550219810944)),
		Direction:           "sell",
		Price:               DerivativesOrderPrice{Value: 118.94},
		Label:               label,
		Amount:              37,
		FilledAmount:        37,
		AveragePrice:        118.94,
	}
}

// testDerivativesSettlementHistoryResponse is the documented settlement history response
func testDerivativesSettlementHistoryResponse() *DerivativesSettlementsResponse {
	return &DerivativesSettlementsResponse{
		Settlements: []DerivativesSettlement{
			{
				Type:              "settlement",
				Timestamp:         types.Time(time.UnixMilli(1550475692526)),
				InstrumentName:    "ETH-22FEB19",
				IndexPrice:        119.8,
				MarkPrice:         121.67,
				Position:          -66,
				ProfitLoss:        -0.001783937,
				SessionProfitLoss: 0.038358299,
			},
		},
		Continuation: "xY7T6cusbMBNpH9SNmKb94jXSBxUPojJEdCPL4YociHBUgAhWQvEP",
	}
}

// testDerivativesTriggerOrderHistoryResponse is the documented trigger order history response; its example sends the
// trailing offset as offset, which the schema names trigger_offset, and the trigger as index
func testDerivativesTriggerOrderHistoryResponse() *DerivativesTriggerOrderHistoryResponse {
	return &DerivativesTriggerOrderHistoryResponse{
		Entries: []DerivativesTriggerOrderHistoryEntry{
			{
				Timestamp:      types.Time(time.UnixMilli(1555918941451)),
				Trigger:        "index",
				TriggerPrice:   5285,
				TriggerOrderID: "SLIS-103",
				OrderID:        "671473",
				OrderState:     "new",
				InstrumentName: "BTC-PERPETUAL",
				Request:        "trigger:order",
				Direction:      "buy",
				Price:          DerivativesOrderPrice{Value: 5179.28},
				Amount:         10,
			},
		},
		Continuation: "1555918941451.SLIS-103",
	}
}

// testDerivativesUserTradesResponse is the documented user trades response
func testDerivativesUserTradesResponse() *DerivativesUserTradesResponse {
	return &DerivativesUserTradesResponse{
		Trades: []DerivativesUserTradeResponse{
			{
				TradeID:           "48078936",
				TradeSequence:     1,
				InstrumentName:    "BTC-27MAY20-8750-C",
				Timestamp:         types.Time(time.UnixMilli(1590480620145)),
				OrderType:         "limit",
				OrderID:           "4008699030",
				Direction:         "sell",
				TickDirection:     1,
				IndexPrice:        8993.47,
				Price:             0.028,
				MarkPrice:         0.03135383,
				Amount:            1,
				ImpliedVolatility: 38.51,
				UnderlyingPrice:   8994.95,
				Liquidity:         "M",
				Fee:               0.0004,
				FeeCurrency:       currency.BTC,
				State:             "filled",
			},
			{
				TradeID:        "47958936",
				TradeSequence:  299513,
				InstrumentName: "BTC-26JUN20",
				Timestamp:      types.Time(time.UnixMilli(1589923311862)),
				OrderType:      "limit",
				OrderID:        "3993343822",
				Direction:      "buy",
				TickDirection:  2,
				IndexPrice:     9679.48,
				Price:          9681.5,
				MarkPrice:      9684,
				Amount:         10,
				Liquidity:      "M",
				Fee:            -0.00000021,
				FeeCurrency:    currency.BTC,
				State:          "filled",
			},
		},
	}
}

// testDerivativesUserTradesByOrder is the documented order trades response
func testDerivativesUserTradesByOrder() []DerivativesUserTradeResponse {
	return []DerivativesUserTradeResponse{
		{
			TradeID:        "ETH-2696068",
			TradeSequence:  1966042,
			InstrumentName: "ETH-PERPETUAL",
			Timestamp:      types.Time(time.UnixMilli(1590480712800)),
			OrderType:      "market",
			OrderID:        "ETH-584830574",
			Direction:      "buy",
			TickDirection:  3,
			IndexPrice:     203.89,
			Price:          203.8,
			MarkPrice:      203.78,
			Amount:         100,
			Liquidity:      "T",
			Fee:            0.00036801,
			FeeCurrency:    currency.ETH,
			State:          "filled",
		},
	}
}

// testDerivativesMarginsResponse is the documented margins response
func testDerivativesMarginsResponse() *DerivativesMarginsResponse {
	return &DerivativesMarginsResponse{
		Buy:          0.0219949,
		BuyMakerFee:  0.000017,
		BuyTakerFee:  0.000085,
		SellMakerFee: 0.000017,
		SellTakerFee: 0.000085,
		MinimumPrice: 3684.8,
		MaximumPrice: 3759.24,
	}
}

// testDerivativesMaxOrderSizeResponse is the documented maximum order size response
func testDerivativesMaxOrderSizeResponse() *DerivativesMaxOrderSizeResponse {
	return &DerivativesMaxOrderSizeResponse{
		BuyPrice:                        65000,
		SellPrice:                       65000,
		MaximumTradableAmount:           DerivativesTradableNotional{BuyNotional: 125000, SellNotional: 150000},
		MaximumTradableAmountReduceOnly: DerivativesTradableNotional{BuyNotional: 50000},
	}
}

// newDerivativesTradingParamsServer returns an exchange whose gateway requests all receive one fixed response body,
// and a function returning the last request, so tests can check what a method sends where its fixture cannot
func newDerivativesTradingParamsServer(t *testing.T, body string) (ex *Exchange, lastRequest func() receivedJSONRPC) {
	t.Helper()
	var mu sync.Mutex
	var last receivedJSONRPC
	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if req, ok := readJSONRPC(t, r); ok {
			assert.Equal(t, derivativesPath+req.Method, r.URL.Path, "The request should be posted to its method's path")
			assert.Equal(t, "Bearer "+testDerivativesAccessToken, r.Header.Get("Authorization"), "A private method should send the access token")
			mu.Lock()
			last = req
			mu.Unlock()
		}
		writeJSONRPC(t, w, body)
	}))
	return ex, func() receivedJSONRPC {
		mu.Lock()
		defer mu.Unlock()
		return last
	}
}

// derivativesTradingLiveTicker is the live instrument's mark price and the prices the gateway clamps sell and buy
// orders to
type derivativesTradingLiveTicker struct {
	MarkPrice    float64 `json:"mark_price"`
	MinimumPrice float64 `json:"min_price"`
	MaximumPrice float64 `json:"max_price"`
}

// getDerivativesTradingLiveTicker returns the live instrument's prices, which live tests price their orders from
func getDerivativesTradingLiveTicker(t *testing.T) *derivativesTradingLiveTicker {
	t.Helper()
	var ticker *derivativesTradingLiveTicker
	err := e.GlobalDerivatives().SendHTTPRequest(t.Context(), derivativesNonMatchingEPL, "public/ticker", &DerivativesInstrumentNameWireRequest{InstrumentName: derivativesTradingLiveInstrument}, &ticker)
	require.NoError(t, err, "public/ticker must not error")
	require.NotNil(t, ticker, "public/ticker must return the ticker")
	return ticker
}

// newDerivativesTradingLiveOrderRequest returns a post only limit order of the live instrument's smallest amount, a buy
// at half the lowest sell price the gateway accepts or a sell at double the highest buy price, so no market move fills
// it while the test runs; orders with its label are cancelled when the test ends
func newDerivativesTradingLiveOrderRequest(t *testing.T, direction string) *DerivativesOrderRequest {
	t.Helper()
	ticker := getDerivativesTradingLiveTicker(t)
	price := math.Floor(ticker.MinimumPrice/20) * 10
	if direction == DerivativesDirectionSell {
		price = math.Ceil(ticker.MaximumPrice/5) * 10
	}
	label := "gct-" + strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + t.Name()
	label = label[:min(len(label), derivativesTradingLiveLabelLength)]
	t.Cleanup(func() {
		_, err := e.GlobalDerivatives().CancelByLabel(context.WithoutCancel(t.Context()), &DerivativesCancelByLabelRequest{Label: label, Currency: derivativesTradingLiveCurrency})
		assert.NoError(t, err, "CancelByLabel should cancel the test's orders")
	})
	return &DerivativesOrderRequest{
		InstrumentName: derivativesTradingLiveInstrument,
		Amount:         derivativesTradingLiveAmount,
		Type:           DerivativesOrderTypeLimit,
		Label:          label,
		Price:          price,
		PostOnly:       true,
		RejectPostOnly: true,
	}
}

// placeDerivativesTradingLiveOrder places a live buy order no market move fills while the test runs, and returns the
// request and the order; it is cancelled when the test ends
func placeDerivativesTradingLiveOrder(t *testing.T) (*DerivativesOrderRequest, *DerivativesOrderResponse) {
	t.Helper()
	req := newDerivativesTradingLiveOrderRequest(t, DerivativesDirectionBuy)
	resp, err := e.GlobalDerivatives().Buy(t.Context(), req)
	require.NoError(t, err, "Buy must not error")
	require.Equal(t, DerivativesOrderStateOpen, resp.Order.OrderState, "Buy must leave the order open")
	return req, &resp.Order
}

// derivativesTradingLiveLeverage is the live instrument's custom leverage, nil when it uses the default
type derivativesTradingLiveLeverage struct {
	Leverage *float64 `json:"leverage"`
}

// derivativesTradingLiveLeverageMu serialises the live tests that set the live instrument's leverage, so each restores
// the leverage it found
var derivativesTradingLiveLeverageMu sync.Mutex

// setDerivativesTradingLiveLeverage sets the live instrument's leverage to 2 with set, and restores the leverage it
// found when the test ends
func setDerivativesTradingLiveLeverage(t *testing.T, set func(context.Context, *DerivativesSetLeverageRequest) (float64, error)) float64 {
	t.Helper()
	derivativesTradingLiveLeverageMu.Lock()
	var current *derivativesTradingLiveLeverage
	err := e.GlobalDerivatives().SendAuthenticatedHTTPRequest(t.Context(), derivativesNonMatchingEPL, "private/get_leverage", &DerivativesInstrumentNameWireRequest{InstrumentName: derivativesTradingLiveInstrument}, &current)
	if err != nil || current == nil {
		derivativesTradingLiveLeverageMu.Unlock()
		require.NoError(t, err, "private/get_leverage must not error")
		require.NotNil(t, current, "private/get_leverage must return the leverage")
	}
	t.Cleanup(func() {
		defer derivativesTradingLiveLeverageMu.Unlock()
		_, err := e.GlobalDerivatives().SetLeverage(context.WithoutCancel(t.Context()), &DerivativesSetLeverageRequest{InstrumentName: derivativesTradingLiveInstrument, Leverage: current.Leverage})
		assert.NoError(t, err, "SetLeverage should restore the instrument's leverage")
	})
	leverage, err := set(t.Context(), &DerivativesSetLeverageRequest{InstrumentName: derivativesTradingLiveInstrument, Leverage: new(2.0)})
	require.NoError(t, err, "SetLeverage must not error")
	return leverage
}

func TestGlobalDerivativesBuy(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().Buy(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Buy must reject a nil request")
	_, err = e.GlobalDerivatives().Buy(t.Context(), &DerivativesOrderRequest{Amount: 1})
	require.ErrorIs(t, err, errInstrumentNameRequired, "Buy must require an instrument name")
	_, err = e.GlobalDerivatives().Buy(t.Context(), &DerivativesOrderRequest{InstrumentName: "SOL_USDC-PERPETUAL", Amount: 1, Type: DerivativesOrderTypeStopMarket, TriggerPrice: 90, Trigger: DerivativesTriggerMarkPrice, Isolated: true})
	require.ErrorIs(t, err, errDerivativesIsolatedOrderUnsupported, "Buy must refuse an isolated trigger order")

	ex, lastRequest := newDerivativesTradingParamsServer(t, derivativesTradingOrderResult)
	_, err = ex.GlobalDerivatives().Buy(t.Context(), &DerivativesOrderRequest{
		InstrumentName:  "SOL_USDC-PERPETUAL",
		Contracts:       10,
		Type:            DerivativesOrderTypeLimit,
		Label:           "gct",
		Price:           100,
		TimeInForce:     DerivativesTimeInForceGoodTilCancelled,
		DisplayAmount:   1,
		PostOnly:        true,
		RejectPostOnly:  true,
		ValidUntil:      time.UnixMilli(1792368000000),
		Isolated:        true,
		AllocatedMargin: 25,
	})
	require.NoError(t, err, "Buy must not error for an isolated margin order")
	sent := lastRequest()
	assert.Equal(t, "private/buy", sent.Method, "Buy should call private/buy")
	exp := `{"instrument_name":"SOL_USDC-PERPETUAL","contracts":10,"type":"limit","label":"gct","price":100,"time_in_force":"good_til_cancelled","display_amount":1,"post_only":true,"reject_post_only":true,"valid_until":1792368000000,"isolated":true,"allocated_margin":25}`
	assert.JSONEq(t, exp, string(sent.Params), "Buy should send an isolated order's margin and the valid until time in Unix milliseconds")
	_, err = ex.GlobalDerivatives().Buy(t.Context(), &DerivativesOrderRequest{InstrumentName: "BTC-27NOV26-90000-C", Amount: 1, Price: 50, ReduceOnly: true, Advanced: DerivativesAdvancedImpliedVolatility, MMP: true})
	require.NoError(t, err, "Buy must not error for an advanced option order")
	exp = `{"instrument_name":"BTC-27NOV26-90000-C","amount":1,"price":50,"post_only":false,"reduce_only":true,"advanced":"implv","mmp":true}`
	assert.JSONEq(t, exp, string(lastRequest().Params), "Buy should always send post_only, which the gateway defaults to true")

	skipLiveUnlessCanManipulate(t)
	req := mockDerivativesBuyRequest()
	if !mockTests {
		req = newDerivativesTradingLiveOrderRequest(t, DerivativesDirectionBuy)
	}
	resp, err := e.GlobalDerivatives().Buy(t.Context(), req)
	require.NoError(t, err, "Buy must not error")
	if mockTests {
		assert.Equal(t, testDerivativesPlacedOrderResponse(), resp, "Buy should decode every field")
	} else {
		assert.Equal(t, DerivativesOrderStateOpen, resp.Order.OrderState, "Buy should place a resting order")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().Buy(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "Buy should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().Buy(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "Buy should return the request failure")
}

func TestGlobalDerivativesCancelOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().CancelOrderByID(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelOrderByID must reject a nil request")
	_, err = e.GlobalDerivatives().CancelOrderByID(t.Context(), &DerivativesCancelOrderByIDRequest{Isolated: true})
	require.ErrorIs(t, err, errDerivativesOrderIDRequired, "CancelOrderByID must require an order ID")

	skipLiveUnlessCanManipulate(t)
	req := mockDerivativesCancelOrderByIDRequest()
	if !mockTests {
		_, order := placeDerivativesTradingLiveOrder(t)
		req = &DerivativesCancelOrderByIDRequest{OrderID: order.OrderID}
	}
	resp, err := e.GlobalDerivatives().CancelOrderByID(t.Context(), req)
	require.NoError(t, err, "CancelOrderByID must not error")
	if mockTests {
		assert.Equal(t, testDerivativesCancelledOrder(), resp, "CancelOrderByID should decode every field")
	} else {
		assert.Equal(t, DerivativesOrderStateCancelled, resp.OrderState, "CancelOrderByID should cancel the order")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().CancelOrderByID(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelOrderByID should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().CancelOrderByID(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelOrderByID should return the request failure")
}

func TestGlobalDerivativesCancelAll(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().CancelAll(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAll must reject a nil request")

	if !mockTests {
		t.Skip("CancelAll is not tested live, as " + derivativesTradingSkipCancelAll)
	}
	resp, err := e.GlobalDerivatives().CancelAll(t.Context(), mockDerivativesCancelAllRequest())
	require.NoError(t, err, "CancelAll must not error")
	assert.Equal(t, &DerivativesCancelAllResponse{CancelledCount: 4}, resp, "CancelAll should decode the count")

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().CancelAll(t.Context(), mockDerivativesCancelAllRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelAll should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().CancelAll(t.Context(), mockDerivativesCancelAllRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelAll should return the request failure")
}

func TestGlobalDerivativesCancelAllByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().CancelAllByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllByCurrency must reject a nil request")
	_, err = e.GlobalDerivatives().CancelAllByCurrency(t.Context(), &DerivativesCancelAllByCurrencyRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelAllByCurrency must require a currency")
	_, err = e.GlobalDerivatives().CancelAllByCurrency(t.Context(), &DerivativesCancelAllByCurrencyRequest{Currency: currency.USDC, Kind: "perpetual"})
	require.ErrorIs(t, err, errDerivativesInvalidTradingKind, "CancelAllByCurrency must refuse an undocumented kind")
	_, err = e.GlobalDerivatives().CancelAllByCurrency(t.Context(), &DerivativesCancelAllByCurrencyRequest{Currency: currency.USDC, Type: DerivativesOrderTypeMarket})
	require.ErrorIs(t, err, errDerivativesInvalidCancelOrderType, "CancelAllByCurrency must refuse an undocumented order type")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":0}`)
	_, err = ex.GlobalDerivatives().CancelAllByCurrency(t.Context(), &DerivativesCancelAllByCurrencyRequest{Currency: currency.NewCode("usdc"), Kind: DerivativesKindAny, Detailed: true, FreezeQuotes: true, IncludeIsolated: true})
	require.NoError(t, err, "CancelAllByCurrency must not error")
	assert.JSONEq(t, `{"currency":"USDC","kind":"any","detailed":true,"freeze_quotes":true,"include_isolated":true}`, string(lastRequest().Params), "CancelAllByCurrency should send the currency in upper case")

	if !mockTests {
		t.Skip("CancelAllByCurrency is not tested live, as " + derivativesTradingSkipCancelAll)
	}
	resp, err := e.GlobalDerivatives().CancelAllByCurrency(t.Context(), mockDerivativesCancelAllByCurrencyRequest())
	require.NoError(t, err, "CancelAllByCurrency must not error")
	assert.Equal(t, &DerivativesCancelAllResponse{CancelledCount: 4}, resp, "CancelAllByCurrency should decode the count")

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().CancelAllByCurrency(t.Context(), mockDerivativesCancelAllByCurrencyRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelAllByCurrency should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().CancelAllByCurrency(t.Context(), mockDerivativesCancelAllByCurrencyRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelAllByCurrency should return the request failure")
}

func TestGlobalDerivativesCancelAllByCurrencyPair(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().CancelAllByCurrencyPair(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllByCurrencyPair must reject a nil request")
	_, err = e.GlobalDerivatives().CancelAllByCurrencyPair(t.Context(), &DerivativesCancelAllByCurrencyPairRequest{Kind: DerivativesKindFuture})
	require.ErrorIs(t, err, errDerivativesCurrencyPairRequired, "CancelAllByCurrencyPair must require a currency pair")
	_, err = e.GlobalDerivatives().CancelAllByCurrencyPair(t.Context(), &DerivativesCancelAllByCurrencyPairRequest{CurrencyPair: "btc_usdc", Kind: "perpetual"})
	require.ErrorIs(t, err, errDerivativesInvalidTradingKind, "CancelAllByCurrencyPair must refuse an undocumented kind")
	_, err = e.GlobalDerivatives().CancelAllByCurrencyPair(t.Context(), &DerivativesCancelAllByCurrencyPairRequest{CurrencyPair: "btc_usdc", Type: DerivativesOrderTypeMarket})
	require.ErrorIs(t, err, errDerivativesInvalidCancelOrderType, "CancelAllByCurrencyPair must refuse an undocumented order type")

	if !mockTests {
		t.Skip("CancelAllByCurrencyPair is not tested live, as " + derivativesTradingSkipCancelAll)
	}
	resp, err := e.GlobalDerivatives().CancelAllByCurrencyPair(t.Context(), mockDerivativesCancelAllByCurrencyPairRequest())
	require.NoError(t, err, "CancelAllByCurrencyPair must not error")
	assert.Equal(t, &DerivativesCancelAllResponse{CancelledCount: 4}, resp, "CancelAllByCurrencyPair should decode the count")

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().CancelAllByCurrencyPair(t.Context(), mockDerivativesCancelAllByCurrencyPairRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelAllByCurrencyPair should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().CancelAllByCurrencyPair(t.Context(), mockDerivativesCancelAllByCurrencyPairRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelAllByCurrencyPair should return the request failure")
}

func TestGlobalDerivativesCancelAllByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().CancelAllByInstrument(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllByInstrument must reject a nil request")
	_, err = e.GlobalDerivatives().CancelAllByInstrument(t.Context(), &DerivativesCancelAllByInstrumentRequest{IncludeIsolated: true})
	require.ErrorIs(t, err, errInstrumentNameRequired, "CancelAllByInstrument must require an instrument name")
	_, err = e.GlobalDerivatives().CancelAllByInstrument(t.Context(), &DerivativesCancelAllByInstrumentRequest{InstrumentName: "SOL_USDC-PERPETUAL", Type: DerivativesOpenOrderTypeStopAll})
	require.ErrorIs(t, err, errDerivativesInvalidCancelOrderType, "CancelAllByInstrument must refuse an order type only open orders methods take")

	if !mockTests {
		t.Skip("CancelAllByInstrument is not tested live, as " + derivativesTradingSkipCancelAll)
	}
	resp, err := e.GlobalDerivatives().CancelAllByInstrument(t.Context(), mockDerivativesCancelAllByInstrumentRequest())
	require.NoError(t, err, "CancelAllByInstrument must not error")
	assert.Equal(t, &DerivativesCancelAllResponse{CancelledCount: 4}, resp, "CancelAllByInstrument should decode the count")

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().CancelAllByInstrument(t.Context(), mockDerivativesCancelAllByInstrumentRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelAllByInstrument should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().CancelAllByInstrument(t.Context(), mockDerivativesCancelAllByInstrumentRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelAllByInstrument should return the request failure")
}

func TestGlobalDerivativesCancelAllByKindOrType(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().CancelAllByKindOrType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllByKindOrType must reject a nil request")
	_, err = e.GlobalDerivatives().CancelAllByKindOrType(t.Context(), &DerivativesCancelAllByKindOrTypeRequest{Kind: DerivativesKindFuture})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelAllByKindOrType must require a currency")
	_, err = e.GlobalDerivatives().CancelAllByKindOrType(t.Context(), &DerivativesCancelAllByKindOrTypeRequest{Currencies: []currency.Code{currency.USDC}, Kind: "perpetual"})
	require.ErrorIs(t, err, errDerivativesInvalidTradingKind, "CancelAllByKindOrType must refuse an undocumented kind")
	_, err = e.GlobalDerivatives().CancelAllByKindOrType(t.Context(), &DerivativesCancelAllByKindOrTypeRequest{Currencies: []currency.Code{currency.USDC}, Type: DerivativesOrderTypeMarket})
	require.ErrorIs(t, err, errDerivativesInvalidCancelOrderType, "CancelAllByKindOrType must refuse an undocumented order type")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":0}`)
	for _, tc := range []struct {
		currencies []currency.Code
		exp        string
	}{
		{currencies: []currency.Code{currency.NewCode(DerivativesCurrencyAny)}, exp: `{"currency":"any","type":"take"}`},
		{currencies: []currency.Code{currency.NewCode("usdc")}, exp: `{"currency":"USDC","type":"take"}`},
		{currencies: []currency.Code{currency.USDC, currency.NewCode("btc")}, exp: `{"currency":["USDC","BTC"],"type":"take"}`},
	} {
		_, err = ex.GlobalDerivatives().CancelAllByKindOrType(t.Context(), &DerivativesCancelAllByKindOrTypeRequest{Currencies: tc.currencies, Type: DerivativesCancelOrderTypeTake})
		require.NoErrorf(t, err, "CancelAllByKindOrType must not error for currencies %s", tc.currencies)
		assert.JSONEqf(t, tc.exp, string(lastRequest().Params), "CancelAllByKindOrType should send currencies %s as a single currency or a list", tc.currencies)
	}

	if !mockTests {
		t.Skip("CancelAllByKindOrType is not tested live, as " + derivativesTradingSkipCancelAll)
	}
	resp, err := e.GlobalDerivatives().CancelAllByKindOrType(t.Context(), mockDerivativesCancelAllByKindOrTypeRequest())
	require.NoError(t, err, "CancelAllByKindOrType must not error")
	assert.Equal(t, &DerivativesCancelAllResponse{CancelledCount: 4}, resp, "CancelAllByKindOrType should decode the count")

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().CancelAllByKindOrType(t.Context(), mockDerivativesCancelAllByKindOrTypeRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelAllByKindOrType should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().CancelAllByKindOrType(t.Context(), mockDerivativesCancelAllByKindOrTypeRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelAllByKindOrType should return the request failure")
}

func TestGlobalDerivativesCancelByLabel(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().CancelByLabel(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelByLabel must reject a nil request")
	_, err = e.GlobalDerivatives().CancelByLabel(t.Context(), &DerivativesCancelByLabelRequest{Currency: currency.USDC})
	require.ErrorIs(t, err, errDerivativesLabelRequired, "CancelByLabel must require a label")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":0}`)
	_, err = ex.GlobalDerivatives().CancelByLabel(t.Context(), &DerivativesCancelByLabelRequest{Label: "gct"})
	require.NoError(t, err, "CancelByLabel must not error without a currency")
	assert.JSONEq(t, `{"label":"gct"}`, string(lastRequest().Params), "CancelByLabel should leave an unset currency out")
	_, err = ex.GlobalDerivatives().CancelByLabel(t.Context(), &DerivativesCancelByLabelRequest{Label: "gct", Currency: currency.NewCode("usdc")})
	require.NoError(t, err, "CancelByLabel must not error with a currency")
	assert.JSONEq(t, `{"label":"gct","currency":"USDC"}`, string(lastRequest().Params), "CancelByLabel should send the currency in upper case")

	skipLiveUnlessCanManipulate(t)
	req := mockDerivativesCancelByLabelRequest()
	if !mockTests {
		placed, _ := placeDerivativesTradingLiveOrder(t)
		req = &DerivativesCancelByLabelRequest{Label: placed.Label, Currency: derivativesTradingLiveCurrency}
	}
	resp, err := e.GlobalDerivatives().CancelByLabel(t.Context(), req)
	require.NoError(t, err, "CancelByLabel must not error")
	if mockTests {
		assert.Equal(t, &DerivativesCancelAllResponse{CancelledCount: 4}, resp, "CancelByLabel should decode the count")
	} else {
		assert.Equal(t, &DerivativesCancelAllResponse{CancelledCount: 1}, resp, "CancelByLabel should cancel the order with the label")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().CancelByLabel(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelByLabel should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().CancelByLabel(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelByLabel should return the request failure")
}

func TestGlobalDerivativesClosePosition(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().ClosePosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ClosePosition must reject a nil request")
	_, err = e.GlobalDerivatives().ClosePosition(t.Context(), &DerivativesClosePositionRequest{Type: DerivativesOrderTypeMarket})
	require.ErrorIs(t, err, errInstrumentNameRequired, "ClosePosition must require an instrument name")
	_, err = e.GlobalDerivatives().ClosePosition(t.Context(), &DerivativesClosePositionRequest{InstrumentName: "SOL_USDC-PERPETUAL"})
	require.ErrorIs(t, err, errDerivativesInvalidOrderType, "ClosePosition must require an order type")
	_, err = e.GlobalDerivatives().ClosePosition(t.Context(), &DerivativesClosePositionRequest{InstrumentName: "SOL_USDC-PERPETUAL", Type: DerivativesOrderTypeStopMarket})
	require.ErrorIs(t, err, errDerivativesInvalidOrderType, "ClosePosition must refuse an order type other than limit or market")

	if !mockTests {
		t.Skip("ClosePosition is not tested live, as " + derivativesTradingSkipClosePosition)
	}
	resp, err := e.GlobalDerivatives().ClosePosition(t.Context(), mockDerivativesClosePositionRequest())
	require.NoError(t, err, "ClosePosition must not error")
	assert.Equal(t, testDerivativesPlacedOrderResponse(), resp, "ClosePosition should decode every field")

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().ClosePosition(t.Context(), mockDerivativesClosePositionRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "ClosePosition should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().ClosePosition(t.Context(), mockDerivativesClosePositionRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "ClosePosition should return the request failure")
}

func TestGlobalDerivativesEditOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().EditOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "EditOrder must reject a nil request")
	_, err = e.GlobalDerivatives().EditOrder(t.Context(), &DerivativesEditOrderRequest{Amount: 1})
	require.ErrorIs(t, err, errDerivativesOrderIDRequired, "EditOrder must require an order ID")
	_, err = e.GlobalDerivatives().EditOrder(t.Context(), &DerivativesEditOrderRequest{OrderID: "94166"})
	require.ErrorIs(t, err, errInvalidAmount, "EditOrder must require a size")
	_, err = e.GlobalDerivatives().EditOrder(t.Context(), &DerivativesEditOrderRequest{OrderID: "94166", Amount: 1, AllocatedMargin: 5})
	require.ErrorIs(t, err, errDerivativesAllocatedMarginWithoutIsolated, "EditOrder must refuse to allocate margin to an order that is not isolated")

	ex, lastRequest := newDerivativesTradingParamsServer(t, derivativesTradingOrderResult)
	_, err = ex.GlobalDerivatives().EditOrder(t.Context(), &DerivativesEditOrderRequest{
		OrderID:         "94166",
		Contracts:       15,
		Price:           50000,
		TriggerOffset:   10,
		ValidUntil:      time.UnixMilli(1792368000000),
		DisplayAmount:   5,
		Isolated:        true,
		AllocatedMargin: 12.5,
	})
	require.NoError(t, err, "EditOrder must not error")
	sent := lastRequest()
	assert.Equal(t, "private/edit", sent.Method, "EditOrder should call private/edit")
	exp := `{"order_id":"94166","contracts":15,"price":50000,"post_only":false,"trigger_offset":10,"valid_until":1792368000000,"display_amount":5,"isolated":true,"allocated_margin":12.5}`
	assert.JSONEq(t, exp, string(sent.Params), "EditOrder should always send post_only and leave an unset MMP flag out, which the gateway refuses changing")

	skipLiveUnlessCanManipulate(t)
	req := mockDerivativesEditOrderRequest()
	if !mockTests {
		placed, order := placeDerivativesTradingLiveOrder(t)
		req = &DerivativesEditOrderRequest{OrderID: order.OrderID, Amount: placed.Amount, Price: placed.Price - 10, PostOnly: true, RejectPostOnly: true}
	}
	resp, err := e.GlobalDerivatives().EditOrder(t.Context(), req)
	require.NoError(t, err, "EditOrder must not error")
	if mockTests {
		assert.Equal(t, testDerivativesEditedOrderResponse(), resp, "EditOrder should decode every field")
	} else {
		assert.Equal(t, req.Price, resp.Order.Price.Value, "EditOrder should reprice the order")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().EditOrder(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "EditOrder should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().EditOrder(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "EditOrder should return the request failure")
}

func TestGlobalDerivativesEditOrderByLabel(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().EditOrderByLabel(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "EditOrderByLabel must reject a nil request")
	_, err = e.GlobalDerivatives().EditOrderByLabel(t.Context(), &DerivativesEditOrderByLabelRequest{Label: "gct", Amount: 1})
	require.ErrorIs(t, err, errInstrumentNameRequired, "EditOrderByLabel must require an instrument name")
	_, err = e.GlobalDerivatives().EditOrderByLabel(t.Context(), &DerivativesEditOrderByLabelRequest{InstrumentName: "BTC-PERPETUAL", Amount: -1})
	require.ErrorIs(t, err, errInvalidAmount, "EditOrderByLabel must refuse a negative amount")
	_, err = e.GlobalDerivatives().EditOrderByLabel(t.Context(), &DerivativesEditOrderByLabelRequest{InstrumentName: "BTC-PERPETUAL", Amount: 1, RejectPostOnly: true})
	require.ErrorIs(t, err, errDerivativesRejectPostOnlyWithoutPostOnly, "EditOrderByLabel must refuse reject post only without post only")

	ex, lastRequest := newDerivativesTradingParamsServer(t, derivativesTradingOrderResult)
	_, err = ex.GlobalDerivatives().EditOrderByLabel(t.Context(), &DerivativesEditOrderByLabelRequest{InstrumentName: "BTC-PERPETUAL", Label: "gct", Amount: 20, PostOnly: true, ValidUntil: time.UnixMilli(1792368000000), Isolated: true, AllocatedMargin: 3})
	require.NoError(t, err, "EditOrderByLabel must not error")
	sent := lastRequest()
	assert.Equal(t, "private/edit_by_label", sent.Method, "EditOrderByLabel should call private/edit_by_label")
	assert.JSONEq(t, `{"instrument_name":"BTC-PERPETUAL","label":"gct","amount":20,"post_only":true,"valid_until":1792368000000,"isolated":true,"allocated_margin":3}`, string(sent.Params), "EditOrderByLabel should send the valid until time in Unix milliseconds")

	skipLiveUnlessCanManipulate(t)
	req := mockDerivativesEditOrderByLabelRequest()
	if !mockTests {
		placed, _ := placeDerivativesTradingLiveOrder(t)
		req = &DerivativesEditOrderByLabelRequest{InstrumentName: placed.InstrumentName, Label: placed.Label, Amount: placed.Amount, Price: placed.Price - 10, PostOnly: true, RejectPostOnly: true}
	}
	resp, err := e.GlobalDerivatives().EditOrderByLabel(t.Context(), req)
	require.NoError(t, err, "EditOrderByLabel must not error")
	if mockTests {
		assert.Equal(t, testDerivativesEditedOrderResponse(), resp, "EditOrderByLabel should decode every field")
	} else {
		assert.Equal(t, req.Price, resp.Order.Price.Value, "EditOrderByLabel should reprice the order")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().EditOrderByLabel(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "EditOrderByLabel should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().EditOrderByLabel(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "EditOrderByLabel should return the request failure")
}

func TestGlobalDerivativesGetMargins(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetMargins(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMargins must reject a nil request")
	_, err = e.GlobalDerivatives().GetMargins(t.Context(), &DerivativesMarginsRequest{Amount: 1, Price: 1})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetMargins must require an instrument name")
	_, err = e.GlobalDerivatives().GetMargins(t.Context(), &DerivativesMarginsRequest{InstrumentName: "SOL_USDC-PERPETUAL", Price: 1})
	require.ErrorIs(t, err, errInvalidAmount, "GetMargins must require an amount")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesMarginsRequest()
	if !mockTests {
		req = &DerivativesMarginsRequest{InstrumentName: derivativesTradingLiveInstrument, Amount: derivativesTradingLiveAmount, Price: getDerivativesTradingLiveTicker(t).MarkPrice}
	}
	resp, err := e.GlobalDerivatives().GetMargins(t.Context(), req)
	require.NoError(t, err, "GetMargins must not error")
	if mockTests {
		assert.Equal(t, testDerivativesMarginsResponse(), resp, "GetMargins should decode every field")
	} else {
		assert.Positive(t, resp.MaximumPrice, "GetMargins should return the highest buy price")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().GetMargins(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetMargins should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetMargins(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMargins should return the request failure")
}

func TestGlobalDerivativesGetMaxOrderSize(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetMaxOrderSize(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMaxOrderSize must reject a nil request")
	_, err = e.GlobalDerivatives().GetMaxOrderSize(t.Context(), &DerivativesMaxOrderSizeRequest{Leverage: 2})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetMaxOrderSize must require an instrument name")
	_, err = e.GlobalDerivatives().GetMaxOrderSize(t.Context(), &DerivativesMaxOrderSizeRequest{InstrumentName: derivativesTradingLiveInstrument, Leverage: 0.5})
	require.ErrorIs(t, err, errDerivativesInvalidLeverage, "GetMaxOrderSize must refuse a leverage under 1")
	_, err = e.GlobalDerivatives().GetMaxOrderSize(t.Context(), &DerivativesMaxOrderSizeRequest{InstrumentName: derivativesTradingLiveInstrument, TakerFeeRate: new(-0.0001)})
	require.ErrorIs(t, err, errDerivativesInvalidTakerFeeRate, "GetMaxOrderSize must refuse a negative taker fee rate")
	_, err = e.GlobalDerivatives().GetMaxOrderSize(t.Context(), &DerivativesMaxOrderSizeRequest{InstrumentName: derivativesTradingLiveInstrument, Price: -1})
	require.ErrorIs(t, err, errInvalidPrice, "GetMaxOrderSize must refuse a negative price")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":{}}`)
	_, err = ex.GlobalDerivatives().GetMaxOrderSize(t.Context(), &DerivativesMaxOrderSizeRequest{InstrumentName: derivativesTradingLiveInstrument, SubaccountID: 7, Isolated: true, Leverage: 5, TakerFeeRate: new(0.0)})
	require.NoError(t, err, "GetMaxOrderSize must not error")
	assert.JSONEq(t, `{"instrument_name":"BTC_USDC-PERPETUAL","subaccount_id":7,"isolated":true,"leverage":5,"taker_fee_rate":0}`, string(lastRequest().Params), "GetMaxOrderSize should send a zero taker fee rate and leave an unset price out")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesMaxOrderSizeRequest()
	if !mockTests {
		req = &DerivativesMaxOrderSizeRequest{InstrumentName: derivativesTradingLiveInstrument}
	}
	resp, err := e.GlobalDerivatives().GetMaxOrderSize(t.Context(), req)
	require.NoError(t, err, "GetMaxOrderSize must not error")
	if mockTests {
		assert.Equal(t, testDerivativesMaxOrderSizeResponse(), resp, "GetMaxOrderSize should decode every field")
	} else {
		assert.Positive(t, resp.BuyPrice, "GetMaxOrderSize should return the reference buy price")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().GetMaxOrderSize(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetMaxOrderSize should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetMaxOrderSize(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMaxOrderSize should return the request failure")
}

func TestGlobalDerivativesGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetOpenOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenOrders must reject a nil request")
	_, err = e.GlobalDerivatives().GetOpenOrders(t.Context(), &DerivativesOpenOrdersRequest{Kind: DerivativesKindAny})
	require.ErrorIs(t, err, errDerivativesInvalidTradingKind, "GetOpenOrders must refuse a kind it does not take")
	_, err = e.GlobalDerivatives().GetOpenOrders(t.Context(), &DerivativesOpenOrdersRequest{Type: DerivativesCancelOrderTypeStop})
	require.ErrorIs(t, err, errDerivativesInvalidOpenOrderType, "GetOpenOrders must refuse an order type only cancel methods take")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":[]}`)
	_, err = ex.GlobalDerivatives().GetOpenOrders(t.Context(), &DerivativesOpenOrdersRequest{Kind: DerivativesKindFuture, Type: DerivativesOpenOrderTypeTriggerAll, SubaccountID: 123456})
	require.NoError(t, err, "GetOpenOrders must not error")
	assert.JSONEq(t, `{"kind":"future","type":"trigger_all","subaccount_id":123456}`, string(lastRequest().Params), "GetOpenOrders should send a subaccount ID")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesOpenOrdersRequest()
	if !mockTests {
		req = &DerivativesOpenOrdersRequest{}
	}
	resp, err := e.GlobalDerivatives().GetOpenOrders(t.Context(), req)
	require.NoError(t, err, "GetOpenOrders must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOpenOrders(), resp, "GetOpenOrders should decode every field")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetOpenOrders(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOpenOrders should return the request failure")
}

func TestGlobalDerivativesGetOpenOrdersByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetOpenOrdersByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenOrdersByCurrency must reject a nil request")
	_, err = e.GlobalDerivatives().GetOpenOrdersByCurrency(t.Context(), &DerivativesOpenOrdersByCurrencyRequest{Kind: DerivativesKindFuture})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetOpenOrdersByCurrency must require a currency")
	_, err = e.GlobalDerivatives().GetOpenOrdersByCurrency(t.Context(), &DerivativesOpenOrdersByCurrencyRequest{Currency: currency.USDC, Kind: DerivativesKindCombo})
	require.ErrorIs(t, err, errDerivativesInvalidTradingKind, "GetOpenOrdersByCurrency must refuse a kind it does not take")
	_, err = e.GlobalDerivatives().GetOpenOrdersByCurrency(t.Context(), &DerivativesOpenOrdersByCurrencyRequest{Currency: currency.USDC, Type: DerivativesCancelOrderTypeTake})
	require.ErrorIs(t, err, errDerivativesInvalidOpenOrderType, "GetOpenOrdersByCurrency must refuse an order type only cancel methods take")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":[]}`)
	_, err = ex.GlobalDerivatives().GetOpenOrdersByCurrency(t.Context(), &DerivativesOpenOrdersByCurrencyRequest{Currency: currency.NewCode("usdc")})
	require.NoError(t, err, "GetOpenOrdersByCurrency must not error")
	assert.JSONEq(t, `{"currency":"USDC"}`, string(lastRequest().Params), "GetOpenOrdersByCurrency should send the currency in upper case")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesOpenOrdersByCurrencyRequest()
	if !mockTests {
		req = &DerivativesOpenOrdersByCurrencyRequest{Currency: derivativesTradingLiveCurrency}
	}
	resp, err := e.GlobalDerivatives().GetOpenOrdersByCurrency(t.Context(), req)
	require.NoError(t, err, "GetOpenOrdersByCurrency must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOpenOrders(), resp, "GetOpenOrdersByCurrency should decode every field")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetOpenOrdersByCurrency(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOpenOrdersByCurrency should return the request failure")
}

func TestGlobalDerivativesGetOpenOrdersByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetOpenOrdersByInstrument(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenOrdersByInstrument must reject a nil request")
	_, err = e.GlobalDerivatives().GetOpenOrdersByInstrument(t.Context(), &DerivativesOpenOrdersByInstrumentRequest{Type: DerivativesOpenOrderTypeAll})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetOpenOrdersByInstrument must require an instrument name")
	_, err = e.GlobalDerivatives().GetOpenOrdersByInstrument(t.Context(), &DerivativesOpenOrdersByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument, Type: DerivativesCancelOrderTypeStop})
	require.ErrorIs(t, err, errDerivativesInvalidOpenOrderType, "GetOpenOrdersByInstrument must refuse an order type only cancel methods take")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesOpenOrdersByInstrumentRequest()
	if !mockTests {
		req = &DerivativesOpenOrdersByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument}
	}
	resp, err := e.GlobalDerivatives().GetOpenOrdersByInstrument(t.Context(), req)
	require.NoError(t, err, "GetOpenOrdersByInstrument must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOpenOrders(), resp, "GetOpenOrdersByInstrument should decode every field")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetOpenOrdersByInstrument(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOpenOrdersByInstrument should return the request failure")
}

func TestGlobalDerivativesGetOpenOrdersByLabel(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetOpenOrdersByLabel(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenOrdersByLabel must reject a nil request")
	_, err = e.GlobalDerivatives().GetOpenOrdersByLabel(t.Context(), &DerivativesOpenOrdersByLabelRequest{Label: "fooBar"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetOpenOrdersByLabel must require a currency")
	_, err = e.GlobalDerivatives().GetOpenOrdersByLabel(t.Context(), &DerivativesOpenOrdersByLabelRequest{Currency: currency.USDC})
	require.ErrorIs(t, err, errDerivativesLabelRequired, "GetOpenOrdersByLabel must require a label")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":[]}`)
	_, err = ex.GlobalDerivatives().GetOpenOrdersByLabel(t.Context(), &DerivativesOpenOrdersByLabelRequest{Currency: currency.NewCode("usdc"), Label: "gct"})
	require.NoError(t, err, "GetOpenOrdersByLabel must not error")
	assert.JSONEq(t, `{"currency":"USDC","label":"gct"}`, string(lastRequest().Params), "GetOpenOrdersByLabel should send the currency in upper case")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesOpenOrdersByLabelRequest()
	if !mockTests {
		req = &DerivativesOpenOrdersByLabelRequest{Currency: derivativesTradingLiveCurrency, Label: "gct-no-orders"}
	}
	resp, err := e.GlobalDerivatives().GetOpenOrdersByLabel(t.Context(), req)
	require.NoError(t, err, "GetOpenOrdersByLabel must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOpenOrders(), resp, "GetOpenOrdersByLabel should decode every field")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetOpenOrdersByLabel(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOpenOrdersByLabel should return the request failure")
}

func TestGlobalDerivativesGetOrderHistoryByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetOrderHistoryByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderHistoryByCurrency must reject a nil request")
	_, err = e.GlobalDerivatives().GetOrderHistoryByCurrency(t.Context(), &DerivativesOrderHistoryByCurrencyRequest{Count: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetOrderHistoryByCurrency must require a currency")
	_, err = e.GlobalDerivatives().GetOrderHistoryByCurrency(t.Context(), &DerivativesOrderHistoryByCurrencyRequest{Currency: currency.USDC, Kind: "perpetual"})
	require.ErrorIs(t, err, errDerivativesInvalidTradingKind, "GetOrderHistoryByCurrency must refuse an undocumented kind")
	_, err = e.GlobalDerivatives().GetOrderHistoryByCurrency(t.Context(), &DerivativesOrderHistoryByCurrencyRequest{Currency: currency.USDC, Count: derivativesMaximumTradingCount + 1})
	require.ErrorIs(t, err, errDerivativesInvalidTradingCount, "GetOrderHistoryByCurrency must refuse a count over 1000")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":[]}`)
	_, err = ex.GlobalDerivatives().GetOrderHistoryByCurrency(t.Context(), &DerivativesOrderHistoryByCurrencyRequest{Currency: currency.NewCode("usdc"), Kind: DerivativesKindCombo, WithContinuation: true, Continuation: "next", Historical: true})
	require.NoError(t, err, "GetOrderHistoryByCurrency must not error")
	assert.JSONEq(t, `{"currency":"USDC","kind":"combo","with_continuation":true,"continuation":"next","historical":true}`, string(lastRequest().Params), "GetOrderHistoryByCurrency should send the currency in upper case")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesOrderHistoryByCurrencyRequest()
	if !mockTests {
		req = &DerivativesOrderHistoryByCurrencyRequest{Currency: derivativesTradingLiveCurrency, Count: 1}
	}
	resp, err := e.GlobalDerivatives().GetOrderHistoryByCurrency(t.Context(), req)
	require.NoError(t, err, "GetOrderHistoryByCurrency must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOrderHistoryResponse(), resp, "GetOrderHistoryByCurrency should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Orders), 1, "GetOrderHistoryByCurrency should return at most the count of orders")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().GetOrderHistoryByCurrency(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOrderHistoryByCurrency should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetOrderHistoryByCurrency(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderHistoryByCurrency should return the request failure")
}

func TestGlobalDerivativesGetOrderHistoryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetOrderHistoryByInstrument(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderHistoryByInstrument must reject a nil request")
	_, err = e.GlobalDerivatives().GetOrderHistoryByInstrument(t.Context(), &DerivativesOrderHistoryByInstrumentRequest{Count: 1})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetOrderHistoryByInstrument must require an instrument name")
	_, err = e.GlobalDerivatives().GetOrderHistoryByInstrument(t.Context(), &DerivativesOrderHistoryByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument, Count: derivativesMaximumTradingCount + 1})
	require.ErrorIs(t, err, errDerivativesInvalidTradingCount, "GetOrderHistoryByInstrument must refuse a count over 1000")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesOrderHistoryByInstrumentRequest()
	if !mockTests {
		req = &DerivativesOrderHistoryByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument, Count: 1}
	}
	resp, err := e.GlobalDerivatives().GetOrderHistoryByInstrument(t.Context(), req)
	require.NoError(t, err, "GetOrderHistoryByInstrument must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOrderHistoryResponse(), resp, "GetOrderHistoryByInstrument should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Orders), 1, "GetOrderHistoryByInstrument should return at most the count of orders")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().GetOrderHistoryByInstrument(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOrderHistoryByInstrument should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetOrderHistoryByInstrument(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderHistoryByInstrument should return the request failure")
}

func TestGlobalDerivativesGetOrderMarginByIDs(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetOrderMarginByIDs(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderMarginByIDs must reject a nil request")
	_, err = e.GlobalDerivatives().GetOrderMarginByIDs(t.Context(), &DerivativesOrderMarginByIDsRequest{Isolated: true})
	require.ErrorIs(t, err, errDerivativesOrderIDRequired, "GetOrderMarginByIDs must require an order ID")
	_, err = e.GlobalDerivatives().GetOrderMarginByIDs(t.Context(), &DerivativesOrderMarginByIDsRequest{OrderIDs: []string{"ETH-349278", ""}})
	require.ErrorIs(t, err, errDerivativesOrderIDRequired, "GetOrderMarginByIDs must refuse an empty order ID in the list")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":[]}`)
	_, err = ex.GlobalDerivatives().GetOrderMarginByIDs(t.Context(), &DerivativesOrderMarginByIDsRequest{OrderIDs: []string{"SOL_USDC-1"}, Isolated: true})
	require.NoError(t, err, "GetOrderMarginByIDs must not error")
	assert.JSONEq(t, `{"ids":["SOL_USDC-1"],"isolated":true}`, string(lastRequest().Params), "GetOrderMarginByIDs should send the order IDs as a list")

	skipLiveUnlessCanManipulate(t)
	req := mockDerivativesOrderMarginByIDsRequest()
	if !mockTests {
		_, order := placeDerivativesTradingLiveOrder(t)
		req = &DerivativesOrderMarginByIDsRequest{OrderIDs: []string{order.OrderID}}
	}
	resp, err := e.GlobalDerivatives().GetOrderMarginByIDs(t.Context(), req)
	require.NoError(t, err, "GetOrderMarginByIDs must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOrderMargins(), resp, "GetOrderMarginByIDs should decode every field")
	} else if assert.Len(t, resp, 1, "GetOrderMarginByIDs should return the order's margin") {
		assert.Equal(t, req.OrderIDs[0], resp[0].OrderID, "GetOrderMarginByIDs should return the order's margin")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetOrderMarginByIDs(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderMarginByIDs should return the request failure")
}

func TestGlobalDerivativesGetOrderState(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetOrderState(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderState must reject a nil request")
	_, err = e.GlobalDerivatives().GetOrderState(t.Context(), &DerivativesOrderStateRequest{Isolated: true})
	require.ErrorIs(t, err, errDerivativesOrderIDRequired, "GetOrderState must require an order ID")

	skipLiveUnlessCanManipulate(t)
	req := mockDerivativesOrderStateRequest()
	if !mockTests {
		_, order := placeDerivativesTradingLiveOrder(t)
		req = &DerivativesOrderStateRequest{OrderID: order.OrderID}
	}
	resp, err := e.GlobalDerivatives().GetOrderState(t.Context(), req)
	require.NoError(t, err, "GetOrderState must not error")
	if mockTests {
		assert.Equal(t, testDerivativesOrderState(""), resp, "GetOrderState should decode every field")
	} else {
		assert.Equal(t, req.OrderID, resp.OrderID, "GetOrderState should return the order")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().GetOrderState(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOrderState should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetOrderState(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderState should return the request failure")
}

func TestGlobalDerivativesGetOrderStateByLabel(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetOrderStateByLabel(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderStateByLabel must reject a nil request")
	_, err = e.GlobalDerivatives().GetOrderStateByLabel(t.Context(), &DerivativesOrderStateByLabelRequest{Label: "fooBar"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetOrderStateByLabel must require a currency")
	_, err = e.GlobalDerivatives().GetOrderStateByLabel(t.Context(), &DerivativesOrderStateByLabelRequest{Currency: currency.ETH})
	require.ErrorIs(t, err, errDerivativesLabelRequired, "GetOrderStateByLabel must require a label")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":[]}`)
	_, err = ex.GlobalDerivatives().GetOrderStateByLabel(t.Context(), &DerivativesOrderStateByLabelRequest{Currency: currency.NewCode("eth"), Label: "gct"})
	require.NoError(t, err, "GetOrderStateByLabel must not error")
	assert.JSONEq(t, `{"currency":"ETH","label":"gct"}`, string(lastRequest().Params), "GetOrderStateByLabel should send the currency in upper case")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesOrderStateByLabelRequest()
	if !mockTests {
		req = &DerivativesOrderStateByLabelRequest{Currency: derivativesTradingLiveCurrency, Label: "gct-no-orders"}
	}
	resp, err := e.GlobalDerivatives().GetOrderStateByLabel(t.Context(), req)
	require.NoError(t, err, "GetOrderStateByLabel must not error")
	if mockTests {
		assert.Equal(t, []DerivativesOrderResponse{*testDerivativesOrderState("fooBar")}, resp, "GetOrderStateByLabel should decode every field")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetOrderStateByLabel(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderStateByLabel should return the request failure")
}

func TestGlobalDerivativesGetSettlementHistoryByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetSettlementHistoryByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSettlementHistoryByCurrency must reject a nil request")
	_, err = e.GlobalDerivatives().GetSettlementHistoryByCurrency(t.Context(), &DerivativesSettlementHistoryByCurrencyRequest{Count: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSettlementHistoryByCurrency must require a currency")
	_, err = e.GlobalDerivatives().GetSettlementHistoryByCurrency(t.Context(), &DerivativesSettlementHistoryByCurrencyRequest{Currency: currency.USDC, Type: "exercise"})
	require.ErrorIs(t, err, errDerivativesInvalidSettlementHistoryType, "GetSettlementHistoryByCurrency must refuse an undocumented settlement type")
	_, err = e.GlobalDerivatives().GetSettlementHistoryByCurrency(t.Context(), &DerivativesSettlementHistoryByCurrencyRequest{Currency: currency.USDC, Count: derivativesMaximumTradingCount + 1})
	require.ErrorIs(t, err, errDerivativesInvalidTradingCount, "GetSettlementHistoryByCurrency must refuse a count over 1000")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":{"settlements":[]}}`)
	_, err = ex.GlobalDerivatives().GetSettlementHistoryByCurrency(t.Context(), &DerivativesSettlementHistoryByCurrencyRequest{Currency: currency.NewCode("usdc"), Type: "delivery", SearchStartTime: time.UnixMilli(1792368000000)})
	require.NoError(t, err, "GetSettlementHistoryByCurrency must not error")
	assert.JSONEq(t, `{"currency":"USDC","type":"delivery","search_start_timestamp":1792368000000}`, string(lastRequest().Params), "GetSettlementHistoryByCurrency should send the search start time in Unix milliseconds")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesSettlementHistoryByCurrencyRequest()
	if !mockTests {
		req = &DerivativesSettlementHistoryByCurrencyRequest{Currency: derivativesTradingLiveCurrency, Count: 1}
	}
	resp, err := e.GlobalDerivatives().GetSettlementHistoryByCurrency(t.Context(), req)
	require.NoError(t, err, "GetSettlementHistoryByCurrency must not error")
	if mockTests {
		assert.Equal(t, testDerivativesSettlementHistoryResponse(), resp, "GetSettlementHistoryByCurrency should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Settlements), 1, "GetSettlementHistoryByCurrency should return at most the count of events")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().GetSettlementHistoryByCurrency(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetSettlementHistoryByCurrency should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetSettlementHistoryByCurrency(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSettlementHistoryByCurrency should return the request failure")
}

func TestGlobalDerivativesGetSettlementHistoryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetSettlementHistoryByInstrument(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSettlementHistoryByInstrument must reject a nil request")
	_, err = e.GlobalDerivatives().GetSettlementHistoryByInstrument(t.Context(), &DerivativesSettlementHistoryByInstrumentRequest{Count: 1})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetSettlementHistoryByInstrument must require an instrument name")
	_, err = e.GlobalDerivatives().GetSettlementHistoryByInstrument(t.Context(), &DerivativesSettlementHistoryByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument, Type: "exercise"})
	require.ErrorIs(t, err, errDerivativesInvalidSettlementHistoryType, "GetSettlementHistoryByInstrument must refuse an undocumented settlement type")
	_, err = e.GlobalDerivatives().GetSettlementHistoryByInstrument(t.Context(), &DerivativesSettlementHistoryByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument, Count: derivativesMaximumTradingCount + 1})
	require.ErrorIs(t, err, errDerivativesInvalidTradingCount, "GetSettlementHistoryByInstrument must refuse a count over 1000")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":{"settlements":[]}}`)
	_, err = ex.GlobalDerivatives().GetSettlementHistoryByInstrument(t.Context(), &DerivativesSettlementHistoryByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument, Continuation: "next", SearchStartTime: time.UnixMilli(1792368000000)})
	require.NoError(t, err, "GetSettlementHistoryByInstrument must not error")
	assert.JSONEq(t, `{"instrument_name":"BTC_USDC-PERPETUAL","continuation":"next","search_start_timestamp":1792368000000}`, string(lastRequest().Params), "GetSettlementHistoryByInstrument should send the search start time in Unix milliseconds")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesSettlementHistoryByInstrumentRequest()
	if !mockTests {
		req = &DerivativesSettlementHistoryByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument, Count: 1}
	}
	resp, err := e.GlobalDerivatives().GetSettlementHistoryByInstrument(t.Context(), req)
	require.NoError(t, err, "GetSettlementHistoryByInstrument must not error")
	if mockTests {
		assert.Equal(t, testDerivativesSettlementHistoryResponse(), resp, "GetSettlementHistoryByInstrument should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Settlements), 1, "GetSettlementHistoryByInstrument should return at most the count of events")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().GetSettlementHistoryByInstrument(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetSettlementHistoryByInstrument should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetSettlementHistoryByInstrument(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSettlementHistoryByInstrument should return the request failure")
}

func TestGlobalDerivativesGetTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetTriggerOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTriggerOrderHistory must reject a nil request")
	_, err = e.GlobalDerivatives().GetTriggerOrderHistory(t.Context(), &DerivativesTriggerOrderHistoryRequest{Count: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTriggerOrderHistory must require a currency")
	_, err = e.GlobalDerivatives().GetTriggerOrderHistory(t.Context(), &DerivativesTriggerOrderHistoryRequest{Currency: currency.USDC, Count: derivativesMaximumTradingCount + 1})
	require.ErrorIs(t, err, errDerivativesInvalidTradingCount, "GetTriggerOrderHistory must refuse a count over 1000")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":{"entries":[]}}`)
	_, err = ex.GlobalDerivatives().GetTriggerOrderHistory(t.Context(), &DerivativesTriggerOrderHistoryRequest{Currency: currency.NewCode("usdc"), Continuation: "next"})
	require.NoError(t, err, "GetTriggerOrderHistory must not error")
	assert.JSONEq(t, `{"currency":"USDC","continuation":"next"}`, string(lastRequest().Params), "GetTriggerOrderHistory should send the currency in upper case")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesTriggerOrderHistoryRequest()
	if !mockTests {
		req = &DerivativesTriggerOrderHistoryRequest{Currency: derivativesTradingLiveCurrency, Count: 1}
	}
	resp, err := e.GlobalDerivatives().GetTriggerOrderHistory(t.Context(), req)
	require.NoError(t, err, "GetTriggerOrderHistory must not error")
	if mockTests {
		assert.Equal(t, testDerivativesTriggerOrderHistoryResponse(), resp, "GetTriggerOrderHistory should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Entries), 1, "GetTriggerOrderHistory should return at most the count of entries")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().GetTriggerOrderHistory(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTriggerOrderHistory should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetTriggerOrderHistory(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTriggerOrderHistory should return the request failure")
}

func TestGlobalDerivativesGetUserTradesByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetUserTradesByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserTradesByCurrency must reject a nil request")
	_, err = e.GlobalDerivatives().GetUserTradesByCurrency(t.Context(), &DerivativesUserTradesByCurrencyRequest{Count: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUserTradesByCurrency must require a currency")
	_, err = e.GlobalDerivatives().GetUserTradesByCurrency(t.Context(), &DerivativesUserTradesByCurrencyRequest{Currency: currency.USDC, Kind: "perpetual"})
	require.ErrorIs(t, err, errDerivativesInvalidTradingKind, "GetUserTradesByCurrency must refuse an undocumented kind")
	_, err = e.GlobalDerivatives().GetUserTradesByCurrency(t.Context(), &DerivativesUserTradesByCurrencyRequest{Currency: currency.USDC, Count: derivativesMaximumTradingCount + 1})
	require.ErrorIs(t, err, errDerivativesInvalidTradingCount, "GetUserTradesByCurrency must refuse a count over 1000")
	_, err = e.GlobalDerivatives().GetUserTradesByCurrency(t.Context(), &DerivativesUserTradesByCurrencyRequest{Currency: currency.USDC, Sorting: "newest"})
	require.ErrorIs(t, err, errDerivativesInvalidUserTradesSorting, "GetUserTradesByCurrency must refuse an undocumented sorting")
	_, err = e.GlobalDerivatives().GetUserTradesByCurrency(t.Context(), &DerivativesUserTradesByCurrencyRequest{Currency: currency.USDC, StartTime: time.UnixMilli(2), EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserTradesByCurrency must refuse a start after the end")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":{"trades":[]}}`)
	_, err = ex.GlobalDerivatives().GetUserTradesByCurrency(t.Context(), &DerivativesUserTradesByCurrencyRequest{Currency: currency.NewCode("usdc"), StartID: "USDC-1", EndID: "USDC-9", EndTime: time.UnixMilli(1792368000000), Historical: true, SubaccountID: 123456})
	require.NoError(t, err, "GetUserTradesByCurrency must not error")
	assert.JSONEq(t, `{"currency":"USDC","start_id":"USDC-1","end_id":"USDC-9","end_timestamp":1792368000000,"historical":true,"subaccount_id":123456}`, string(lastRequest().Params), "GetUserTradesByCurrency should send a set end time alone, in Unix milliseconds")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesUserTradesByCurrencyRequest()
	if !mockTests {
		req = &DerivativesUserTradesByCurrencyRequest{Currency: derivativesTradingLiveCurrency, Count: 1}
	}
	resp, err := e.GlobalDerivatives().GetUserTradesByCurrency(t.Context(), req)
	require.NoError(t, err, "GetUserTradesByCurrency must not error")
	if mockTests {
		assert.Equal(t, testDerivativesUserTradesResponse(), resp, "GetUserTradesByCurrency should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Trades), 1, "GetUserTradesByCurrency should return at most the count of trades")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().GetUserTradesByCurrency(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserTradesByCurrency should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetUserTradesByCurrency(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserTradesByCurrency should return the request failure")
}

func TestGlobalDerivativesGetUserTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetUserTradesByCurrencyAndTime(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserTradesByCurrencyAndTime must reject a nil request")
	_, err = e.GlobalDerivatives().GetUserTradesByCurrencyAndTime(t.Context(), &DerivativesUserTradesByCurrencyAndTimeRequest{EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUserTradesByCurrencyAndTime must require a currency")
	_, err = e.GlobalDerivatives().GetUserTradesByCurrencyAndTime(t.Context(), &DerivativesUserTradesByCurrencyAndTimeRequest{Currency: currency.USDC})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetUserTradesByCurrencyAndTime must require a start or an end")
	_, err = e.GlobalDerivatives().GetUserTradesByCurrencyAndTime(t.Context(), &DerivativesUserTradesByCurrencyAndTimeRequest{Currency: currency.USDC, EndTime: time.UnixMilli(1), Kind: "perpetual"})
	require.ErrorIs(t, err, errDerivativesInvalidTradingKind, "GetUserTradesByCurrencyAndTime must refuse an undocumented kind")
	_, err = e.GlobalDerivatives().GetUserTradesByCurrencyAndTime(t.Context(), &DerivativesUserTradesByCurrencyAndTimeRequest{Currency: currency.USDC, EndTime: time.UnixMilli(1), Count: derivativesMaximumTradingCount + 1})
	require.ErrorIs(t, err, errDerivativesInvalidTradingCount, "GetUserTradesByCurrencyAndTime must refuse a count over 1000")
	_, err = e.GlobalDerivatives().GetUserTradesByCurrencyAndTime(t.Context(), &DerivativesUserTradesByCurrencyAndTimeRequest{Currency: currency.USDC, EndTime: time.UnixMilli(1), Sorting: "newest"})
	require.ErrorIs(t, err, errDerivativesInvalidUserTradesSorting, "GetUserTradesByCurrencyAndTime must refuse an undocumented sorting")
	_, err = e.GlobalDerivatives().GetUserTradesByCurrencyAndTime(t.Context(), &DerivativesUserTradesByCurrencyAndTimeRequest{Currency: currency.USDC, StartTime: time.UnixMilli(2), EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserTradesByCurrencyAndTime must refuse a start after the end")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":{"trades":[]}}`)
	_, err = ex.GlobalDerivatives().GetUserTradesByCurrencyAndTime(t.Context(), &DerivativesUserTradesByCurrencyAndTimeRequest{Currency: currency.NewCode("usdc"), Kind: DerivativesKindFuture, StartTime: time.UnixMilli(1792368000000), Sorting: DerivativesSortingDefault, SubaccountID: 123456})
	require.NoError(t, err, "GetUserTradesByCurrencyAndTime must not error")
	assert.JSONEq(t, `{"currency":"USDC","kind":"future","start_timestamp":1792368000000,"sorting":"default","subaccount_id":123456}`, string(lastRequest().Params), "GetUserTradesByCurrencyAndTime should send a set start time alone, in Unix milliseconds")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesUserTradesByCurrencyAndTimeRequest()
	if !mockTests {
		req = &DerivativesUserTradesByCurrencyAndTimeRequest{Currency: derivativesTradingLiveCurrency, StartTime: time.Now().Add(-24 * time.Hour), EndTime: time.Now(), Count: 1}
	}
	resp, err := e.GlobalDerivatives().GetUserTradesByCurrencyAndTime(t.Context(), req)
	require.NoError(t, err, "GetUserTradesByCurrencyAndTime must not error")
	if mockTests {
		assert.Equal(t, testDerivativesUserTradesResponse(), resp, "GetUserTradesByCurrencyAndTime should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Trades), 1, "GetUserTradesByCurrencyAndTime should return at most the count of trades")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().GetUserTradesByCurrencyAndTime(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserTradesByCurrencyAndTime should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetUserTradesByCurrencyAndTime(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserTradesByCurrencyAndTime should return the request failure")
}

func TestGlobalDerivativesGetUserTradesByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetUserTradesByInstrument(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserTradesByInstrument must reject a nil request")
	_, err = e.GlobalDerivatives().GetUserTradesByInstrument(t.Context(), &DerivativesUserTradesByInstrumentRequest{Count: 1})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetUserTradesByInstrument must require an instrument name")
	_, err = e.GlobalDerivatives().GetUserTradesByInstrument(t.Context(), &DerivativesUserTradesByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument, Count: derivativesMaximumTradingCount + 1})
	require.ErrorIs(t, err, errDerivativesInvalidTradingCount, "GetUserTradesByInstrument must refuse a count over 1000")
	_, err = e.GlobalDerivatives().GetUserTradesByInstrument(t.Context(), &DerivativesUserTradesByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument, Sorting: "newest"})
	require.ErrorIs(t, err, errDerivativesInvalidUserTradesSorting, "GetUserTradesByInstrument must refuse an undocumented sorting")
	_, err = e.GlobalDerivatives().GetUserTradesByInstrument(t.Context(), &DerivativesUserTradesByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument, StartTime: time.UnixMilli(2), EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserTradesByInstrument must refuse a start after the end")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":{"trades":[]}}`)
	_, err = ex.GlobalDerivatives().GetUserTradesByInstrument(t.Context(), &DerivativesUserTradesByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument, StartSequence: 3, EndSequence: 9, StartTime: time.UnixMilli(1792368000000), EndTime: time.UnixMilli(1792371600000)})
	require.NoError(t, err, "GetUserTradesByInstrument must not error")
	assert.JSONEq(t, `{"instrument_name":"BTC_USDC-PERPETUAL","start_seq":3,"end_seq":9,"start_timestamp":1792368000000,"end_timestamp":1792371600000}`, string(lastRequest().Params), "GetUserTradesByInstrument should send the time range in Unix milliseconds")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesUserTradesByInstrumentRequest()
	if !mockTests {
		req = &DerivativesUserTradesByInstrumentRequest{InstrumentName: derivativesTradingLiveInstrument, Count: 1}
	}
	resp, err := e.GlobalDerivatives().GetUserTradesByInstrument(t.Context(), req)
	require.NoError(t, err, "GetUserTradesByInstrument must not error")
	if mockTests {
		assert.Equal(t, testDerivativesUserTradesResponse(), resp, "GetUserTradesByInstrument should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Trades), 1, "GetUserTradesByInstrument should return at most the count of trades")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().GetUserTradesByInstrument(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserTradesByInstrument should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetUserTradesByInstrument(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserTradesByInstrument should return the request failure")
}

func TestGlobalDerivativesGetUserTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetUserTradesByInstrumentAndTime(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserTradesByInstrumentAndTime must reject a nil request")
	_, err = e.GlobalDerivatives().GetUserTradesByInstrumentAndTime(t.Context(), &DerivativesUserTradesByInstrumentAndTimeRequest{EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetUserTradesByInstrumentAndTime must require an instrument name")
	_, err = e.GlobalDerivatives().GetUserTradesByInstrumentAndTime(t.Context(), &DerivativesUserTradesByInstrumentAndTimeRequest{InstrumentName: derivativesTradingLiveInstrument})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetUserTradesByInstrumentAndTime must require a start or an end")
	_, err = e.GlobalDerivatives().GetUserTradesByInstrumentAndTime(t.Context(), &DerivativesUserTradesByInstrumentAndTimeRequest{InstrumentName: derivativesTradingLiveInstrument, EndTime: time.UnixMilli(1), Count: derivativesMaximumTradingCount + 1})
	require.ErrorIs(t, err, errDerivativesInvalidTradingCount, "GetUserTradesByInstrumentAndTime must refuse a count over 1000")
	_, err = e.GlobalDerivatives().GetUserTradesByInstrumentAndTime(t.Context(), &DerivativesUserTradesByInstrumentAndTimeRequest{InstrumentName: derivativesTradingLiveInstrument, EndTime: time.UnixMilli(1), Sorting: "newest"})
	require.ErrorIs(t, err, errDerivativesInvalidUserTradesSorting, "GetUserTradesByInstrumentAndTime must refuse an undocumented sorting")
	_, err = e.GlobalDerivatives().GetUserTradesByInstrumentAndTime(t.Context(), &DerivativesUserTradesByInstrumentAndTimeRequest{InstrumentName: derivativesTradingLiveInstrument, StartTime: time.UnixMilli(2), EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserTradesByInstrumentAndTime must refuse a start after the end")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":{"trades":[]}}`)
	_, err = ex.GlobalDerivatives().GetUserTradesByInstrumentAndTime(t.Context(), &DerivativesUserTradesByInstrumentAndTimeRequest{InstrumentName: derivativesTradingLiveInstrument, EndTime: time.UnixMilli(1792371600000), Historical: true})
	require.NoError(t, err, "GetUserTradesByInstrumentAndTime must not error")
	assert.JSONEq(t, `{"instrument_name":"BTC_USDC-PERPETUAL","end_timestamp":1792371600000,"historical":true}`, string(lastRequest().Params), "GetUserTradesByInstrumentAndTime should send a set end time alone, in Unix milliseconds")

	skipLiveWithoutCredentials(t)
	req := mockDerivativesUserTradesByInstrumentAndTimeRequest()
	if !mockTests {
		req = &DerivativesUserTradesByInstrumentAndTimeRequest{InstrumentName: derivativesTradingLiveInstrument, StartTime: time.Now().Add(-24 * time.Hour), EndTime: time.Now(), Count: 1}
	}
	resp, err := e.GlobalDerivatives().GetUserTradesByInstrumentAndTime(t.Context(), req)
	require.NoError(t, err, "GetUserTradesByInstrumentAndTime must not error")
	if mockTests {
		assert.Equal(t, testDerivativesUserTradesResponse(), resp, "GetUserTradesByInstrumentAndTime should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Trades), 1, "GetUserTradesByInstrumentAndTime should return at most the count of trades")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().GetUserTradesByInstrumentAndTime(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserTradesByInstrumentAndTime should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetUserTradesByInstrumentAndTime(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserTradesByInstrumentAndTime should return the request failure")
}

func TestGlobalDerivativesGetUserTradesByOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetUserTradesByOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserTradesByOrder must reject a nil request")
	_, err = e.GlobalDerivatives().GetUserTradesByOrder(t.Context(), &DerivativesUserTradesByOrderRequest{Isolated: true})
	require.ErrorIs(t, err, errDerivativesOrderIDRequired, "GetUserTradesByOrder must require an order ID")
	_, err = e.GlobalDerivatives().GetUserTradesByOrder(t.Context(), &DerivativesUserTradesByOrderRequest{OrderID: "ETH-584830574", Sorting: "newest"})
	require.ErrorIs(t, err, errDerivativesInvalidUserTradesSorting, "GetUserTradesByOrder must refuse an undocumented sorting")

	ex, lastRequest := newDerivativesTradingParamsServer(t, `{"jsonrpc":"2.0","result":[]}`)
	_, err = ex.GlobalDerivatives().GetUserTradesByOrder(t.Context(), &DerivativesUserTradesByOrderRequest{OrderID: "SOL_USDC-1", Historical: true, SubaccountID: 123456, Isolated: true})
	require.NoError(t, err, "GetUserTradesByOrder must not error")
	assert.JSONEq(t, `{"order_id":"SOL_USDC-1","historical":true,"subaccount_id":123456,"isolated":true}`, string(lastRequest().Params), "GetUserTradesByOrder should send the isolated flag")

	skipLiveUnlessCanManipulate(t)
	req := mockDerivativesUserTradesByOrderRequest()
	if !mockTests {
		_, order := placeDerivativesTradingLiveOrder(t)
		req = &DerivativesUserTradesByOrderRequest{OrderID: order.OrderID}
	}
	resp, err := e.GlobalDerivatives().GetUserTradesByOrder(t.Context(), req)
	require.NoError(t, err, "GetUserTradesByOrder must not error")
	if mockTests {
		assert.Equal(t, testDerivativesUserTradesByOrder(), resp, "GetUserTradesByOrder should decode every field")
	} else {
		assert.Empty(t, resp, "GetUserTradesByOrder should return no trades for an unfilled order")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetUserTradesByOrder(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserTradesByOrder should return the request failure")
}

func TestGlobalDerivativesSell(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().Sell(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Sell must reject a nil request")
	_, err = e.GlobalDerivatives().Sell(t.Context(), &DerivativesOrderRequest{InstrumentName: "SOL_USDC-PERPETUAL"})
	require.ErrorIs(t, err, errInvalidAmount, "Sell must require a size")
	_, err = e.GlobalDerivatives().Sell(t.Context(), &DerivativesOrderRequest{InstrumentName: "SOL_USDC-PERPETUAL", Amount: 10, AllocatedMargin: 5})
	require.ErrorIs(t, err, errDerivativesAllocatedMarginWithoutIsolated, "Sell must refuse to allocate margin to an order that is not isolated")

	ex, lastRequest := newDerivativesTradingParamsServer(t, derivativesTradingOrderResult)
	_, err = ex.GlobalDerivatives().Sell(t.Context(), &DerivativesOrderRequest{InstrumentName: derivativesTradingLiveInstrument, Amount: 0.001, Type: DerivativesOrderTypeTrailingStop, TriggerOffset: 100, Trigger: DerivativesTriggerMarkPrice, ReduceOnly: true})
	require.NoError(t, err, "Sell must not error for a trailing stop")
	sent := lastRequest()
	assert.Equal(t, "private/sell", sent.Method, "Sell should call private/sell")
	assert.JSONEq(t, `{"instrument_name":"BTC_USDC-PERPETUAL","amount":0.001,"type":"trailing_stop","post_only":false,"reduce_only":true,"trigger_offset":100,"trigger":"mark_price"}`, string(sent.Params), "Sell should send a trailing stop's offset and trigger")

	skipLiveUnlessCanManipulate(t)
	req := mockDerivativesSellRequest()
	if !mockTests {
		req = newDerivativesTradingLiveOrderRequest(t, DerivativesDirectionSell)
	}
	resp, err := e.GlobalDerivatives().Sell(t.Context(), req)
	require.NoError(t, err, "Sell must not error")
	if mockTests {
		assert.Equal(t, testDerivativesPlacedOrderResponse(), resp, "Sell should decode every field")
	} else {
		assert.Equal(t, DerivativesOrderStateOpen, resp.Order.OrderState, "Sell should place a resting order")
	}

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().Sell(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "Sell should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().Sell(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "Sell should return the request failure")
}

func TestGlobalDerivativesSetLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().SetLeverage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetLeverage must reject a nil request")
	_, err = e.GlobalDerivatives().SetLeverage(t.Context(), &DerivativesSetLeverageRequest{Leverage: new(5.0)})
	require.ErrorIs(t, err, errInstrumentNameRequired, "SetLeverage must require an instrument name")
	_, err = e.GlobalDerivatives().SetLeverage(t.Context(), &DerivativesSetLeverageRequest{InstrumentName: "SOL_USDC-PERPETUAL", Leverage: new(0.5)})
	require.ErrorIs(t, err, errDerivativesInvalidLeverage, "SetLeverage must refuse a leverage under 1")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		assert.Equal(t, 2.0, setDerivativesTradingLiveLeverage(t, e.GlobalDerivatives().SetLeverage), "SetLeverage should return the leverage set")
		return
	}
	leverage, err := e.GlobalDerivatives().SetLeverage(t.Context(), mockDerivativesSetLeverageRequest())
	require.NoError(t, err, "SetLeverage must not error")
	assert.Equal(t, 50.0, leverage, "SetLeverage should return the leverage stored")
	leverage, err = e.GlobalDerivatives().SetLeverage(t.Context(), mockDerivativesRemoveLeverageRequest())
	require.NoError(t, err, "SetLeverage must not error removing the custom leverage")
	assert.Zero(t, leverage, "SetLeverage should return zero once the custom leverage is removed")

	_, err = newResponseServerExchange(t, derivativesTradingNullResult).GlobalDerivatives().SetLeverage(t.Context(), mockDerivativesSetLeverageRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "SetLeverage should error for a null result to setting a leverage")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().SetLeverage(t.Context(), mockDerivativesSetLeverageRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "SetLeverage should return the request failure")
}

func TestDerivativesCancelAllResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var resp DerivativesCancelAllResponse
	require.NoError(t, json.Unmarshal([]byte(`4`), &resp), "Unmarshal must not error for a count")
	assert.Equal(t, DerivativesCancelAllResponse{CancelledCount: 4}, resp, "UnmarshalJSON should decode the count")

	const detailed = `[{"instrument_name":"BTC_USDC-PERPETUAL","currency":"USDC","type":"limit","result":[{"order_id":"USDC-1","order_state":"cancelled","cancel_reason":"user_request"},{"order_id":"USDC-2"}]},{"currency":"BTC_USDC","type":"stop","result":[]}]`
	require.NoError(t, json.Unmarshal([]byte(detailed), &resp), "Unmarshal must not error for execution reports")
	exp := DerivativesCancelAllResponse{
		CancelledCount: 2,
		ExecutionReports: []DerivativesCancelExecutionReport{
			{
				InstrumentName: "BTC_USDC-PERPETUAL",
				Currency:       "USDC",
				Type:           "limit",
				Result:         []DerivativesOrderResponse{{OrderID: "USDC-1", OrderState: "cancelled", CancelReason: "user_request"}, {OrderID: "USDC-2"}},
			},
			{Currency: "BTC_USDC", Type: "stop", Result: []DerivativesOrderResponse{}},
		},
	}
	assert.Equal(t, exp, resp, "UnmarshalJSON should decode the execution reports and count their orders")

	// The documented example holds a cancelled order in each report
	const documented = `[{"order":{"order_id":"BTC-1","instrument_name":"BTC-PERPETUAL","order_state":"cancelled"}},{"order":{"order_id":"ETH-2","instrument_name":"ETH-PERPETUAL","order_state":"cancelled"}}]`
	require.NoError(t, json.Unmarshal([]byte(documented), &resp), "Unmarshal must not error for the documented execution reports")
	exp = DerivativesCancelAllResponse{
		CancelledCount: 2,
		ExecutionReports: []DerivativesCancelExecutionReport{
			{Order: &DerivativesOrderResponse{OrderID: "BTC-1", InstrumentName: "BTC-PERPETUAL", OrderState: "cancelled"}},
			{Order: &DerivativesOrderResponse{OrderID: "ETH-2", InstrumentName: "ETH-PERPETUAL", OrderState: "cancelled"}},
		},
	}
	assert.Equal(t, exp, resp, "UnmarshalJSON should decode the documented execution reports and count their orders")

	for _, data := range []string{`"4"`, `{}`, `[4]`, `-1`} {
		assert.ErrorIsf(t, json.Unmarshal([]byte(data), &resp), errDerivativesInvalidCancelAllResult, "UnmarshalJSON should refuse %s", data)
	}
}

func TestDerivativesOrderHistoryResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var resp DerivativesOrderHistoryResponse
	require.NoError(t, json.Unmarshal([]byte(`[{"order_id":"1"}]`), &resp), "Unmarshal must not error for a list")
	assert.Equal(t, DerivativesOrderHistoryResponse{Orders: []DerivativesOrderResponse{{OrderID: "1"}}}, resp, "UnmarshalJSON should decode a list of orders")

	require.NoError(t, json.Unmarshal([]byte(`{"orders":[{"order_id":"2"}],"continuation":"next"}`), &resp), "Unmarshal must not error for a page")
	assert.Equal(t, DerivativesOrderHistoryResponse{Orders: []DerivativesOrderResponse{{OrderID: "2"}}, Continuation: "next"}, resp, "UnmarshalJSON should decode a page of orders and its continuation token")

	require.NoError(t, json.Unmarshal([]byte(`{"orders":[],"continuation":""}`), &resp), "Unmarshal must not error for the last page")
	assert.Equal(t, DerivativesOrderHistoryResponse{Orders: []DerivativesOrderResponse{}}, resp, "UnmarshalJSON should decode an empty last page")

	err := json.Unmarshal([]byte(`{"entries":[{"order_id":"3"}],"continuation":"next"}`), &resp)
	assert.ErrorIs(t, err, errDerivativesOrderHistoryOrdersMissing, "UnmarshalJSON should refuse a page without an orders field")
	for _, data := range []string{`[1]`, `{"orders":{}}`, `"orders"`} {
		assert.Errorf(t, json.Unmarshal([]byte(data), &resp), "UnmarshalJSON should refuse %s", data)
	}
}

func TestDerivativesUserTradeUnmarshal(t *testing.T) {
	t.Parallel()
	// Every field the user_trade schema documents, as the documented examples carry few of them
	const data = `{"trade_id":"USDC-7","trade_seq":42,"instrument_name":"BTC_USDC-PERPETUAL","timestamp":1792368000000,
		"starbase_timestamp":1792368000000123456,"starbase_order_id":103148386170,"starbase_client_order_id":"client-1",
		"starbase_match_id":55,"order_type":"limit","original_order_type":"market","advanced":"usd","order_id":"USDC-9",
		"direction":"buy","tick_direction":2,"index_price":82000.5,"price":82001,"mark_price":82000.75,"amount":0.001,
		"contracts":10,"iv":45.5,"underlying_price":82002,"liquidation":"MT","liquidity":"M","fee":0.0025,
		"fee_currency":"USDC","profit_loss":1.25,"label":"gct","state":"filled","block_trade_id":"154",
		"block_trade_leg_count":3,"block_rfq_id":12,"block_rfq_quote_id":34,"reduce_only":"true","post_only":false,
		"mmp":true,"risk_reducing":true,"api":true,"legs":[{"trade_id":"USDC-8","direction":"sell"}],
		"combo_id":"BTC-FS-27NOV26_PERP","combo_trade_id":"USDC-6","quote_set_id":"set","quote_id":"quote",
		"trade_allocations":[{"user_id":5,"amount":0.0005,"fee":0.001,"client_info":{"client_id":1,"client_link_id":2,"name":"desk"}}],
		"isolated":true,"main_uid":1000,"user_id":123456,"matching_id":null}`
	var trade DerivativesUserTradeResponse
	require.NoError(t, json.Unmarshal([]byte(data), &trade), "Unmarshal must not error")
	exp := DerivativesUserTradeResponse{
		TradeID:               "USDC-7",
		TradeSequence:         42,
		InstrumentName:        "BTC_USDC-PERPETUAL",
		Timestamp:             types.Time(time.UnixMilli(1792368000000)),
		StarbaseTimestamp:     types.Time(time.Unix(0, 1792368000000123456)),
		StarbaseOrderID:       103148386170,
		StarbaseClientOrderID: "client-1",
		StarbaseMatchID:       55,
		OrderType:             "limit",
		OriginalOrderType:     "market",
		Advanced:              "usd",
		OrderID:               "USDC-9",
		Direction:             "buy",
		TickDirection:         2,
		IndexPrice:            82000.5,
		Price:                 82001,
		MarkPrice:             82000.75,
		Amount:                0.001,
		Contracts:             10,
		ImpliedVolatility:     45.5,
		UnderlyingPrice:       82002,
		Liquidation:           "MT",
		Liquidity:             "M",
		Fee:                   0.0025,
		FeeCurrency:           currency.USDC,
		ProfitLoss:            1.25,
		Label:                 "gct",
		State:                 "filled",
		BlockTradeID:          "154",
		BlockTradeLegCount:    3,
		BlockRFQID:            12,
		BlockRFQQuoteID:       34,
		ReduceOnly:            true,
		MMP:                   true,
		RiskReducing:          true,
		API:                   true,
		Legs:                  []DerivativesUserTradeResponse{{TradeID: "USDC-8", Direction: "sell"}},
		ComboID:               "BTC-FS-27NOV26_PERP",
		ComboTradeID:          "USDC-6",
		QuoteSetID:            "set",
		QuoteID:               "quote",
		TradeAllocations: []DerivativesTradeAllocation{
			{UserID: 5, Amount: 0.0005, Fee: 0.001, ClientInfo: &DerivativesTradeAllocationClientInfo{ClientID: 1, ClientLinkID: 2, Name: "desk"}},
		},
		Isolated:   true,
		MainUserID: 1000,
		UserID:     123456,
	}
	assert.Equal(t, exp, trade, "DerivativesUserTrade should decode every documented field")
}

func TestDerivativesOrderRequestValidate(t *testing.T) {
	t.Parallel()
	valid := func() *DerivativesOrderRequest {
		return &DerivativesOrderRequest{InstrumentName: "SOL_USDC-PERPETUAL", Amount: 10}
	}
	for _, tc := range []struct {
		name   string
		modify func(*DerivativesOrderRequest)
		err    error
	}{
		{name: "valid order", modify: func(*DerivativesOrderRequest) {}},
		{name: "empty instrument name", modify: func(o *DerivativesOrderRequest) { o.InstrumentName = "" }, err: errInstrumentNameRequired},
		{name: "no size", modify: func(o *DerivativesOrderRequest) { o.Amount = 0 }, err: errInvalidAmount},
		{name: "contracts alone", modify: func(o *DerivativesOrderRequest) { o.Amount, o.Contracts = 0, 1 }},
		{name: "undocumented type", modify: func(o *DerivativesOrderRequest) { o.Type = "stop" }, err: errDerivativesInvalidOrderType},
		{name: "stop without trigger", modify: func(o *DerivativesOrderRequest) { o.Type, o.TriggerPrice = DerivativesOrderTypeStopMarket, 1 }, err: errDerivativesTriggerRequired},
		{name: "undocumented time in force", modify: func(o *DerivativesOrderRequest) { o.TimeInForce = "good_til_date" }, err: errDerivativesInvalidTimeInForce},
		{name: "undocumented trigger", modify: func(o *DerivativesOrderRequest) { o.Trigger = "spot_price" }, err: errDerivativesInvalidTrigger},
		{name: "undocumented advanced type", modify: func(o *DerivativesOrderRequest) { o.Advanced = "usdc" }, err: errDerivativesInvalidAdvancedType},
		{name: "post only by default", modify: func(o *DerivativesOrderRequest) { o.PostOnly = true }},
		{name: "post only good til cancelled", modify: func(o *DerivativesOrderRequest) {
			o.PostOnly, o.TimeInForce = true, DerivativesTimeInForceGoodTilCancelled
		}},
		{name: "post only immediate or cancel", modify: func(o *DerivativesOrderRequest) {
			o.PostOnly, o.TimeInForce = true, DerivativesTimeInForceImmediateOrCancel
		}, err: errDerivativesPostOnlyTimeInForce},
		{name: "post only good til day", modify: func(o *DerivativesOrderRequest) { o.PostOnly, o.TimeInForce = true, DerivativesTimeInForceGoodTilDay }, err: errDerivativesPostOnlyTimeInForce},
		{name: "reject post only without post only", modify: func(o *DerivativesOrderRequest) { o.RejectPostOnly = true }, err: errDerivativesRejectPostOnlyWithoutPostOnly},
		{name: "reject post only with post only", modify: func(o *DerivativesOrderRequest) { o.PostOnly, o.RejectPostOnly = true, true }},
		{
			name: "secondary orders without a linked order type",
			modify: func(o *DerivativesOrderRequest) {
				o.OTOCOConfig = []DerivativesSecondaryOrder{{Amount: 10, Direction: DerivativesDirectionSell}}
			},
			err: errDerivativesLinkedOrderTypeRequired,
		},
		{name: "negative allocated margin", modify: func(o *DerivativesOrderRequest) { o.Isolated, o.AllocatedMargin = true, -1 }, err: errInvalidAmount},
		{name: "allocated margin without isolated margin", modify: func(o *DerivativesOrderRequest) { o.AllocatedMargin = 5 }, err: errDerivativesAllocatedMarginWithoutIsolated},
		{name: "isolated with allocated margin", modify: func(o *DerivativesOrderRequest) { o.Isolated, o.AllocatedMargin = true, 5 }},
		{name: "isolated market order", modify: func(o *DerivativesOrderRequest) { o.Isolated, o.Type = true, DerivativesOrderTypeMarket }},
		{
			name: "isolated trigger order",
			modify: func(o *DerivativesOrderRequest) {
				o.Isolated, o.Type, o.Trigger, o.TriggerPrice = true, DerivativesOrderTypeTakeLimit, DerivativesTriggerLastPrice, 90
			},
			err: errDerivativesIsolatedOrderUnsupported,
		},
		{name: "isolated trailing stop", modify: func(o *DerivativesOrderRequest) {
			o.Isolated, o.Type, o.Trigger = true, DerivativesOrderTypeTrailingStop, DerivativesTriggerMarkPrice
		}, err: errDerivativesIsolatedOrderUnsupported},
		{name: "isolated advanced order", modify: func(o *DerivativesOrderRequest) { o.Isolated, o.Advanced = true, DerivativesAdvancedUSD }, err: errDerivativesIsolatedOrderUnsupported},
		{
			name: "isolated linked order",
			modify: func(o *DerivativesOrderRequest) {
				o.Isolated, o.LinkedOrderType = true, DerivativesLinkedOrderTypeOneTriggersOther
				o.OTOCOConfig = []DerivativesSecondaryOrder{{Amount: 10, Direction: DerivativesDirectionSell}}
			},
			err: errDerivativesIsolatedOrderUnsupported,
		},
	} {
		req := valid()
		tc.modify(req)
		assert.ErrorIsf(t, req.validate(), tc.err, "validate should return the expected error for %s", tc.name)
	}
	assert.ErrorIs(t, (*DerivativesOrderRequest)(nil).validate(), common.ErrNilPointer, "validate should reject a nil request")
}

func TestDerivativesOrderRequestValidateLinkedOrder(t *testing.T) {
	t.Parallel()
	secondary := func(n int) []DerivativesSecondaryOrder {
		orders := make([]DerivativesSecondaryOrder, n)
		for i := range orders {
			orders[i] = DerivativesSecondaryOrder{Amount: 10, Direction: DerivativesDirectionSell}
		}
		return orders
	}
	for _, tc := range []struct {
		name                 string
		linkedOrderType      string
		triggerFillCondition string
		secondaryOrders      []DerivativesSecondaryOrder
		err                  error
	}{
		{name: "an order that is not linked"},
		{name: "secondary orders alone", secondaryOrders: secondary(1), err: errDerivativesLinkedOrderTypeRequired},
		{name: "a trigger fill condition alone", triggerFillCondition: DerivativesTriggerFillConditionFirstHit, err: errDerivativesLinkedOrderTypeRequired},
		{name: "an undocumented trigger fill condition", linkedOrderType: DerivativesLinkedOrderTypeOneTriggersOther, triggerFillCondition: "last_hit", secondaryOrders: secondary(1), err: errDerivativesInvalidTriggerFillCondition},
		{name: "an undocumented linked order type", linkedOrderType: "one_cancels_others", secondaryOrders: secondary(1), err: errDerivativesInvalidLinkedOrderType},
		{name: "one triggers other without secondary orders", linkedOrderType: DerivativesLinkedOrderTypeOneTriggersOther, err: errDerivativesInvalidSecondaryOrderCount},
		{name: "one triggers other with one", linkedOrderType: DerivativesLinkedOrderTypeOneTriggersOther, triggerFillCondition: DerivativesTriggerFillConditionCompleteFill, secondaryOrders: secondary(1)},
		{name: "one triggers other with three", linkedOrderType: DerivativesLinkedOrderTypeOneTriggersOther, triggerFillCondition: DerivativesTriggerFillConditionIncremental, secondaryOrders: secondary(3)},
		{name: "one cancels other with one", linkedOrderType: DerivativesLinkedOrderTypeOneCancelsOther, secondaryOrders: secondary(1)},
		{name: "one cancels other with two", linkedOrderType: DerivativesLinkedOrderTypeOneCancelsOther, secondaryOrders: secondary(2), err: errDerivativesInvalidSecondaryOrderCount},
		{name: "one triggers one cancels other with two", linkedOrderType: DerivativesLinkedOrderTypeOneTriggersOneCancelsOther, secondaryOrders: secondary(2)},
		{name: "one triggers one cancels other with one", linkedOrderType: DerivativesLinkedOrderTypeOneTriggersOneCancelsOther, secondaryOrders: secondary(1), err: errDerivativesInvalidSecondaryOrderCount},
		{name: "an invalid secondary order", linkedOrderType: DerivativesLinkedOrderTypeOneTriggersOther, secondaryOrders: []DerivativesSecondaryOrder{{Direction: DerivativesDirectionSell}}, err: errInvalidAmount},
	} {
		req := &DerivativesOrderRequest{LinkedOrderType: tc.linkedOrderType, TriggerFillCondition: tc.triggerFillCondition, OTOCOConfig: tc.secondaryOrders}
		assert.ErrorIsf(t, req.validateLinkedOrder(), tc.err, "validateLinkedOrder should return the expected error for %s", tc.name)
	}
}

func TestDerivativesSecondaryOrderValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		order DerivativesSecondaryOrder
		err   error
	}{
		{name: "valid order", order: DerivativesSecondaryOrder{Amount: 10, Direction: DerivativesDirectionSell, Type: DerivativesOrderTypeTakeLimit, Price: 2, TriggerPrice: 1, Trigger: DerivativesTriggerMarkPrice}},
		{name: "no amount", order: DerivativesSecondaryOrder{Direction: DerivativesDirectionSell}, err: errInvalidAmount},
		{name: "undocumented direction", order: DerivativesSecondaryOrder{Amount: 10, Direction: "long"}, err: errDerivativesInvalidOrderDirection},
		{name: "undocumented type", order: DerivativesSecondaryOrder{Amount: 10, Direction: DerivativesDirectionBuy, Type: "take"}, err: errDerivativesInvalidOrderType},
		{name: "take without trigger price", order: DerivativesSecondaryOrder{Amount: 10, Direction: DerivativesDirectionBuy, Type: DerivativesOrderTypeTakeMarket, Trigger: DerivativesTriggerLastPrice}, err: errDerivativesTriggerPriceRequired},
		{name: "undocumented time in force", order: DerivativesSecondaryOrder{Amount: 10, Direction: DerivativesDirectionBuy, TimeInForce: "good_til_date"}, err: errDerivativesInvalidTimeInForce},
		{name: "undocumented trigger", order: DerivativesSecondaryOrder{Amount: 10, Direction: DerivativesDirectionBuy, Trigger: "spot_price"}, err: errDerivativesInvalidTrigger},
	} {
		assert.ErrorIsf(t, tc.order.validate(), tc.err, "validate should return the expected error for %s", tc.name)
	}
}

func TestValidateDerivativesOrderEdit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                     string
		amount, contracts        float64
		advanced                 string
		postOnly, rejectPostOnly bool
		isolated                 bool
		allocatedMargin          float64
		err                      error
	}{
		{name: "valid edit", amount: 10},
		{name: "contracts alone", contracts: 1},
		{name: "no size", err: errInvalidAmount},
		{name: "undocumented advanced type", amount: 10, advanced: "USD", err: errDerivativesInvalidAdvancedType},
		{name: "advanced type", amount: 10, advanced: DerivativesAdvancedImpliedVolatility},
		{name: "reject post only without post only", amount: 10, rejectPostOnly: true, err: errDerivativesRejectPostOnlyWithoutPostOnly},
		{name: "reject post only with post only", amount: 10, postOnly: true, rejectPostOnly: true},
		{name: "negative allocated margin", amount: 10, isolated: true, allocatedMargin: -1, err: errInvalidAmount},
		{name: "allocated margin without isolated margin", amount: 10, allocatedMargin: 5, err: errDerivativesAllocatedMarginWithoutIsolated},
		{name: "isolated with allocated margin", amount: 10, isolated: true, allocatedMargin: 5},
	} {
		err := validateDerivativesOrderEdit(tc.amount, tc.contracts, tc.advanced, tc.postOnly, tc.rejectPostOnly, tc.isolated, tc.allocatedMargin)
		assert.ErrorIsf(t, err, tc.err, "validateDerivativesOrderEdit should return the expected error for %s", tc.name)
	}
}

func TestValidateDerivativesOrderSize(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		amount, contracts float64
		err               error
	}{
		{amount: 10},
		{contracts: 1},
		{amount: 10, contracts: 1},
		{err: errInvalidAmount},
		{amount: -10, err: errInvalidAmount},
		{amount: 10, contracts: -1, err: errInvalidAmount},
	} {
		assert.ErrorIsf(t, validateDerivativesOrderSize(tc.amount, tc.contracts), tc.err, "validateDerivativesOrderSize should return the expected error for amount %v and contracts %v", tc.amount, tc.contracts)
	}
}

func TestIsDerivativesTriggerOrderType(t *testing.T) {
	t.Parallel()
	for _, orderType := range []string{DerivativesOrderTypeStopLimit, DerivativesOrderTypeStopMarket, DerivativesOrderTypeTakeLimit, DerivativesOrderTypeTakeMarket, DerivativesOrderTypeTrailingStop} {
		assert.Truef(t, isDerivativesTriggerOrderType(orderType), "isDerivativesTriggerOrderType should report %s", orderType)
	}
	for _, orderType := range []string{"", DerivativesOrderTypeLimit, DerivativesOrderTypeMarket, DerivativesOrderTypeMarketLimit, "stop"} {
		assert.Falsef(t, isDerivativesTriggerOrderType(orderType), "isDerivativesTriggerOrderType should not report %q", orderType)
	}
}

func TestValidateDerivativesOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType, trigger string
		triggerPrice       float64
		err                error
	}{
		{},
		{orderType: DerivativesOrderTypeLimit},
		{orderType: DerivativesOrderTypeMarket},
		{orderType: DerivativesOrderTypeMarketLimit},
		{orderType: DerivativesOrderTypeStopLimit, trigger: DerivativesTriggerLastPrice, triggerPrice: 1},
		{orderType: DerivativesOrderTypeStopMarket, trigger: DerivativesTriggerIndexPrice, triggerPrice: 1},
		{orderType: DerivativesOrderTypeTakeLimit, trigger: DerivativesTriggerMarkPrice, triggerPrice: 1},
		{orderType: DerivativesOrderTypeTakeMarket, trigger: DerivativesTriggerMarkPrice, triggerPrice: -1},
		{orderType: DerivativesOrderTypeTrailingStop, trigger: DerivativesTriggerMarkPrice},
		{orderType: DerivativesOrderTypeStopLimit, triggerPrice: 1, err: errDerivativesTriggerRequired},
		{orderType: DerivativesOrderTypeTrailingStop, err: errDerivativesTriggerRequired},
		{orderType: DerivativesOrderTypeTakeMarket, trigger: DerivativesTriggerMarkPrice, err: errDerivativesTriggerPriceRequired},
		{orderType: "stop", err: errDerivativesInvalidOrderType},
		{orderType: "Limit", err: errDerivativesInvalidOrderType},
	} {
		assert.ErrorIsf(t, validateDerivativesOrderType(tc.orderType, tc.trigger, tc.triggerPrice), tc.err, "validateDerivativesOrderType should return the expected error for type %q with trigger %q", tc.orderType, tc.trigger)
	}
}

func TestValidateDerivativesTimeInForce(t *testing.T) {
	t.Parallel()
	for _, timeInForce := range []string{"", DerivativesTimeInForceGoodTilCancelled, DerivativesTimeInForceGoodTilDay, DerivativesTimeInForceFillOrKill, DerivativesTimeInForceImmediateOrCancel} {
		assert.NoErrorf(t, validateDerivativesTimeInForce(timeInForce), "validateDerivativesTimeInForce should accept %q", timeInForce)
	}
	assert.ErrorIs(t, validateDerivativesTimeInForce("good_til_time"), errDerivativesInvalidTimeInForce, "validateDerivativesTimeInForce should refuse a good til time, which the gateway does not have")
}

func TestValidateDerivativesTrigger(t *testing.T) {
	t.Parallel()
	for _, trigger := range []string{"", DerivativesTriggerIndexPrice, DerivativesTriggerMarkPrice, DerivativesTriggerLastPrice} {
		assert.NoErrorf(t, validateDerivativesTrigger(trigger), "validateDerivativesTrigger should accept %q", trigger)
	}
	assert.ErrorIs(t, validateDerivativesTrigger("index"), errDerivativesInvalidTrigger, "validateDerivativesTrigger should refuse an undocumented trigger")
}

func TestValidateDerivativesAdvancedType(t *testing.T) {
	t.Parallel()
	for _, advanced := range []string{"", DerivativesAdvancedUSD, DerivativesAdvancedImpliedVolatility} {
		assert.NoErrorf(t, validateDerivativesAdvancedType(advanced), "validateDerivativesAdvancedType should accept %q", advanced)
	}
	assert.ErrorIs(t, validateDerivativesAdvancedType("USD"), errDerivativesInvalidAdvancedType, "validateDerivativesAdvancedType should refuse an undocumented advanced order type")
}

func TestValidateDerivativesTradingKind(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"", DerivativesKindFuture, DerivativesKindOption, DerivativesKindSpot, DerivativesKindFutureCombo, DerivativesKindOptionCombo} {
		assert.NoErrorf(t, validateDerivativesTradingKind(kind, false), "validateDerivativesTradingKind should accept %q without aggregates", kind)
		assert.NoErrorf(t, validateDerivativesTradingKind(kind, true), "validateDerivativesTradingKind should accept %q with aggregates", kind)
	}
	for _, kind := range []string{DerivativesKindCombo, DerivativesKindAny} {
		assert.ErrorIsf(t, validateDerivativesTradingKind(kind, false), errDerivativesInvalidTradingKind, "validateDerivativesTradingKind should refuse %q without aggregates", kind)
		assert.NoErrorf(t, validateDerivativesTradingKind(kind, true), "validateDerivativesTradingKind should accept %q with aggregates", kind)
	}
	assert.ErrorIs(t, validateDerivativesTradingKind("perpetual", true), errDerivativesInvalidTradingKind, "validateDerivativesTradingKind should refuse an undocumented kind")
}

func TestValidateDerivativesCancelOrderType(t *testing.T) {
	t.Parallel()
	for _, orderType := range []string{"", DerivativesCancelOrderTypeAll, DerivativesCancelOrderTypeLimit, DerivativesCancelOrderTypeTriggerAll, DerivativesCancelOrderTypeStop, DerivativesCancelOrderTypeTake, DerivativesCancelOrderTypeTrailingStop} {
		assert.NoErrorf(t, validateDerivativesCancelOrderType(orderType), "validateDerivativesCancelOrderType should accept %q", orderType)
	}
	assert.ErrorIs(t, validateDerivativesCancelOrderType(DerivativesOpenOrderTypeStopAll), errDerivativesInvalidCancelOrderType, "validateDerivativesCancelOrderType should refuse an order type only open orders methods take")
}

func TestValidateDerivativesOpenOrderType(t *testing.T) {
	t.Parallel()
	for _, orderType := range []string{
		"", DerivativesOpenOrderTypeAll, DerivativesOpenOrderTypeLimit, DerivativesOpenOrderTypeTriggerAll, DerivativesOpenOrderTypeStopAll,
		DerivativesOpenOrderTypeStopLimit, DerivativesOpenOrderTypeStopMarket, DerivativesOpenOrderTypeTakeAll, DerivativesOpenOrderTypeTakeLimit,
		DerivativesOpenOrderTypeTakeMarket, DerivativesOpenOrderTypeTrailingAll, DerivativesOpenOrderTypeTrailingStop,
	} {
		assert.NoErrorf(t, validateDerivativesOpenOrderType(orderType), "validateDerivativesOpenOrderType should accept %q", orderType)
	}
	assert.ErrorIs(t, validateDerivativesOpenOrderType(DerivativesCancelOrderTypeTake), errDerivativesInvalidOpenOrderType, "validateDerivativesOpenOrderType should refuse an order type only cancel methods take")
}

func TestValidateDerivativesSettlementHistoryType(t *testing.T) {
	t.Parallel()
	for _, settlementType := range []string{"", "settlement", "delivery", "bankruptcy"} {
		assert.NoErrorf(t, validateDerivativesSettlementHistoryType(settlementType), "validateDerivativesSettlementHistoryType should accept %q", settlementType)
	}
	assert.ErrorIs(t, validateDerivativesSettlementHistoryType("exercise"), errDerivativesInvalidSettlementHistoryType, "validateDerivativesSettlementHistoryType should refuse an undocumented settlement type")
}

func TestValidateDerivativesUserTradesSorting(t *testing.T) {
	t.Parallel()
	for _, sorting := range []string{"", DerivativesSortingAscending, DerivativesSortingDescending, DerivativesSortingDefault} {
		assert.NoErrorf(t, validateDerivativesUserTradesSorting(sorting), "validateDerivativesUserTradesSorting should accept %q", sorting)
	}
	assert.ErrorIs(t, validateDerivativesUserTradesSorting("ascending"), errDerivativesInvalidUserTradesSorting, "validateDerivativesUserTradesSorting should refuse an undocumented sorting")
}

func TestValidateDerivativesTradingCount(t *testing.T) {
	t.Parallel()
	for _, count := range []uint64{0, 1, derivativesMaximumTradingCount} {
		assert.NoErrorf(t, validateDerivativesTradingCount(count), "validateDerivativesTradingCount should accept %d", count)
	}
	assert.ErrorIs(t, validateDerivativesTradingCount(derivativesMaximumTradingCount+1), errDerivativesInvalidTradingCount, "validateDerivativesTradingCount should refuse a count over 1000")
}

func TestDerivativesKindOrTypeCurrencies(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		codes []currency.Code
		exp   any
	}{
		{codes: []currency.Code{currency.NewCode("any")}, exp: "any"},
		{codes: []currency.Code{currency.NewCode("ANY")}, exp: "any"},
		{codes: []currency.Code{currency.NewCode("btc")}, exp: "BTC"},
		{codes: []currency.Code{currency.NewCode("usdc"), currency.ETH}, exp: []string{"USDC", "ETH"}},
	} {
		currencies, err := derivativesKindOrTypeCurrencies(tc.codes)
		require.NoErrorf(t, err, "derivativesKindOrTypeCurrencies must not error for %s", tc.codes)
		assert.Equalf(t, tc.exp, currencies, "derivativesKindOrTypeCurrencies should return %s as the gateway takes them", tc.codes)
	}
	_, err := derivativesKindOrTypeCurrencies(nil)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "derivativesKindOrTypeCurrencies should refuse no currencies")
	_, err = derivativesKindOrTypeCurrencies([]currency.Code{currency.BTC, currency.EMPTYCODE})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "derivativesKindOrTypeCurrencies should refuse an empty currency in the list")
}

func TestDerivativesUserTradesTimeRange(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1792368000000), time.UnixMilli(1792371600000)
	for _, tc := range []struct {
		name             string
		start, end       time.Time
		expStart, expEnd uint64
		err              error
	}{
		{name: "no range"},
		{name: "start alone", start: start, expStart: 1792368000000},
		{name: "end alone", end: end, expEnd: 1792371600000},
		{name: "range", start: start, end: end, expStart: 1792368000000, expEnd: 1792371600000},
		{name: "same start and end", start: end, end: end, err: common.ErrStartEqualsEnd},
		{name: "start after end", start: end, end: start, err: common.ErrStartAfterEnd},
		{name: "start before 1970", start: time.Date(1969, 12, 31, 0, 0, 0, 0, time.UTC), end: end, err: errDerivativesTimeBeforeEpoch},
	} {
		startTimestamp, endTimestamp, err := derivativesUserTradesTimeRange(tc.start, tc.end)
		require.ErrorIsf(t, err, tc.err, "derivativesUserTradesTimeRange must return the expected error for %s", tc.name)
		assert.Equalf(t, tc.expStart, startTimestamp, "derivativesUserTradesTimeRange should return the start in Unix milliseconds for %s", tc.name)
		assert.Equalf(t, tc.expEnd, endTimestamp, "derivativesUserTradesTimeRange should return the end in Unix milliseconds for %s", tc.name)
	}
}
