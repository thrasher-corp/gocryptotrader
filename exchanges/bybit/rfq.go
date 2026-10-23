package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// RFQ trading errors
var (
	errCounterpartiesRequired  = errors.New("at least one counterparty is required")
	errRFQLegsRequired         = errors.New("at least one RFQ leg is required")
	errRFQIDRequired           = errors.New("RFQ ID is required")
	errRFQIdentifierRequired   = errors.New("either RFQ ID or RFQ link ID is required")
	errQuoteLegsRequired       = errors.New("at least one buy or sell quote leg is required")
	errQuoteIDRequired         = errors.New("quote ID is required")
	errQuoteIdentifierRequired = errors.New("quote ID, quote link ID or RFQ ID is required")
)

// CreateRFQ sends a request for quote to the chosen quoting parties (RFQ Trading: Create RFQ)
func (e *Exchange) CreateRFQ(ctx context.Context, req *CreateRFQRequest) (*CreateRFQResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if len(req.Counterparties) == 0 {
		return nil, errCounterpartiesRequired
	}
	if len(req.List) == 0 {
		return nil, errRFQLegsRequired
	}
	legs := make([]map[string]any, len(req.List))
	for i := range req.List {
		leg := &req.List[i]
		var err error
		if legs[i], err = rfqLegBody(leg.Category, leg.Symbol, leg.Side, leg.Quantity); err != nil {
			return nil, fmt.Errorf("%w: list[%d]", err, i)
		}
	}
	body := map[string]any{
		"counterparties": req.Counterparties,
		"list":           legs,
	}
	if req.RFQLinkID != "" {
		body["rfqLinkId"] = req.RFQLinkID
	}
	if req.Anonymous {
		body["anonymous"] = true
	}
	if req.StrategyType != "" {
		body["strategyType"] = req.StrategyType
	}
	if len(req.Hedge) != 0 {
		hedge := make([]map[string]any, len(req.Hedge))
		for i := range req.Hedge {
			leg := &req.Hedge[i]
			var err error
			if hedge[i], err = rfqLegBody(leg.Category, leg.Symbol, leg.Side, leg.Quantity); err != nil {
				return nil, fmt.Errorf("%w: hedge[%d]", err, i)
			}
			if leg.Price != 0 {
				hedge[i]["price"] = strconv.FormatFloat(leg.Price, 'f', -1, 64)
			}
		}
		body["hedge"] = hedge
	}
	var resp *CreateRFQResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/rfq/create-rfq", nil, body, &resp, createRFQEPL)
}

// rfqLegBody returns the body of an RFQ leg, every field of which Bybit requires
func rfqLegBody(category, symbol, side string, quantity float64) (map[string]any, error) {
	switch {
	case category == "":
		return nil, errCategoryNotSet
	case symbol == "":
		return nil, errSymbolMissing
	case side == "":
		return nil, order.ErrSideIsInvalid
	case quantity <= 0:
		return nil, order.ErrAmountIsInvalid
	}
	return map[string]any{
		"category": category,
		"symbol":   symbol,
		"side":     side,
		"qty":      strconv.FormatFloat(quantity, 'f', -1, 64),
	}, nil
}

// GetRFQConfig returns the caller's desk code, RFQ limits and the quoting parties RFQs can be sent to (RFQ Trading:
// Get RFQ Configuration)
func (e *Exchange) GetRFQConfig(ctx context.Context) (*RFQConfigResponse, error) {
	var resp *RFQConfigResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/rfq/config", nil, nil, &resp, getRFQConfigEPL)
}

// CancelRFQ cancels an RFQ by its ID or link ID; Bybit uses the ID when both are given (RFQ Trading: Cancel RFQ)
func (e *Exchange) CancelRFQ(ctx context.Context, rfqID, rfqLinkID string) (*CancelRFQResponse, error) {
	if rfqID == "" && rfqLinkID == "" {
		return nil, errRFQIdentifierRequired
	}
	body := map[string]any{}
	if rfqID != "" {
		body["rfqId"] = rfqID
	}
	if rfqLinkID != "" {
		body["rfqLinkId"] = rfqLinkID
	}
	var resp *CancelRFQResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/rfq/cancel-rfq", nil, body, &resp, cancelRFQEPL)
}

// CancelAllRFQs cancels every active RFQ of the caller and returns each cancellation's outcome; an inquiring party's
// cancellation also invalidates the RFQ's quotes (RFQ Trading: Cancel All RFQs)
func (e *Exchange) CancelAllRFQs(ctx context.Context) ([]RFQCancellationResponse, error) {
	var resp []RFQCancellationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/rfq/cancel-all-rfq", nil, nil, &resp, cancelAllRFQsEPL)
}

