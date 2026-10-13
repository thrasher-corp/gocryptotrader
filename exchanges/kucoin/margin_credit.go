package kucoin

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// errInvalidLoanMarketInterestRateTime is returned when a loan market interest rate time is not YYYYMMDDHH00
var errInvalidLoanMarketInterestRateTime = errors.New("invalid loan market interest rate time")

// GetLendingCurrencyInformation returns the margin lending market configuration of every currency, or of the
// currency given (Get Loan Market)
func (e *Exchange) GetLendingCurrencyInformation(ctx context.Context, ccy currency.Code) ([]LendingCurrencyInfo, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.Upper().String())
	}
	var resp []LendingCurrencyInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getLendingCurrencyInformationEPL, http.MethodGet, common.EncodeURLValues("/v3/project/list", params), nil, &resp)
}

// GetLoanMarketInterestRate returns a currency's hourly margin lending market interest rate over the past 7 days (Get
// Loan Market Interest Rate)
func (e *Exchange) GetLoanMarketInterestRate(ctx context.Context, ccy currency.Code) ([]LoanMarketInterestRate, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.Upper().String())
	var resp []LoanMarketInterestRate
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getLoanMarketInterestRateEPL, common.EncodeURLValues("/v3/project/marketInterestRate", params), &resp)
}

// MarginLendingSubscription lends funds from the funding account on the margin lending market (Purchase)
func (e *Exchange) MarginLendingSubscription(ctx context.Context, req *MarginLendingSubscriptionRequest) (*OrderNumberResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Size <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.InterestRate <= 0 {
		return nil, errMissingInterestRate
	}
	body := make(map[string]any)
	body["currency"] = req.Currency.Upper().String()
	body["size"] = strconv.FormatFloat(req.Size, 'f', -1, 64)
	body["interestRate"] = strconv.FormatFloat(req.InterestRate, 'f', -1, 64)
	var resp *OrderNumberResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, marginLendingSubscriptionEPL, http.MethodPost, "/v3/purchase", body, &resp)
}

// ModifySubscriptionOrder changes a purchase order's interest rate from the start of the next hour (Modify Purchase)
func (e *Exchange) ModifySubscriptionOrder(ctx context.Context, req *ModifySubscriptionOrderRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if req.InterestRate <= 0 {
		return errMissingInterestRate
	}
	if req.PurchaseOrderNumber == "" {
		return errMissingPurchaseOrderNumber
	}
	body := make(map[string]any)
	body["currency"] = req.Currency.Upper().String()
	body["interestRate"] = strconv.FormatFloat(req.InterestRate, 'f', -1, 64)
	body["purchaseOrderNo"] = req.PurchaseOrderNumber
	// data is documented as a string and is null on success
	var resp string
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, modifySubscriptionOrderEPL, http.MethodPost, "/v3/lend/purchase/update", body, &resp)
}

// GetSubscriptionOrders returns the margin lending purchase orders (Get Purchase Orders)
func (e *Exchange) GetSubscriptionOrders(ctx context.Context, req *SubscriptionOrdersRequest) (*SubscriptionOrdersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Status == "" {
		return nil, errStatusMissing
	}
	params := url.Values{}
	params.Set("status", req.Status)
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	if req.PurchaseOrderNumber != "" {
		params.Set("purchaseOrderNo", req.PurchaseOrderNumber)
	}
	if req.CurrentPage > 0 {
		params.Set("currentPage", strconv.FormatUint(req.CurrentPage, 10))
	}
	if req.PageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp *SubscriptionOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getSubscriptionOrdersEPL, http.MethodGet, common.EncodeURLValues("/v3/purchase/orders", params), nil, &resp)
}

// Redemption redeems funds lent by a margin lending purchase order (Redeem)
func (e *Exchange) Redemption(ctx context.Context, req *RedemptionRequest) (*OrderNumberResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Size <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.PurchaseOrderNumber == "" {
		return nil, errMissingPurchaseOrderNumber
	}
	body := make(map[string]any)
	body["currency"] = req.Currency.Upper().String()
	body["size"] = strconv.FormatFloat(req.Size, 'f', -1, 64)
	body["purchaseOrderNo"] = req.PurchaseOrderNumber
	var resp *OrderNumberResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, redemptionEPL, http.MethodPost, "/v3/redeem", body, &resp)
}

// GetRedemptionOrders returns the margin lending redemption orders (Get Redeem Orders)
func (e *Exchange) GetRedemptionOrders(ctx context.Context, req *RedemptionOrdersRequest) (*RedemptionOrdersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Status == "" {
		return nil, errStatusMissing
	}
	params := url.Values{}
	params.Set("status", req.Status)
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	if req.RedeemOrderNumber != "" {
		params.Set("redeemOrderNo", req.RedeemOrderNumber)
	}
	if req.CurrentPage > 0 {
		params.Set("currentPage", strconv.FormatUint(req.CurrentPage, 10))
	}
	if req.PageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp *RedemptionOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getRedemptionOrdersEPL, http.MethodGet, common.EncodeURLValues("/v3/redeem/orders", params), nil, &resp)
}
