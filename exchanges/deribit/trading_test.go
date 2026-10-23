package deribit

import (
	"context"
	"io"
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
	// liveTradingInstrument is the instrument live trading tests place orders in and query
	liveTradingInstrument = "BTC-PERPETUAL"
	// liveTradingOrderAmount is the smallest BTC-PERPETUAL order, in USD
	liveTradingOrderAmount = 10
	// liveTradingIndexName is the index live MMP tests configure and query
	liveTradingIndexName = "btc_usd"

	// The reasons live tests skip methods no live test can safely or possibly exercise
	liveTradingSkipCancelAll  = "it would cancel every open order in its scope, not only the test's"
	liveTradingSkipMassQuotes = "mass quotes need Deribit's approval of the account and an MMP group"

	// mockCancelOrderID, mockOrderStateID and mockUserTradesOrderID are the orders of Deribit's documented examples
	mockCancelOrderID     = "ETH-SLIS-12"
	mockOrderStateID      = "ETH-331562"
	mockUserTradesOrderID = "ETH-584830574"
)

// mockOrderMarginIDs are the orders of Deribit's documented private/get_order_margin_by_ids example
var mockOrderMarginIDs = []string{"ETH-349280", "ETH-349279", "ETH-349278"}

// The mock requests are Deribit's documented examples, whose documented responses testdata/http.json records

// mockBuyRequest is the market buy order of Deribit's documented example
func mockBuyRequest() *OrderRequest {
	return &OrderRequest{InstrumentName: "ETH-PERPETUAL", Amount: 40, Type: "market", Label: "market0000234"}
}

// mockSellRequest is the stop limit sell order of Deribit's documented example
func mockSellRequest() *OrderRequest {
	return &OrderRequest{InstrumentName: "ETH-PERPETUAL", Amount: 123, Type: "stop_limit", Price: 145.61, TriggerPrice: 145, Trigger: "last_price"}
}

func mockCancelAllByCurrencyRequest() *CancelAllByCurrencyRequest {
	return &CancelAllByCurrencyRequest{Currency: currency.BTC, Kind: KindOption}
}

// mockCancelAllByCurrencyPairRequest uses the documented example's currency pair in the documented lower case form
func mockCancelAllByCurrencyPairRequest() *CancelAllByCurrencyPairRequest {
	return &CancelAllByCurrencyPairRequest{CurrencyPair: "btc_usd", Kind: KindOption}
}

func mockCancelAllByInstrumentRequest() *CancelAllByInstrumentRequest {
	return &CancelAllByInstrumentRequest{InstrumentName: "ETH-22FEB19-120-P", Type: "all"}
}

func mockCancelAllByKindOrTypeRequest() *CancelAllByKindOrTypeRequest {
	return &CancelAllByKindOrTypeRequest{Currencies: []currency.Code{currency.BTC, currency.ETH}, Kind: KindFuture}
}

func mockCancelByLabelRequest() *CancelByLabelRequest {
	return &CancelByLabelRequest{Label: "label"}
}

func mockCancelQuotesRequest() *CancelQuotesRequest {
	return &CancelQuotesRequest{CancelType: "delta", MinDelta: 0.4, MaxDelta: 0.6}
}

func mockClosePositionRequest() *ClosePositionRequest {
	return &ClosePositionRequest{InstrumentName: "ETH-PERPETUAL", Type: "limit", Price: 145.17}
}

func mockEditOrderRequest() *EditOrderRequest {
	return &EditOrderRequest{OrderID: "438994", Amount: 4, Price: 222, Advanced: "implv"}
}

func mockEditOrderByLabelRequest() *EditOrderByLabelRequest {
	return &EditOrderByLabelRequest{InstrumentName: "BTC-PERPETUAL", Label: "i_love_deribit", Amount: 150, Price: 50111}
}

func mockMarginsRequest() *MarginsRequest {
	return &MarginsRequest{InstrumentName: "BTC-PERPETUAL", Amount: 10000, Price: 3725}
}

// mockMMPGroupRequest selects the MMP group of Deribit's documented MMP configuration, status and reset examples
func mockMMPGroupRequest() *MMPGroupRequest {
	return &MMPGroupRequest{IndexName: "btc_usd", MMPGroup: "MassQuoteBot7"}
}

func mockOpenOrdersByCurrencyRequest() *OpenOrdersByCurrencyRequest {
	return &OpenOrdersByCurrencyRequest{Currency: currency.BTC}
}

func mockOpenOrdersByInstrumentRequest() *OpenOrdersByInstrumentRequest {
	return &OpenOrdersByInstrumentRequest{InstrumentName: "ETH-22FEB19-120-C"}
}

func mockOpenOrdersByLabelRequest() *OrdersByLabelRequest {
	return &OrdersByLabelRequest{Currency: currency.BTC, Label: "fooBar"}
}

func mockOrderHistoryByCurrencyRequest() *OrderHistoryByCurrencyRequest {
	return &OrderHistoryByCurrencyRequest{Currency: currency.BTC, Kind: KindFuture, Count: 1}
}

func mockOrderHistoryByInstrumentRequest() *OrderHistoryByInstrumentRequest {
	return &OrderHistoryByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", Count: 1}
}

func mockOrderStateByLabelRequest() *OrdersByLabelRequest {
	return &OrdersByLabelRequest{Currency: currency.ETH, Label: "fooBar"}
}

func mockSettlementHistoryByCurrencyRequest() *SettlementHistoryByCurrencyRequest {
	return &SettlementHistoryByCurrencyRequest{Currency: currency.BTC, Type: "delivery", Count: 1}
}

func mockSettlementHistoryByInstrumentRequest() *SettlementHistoryByInstrumentRequest {
	return &SettlementHistoryByInstrumentRequest{InstrumentName: "ETH-22FEB19", Type: "settlement", Count: 1}
}

func mockTriggerOrderHistoryRequest() *TriggerOrderHistoryRequest {
	return &TriggerOrderHistoryRequest{Currency: currency.ETH, Count: 10}
}

func mockUserTradesByCurrencyRequest() *UserTradesByCurrencyRequest {
	return &UserTradesByCurrencyRequest{Currency: currency.ETH, StartID: "ETH-34066", Count: 2}
}

// mockUserTradesByCurrencyAndTimeRequest swaps the documented example's times, whose start follows its end
func mockUserTradesByCurrencyAndTimeRequest() *UserTradesByCurrencyAndTimeRequest {
	return &UserTradesByCurrencyAndTimeRequest{Currency: currency.BTC, StartTime: time.UnixMilli(1510480630731), EndTime: time.UnixMilli(1590480630731), Count: 2}
}

func mockUserTradesByInstrumentRequest() *UserTradesByInstrumentRequest {
	return &UserTradesByInstrumentRequest{InstrumentName: "ETH-PERPETUAL", StartSequence: 1966042, Count: 2}
}

func mockUserTradesByInstrumentAndTimeRequest() *UserTradesByInstrumentAndTimeRequest {
	return &UserTradesByInstrumentAndTimeRequest{InstrumentName: "BTC-PERPETUAL", StartTime: time.UnixMilli(1590470872894), EndTime: time.UnixMilli(1590480872894), Count: 2}
}

func mockMassQuoteRequest() *MassQuoteRequest {
	return &MassQuoteRequest{
		Detailed: true,
		QuoteID:  "1",
		MMPGroup: "default",
		Quotes: []MassQuoteEntry{
			{InstrumentName: "BTC-PERPETUAL", QuoteSetID: "futures", Ask: &MassQuoteSide{Price: new(43800.0), Amount: new(10.0)}, Bid: &MassQuoteSide{Price: new(43700.0), Amount: new(10.0)}},
			{InstrumentName: "BTC-22DEC23-41600-C", QuoteSetID: "options", Ask: &MassQuoteSide{Price: new(0.05), Amount: new(1.0)}, Bid: &MassQuoteSide{Price: new(0.04), Amount: new(1.0)}},
		},
	}
}

// mockMovePositionsRequest sends the documented example's prices and amounts as the numbers Deribit documents, where
// the example sends strings
func mockMovePositionsRequest() *MovePositionsRequest {
	return &MovePositionsRequest{
		Currency:  currency.BTC,
		SourceUID: 3,
		TargetUID: 23,
		Trades:    []PositionMove{{InstrumentName: "BTC-PERPETUAL", Price: 35800, Amount: 110}, {InstrumentName: "BTC-28JAN22-32500-C", Amount: 0.1}},
	}
}

func mockSetMMPConfigRequest() *SetMMPConfigRequest {
	return &SetMMPConfigRequest{IndexName: "btc_usd", MMPGroup: "MassQuoteBot7", Interval: time.Minute, QuantityLimit: 3, MaxQuoteQuantity: 2.5}
}

// tradingTime returns a Unix millisecond timestamp as Deribit's responses decode it
func tradingTime(milliseconds int64) types.Time {
	return types.Time(time.UnixMilli(milliseconds))
}

// testPlacedOrderResponse is the documented buy, sell and close position response the mock requests return
func testPlacedOrderResponse() *OrderWithTradesResponse {
	return &OrderWithTradesResponse{
		Order: OrderResponse{
			OrderID:             "ETH-584864807",
			OrderState:          "filled",
			OrderType:           "limit",
			TimeInForce:         "good_til_cancelled",
			InstrumentName:      "ETH-PERPETUAL",
			CreationTimestamp:   tradingTime(1590486335742),
			LastUpdateTimestamp: tradingTime(1590486335742),
			Direction:           DirectionSell,
			Price:               OrderPrice{Value: 198.75},
			ReduceOnly:          true,
			API:                 true,
			Amount:              21,
			FilledAmount:        21,
			AveragePrice:        202.8,
		},
		Trades: []UserTradeResponse{{
			TradeID:        "ETH-2696097",
			TradeSequence:  1966068,
			InstrumentName: "ETH-PERPETUAL",
			Timestamp:      tradingTime(1590486335742),
			OrderType:      "limit",
			OrderID:        "ETH-584864807",
			Direction:      DirectionSell,
			IndexPrice:     202.86,
			Price:          202.8,
			Amount:         21,
			Liquidity:      "T",
			Fee:            7.766e-05,
			FeeCurrency:    currency.ETH,
			State:          "filled",
			ReduceOnly:     true,
			MarkPrice:      202.79,
		}},
	}
}

// testEditedOrderResponse is the documented edit and edit by label response the mock requests return
func testEditedOrderResponse() *OrderWithTradesResponse {
	return &OrderWithTradesResponse{
		Order: OrderResponse{
			OrderID:             "94166",
			OrderState:          "open",
			OrderType:           "limit",
			TimeInForce:         "good_til_cancelled",
			InstrumentName:      "BTC-PERPETUAL",
			CreationTimestamp:   tradingTime(1616155547764),
			LastUpdateTimestamp: tradingTime(1616155550773),
			Direction:           DirectionBuy,
			Price:               OrderPrice{Value: 50111},
			Label:               "i_love_deribit",
			API:                 true,
			Amount:              150,
			Replaced:            true,
		},
		Trades: []UserTradeResponse{},
	}
}

// testCancelledOrder is the documented cancel response the mock request returns, an untriggered stop market order
func testCancelledOrder() *OrderResponse {
	return &OrderResponse{
		OrderID:             mockCancelOrderID,
		OrderState:          "untriggered",
		OrderType:           "stop_market",
		TimeInForce:         "good_til_cancelled",
		InstrumentName:      "ETH-PERPETUAL",
		CreationTimestamp:   tradingTime(1550575961291),
		LastUpdateTimestamp: tradingTime(1550575961291),
		Direction:           DirectionSell,
		Price:               OrderPrice{Market: true},
		Amount:              5,
		Trigger:             "index_price",
		TriggerPrice:        144.73,
	}
}

// testDetailedCancelAllResponse is the detailed cancel all response the mock request returns, recorded from Deribit by
// GoCryptoTrader's earlier implementation, as Deribit documents only the count
func testDetailedCancelAllResponse() *CancelAllResponse {
	return &CancelAllResponse{
		CancelledCount: 1,
		ExecutionReports: []CancelExecutionReport{{
			InstrumentName: "BTC_USDC",
			Currency:       "BTC_USDC",
			Type:           "limit",
			Result: []OrderResponse{{
				OrderID:             "BTC_USDC-13133482",
				OrderState:          "cancelled",
				OrderType:           "limit",
				TimeInForce:         "good_til_cancelled",
				InstrumentName:      "BTC_USDC",
				CreationTimestamp:   tradingTime(1715302998260),
				LastUpdateTimestamp: tradingTime(1715303041949),
				Direction:           DirectionBuy,
				Price:               OrderPrice{Value: 30},
				Label:               "test",
				API:                 true,
				Amount:              30,
				Contracts:           300000,
				CancelReason:        "user_request",
			}},
		}},
	}
}

// testOpenOrders is the documented open orders response the mock requests return
func testOpenOrders() []OrderResponse {
	return []OrderResponse{{
		OrderID:             "146062",
		OrderState:          "open",
		OrderType:           "limit",
		TimeInForce:         "good_til_cancelled",
		InstrumentName:      "BTC-15FEB19-3250-P",
		CreationTimestamp:   tradingTime(1550050597036),
		LastUpdateTimestamp: tradingTime(1550050597036),
		Direction:           DirectionBuy,
		Price:               OrderPrice{Value: 0.0028},
		Label:               "fooBar",
		API:                 true,
		Amount:              10,
	}}
}

// testOrderHistoryResponse is the documented order history response the mock requests return
func testOrderHistoryResponse() *OrderHistoryResponse {
	return &OrderHistoryResponse{Orders: []OrderResponse{{
		OrderID:             "146475",
		OrderState:          "filled",
		OrderType:           "limit",
		TimeInForce:         "good_til_cancelled",
		InstrumentName:      "BTC-PERPETUAL",
		CreationTimestamp:   tradingTime(1550661808761),
		LastUpdateTimestamp: tradingTime(1550661808761),
		Direction:           DirectionBuy,
		Price:               OrderPrice{Value: 3886.5},
		API:                 true,
		Amount:              40,
		FilledAmount:        40,
		AveragePrice:        3659.8,
	}}}
}

