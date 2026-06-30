package gateio

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errStartTimeRequired = errors.New("start time required")
	errGasModeRequired   = errors.New("gas mode is required")
	errQuoteIDRequired   = errors.New("quote ID is required")
)

// GetAlphaAccounts retrieves accounts position assets
func (e *Exchange) GetAlphaAccounts(ctx context.Context) ([]*AlphaAccount, error) {
	var resp []*AlphaAccount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, alphaAccountsEPL, http.MethodGet, "alpha/accounts", nil, nil, &resp)
}

// GetAlphaAccountTransactionHistory retrieves alpha account asset transactions
// gas_mode possible values are 'speed' : Smart mode 'custom' : Custom mode, uses slippage parameter
func (e *Exchange) GetAlphaAccountTransactionHistory(ctx context.Context, from, to time.Time, page, limit uint64) ([]*AlphaAccountTransactionItem, error) {
	if from.IsZero() {
		return nil, fmt.Errorf("%w: from is missing", errStartTimeRequired)
	}
	if !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("from", strconv.FormatInt(from.Unix(), 10))
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*AlphaAccountTransactionItem
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, alphaAccountBookEPL, http.MethodGet, "alpha/account_book", params, nil, &resp)
}

// CreateAlphaCurrencyQuoteID returns quote information for a currency.
// The quote_id returned by the price inquiry interface is valid for one minute; an order must be placed within this minute.
func (e *Exchange) CreateAlphaCurrencyQuoteID(ctx context.Context, arg *AlphaCurrencyQuoteInfoRequest) (*AlphaCurrencyQuoteDetail, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Side != order.Buy && arg.Side != order.Sell {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.GasMode == "" {
		return nil, errGasModeRequired
	}
	var resp *AlphaCurrencyQuoteDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, alphaCreateQuoteEPL, http.MethodPost, "alpha/quote", nil, arg, &resp)
}

// PlaceAlphaTradeOrder places a quote order
func (e *Exchange) PlaceAlphaTradeOrder(ctx context.Context, arg *AlphaCurrencyQuoteInfoRequest) (*AlphaPlaceOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Side != order.Buy && arg.Side != order.Sell {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.GasMode == "" {
		return nil, errGasModeRequired
	}
	if arg.QuoteID == "" {
		return nil, errQuoteIDRequired
	}
	var resp *AlphaPlaceOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, alphaPlaceOrderEPL, http.MethodPost, "alpha/orders", nil, arg, &resp)
}

// GetAlphaOrders retrieves alpha orders
// possible state values are: 0 : All 1 : Processing 2 : Successful 3 : Failed 4 : Cancelled 5 : Buy order placed but transfer not completed 6 : Order cancelled but transfer not completed
func (e *Exchange) GetAlphaOrders(ctx context.Context, memeCcy currency.Code, orderSide order.Side, state uint8, from, to time.Time, page, limit uint64) ([]*AlphaOrderDetail, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !memeCcy.IsEmpty() {
		params.Set("currency", memeCcy.String())
	}
	if orderSide != order.UnknownSide {
		params.Set("side", orderSide.Lower())
	}
	if state != 0 {
		params.Set("status", strconv.FormatUint(uint64(state), 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*AlphaOrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, alphaListOrdersEPL, http.MethodGet, "alpha/orders", params, nil, &resp)
}

// GetAlphaOrderByID retrieves a single alpha order detail by ID
func (e *Exchange) GetAlphaOrderByID(ctx context.Context, orderID string) (*AlphaOrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("order_id", orderID)
	var resp *AlphaOrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, alphaGetOrderEPL, http.MethodGet, "alpha/order", params, nil, &resp)
}

// GetAlphaCurrenciesDetail retrieves currency details
func (e *Exchange) GetAlphaCurrenciesDetail(ctx context.Context, memeCcy currency.Code, limit, page uint64) ([]*AlphaCurrencyDetail, error) {
	params := url.Values{}
	if !memeCcy.IsEmpty() {
		params.Set("currency", memeCcy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*AlphaCurrencyDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, alphaCurrenciesEPL, common.EncodeURLValues("alpha/currencies", params), &resp)
}

// GetAlphaCurrencyTicker retrieves currency ticker information
func (e *Exchange) GetAlphaCurrencyTicker(ctx context.Context, memeCcy currency.Code, limit, page uint64) ([]*AlphaCurrencyTickerInfo, error) {
	params := url.Values{}
	if !memeCcy.IsEmpty() {
		params.Set("currency", memeCcy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*AlphaCurrencyTickerInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, alphaTickersEPL, common.EncodeURLValues("alpha/tickers", params), &resp)
}
