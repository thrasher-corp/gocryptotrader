package gateio

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// ListSpotCurrencies retrieves every spot currency, including the deposit and withdrawal status of each of its chains
func (e *Exchange) ListSpotCurrencies(ctx context.Context) ([]*CurrencyInfoResponse, error) {
	var resp []*CurrencyInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrenciesSpotEPL, "spot/currencies", &resp)
}

// GetCurrencyDetail retrieves the details of a single spot currency
func (e *Exchange) GetCurrencyDetail(ctx context.Context, ccy currency.Code) (*CurrencyInfoResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *CurrencyInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrenciesSpotEPL, "spot/currencies/"+ccy.String(), &resp)
}

// ListSpotCurrencyPairs retrieves every spot currency pair
func (e *Exchange) ListSpotCurrencyPairs(ctx context.Context) ([]*CurrencyPairDetailResponse, error) {
	var resp []*CurrencyPairDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicListCurrencyPairsSpotEPL, "spot/currency_pairs", &resp)
}

// GetCurrencyPairDetail retrieves the details of a single spot currency pair
func (e *Exchange) GetCurrencyPairDetail(ctx context.Context, currencyPair currency.Pair) (*CurrencyPairDetailResponse, error) {
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *CurrencyPairDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrencyPairDetailSpotEPL, "spot/currency_pairs/"+currencyPair.String(), &resp)
}

// GetTickers retrieves ticker information for every currency pair, or only the supplied pair when one is set
// timezone selects the 24 hour change windows returned: utc0, utc8 or all; Gate's default applies when it is empty
// lowest_size and highest_size are only returned when a currency pair is supplied
func (e *Exchange) GetTickers(ctx context.Context, currencyPair currency.Pair, timezone string) ([]*TickerResponse, error) {
	params := url.Values{}
	if !currencyPair.IsEmpty() {
		params.Set("currency_pair", currencyPair.String())
	}
	switch timezone {
	case "":
	case utc0TimeZone, utc8TimeZone, allTimeZones:
		params.Set("timezone", timezone)
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidTimezone, timezone)
	}
	var tickers []*TickerResponse
	return tickers, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTickersSpotEPL, common.EncodeURLValues("spot/tickers", params), &tickers)
}

// GetTicker retrieves a single ticker information for a currency pair.
func (e *Exchange) GetTicker(ctx context.Context, currencyPair currency.Pair, timezone string) (*TickerResponse, error) {
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	tickers, err := e.GetTickers(ctx, currencyPair, timezone)
	if err != nil {
		return nil, err
	}
	if len(tickers) > 0 {
		return tickers[0], nil
	}
	return nil, fmt.Errorf("%w for currency pair %v", errNoTickerData, currencyPair)
}

// GetOrderbook returns the orderbook for the supplied currency pair; interval merges price levels by the given
// precision and withOrderbookID requests the orderbook update ID
func (e *Exchange) GetOrderbook(ctx context.Context, pairString currency.Pair, interval string, limit uint64, withOrderbookID bool) (*OrderbookResponse, error) {
	if pairString.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if interval != "" {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("currency_pair", pairString.String())
	params.Set("with_id", strconv.FormatBool(withOrderbookID))
	var response *SpotOrderbookResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookSpotEPL, common.EncodeURLValues("spot/order_book", params), &response); err != nil {
		return nil, err
	}
	return response.MakeOrderbook(), nil
}

// GetMarketTrades retrieves public market trades for a currency pair, filtered either by time range or by lastID
// Gate ignores the time range when lastID is set and only applies reverse alongside lastID
func (e *Exchange) GetMarketTrades(ctx context.Context, pairString currency.Pair, lastID string, reverse bool, startTime, endTime time.Time, page, limit uint64) ([]*TradeResponse, error) {
	if pairString.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", pairString.String())
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if reverse {
		params.Set("reverse", strconv.FormatBool(reverse))
	}
	if err := setUnixTimeRangeParams(&params, startTime, endTime); err != nil {
		return nil, err
	}
	if page != 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*TradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicMarketTradesSpotEPL, common.EncodeURLValues("spot/trades", params), &resp)
}