// testOrderMargins is the documented order margins response the mock requests return
func testOrderMargins() []OrderMarginResponse {
	return []OrderMarginResponse{
		{OrderID: "ETH-349278", InitialMargin: 0.00091156, InitialMarginCurrency: currency.ETH},
		{OrderID: "ETH-349279", InitialMarginCurrency: currency.ETH},
		{OrderID: "ETH-349280", InitialMarginCurrency: currency.ETH},
	}
}

// testOrderState is the documented order state response the mock requests return, with the label the by label
// response carries
func testOrderState(label string) *OrderResponse {
	return &OrderResponse{
		OrderID:             mockOrderStateID,
		OrderState:          "filled",
		OrderType:           "limit",
		TimeInForce:         "good_til_cancelled",
		InstrumentName:      "ETH-PERPETUAL",
		CreationTimestamp:   tradingTime(1550219749176),
		LastUpdateTimestamp: tradingTime(1550219810944),
		Direction:           DirectionSell,
		Price:               OrderPrice{Value: 118.94},
		Label:               label,
		Amount:              37,
		FilledAmount:        37,
		AveragePrice:        118.94,
	}
}

// testSettlementHistoryResponse is the documented settlement history response the mock requests return
func testSettlementHistoryResponse() *SettlementsResponse {
	return &SettlementsResponse{
		Settlements: []Settlement{{
			Type:              "settlement",
			Timestamp:         tradingTime(1550475692526),
			InstrumentName:    "ETH-22FEB19",
			IndexPrice:        119.8,
			MarkPrice:         121.67,
			Position:          -66,
			ProfitLoss:        -0.001783937,
			SessionProfitLoss: 0.038358299,
		}},
		Continuation: "xY7T6cusbMBNpH9SNmKb94jXSBxUPojJEdCPL4YociHBUgAhWQvEP",
	}
}

// testTriggerOrderHistoryResponse is the documented trigger order history response the mock requests return; its
// example sends the trailing offset as offset, which the schema names trigger_offset
func testTriggerOrderHistoryResponse() *TriggerOrderHistoryResponse {
	return &TriggerOrderHistoryResponse{
		Entries: []TriggerOrderHistoryEntry{{
			Timestamp:      tradingTime(1555918941451),
			Trigger:        "index",
			TriggerPrice:   5285,
			TriggerOrderID: "SLIS-103",
			OrderID:        "671473",
			OrderState:     "new",
			InstrumentName: "BTC-PERPETUAL",
			Request:        "trigger:order",
			Direction:      DirectionBuy,
			Price:          OrderPrice{Value: 5179.28},
			Amount:         10,
		}},
		Continuation: "1555918941451.SLIS-103",
	}
}

// testUserTradesResponse is the documented user trades response the mock requests return
func testUserTradesResponse() *UserTradesResponse {
	return &UserTradesResponse{Trades: []UserTradeResponse{
		{
			TradeID:           "48078936",
			TradeSequence:     1,
			InstrumentName:    "BTC-27MAY20-8750-C",
			Timestamp:         tradingTime(1590480620145),
			OrderType:         "limit",
			OrderID:           "4008699030",
			Direction:         DirectionSell,
			TickDirection:     1,
			IndexPrice:        8993.47,
			Price:             0.028,
			Amount:            1,
			ImpliedVolatility: 38.51,
			UnderlyingPrice:   8994.95,
			Liquidity:         "M",
			Fee:               0.0004,
			FeeCurrency:       currency.BTC,
			State:             "filled",
			MarkPrice:         0.03135383,
		},
		{
			TradeID:        "47958936",
			TradeSequence:  299513,
			InstrumentName: "BTC-26JUN20",
			Timestamp:      tradingTime(1589923311862),
			OrderType:      "limit",
			OrderID:        "3993343822",
			Direction:      DirectionBuy,
			TickDirection:  2,
			IndexPrice:     9679.48,
			Price:          9681.5,
			Amount:         10,
			Liquidity:      "M",
			Fee:            -2.1e-07,
			FeeCurrency:    currency.BTC,
			State:          "filled",
			MarkPrice:      9684,
		},
	}}
}

// testUserTradesByOrder is the documented order trades response the mock requests return
func testUserTradesByOrder() []UserTradeResponse {
	return []UserTradeResponse{{
		TradeID:        "ETH-2696068",
		TradeSequence:  1966042,
		InstrumentName: "ETH-PERPETUAL",
		Timestamp:      tradingTime(1590480712800),
		OrderType:      "market",
		OrderID:        mockUserTradesOrderID,
		Direction:      DirectionBuy,
		TickDirection:  3,
		IndexPrice:     203.89,
		Price:          203.8,
		Amount:         100,
		Liquidity:      "T",
		Fee:            0.00036801,
		FeeCurrency:    currency.ETH,
		State:          "filled",
		MarkPrice:      203.78,
	}}
}

// testMassQuoteResponse is the documented detailed mass quote response the mock requests return
func testMassQuoteResponse() *MassQuoteResponse {
	quoteOrder := func(orderID, instrument, quoteSetID, direction string, price, amount float64) OrderResponse {
		return OrderResponse{
			OrderID:             orderID,
			OrderState:          "open",
			OrderType:           "limit",
			TimeInForce:         "good_til_cancelled",
			InstrumentName:      instrument,
			CreationTimestamp:   tradingTime(1703162478689),
			LastUpdateTimestamp: tradingTime(1703162550180),
			Direction:           direction,
			Price:               OrderPrice{Value: price},
			Amount:              amount,
			MMP:                 true,
			Quote:               true,
			MMPGroup:            "default",
			QuoteSetID:          quoteSetID,
			QuoteID:             "1",
		}
	}
	return &MassQuoteResponse{
		Orders: []OrderResponse{
			quoteOrder("6653852", "BTC-22DEC23-41600-C", "options", DirectionBuy, 0.04, 1),
			quoteOrder("6653853", "BTC-22DEC23-41600-C", "options", DirectionSell, 0.05, 1),
			quoteOrder("6653855", "BTC-PERPETUAL", "futures", DirectionSell, 43800, 10),
		},
		Trades: []UserTradeResponse{},
		Errors: []MassQuoteError{{
			InstrumentName: "BTC-PERPETUAL",
			Side:           "bid",
			Error:          &APIError{Code: 10007, Message: "price_too_high 43666.4288"},
		}},
	}
}

// testPositionMoveTrades is the documented position move response the mock requests return
func testPositionMoveTrades() []PositionMoveTradeResponse {
	return []PositionMoveTradeResponse{
		{SourceUID: 3, TargetUID: 23, InstrumentName: "BTC-28JAN22-32500-C", Direction: DirectionSell, Price: 0.1223, Amount: 0.1},
		{SourceUID: 3, TargetUID: 23, InstrumentName: "BTC-PERPETUAL", Direction: DirectionBuy, Price: 35800, Amount: 110},
	}
}

// testMarginsResponse is the documented margins response the mock requests return
func testMarginsResponse() *MarginsResponse {
	return &MarginsResponse{
		Buy:          0.0219949,
		BuyTakerFee:  8.5e-05,
		BuyMakerFee:  1.7e-05,
		SellTakerFee: 8.5e-05,
		SellMakerFee: 1.7e-05,
		MinPrice:     3684.8,
		MaxPrice:     3759.24,
	}
}

// testMMPConfigs is the documented MMP configuration response the mock requests return
func testMMPConfigs() []MMPConfigResponse {
	return []MMPConfigResponse{{
		IndexName:        "btc_usd",
		Interval:         60,
		MMPGroup:         "MassQuoteBot7",
		QuantityLimit:    0.5,
		DeltaLimit:       0.3,
		VegaLimit:        0.1,
		MaxQuoteQuantity: 0.4,
	}}
}

// testSetMMPConfigs is the documented set MMP configuration response the mock requests return, a single configuration
// where the schema documents a list
func testSetMMPConfigs() []MMPConfigResponse {
	return []MMPConfigResponse{{IndexName: "btc_usd", Interval: 60, MMPGroup: "MassQuoteBot7", QuantityLimit: 3, MaxQuoteQuantity: 2.5}}
}

// testMMPStatuses is the documented MMP status response the mock requests return
func testMMPStatuses() []MMPStatusResponse {
	return []MMPStatusResponse{{IndexName: "btc_usd", FrozenUntil: tradingTime(1744275841861), MMPGroup: "MassQuoteBot7"}}
}

// newTradingParamsCaptureExchange returns an exchange whose REST requests all receive one fixed response body, and a
// function returning the params of the last request, so tests can check what a method sends against Deribit's
// documentation rather than a recording of the method itself
func newTradingParamsCaptureExchange(t *testing.T, body string) (ex *Exchange, lastParams func() string) {
	t.Helper()
	var mu sync.Mutex
	var params json.RawMessage
	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if assert.NoError(t, err, "Reading the request body should not error") {
			var req websocketVenueRequest
			if assert.NoError(t, json.Unmarshal(raw, &req), "Decoding the request body should not error") {
				mu.Lock()
				params = req.Params
				mu.Unlock()
			}
		}
		_, err = w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	return ex, func() string {
		mu.Lock()
		defer mu.Unlock()
		return string(params)
	}
}

// liveTradingTicker contains the BTC-PERPETUAL prices live tests price their orders from
type liveTradingTicker struct {
	MarkPrice float64 `json:"mark_price"`
	MinPrice  float64 `json:"min_price"`
	MaxPrice  float64 `json:"max_price"`
}

// getLiveTradingTicker returns BTC-PERPETUAL's mark price and the prices Deribit clamps sell and buy orders to
func getLiveTradingTicker(t *testing.T) *liveTradingTicker {
	t.Helper()
	var ticker *liveTradingTicker
	err := e.SendHTTPRequest(t.Context(), nonMatchingEPL, "public/ticker", &InstrumentNameRequest{InstrumentName: liveTradingInstrument}, &ticker)
	require.NoError(t, err, "public/ticker must not error")
	require.NotNil(t, ticker, "public/ticker must return the ticker")
	return ticker
}

// newLiveTradingOrderRequest returns a post only limit order of the smallest BTC-PERPETUAL amount, a buy at half the
// lowest sell price Deribit accepts or a sell at double the highest buy price, so no market move fills it while the
// test runs; orders with its label are cancelled when the test ends
func newLiveTradingOrderRequest(t *testing.T, direction string) *OrderRequest {
	t.Helper()
	ticker := getLiveTradingTicker(t)
	price := math.Floor(ticker.MinPrice/20) * 10
	if direction == DirectionSell {
		price = math.Ceil(ticker.MaxPrice/5) * 10
	}
	label := "gct-" + t.Name() + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	t.Cleanup(func() {
		_, err := e.CancelByLabel(context.WithoutCancel(t.Context()), &CancelByLabelRequest{Label: label, Currency: currency.BTC})
		assert.NoError(t, err, "CancelByLabel should cancel the test's orders")
	})
	return &OrderRequest{InstrumentName: liveTradingInstrument, Amount: liveTradingOrderAmount, Type: "limit", Label: label, Price: price, PostOnly: true, RejectPostOnly: true}
}

// placeLiveTradingOrder places a live buy order no market move fills while the test runs, and returns the request and
// the order; it is cancelled when the test ends
func placeLiveTradingOrder(t *testing.T) (*OrderRequest, *OrderResponse) {
	t.Helper()
	arg := newLiveTradingOrderRequest(t, DirectionBuy)
	result, err := e.Buy(t.Context(), arg)
	require.NoError(t, err, "Buy must not error")
	require.Equal(t, "open", result.Order.OrderState, "Buy must leave the order open")
	return arg, &result.Order
}

// liveTradingMMPConfigMu serialises the live tests that set the account's MMP configuration, so each restores the
// configuration it found
var liveTradingMMPConfigMu sync.Mutex

// setLiveTradingMMPConfig sets a test MMP configuration on the orders MMP group of the btc_usd index with set, and
// restores the configuration it found, or removes the test's, when the test ends
func setLiveTradingMMPConfig(t *testing.T, set func(context.Context, *SetMMPConfigRequest) ([]MMPConfigResponse, error)) []MMPConfigResponse {
	t.Helper()
	liveTradingMMPConfigMu.Lock()
	configs, err := e.GetMMPConfig(t.Context(), &MMPGroupRequest{IndexName: liveTradingIndexName})
	if err != nil {
		liveTradingMMPConfigMu.Unlock()
		require.NoError(t, err, "GetMMPConfig must not error")
	}
	restore := &SetMMPConfigRequest{IndexName: liveTradingIndexName}
	for i := range configs {
		if configs[i].MMPGroup == "" && !configs[i].BlockRFQ && configs[i].Interval != 0 {
			restore = &SetMMPConfigRequest{
				IndexName:        liveTradingIndexName,
				Interval:         time.Duration(float64(configs[i].Interval) * float64(time.Second)),
				FrozenTime:       time.Duration(float64(configs[i].FrozenTime) * float64(time.Second)),
				QuantityLimit:    configs[i].QuantityLimit,
				DeltaLimit:       configs[i].DeltaLimit,
				VegaLimit:        configs[i].VegaLimit,
				MaxQuoteQuantity: configs[i].MaxQuoteQuantity,
			}
		}
	}
	t.Cleanup(func() {
		defer liveTradingMMPConfigMu.Unlock()
		_, err := e.SetMMPConfig(context.WithoutCancel(t.Context()), restore)
		assert.NoError(t, err, "SetMMPConfig should restore the account's MMP configuration")
	})
	result, err := set(t.Context(), &SetMMPConfigRequest{IndexName: liveTradingIndexName, Interval: time.Minute, FrozenTime: 10 * time.Second, QuantityLimit: 1, MaxQuoteQuantity: 0.1})
	require.NoError(t, err, "SetMMPConfig must not error")
	return result
}

