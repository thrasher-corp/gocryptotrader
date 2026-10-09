package kraken

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// SendFuturesOrder calls Send order, placing an order on a futures contract. On a market with maker protection, a
// limit, IOC, FOK or market order is held for the market's window before it reaches the matching engine, so the
// response arrives after the hold and ProcessBefore must allow for it
func (e *Exchange) SendFuturesOrder(ctx context.Context, req *FuturesSendOrderRequest) (*FuturesSendOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := futuresValidateNewOrder(req.Symbol, req.Side, req.OrderType, req.Size, req.StopPrice, req.TrailingStopMaximumDeviation, req.TrailingStopDeviationUnit, req.ClientOrderID); err != nil {
		return nil, err
	}
	if req.LimitPriceOffsetValue != 0 && req.LimitPriceOffsetUnit == "" {
		return nil, errFuturesLimitPriceOffsetUnitRequired
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Futures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", req.Side)
	params.Set("orderType", req.OrderType)
	params.Set("size", strconv.FormatFloat(req.Size, 'f', -1, 64))
	if req.LimitPrice != 0 {
		params.Set("limitPrice", strconv.FormatFloat(req.LimitPrice, 'f', -1, 64))
	}
	if req.StopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(req.StopPrice, 'f', -1, 64))
	}
	if req.ClientOrderID != "" {
		params.Set("cliOrdId", req.ClientOrderID)
	}
	if req.TriggerSignal != "" {
		params.Set("triggerSignal", req.TriggerSignal)
	}
	if req.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if req.TrailingStopMaximumDeviation != 0 {
		params.Set("trailingStopMaxDeviation", strconv.FormatFloat(req.TrailingStopMaximumDeviation, 'f', -1, 64))
	}
	if req.TrailingStopDeviationUnit != "" {
		params.Set("trailingStopDeviationUnit", req.TrailingStopDeviationUnit)
	}
	// A zero offset is meaningful, so the offset is sent whenever its unit is
	if req.LimitPriceOffsetUnit != "" {
		params.Set("limitPriceOffsetValue", strconv.FormatFloat(req.LimitPriceOffsetValue, 'f', -1, 64))
		params.Set("limitPriceOffsetUnit", req.LimitPriceOffsetUnit)
	}
	if req.Broker != "" {
		params.Set("broker", req.Broker)
	}
	futuresSetProcessBefore(params, req.ProcessBefore)
	var resp *FuturesSendOrderResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/sendorder", params, futuresAlgorithmHeader(req.AlgorithmID), &resp)
}

// EditFuturesOrder calls Edit order, changing an open order's size or prices. Sending a trailing stop's maximum
// deviation and unit unchanged has Kraken recalculate its stop price. On a market with maker protection, an edit of an
// order that could take liquidity is held like a placement
func (e *Exchange) EditFuturesOrder(ctx context.Context, req *FuturesEditOrderRequest) (*FuturesEditOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == "" && req.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if err := futuresValidateTrailingStopDeviation(req.TrailingStopMaximumDeviation, req.TrailingStopDeviationUnit); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.ClientOrderID != "" {
		params.Set("cliOrdId", req.ClientOrderID)
	}
	if req.Size != 0 {
		params.Set("size", strconv.FormatFloat(req.Size, 'f', -1, 64))
	}
	if req.LimitPrice != 0 {
		params.Set("limitPrice", strconv.FormatFloat(req.LimitPrice, 'f', -1, 64))
	}
	if req.StopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(req.StopPrice, 'f', -1, 64))
	}
	if req.TrailingStopMaximumDeviation != 0 {
		params.Set("trailingStopMaxDeviation", strconv.FormatFloat(req.TrailingStopMaximumDeviation, 'f', -1, 64))
	}
	if req.TrailingStopDeviationUnit != "" {
		params.Set("trailingStopDeviationUnit", req.TrailingStopDeviationUnit)
	}
	if req.QuantityMode != "" {
		params.Set("qtyMode", req.QuantityMode)
	}
	futuresSetProcessBefore(params, req.ProcessBefore)
	var resp *FuturesEditOrderResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/editorder", params, futuresAlgorithmHeader(req.AlgorithmID), &resp)
}

