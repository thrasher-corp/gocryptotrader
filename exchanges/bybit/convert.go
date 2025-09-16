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

var supportedAccountTypes = []WalletAccountType{Funding, Uta, Spot, Contract, Inverse}

var (
	errUnsupportedAccountType  = errors.New("unsupported account type")
	errCurrencyCodesEqual      = errors.New("from and to currency codes cannot be equal")
	errRequestCoinInvalid      = errors.New("request coin must match from coin if provided")
	errQuoteTransactionIDEmpty = errors.New("quoteTransactionID cannot be empty")
)

// GetConvertCoinList returns a list of coins you can convert to/from
func (e *Exchange) GetConvertCoinList(ctx context.Context, accountType WalletAccountType, coin currency.Code, isCoinToBuy bool) ([]CoinResponse, error) {
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
		List []CoinResponse `json:"coins"`
	}

	return resp.List, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/exchange/query-coin-list", params, nil, &resp, defaultEPL)
}

// RequestAQuote requests a conversion quote between two coins with the specified parameters.
func (e *Exchange) RequestAQuote(ctx context.Context, params *RequestAQuoteRequest) (*RequestAQuoteResponse, error) {
	if !slices.Contains(supportedAccountTypes, params.AccountType) {
		return nil, fmt.Errorf("%w: %q", errUnsupportedAccountType, params.AccountType)
	}

	if params.From.IsEmpty() {
		return nil, fmt.Errorf("from %w", currency.ErrCurrencyCodeEmpty)
	}
	if params.To.IsEmpty() {
		return nil, fmt.Errorf("to %w", currency.ErrCurrencyCodeEmpty)
	}

	if params.From.Equal(params.To) {
		return nil, errCurrencyCodesEqual
	}

	if !params.RequestCoin.IsEmpty() {
		if !params.RequestCoin.Equal(params.From) {
			return nil, errRequestCoinInvalid
		}
	} else {
		params.RequestCoin = params.From
	}

	if params.Amount <= 0 {
		return nil, fmt.Errorf("amount %w", order.ErrAmountIsInvalid)
	}

	params.From = params.From.Upper()
	params.To = params.To.Upper()
	params.RequestCoin = params.RequestCoin.Upper()

	var resp *RequestAQuoteResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/exchange/quote-apply", nil, params, &resp, defaultEPL)
}

// ConfirmAQuote confirms a quote transaction and executes the conversion
func (e *Exchange) ConfirmAQuote(ctx context.Context, quoteTransactionID string) (*ConfirmAQuoteResponse, error) {
	if quoteTransactionID == "" {
		return nil, errQuoteTransactionIDEmpty
	}

	payload := struct {
		QuoteTransactionID string `json:"quoteTxId"`
	}{
		QuoteTransactionID: quoteTransactionID,
	}

	var resp *ConfirmAQuoteResponse
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

// GetConvertHistory retrieves the conversion history for the specified account types.
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
