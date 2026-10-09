package kraken

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// GetFuturesExecutionEvents calls Get execution events, listing the account's executions a page at a time. A nil
// request lists the latest page across every market
func (e *Exchange) GetFuturesExecutionEvents(ctx context.Context, req *FuturesHistoryExecutionEventsRequest) (*FuturesHistoryExecutionEventsResponse, error) {
	if req == nil {
		req = new(FuturesHistoryExecutionEventsRequest)
	}
	params, err := e.futuresHistoryAccountParams(req.Pair, req.Since, req.Before, req.Ascending, req.ContinuationToken, req.Count)
	if err != nil {
		return nil, err
	}
	var resp *FuturesHistoryExecutionEventsResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/history/v3/executions", params, nil, &resp)
}

// GetFuturesOrderEvents calls Get order events, listing the account's order events a page at a time. A nil request
// lists the latest page across every market
func (e *Exchange) GetFuturesOrderEvents(ctx context.Context, req *FuturesHistoryOrderEventsRequest) (*FuturesHistoryOrderEventsResponse, error) {
	if req == nil {
		req = new(FuturesHistoryOrderEventsRequest)
	}
	params, err := e.futuresHistoryAccountParams(req.Pair, req.Since, req.Before, req.Ascending, req.ContinuationToken, req.Count)
	if err != nil {
		return nil, err
	}
	if req.Opened != nil {
		params.Set("opened", strconv.FormatBool(*req.Opened))
	}
	if req.Closed != nil {
		params.Set("closed", strconv.FormatBool(*req.Closed))
	}
	var resp *FuturesHistoryOrderEventsResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/history/v3/orders", params, nil, &resp)
}

// GetFuturesTriggerEvents calls Get trigger events, listing the account's trigger order events a page at a time. A nil
// request lists the latest page across every market
func (e *Exchange) GetFuturesTriggerEvents(ctx context.Context, req *FuturesHistoryTriggerEventsRequest) (*FuturesHistoryTriggerEventsResponse, error) {
	if req == nil {
		req = new(FuturesHistoryTriggerEventsRequest)
	}
	params, err := e.futuresHistoryAccountParams(req.Pair, req.Since, req.Before, req.Ascending, req.ContinuationToken, req.Count)
	if err != nil {
		return nil, err
	}
	if req.Opened != nil {
		params.Set("opened", strconv.FormatBool(*req.Opened))
	}
	if req.Closed != nil {
		params.Set("closed", strconv.FormatBool(*req.Closed))
	}
	var resp *FuturesHistoryTriggerEventsResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/history/v3/triggers", params, nil, &resp)
}

// GetFuturesPositionEvents calls Get position update events, listing the changes to the account's positions a page at
// a time. A nil request lists the latest page across every market
func (e *Exchange) GetFuturesPositionEvents(ctx context.Context, req *FuturesHistoryPositionEventsRequest) (*FuturesHistoryPositionEventsResponse, error) {
	if req == nil {
		req = new(FuturesHistoryPositionEventsRequest)
	}
	params, err := e.futuresHistoryAccountParams(req.Pair, req.Since, req.Before, req.Ascending, req.ContinuationToken, req.Count)
	if err != nil {
		return nil, err
	}
	// Kraken ignores a filter sent as false, so only set filters are sent
	for name, include := range map[string]bool{
		"opened":              req.Opened,
		"closed":              req.Closed,
		"increased":           req.Increased,
		"decreased":           req.Decreased,
		"reversed":            req.Reversed,
		"no_change":           req.NoChange,
		"trades":              req.Trades,
		"funding_realization": req.FundingRealisation,
		"settlement":          req.Settlement,
	} {
		if include {
			params.Set(name, "true")
		}
	}
	var resp *FuturesHistoryPositionEventsResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/history/v3/positions", params, nil, &resp)
}

// GetFuturesAccountLog calls Get account log, listing the account's balance and position changes. A nil request lists
// the latest 500 entries
func (e *Exchange) GetFuturesAccountLog(ctx context.Context, req *FuturesHistoryAccountLogRequest) (*FuturesHistoryAccountLogResponse, error) {
	if req == nil {
		req = new(FuturesHistoryAccountLogRequest)
	}
	if req.FromID != 0 && req.ToID != 0 && req.FromID > req.ToID {
		return nil, fmt.Errorf("%w: from %d is after to %d", errInvalidAccountLogRange, req.FromID, req.ToID)
	}
	params, err := futuresHistoryPagingParams(req.Since, req.Before, req.Ascending, "", req.Count)
	if err != nil {
		return nil, err
	}
	if req.FromID != 0 {
		params.Set("from", strconv.FormatUint(req.FromID, 10))
	}
	if req.ToID != 0 {
		params.Set("to", strconv.FormatUint(req.ToID, 10))
	}
	for _, entryType := range req.EntryTypes {
		params.Add("info", entryType)
	}
	if req.ConversionDetails {
		params.Set("conversion_details", "true")
	}
	var resp *FuturesHistoryAccountLogResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/history/v3/account-log", params, nil, &resp)
}

