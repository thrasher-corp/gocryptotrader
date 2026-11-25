package lbank

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
)

const (
	maxOrdersPageLength  = 200
	maxTransactionsLimit = 100
	// transactionHistoryMaxWindow is the longest time range a transaction history request may cover
	transactionHistoryMaxWindow = 48 * time.Hour
	// transactionHistoryTimeFormat is the format LBank reads a transaction history request's times in, at UTC+8
	transactionHistoryTimeFormat = time.DateTime
)

// orderTypes holds the order types LBank accepts: limit orders as the side alone, and market, post only (maker),
// immediate or cancel and fill or kill orders as the side and a suffix
var orderTypes = []string{
	"buy", "sell",
	"buy_market", "sell_market",
	"buy_maker", "sell_maker",
	"buy_ioc", "sell_ioc",
	"buy_fok", "sell_fok",
}

// PlaceTestOrder validates an order as PlaceOrder would without placing it
func (e *Exchange) PlaceTestOrder(ctx context.Context, arg *PlaceOrderRequest) (*PlaceOrderResponse, error) {
	return e.placeOrder(ctx, "/v2/supplement/create_order_test.do", arg)
}

// PlaceOrder places an order
func (e *Exchange) PlaceOrder(ctx context.Context, arg *PlaceOrderRequest) (*PlaceOrderResponse, error) {
	return e.placeOrder(ctx, "/v2/supplement/create_order.do", arg)
}

