package gate

import (
	"context"
	"errors"
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
func (e *Exchange) GetAlphaAccounts(ctx context.Context) ([]*AlphaAccountResponse, error) {
	var resp []*AlphaAccountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, alphaAccountsEPL, http.MethodGet, "alpha/accounts", nil, nil, &resp)
}

// GetAlphaAccountTransactionHistory retrieves alpha account asset transactions
func (e *Exchange) GetAlphaAccountTransactionHistory(ctx context.Context, from, to time.Time, page, limit uint64) ([]*AlphaAccountTransactionItemResponse, error) {
	if from.IsZero() {
		return nil, errStartTimeRequired
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
	var resp []*AlphaAccountTransactionItemResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, alphaAccountBookEPL, http.MethodGet, "alpha/account_book", params, nil, &resp)
}

// CreateAlphaCurrencyQuoteID returns quote information for a currency.
// The quote_id returned by the price inquiry interface is valid for one minute; an order must be placed within this minute.
func (e *Exchange) CreateAlphaCurrencyQuoteID(ctx context.Context, arg *AlphaCurrencyQuoteInfoRequest) (*AlphaCurrencyQuoteDetailResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency == "" {
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
	var resp *AlphaCurrencyQuoteDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, alphaCreateQuoteEPL, http.MethodPost, "alpha/quote", nil, arg, &resp)
}

// PlaceAlphaTradeOrder places a quote order
func (e *Exchange) PlaceAlphaTradeOrder(ctx context.Context, arg *AlphaCurrencyQuoteInfoRequest) (*AlphaPlaceOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency == "" {
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
func (e *Exchange) GetAlphaOrders(ctx context.Context, arg *GetAlphaOrdersRequest) ([]*AlphaOrderDetailResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.Currency != "" {
		params.Set("currency", arg.Currency)
	}
	if arg.Side != order.UnknownSide {
		params.Set("side", arg.Side.Lower())
	}
	if arg.Status != 0 {
		params.Set("status", strconv.FormatUint(uint64(arg.Status), 10))
	}
	if !arg.StartTime.IsZero() {
		params.Set("from", strconv.FormatInt(arg.StartTime.Unix(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("to", strconv.FormatInt(arg.EndTime.Unix(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	var resp []*AlphaOrderDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, alphaListOrdersEPL, http.MethodGet, "alpha/orders", params, nil, &resp)
}

// GetAlphaOrderByID retrieves a single alpha order detail by ID
func (e *Exchange) GetAlphaOrderByID(ctx context.Context, orderID string) (*AlphaOrderDetailResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("order_id", orderID)
	var resp *AlphaOrderDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, alphaGetOrderEPL, http.MethodGet, "alpha/order", params, nil, &resp)
}

// GetAlphaCurrenciesDetail retrieves currency details, for every currency or for the one alphaCurrency names. It stays a
// string, as alpha symbols such as memeboxTUT are case sensitive
func (e *Exchange) GetAlphaCurrenciesDetail(ctx context.Context, alphaCurrency string, limit, page uint64) ([]*AlphaCurrencyDetailResponse, error) {
	params := url.Values{}
	if alphaCurrency != "" {
		params.Set("currency", alphaCurrency)
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*AlphaCurrencyDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, alphaCurrenciesEPL, common.EncodeURLValues("alpha/currencies", params), &resp)
}

// GetAlphaCurrencyTicker retrieves currency ticker information, for every currency or for the one alphaCurrency names. It
// stays a string, as alpha symbols such as memeboxTUT are case sensitive
func (e *Exchange) GetAlphaCurrencyTicker(ctx context.Context, alphaCurrency string, limit, page uint64) ([]*AlphaCurrencyTickerInfoResponse, error) {
	params := url.Values{}
	if alphaCurrency != "" {
		params.Set("currency", alphaCurrency)
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*AlphaCurrencyTickerInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, alphaTickersEPL, common.EncodeURLValues("alpha/tickers", params), &resp)
}

// GetAlphaTokens retrieves token information, optionally filtered by chain, launch platform and contract address
func (e *Exchange) GetAlphaTokens(ctx context.Context, chain, launchPlatform, address string, page uint64) ([]*AlphaCurrencyDetailResponse, error) {
	params := url.Values{}
	if chain != "" {
		params.Set("chain", chain)
	}
	if launchPlatform != "" {
		params.Set("launch_platform", launchPlatform)
	}
	if address != "" {
		params.Set("address", address)
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	var resp []*AlphaCurrencyDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, alphaTokensEPL, common.EncodeURLValues("alpha/tokens", params), &resp)
}
