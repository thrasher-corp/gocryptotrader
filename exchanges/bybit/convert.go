package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var supportedAccountTypes = []WalletAccountType{Funding, UTA, Spot, Contract, Inverse}

var (
	errUnsupportedAccountType  = errors.New("unsupported account type")
	errCurrencyCodesEqual      = errors.New("from and to currency codes cannot be equal")
	errRequestCoinInvalid      = errors.New("request coin must match from coin if provided")
	errQuoteTransactionIDEmpty = errors.New("quoteTransactionID cannot be empty")
)

// GetConvertCoinList returns a list of coins you can convert to/from
func (e *Exchange) GetConvertCoinList(ctx context.Context, accountType WalletAccountType, coin currency.Code, isCoinToBuy bool) ([]ConvertCoinResponse, error) {
	if !slices.Contains(supportedAccountTypes, accountType) {
		return nil, fmt.Errorf("%w: %q", errUnsupportedAccountType, accountType)
	}

	params := url.Values{}
	params.Set("accountType", string(accountType))
	if isCoinToBuy {
		if coin.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		params.Set("coin", coin.Upper().String())
		params.Set("side", "1")
	} else {
		params.Set("side", "0")
	}

	var resp struct {
		List []ConvertCoinResponse `json:"coins"`
	}

	return resp.List, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/exchange/query-coin-list", params, nil, &resp, defaultEPL)
}

// RequestQuote requests a conversion quote between two coins with the specified parameters.
func (e *Exchange) RequestQuote(ctx context.Context, params *RequestQuoteRequest) (*RequestQuoteResponse, error) {
	if !slices.Contains(supportedAccountTypes, params.AccountType) {
		return nil, fmt.Errorf("%w: %q", errUnsupportedAccountType, params.AccountType)
	}
	if params.FromCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: `from` coin", currency.ErrCurrencyCodeEmpty)
	}
	if params.ToCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: `to` coin", currency.ErrCurrencyCodeEmpty)
	}
	if params.FromCoin.Equal(params.ToCoin) {
		return nil, errCurrencyCodesEqual
	}
	if !params.RequestCoin.IsEmpty() {
		if !params.RequestCoin.Equal(params.FromCoin) {
			return nil, errRequestCoinInvalid
		}
	} else {
		params.RequestCoin = params.FromCoin
	}
	if params.RequestAmount <= 0 {
		return nil, fmt.Errorf("%w: %v", order.ErrAmountIsInvalid, params.RequestAmount)
	}

	params.FromCoin = params.FromCoin.Upper()
	params.ToCoin = params.ToCoin.Upper()
	params.RequestCoin = params.RequestCoin.Upper()

	var resp *RequestQuoteResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/exchange/quote-apply", nil, params, &resp, defaultEPL)
}

// ConfirmQuote confirms a quote transaction and executes the conversion
func (e *Exchange) ConfirmQuote(ctx context.Context, quoteTransactionID string) (*ConfirmQuoteResponse, error) {
	if quoteTransactionID == "" {
		return nil, errQuoteTransactionIDEmpty
	}

	payload := struct {
		QuoteTransactionID string `json:"quoteTxId"`
	}{
		QuoteTransactionID: quoteTransactionID,
	}

	var resp *ConfirmQuoteResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/exchange/convert-execute", nil, payload, &resp, defaultEPL)
}

// GetConvertStatus retrieves the status of a conversion transaction
func (e *Exchange) GetConvertStatus(ctx context.Context, accountType WalletAccountType, quoteTransactionID string) (*ConvertStatusResponse, error) {
	if !slices.Contains(supportedAccountTypes, accountType) {
		return nil, fmt.Errorf("%w: %q", errUnsupportedAccountType, accountType)
	}
	if quoteTransactionID == "" {
		return nil, errQuoteTransactionIDEmpty
	}

	params := url.Values{}
	params.Set("quoteTxId", quoteTransactionID)
	params.Set("accountType", string(accountType))

	var resp struct {
		ConvertStatus *ConvertStatusResponse `json:"result"`
	}
	return resp.ConvertStatus, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/exchange/convert-result-query", params, nil, &resp, defaultEPL)
}

// GetConvertHistory retrieves the conversion history.
// All params are optional
func (e *Exchange) GetConvertHistory(ctx context.Context, accountTypes []WalletAccountType, index, limit uint64) ([]ConvertHistoryResponse, error) {
	atOut := make([]string, 0, len(accountTypes))
	for _, accountType := range accountTypes {
		if !slices.Contains(supportedAccountTypes, accountType) {
			return nil, fmt.Errorf("%w: %q", errUnsupportedAccountType, accountType)
		}
		atOut = append(atOut, string(accountType))
	}

	params := url.Values{}
	if len(atOut) > 0 {
		params.Add("accountType", strings.Join(atOut, ","))
	}
	if index != 0 {
		params.Set("index", strconv.FormatUint(index, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}

	var resp struct {
		List []ConvertHistoryResponse `json:"list"`
	}
	return resp.List, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/exchange/query-convert-history", params, nil, &resp, defaultEPL)
}