// GetCandlesticks retrieves market candlesticks, returning at most 1000 per request
func (e *Exchange) GetCandlesticks(ctx context.Context, currencyPair currency.Pair, limit uint64, from, to time.Time, interval kline.Interval) ([]*CandlestickResponse, error) {
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if interval.Duration().Microseconds() != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("currency_pair", currencyPair.String())
	var candles []*CandlestickResponse
	return candles, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCandleStickSpotEPL, common.EncodeURLValues("spot/candlesticks", params), &candles)
}

// GetAccountBatchFeeRates retrieves the account's spot fee rates for up to 50 currency pairs, keyed by currency pair
func (e *Exchange) GetAccountBatchFeeRates(ctx context.Context, currencyPairs []string) (map[string]*SpotTradingFeeRateResponse, error) {
	if len(currencyPairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	if len(currencyPairs) > 50 {
		return nil, fmt.Errorf("%w: %d supplied, maximum is 50", errTooManyCurrencyPairs, len(currencyPairs))
	}
	params := url.Values{}
	params.Set("currency_pairs", strings.Join(currencyPairs, ","))
	var resp map[string]*SpotTradingFeeRateResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotTradingFeeEPL, http.MethodGet, "spot/batch_fee", params, nil, &resp)
}

// GetSpotAccounts retrieves the spot account balances, optionally for a single currency
func (e *Exchange) GetSpotAccounts(ctx context.Context, ccy currency.Code) ([]*SpotAccountResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response []*SpotAccountResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "spot/accounts", params, nil, &response)
}

// GetSpotAccountBook retrieves the spot account's balance change history
// Gate rejects ranges longer than 30 days, and code takes priority over type when both are set
func (e *Exchange) GetSpotAccountBook(ctx context.Context, arg *GetSpotAccountBookRequest) ([]*SpotAccountBookResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	// Gate receives whole seconds, so the 30 day cap is checked in the units sent
	if !arg.From.IsZero() && !arg.To.IsZero() && arg.To.Unix()-arg.From.Unix() > int64(30*24*time.Hour/time.Second) {
		return nil, fmt.Errorf("%w: account book range must not exceed 30 days", errTimeRangeTooLong)
	}
	params := url.Values{}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.String())
	}
	if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
		return nil, err
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	if arg.Code != "" {
		params.Set("code", arg.Code)
	}
	var resp []*SpotAccountBookResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountBookEPL, http.MethodGet, "spot/account_book", params, nil, &resp)
}

// CreateBatchOrders places up to 10 orders on each of up to 4 currency pairs in a single request
// Every order must carry a custom text ID and use the same account; Gate reports each order's outcome individually
func (e *Exchange) CreateBatchOrders(ctx context.Context, args []CreateOrderRequest) ([]*SpotOrderResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	pairs := make(currency.Pairs, len(args))
	for x := range args {
		if x != 0 && args[x-1].Account != args[x].Account {
			return nil, errDifferentAccount
		}
		if args[x].Text == "" {
			return nil, order.ErrClientOrderIDMustBeSet
		} else if !strings.HasPrefix(args[x].Text, "t-") {
			return nil, errInvalidTextPrefix
		}
		if args[x].CurrencyPair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if args[x].Side != order.Buy.Lower() && args[x].Side != order.Sell.Lower() {
			return nil, order.ErrSideIsInvalid
		}
		if args[x].Account != asset.Spot &&
			args[x].Account != asset.CrossMargin &&
			args[x].Account != asset.Margin {
			return nil, fmt.Errorf("%w: only spot, margin, and cross_margin are allowed", asset.ErrInvalidAsset)
		}
		if args[x].Amount <= 0 {
			return nil, order.ErrAmountIsInvalid
		}
		if err := validateSpotOrderPrice(args[x].Type, args[x].Price.Float64()); err != nil {
			return nil, err
		}
		pairs[x] = args[x].CurrencyPair
	}
	if err := checkSpotBatchLimits(pairs); err != nil {
		return nil, err
	}
	var response []*SpotOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotBatchOrdersEPL, http.MethodPost, "spot/batch_orders", nil, &args, &response)
}

