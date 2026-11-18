package bybit

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// errInvalidGrossPayoutRatio is returned when an event contract quote has no gross payout ratio
var errInvalidGrossPayoutRatio = errors.New("gross payout ratio must be greater than zero")

// GetEventInstrumentsInfo returns event contract specifications (Event Trading: Get Instrument Info)
func (e *Exchange) GetEventInstrumentsInfo(ctx context.Context, req *EventInstrumentsInfoRequest) (*EventInstrumentsInfoResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if req.Status != "" {
		params.Set("status", req.Status)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *EventInstrumentsInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/event/instruments-info", params), defaultEPL, &resp)
}

// GetEventOrderbook returns the 25 level payout ratio orderbook of an event contract (Event Trading: Get Orderbook)
func (e *Exchange) GetEventOrderbook(ctx context.Context, symbol string) (*EventOrderbookResponse, error) {
	if symbol == "" {
		return nil, errSymbolMissing
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *EventOrderbookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/event/orderbook", params), defaultEPL, &resp)
}

// GetEventOrderHistory returns event contract orders in a final state from the last 2 years, newest first; only event
// contract market makers can call it (Event Trading: Get Event Contract Order History)
func (e *Exchange) GetEventOrderHistory(ctx context.Context, req *EventOrderHistoryRequest) (*EventOrderHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.OrderLinkID != "" {
		params.Set("orderLinkId", req.OrderLinkID)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *EventOrderHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/event/order-list", params, nil, &resp, defaultEPL)
}

// GetEventActiveOrders returns unfilled and partially filled event contract orders; only event contract market makers
// can call it (Event Trading: Get Event Contract Active Orders)
func (e *Exchange) GetEventActiveOrders(ctx context.Context, req *EventActiveOrdersRequest) (*EventActiveOrdersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.OrderLinkID != "" {
		params.Set("orderLinkId", req.OrderLinkID)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *EventActiveOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/event/order-realtime", params, nil, &resp, defaultEPL)
}

// GetEventPositionInfo returns event contract positions; only event contract market makers can call it (Event
// Trading: Get Event Contract Position Info)
func (e *Exchange) GetEventPositionInfo(ctx context.Context, req *EventPositionInfoRequest) (*EventPositionInfoResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if !req.BaseCoin.IsEmpty() {
		params.Set("baseCoin", req.BaseCoin.Upper().String())
	}
	if !req.SettleCoin.IsEmpty() {
		params.Set("settleCoin", req.SettleCoin.Upper().String())
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *EventPositionInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/event/positions", params, nil, &resp, defaultEPL)
}

// GetEventSettlementRecords returns an event contract's settlements from the last 2 years; only event contract market
// makers can call it (Event Trading: Get Event Contract Settlement Records)
func (e *Exchange) GetEventSettlementRecords(ctx context.Context, req *EventSettlementRecordsRequest) (*EventSettlementRecordsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol == "" {
		return nil, errSymbolMissing
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *EventSettlementRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/event/settlements", params, nil, &resp, defaultEPL)
}

// GetEventTradeHistory returns an event contract's executions from the last 2 years; only event contract market makers
// can call it (Event Trading: Get Event Contract Trade History)
func (e *Exchange) GetEventTradeHistory(ctx context.Context, req *EventTradeHistoryRequest) (*EventTradeHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol == "" {
		return nil, errSymbolMissing
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.OrderLinkID != "" {
		params.Set("orderLinkId", req.OrderLinkID)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *EventTradeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/event/trades", params, nil, &resp, defaultEPL)
}

// SubmitEventQuote submits a payout ratio quote for an event contract; only event contract market makers can call it
// (Event Trading: Submit Quote)
func (e *Exchange) SubmitEventQuote(ctx context.Context, req *SubmitEventQuoteRequest) (*EventQuoteResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol == "" {
		return nil, errSymbolMissing
	}
	if req.OrderLinkID == "" {
		return nil, errOrderLinkIDMissing
	}
	if req.GrossPayoutRatio <= 0 {
		return nil, errInvalidGrossPayoutRatio
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	body := make(map[string]any)
	body["symbol"] = req.Symbol
	body["orderLinkId"] = req.OrderLinkID
	body["grossPayoutRatio"] = strconv.FormatFloat(req.GrossPayoutRatio, 'f', -1, 64)
	body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	var resp *EventQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/event/quotes", nil, body, &resp, defaultEPL)
}

// CancelEventQuote cancels an event contract quote by its order ID or order link ID; only event contract market makers
// can call it (Event Trading: Cancel Quote)
func (e *Exchange) CancelEventQuote(ctx context.Context, orderID, orderLinkID string) (*EventQuoteResponse, error) {
	if orderID == "" && orderLinkID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	body := map[string]any{}
	if orderLinkID != "" {
		body["orderLinkId"] = orderLinkID
	}
	if orderID != "" {
		body["orderId"] = orderID
	}
	var resp *EventQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/event/cancel", nil, body, &resp, defaultEPL)
}
