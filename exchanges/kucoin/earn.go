package kucoin

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// errProductCategoryRequired is returned when a structured product request names no product category
var errProductCategoryRequired = errors.New("product category is required")

// SubscribeToEarnFixedIncomeProduct subscribes to an Earn product (Purchase)
func (e *Exchange) SubscribeToEarnFixedIncomeProduct(ctx context.Context, req *EarnPurchaseRequest) (*EarnPurchaseResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProductID == "" {
		return nil, errProductIDMissing
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.AccountType == "" {
		return nil, errAccountTypeMissing
	}
	body := make(map[string]any)
	body["productId"] = req.ProductID
	body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	body["accountType"] = req.AccountType
	var resp *EarnPurchaseResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, subscribeToEarnFixedIncomeProductEPL, http.MethodPost, "/v1/earn/orders", body, &resp)
}

// GetEarnRedeemPreviewByHoldingID returns what redeeming an Earn holding would pay out; a holding that is fully
// redeemed or being redeemed no longer exists (Get Redeem Preview). fromAccountType only applies to the ETH staking
// holding, whose holding ID is ETH2
func (e *Exchange) GetEarnRedeemPreviewByHoldingID(ctx context.Context, orderID, fromAccountType string) (*EarnRedeemPreviewResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("orderId", orderID)
	if fromAccountType != "" {
		params.Set("fromAccountType", fromAccountType)
	}
	var resp *EarnRedeemPreviewResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getEarnRedeemPreviewByHoldingIDEPL, http.MethodGet, common.EncodeURLValues("/v1/earn/redeem-preview", params), nil, &resp)
}

// RedeemByEarnHoldingID redeems an Earn holding; a holding that is fully redeemed or being redeemed no longer exists
// (Redeem)
func (e *Exchange) RedeemByEarnHoldingID(ctx context.Context, req *EarnRedeemRequest) (*EarnRedeemResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("orderId", req.OrderID)
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	if req.FromAccountType != "" {
		params.Set("fromAccountType", req.FromAccountType)
	}
	if req.ConfirmPunishRedeem {
		params.Set("confirmPunishRedeem", "1")
	}
	var resp *EarnRedeemResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, redeemByEarnHoldingIDEPL, http.MethodDelete, common.EncodeURLValues("/v1/earn/orders", params), nil, &resp)
}

// GetEarnSavingsProducts returns the savings products available, of every currency when ccy is empty (Get Savings
// Products)
func (e *Exchange) GetEarnSavingsProducts(ctx context.Context, ccy currency.Code) ([]EarnProduct, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.Upper().String())
	}
	var resp []EarnProduct
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getEarnSavingsProductsEPL, http.MethodGet, common.EncodeURLValues("/v1/earn/saving/products", params), nil, &resp)
}

// GetLimitedTimePromotionProducts returns the limited-time promotion products available, of every currency when ccy
// is empty (Get Promotion Products)
func (e *Exchange) GetLimitedTimePromotionProducts(ctx context.Context, ccy currency.Code) ([]EarnProduct, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.Upper().String())
	}
	var resp []EarnProduct
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getLimitedTimePromotionProductsEPL, http.MethodGet, common.EncodeURLValues("/v1/earn/promotion/products", params), nil, &resp)
}

// GetEarnStakingProducts returns the staking products available, of every currency when ccy is empty (Get Staking
// Products)
func (e *Exchange) GetEarnStakingProducts(ctx context.Context, ccy currency.Code) ([]EarnProduct, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.Upper().String())
	}
	var resp []EarnProduct
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getEarnStakingProductsEPL, http.MethodGet, common.EncodeURLValues("/v1/earn/staking/products", params), nil, &resp)
}

// GetEarnKCSStakingProducts returns the KCS staking products available, of every currency when ccy is empty (Get KCS
// Staking Products)
func (e *Exchange) GetEarnKCSStakingProducts(ctx context.Context, ccy currency.Code) ([]EarnProduct, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.Upper().String())
	}
	var resp []EarnProduct
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getEarnKCSStakingProductsEPL, http.MethodGet, common.EncodeURLValues("/v1/earn/kcs-staking/products", params), nil, &resp)
}