// checkSpotBatchLimits checks a batch of spot orders against Gate's limits of 10 orders for each currency pair and 4
// currency pairs in one request, which apply to REST and websocket batches alike
func checkSpotBatchLimits(pairs currency.Pairs) error {
	ordersPerPair := make(map[string]int, len(pairs))
	for _, p := range pairs {
		pair := p.String()
		ordersPerPair[pair]++
		if ordersPerPair[pair] > 10 {
			return fmt.Errorf("%w: at most 10 orders are allowed per currency pair, %s has more", errTooManyBatchOrders, pair)
		}
	}
	if len(ordersPerPair) > 4 {
		return fmt.Errorf("%w: at most 4 currency pairs are allowed, %d supplied", errTooManyBatchOrders, len(ordersPerPair))
	}
	return nil
}

// validateSpotOrderPrice requires a positive price for limit orders, which is Gate's default order type, while market
// orders ignore the price
func validateSpotOrderPrice(orderType string, price float64) error {
	if price < 0 || (price == 0 && orderType != order.Market.Lower()) {
		return fmt.Errorf("%w: %v", limits.ErrPriceBelowMin, price)
	}
	return nil
}

// GetSpotOpenOrders retrieves open orders across all currency pairs, optionally for a single account (spot, margin,
// cross_margin or unified)
// Pagination applies per currency pair, not to the number of currency pairs returned.
// Gate rejects an account filter of cross_margin for some account types, so callers needing every account type should
// leave account empty and filter the returned orders by their Account field.
func (e *Exchange) GetSpotOpenOrders(ctx context.Context, page, limit uint64, account string) ([]*SpotOrdersDetailResponse, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if account != "" {
		params.Set("account", account)
	}
	var response []*SpotOrdersDetailResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetOpenOrdersEPL, http.MethodGet, "spot/open_orders", params, nil, &response)
}

// SpotClosePositionWhenCrossCurrencyDisabled places a cross margin buy order to close a position in a currency whose
// cross-currency trading is disabled
func (e *Exchange) SpotClosePositionWhenCrossCurrencyDisabled(ctx context.Context, arg *ClosePositionRequest) (*SpotOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CurrencyPair.IsInvalid() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	var response *SpotOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotClosePositionEPL, http.MethodPost, "spot/cross_liquidate_orders", nil, arg, &response)
}