// GetFuturesAccountLogCSV calls Account log (CSV), returning up to the account's most recent 500,000 log entries as a
// CSV file. A nil request leaves out the conversion details
func (e *Exchange) GetFuturesAccountLogCSV(ctx context.Context, req *FuturesHistoryAccountLogCSVRequest) ([]byte, error) {
	params := url.Values{}
	if req != nil && req.ConversionDetails {
		params.Set("conversion_details", "true")
	}
	return e.sendFuturesAuthenticatedRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/history/v3/accountlogcsv", params, nil, nil, true)
}

// GetFuturesPublicExecutionEvents calls Get public execution events, listing a market's trades a page at a time
func (e *Exchange) GetFuturesPublicExecutionEvents(ctx context.Context, req *FuturesHistoryMarketEventsRequest) (*FuturesHistoryPublicExecutionEventsResponse, error) {
	symbol, params, err := e.futuresHistoryMarketParams(req)
	if err != nil {
		return nil, err
	}
	var resp *FuturesHistoryPublicExecutionEventsResponse
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFuturesSupplementary, common.EncodeURLValues("/history/v3/market/"+url.PathEscape(symbol)+"/executions", params), &resp)
}

// GetFuturesPublicOrderEvents calls Get public order events, listing a market's order events a page at a time
func (e *Exchange) GetFuturesPublicOrderEvents(ctx context.Context, req *FuturesHistoryMarketEventsRequest) (*FuturesHistoryPublicOrderEventsResponse, error) {
	symbol, params, err := e.futuresHistoryMarketParams(req)
	if err != nil {
		return nil, err
	}
	var resp *FuturesHistoryPublicOrderEventsResponse
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFuturesSupplementary, common.EncodeURLValues("/history/v3/market/"+url.PathEscape(symbol)+"/orders", params), &resp)
}

// GetFuturesPublicMarkPriceEvents calls Get public mark price events, listing a market's mark prices a page at a time
func (e *Exchange) GetFuturesPublicMarkPriceEvents(ctx context.Context, req *FuturesHistoryMarketEventsRequest) (*FuturesHistoryPublicMarkPriceEventsResponse, error) {
	symbol, params, err := e.futuresHistoryMarketParams(req)
	if err != nil {
		return nil, err
	}
	var resp *FuturesHistoryPublicMarkPriceEventsResponse
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFuturesSupplementary, common.EncodeURLValues("/history/v3/market/"+url.PathEscape(symbol)+"/price", params), &resp)
}

// futuresHistoryAccountParams returns the parameters every account history event listing takes, filtering by market
// when pair is set
func (e *Exchange) futuresHistoryAccountParams(pair currency.Pair, since, before time.Time, ascending bool, continuationToken string, count uint64) (url.Values, error) {
	params, err := futuresHistoryPagingParams(since, before, ascending, continuationToken, count)
	if err != nil {
		return nil, err
	}
	if !pair.IsEmpty() {
		symbol, err := e.FormatSymbol(pair, asset.Futures)
		if err != nil {
			return nil, err
		}
		params.Set("tradeable", symbol)
	}
	return params, nil
}

// futuresHistoryMarketParams returns the market symbol a public history listing takes in its path and its paging
// parameters
func (e *Exchange) futuresHistoryMarketParams(req *FuturesHistoryMarketEventsRequest) (string, url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return "", nil, err
	}
	if req.Pair.IsEmpty() {
		return "", nil, currency.ErrCurrencyPairEmpty
	}
	params, err := futuresHistoryPagingParams(req.Since, req.Before, req.Ascending, req.ContinuationToken, req.Count)
	if err != nil {
		return "", nil, err
	}
	symbol, err := e.FormatSymbol(req.Pair, asset.Futures)
	if err != nil {
		return "", nil, err
	}
	return symbol, params, nil
}

// futuresHistoryPagingParams returns the paging parameters the history listings share. Kraken returns nothing rather
// than an error for a window that ends before it starts, so one is rejected here
func futuresHistoryPagingParams(since, before time.Time, ascending bool, continuationToken string, count uint64) (url.Values, error) {
	if !since.IsZero() && !before.IsZero() && since.After(before) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if !since.IsZero() {
		params.Set("since", strconv.FormatInt(since.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if ascending {
		params.Set("sort", "asc")
	}
	if continuationToken != "" {
		params.Set("continuation_token", continuationToken)
	}
	if count != 0 {
		params.Set("count", strconv.FormatUint(count, 10))
	}
	return params, nil
}