// CancelFuturesOrder calls Cancel order. On a market with maker protection, cancelling a held order that could take
// liquidity does not stop it trading: it still reaches the matching engine at the end of the hold, only losing its
// right to rest
func (e *Exchange) CancelFuturesOrder(ctx context.Context, req *FuturesCancelOrderRequest) (*FuturesCancelOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == "" && req.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if req.OrderID != "" {
		params.Set("order_id", req.OrderID)
	}
	if req.ClientOrderID != "" {
		params.Set("cliOrdId", req.ClientOrderID)
	}
	futuresSetProcessBefore(params, req.ProcessBefore)
	var resp *FuturesCancelOrderResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/cancelorder", params, futuresAlgorithmHeader(req.AlgorithmID), &resp)
}

// CancelAllFuturesOrders calls Cancel all orders, cancelling a contract's open orders, or every open order when the
// request or its symbol is empty. On a market with maker protection, orders still held are converted to
// immediate-or-cancel rather than withdrawn, and an order whose edit is held is deliberately left resting until the
// edit releases
func (e *Exchange) CancelAllFuturesOrders(ctx context.Context, req *FuturesCancelAllOrdersRequest) (*FuturesCancelAllOrdersResponse, error) {
	params := url.Values{}
	var headers map[string]string
	if req != nil {
		if !req.Symbol.IsEmpty() {
			symbol, err := e.FormatSymbol(req.Symbol, asset.Futures)
			if err != nil {
				return nil, err
			}
			params.Set("symbol", symbol)
		}
		headers = futuresAlgorithmHeader(req.AlgorithmID)
	}
	var resp *FuturesCancelAllOrdersResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/cancelallorders", params, headers, &resp)
}

// CancelAllFuturesOrdersAfter calls Dead man's switch, cancelling every order once the timeout passes unless it is
// called again before then; a zero timeout deactivates the switch. Kraken recommends calling it every 15 to 20 seconds
// with a 60 second timeout
func (e *Exchange) CancelAllFuturesOrdersAfter(ctx context.Context, req *FuturesCancelAllOrdersAfterRequest) (*FuturesCancelAllOrdersAfterResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	// A fraction of a second would be truncated, and truncating a sub-second timeout to zero deactivates the switch
	if req.Timeout < 0 || req.Timeout%time.Second != 0 || req.Timeout > math.MaxUint32*time.Second {
		return nil, fmt.Errorf("%w: %s is not a whole number of seconds from 0 to %d", errFuturesInvalidTimeout, req.Timeout, uint32(math.MaxUint32))
	}
	params := url.Values{}
	params.Set("timeout", strconv.FormatInt(int64(req.Timeout/time.Second), 10))
	var resp *FuturesCancelAllOrdersAfterResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/cancelallordersafter", params, futuresAlgorithmHeader(req.AlgorithmID), &resp)
}

