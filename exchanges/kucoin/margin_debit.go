package kucoin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetBorrowInterestRate returns the borrowing interest rates of a VIP level (Get Borrow Interest Rate)
func (e *Exchange) GetBorrowInterestRate(ctx context.Context, req *BorrowInterestRateRequest) (*BorrowInterestRateResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.VIPLevel != nil {
		params.Set("vipLevel", strconv.FormatUint(*req.VIPLevel, 10))
	}
	if len(req.Currencies) > 0 {
		list, err := currencyListParam(req.Currencies)
		if err != nil {
			return nil, err
		}
		params.Set("currency", list)
	}
	var resp *BorrowInterestRateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getBorrowInterestRateEPL, http.MethodGet, common.EncodeURLValues("/v3/margin/borrowRate", params), nil, &resp)
}

// PostMarginBorrowOrder borrows into the cross or isolated margin account (Borrow)
func (e *Exchange) PostMarginBorrowOrder(ctx context.Context, req *MarginBorrowOrderRequest) (*MarginBorrowOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Size <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.TimeInForce == "" {
		return nil, errTimeInForceRequired
	}
	if req.IsIsolated && req.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	body := map[string]any{
		"currency":    req.Currency.Upper().String(),
		"size":        req.Size, // documented as a JSON number, unlike most decimals
		"timeInForce": req.TimeInForce,
	}
	if req.Symbol != "" {
		body["symbol"] = req.Symbol
	}
	if req.IsIsolated {
		body["isIsolated"] = true
	}
	if req.IsHF {
		body["isHf"] = true
	}
	var resp *MarginBorrowOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, postMarginBorrowOrderEPL, http.MethodPost, "/v3/margin/borrow", body, &resp)
}

// GetMarginBorrowingHistory returns the cross or isolated margin account's borrow orders (Get Borrow History)
func (e *Exchange) GetMarginBorrowingHistory(ctx context.Context, req *MarginBorrowingHistoryRequest) (*MarginBorrowingHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.IsIsolated && req.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("currency", req.Currency.Upper().String())
	if req.IsIsolated {
		params.Set("isIsolated", "true")
	}
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if req.OrderNumber != "" {
		params.Set("orderNo", req.OrderNumber)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.CurrentPage > 0 {
		params.Set("currentPage", strconv.FormatUint(req.CurrentPage, 10))
	}
	if req.PageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp *MarginBorrowingHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getMarginBorrowingHistoryEPL, http.MethodGet, common.EncodeURLValues("/v3/margin/borrow", params), nil, &resp)
}

// PostRepayment repays the cross or isolated margin account's borrowing (Repay)
func (e *Exchange) PostRepayment(ctx context.Context, req *MarginRepaymentRequest) (*MarginRepaymentResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Size <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.IsIsolated && req.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	body := map[string]any{
		"currency": req.Currency.Upper().String(),
		"size":     req.Size, // documented as a JSON number, unlike most decimals
	}
	if req.Symbol != "" {
		body["symbol"] = req.Symbol
	}
	if req.IsIsolated {
		body["isIsolated"] = true
	}
	if req.IsHF {
		body["isHf"] = true
	}
	var resp *MarginRepaymentResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, postRepaymentEPL, http.MethodPost, "/v3/margin/repay", body, &resp)
}

// GetRepaymentHistory returns the cross or isolated margin account's repayment orders (Get Repay History)
func (e *Exchange) GetRepaymentHistory(ctx context.Context, req *MarginRepaymentHistoryRequest) (*MarginRepaymentHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.IsIsolated && req.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("currency", req.Currency.Upper().String())
	if req.IsIsolated {
		params.Set("isIsolated", "true")
	}
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if req.OrderNumber != "" {
		params.Set("orderNo", req.OrderNumber)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.CurrentPage > 0 {
		params.Set("currentPage", strconv.FormatUint(req.CurrentPage, 10))
	}
	if req.PageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp *MarginRepaymentHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getRepaymentHistoryEPL, http.MethodGet, common.EncodeURLValues("/v3/margin/repay", params), nil, &resp)
}

// GetCrossIsolatedMarginInterestRecords returns the interest charged on the cross or isolated margin account's loans
// (Get Interest History)
func (e *Exchange) GetCrossIsolatedMarginInterestRecords(ctx context.Context, req *MarginInterestRecordsRequest) (*MarginInterestRecordsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.IsIsolated && req.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	if req.IsIsolated {
		params.Set("isIsolated", "true")
	}
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.CurrentPage > 0 {
		params.Set("currentPage", strconv.FormatUint(req.CurrentPage, 10))
	}
	if req.PageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp *MarginInterestRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getCrossIsolatedMarginInterestRecordsEPL, http.MethodGet, common.EncodeURLValues("/v3/margin/interest", params), nil, &resp)
}

// ModifyLeverageMultiplier sets the cross or isolated margin account's leverage (Modify Leverage)
func (e *Exchange) ModifyLeverageMultiplier(ctx context.Context, req *ModifyLeverageMultiplierRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.IsIsolated && req.Symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	if req.Leverage <= 0 {
		return errInvalidLeverage
	}
	leverage := strconv.FormatFloat(req.Leverage, 'f', -1, 64)
	if _, decimals, ok := strings.Cut(leverage, "."); ok && len(decimals) > 2 {
		return fmt.Errorf("%w: %s has more than the two decimal places KuCoin takes", errInvalidLeverage, leverage)
	}
	body := map[string]any{"leverage": leverage}
	if req.Symbol != "" {
		body["symbol"] = req.Symbol
	}
	if req.IsIsolated {
		body["isIsolated"] = true
	}
	// data is documented as a string and is null on success
	var resp string
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, modifyLeverageMultiplierEPL, http.MethodPost, "/v3/position/update-user-leverage", body, &resp)
}