// AcceptNonLPQuote lets an RFQ accept quotes from parties other than liquidity providers (RFQ Trading: Accept non-LP
// Quote)
func (e *Exchange) AcceptNonLPQuote(ctx context.Context, rfqID string) (*AcceptNonLPQuoteResponse, error) {
	if rfqID == "" {
		return nil, errRFQIDRequired
	}
	var resp *AcceptNonLPQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/rfq/accept-other-quote", nil, map[string]any{"rfqId": rfqID}, &resp, acceptNonLPQuoteEPL)
}

// CreateRFQQuote quotes both directions of an RFQ the caller received (RFQ Trading: Create Quote)
func (e *Exchange) CreateRFQQuote(ctx context.Context, req *CreateRFQQuoteRequest) (*CreateRFQQuoteResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.RFQID == "" {
		return nil, errRFQIDRequired
	}
	if len(req.QuoteBuyList) == 0 && len(req.QuoteSellList) == 0 {
		return nil, errQuoteLegsRequired
	}
	body := map[string]any{"rfqId": req.RFQID}
	if req.QuoteLinkID != "" {
		body["quoteLinkId"] = req.QuoteLinkID
	}
	if req.Anonymous {
		body["anonymous"] = true
	}
	if req.ExpireIn != 0 {
		body["expireIn"] = req.ExpireIn
	}
	if len(req.QuoteBuyList) != 0 {
		legs, err := rfqQuoteLegsBody(req.QuoteBuyList)
		if err != nil {
			return nil, fmt.Errorf("%w: quoteBuyList", err)
		}
		body["quoteBuyList"] = legs
	}
	if len(req.QuoteSellList) != 0 {
		legs, err := rfqQuoteLegsBody(req.QuoteSellList)
		if err != nil {
			return nil, fmt.Errorf("%w: quoteSellList", err)
		}
		body["quoteSellList"] = legs
	}
	var resp *CreateRFQQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/rfq/create-quote", nil, body, &resp, createRFQQuoteEPL)
}

// rfqQuoteLegsBody returns the body of one direction of a quote; a leg without a price is sent without one, so a hedge
// leg takes the hedge price in the RFQ
func rfqQuoteLegsBody(legs []RFQQuoteLegRequest) ([]map[string]any, error) {
	body := make([]map[string]any, len(legs))
	for i := range legs {
		leg := &legs[i]
		if leg.Category == "" {
			return nil, fmt.Errorf("%w: leg %d", errCategoryNotSet, i)
		}
		if leg.Symbol == "" {
			return nil, fmt.Errorf("%w: leg %d", errSymbolMissing, i)
		}
		body[i] = map[string]any{"category": leg.Category, "symbol": leg.Symbol}
		if leg.Price != 0 {
			body[i]["price"] = strconv.FormatFloat(leg.Price, 'f', -1, 64)
		}
	}
	return body, nil
}

// ExecuteRFQQuote executes a quote on the caller's RFQ. Bybit processes it asynchronously, so the trade history and the
// execution stream confirm the outcome (RFQ Trading: Execute Quote)
func (e *Exchange) ExecuteRFQQuote(ctx context.Context, req *ExecuteRFQQuoteRequest) (*ExecuteRFQQuoteResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.RFQID == "" {
		return nil, errRFQIDRequired
	}
	if req.QuoteID == "" {
		return nil, errQuoteIDRequired
	}
	if req.QuoteSide == "" {
		return nil, order.ErrSideIsInvalid
	}
	body := map[string]any{
		"rfqId":     req.RFQID,
		"quoteId":   req.QuoteID,
		"quoteSide": req.QuoteSide,
	}
	if req.IsHedge {
		body["isHedge"] = true
	}
	var resp *ExecuteRFQQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/rfq/execute-quote", nil, body, &resp, executeRFQQuoteEPL)
}

// CancelRFQQuote cancels a quote by its ID, link ID or RFQ ID (RFQ Trading: Cancel Quote)
func (e *Exchange) CancelRFQQuote(ctx context.Context, req *CancelRFQQuoteRequest) (*CancelRFQQuoteResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.QuoteID == "" && req.QuoteLinkID == "" && req.RFQID == "" {
		return nil, errQuoteIdentifierRequired
	}
	body := map[string]any{}
	if req.QuoteID != "" {
		body["quoteId"] = req.QuoteID
	}
	if req.RFQID != "" {
		body["rfqId"] = req.RFQID
	}
	if req.QuoteLinkID != "" {
		body["quoteLinkId"] = req.QuoteLinkID
	}
	var resp *CancelRFQQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/rfq/cancel-quote", nil, body, &resp, cancelRFQQuoteEPL)
}

// CancelAllRFQQuotes cancels every active quote of the caller and returns each cancellation's outcome (RFQ Trading:
// Cancel All Quotes)
func (e *Exchange) CancelAllRFQQuotes(ctx context.Context) ([]RFQQuoteCancellationResponse, error) {
	var resp []RFQQuoteCancellationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/rfq/cancel-all-quotes", nil, nil, &resp, cancelAllRFQQuotesEPL)
}

