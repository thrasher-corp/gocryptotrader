package bybit

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetSmallBalanceCoins returns the Unified wallet's coins worth less than 10 USDT, which can be converted in batches
// worth 1e-8 to 200 USDT; the API key needs the Convert permission (Get Small Balance Coins)
func (e *Exchange) GetSmallBalanceCoins(ctx context.Context, accountType string, fromCoin currency.Code) (*SmallBalanceCoinsResponse, error) {
	if accountType == "" {
		return nil, errMissingAccountType
	}
	params := url.Values{}
	params.Set("accountType", accountType)
	if !fromCoin.IsEmpty() {
		params.Set("fromCoin", fromCoin.Upper().String())
	}
	var resp *SmallBalanceCoinsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/covert/small-balance-list", params, nil, &resp, getSmallBalanceCoinsEPL)
}

// RequestSmallBalanceQuote requests a quote to convert up to 20 small balance coins to MNT, USDT or USDC; custody
// accounts cannot convert, and confirm the quote with ConfirmSmallBalanceQuote within 30 seconds (Request a Quote)
func (e *Exchange) RequestSmallBalanceQuote(ctx context.Context, arg *SmallBalanceQuoteRequest) (*SmallBalanceQuoteResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.AccountType == "" {
		return nil, errMissingAccountType
	}
	if len(arg.FromCoins) == 0 {
		return nil, fmt.Errorf("%w: from coins", currency.ErrCurrencyCodeEmpty)
	}
	fromCoins := make([]string, len(arg.FromCoins))
	for i := range arg.FromCoins {
		if arg.FromCoins[i].IsEmpty() {
			return nil, fmt.Errorf("%w: from coin %d", currency.ErrCurrencyCodeEmpty, i)
		}
		fromCoins[i] = arg.FromCoins[i].Upper().String()
	}
	if arg.ToCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: to coin", currency.ErrCurrencyCodeEmpty)
	}
	body := map[string]any{
		"accountType":  arg.AccountType,
		"fromCoinList": fromCoins,
		"toCoin":       arg.ToCoin.Upper().String(),
	}
	var resp *SmallBalanceQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/covert/get-quote", nil, body, &resp, requestSmallBalanceQuoteEPL)
}

// ConfirmSmallBalanceQuote confirms a small balance quote before it expires; the conversion completes asynchronously,
// so GetSmallBalanceExchangeHistory gives its outcome (Confirm a Quote)
func (e *Exchange) ConfirmSmallBalanceQuote(ctx context.Context, quoteID string) (*SmallBalanceConfirmQuoteResponse, error) {
	if quoteID == "" {
		return nil, errQuoteIDRequired
	}
	body := map[string]any{"quoteId": quoteID}
	var resp *SmallBalanceConfirmQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/covert/small-balance-execute", nil, body, &resp, confirmSmallBalanceQuoteEPL)
}

// GetSmallBalanceExchangeHistory returns small balance conversions of the Unified and Funding wallets, made through the
// API or on the web or app (Get Exchange History)
func (e *Exchange) GetSmallBalanceExchangeHistory(ctx context.Context, arg *SmallBalanceExchangeHistoryRequest) (*SmallBalanceExchangeHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.AccountType != "" {
		params.Set("accountType", arg.AccountType)
	}
	if arg.QuoteID != "" {
		params.Set("quoteId", arg.QuoteID)
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Cursor > 0 {
		params.Set("cursor", strconv.FormatUint(arg.Cursor, 10))
	}
	if arg.Size > 0 {
		params.Set("size", strconv.FormatUint(arg.Size, 10))
	}
	var resp *SmallBalanceExchangeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/covert/small-balance-history", params, nil, &resp, getSmallBalanceExchangeHistoryEPL)
}
