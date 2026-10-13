package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var errQuoteTransactionIDEmpty = errors.New("quote transaction ID required")

// GetConvertCoinList returns the coins an account can convert from, with their balances, or the coins it can convert
// to; the API key needs the Exchange permission (Get Convert Coin List)
func (e *Exchange) GetConvertCoinList(ctx context.Context, arg *ConvertCoinListRequest) (*ConvertCoinListResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.AccountType == "" {
		return nil, errMissingAccountType
	}
	params := url.Values{}
	params.Set("accountType", arg.AccountType)
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.Upper().String())
	}
	params.Set("side", strconv.FormatUint(arg.Side, 10))
	var resp *ConvertCoinListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/exchange/query-coin-list", params, nil, &resp, getConvertCoinListEPL)
}

// RequestQuote requests a quote to convert one coin to another, after checking the balance; confirm it with
// ConfirmQuote within 15 seconds (Request a Quote)
func (e *Exchange) RequestQuote(ctx context.Context, arg *RequestQuoteRequest) (*RequestQuoteResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.AccountType == "" {
		return nil, errMissingAccountType
	}
	if arg.FromCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: from coin", currency.ErrCurrencyCodeEmpty)
	}
	if arg.ToCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: to coin", currency.ErrCurrencyCodeEmpty)
	}
	if arg.RequestAmount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	requestCoin := arg.RequestCoin
	if requestCoin.IsEmpty() {
		requestCoin = arg.FromCoin
	}
	body := map[string]any{
		"accountType":   arg.AccountType,
		"fromCoin":      arg.FromCoin.Upper().String(),
		"toCoin":        arg.ToCoin.Upper().String(),
		"requestCoin":   requestCoin.Upper().String(),
		"requestAmount": strconv.FormatFloat(arg.RequestAmount, 'f', -1, 64),
	}
	for key, value := range map[string]string{
		"fromCoinType": arg.FromCoinType,
		"toCoinType":   arg.ToCoinType,
		"paramType":    arg.ParamType,
		"paramValue":   arg.ParamValue,
		"requestId":    arg.RequestID,
	} {
		if value != "" {
			body[key] = value
		}
	}
	var resp *RequestQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/exchange/quote-apply", nil, body, &resp, requestQuoteEPL)
}

// ConfirmQuote confirms a quote before it expires; the conversion completes asynchronously, so GetConvertStatus gives
// its outcome (Confirm a Quote)
func (e *Exchange) ConfirmQuote(ctx context.Context, quoteTransactionID string) (*ConfirmQuoteResponse, error) {
	if quoteTransactionID == "" {
		return nil, errQuoteTransactionIDEmpty
	}
	body := map[string]any{"quoteTxId": quoteTransactionID}
	var resp *ConfirmQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/exchange/convert-execute", nil, body, &resp, confirmQuoteEPL)
}

// GetConvertStatus returns a conversion and its status; accountType must be the wallet the quote was requested for
// (Get Convert Status)
func (e *Exchange) GetConvertStatus(ctx context.Context, accountType, quoteTransactionID string) (*ConvertStatusResponse, error) {
	if accountType == "" {
		return nil, errMissingAccountType
	}
	if quoteTransactionID == "" {
		return nil, errQuoteTransactionIDEmpty
	}
	params := url.Values{}
	params.Set("quoteTxId", quoteTransactionID)
	params.Set("accountType", accountType)
	var resp *ConvertStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/exchange/convert-result-query", params, nil, &resp, getConvertStatusEPL)
}

// GetConvertHistory returns confirmed conversions, including those made on the web or app since 10 September 2025,
// for the listed wallet types or for all of them (Get Convert History)
func (e *Exchange) GetConvertHistory(ctx context.Context, arg *ConvertHistoryRequest) (*ConvertHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if len(arg.AccountTypes) > 0 {
		params.Set("accountType", strings.Join(arg.AccountTypes, ","))
	}
	if arg.Index > 0 {
		params.Set("index", strconv.FormatUint(arg.Index, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp *ConvertHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/exchange/query-convert-history", params, nil, &resp, getConvertHistoryEPL)
}