// PlaceSpotOrder places an order using the spot, margin or cross margin account selected by Account
func (e *Exchange) PlaceSpotOrder(ctx context.Context, arg *CreateOrderRequest) (*SpotOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CurrencyPair.IsInvalid() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side != order.Buy.Lower() && arg.Side != order.Sell.Lower() {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Account != asset.Spot &&
		arg.Account != asset.CrossMargin &&
		arg.Account != asset.Margin {
		return nil, fmt.Errorf("%w: only 'spot', 'cross_margin', and 'margin' are allowed", asset.ErrInvalidAsset)
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if err := validateSpotOrderPrice(arg.Type, arg.Price.Float64()); err != nil {
		return nil, err
	}
	var response *SpotOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotPlaceOrderEPL, http.MethodPost, "spot/orders", nil, arg, &response)
}

// GetSpotOrders retrieves open or finished spot orders
// Open orders require a currency pair and only support page and limit; side and the time range filter finished orders
func (e *Exchange) GetSpotOrders(ctx context.Context, arg *GetSpotOrdersRequest) ([]*SpotOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Status != statusOpen && arg.Status != statusFinished {
		return nil, fmt.Errorf("%w: %q, only 'open' and 'finished' are supported", errInvalidOrderStatus, arg.Status)
	}
	if arg.Status == statusOpen && arg.CurrencyPair.IsEmpty() {
		return nil, fmt.Errorf("%w: open orders require a currency pair", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("status", arg.Status)
	if !arg.CurrencyPair.IsEmpty() {
		params.Set("currency_pair", arg.CurrencyPair.String())
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Account != "" {
		params.Set("account", arg.Account)
	}
	if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
		return nil, err
	}
	switch arg.Side {
	case order.UnknownSide, order.AnySide:
	case order.Buy, order.Sell:
		params.Set("side", arg.Side.Lower())
	default:
		return nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, arg.Side)
	}
	var response []*SpotOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetOrdersEPL, http.MethodGet, "spot/orders", params, nil, &response)
}

// CancelAllOpenOrdersSpecifiedCurrencyPair cancels all open orders, optionally limited to a currency pair, side,
// account and unified market quote currency; Gate cancels the orders of every pair when the currency pair is empty
func (e *Exchange) CancelAllOpenOrdersSpecifiedCurrencyPair(ctx context.Context, arg *CancelAllSpotOrdersRequest) ([]*SpotOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !arg.CurrencyPair.IsEmpty() {
		params.Set("currency_pair", arg.CurrencyPair.String())
	}
	switch arg.Side {
	case order.Buy, order.Sell:
		params.Set("side", arg.Side.Lower())
	case order.UnknownSide, order.AnySide:
		// Do not set side. Gate will cancel both bids and asks.
	default:
		return nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, arg.Side)
	}
	if arg.Account != "" {
		params.Set("account", arg.Account)
	}
	if !arg.TradeQuote.IsEmpty() {
		params.Set("trade_quote", arg.TradeQuote.String())
	}
	if arg.ActionMode != "" {
		params.Set("action_mode", arg.ActionMode)
	}
	var response []*SpotOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelAllOpenOrdersEPL, http.MethodDelete, "spot/orders", params, nil, &response)
}

// CancelBatchOrdersWithIDList cancels up to 20 orders, which may span several currency pairs, by order ID
func (e *Exchange) CancelBatchOrdersWithIDList(ctx context.Context, args []CancelOrderByIDRequest) ([]*CancelOrderByIDResponse, error) {
	if len(args) == 0 {
		return nil, errNoValidParameterPassed
	} else if len(args) > 20 {
		return nil, fmt.Errorf("%w maximum order size to cancel is 20", errInvalidOrderSize)
	}
	for x := range args {
		if args[x].CurrencyPair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if args[x].ID == "" {
			return nil, order.ErrOrderIDNotSet
		}
	}
	var response []*CancelOrderByIDResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelBatchOrdersEPL, http.MethodPost, "spot/cancel_batch_orders", nil, &args, &response)
}

// GetSpotOrder retrieves a single spot order by order ID or custom text ID
func (e *Exchange) GetSpotOrder(ctx context.Context, orderID string, currencyPair currency.Pair, a asset.Item) (*SpotOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", currencyPair.String())
	if accountType := a.String(); accountType != "" {
		params.Set("account", accountType)
	}
	var response *SpotOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetOrderEPL, http.MethodGet, "spot/orders/"+orderID, params, nil, &response)
}

// AmendSpotOrder amends the price, amount or take profit and stop loss of an open order
// Spot, unified and margin account orders are amended by default; cross margin orders require an account of
// cross_margin. Only reducing the amount keeps the order's matching priority.
func (e *Exchange) AmendSpotOrder(ctx context.Context, orderID string, arg *AmendSpotOrderRequest) (*SpotOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var resp *SpotOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAmendOrderEPL, http.MethodPatch, "spot/orders/"+orderID, nil, arg, &resp)
}