func TestBuy(t *testing.T) {
	t.Parallel()
	_, err := e.Buy(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Buy must reject a nil request")
	_, err = e.Buy(t.Context(), &OrderRequest{Amount: 1})
	require.ErrorIs(t, err, errInstrumentNameRequired, "Buy must reject an empty instrument name")

	skipLiveUnlessCanManipulate(t)
	arg := mockBuyRequest()
	if !mockTests {
		arg = newLiveTradingOrderRequest(t, DirectionBuy)
	}
	result, err := e.Buy(t.Context(), arg)
	require.NoError(t, err, "Buy must not error")
	if mockTests {
		assert.Equal(t, testPlacedOrderResponse(), result, "Buy should decode every field")
	} else {
		assert.Equal(t, "open", result.Order.OrderState, "Buy should place a resting order")
	}

	ex, lastParams := newTradingParamsCaptureExchange(t, `{"jsonrpc":"2.0","result":{"order":{},"trades":[]}}`)
	_, err = ex.Buy(t.Context(), &OrderRequest{
		InstrumentName:       "BTC-PERPETUAL",
		Contracts:            2,
		Type:                 "trailing_stop",
		TriggerOffset:        50,
		Trigger:              "mark_price",
		ValidUntil:           time.UnixMilli(1700000000000),
		LinkedOrderType:      "one_triggers_one_cancels_other",
		TriggerFillCondition: "incremental",
		OTOCOConfig:          []SecondaryOrder{{Amount: 20, Direction: DirectionSell, Type: "take_limit", Price: 90000, TriggerPrice: 89000, Trigger: "last_price", ReduceOnly: true}},
	})
	require.NoError(t, err, "Buy must not error for a linked trailing stop")
	exp := `{"instrument_name":"BTC-PERPETUAL","contracts":2,"type":"trailing_stop","post_only":false,"trigger_offset":50,"trigger":"mark_price","valid_until":1700000000000,"linked_order_type":"one_triggers_one_cancels_other","trigger_fill_condition":"incremental","otoco_config":[{"amount":20,"direction":"sell","type":"take_limit","price":90000,"reduce_only":true,"trigger_price":89000,"trigger":"last_price"}]}`
	assert.JSONEq(t, exp, lastParams(), "Buy should send the documented params, always with post_only, which Deribit defaults to true")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).Buy(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "Buy should reject a null result")
	_, err = newUnavailableServerExchange(t).Buy(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "Buy should return the request failure")
}

func TestCancelOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOrderByID(t.Context(), "")
	require.ErrorIs(t, err, errOrderIDRequired, "CancelOrderByID must reject an empty order ID")

	skipLiveUnlessCanManipulate(t)
	orderID := mockCancelOrderID
	if !mockTests {
		_, order := placeLiveTradingOrder(t)
		orderID = order.OrderID
	}
	result, err := e.CancelOrderByID(t.Context(), orderID)
	require.NoError(t, err, "CancelOrderByID must not error")
	if mockTests {
		assert.Equal(t, testCancelledOrder(), result, "CancelOrderByID should decode every field")
	} else {
		assert.Equal(t, "cancelled", result.OrderState, "CancelOrderByID should cancel the order")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CancelOrderByID(t.Context(), orderID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelOrderByID should reject a null result")
	_, err = newUnavailableServerExchange(t).CancelOrderByID(t.Context(), orderID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelOrderByID should return the request failure")
}

func TestCancelAll(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAll(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAll must reject a nil request")
	_, err = e.CancelAll(t.Context(), &CancelAllRequest{Detailed: true, IncludeStarbaseOrders: true})
	require.ErrorIs(t, err, errStarbaseOrdersWithDetailed, "CancelAll must reject Starbase orders with a detailed response")

	if !mockTests {
		t.Skip("CancelAll is not tested live, as " + liveTradingSkipCancelAll)
	}
	result, err := e.CancelAll(t.Context(), &CancelAllRequest{})
	require.NoError(t, err, "CancelAll must not error")
	assert.Equal(t, &CancelAllResponse{CancelledCount: 4}, result, "CancelAll should decode the count")

	result, err = e.CancelAll(t.Context(), &CancelAllRequest{Detailed: true})
	require.NoError(t, err, "CancelAll must not error with a detailed response")
	assert.Equal(t, testDetailedCancelAllResponse(), result, "CancelAll should decode every field of the execution reports")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CancelAll(t.Context(), &CancelAllRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelAll should reject a null result")
	_, err = newUnavailableServerExchange(t).CancelAll(t.Context(), &CancelAllRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelAll should return the request failure")
}

func TestCancelAllByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllByCurrency must reject a nil request")
	_, err = e.CancelAllByCurrency(t.Context(), &CancelAllByCurrencyRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelAllByCurrency must reject an empty currency")
	_, err = e.CancelAllByCurrency(t.Context(), &CancelAllByCurrencyRequest{Currency: currency.BTC, Kind: "perpetual"})
	require.ErrorIs(t, err, errInvalidKind, "CancelAllByCurrency must reject an undocumented kind")
	_, err = e.CancelAllByCurrency(t.Context(), &CancelAllByCurrencyRequest{Currency: currency.BTC, Type: "market"})
	require.ErrorIs(t, err, errInvalidCancelOrderType, "CancelAllByCurrency must reject an undocumented order type")

	if !mockTests {
		t.Skip("CancelAllByCurrency is not tested live, as " + liveTradingSkipCancelAll)
	}
	result, err := e.CancelAllByCurrency(t.Context(), mockCancelAllByCurrencyRequest())
	require.NoError(t, err, "CancelAllByCurrency must not error")
	assert.Equal(t, &CancelAllResponse{CancelledCount: 4}, result, "CancelAllByCurrency should decode the count")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CancelAllByCurrency(t.Context(), mockCancelAllByCurrencyRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelAllByCurrency should reject a null result")
	_, err = newUnavailableServerExchange(t).CancelAllByCurrency(t.Context(), mockCancelAllByCurrencyRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelAllByCurrency should return the request failure")
}

func TestCancelAllByCurrencyPair(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllByCurrencyPair(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllByCurrencyPair must reject a nil request")
	_, err = e.CancelAllByCurrencyPair(t.Context(), &CancelAllByCurrencyPairRequest{})
	require.ErrorIs(t, err, errCancelCurrencyPairRequired, "CancelAllByCurrencyPair must reject an empty currency pair")
	_, err = e.CancelAllByCurrencyPair(t.Context(), &CancelAllByCurrencyPairRequest{CurrencyPair: "btc_usd", Kind: "perpetual"})
	require.ErrorIs(t, err, errInvalidKind, "CancelAllByCurrencyPair must reject an undocumented kind")
	_, err = e.CancelAllByCurrencyPair(t.Context(), &CancelAllByCurrencyPairRequest{CurrencyPair: "btc_usd", Type: "market"})
	require.ErrorIs(t, err, errInvalidCancelOrderType, "CancelAllByCurrencyPair must reject an undocumented order type")

	if !mockTests {
		t.Skip("CancelAllByCurrencyPair is not tested live, as " + liveTradingSkipCancelAll)
	}
	result, err := e.CancelAllByCurrencyPair(t.Context(), mockCancelAllByCurrencyPairRequest())
	require.NoError(t, err, "CancelAllByCurrencyPair must not error")
	assert.Equal(t, &CancelAllResponse{CancelledCount: 4}, result, "CancelAllByCurrencyPair should decode the count")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CancelAllByCurrencyPair(t.Context(), mockCancelAllByCurrencyPairRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelAllByCurrencyPair should reject a null result")
	_, err = newUnavailableServerExchange(t).CancelAllByCurrencyPair(t.Context(), mockCancelAllByCurrencyPairRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelAllByCurrencyPair should return the request failure")
}

func TestCancelAllByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllByInstrument(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllByInstrument must reject a nil request")
	_, err = e.CancelAllByInstrument(t.Context(), &CancelAllByInstrumentRequest{})
	require.ErrorIs(t, err, errInstrumentNameRequired, "CancelAllByInstrument must reject an empty instrument name")
	_, err = e.CancelAllByInstrument(t.Context(), &CancelAllByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", Type: "market"})
	require.ErrorIs(t, err, errInvalidCancelOrderType, "CancelAllByInstrument must reject an undocumented order type")

	if !mockTests {
		t.Skip("CancelAllByInstrument is not tested live, as " + liveTradingSkipCancelAll)
	}
	result, err := e.CancelAllByInstrument(t.Context(), mockCancelAllByInstrumentRequest())
	require.NoError(t, err, "CancelAllByInstrument must not error")
	assert.Equal(t, &CancelAllResponse{CancelledCount: 4}, result, "CancelAllByInstrument should decode the count")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CancelAllByInstrument(t.Context(), mockCancelAllByInstrumentRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelAllByInstrument should reject a null result")
	_, err = newUnavailableServerExchange(t).CancelAllByInstrument(t.Context(), mockCancelAllByInstrumentRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelAllByInstrument should return the request failure")
}

func TestCancelAllByKindOrType(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllByKindOrType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllByKindOrType must reject a nil request")
	_, err = e.CancelAllByKindOrType(t.Context(), &CancelAllByKindOrTypeRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelAllByKindOrType must reject an empty currency list")
	_, err = e.CancelAllByKindOrType(t.Context(), &CancelAllByKindOrTypeRequest{Currencies: []currency.Code{currency.BTC, currency.EMPTYCODE}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelAllByKindOrType must reject an empty currency in the list")
	_, err = e.CancelAllByKindOrType(t.Context(), &CancelAllByKindOrTypeRequest{Currencies: []currency.Code{currency.BTC}, Kind: "perpetual"})
	require.ErrorIs(t, err, errInvalidKind, "CancelAllByKindOrType must reject an undocumented kind")
	_, err = e.CancelAllByKindOrType(t.Context(), &CancelAllByKindOrTypeRequest{Currencies: []currency.Code{currency.BTC}, Type: "market"})
	require.ErrorIs(t, err, errInvalidCancelOrderType, "CancelAllByKindOrType must reject an undocumented order type")
	_, err = e.CancelAllByKindOrType(t.Context(), &CancelAllByKindOrTypeRequest{Currencies: []currency.Code{currency.BTC}, Detailed: true, IncludeStarbaseOrders: true})
	require.ErrorIs(t, err, errStarbaseOrdersWithDetailed, "CancelAllByKindOrType must reject Starbase orders with a detailed response")

	ex, lastParams := newTradingParamsCaptureExchange(t, `{"jsonrpc":"2.0","result":0}`)
	for _, tc := range []struct {
		currencies []currency.Code
		exp        string
	}{
		{currencies: []currency.Code{currency.NewCode("any")}, exp: `{"currency":"any"}`},
		{currencies: []currency.Code{currency.BTC}, exp: `{"currency":"BTC"}`},
		{currencies: []currency.Code{currency.BTC, currency.ETH}, exp: `{"currency":["BTC","ETH"]}`},
	} {
		_, err = ex.CancelAllByKindOrType(t.Context(), &CancelAllByKindOrTypeRequest{Currencies: tc.currencies})
		require.NoErrorf(t, err, "CancelAllByKindOrType must not error for currencies %s", tc.currencies)
		assert.JSONEqf(t, tc.exp, lastParams(), "CancelAllByKindOrType should send currencies %s as a single currency or a list", tc.currencies)
	}

	if !mockTests {
		t.Skip("CancelAllByKindOrType is not tested live, as " + liveTradingSkipCancelAll)
	}
	result, err := e.CancelAllByKindOrType(t.Context(), mockCancelAllByKindOrTypeRequest())
	require.NoError(t, err, "CancelAllByKindOrType must not error")
	assert.Equal(t, &CancelAllResponse{CancelledCount: 4}, result, "CancelAllByKindOrType should decode the count")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CancelAllByKindOrType(t.Context(), mockCancelAllByKindOrTypeRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelAllByKindOrType should reject a null result")
	_, err = newUnavailableServerExchange(t).CancelAllByKindOrType(t.Context(), mockCancelAllByKindOrTypeRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelAllByKindOrType should return the request failure")
}

func TestCancelByLabel(t *testing.T) {
	t.Parallel()
	_, err := e.CancelByLabel(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelByLabel must reject a nil request")
	_, err = e.CancelByLabel(t.Context(), &CancelByLabelRequest{})
	require.ErrorIs(t, err, errLabelRequired, "CancelByLabel must reject an empty label")

	skipLiveUnlessCanManipulate(t)
	arg := mockCancelByLabelRequest()
	if !mockTests {
		placed, _ := placeLiveTradingOrder(t)
		arg = &CancelByLabelRequest{Label: placed.Label, Currency: currency.BTC}
	}
	result, err := e.CancelByLabel(t.Context(), arg)
	require.NoError(t, err, "CancelByLabel must not error")
	if mockTests {
		assert.Equal(t, &CancelAllResponse{CancelledCount: 4}, result, "CancelByLabel should decode the count")
	} else {
		assert.Equal(t, &CancelAllResponse{CancelledCount: 1}, result, "CancelByLabel should cancel the order with the label")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CancelByLabel(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelByLabel should reject a null result")
	_, err = newUnavailableServerExchange(t).CancelByLabel(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelByLabel should return the request failure")
}

func TestCancelQuotes(t *testing.T) {
	t.Parallel()
	_, err := e.CancelQuotes(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelQuotes must reject a nil request")
	_, err = e.CancelQuotes(t.Context(), &CancelQuotesRequest{CancelType: "label"})
	require.ErrorIs(t, err, errInvalidCancelQuotesType, "CancelQuotes must reject an undocumented cancel type")

	ex, lastParams := newTradingParamsCaptureExchange(t, `{"jsonrpc":"2.0","result":0}`)
	for _, tc := range []struct {
		arg *CancelQuotesRequest
		exp string
	}{
		{arg: &CancelQuotesRequest{CancelType: "delta", MinDelta: -0.5}, exp: `{"cancel_type":"delta","min_delta":-0.5,"max_delta":0}`},
		{arg: &CancelQuotesRequest{CancelType: "currency", Currency: currency.BTC, MinDelta: 0.4}, exp: `{"cancel_type":"currency","currency":"BTC"}`},
	} {
		_, err = ex.CancelQuotes(t.Context(), tc.arg)
		require.NoErrorf(t, err, "CancelQuotes must not error for cancel type %s", tc.arg.CancelType)
		assert.JSONEqf(t, tc.exp, lastParams(), "CancelQuotes should send the delta bounds, zero ones included, for cancel type %s alone", tc.arg.CancelType)
	}

	if !mockTests {
		t.Skip("CancelQuotes is not tested live, as " + liveTradingSkipMassQuotes)
	}
	result, err := e.CancelQuotes(t.Context(), mockCancelQuotesRequest())
	require.NoError(t, err, "CancelQuotes must not error")
	assert.Equal(t, &CancelAllResponse{CancelledCount: 4}, result, "CancelQuotes should decode the count")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CancelQuotes(t.Context(), mockCancelQuotesRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelQuotes should reject a null result")
	_, err = newUnavailableServerExchange(t).CancelQuotes(t.Context(), mockCancelQuotesRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelQuotes should return the request failure")
}

func TestClosePosition(t *testing.T) {
	t.Parallel()
	_, err := e.ClosePosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ClosePosition must reject a nil request")
	_, err = e.ClosePosition(t.Context(), &ClosePositionRequest{Type: "market"})
	require.ErrorIs(t, err, errInstrumentNameRequired, "ClosePosition must reject an empty instrument name")
	_, err = e.ClosePosition(t.Context(), &ClosePositionRequest{InstrumentName: "BTC-PERPETUAL", Type: "stop_market"})
	require.ErrorIs(t, err, errInvalidOrderType, "ClosePosition must reject an order type other than limit or market")

	if !mockTests {
		t.Skip("ClosePosition needs an open position, which a live test would have to open at market")
	}
	result, err := e.ClosePosition(t.Context(), mockClosePositionRequest())
	require.NoError(t, err, "ClosePosition must not error")
	assert.Equal(t, testPlacedOrderResponse(), result, "ClosePosition should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).ClosePosition(t.Context(), mockClosePositionRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "ClosePosition should reject a null result")
	_, err = newUnavailableServerExchange(t).ClosePosition(t.Context(), mockClosePositionRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "ClosePosition should return the request failure")
}

func TestEditOrder(t *testing.T) {
	t.Parallel()
	_, err := e.EditOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "EditOrder must reject a nil request")
	_, err = e.EditOrder(t.Context(), &EditOrderRequest{Amount: 1})
	require.ErrorIs(t, err, errOrderIDRequired, "EditOrder must reject an empty order ID")
	_, err = e.EditOrder(t.Context(), &EditOrderRequest{OrderID: "94166"})
	require.ErrorIs(t, err, errInvalidAmount, "EditOrder must reject an edit without a size")
	_, err = e.EditOrder(t.Context(), &EditOrderRequest{OrderID: "94166", Amount: 1, Advanced: "usdc"})
	require.ErrorIs(t, err, errInvalidAdvancedOrderType, "EditOrder must reject an undocumented advanced order type")

	ex, lastParams := newTradingParamsCaptureExchange(t, `{"jsonrpc":"2.0","result":{"order":{},"trades":[]}}`)
	_, err = ex.EditOrder(t.Context(), &EditOrderRequest{OrderID: "94166", Contracts: 15, Price: 50000, TriggerOffset: 10, ValidUntil: time.UnixMilli(1700000000000), DisplayAmount: 5})
	require.NoError(t, err, "EditOrder must not error")
	exp := `{"order_id":"94166","contracts":15,"price":50000,"post_only":false,"trigger_offset":10,"valid_until":1700000000000,"display_amount":5}`
	assert.JSONEq(t, exp, lastParams(), "EditOrder should always send post_only, and leave out an unset MMP flag, which Deribit rejects changing")

	skipLiveUnlessCanManipulate(t)
	arg := mockEditOrderRequest()
	if !mockTests {
		placed, order := placeLiveTradingOrder(t)
		arg = &EditOrderRequest{OrderID: order.OrderID, Amount: placed.Amount, Price: placed.Price - 10, PostOnly: true, RejectPostOnly: true}
	}
	result, err := e.EditOrder(t.Context(), arg)
	require.NoError(t, err, "EditOrder must not error")
	if mockTests {
		assert.Equal(t, testEditedOrderResponse(), result, "EditOrder should decode every field")
	} else {
		assert.Equal(t, arg.Price, result.Order.Price.Value, "EditOrder should reprice the order")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).EditOrder(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "EditOrder should reject a null result")
	_, err = newUnavailableServerExchange(t).EditOrder(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "EditOrder should return the request failure")
}

func TestEditOrderByLabel(t *testing.T) {
	t.Parallel()
	_, err := e.EditOrderByLabel(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "EditOrderByLabel must reject a nil request")
	_, err = e.EditOrderByLabel(t.Context(), &EditOrderByLabelRequest{Amount: 1})
	require.ErrorIs(t, err, errInstrumentNameRequired, "EditOrderByLabel must reject an empty instrument name")
	_, err = e.EditOrderByLabel(t.Context(), &EditOrderByLabelRequest{InstrumentName: "BTC-PERPETUAL", Amount: -1})
	require.ErrorIs(t, err, errInvalidAmount, "EditOrderByLabel must reject a negative amount")
	_, err = e.EditOrderByLabel(t.Context(), &EditOrderByLabelRequest{InstrumentName: "BTC-PERPETUAL", Amount: 1, Advanced: "usdc"})
	require.ErrorIs(t, err, errInvalidAdvancedOrderType, "EditOrderByLabel must reject an undocumented advanced order type")

	ex, lastParams := newTradingParamsCaptureExchange(t, `{"jsonrpc":"2.0","result":{"order":{},"trades":[]}}`)
	_, err = ex.EditOrderByLabel(t.Context(), &EditOrderByLabelRequest{InstrumentName: "BTC-PERPETUAL", Label: "gct", Amount: 20, PostOnly: true, ValidUntil: time.UnixMilli(1700000000000)})
	require.NoError(t, err, "EditOrderByLabel must not error")
	assert.JSONEq(t, `{"instrument_name":"BTC-PERPETUAL","label":"gct","amount":20,"post_only":true,"valid_until":1700000000000}`, lastParams(), "EditOrderByLabel should send the valid until time in Unix milliseconds")

	skipLiveUnlessCanManipulate(t)
	arg := mockEditOrderByLabelRequest()
	if !mockTests {
		placed, _ := placeLiveTradingOrder(t)
		arg = &EditOrderByLabelRequest{InstrumentName: placed.InstrumentName, Label: placed.Label, Amount: placed.Amount, Price: placed.Price - 10, PostOnly: true, RejectPostOnly: true}
	}
	result, err := e.EditOrderByLabel(t.Context(), arg)
	require.NoError(t, err, "EditOrderByLabel must not error")
	if mockTests {
		assert.Equal(t, testEditedOrderResponse(), result, "EditOrderByLabel should decode every field")
	} else {
		assert.Equal(t, arg.Price, result.Order.Price.Value, "EditOrderByLabel should reprice the order")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).EditOrderByLabel(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "EditOrderByLabel should reject a null result")
	_, err = newUnavailableServerExchange(t).EditOrderByLabel(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "EditOrderByLabel should return the request failure")
}

func TestGetMargins(t *testing.T) {
	t.Parallel()
	_, err := e.GetMargins(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMargins must reject a nil request")
	_, err = e.GetMargins(t.Context(), &MarginsRequest{Amount: 1, Price: 1})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetMargins must reject an empty instrument name")
	_, err = e.GetMargins(t.Context(), &MarginsRequest{InstrumentName: "BTC-PERPETUAL", Price: 1})
	require.ErrorIs(t, err, errInvalidAmount, "GetMargins must reject an empty amount")
	_, err = e.GetMargins(t.Context(), &MarginsRequest{InstrumentName: "BTC_USDC", Amount: 1, Price: 1, AmountType: "notional"})
	require.ErrorIs(t, err, errInvalidMarginAmountType, "GetMargins must reject an undocumented amount type")

	skipLiveWithoutCredentials(t)
	arg := mockMarginsRequest()
	if !mockTests {
		arg = &MarginsRequest{InstrumentName: liveTradingInstrument, Amount: liveTradingOrderAmount, Price: getLiveTradingTicker(t).MarkPrice}
	}
	result, err := e.GetMargins(t.Context(), arg)
	require.NoError(t, err, "GetMargins must not error")
	if mockTests {
		assert.Equal(t, testMarginsResponse(), result, "GetMargins should decode every field")
	} else {
		assert.Positive(t, result.MaxPrice, "GetMargins should return the highest buy price")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetMargins(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetMargins should reject a null result")
	_, err = newUnavailableServerExchange(t).GetMargins(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMargins should return the request failure")
}

func TestGetMMPConfig(t *testing.T) {
	t.Parallel()
	_, err := e.GetMMPConfig(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMMPConfig must reject a nil request")
	_, err = e.GetMMPConfig(t.Context(), &MMPGroupRequest{MMPGroup: "MassQuoteBot7"})
	require.ErrorIs(t, err, errMMPIndexNameRequired, "GetMMPConfig must reject an MMP group without its index")
	limited, noDelay := newRateLimitedServerExchange(t, `{"jsonrpc":"2.0","result":[]}`)
	_, err = limited.GetMMPConfig(noDelay, mockMMPGroupRequest())
	assert.NoError(t, err, "GetMMPConfig should be limited as a non-matching engine request")

	skipLiveWithoutCredentials(t)
	arg := mockMMPGroupRequest()
	if !mockTests {
		arg = &MMPGroupRequest{}
	}
	result, err := e.GetMMPConfig(t.Context(), arg)
	require.NoError(t, err, "GetMMPConfig must not error")
	if mockTests {
		assert.Equal(t, testMMPConfigs(), result, "GetMMPConfig should decode every field")
	}

	_, err = newUnavailableServerExchange(t).GetMMPConfig(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMMPConfig should return the request failure")
}

func TestGetMMPStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetMMPStatus(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMMPStatus must reject a nil request")
	_, err = e.GetMMPStatus(t.Context(), &MMPGroupRequest{MMPGroup: "MassQuoteBot7"})
	require.ErrorIs(t, err, errMMPIndexNameRequired, "GetMMPStatus must reject an MMP group without its index")
	limited, noDelay := newRateLimitedServerExchange(t, `{"jsonrpc":"2.0","result":[]}`)
	_, err = limited.GetMMPStatus(noDelay, mockMMPGroupRequest())
	assert.NoError(t, err, "GetMMPStatus should be limited as a non-matching engine request")

	skipLiveWithoutCredentials(t)
	arg := mockMMPGroupRequest()
	if !mockTests {
		arg = &MMPGroupRequest{}
	}
	result, err := e.GetMMPStatus(t.Context(), arg)
	require.NoError(t, err, "GetMMPStatus must not error")
	if mockTests {
		assert.Equal(t, testMMPStatuses(), result, "GetMMPStatus should decode every field")
	}

	_, err = newUnavailableServerExchange(t).GetMMPStatus(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMMPStatus should return the request failure")
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenOrders must reject a nil request")
	_, err = e.GetOpenOrders(t.Context(), &OpenOrdersRequest{Kind: KindAny})
	require.ErrorIs(t, err, errInvalidKind, "GetOpenOrders must reject a kind it does not document")
	_, err = e.GetOpenOrders(t.Context(), &OpenOrdersRequest{Type: "stop"})
	require.ErrorIs(t, err, errInvalidOpenOrderType, "GetOpenOrders must reject an undocumented order type")

	skipLiveWithoutCredentials(t)
	result, err := e.GetOpenOrders(t.Context(), &OpenOrdersRequest{})
	require.NoError(t, err, "GetOpenOrders must not error")
	if mockTests {
		assert.Equal(t, testOpenOrders(), result, "GetOpenOrders should decode every field")
	}

	_, err = newUnavailableServerExchange(t).GetOpenOrders(t.Context(), &OpenOrdersRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOpenOrders should return the request failure")
}

func TestGetOpenOrdersByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrdersByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenOrdersByCurrency must reject a nil request")
	_, err = e.GetOpenOrdersByCurrency(t.Context(), &OpenOrdersByCurrencyRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetOpenOrdersByCurrency must reject an empty currency")
	_, err = e.GetOpenOrdersByCurrency(t.Context(), &OpenOrdersByCurrencyRequest{Currency: currency.BTC, Kind: KindCombo})
	require.ErrorIs(t, err, errInvalidKind, "GetOpenOrdersByCurrency must reject a kind it does not document")
	_, err = e.GetOpenOrdersByCurrency(t.Context(), &OpenOrdersByCurrencyRequest{Currency: currency.BTC, Type: "stop"})
	require.ErrorIs(t, err, errInvalidOpenOrderType, "GetOpenOrdersByCurrency must reject an undocumented order type")

	skipLiveWithoutCredentials(t)
	result, err := e.GetOpenOrdersByCurrency(t.Context(), mockOpenOrdersByCurrencyRequest())
	require.NoError(t, err, "GetOpenOrdersByCurrency must not error")
	if mockTests {
		assert.Equal(t, testOpenOrders(), result, "GetOpenOrdersByCurrency should decode every field")
	}

	_, err = newUnavailableServerExchange(t).GetOpenOrdersByCurrency(t.Context(), mockOpenOrdersByCurrencyRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOpenOrdersByCurrency should return the request failure")
}

func TestGetOpenOrdersByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrdersByInstrument(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenOrdersByInstrument must reject a nil request")
	_, err = e.GetOpenOrdersByInstrument(t.Context(), &OpenOrdersByInstrumentRequest{})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetOpenOrdersByInstrument must reject an empty instrument name")
	_, err = e.GetOpenOrdersByInstrument(t.Context(), &OpenOrdersByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", Type: "stop"})
	require.ErrorIs(t, err, errInvalidOpenOrderType, "GetOpenOrdersByInstrument must reject an undocumented order type")

	skipLiveWithoutCredentials(t)
	arg := mockOpenOrdersByInstrumentRequest()
	if !mockTests {
		arg = &OpenOrdersByInstrumentRequest{InstrumentName: liveTradingInstrument}
	}
	result, err := e.GetOpenOrdersByInstrument(t.Context(), arg)
	require.NoError(t, err, "GetOpenOrdersByInstrument must not error")
	if mockTests {
		assert.Equal(t, testOpenOrders(), result, "GetOpenOrdersByInstrument should decode every field")
	}

	_, err = newUnavailableServerExchange(t).GetOpenOrdersByInstrument(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOpenOrdersByInstrument should return the request failure")
}

func TestGetOpenOrdersByLabel(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrdersByLabel(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenOrdersByLabel must reject a nil request")
	_, err = e.GetOpenOrdersByLabel(t.Context(), &OrdersByLabelRequest{Label: "fooBar"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetOpenOrdersByLabel must reject an empty currency")
	_, err = e.GetOpenOrdersByLabel(t.Context(), &OrdersByLabelRequest{Currency: currency.BTC})
	require.ErrorIs(t, err, errLabelRequired, "GetOpenOrdersByLabel must reject an empty label")

	skipLiveWithoutCredentials(t)
	result, err := e.GetOpenOrdersByLabel(t.Context(), mockOpenOrdersByLabelRequest())
	require.NoError(t, err, "GetOpenOrdersByLabel must not error")
	if mockTests {
		assert.Equal(t, testOpenOrders(), result, "GetOpenOrdersByLabel should decode every field")
	}

	_, err = newUnavailableServerExchange(t).GetOpenOrdersByLabel(t.Context(), mockOpenOrdersByLabelRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOpenOrdersByLabel should return the request failure")
}

func TestGetOrderHistoryByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistoryByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderHistoryByCurrency must reject a nil request")
	_, err = e.GetOrderHistoryByCurrency(t.Context(), &OrderHistoryByCurrencyRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetOrderHistoryByCurrency must reject an empty currency")
	_, err = e.GetOrderHistoryByCurrency(t.Context(), &OrderHistoryByCurrencyRequest{Currency: currency.BTC, Kind: "perpetual"})
	require.ErrorIs(t, err, errInvalidKind, "GetOrderHistoryByCurrency must reject an undocumented kind")
	_, err = e.GetOrderHistoryByCurrency(t.Context(), &OrderHistoryByCurrencyRequest{Currency: currency.BTC, Count: 1001})
	require.ErrorIs(t, err, errInvalidCount, "GetOrderHistoryByCurrency must reject a count over 1000")

	skipLiveWithoutCredentials(t)
	result, err := e.GetOrderHistoryByCurrency(t.Context(), mockOrderHistoryByCurrencyRequest())
	require.NoError(t, err, "GetOrderHistoryByCurrency must not error")
	if mockTests {
		assert.Equal(t, testOrderHistoryResponse(), result, "GetOrderHistoryByCurrency should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Orders), 1, "GetOrderHistoryByCurrency should return at most the count of orders")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetOrderHistoryByCurrency(t.Context(), mockOrderHistoryByCurrencyRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOrderHistoryByCurrency should reject a null result")
	_, err = newUnavailableServerExchange(t).GetOrderHistoryByCurrency(t.Context(), mockOrderHistoryByCurrencyRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderHistoryByCurrency should return the request failure")
}

func TestGetOrderHistoryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistoryByInstrument(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderHistoryByInstrument must reject a nil request")
	_, err = e.GetOrderHistoryByInstrument(t.Context(), &OrderHistoryByInstrumentRequest{})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetOrderHistoryByInstrument must reject an empty instrument name")
	_, err = e.GetOrderHistoryByInstrument(t.Context(), &OrderHistoryByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", Count: 1001})
	require.ErrorIs(t, err, errInvalidCount, "GetOrderHistoryByInstrument must reject a count over 1000")

	skipLiveWithoutCredentials(t)
	result, err := e.GetOrderHistoryByInstrument(t.Context(), mockOrderHistoryByInstrumentRequest())
	require.NoError(t, err, "GetOrderHistoryByInstrument must not error")
	if mockTests {
		assert.Equal(t, testOrderHistoryResponse(), result, "GetOrderHistoryByInstrument should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Orders), 1, "GetOrderHistoryByInstrument should return at most the count of orders")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetOrderHistoryByInstrument(t.Context(), mockOrderHistoryByInstrumentRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOrderHistoryByInstrument should reject a null result")
	_, err = newUnavailableServerExchange(t).GetOrderHistoryByInstrument(t.Context(), mockOrderHistoryByInstrumentRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderHistoryByInstrument should return the request failure")
}

func TestGetOrderMarginByIDs(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderMarginByIDs(t.Context(), nil)
	require.ErrorIs(t, err, errOrderIDRequired, "GetOrderMarginByIDs must reject an empty order ID list")
	_, err = e.GetOrderMarginByIDs(t.Context(), []string{"ETH-349280", ""})
	require.ErrorIs(t, err, errOrderIDRequired, "GetOrderMarginByIDs must reject an empty order ID in the list")

	skipLiveUnlessCanManipulate(t)
	orderIDs := mockOrderMarginIDs
	if !mockTests {
		_, order := placeLiveTradingOrder(t)
		orderIDs = []string{order.OrderID}
	}
	result, err := e.GetOrderMarginByIDs(t.Context(), orderIDs)
	require.NoError(t, err, "GetOrderMarginByIDs must not error")
	if mockTests {
		assert.Equal(t, testOrderMargins(), result, "GetOrderMarginByIDs should decode every field")
	} else if assert.Len(t, result, 1, "GetOrderMarginByIDs should return the order's margin") {
		assert.Equal(t, orderIDs[0], result[0].OrderID, "GetOrderMarginByIDs should return the order's margin")
	}

	_, err = newUnavailableServerExchange(t).GetOrderMarginByIDs(t.Context(), orderIDs)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderMarginByIDs should return the request failure")
}

func TestGetOrderState(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderState(t.Context(), "")
	require.ErrorIs(t, err, errOrderIDRequired, "GetOrderState must reject an empty order ID")

	skipLiveUnlessCanManipulate(t)
	orderID := mockOrderStateID
	if !mockTests {
		_, order := placeLiveTradingOrder(t)
		orderID = order.OrderID
	}
	result, err := e.GetOrderState(t.Context(), orderID)
	require.NoError(t, err, "GetOrderState must not error")
	if mockTests {
		assert.Equal(t, testOrderState(""), result, "GetOrderState should decode every field")
	} else {
		assert.Equal(t, orderID, result.OrderID, "GetOrderState should return the order")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetOrderState(t.Context(), orderID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOrderState should reject a null result")
	_, err = newUnavailableServerExchange(t).GetOrderState(t.Context(), orderID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderState should return the request failure")
}

func TestGetOrderStateByLabel(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderStateByLabel(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderStateByLabel must reject a nil request")
	_, err = e.GetOrderStateByLabel(t.Context(), &OrdersByLabelRequest{Label: "fooBar"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetOrderStateByLabel must reject an empty currency")
	_, err = e.GetOrderStateByLabel(t.Context(), &OrdersByLabelRequest{Currency: currency.ETH})
	require.ErrorIs(t, err, errLabelRequired, "GetOrderStateByLabel must reject an empty label")

	skipLiveWithoutCredentials(t)
	result, err := e.GetOrderStateByLabel(t.Context(), mockOrderStateByLabelRequest())
	require.NoError(t, err, "GetOrderStateByLabel must not error")
	if mockTests {
		assert.Equal(t, []OrderResponse{*testOrderState("fooBar")}, result, "GetOrderStateByLabel should decode every field")
	}

	_, err = newUnavailableServerExchange(t).GetOrderStateByLabel(t.Context(), mockOrderStateByLabelRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderStateByLabel should return the request failure")
}

func TestGetSettlementHistoryByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetSettlementHistoryByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSettlementHistoryByCurrency must reject a nil request")
	_, err = e.GetSettlementHistoryByCurrency(t.Context(), &SettlementHistoryByCurrencyRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSettlementHistoryByCurrency must reject an empty currency")
	_, err = e.GetSettlementHistoryByCurrency(t.Context(), &SettlementHistoryByCurrencyRequest{Currency: currency.BTC, Type: "exercise"})
	require.ErrorIs(t, err, errInvalidSettlementHistoryType, "GetSettlementHistoryByCurrency must reject an undocumented settlement type")
	_, err = e.GetSettlementHistoryByCurrency(t.Context(), &SettlementHistoryByCurrencyRequest{Currency: currency.BTC, Count: 1001})
	require.ErrorIs(t, err, errInvalidCount, "GetSettlementHistoryByCurrency must reject a count over 1000")

	ex, lastParams := newTradingParamsCaptureExchange(t, `{"jsonrpc":"2.0","result":{"settlements":[]}}`)
	_, err = ex.GetSettlementHistoryByCurrency(t.Context(), &SettlementHistoryByCurrencyRequest{Currency: currency.BTC, SearchStartTime: time.UnixMilli(1700000000000)})
	require.NoError(t, err, "GetSettlementHistoryByCurrency must not error")
	assert.JSONEq(t, `{"currency":"BTC","search_start_timestamp":1700000000000}`, lastParams(), "GetSettlementHistoryByCurrency should send the search start time in Unix milliseconds")

	skipLiveWithoutCredentials(t)
	arg := mockSettlementHistoryByCurrencyRequest()
	if !mockTests {
		arg = &SettlementHistoryByCurrencyRequest{Currency: currency.BTC, Count: 1}
	}
	result, err := e.GetSettlementHistoryByCurrency(t.Context(), arg)
	require.NoError(t, err, "GetSettlementHistoryByCurrency must not error")
	if mockTests {
		assert.Equal(t, testSettlementHistoryResponse(), result, "GetSettlementHistoryByCurrency should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Settlements), 1, "GetSettlementHistoryByCurrency should return at most the count of events")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetSettlementHistoryByCurrency(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetSettlementHistoryByCurrency should reject a null result")
	_, err = newUnavailableServerExchange(t).GetSettlementHistoryByCurrency(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSettlementHistoryByCurrency should return the request failure")
}

func TestGetSettlementHistoryByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetSettlementHistoryByInstrument(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSettlementHistoryByInstrument must reject a nil request")
	_, err = e.GetSettlementHistoryByInstrument(t.Context(), &SettlementHistoryByInstrumentRequest{})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetSettlementHistoryByInstrument must reject an empty instrument name")
	_, err = e.GetSettlementHistoryByInstrument(t.Context(), &SettlementHistoryByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", Type: "exercise"})
	require.ErrorIs(t, err, errInvalidSettlementHistoryType, "GetSettlementHistoryByInstrument must reject an undocumented settlement type")
	_, err = e.GetSettlementHistoryByInstrument(t.Context(), &SettlementHistoryByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", Count: 1001})
	require.ErrorIs(t, err, errInvalidCount, "GetSettlementHistoryByInstrument must reject a count over 1000")

	ex, lastParams := newTradingParamsCaptureExchange(t, `{"jsonrpc":"2.0","result":{"settlements":[]}}`)
	_, err = ex.GetSettlementHistoryByInstrument(t.Context(), &SettlementHistoryByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", SearchStartTime: time.UnixMilli(1700000000000)})
	require.NoError(t, err, "GetSettlementHistoryByInstrument must not error")
	assert.JSONEq(t, `{"instrument_name":"BTC-PERPETUAL","search_start_timestamp":1700000000000}`, lastParams(), "GetSettlementHistoryByInstrument should send the search start time in Unix milliseconds")

	skipLiveWithoutCredentials(t)
	arg := mockSettlementHistoryByInstrumentRequest()
	if !mockTests {
		arg = &SettlementHistoryByInstrumentRequest{InstrumentName: liveTradingInstrument, Count: 1}
	}
	result, err := e.GetSettlementHistoryByInstrument(t.Context(), arg)
	require.NoError(t, err, "GetSettlementHistoryByInstrument must not error")
	if mockTests {
		assert.Equal(t, testSettlementHistoryResponse(), result, "GetSettlementHistoryByInstrument should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Settlements), 1, "GetSettlementHistoryByInstrument should return at most the count of events")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetSettlementHistoryByInstrument(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetSettlementHistoryByInstrument should reject a null result")
	_, err = newUnavailableServerExchange(t).GetSettlementHistoryByInstrument(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSettlementHistoryByInstrument should return the request failure")
}

func TestGetTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetTriggerOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTriggerOrderHistory must reject a nil request")
	_, err = e.GetTriggerOrderHistory(t.Context(), &TriggerOrderHistoryRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTriggerOrderHistory must reject an empty currency")
	_, err = e.GetTriggerOrderHistory(t.Context(), &TriggerOrderHistoryRequest{Currency: currency.ETH, Count: 1001})
	require.ErrorIs(t, err, errInvalidCount, "GetTriggerOrderHistory must reject a count over 1000")

	skipLiveWithoutCredentials(t)
	arg := mockTriggerOrderHistoryRequest()
	if !mockTests {
		arg = &TriggerOrderHistoryRequest{Currency: currency.BTC, Count: 1}
	}
	result, err := e.GetTriggerOrderHistory(t.Context(), arg)
	require.NoError(t, err, "GetTriggerOrderHistory must not error")
	if mockTests {
		assert.Equal(t, testTriggerOrderHistoryResponse(), result, "GetTriggerOrderHistory should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Entries), 1, "GetTriggerOrderHistory should return at most the count of entries")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetTriggerOrderHistory(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTriggerOrderHistory should reject a null result")
	_, err = newUnavailableServerExchange(t).GetTriggerOrderHistory(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTriggerOrderHistory should return the request failure")
}

func TestGetUserTradesByCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserTradesByCurrency(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserTradesByCurrency must reject a nil request")
	_, err = e.GetUserTradesByCurrency(t.Context(), &UserTradesByCurrencyRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUserTradesByCurrency must reject an empty currency")
	_, err = e.GetUserTradesByCurrency(t.Context(), &UserTradesByCurrencyRequest{Currency: currency.BTC, Kind: "perpetual"})
	require.ErrorIs(t, err, errInvalidKind, "GetUserTradesByCurrency must reject an undocumented kind")
	_, err = e.GetUserTradesByCurrency(t.Context(), &UserTradesByCurrencyRequest{Currency: currency.BTC, Count: 1001})
	require.ErrorIs(t, err, errInvalidCount, "GetUserTradesByCurrency must reject a count over 1000")
	_, err = e.GetUserTradesByCurrency(t.Context(), &UserTradesByCurrencyRequest{Currency: currency.BTC, Sorting: "newest"})
	require.ErrorIs(t, err, errInvalidUserTradesSorting, "GetUserTradesByCurrency must reject an undocumented sorting")
	_, err = e.GetUserTradesByCurrency(t.Context(), &UserTradesByCurrencyRequest{Currency: currency.BTC, StartTime: time.UnixMilli(2), EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserTradesByCurrency must reject a start after the end")

	ex, lastParams := newTradingParamsCaptureExchange(t, `{"jsonrpc":"2.0","result":{"trades":[]}}`)
	_, err = ex.GetUserTradesByCurrency(t.Context(), &UserTradesByCurrencyRequest{Currency: currency.BTC, EndTime: time.UnixMilli(1700000000000), Historical: true, SubaccountID: 7})
	require.NoError(t, err, "GetUserTradesByCurrency must not error")
	assert.JSONEq(t, `{"currency":"BTC","end_timestamp":1700000000000,"historical":true,"subaccount_id":7}`, lastParams(), "GetUserTradesByCurrency should send a set end time alone, in Unix milliseconds")

	skipLiveWithoutCredentials(t)
	arg := mockUserTradesByCurrencyRequest()
	if !mockTests {
		arg = &UserTradesByCurrencyRequest{Currency: currency.BTC, Count: 1}
	}
	result, err := e.GetUserTradesByCurrency(t.Context(), arg)
	require.NoError(t, err, "GetUserTradesByCurrency must not error")
	if mockTests {
		assert.Equal(t, testUserTradesResponse(), result, "GetUserTradesByCurrency should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Trades), 1, "GetUserTradesByCurrency should return at most the count of trades")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetUserTradesByCurrency(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserTradesByCurrency should reject a null result")
	_, err = newUnavailableServerExchange(t).GetUserTradesByCurrency(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserTradesByCurrency should return the request failure")
}

func TestGetUserTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserTradesByCurrencyAndTime(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserTradesByCurrencyAndTime must reject a nil request")
	_, err = e.GetUserTradesByCurrencyAndTime(t.Context(), &UserTradesByCurrencyAndTimeRequest{EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUserTradesByCurrencyAndTime must reject an empty currency")
	_, err = e.GetUserTradesByCurrencyAndTime(t.Context(), &UserTradesByCurrencyAndTimeRequest{Currency: currency.BTC})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetUserTradesByCurrencyAndTime must reject a range without a start or an end")
	_, err = e.GetUserTradesByCurrencyAndTime(t.Context(), &UserTradesByCurrencyAndTimeRequest{Currency: currency.BTC, EndTime: time.UnixMilli(1), Kind: "perpetual"})
	require.ErrorIs(t, err, errInvalidKind, "GetUserTradesByCurrencyAndTime must reject an undocumented kind")
	_, err = e.GetUserTradesByCurrencyAndTime(t.Context(), &UserTradesByCurrencyAndTimeRequest{Currency: currency.BTC, EndTime: time.UnixMilli(1), Count: 1001})
	require.ErrorIs(t, err, errInvalidCount, "GetUserTradesByCurrencyAndTime must reject a count over 1000")
	_, err = e.GetUserTradesByCurrencyAndTime(t.Context(), &UserTradesByCurrencyAndTimeRequest{Currency: currency.BTC, EndTime: time.UnixMilli(1), Sorting: "newest"})
	require.ErrorIs(t, err, errInvalidUserTradesSorting, "GetUserTradesByCurrencyAndTime must reject an undocumented sorting")
	_, err = e.GetUserTradesByCurrencyAndTime(t.Context(), &UserTradesByCurrencyAndTimeRequest{Currency: currency.BTC, StartTime: time.UnixMilli(2), EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserTradesByCurrencyAndTime must reject a start after the end")

	skipLiveWithoutCredentials(t)
	arg := mockUserTradesByCurrencyAndTimeRequest()
	if !mockTests {
		arg = &UserTradesByCurrencyAndTimeRequest{Currency: currency.BTC, StartTime: time.Now().Add(-24 * time.Hour), EndTime: time.Now(), Count: 1}
	}
	result, err := e.GetUserTradesByCurrencyAndTime(t.Context(), arg)
	require.NoError(t, err, "GetUserTradesByCurrencyAndTime must not error")
	if mockTests {
		assert.Equal(t, testUserTradesResponse(), result, "GetUserTradesByCurrencyAndTime should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Trades), 1, "GetUserTradesByCurrencyAndTime should return at most the count of trades")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetUserTradesByCurrencyAndTime(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserTradesByCurrencyAndTime should reject a null result")
	_, err = newUnavailableServerExchange(t).GetUserTradesByCurrencyAndTime(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserTradesByCurrencyAndTime should return the request failure")
}

func TestGetUserTradesByInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserTradesByInstrument(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserTradesByInstrument must reject a nil request")
	_, err = e.GetUserTradesByInstrument(t.Context(), &UserTradesByInstrumentRequest{})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetUserTradesByInstrument must reject an empty instrument name")
	_, err = e.GetUserTradesByInstrument(t.Context(), &UserTradesByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", Count: 1001})
	require.ErrorIs(t, err, errInvalidCount, "GetUserTradesByInstrument must reject a count over 1000")
	_, err = e.GetUserTradesByInstrument(t.Context(), &UserTradesByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", Sorting: "newest"})
	require.ErrorIs(t, err, errInvalidUserTradesSorting, "GetUserTradesByInstrument must reject an undocumented sorting")
	_, err = e.GetUserTradesByInstrument(t.Context(), &UserTradesByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", StartTime: time.UnixMilli(2), EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserTradesByInstrument must reject a start after the end")

	ex, lastParams := newTradingParamsCaptureExchange(t, `{"jsonrpc":"2.0","result":{"trades":[]}}`)
	_, err = ex.GetUserTradesByInstrument(t.Context(), &UserTradesByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", StartTime: time.UnixMilli(1600000000000), EndTime: time.UnixMilli(1700000000000), EndSequence: 9})
	require.NoError(t, err, "GetUserTradesByInstrument must not error")
	assert.JSONEq(t, `{"instrument_name":"BTC-PERPETUAL","end_seq":9,"start_timestamp":1600000000000,"end_timestamp":1700000000000}`, lastParams(), "GetUserTradesByInstrument should send the time range in Unix milliseconds")

	skipLiveWithoutCredentials(t)
	arg := mockUserTradesByInstrumentRequest()
	if !mockTests {
		arg = &UserTradesByInstrumentRequest{InstrumentName: liveTradingInstrument, Count: 1}
	}
	result, err := e.GetUserTradesByInstrument(t.Context(), arg)
	require.NoError(t, err, "GetUserTradesByInstrument must not error")
	if mockTests {
		assert.Equal(t, testUserTradesResponse(), result, "GetUserTradesByInstrument should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Trades), 1, "GetUserTradesByInstrument should return at most the count of trades")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetUserTradesByInstrument(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserTradesByInstrument should reject a null result")
	_, err = newUnavailableServerExchange(t).GetUserTradesByInstrument(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserTradesByInstrument should return the request failure")
}

func TestGetUserTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserTradesByInstrumentAndTime(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserTradesByInstrumentAndTime must reject a nil request")
	_, err = e.GetUserTradesByInstrumentAndTime(t.Context(), &UserTradesByInstrumentAndTimeRequest{EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetUserTradesByInstrumentAndTime must reject an empty instrument name")
	_, err = e.GetUserTradesByInstrumentAndTime(t.Context(), &UserTradesByInstrumentAndTimeRequest{InstrumentName: "BTC-PERPETUAL"})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetUserTradesByInstrumentAndTime must reject a range without a start or an end")
	_, err = e.GetUserTradesByInstrumentAndTime(t.Context(), &UserTradesByInstrumentAndTimeRequest{InstrumentName: "BTC-PERPETUAL", EndTime: time.UnixMilli(1), Count: 1001})
	require.ErrorIs(t, err, errInvalidCount, "GetUserTradesByInstrumentAndTime must reject a count over 1000")
	_, err = e.GetUserTradesByInstrumentAndTime(t.Context(), &UserTradesByInstrumentAndTimeRequest{InstrumentName: "BTC-PERPETUAL", EndTime: time.UnixMilli(1), Sorting: "newest"})
	require.ErrorIs(t, err, errInvalidUserTradesSorting, "GetUserTradesByInstrumentAndTime must reject an undocumented sorting")
	_, err = e.GetUserTradesByInstrumentAndTime(t.Context(), &UserTradesByInstrumentAndTimeRequest{InstrumentName: "BTC-PERPETUAL", StartTime: time.UnixMilli(2), EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserTradesByInstrumentAndTime must reject a start after the end")

	skipLiveWithoutCredentials(t)
	arg := mockUserTradesByInstrumentAndTimeRequest()
	if !mockTests {
		arg = &UserTradesByInstrumentAndTimeRequest{InstrumentName: liveTradingInstrument, StartTime: time.Now().Add(-24 * time.Hour), EndTime: time.Now(), Count: 1}
	}
	result, err := e.GetUserTradesByInstrumentAndTime(t.Context(), arg)
	require.NoError(t, err, "GetUserTradesByInstrumentAndTime must not error")
	if mockTests {
		assert.Equal(t, testUserTradesResponse(), result, "GetUserTradesByInstrumentAndTime should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Trades), 1, "GetUserTradesByInstrumentAndTime should return at most the count of trades")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetUserTradesByInstrumentAndTime(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserTradesByInstrumentAndTime should reject a null result")
	_, err = newUnavailableServerExchange(t).GetUserTradesByInstrumentAndTime(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserTradesByInstrumentAndTime should return the request failure")
}

func TestGetUserTradesByOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserTradesByOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserTradesByOrder must reject a nil request")
	_, err = e.GetUserTradesByOrder(t.Context(), &UserTradesByOrderRequest{})
	require.ErrorIs(t, err, errOrderIDRequired, "GetUserTradesByOrder must reject an empty order ID")
	_, err = e.GetUserTradesByOrder(t.Context(), &UserTradesByOrderRequest{OrderID: mockUserTradesOrderID, Sorting: "newest"})
	require.ErrorIs(t, err, errInvalidUserTradesSorting, "GetUserTradesByOrder must reject an undocumented sorting")

	skipLiveUnlessCanManipulate(t)
	arg := &UserTradesByOrderRequest{OrderID: mockUserTradesOrderID}
	if !mockTests {
		_, order := placeLiveTradingOrder(t)
		arg = &UserTradesByOrderRequest{OrderID: order.OrderID}
	}
	result, err := e.GetUserTradesByOrder(t.Context(), arg)
	require.NoError(t, err, "GetUserTradesByOrder must not error")
	if mockTests {
		assert.Equal(t, testUserTradesByOrder(), result, "GetUserTradesByOrder should decode every field")
	} else {
		assert.Empty(t, result, "GetUserTradesByOrder should return no trades for an unfilled order")
	}

	_, err = newUnavailableServerExchange(t).GetUserTradesByOrder(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserTradesByOrder should return the request failure")
}

func TestMassQuote(t *testing.T) {
	t.Parallel()
	_, err := e.MassQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "MassQuote must reject a nil request")
	_, err = e.MassQuote(t.Context(), &MassQuoteRequest{MMPGroup: "default"})
	require.ErrorIs(t, err, errMassQuoteIDRequired, "MassQuote must reject an empty quote ID")

	ex, lastParams := newTradingParamsCaptureExchange(t, `{"jsonrpc":"2.0","result":{"orders":[]}}`)
	_, err = ex.MassQuote(t.Context(), &MassQuoteRequest{
		WaitForResponse: new(false),
		QuoteID:         "2",
		MMPGroup:        "default",
		ValidUntil:      time.UnixMilli(1700000000000),
		Quotes:          []MassQuoteEntry{{InstrumentName: "BTC-PERPETUAL", Bid: &MassQuoteSide{Amount: new(0.0)}, Ask: &MassQuoteSide{Price: new(43800.0), PostOnly: true}}},
	})
	require.NoError(t, err, "MassQuote must not error")
	exp := `{"wait_for_response":false,"quote_id":"2","mmp_group":"default","valid_until":1700000000000,"quotes":[{"instrument_name":"BTC-PERPETUAL","bid":{"amount":0},"ask":{"price":43800,"post_only":true}}]}`
	assert.JSONEq(t, exp, lastParams(), "MassQuote should send a false wait_for_response and a zero amount, which cancels the side, and leave out an unset price")

	if !mockTests {
		t.Skip("MassQuote is not tested live, as " + liveTradingSkipMassQuotes)
	}
	result, err := e.MassQuote(t.Context(), mockMassQuoteRequest())
	require.NoError(t, err, "MassQuote must not error")
	assert.Equal(t, testMassQuoteResponse(), result, "MassQuote should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).MassQuote(t.Context(), mockMassQuoteRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "MassQuote should reject a null result")
	_, err = newUnavailableServerExchange(t).MassQuote(t.Context(), mockMassQuoteRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "MassQuote should return the request failure")
}

func TestMovePositions(t *testing.T) {
	t.Parallel()
	_, err := e.MovePositions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "MovePositions must reject a nil request")
	_, err = e.MovePositions(t.Context(), &MovePositionsRequest{TargetUID: 23})
	require.ErrorIs(t, err, errSourceUIDRequired, "MovePositions must reject an empty source subaccount")

	if !mockTests {
		t.Skip("MovePositions needs two subaccounts and a position to move, which live tests do not have")
	}
	result, err := e.MovePositions(t.Context(), mockMovePositionsRequest())
	require.NoError(t, err, "MovePositions must not error")
	assert.Equal(t, testPositionMoveTrades(), result, "MovePositions should decode every field")

	_, err = newUnavailableServerExchange(t).MovePositions(t.Context(), mockMovePositionsRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "MovePositions should return the request failure")
}

func TestResetMMP(t *testing.T) {
	t.Parallel()
	err := e.ResetMMP(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ResetMMP must reject a nil request")
	err = e.ResetMMP(t.Context(), &MMPGroupRequest{MMPGroup: "MassQuoteBot7"})
	require.ErrorIs(t, err, errMMPIndexNameRequired, "ResetMMP must reject an empty index name")
	err = e.ResetMMP(t.Context(), &MMPGroupRequest{IndexName: mmpIndexNameAll})
	require.ErrorIs(t, err, errMMPIndexNameAllNeedsBlockRFQ, "ResetMMP must reject every index outside block RFQ MMP")
	limited, noDelay := newRateLimitedServerExchange(t, `{"jsonrpc":"2.0","result":"ok"}`)
	assert.NoError(t, limited.ResetMMP(noDelay, mockMMPGroupRequest()), "ResetMMP should be limited as a non-matching engine request")

	skipLiveUnlessCanManipulate(t)
	arg := mockMMPGroupRequest()
	if !mockTests {
		setLiveTradingMMPConfig(t, e.SetMMPConfig)
		arg = &MMPGroupRequest{IndexName: liveTradingIndexName}
	}
	require.NoError(t, e.ResetMMP(t.Context(), arg), "ResetMMP must not error")

	err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":"failed"}`).ResetMMP(t.Context(), arg)
	assert.ErrorIs(t, err, errUnexpectedResult, "ResetMMP should reject a result other than ok")
	err = newUnavailableServerExchange(t).ResetMMP(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "ResetMMP should return the request failure")
}

func TestSell(t *testing.T) {
	t.Parallel()
	_, err := e.Sell(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Sell must reject a nil request")
	_, err = e.Sell(t.Context(), &OrderRequest{InstrumentName: "ETH-PERPETUAL"})
	require.ErrorIs(t, err, errInvalidAmount, "Sell must reject an order without a size")

	skipLiveUnlessCanManipulate(t)
	arg := mockSellRequest()
	if !mockTests {
		arg = newLiveTradingOrderRequest(t, DirectionSell)
	}
	result, err := e.Sell(t.Context(), arg)
	require.NoError(t, err, "Sell must not error")
	if mockTests {
		assert.Equal(t, testPlacedOrderResponse(), result, "Sell should decode every field")
	} else {
		assert.Equal(t, "open", result.Order.OrderState, "Sell should place a resting order")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).Sell(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "Sell should reject a null result")
	_, err = newUnavailableServerExchange(t).Sell(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "Sell should return the request failure")
}

func TestSetMMPConfig(t *testing.T) {
	t.Parallel()
	_, err := e.SetMMPConfig(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetMMPConfig must reject a nil request")
	_, err = e.SetMMPConfig(t.Context(), &SetMMPConfigRequest{Interval: time.Minute})
	require.ErrorIs(t, err, errMMPIndexNameRequired, "SetMMPConfig must reject an empty index name")
	_, err = e.SetMMPConfig(t.Context(), &SetMMPConfigRequest{IndexName: "btc_usd", Interval: time.Hour + time.Second})
	require.ErrorIs(t, err, errInvalidMMPInterval, "SetMMPConfig must reject an interval over an hour")
	_, err = e.SetMMPConfig(t.Context(), &SetMMPConfigRequest{IndexName: "btc_usd", FrozenTime: 1500 * time.Millisecond})
	require.ErrorIs(t, err, errInvalidMMPFrozenTime, "SetMMPConfig must reject a frozen time that is not whole seconds")
	_, err = e.SetMMPConfig(t.Context(), &SetMMPConfigRequest{IndexName: mmpIndexNameAll, BlockRFQ: true, TradeCountLimit: 1001})
	require.ErrorIs(t, err, errInvalidMMPTradeCountLimit, "SetMMPConfig must reject a trade count limit over 1000")

	ex, lastParams := newTradingParamsCaptureExchange(t, `{"jsonrpc":"2.0","result":[]}`)
	_, err = ex.SetMMPConfig(t.Context(), &SetMMPConfigRequest{IndexName: "btc_usd", FrozenTime: 90 * time.Second})
	require.NoError(t, err, "SetMMPConfig must not error")
	assert.JSONEq(t, `{"index_name":"btc_usd","interval":0,"frozen_time":90,"max_quote_quantity":0}`, lastParams(), "SetMMPConfig should send the times in seconds, and a zero interval, which removes the configuration")
	limited, noDelay := newRateLimitedServerExchange(t, `{"jsonrpc":"2.0","result":[]}`)
	_, err = limited.SetMMPConfig(noDelay, mockSetMMPConfigRequest())
	assert.NoError(t, err, "SetMMPConfig should be limited as a non-matching engine request")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		result := setLiveTradingMMPConfig(t, e.SetMMPConfig)
		assert.NotEmpty(t, result, "SetMMPConfig should return the configuration")
		return
	}
	result, err := e.SetMMPConfig(t.Context(), mockSetMMPConfigRequest())
	require.NoError(t, err, "SetMMPConfig must not error")
	assert.Equal(t, testSetMMPConfigs(), result, "SetMMPConfig should decode every field")

	_, err = newUnavailableServerExchange(t).SetMMPConfig(t.Context(), mockSetMMPConfigRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "SetMMPConfig should return the request failure")
}

func TestCancelAllResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result CancelAllResponse
	require.NoError(t, json.Unmarshal([]byte(`4`), &result), "Unmarshal must not error for a count")
	assert.Equal(t, CancelAllResponse{CancelledCount: 4}, result, "UnmarshalJSON should decode the count")

	data := `[{"instrument_name":"BTC-PERPETUAL","currency":"BTC","type":"limit","result":[{"order_id":"1"},{"order_id":"2"}]},{"currency":"ETH","type":"stop","result":[]}]`
	require.NoError(t, json.Unmarshal([]byte(data), &result), "Unmarshal must not error for execution reports")
	exp := CancelAllResponse{
		CancelledCount: 2,
		ExecutionReports: []CancelExecutionReport{
			{InstrumentName: "BTC-PERPETUAL", Currency: "BTC", Type: "limit", Result: []OrderResponse{{OrderID: "1"}, {OrderID: "2"}}},
			{Currency: "ETH", Type: "stop", Result: []OrderResponse{}},
		},
	}
	assert.Equal(t, exp, result, "UnmarshalJSON should decode the execution reports and count their orders")

	for _, data := range []string{`"4"`, `{}`, `[4]`, `-1`} {
		assert.ErrorIsf(t, json.Unmarshal([]byte(data), &result), errInvalidCancelAllResult, "UnmarshalJSON should reject %s", data)
	}
}

func TestOrderHistoryResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result OrderHistoryResponse
	require.NoError(t, json.Unmarshal([]byte(`[{"order_id":"1"}]`), &result), "Unmarshal must not error for a list")
	assert.Equal(t, OrderHistoryResponse{Orders: []OrderResponse{{OrderID: "1"}}}, result, "UnmarshalJSON should decode a list of orders")

	require.NoError(t, json.Unmarshal([]byte(`{"orders":[{"order_id":"2"}],"continuation":"next"}`), &result), "Unmarshal must not error for a page")
	assert.Equal(t, OrderHistoryResponse{Orders: []OrderResponse{{OrderID: "2"}}, Continuation: "next"}, result, "UnmarshalJSON should decode a page of orders and its continuation token")

	require.NoError(t, json.Unmarshal([]byte(`{"orders":[],"continuation":""}`), &result), "Unmarshal must not error for the last page")
	assert.Equal(t, OrderHistoryResponse{Orders: []OrderResponse{}}, result, "UnmarshalJSON should decode an empty last page")

	err := json.Unmarshal([]byte(`{"entries":[{"order_id":"3"}],"continuation":"next"}`), &result)
	assert.ErrorIs(t, err, errOrderHistoryOrdersMissing, "UnmarshalJSON should reject a page without an orders field")
	for _, data := range []string{`[1]`, `{"orders":{}}`, `"orders"`} {
		assert.Errorf(t, json.Unmarshal([]byte(data), &result), "UnmarshalJSON should reject %s", data)
	}
}

func TestMMPConfigsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result mmpConfigs
	require.NoError(t, json.Unmarshal([]byte(`{"index_name":"btc_usd","interval":60}`), &result), "Unmarshal must not error for a configuration")
	assert.Equal(t, mmpConfigs{{IndexName: "btc_usd", Interval: 60}}, result, "UnmarshalJSON should decode a configuration as a list of one")

	require.NoError(t, json.Unmarshal([]byte(`[{"index_name":"btc_usd","id":1},{"index_name":"eth_usd","mmp_group":"bot"}]`), &result), "Unmarshal must not error for a list")
	assert.Equal(t, mmpConfigs{{IndexName: "btc_usd", ID: 1}, {IndexName: "eth_usd", MMPGroup: "bot"}}, result, "UnmarshalJSON should decode a list of configurations")

	for _, data := range []string{`{"interval":"60"}`, `[1]`, `"btc_usd"`} {
		assert.Errorf(t, json.Unmarshal([]byte(data), &result), "UnmarshalJSON should reject %s", data)
	}
}

func TestPositionMoveTradesUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result positionMoveTrades
	require.NoError(t, json.Unmarshal([]byte(`[{"source_uid":3,"target_uid":23,"amount":110}]`), &result), "Unmarshal must not error for a list")
	assert.Equal(t, positionMoveTrades{{SourceUID: 3, TargetUID: 23, Amount: 110}}, result, "UnmarshalJSON should decode a list of trades")

	require.NoError(t, json.Unmarshal([]byte(`{"trades":[{"instrument_name":"BTC-PERPETUAL","direction":"buy"}]}`), &result), "Unmarshal must not error for an object")
	assert.Equal(t, positionMoveTrades{{InstrumentName: "BTC-PERPETUAL", Direction: DirectionBuy}}, result, "UnmarshalJSON should decode the trades of an object")

	for _, data := range []string{`{"trades":{}}`, `[1]`, `"trades"`} {
		assert.Errorf(t, json.Unmarshal([]byte(data), &result), "UnmarshalJSON should reject %s", data)
	}
}

func TestOrderRequestValidate(t *testing.T) {
	t.Parallel()
	valid := func() *OrderRequest { return &OrderRequest{InstrumentName: "BTC-PERPETUAL", Amount: 10} }
	for _, tc := range []struct {
		name   string
		modify func(*OrderRequest)
		err    error
	}{
		{name: "valid order", modify: func(*OrderRequest) {}},
		{name: "empty instrument name", modify: func(o *OrderRequest) { o.InstrumentName = "" }, err: errInstrumentNameRequired},
		{name: "no size", modify: func(o *OrderRequest) { o.Amount = 0 }, err: errInvalidAmount},
		{name: "contracts alone", modify: func(o *OrderRequest) { o.Amount, o.Contracts = 0, 1 }},
		{name: "undocumented type", modify: func(o *OrderRequest) { o.Type = "stop" }, err: errInvalidOrderType},
		{name: "stop without trigger", modify: func(o *OrderRequest) { o.Type, o.TriggerPrice = "stop_market", 1 }, err: errOrderTriggerRequired},
		{name: "undocumented time in force", modify: func(o *OrderRequest) { o.TimeInForce = "good_til_date" }, err: errInvalidOrderTimeInForce},
		{name: "undocumented trigger", modify: func(o *OrderRequest) { o.Trigger = "spot_price" }, err: errInvalidOrderTrigger},
		{name: "undocumented advanced type", modify: func(o *OrderRequest) { o.Advanced = "usdc" }, err: errInvalidAdvancedOrderType},
		{name: "undocumented linked order type", modify: func(o *OrderRequest) { o.LinkedOrderType = "one_cancels_others" }, err: errInvalidLinkedOrderType},
		{name: "undocumented trigger fill condition", modify: func(o *OrderRequest) { o.TriggerFillCondition = "last_hit" }, err: errInvalidTriggerFillCondition},
		{name: "secondary order without amount", modify: func(o *OrderRequest) { o.OTOCOConfig = []SecondaryOrder{{Direction: DirectionSell}} }, err: errInvalidAmount},
		{
			name: "linked order",
			modify: func(o *OrderRequest) {
				o.LinkedOrderType, o.OTOCOConfig = "one_triggers_other", []SecondaryOrder{{Amount: 10, Direction: DirectionSell}}
			},
		},
	} {
		arg := valid()
		tc.modify(arg)
		assert.ErrorIsf(t, arg.validate(), tc.err, "validate should return the expected error for %s", tc.name)
	}
	assert.ErrorIs(t, (*OrderRequest)(nil).validate(), common.ErrNilPointer, "validate should reject a nil request")
}

func TestSecondaryOrderValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		order SecondaryOrder
		err   error
	}{
		{name: "valid order", order: SecondaryOrder{Amount: 10, Direction: DirectionSell, Type: "take_limit", Price: 2, TriggerPrice: 1, Trigger: "mark_price"}},
		{name: "no amount", order: SecondaryOrder{Direction: DirectionSell}, err: errInvalidAmount},
		{name: "undocumented direction", order: SecondaryOrder{Amount: 10, Direction: "long"}, err: errInvalidDirection},
		{name: "undocumented type", order: SecondaryOrder{Amount: 10, Direction: DirectionBuy, Type: "take"}, err: errInvalidOrderType},
		{name: "take without trigger price", order: SecondaryOrder{Amount: 10, Direction: DirectionBuy, Type: "take_market", Trigger: "last_price"}, err: errOrderTriggerPriceRequired},
		{name: "undocumented time in force", order: SecondaryOrder{Amount: 10, Direction: DirectionBuy, TimeInForce: "good_til_date"}, err: errInvalidOrderTimeInForce},
		{name: "undocumented trigger", order: SecondaryOrder{Amount: 10, Direction: DirectionBuy, Trigger: "spot_price"}, err: errInvalidOrderTrigger},
	} {
		assert.ErrorIsf(t, tc.order.validate(), tc.err, "validate should return the expected error for %s", tc.name)
	}
}

func TestCancelQuotesRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		arg *CancelQuotesRequest
		err error
	}{
		{err: common.ErrNilPointer},
		{arg: &CancelQuotesRequest{}, err: errInvalidCancelQuotesType},
		{arg: &CancelQuotesRequest{CancelType: "delta"}},
		{arg: &CancelQuotesRequest{CancelType: "all"}},
		{arg: &CancelQuotesRequest{CancelType: "quote_set_id"}, err: errCancelQuoteSetIDRequired},
		{arg: &CancelQuotesRequest{CancelType: "quote_set_id", QuoteSetID: "futures"}},
		{arg: &CancelQuotesRequest{CancelType: "instrument"}, err: errInstrumentNameRequired},
		{arg: &CancelQuotesRequest{CancelType: "instrument", InstrumentName: "BTC-PERPETUAL"}},
		{arg: &CancelQuotesRequest{CancelType: "instrument_kind"}, err: errInvalidKind},
		{arg: &CancelQuotesRequest{CancelType: "instrument_kind", Kind: "perpetual"}, err: errInvalidKind},
		{arg: &CancelQuotesRequest{CancelType: "instrument_kind", Kind: KindOptionCombo}},
		{arg: &CancelQuotesRequest{CancelType: "currency"}, err: currency.ErrCurrencyCodeEmpty},
		{arg: &CancelQuotesRequest{CancelType: "currency", Currency: currency.BTC}},
		{arg: &CancelQuotesRequest{CancelType: "currency_pair"}, err: errCancelCurrencyPairRequired},
		{arg: &CancelQuotesRequest{CancelType: "currency_pair", CurrencyPair: "btc_usd"}},
	} {
		cancelType := "nil request"
		if tc.arg != nil {
			cancelType = tc.arg.CancelType
		}
		assert.ErrorIsf(t, tc.arg.validate(), tc.err, "validate should return the expected error for cancel type %q", cancelType)
	}
}

func TestMMPGroupRequestValidateQuery(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, (*MMPGroupRequest)(nil).validateQuery(), common.ErrNilPointer, "validateQuery should reject a nil request")
	assert.ErrorIs(t, (&MMPGroupRequest{MMPGroup: "bot"}).validateQuery(), errMMPIndexNameRequired, "validateQuery should reject an MMP group without its index")
	for _, arg := range []*MMPGroupRequest{{}, {IndexName: "btc_usd"}, {IndexName: "btc_usd", MMPGroup: "bot"}, {BlockRFQ: true}} {
		assert.NoErrorf(t, arg.validateQuery(), "validateQuery should accept %+v", arg)
	}
}

func TestMassQuoteRequestValidate(t *testing.T) {
	t.Parallel()
	side := &MassQuoteSide{Price: new(1.0), Amount: new(1.0)}
	for _, tc := range []struct {
		name string
		arg  *MassQuoteRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty quote ID", arg: &MassQuoteRequest{MMPGroup: "default", Quotes: []MassQuoteEntry{{InstrumentName: "BTC-PERPETUAL", Bid: side}}}, err: errMassQuoteIDRequired},
		{name: "empty MMP group", arg: &MassQuoteRequest{QuoteID: "1", Quotes: []MassQuoteEntry{{InstrumentName: "BTC-PERPETUAL", Bid: side}}}, err: errMMPGroupRequired},
		{name: "no quotes", arg: &MassQuoteRequest{QuoteID: "1", MMPGroup: "default"}, err: errMassQuotesRequired},
		{name: "quote without instrument", arg: &MassQuoteRequest{QuoteID: "1", MMPGroup: "default", Quotes: []MassQuoteEntry{{Bid: side}}}, err: errInstrumentNameRequired},
		{name: "quote without side", arg: &MassQuoteRequest{QuoteID: "1", MMPGroup: "default", Quotes: []MassQuoteEntry{{InstrumentName: "BTC-PERPETUAL"}}}, err: errMassQuoteSideRequired},
		{name: "ask alone", arg: &MassQuoteRequest{QuoteID: "1", MMPGroup: "default", Quotes: []MassQuoteEntry{{InstrumentName: "BTC-PERPETUAL", Ask: side}}}},
	} {
		assert.ErrorIsf(t, tc.arg.validate(), tc.err, "validate should return the expected error for %s", tc.name)
	}
}

func TestMovePositionsRequestValidate(t *testing.T) {
	t.Parallel()
	move := []PositionMove{{InstrumentName: "BTC-PERPETUAL", Amount: 10}}
	for _, tc := range []struct {
		name string
		arg  *MovePositionsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty source", arg: &MovePositionsRequest{TargetUID: 2, Trades: move}, err: errSourceUIDRequired},
		{name: "empty target", arg: &MovePositionsRequest{SourceUID: 1, Trades: move}, err: errTargetUIDRequired},
		{name: "no moves", arg: &MovePositionsRequest{SourceUID: 1, TargetUID: 2}, err: errPositionMovesRequired},
		{name: "move without instrument", arg: &MovePositionsRequest{SourceUID: 1, TargetUID: 2, Trades: []PositionMove{{Amount: 10}}}, err: errInstrumentNameRequired},
		{name: "move without amount", arg: &MovePositionsRequest{SourceUID: 1, TargetUID: 2, Trades: []PositionMove{{InstrumentName: "BTC-PERPETUAL"}}}, err: errInvalidAmount},
		{name: "valid move", arg: &MovePositionsRequest{SourceUID: 1, TargetUID: 2, Trades: move}},
	} {
		assert.ErrorIsf(t, tc.arg.validate(), tc.err, "validate should return the expected error for %s", tc.name)
	}
}

func TestValidateOrderSize(t *testing.T) {
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
		assert.ErrorIsf(t, validateOrderSize(tc.amount, tc.contracts), tc.err, "validateOrderSize should return the expected error for amount %v and contracts %v", tc.amount, tc.contracts)
	}
}

func TestValidateOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType, trigger string
		triggerPrice       float64
		err                error
	}{
		{},
		{orderType: "limit"},
		{orderType: "market"},
		{orderType: "market_limit"},
		{orderType: "stop_limit", trigger: "last_price", triggerPrice: 1},
		{orderType: "stop_market", trigger: "index_price", triggerPrice: 1},
		{orderType: "take_limit", trigger: "mark_price", triggerPrice: 1},
		{orderType: "take_market", trigger: "mark_price", triggerPrice: -1},
		{orderType: "trailing_stop", trigger: "mark_price"},
		{orderType: "stop_limit", triggerPrice: 1, err: errOrderTriggerRequired},
		{orderType: "trailing_stop", err: errOrderTriggerRequired},
		{orderType: "take_market", trigger: "mark_price", err: errOrderTriggerPriceRequired},
		{orderType: "stop", err: errInvalidOrderType},
		{orderType: "Limit", err: errInvalidOrderType},
	} {
		assert.ErrorIsf(t, validateOrderType(tc.orderType, tc.trigger, tc.triggerPrice), tc.err, "validateOrderType should return the expected error for type %q with trigger %q", tc.orderType, tc.trigger)
	}
}

func TestValidateOrderTimeInForce(t *testing.T) {
	t.Parallel()
	for _, timeInForce := range []string{"", "good_til_cancelled", "good_til_day", "fill_or_kill", "immediate_or_cancel"} {
		assert.NoErrorf(t, validateOrderTimeInForce(timeInForce), "validateOrderTimeInForce should accept %q", timeInForce)
	}
	assert.ErrorIs(t, validateOrderTimeInForce("gtc"), errInvalidOrderTimeInForce, "validateOrderTimeInForce should reject an undocumented time in force")
}

func TestValidateOrderTrigger(t *testing.T) {
	t.Parallel()
	for _, trigger := range []string{"", "index_price", "mark_price", "last_price"} {
		assert.NoErrorf(t, validateOrderTrigger(trigger), "validateOrderTrigger should accept %q", trigger)
	}
	assert.ErrorIs(t, validateOrderTrigger("index"), errInvalidOrderTrigger, "validateOrderTrigger should reject an undocumented trigger")
}

func TestValidateAdvancedOrderType(t *testing.T) {
	t.Parallel()
	for _, advanced := range []string{"", "usd", "implv"} {
		assert.NoErrorf(t, validateAdvancedOrderType(advanced), "validateAdvancedOrderType should accept %q", advanced)
	}
	assert.ErrorIs(t, validateAdvancedOrderType("USD"), errInvalidAdvancedOrderType, "validateAdvancedOrderType should reject an undocumented advanced order type")
}

func TestValidateTradingKind(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"", KindFuture, KindOption, KindSpot, KindFutureCombo, KindOptionCombo} {
		assert.NoErrorf(t, validateTradingKind(kind, false), "validateTradingKind should accept %q without aggregates", kind)
		assert.NoErrorf(t, validateTradingKind(kind, true), "validateTradingKind should accept %q with aggregates", kind)
	}
	for _, kind := range []string{KindCombo, KindAny} {
		assert.ErrorIsf(t, validateTradingKind(kind, false), errInvalidKind, "validateTradingKind should reject %q without aggregates", kind)
		assert.NoErrorf(t, validateTradingKind(kind, true), "validateTradingKind should accept %q with aggregates", kind)
	}
	assert.ErrorIs(t, validateTradingKind("perpetual", true), errInvalidKind, "validateTradingKind should reject an undocumented kind")
}

func TestValidateCancelOrderType(t *testing.T) {
	t.Parallel()
	for _, orderType := range []string{"", "all", "limit", "trigger_all", "stop", "take", "trailing_stop"} {
		assert.NoErrorf(t, validateCancelOrderType(orderType), "validateCancelOrderType should accept %q", orderType)
	}
	assert.ErrorIs(t, validateCancelOrderType("stop_all"), errInvalidCancelOrderType, "validateCancelOrderType should reject an order type only open orders methods accept")
}

func TestValidateOpenOrderType(t *testing.T) {
	t.Parallel()
	for _, orderType := range []string{"", "all", "limit", "trigger_all", "stop_all", "stop_limit", "stop_market", "take_all", "take_limit", "take_market", "trailing_all", "trailing_stop"} {
		assert.NoErrorf(t, validateOpenOrderType(orderType), "validateOpenOrderType should accept %q", orderType)
	}
	assert.ErrorIs(t, validateOpenOrderType("take"), errInvalidOpenOrderType, "validateOpenOrderType should reject an order type only cancel methods accept")
}

func TestValidateSettlementHistoryType(t *testing.T) {
	t.Parallel()
	for _, settlementType := range []string{"", "settlement", "delivery", "bankruptcy"} {
		assert.NoErrorf(t, validateSettlementHistoryType(settlementType), "validateSettlementHistoryType should accept %q", settlementType)
	}
	assert.ErrorIs(t, validateSettlementHistoryType("exercise"), errInvalidSettlementHistoryType, "validateSettlementHistoryType should reject a type Deribit does not accept in requests")
}

func TestValidateUserTradesSorting(t *testing.T) {
	t.Parallel()
	for _, sorting := range []string{"", "asc", "desc", "default"} {
		assert.NoErrorf(t, validateUserTradesSorting(sorting), "validateUserTradesSorting should accept %q", sorting)
	}
	assert.ErrorIs(t, validateUserTradesSorting("ascending"), errInvalidUserTradesSorting, "validateUserTradesSorting should reject an undocumented sorting")
}

func TestValidateTradingCount(t *testing.T) {
	t.Parallel()
	for _, count := range []uint64{0, 1, 1000} {
		assert.NoErrorf(t, validateTradingCount(count), "validateTradingCount should accept %d", count)
	}
	assert.ErrorIs(t, validateTradingCount(1001), errInvalidCount, "validateTradingCount should reject a count over 1000")
}

func TestIsValidMMPDuration(t *testing.T) {
	t.Parallel()
	for _, d := range []time.Duration{0, time.Second, time.Minute, time.Hour} {
		assert.Truef(t, isValidMMPDuration(d), "isValidMMPDuration should accept %s", d)
	}
	for _, d := range []time.Duration{-time.Second, time.Hour + time.Second, 1500 * time.Millisecond} {
		assert.Falsef(t, isValidMMPDuration(d), "isValidMMPDuration should reject %s", d)
	}
}

func TestUserTradesTimeRange(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1600000000000), time.UnixMilli(1700000000000)
	for _, tc := range []struct {
		start, end       time.Time
		expStart, expEnd uint64
		err              error
		name             string
	}{
		{name: "no range"},
		{name: "start alone", start: start, expStart: 1600000000000},
		{name: "end alone", end: end, expEnd: 1700000000000},
		{name: "range", start: start, end: end, expStart: 1600000000000, expEnd: 1700000000000},
		{name: "same start and end", start: end, end: end, expStart: 1700000000000, expEnd: 1700000000000},
		{name: "start after end", start: end, end: start, err: common.ErrStartAfterEnd},
		{name: "end before 1970", end: time.UnixMilli(-1), expEnd: 1},
	} {
		startTimestamp, endTimestamp, err := userTradesTimeRange(tc.start, tc.end)
		require.ErrorIsf(t, err, tc.err, "userTradesTimeRange must return the expected error for %s", tc.name)
		assert.Equalf(t, tc.expStart, startTimestamp, "userTradesTimeRange should return the start in Unix milliseconds for %s", tc.name)
		assert.Equalf(t, tc.expEnd, endTimestamp, "userTradesTimeRange should return the end in Unix milliseconds for %s", tc.name)
	}
}
