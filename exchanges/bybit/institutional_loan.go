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
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Institutional loan validation errors
var (
	errUIDRequired     = errors.New("UID is required")
	errOperateRequired = errors.New("operate is required")
	errInvalidOperate  = errors.New("operate must be 0 to bind or 1 to unbind")
)

// GetInstitutionalLoanProductInfo returns institutional loan products, or the product given by productID (Get Product
// Info). The request is signed when credentials are set, since Bybit then returns the private product data of a UID
// bound to an institutional loan, and public data otherwise
func (e *Exchange) GetInstitutionalLoanProductInfo(ctx context.Context, productID string) (*InstitutionalLoanProductInfoResponse, error) {
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}
	var resp *InstitutionalLoanProductInfoResponse
	if e.AreCredentialsValid(ctx) {
		return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins-loan/product-infos", params, nil, &resp, getInstitutionalLoanProductInfoEPL)
	}
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/ins-loan/product-infos", params), getInstitutionalLoanProductInfoEPL, &resp)
}

// GetInstitutionalLoanMarginCoinInfo returns the margin coins and their tiered conversion ratios of institutional loan
// products, or of the product given by productID (Get Margin Coin Info). Like GetInstitutionalLoanProductInfo, it signs
// the request when credentials are set so a UID bound to an institutional loan gets its private data
func (e *Exchange) GetInstitutionalLoanMarginCoinInfo(ctx context.Context, productID string) (*InstitutionalLoanMarginCoinInfoResponse, error) {
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}
	var resp *InstitutionalLoanMarginCoinInfoResponse
	if e.AreCredentialsValid(ctx) {
		return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins-loan/ensure-tokens-convert", params, nil, &resp, getInstitutionalLoanMarginCoinInfoEPL)
	}
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/ins-loan/ensure-tokens-convert", params), getInstitutionalLoanMarginCoinInfoEPL, &resp)
}

// GetInstitutionalLoanOrders returns up to two years of institutional loan orders, newest first (Get Loan Orders)
func (e *Exchange) GetInstitutionalLoanOrders(ctx context.Context, req *InstitutionalLoanOrdersRequest) (*InstitutionalLoanOrdersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *InstitutionalLoanOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins-loan/loan-order", params, nil, &resp, getInstitutionalLoanOrdersEPL)
}

// GetInstitutionalLoanRepaymentOrders returns up to two years of institutional loan repayment orders (Get Repayment
// Orders)
func (e *Exchange) GetInstitutionalLoanRepaymentOrders(ctx context.Context, req *InstitutionalLoanRepaymentOrdersRequest) (*InstitutionalLoanRepaymentOrdersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *InstitutionalLoanRepaymentOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins-loan/repaid-history", params, nil, &resp, getInstitutionalLoanRepaymentOrdersEPL)
}

// GetInstitutionalLoanLTV returns the loan-to-value ratio, debts and margin assets of each institutional loan risk
// unit (Get LTV)
func (e *Exchange) GetInstitutionalLoanLTV(ctx context.Context) (*InstitutionalLoanLTVResponse, error) {
	var resp *InstitutionalLoanLTVResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins-loan/ltv-convert", nil, nil, &resp, getInstitutionalLoanLTVEPL)
}

// BindOrUnbindUID binds a UID to, or unbinds it from, the institutional loan risk unit of the API key's UID (Bind Or
// Unbind UID). operate is "0" to bind and "1" to unbind
func (e *Exchange) BindOrUnbindUID(ctx context.Context, uid, operate string) (*BindOrUnbindUIDResponse, error) {
	if uid == "" {
		return nil, errUIDRequired
	}
	if operate == "" {
		return nil, errOperateRequired
	}
	if operate != "0" && operate != "1" {
		return nil, fmt.Errorf("%w: %q", errInvalidOperate, operate)
	}
	var resp *BindOrUnbindUIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/ins-loan/association-uid", nil, map[string]any{
		"uid":     uid,
		"operate": operate,
	}, &resp, bindOrUnbindUIDEPL)
}

// RepayInstitutionalLoan repays institutional loan principal from the unified wallet (Repay). Only the designated risk
// unit UID may call it, and the repayment settles asynchronously over a few minutes
func (e *Exchange) RepayInstitutionalLoan(ctx context.Context, token currency.Code, quantity float64) (*RepayInstitutionalLoanResponse, error) {
	if token.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp *RepayInstitutionalLoanResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/ins-loan/repay-loan", nil, map[string]any{
		"token":    token.Upper().String(),
		"quantity": strconv.FormatFloat(quantity, 'f', -1, 64),
	}, &resp, repayInstitutionalLoanEPL)
}

// GetInstitutionalLoanCoinDeltaAmount returns the delta limits of an institutional loan hedge product's risk unit and
// coins, or of coin only (Get Coin Delta Amount)
func (e *Exchange) GetInstitutionalLoanCoinDeltaAmount(ctx context.Context, coin currency.Code) (*InstitutionalLoanCoinDeltaAmountResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.Upper().String())
	}
	var resp *InstitutionalLoanCoinDeltaAmountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins-loan/coin-delta-amount", params, nil, &resp, defaultEPL)
}

// GetInstitutionalLoanDelayLiquidationStatus returns an institutional loan account's LTV, liquidation status and delay
// liquidation timing (Get Delay Liquidation Status)
func (e *Exchange) GetInstitutionalLoanDelayLiquidationStatus(ctx context.Context) (*InstitutionalLoanDelayLiquidationStatusResponse, error) {
	var resp *InstitutionalLoanDelayLiquidationStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins-loan/delay-liq-status", nil, nil, &resp, getInstitutionalLoanDelayLiquidationStatusEPL)
}