// validate checks the fields Gate requires to amend a spot order, so single and batch amendments share the same rules
func (a *AmendSpotOrderRequest) validate() error {
	if err := common.NilGuard(a); err != nil {
		return err
	}
	if a.CurrencyPair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if a.Price <= 0 && a.Amount <= 0 {
		return fmt.Errorf("%w %w: either price or amount has to be set", order.ErrAmountIsInvalid, limits.ErrPriceBelowMin)
	}
	return nil
}

// CancelSingleSpotOrder cancels a single order
// Spot, unified and margin account orders are cancelled by default; cross margin orders require isCrossMarginAccount
func (e *Exchange) CancelSingleSpotOrder(ctx context.Context, orderID string, currencyPair currency.Pair, isCrossMarginAccount bool) (*SpotOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if isCrossMarginAccount {
		params.Set("account", crossMarginAccount)
	}
	params.Set("currency_pair", currencyPair.String())
	var response *SpotOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelSingleOrderEPL, http.MethodDelete, "spot/orders/"+orderID, params, nil, &response)
}

// GetMySpotTradingHistory retrieves the account's spot trades
// Without a time range Gate returns the last 7 days, and a range must not exceed 30 days
func (e *Exchange) GetMySpotTradingHistory(ctx context.Context, arg *GetMySpotTradingHistoryRequest) ([]*SpotPersonalTradeHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderID != "" && arg.CurrencyPair.IsEmpty() {
		return nil, fmt.Errorf("%w: filtering by order ID requires a currency pair", currency.ErrCurrencyPairEmpty)
	}
	// Gate receives whole seconds, so the 30 day cap is checked in the units sent
	if !arg.From.IsZero() && !arg.To.IsZero() && arg.To.Unix()-arg.From.Unix() > int64(30*24*time.Hour/time.Second) {
		return nil, fmt.Errorf("%w: trading history range must not exceed 30 days", errTimeRangeTooLong)
	}
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
		return nil, err
	}
	if !arg.CurrencyPair.IsEmpty() {
		params.Set("currency_pair", arg.CurrencyPair.String())
	}
	if arg.OrderID != "" {
		params.Set("order_id", arg.OrderID)
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var response []*SpotPersonalTradeHistoryResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotTradingHistoryEPL, http.MethodGet, "spot/my_trades", params, nil, &response)
}

// GetServerTime retrieves current server time
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	var resp serverTimeReply
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, publicGetServerTimeEPL, "spot/time", &resp); err != nil {
		return time.Time{}, err
	}
	return resp.ServerTime.Time(), nil
}

// CountdownCancelSpotOrders sets, renews or stops a countdown that cancels open spot orders when it expires
// Calling it again before the timeout renews the countdown, and a timeout of 0 stops it. Orders on every currency pair
// are cancelled when the currency pair is empty.
func (e *Exchange) CountdownCancelSpotOrders(ctx context.Context, arg *CountdownCancelOrderRequest) (*TriggerTimeResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if err := validateCountdownTimeout(arg.Countdown.Duration()); err != nil {
		return nil, err
	}
	var response *TriggerTimeResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCountdownCancelEPL, http.MethodPost, "spot/countdown_cancel_all", nil, arg, &response)
}

// AmendBatchSpotOrders amends up to 5 open orders in one request
// Gate amends the orders in the order supplied, continues past failures and reports each order's outcome individually
func (e *Exchange) AmendBatchSpotOrders(ctx context.Context, args []*BatchAmendSpotOrderRequest) ([]*SpotOrderResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	if len(args) > 5 {
		return nil, fmt.Errorf("%w: at most 5 orders can be amended at once, %d supplied", errTooManyBatchOrders, len(args))
	}
	for _, arg := range args {
		if err := common.NilGuard(arg); err != nil {
			return nil, err
		}
		if arg.OrderID == "" {
			return nil, order.ErrOrderIDNotSet
		}
		if err := arg.validate(); err != nil {
			return nil, err
		}
	}
	var resp []*SpotOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAmendBatchOrdersEPL, http.MethodPost, "spot/amend_batch_orders", nil, args, &resp)
}

