package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var errSizeOrPriceIsRequired = errors.New("either size or price is required")

// PlaceSpreadOrder places a spread order
func (e *Exchange) PlaceSpreadOrder(ctx context.Context, arg *PlaceSpreadOrderRequest) (*SpreadOrderResponse, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	var resp *SpreadOrderResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, placeSpreadOrderEPL, http.MethodPost, "sprd/order", arg, &resp, request.AuthenticatedRequest); err != nil {
		if resp != nil {
			// OKX's top-level reply only says all operations failed; the row carries the code and reason, such as
			// 51008 for a balance reject
			err = common.AppendError(err, resp.Error())
		}
		return nil, err
	}
	return resp, nil
}

// CancelSpreadOrder cancels a pending spread order
func (e *Exchange) CancelSpreadOrder(ctx context.Context, arg *CancelSpreadOrderRequest) (*SpreadOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *SpreadOrderResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, cancelSpreadOrderEPL, http.MethodPost, "sprd/cancel-order", arg, &resp, request.AuthenticatedRequest); err != nil {
		if resp != nil {
			err = common.AppendError(err, resp.Error())
		}
		return nil, err
	}
	return resp, nil
}

// CancelAllSpreadOrders cancels the pending orders of a spread, or of every spread when the request's SpreadID is
// empty
func (e *Exchange) CancelAllSpreadOrders(ctx context.Context, arg *CancelAllSpreadOrdersRequest) (*CancelAllSpreadOrdersResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	var resp *CancelAllSpreadOrdersResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllSpreadOrderEPL, http.MethodPost, "sprd/mass-cancel", arg, &resp, request.AuthenticatedRequest)
}

// AmendSpreadOrder amends the size or price of a pending spread order
func (e *Exchange) AmendSpreadOrder(ctx context.Context, arg *AmendSpreadOrderRequest) (*SpreadOrderResponse, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	var resp *SpreadOrderResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, amendSpreadOrderEPL, http.MethodPost, "sprd/amend-order", arg, &resp, request.AuthenticatedRequest); err != nil {
		if resp != nil {
			err = common.AppendError(err, resp.Error())
		}
		return nil, err
	}
	return resp, nil
}

// Validate checks the amend request names an order and a new size or price
func (arg *AmendSpreadOrderRequest) Validate() error {
	if arg == nil {
		return fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	if arg.NewSize == 0 && arg.NewPrice == nil {
		return errSizeOrPriceIsRequired
	}
	return nil
}

// GetSpreadOrderDetails retrieves a spread order
func (e *Exchange) GetSpreadOrderDetails(ctx context.Context, arg *GetSpreadOrderDetailsRequest) (*SpreadOrderDetailResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if arg.OrderID != "" {
		params.Set("ordId", arg.OrderID)
	}
	if arg.ClientOrderID != "" {
		params.Set("clOrdId", arg.ClientOrderID)
	}
	var resp *SpreadOrderDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadOrderDetailsEPL, http.MethodGet, common.EncodeURLValues("sprd/order", params), nil, &resp, request.AuthenticatedRequest)
}

// GetActiveSpreadOrders retrieves the account's pending spread orders, newest first
func (e *Exchange) GetActiveSpreadOrders(ctx context.Context, arg *GetActiveSpreadOrdersRequest) ([]SpreadOrderDetailResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.SpreadID != "" {
		params.Set("sprdId", arg.SpreadID)
	}
	if arg.OrderType != "" {
		params.Set("ordType", arg.OrderType)
	}
	if arg.State != "" {
		params.Set("state", arg.State)
	}
	if arg.BeginID != "" {
		params.Set("beginId", arg.BeginID)
	}
	if arg.EndID != "" {
		params.Set("endId", arg.EndID)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []SpreadOrderDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getActiveSpreadOrdersEPL, http.MethodGet, common.EncodeURLValues("sprd/orders-pending", params), nil, &resp, request.AuthenticatedRequest)
}