// GetRFQsRealtime returns RFQs without the delay of the historical query (RFQ Trading: Get RFQs (real-time))
func (e *Exchange) GetRFQsRealtime(ctx context.Context, req *RFQsRealtimeRequest) (*RFQsRealtimeResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.RFQID != "" {
		params.Set("rfqId", req.RFQID)
	}
	if req.RFQLinkID != "" {
		params.Set("rfqLinkId", req.RFQLinkID)
	}
	if req.TraderType != "" {
		params.Set("traderType", req.TraderType)
	}
	var resp *RFQsRealtimeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/rfq/rfq-realtime", params, nil, &resp, getRFQsRealtimeEPL)
}

// GetRFQs returns historical RFQs, newest first (RFQ Trading: Get RFQs)
func (e *Exchange) GetRFQs(ctx context.Context, req *RFQsRequest) (*RFQsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.RFQID != "" {
		params.Set("rfqId", req.RFQID)
	}
	if req.RFQLinkID != "" {
		params.Set("rfqLinkId", req.RFQLinkID)
	}
	if req.TraderType != "" {
		params.Set("traderType", req.TraderType)
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
	var resp *RFQsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/rfq/rfq-list", params, nil, &resp, getRFQsEPL)
}

// GetRFQDetails returns historical RFQs with their active quotes and the caller's trade legs, newest first (RFQ
// Trading: Get RFQ Details)
func (e *Exchange) GetRFQDetails(ctx context.Context, req *RFQDetailsRequest) (*RFQDetailsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.RFQID != "" {
		params.Set("rfqId", req.RFQID)
	}
	if req.RFQLinkID != "" {
		params.Set("rfqLinkId", req.RFQLinkID)
	}
	if req.Status != "" {
		params.Set("status", req.Status)
	}
	if req.TraderType != "" {
		params.Set("traderType", req.TraderType)
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
	var resp *RFQDetailsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/rfq/rfq-detail-list", params, nil, &resp, getRFQDetailsEPL)
}

// GetRFQQuotesRealtime returns the quotes the caller sent or received that are not in a final state, without the delay
// of the historical query (RFQ Trading: Get Quotes (real-time))
func (e *Exchange) GetRFQQuotesRealtime(ctx context.Context, req *RFQQuotesRealtimeRequest) (*RFQQuotesRealtimeResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.RFQID != "" {
		params.Set("rfqId", req.RFQID)
	}
	if req.QuoteID != "" {
		params.Set("quoteId", req.QuoteID)
	}
	if req.QuoteLinkID != "" {
		params.Set("quoteLinkId", req.QuoteLinkID)
	}
	if req.TraderType != "" {
		params.Set("traderType", req.TraderType)
	}
	var resp *RFQQuotesRealtimeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/rfq/quote-realtime", params, nil, &resp, getRFQQuotesRealtimeEPL)
}

// GetRFQQuotes returns historical quotes, newest first (RFQ Trading: Get Quotes)
func (e *Exchange) GetRFQQuotes(ctx context.Context, req *RFQQuotesRequest) (*RFQQuotesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.RFQID != "" {
		params.Set("rfqId", req.RFQID)
	}
	if req.QuoteID != "" {
		params.Set("quoteId", req.QuoteID)
	}
	if req.QuoteLinkID != "" {
		params.Set("quoteLinkId", req.QuoteLinkID)
	}
	if req.TraderType != "" {
		params.Set("traderType", req.TraderType)
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
	var resp *RFQQuotesResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/rfq/quote-list", params, nil, &resp, getRFQQuotesEPL)
}

// GetRFQTradeHistory returns the RFQ trades the caller took part in (RFQ Trading: Get Trade History)
func (e *Exchange) GetRFQTradeHistory(ctx context.Context, req *RFQTradeHistoryRequest) (*RFQTradeHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.RFQID != "" {
		params.Set("rfqId", req.RFQID)
	}
	if req.RFQLinkID != "" {
		params.Set("rfqLinkId", req.RFQLinkID)
	}
	if req.QuoteID != "" {
		params.Set("quoteId", req.QuoteID)
	}
	if req.QuoteLinkID != "" {
		params.Set("quoteLinkId", req.QuoteLinkID)
	}
	if req.TraderType != "" {
		params.Set("traderType", req.TraderType)
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
	var resp *RFQTradeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/rfq/trade-list", params, nil, &resp, getRFQTradeHistoryEPL)
}

// GetRFQPublicTrades returns recently executed RFQs without party information. Despite its name the endpoint requires
// a signed request (RFQ Trading: Get Public Trades)
func (e *Exchange) GetRFQPublicTrades(ctx context.Context, req *RFQPublicTradesRequest) (*RFQPublicTradesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
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
	var resp *RFQPublicTradesResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/rfq/public-trades", params, nil, &resp, getRFQPublicTradesEPL)
}