// FuturesBatchOrder calls Batch order management, placing, editing and cancelling orders in one request, applying
// the instructions in the order given. On a market with maker protection, the instructions before the first one that
// could take liquidity are forwarded at once, that one and every later one wait out the hold together, and cancels are
// always forwarded at once, so a cancel after the order it names converts that order's hold
func (e *Exchange) FuturesBatchOrder(ctx context.Context, req *FuturesBatchOrderRequest) (*FuturesBatchOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if len(req.Instructions) == 0 {
		return nil, fmt.Errorf("%w: no batch instructions", common.ErrEmptyParams)
	}
	if len(req.Instructions) > 500 {
		return nil, fmt.Errorf("%w: %d batch instructions exceed 500", errFuturesTooManyEntries, len(req.Instructions))
	}
	batch := futuresBatchOrderJSON{BatchOrder: make([]futuresBatchInstructionJSON, len(req.Instructions))}
	for i := range req.Instructions {
		instruction, err := e.futuresBatchInstruction(&req.Instructions[i])
		if err != nil {
			return nil, fmt.Errorf("batch instruction %d: %w", i, err)
		}
		batch.BatchOrder[i] = instruction
	}
	data, err := json.Marshal(batch)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("json", string(data))
	if req.Broker != "" {
		params.Set("broker", req.Broker)
	}
	futuresSetProcessBefore(params, req.ProcessBefore)
	var resp *FuturesBatchOrderResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/batchorder", params, futuresAlgorithmHeader(req.AlgorithmID), &resp)
}

// futuresBatchInstruction validates a batch instruction and converts it to the form Batch order management takes
func (e *Exchange) futuresBatchInstruction(instruction *FuturesBatchInstruction) (futuresBatchInstructionJSON, error) {
	var kinds int
	for _, set := range []bool{instruction.Send != nil, instruction.Edit != nil, instruction.Cancel != nil} {
		if set {
			kinds++
		}
	}
	if kinds != 1 {
		return futuresBatchInstructionJSON{}, fmt.Errorf("%w: %d of send, edit and cancel set, expected 1", errFuturesInvalidBatchInstruction, kinds)
	}
	switch {
	case instruction.Send != nil:
		send := instruction.Send
		if send.OrderTag == "" {
			return futuresBatchInstructionJSON{}, errFuturesOrderTagRequired
		}
		if err := futuresValidateNewOrder(send.Symbol, send.Side, send.OrderType, send.Size, send.StopPrice, send.TrailingStopMaximumDeviation, send.TrailingStopDeviationUnit, send.ClientOrderID); err != nil {
			return futuresBatchInstructionJSON{}, err
		}
		symbol, err := e.FormatSymbol(send.Symbol, asset.Futures)
		if err != nil {
			return futuresBatchInstructionJSON{}, err
		}
		return futuresBatchInstructionJSON{
			Order:                        "send",
			OrderTag:                     send.OrderTag,
			OrderType:                    send.OrderType,
			Symbol:                       symbol,
			Side:                         send.Side,
			Size:                         send.Size,
			LimitPrice:                   send.LimitPrice,
			StopPrice:                    send.StopPrice,
			ClientOrderID:                send.ClientOrderID,
			TriggerSignal:                send.TriggerSignal,
			ReduceOnly:                   send.ReduceOnly,
			TrailingStopMaximumDeviation: send.TrailingStopMaximumDeviation,
			TrailingStopDeviationUnit:    send.TrailingStopDeviationUnit,
		}, nil
	case instruction.Edit != nil:
		edit := instruction.Edit
		if edit.OrderID == "" && edit.ClientOrderID == "" {
			return futuresBatchInstructionJSON{}, order.ErrOrderIDNotSet
		}
		if err := futuresValidateTrailingStopDeviation(edit.TrailingStopMaximumDeviation, edit.TrailingStopDeviationUnit); err != nil {
			return futuresBatchInstructionJSON{}, err
		}
		return futuresBatchInstructionJSON{
			Order:                        "edit",
			OrderID:                      edit.OrderID,
			ClientOrderID:                edit.ClientOrderID,
			Size:                         edit.Size,
			LimitPrice:                   edit.LimitPrice,
			StopPrice:                    edit.StopPrice,
			TrailingStopMaximumDeviation: edit.TrailingStopMaximumDeviation,
			TrailingStopDeviationUnit:    edit.TrailingStopDeviationUnit,
			QuantityMode:                 edit.QuantityMode,
		}, nil
	default:
		cancel := instruction.Cancel
		if cancel.OrderID == "" && cancel.ClientOrderID == "" {
			return futuresBatchInstructionJSON{}, order.ErrOrderIDNotSet
		}
		return futuresBatchInstructionJSON{Order: "cancel", OrderID: cancel.OrderID, ClientOrderID: cancel.ClientOrderID}, nil
	}
}