// GetCompletedSpreadOrdersLast21Days retrieves the spread orders completed in the last 21 days, newest first.
// Cancelled orders that filled nothing are kept for 2 hours
func (e *Exchange) GetCompletedSpreadOrdersLast21Days(ctx context.Context, arg *GetCompletedSpreadOrdersRequest) ([]SpreadOrderDetailResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params, err := spreadOrderHistoryParams(arg.SpreadID, arg.OrderType, arg.State, arg.BeginID, arg.EndID, arg.BeginTime, arg.EndTime, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []SpreadOrderDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadOrders21DaysEPL, http.MethodGet, common.EncodeURLValues("sprd/orders-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetCompletedSpreadOrdersLast3Months retrieves the spread orders completed in the last 3 months, newest first. Recent
// orders can lag behind GetCompletedSpreadOrdersLast21Days while they reach the archive
func (e *Exchange) GetCompletedSpreadOrdersLast3Months(ctx context.Context, arg *GetCompletedSpreadOrdersArchiveRequest) ([]SpreadOrderDetailResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params, err := spreadOrderHistoryParams(arg.SpreadID, arg.OrderType, arg.State, arg.BeginID, arg.EndID, arg.BeginTime, arg.EndTime, arg.Limit)
	if err != nil {
		return nil, err
	}
	if arg.InstrumentType != "" {
		params.Set("instType", arg.InstrumentType)
	}
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	var resp []SpreadOrderDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadOrders3MonthsEPL, http.MethodGet, common.EncodeURLValues("sprd/orders-history-archive", params), nil, &resp, request.AuthenticatedRequest)
}

// spreadOrderHistoryParams returns the query filters both completed spread order listings take
func spreadOrderHistoryParams(spreadID, orderType, state, beginID, endID string, beginTime, endTime time.Time, limit uint64) (url.Values, error) {
	if !beginTime.IsZero() && !endTime.IsZero() && beginTime.After(endTime) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if spreadID != "" {
		params.Set("sprdId", spreadID)
	}
	if orderType != "" {
		params.Set("ordType", orderType)
	}
	if state != "" {
		params.Set("state", state)
	}
	if beginID != "" {
		params.Set("beginId", beginID)
	}
	if endID != "" {
		params.Set("endId", endID)
	}
	if !beginTime.IsZero() {
		params.Set("begin", strconv.FormatInt(beginTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	return params, nil
}

// GetSpreadTradesOfLast7Days retrieves the account's spread trades of the last 7 days, newest first
func (e *Exchange) GetSpreadTradesOfLast7Days(ctx context.Context, arg *GetSpreadTradesRequest) ([]SpreadTradeResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if !arg.BeginTime.IsZero() && !arg.EndTime.IsZero() && arg.BeginTime.After(arg.EndTime) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if arg.SpreadID != "" {
		params.Set("sprdId", arg.SpreadID)
	}
	if arg.TradeID != "" {
		params.Set("tradeId", arg.TradeID)
	}
	if arg.OrderID != "" {
		params.Set("ordId", arg.OrderID)
	}
	if arg.BeginID != "" {
		params.Set("beginId", arg.BeginID)
	}
	if arg.EndID != "" {
		params.Set("endId", arg.EndID)
	}
	if !arg.BeginTime.IsZero() {
		params.Set("begin", strconv.FormatInt(arg.BeginTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("end", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []SpreadTradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadOrderTradesEPL, http.MethodGet, common.EncodeURLValues("sprd/trades", params), nil, &resp, request.AuthenticatedRequest)
}

// GetPublicSpreads retrieves the spreads matching the request's filters
func (e *Exchange) GetPublicSpreads(ctx context.Context, arg *GetPublicSpreadsRequest) ([]SpreadInstrumentResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if !arg.BaseCurrency.IsEmpty() {
		params.Set("baseCcy", arg.BaseCurrency.Upper().String())
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.SpreadID != "" {
		params.Set("sprdId", arg.SpreadID)
	}
	if arg.State != "" {
		params.Set("state", arg.State)
	}
	var resp []SpreadInstrumentResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadsEPL, http.MethodGet, common.EncodeURLValues("sprd/spreads", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetPublicSpreadOrderBooks retrieves a spread's order book
func (e *Exchange) GetPublicSpreadOrderBooks(ctx context.Context, arg *GetPublicSpreadOrderBooksRequest) ([]SpreadOrderBookResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.SpreadID == "" {
		return nil, fmt.Errorf("%w, spread ID missing", errMissingInstrumentID)
	}
	params := url.Values{}
	params.Set("sprdId", arg.SpreadID)
	if arg.Depth != 0 {
		params.Set("sz", strconv.FormatUint(arg.Depth, 10))
	}
	var resp []SpreadOrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadOrderbookEPL, http.MethodGet, common.EncodeURLValues("sprd/books", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetPublicSpreadTickers retrieves a spread's latest trade, best bid and ask, and its trading over the last 24 hours
func (e *Exchange) GetPublicSpreadTickers(ctx context.Context, spreadID string) ([]SpreadTickerResponse, error) {
	if spreadID == "" {
		return nil, fmt.Errorf("%w, spread ID is required", errMissingInstrumentID)
	}
	params := url.Values{}
	params.Set("sprdId", spreadID)
	var resp []SpreadTickerResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadTickerEPL, http.MethodGet, common.EncodeURLValues("market/sprd-ticker", params), nil, &resp, request.UnauthenticatedRequest); err != nil {
		return nil, err
	}
	// market/sprd-ticker serves a cached snapshot whose bid, ask and timestamp lag the spread's book and can move
	// backwards between polls, so the top of book comes from sprd/books while the last and 24 hour figures stay
	books, err := e.GetPublicSpreadOrderBooks(ctx, &GetPublicSpreadOrderBooksRequest{SpreadID: spreadID, Depth: 1})
	if err != nil {
		return nil, err
	}
	if len(books) == 0 {
		return resp, nil
	}
	for i := range resp {
		overlaySpreadBook(&resp[i], &books[0])
	}
	return resp, nil
}

// overlaySpreadBook replaces a spread ticker's cached top of book with the book's, and advances its timestamp to the
// book's when that is newer. A side the book lacks is cleared rather than kept from the cache, which could cross the
// fresh side
func overlaySpreadBook(tk *SpreadTickerResponse, book *SpreadOrderBookResponse) {
	tk.BidPrice, tk.BidSize = spreadBookTop(book.Bids)
	tk.AskPrice, tk.AskSize = spreadBookTop(book.Asks)
	if book.Timestamp.Time().After(tk.Timestamp.Time()) {
		tk.Timestamp = book.Timestamp
	}
}

// spreadBookTop returns the price and size of a book side's best level, or zeros when the side is empty
func spreadBookTop(levels [][]types.Number) (price, size types.Number) {
	if len(levels) == 0 || len(levels[0]) < 2 {
		return 0, 0
	}
	return levels[0][0], levels[0][1]
}

// GetPublicSpreadTrades retrieves up to 500 of the latest public trades of a spread, or of every spread when the
// request's SpreadID is empty, newest first
func (e *Exchange) GetPublicSpreadTrades(ctx context.Context, arg *GetPublicSpreadTradesRequest) ([]SpreadPublicTradeResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.SpreadID != "" {
		params.Set("sprdId", arg.SpreadID)
	}
	var resp []SpreadPublicTradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadPublicTradesEPL, http.MethodGet, common.EncodeURLValues("sprd/public-trades", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetSpreadCandlesticks retrieves a spread's candlesticks from among its latest 1,440
func (e *Exchange) GetSpreadCandlesticks(ctx context.Context, arg *GetSpreadCandlesticksRequest) ([]SpreadCandlestickResponse, error) {
	params, err := spreadCandlestickParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []SpreadCandlestickResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadCandlesticksEPL, http.MethodGet, common.EncodeURLValues("market/sprd-candles", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetSpreadCandlesticksHistory retrieves a spread's candlesticks from recent years
func (e *Exchange) GetSpreadCandlesticksHistory(ctx context.Context, arg *GetSpreadCandlesticksRequest) ([]SpreadCandlestickResponse, error) {
	params, err := spreadCandlestickParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []SpreadCandlestickResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadCandlesticksHistoryEPL, http.MethodGet, common.EncodeURLValues("market/sprd-history-candles", params), nil, &resp, request.UnauthenticatedRequest)
}

// spreadCandlestickParams validates a spread candlestick request and returns its query
func spreadCandlestickParams(arg *GetSpreadCandlesticksRequest) (url.Values, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.SpreadID == "" {
		return nil, fmt.Errorf("%w, spread ID is required", errMissingInstrumentID)
	}
	if !arg.Before.IsZero() && !arg.After.IsZero() && arg.Before.After(arg.After) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("sprdId", arg.SpreadID)
	if arg.Interval != 0 {
		// The spread candle endpoints list fewer bars than the market ones
		switch arg.Interval {
		case kline.OneMin, kline.ThreeMin, kline.FiveMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.TwoHour, kline.FourHour,
			kline.SixHour, kline.TwelveHour, kline.OneDay, kline.TwoDay, kline.ThreeDay, kline.OneWeek, kline.OneMonth, kline.ThreeMonth:
			params.Set("bar", IntervalFromString(arg.Interval, true))
		default:
			return nil, fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, arg.Interval)
		}
	}
	if !arg.Before.IsZero() {
		params.Set("before", strconv.FormatInt(arg.Before.UnixMilli(), 10))
	}
	if !arg.After.IsZero() {
		params.Set("after", strconv.FormatInt(arg.After.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	return params, nil
}

// CancelAllSpreadOrdersAfterCountdown cancels every pending spread order once the timeout elapses without another
// call; the timeout is whole seconds from 10 to 120, or 0 to disable the countdown
func (e *Exchange) CancelAllSpreadOrdersAfterCountdown(ctx context.Context, timeout time.Duration) (*CancelAllSpreadOrdersAfterCountdownResponse, error) {
	if timeout != 0 && (timeout < 10*time.Second || timeout > 120*time.Second) {
		return nil, fmt.Errorf("%w: %s must be 0 or from 10s to 120s", errCountdownTimeoutRequired, timeout)
	}
	arg := &CancelAllSpreadOrdersAfterCountdownRequest{Timeout: uint64(timeout / time.Second)} //nolint:gosec // timeout is checked above to be 0 or from 10s to 120s
	var resp *CancelAllSpreadOrdersAfterCountdownResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllSpreadOrdersAfterEPL, http.MethodPost, "sprd/cancel-all-after", arg, &resp, request.AuthenticatedRequest)
}
