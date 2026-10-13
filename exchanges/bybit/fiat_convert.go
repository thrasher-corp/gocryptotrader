package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errCoinTypeRequired                       = errors.New("coin type required")
	errSubUserIDRequired                      = errors.New("sub user ID required")
	errTradeNumberOrMerchantRequestIDRequired = errors.New("trade number or merchant request ID required")
)

// GetFiatTradingPairList returns the fiat and crypto coins of fiat conversions; side is 0 to buy crypto with fiat or 1
// to sell crypto for fiat, and is not sent when empty (Get Trading Pair List)
func (e *Exchange) GetFiatTradingPairList(ctx context.Context, side string) (*FiatTradingPairListResponse, error) {
	params := url.Values{}
	if side != "" {
		params.Set("side", side)
	}
	var resp *FiatTradingPairListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/fiat/query-coin-list", params, nil, &resp, getFiatTradingPairListEPL)
}

// GetFiatReferencePrice returns a fiat and crypto pair's reference prices per payment method, for a symbol such as
// EUR-USDT. The API answers without authentication, and returns one object although the documentation describes an
// array (Get Reference Price)
func (e *Exchange) GetFiatReferencePrice(ctx context.Context, symbol string) (*FiatReferencePriceResponse, error) {
	if symbol == "" {
		return nil, errSymbolMissing
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FiatReferencePriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/fiat/reference-price", params), getFiatReferencePriceEPL, &resp)
}

// RequestFiatQuote requests a quote to convert between fiat and crypto, using the master UID's API key; confirm it
// with ConfirmFiatQuote before it expires (Request a Quote)
func (e *Exchange) RequestFiatQuote(ctx context.Context, arg *FiatQuoteRequest) (*FiatQuoteResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.FromCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: from coin", currency.ErrCurrencyCodeEmpty)
	}
	if arg.FromCoinType == "" {
		return nil, fmt.Errorf("%w: from coin type", errCoinTypeRequired)
	}
	if arg.ToCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: to coin", currency.ErrCurrencyCodeEmpty)
	}
	if arg.ToCoinType == "" {
		return nil, fmt.Errorf("%w: to coin type", errCoinTypeRequired)
	}
	if arg.RequestAmount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	body := map[string]any{
		"fromCoin":      arg.FromCoin.Upper().String(),
		"fromCoinType":  arg.FromCoinType,
		"toCoin":        arg.ToCoin.Upper().String(),
		"toCoinType":    arg.ToCoinType,
		"requestAmount": strconv.FormatFloat(arg.RequestAmount, 'f', -1, 64),
	}
	if arg.RequestCoinType != "" {
		body["requestCoinType"] = arg.RequestCoinType
	}
	var resp *FiatQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/fiat/quote-apply", nil, body, &resp, requestFiatQuoteEPL)
}

// ConfirmFiatQuote confirms a fiat conversion quote for a sub user before it expires; the conversion completes
// asynchronously, so GetFiatConvertStatus gives its outcome, and Bybit calls the webhook URL when it ends (Confirm a
// Quote)
func (e *Exchange) ConfirmFiatQuote(ctx context.Context, arg *FiatConfirmQuoteRequest) (*FiatConfirmQuoteResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.QuoteTransactionID == "" {
		return nil, errQuoteTransactionIDEmpty
	}
	if arg.SubUserID == "" {
		return nil, errSubUserIDRequired
	}
	body := map[string]any{
		"quoteTxId": arg.QuoteTransactionID,
		"subUserId": arg.SubUserID,
	}
	if arg.WebhookURL != "" {
		body["webhookUrl"] = arg.WebhookURL
	}
	if arg.MerchantRequestID != "" {
		body["merchantRequestId"] = arg.MerchantRequestID
	}
	var resp *FiatConfirmQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/fiat/trade-execute", nil, body, &resp, confirmFiatQuoteEPL)
}

// GetFiatConvertStatus returns a fiat conversion by its trade number or merchant request ID, one of which is required
// (Get Convert Status)
func (e *Exchange) GetFiatConvertStatus(ctx context.Context, tradeNumber, merchantRequestID string) (*FiatConvertStatusResponse, error) {
	if tradeNumber == "" && merchantRequestID == "" {
		return nil, errTradeNumberOrMerchantRequestIDRequired
	}
	params := url.Values{}
	if tradeNumber != "" {
		params.Set("tradeNo", tradeNumber)
	}
	if merchantRequestID != "" {
		params.Set("merchantRequestId", merchantRequestID)
	}
	var resp *FiatConvertStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/fiat/trade-query", params, nil, &resp, getFiatConvertStatusEPL)
}

// GetFiatConvertHistory returns fiat conversions (Get Convert History)
func (e *Exchange) GetFiatConvertHistory(ctx context.Context, arg *FiatConvertHistoryRequest) ([]FiatTrade, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.Index > 0 {
		params.Set("index", strconv.FormatUint(arg.Index, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	var resp []FiatTrade
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/fiat/query-trade-history", params, nil, &resp, getFiatConvertHistoryEPL)
}

// GetFiatBalance returns the fiat balances, of one ISO 4217 currency or of all of them when fiatCurrency is empty (Get
// Balance)
func (e *Exchange) GetFiatBalance(ctx context.Context, fiatCurrency currency.Code) (FiatBalances, error) {
	params := url.Values{}
	if !fiatCurrency.IsEmpty() {
		params.Set("currency", fiatCurrency.Upper().String())
	}
	var resp FiatBalances
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/fiat/balance-query", params, nil, &resp, getFiatBalanceEPL)
}