// GetSpotInsuranceHistory retrieves the historical balances of the margin or unified account insurance fund
func (e *Exchange) GetSpotInsuranceHistory(ctx context.Context, arg *GetSpotInsuranceHistoryRequest) ([]*SpotInsuranceHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Business == "" {
		return nil, errBusinessTypeRequired
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := common.StartEndTimeCheck(arg.From, arg.To); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("business", arg.Business)
	params.Set("currency", arg.Currency.String())
	params.Set("from", strconv.FormatInt(arg.From.Unix(), 10))
	params.Set("to", strconv.FormatInt(arg.To.Unix(), 10))
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []*SpotInsuranceHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicInsuranceHistorySpotEPL, common.EncodeURLValues("spot/insurance_history", params), &resp)
}

// CreatePriceTriggeredOrder creates a price-triggered order, which places the Put order once the market price meets
// the Trigger rule
// Put.Account defaults to normal, the spot account; margin and unified are also accepted
func (e *Exchange) CreatePriceTriggeredOrder(ctx context.Context, arg *PriceTriggeredOrderRequest) (*TriggerOrderIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Market.IsEmpty() {
		return nil, fmt.Errorf("%w: market is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.Trigger.Price <= 0 {
		return nil, fmt.Errorf("%w: trigger price must be positive, got %v", limits.ErrPriceBelowMin, arg.Trigger.Price)
	}
	if arg.Trigger.Rule != ">=" && arg.Trigger.Rule != "<=" {
		return nil, fmt.Errorf("%w: %q, expected '>=' or '<='", errInvalidTriggerRule, arg.Trigger.Rule)
	}
	if err := checkWholeSeconds(arg.Trigger.ExpireAfter, "trigger expiry"); err != nil {
		return nil, err
	}
	arg.Put.Side = strings.ToLower(arg.Put.Side)
	if arg.Put.Side != order.Buy.Lower() && arg.Put.Side != order.Sell.Lower() {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Put.Price < 0 {
		return nil, fmt.Errorf("%w: put price must not be negative", limits.ErrPriceBelowMin)
	}
	if arg.Put.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	arg.Put.TimeInForce = strings.ToLower(arg.Put.TimeInForce)
	if arg.Put.TimeInForce != gtcTIF && arg.Put.TimeInForce != iocTIF {
		return nil, fmt.Errorf("%w: %q, only 'gtc' and 'ioc' are supported", order.ErrInvalidTimeInForce, arg.Put.TimeInForce)
	}
	arg.Put.Type = strings.ToLower(arg.Put.Type)
	arg.Put.Account = strings.ToLower(arg.Put.Account)
	if arg.Put.Account == "" {
		arg.Put.Account = "normal"
	}
	var response *TriggerOrderIDResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCreateTriggerOrderEPL, http.MethodPost, "spot/price_orders", nil, arg, &response)
}

// GetPriceTriggeredOrderList retrieves open or finished price-triggered orders, optionally filtered by market and
// account (normal, margin or unified)
func (e *Exchange) GetPriceTriggeredOrderList(ctx context.Context, status string, market currency.Pair, account string, offset, limit uint64) ([]*SpotPriceTriggeredOrderResponse, error) {
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w status %s", errInvalidOrderStatus, status)
	}
	params := url.Values{}
	params.Set("status", status)
	if market.IsPopulated() {
		params.Set("market", market.String())
	}
	if account != "" {
		params.Set("account", account)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	var response []*SpotPriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetTriggerOrderListEPL, http.MethodGet, "spot/price_orders", params, nil, &response)
}