// GetFuturesOpenOrders calls Get open orders, returning every open order on every contract, newest first
func (e *Exchange) GetFuturesOpenOrders(ctx context.Context) (*FuturesOpenOrdersResponse, error) {
	var resp *FuturesOpenOrdersResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/openorders", nil, nil, &resp)
}

// GetFuturesOrdersStatus calls Get Specific Orders' Status, returning the status of the orders and trigger orders named
// that are open or closed within the last 5 seconds
func (e *Exchange) GetFuturesOrdersStatus(ctx context.Context, req *FuturesOrdersStatusRequest) (*FuturesOrdersStatusResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if len(req.OrderIDs) == 0 && len(req.ClientOrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if slices.Contains(req.OrderIDs, "") || slices.Contains(req.ClientOrderIDs, "") {
		return nil, fmt.Errorf("%w: empty ID", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	for _, id := range req.OrderIDs {
		params.Add("orderIds", id)
	}
	for _, id := range req.ClientOrderIDs {
		params.Add("cliOrdIds", id)
	}
	var resp *FuturesOrdersStatusResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/orders/status", params, nil, &resp)
}

// futuresValidateNewOrder checks the parameters Send order and a batch's send instructions require
func futuresValidateNewOrder(symbol currency.Pair, side, orderType string, size, stopPrice, trailingStopMaximumDeviation float64, trailingStopDeviationUnit, clientOrderID string) error {
	switch {
	case symbol.IsEmpty():
		return currency.ErrCurrencyPairEmpty
	case side == "":
		return order.ErrSideIsInvalid
	case orderType == "":
		return order.ErrTypeIsInvalid
	case size <= 0:
		return fmt.Errorf("%w: size %v", order.ErrAmountIsInvalid, size)
	case stopPrice == 0 && (orderType == "stp" || orderType == "take_profit"):
		return fmt.Errorf("%w for a %s order", errFuturesStopPriceRequired, orderType)
	case orderType == "trailing_stop" && (trailingStopMaximumDeviation == 0 || trailingStopDeviationUnit == ""):
		return errFuturesTrailingStopDeviationRequired
	case utf8.RuneCountInString(clientOrderID) > 100:
		return fmt.Errorf("%w: %d characters exceed 100", errFuturesClientOrderIDTooLong, utf8.RuneCountInString(clientOrderID))
	}
	return futuresValidateTrailingStopDeviation(trailingStopMaximumDeviation, trailingStopDeviationUnit)
}

// futuresValidateTrailingStopDeviation checks a trailing stop's maximum deviation, which as a percentage must be from
// 0.1 to 50. Kraken's specification bounds every deviation so, but its own example trails by 100 in quote currency
func futuresValidateTrailingStopDeviation(maximumDeviation float64, unit string) error {
	if maximumDeviation != 0 && unit == "PERCENT" && (maximumDeviation < 0.1 || maximumDeviation > 50) {
		return fmt.Errorf("%w: %v percent is outside 0.1 to 50", errFuturesInvalidTrailingStopDeviation, maximumDeviation)
	}
	return nil
}

// futuresSetProcessBefore sets the processBefore parameter when the time is set, in the microsecond precision of
// Kraken's example
func futuresSetProcessBefore(params url.Values, processBefore time.Time) {
	if !processBefore.IsZero() {
		params.Set("processBefore", processBefore.UTC().Format("2006-01-02T15:04:05.000000Z07:00"))
	}
}

// futuresAlgorithmHeader returns the algoId header the order management endpoints take, or nil without an algorithm ID
func futuresAlgorithmHeader(algorithmID string) map[string]string {
	if algorithmID == "" {
		return nil
	}
	return map[string]string{"algoId": algorithmID}
}