func (e *Exchange) placeOrder(ctx context.Context, path string, arg *PlaceOrderRequest) (*PlaceOrderResponse, error) {
	params, err := e.placeOrderParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *PlaceOrderResponse
	if err := e.SendAuthHTTPRequest(ctx, orderEPL, path, params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// placeOrderParams validates an order and returns its parameters. A market buy's Price is the quote amount to spend
// and it takes no Amount; a market sell takes no Price
func (e *Exchange) placeOrderParams(arg *PlaceOrderRequest) (url.Values, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if !slices.Contains(orderTypes, arg.Type) {
		return nil, fmt.Errorf("%w: %q", errInvalidOrderType, arg.Type)
	}
	if arg.Type != "sell_market" && arg.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	if arg.Type != "buy_market" && arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params, err := e.symbolParams(arg.Pair)
	if err != nil {
		return nil, err
	}
	params.Set("type", arg.Type)
	if arg.Price > 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.Amount > 0 {
		params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	}
	if arg.CustomID != "" {
		params.Set("custom_id", arg.CustomID)
	}
	if arg.Window > 0 {
		params.Set("window", strconv.FormatInt(arg.Window.Milliseconds(), 10))
	}
	return params, nil
}

// CancelExistingOrder cancels an order by its order ID or client order ID
func (e *Exchange) CancelExistingOrder(ctx context.Context, arg *OrderReferenceRequest) (*CancelOrderResponse, error) {
	params, err := e.orderReferenceParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *CancelOrderResponse
	if err := e.SendAuthHTTPRequest(ctx, orderEPL, "/v2/supplement/cancel_order.do", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelPairOrders cancels every open order on a pair, returning the orders it cancels
func (e *Exchange) CancelPairOrders(ctx context.Context, pair currency.Pair) ([]CancelOrderResponse, error) {
	params, err := e.symbolParams(pair)
	if err != nil {
		return nil, err
	}
	var resp []CancelOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, orderEPL, "/v2/supplement/cancel_order_by_symbol.do", params, &resp)
}

// QueryOrder returns an order in any state by its order ID or client order ID
func (e *Exchange) QueryOrder(ctx context.Context, arg *OrderReferenceRequest) (*OrderResponse, error) {
	params, err := e.orderReferenceParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *OrderResponse
	if err := e.SendAuthHTTPRequest(ctx, defaultEPL, "/v2/spot/trade/orders_info.do", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetOpenOrders returns a page of a pair's open orders
func (e *Exchange) GetOpenOrders(ctx context.Context, arg *OrdersPageRequest) (*OrdersPageResponse, error) {
	params, err := e.ordersPageParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *OrdersPageResponse
	if err := e.SendAuthHTTPRequest(ctx, defaultEPL, "/v2/supplement/orders_info_no_deal.do", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// QueryOrderHistory returns a page of a pair's orders placed in the last 24 hours. Without a status LBank returns the
// cancelled and filled orders
func (e *Exchange) QueryOrderHistory(ctx context.Context, arg *OrdersPageRequest) (*OrdersPageResponse, error) {
	params, err := e.ordersPageParams(arg)
	if err != nil {
		return nil, err
	}
	if arg.Status != "" {
		params.Set("status", arg.Status)
	}
	var resp *OrdersPageResponse
	if err := e.SendAuthHTTPRequest(ctx, defaultEPL, "/v2/spot/trade/orders_info_history.do", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAccountInformation returns the account's spot balances and permissions
func (e *Exchange) GetAccountInformation(ctx context.Context) (*AccountInformationResponse, error) {
	var resp *AccountInformationResponse
	if err := e.SendAuthHTTPRequest(ctx, defaultEPL, "/v2/supplement/user_info_account.do", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetTransactionHistory returns the account's trades on a pair. LBank defaults to trades since the start of yesterday,
// UTC+8, and a time range may cover 2 days at most
func (e *Exchange) GetTransactionHistory(ctx context.Context, arg *TransactionHistoryRequest) ([]TransactionResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Limit > maxTransactionsLimit {
		return nil, fmt.Errorf("%w: limit %d must be at most %d", errLimitOutOfRange, arg.Limit, maxTransactionsLimit)
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
		if arg.EndTime.Sub(arg.StartTime) > transactionHistoryMaxWindow {
			return nil, fmt.Errorf("%w: %s exceeds %s", errTimeRangeTooLong, arg.EndTime.Sub(arg.StartTime), transactionHistoryMaxWindow)
		}
	}
	params, err := e.symbolParams(arg.Pair)
	if err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", arg.StartTime.In(beijingTime).Format(transactionHistoryTimeFormat))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", arg.EndTime.In(beijingTime).Format(transactionHistoryTimeFormat))
	}
	if arg.FromID != "" {
		params.Set("fromId", arg.FromID)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []TransactionResponse
	return resp, e.SendAuthHTTPRequest(ctx, defaultEPL, "/v2/supplement/transaction_history.do", params, &resp)
}

// orderReferenceParams returns the parameters naming an order by its order ID or client order ID
func (e *Exchange) orderReferenceParams(arg *OrderReferenceRequest) (url.Values, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, errOrderIDRequired
	}
	params, err := e.symbolParams(arg.Pair)
	if err != nil {
		return nil, err
	}
	if arg.OrderID != "" {
		params.Set("orderId", arg.OrderID)
	}
	if arg.ClientOrderID != "" {
		params.Set("origClientOrderId", arg.ClientOrderID)
	}
	return params, nil
}

// ordersPageParams returns the parameters of an orders page request, which asks for the first page of 200 orders by
// default
func (e *Exchange) ordersPageParams(arg *OrdersPageRequest) (url.Values, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.PageLength > maxOrdersPageLength {
		return nil, fmt.Errorf("%w: page length %d must be at most %d", errLimitOutOfRange, arg.PageLength, maxOrdersPageLength)
	}
	params, err := e.symbolParams(arg.Pair)
	if err != nil {
		return nil, err
	}
	page, pageLength := arg.CurrentPage, arg.PageLength
	if page == 0 {
		page = 1
	}
	if pageLength == 0 {
		pageLength = maxOrdersPageLength
	}
	params.Set("current_page", strconv.FormatUint(page, 10))
	params.Set("page_length", strconv.FormatUint(pageLength, 10))
	return params, nil
}