// CancelMultipleSpotOpenOrders cancels all open price-triggered orders, optionally limited to a market and account
// (normal, margin or unified)
func (e *Exchange) CancelMultipleSpotOpenOrders(ctx context.Context, market currency.Pair, account string) ([]*SpotPriceTriggeredOrderResponse, error) {
	params := url.Values{}
	if market.IsPopulated() {
		params.Set("market", market.String())
	}
	if account != "" {
		params.Set("account", account)
	}
	var response []*SpotPriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelTriggerOrdersEPL, http.MethodDelete, "spot/price_orders", params, nil, &response)
}

// GetSinglePriceTriggeredOrder retrieves a single price-triggered order
func (e *Exchange) GetSinglePriceTriggeredOrder(ctx context.Context, orderID string) (*SpotPriceTriggeredOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *SpotPriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetTriggerOrderEPL, http.MethodGet, "spot/price_orders/"+orderID, nil, nil, &response)
}

// CancelPriceTriggeredOrder cancels a single price-triggered order
func (e *Exchange) CancelPriceTriggeredOrder(ctx context.Context, orderID string) (*SpotPriceTriggeredOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *SpotPriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelTriggerOrderEPL, http.MethodDelete, "spot/price_orders/"+orderID, nil, nil, &response)
}

// CreateSpotPOVOrder creates a percentage of volume (POV) order, which trades a target share of market volume until
// it fills or its time to live expires
func (e *Exchange) CreateSpotPOVOrder(ctx context.Context, arg *SpotPOVOrderRequest) (*SpotPOVOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CurrencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side != order.Buy.Lower() && arg.Side != order.Sell.Lower() {
		return nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, arg.Side)
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.ParticipationRatePercentage == 0 {
		return nil, errParticipationRateRequired
	}
	if arg.TimeToLive == "" {
		return nil, errTimeToLiveRequired
	}
	var resp *SpotPOVOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCreatePOVOrderEPL, http.MethodPost, "spot/pov_orders", nil, arg, &resp)
}

// GetSpotPOVOrders retrieves open or finished POV orders
func (e *Exchange) GetSpotPOVOrders(ctx context.Context, arg *GetSpotPOVOrdersRequest) ([]*SpotPOVOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Status != statusOpen && arg.Status != statusFinished {
		return nil, fmt.Errorf("%w: %q, only 'open' and 'finished' are supported", errInvalidOrderStatus, arg.Status)
	}
	params := url.Values{}
	params.Set("status", arg.Status)
	if !arg.CurrencyPair.IsEmpty() {
		params.Set("currency_pair", arg.CurrencyPair.String())
	}
	switch arg.Side {
	case order.UnknownSide, order.AnySide:
	case order.Buy, order.Sell:
		params.Set("side", arg.Side.Lower())
	default:
		return nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, arg.Side)
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []*SpotPOVOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetPOVOrdersEPL, http.MethodGet, "spot/pov_orders", params, nil, &resp)
}

// CancelSpotPOVOrders cancels all POV orders, or only those on the supplied currency pair
func (e *Exchange) CancelSpotPOVOrders(ctx context.Context, currencyPair currency.Pair) ([]*SpotPOVOrderResponse, error) {
	params := url.Values{}
	if !currencyPair.IsEmpty() {
		params.Set("currency_pair", currencyPair.String())
	}
	var resp []*SpotPOVOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelPOVOrdersEPL, http.MethodDelete, "spot/pov_orders", params, nil, &resp)
}

// GetSpotPOVOrder retrieves a single POV order by order ID or custom text ID
func (e *Exchange) GetSpotPOVOrder(ctx context.Context, orderID string) (*SpotPOVOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *SpotPOVOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetPOVOrderEPL, http.MethodGet, "spot/pov_orders/"+orderID, nil, nil, &resp)
}

// CancelSpotPOVOrder cancels a single POV order by order ID or custom text ID
func (e *Exchange) CancelSpotPOVOrder(ctx context.Context, orderID string) (*SpotPOVOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *SpotPOVOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelPOVOrderEPL, http.MethodDelete, "spot/pov_orders/"+orderID, nil, nil, &resp)
}
