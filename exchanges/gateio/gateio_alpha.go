package gateio

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	alphaLiveTradingURL    = "https://api.gateio.ws/api/v4"
	alphaTestnetTradingURL = "https://api-testnet.gateapi.io/api/v4"
)

var (
	errStartTimeRequired = errors.New("start time required")
	errGasModeRequired   = errors.New("gas mode is required")
)

// GetAlphaAccounts retrieves accounts position assets
func (e *Exchange) GetAlphaAccounts(ctx context.Context) ([]AlphaAccount, error) {
	var resp []AlphaAccount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestAlpha, request.UnAuth, http.MethodGet, "alpha/accounts", nil, nil, &resp)
}

// GetAlphaAccountTransactionHistory retrieves alpha account asset transactions
func (e *Exchange) GetAlphaAccountTransactionHistory(ctx context.Context, from, to time.Time, page, limit uint64) ([]AlphaAccountTransactionItem, error) {
	if from.IsZero() {
		return nil, fmt.Errorf("%w: from is missing", errStartTimeRequired)
	}
	params := url.Values{}
	params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []AlphaAccountTransactionItem
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestAlpha, request.UnAuth, http.MethodGet, "alpha/account_book", params, nil, &resp)
}

// GetAlphaCurrencyQuoteInfo returns quote information for a currency.
// The quote_id returned by the price inquiry interface is valid for one minute; an order must be placed within this minute.
func (e *Exchange) GetAlphaCurrencyQuoteInfo(ctx context.Context, arg *AlphaCurrencyQuoteInfoRequest) (*AlphaCurrencyQuoteDetail, error) {
	if err := validateCurrencyQuoteRequest(arg); err != nil {
		return nil, err
	}
	var resp *AlphaCurrencyQuoteDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestAlpha, request.UnAuth, http.MethodGet, "alpha/quote", nil, arg, &resp)
}

func validateCurrencyQuoteRequest(arg *AlphaCurrencyQuoteInfoRequest) error {
	if arg.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if arg.Side != order.Buy && arg.Side != order.Sell {
		return order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return limits.ErrAmountBelowMin
	}
	if arg.GasMode == "" {
		return errGasModeRequired
	}
	return nil
}

// PlaceAlphaTradeOrder places a quote order
func (e *Exchange) PlaceAlphaTradeOrder(ctx context.Context, arg *AlphaCurrencyQuoteInfoRequest) (*AlphaCurrencyQuoteDetail, error) {
	if err := validateCurrencyQuoteRequest(arg); err != nil {
		return nil, err
	}
	var resp *AlphaCurrencyQuoteDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestAlpha, request.UnAuth, http.MethodPost, "alpha/quote", nil, arg, &resp)
}