// GetEarnETHStakingProducts returns the ETH staking products available, of every currency when ccy is empty (Get ETH
// Staking Products)
func (e *Exchange) GetEarnETHStakingProducts(ctx context.Context, ccy currency.Code) ([]EarnProduct, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.Upper().String())
	}
	var resp []EarnProduct
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getEarnETHStakingProductsEPL, http.MethodGet, common.EncodeURLValues("/v1/earn/eth-staking/products", params), nil, &resp)
}

// GetEarnFixedIncomeCurrentHoldings returns a page of the account's current Earn holdings (Get Account Holding)
func (e *Exchange) GetEarnFixedIncomeCurrentHoldings(ctx context.Context, req *EarnAccountHoldingRequest) (*EarnAccountHoldingResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	if req.ProductID != "" {
		params.Set("productId", req.ProductID)
	}
	if req.ProductCategory != "" {
		params.Set("productCategory", req.ProductCategory)
	}
	if req.CurrentPage != 0 {
		params.Set("currentPage", strconv.FormatUint(req.CurrentPage, 10))
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp *EarnAccountHoldingResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getEarnFixedIncomeCurrentHoldingsEPL, http.MethodGet, common.EncodeURLValues("/v1/earn/hold-assets", params), nil, &resp)
}

// PurchaseStructuredProduct subscribes to a structured Earn product such as a dual investment product (Structured
// Product Purchase)
func (e *Exchange) PurchaseStructuredProduct(ctx context.Context, req *StructuredProductPurchaseRequest) (*StructuredProductPurchaseResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ProductID == "" {
		return nil, errProductIDMissing
	}
	if req.InvestCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.InvestAmount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.AccountType == "" {
		return nil, errAccountTypeMissing
	}
	body := make(map[string]any)
	body["productId"] = req.ProductID
	body["investCurrency"] = req.InvestCurrency.Upper().String()
	body["investAmount"] = strconv.FormatFloat(req.InvestAmount, 'f', -1, 64)
	body["accountType"] = req.AccountType
	var resp *StructuredProductPurchaseResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, purchaseStructuredProductEPL, http.MethodPost, "/v1/struct-earn/orders", body, &resp)
}

// GetDualInvestmentProducts returns the dual investment products available in a category (Get Dual Investment
// Products). The docs list it as public, but without an API key it answers code 400001, so it is signed
func (e *Exchange) GetDualInvestmentProducts(ctx context.Context, req *DualInvestmentProductsRequest) ([]DualInvestmentProduct, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errProductCategoryRequired
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if !req.StrikeCurrency.IsEmpty() {
		params.Set("strikeCurrency", req.StrikeCurrency.Upper().String())
	}
	if !req.InvestCurrency.IsEmpty() {
		params.Set("investCurrency", req.InvestCurrency.Upper().String())
	}
	if req.Side != "" {
		params.Set("side", req.Side)
	}
	var resp []DualInvestmentProduct
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getDualInvestmentProductsEPL, http.MethodGet, common.EncodeURLValues("/v1/struct-earn/dual/products", params), nil, &resp)
}

// GetStructuredProductOrders returns a page of the account's structured product orders (Get Structured Product
// Orders)
func (e *Exchange) GetStructuredProductOrders(ctx context.Context, req *StructuredProductOrdersRequest) (*StructuredProductOrdersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if len(req.Categories) == 0 {
		return nil, errProductCategoryRequired
	}
	params := url.Values{}
	params.Set("categories", strings.Join(req.Categories, ","))
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if !req.InvestCurrency.IsEmpty() {
		params.Set("investCurrency", req.InvestCurrency.Upper().String())
	}
	if req.CurrentPage != 0 {
		params.Set("currentPage", strconv.FormatUint(req.CurrentPage, 10))
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp *StructuredProductOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getStructuredProductOrdersEPL, http.MethodGet, common.EncodeURLValues("/v1/struct-earn/orders", params), nil, &resp)
}
