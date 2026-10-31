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
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Crypto loan validation errors
var (
	errCollateralAdjustmentDirectionRequired = errors.New("collateral adjustment direction is required")
	errOrderByRequired                       = errors.New("order by is required")
	errLoanTermRequired                      = errors.New("loan term is required")
	errAnnualRateRequired                    = errors.New("annual rate is required")
	errLoanIDRequired                        = errors.New("loan ID is required")
	errLoanIDOrLoanCurrencyRequired          = errors.New("loan ID or loan currency is required")
)

// GetCryptoLoanBorrowableCoins returns the crypto loan borrowable coins, optionally filtered by VIP level and coin (Get
// Borrowable Coins)
func (e *Exchange) GetCryptoLoanBorrowableCoins(ctx context.Context, vipLevel string, coin currency.Code) (*CryptoLoanBorrowableCoinsResponse, error) {
	params := url.Values{}
	if vipLevel != "" {
		params.Set("vipLevel", vipLevel)
	}
	if !coin.IsEmpty() {
		params.Set("currency", coin.Upper().String())
	}
	var resp *CryptoLoanBorrowableCoinsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/crypto-loan-common/loanable-data", params), defaultEPL, &resp)
}

// GetCryptoLoanCollateralCoins returns the crypto loan collateral ratio tiers and the liquidation order of collateral
// coins, optionally for one coin (Get Collateral Coins)
func (e *Exchange) GetCryptoLoanCollateralCoins(ctx context.Context, coin currency.Code) (*CryptoLoanCollateralCoinsResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("currency", coin.Upper().String())
	}
	var resp *CryptoLoanCollateralCoinsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/crypto-loan-common/collateral-data", params), defaultEPL, &resp)
}

// GetCryptoLoanMaxCollateralReductionAmount returns the maximum amount of a collateral coin that can be withdrawn
// given the position's LTV (Get Max. Allowed Collateral Reduction Amount)
func (e *Exchange) GetCryptoLoanMaxCollateralReductionAmount(ctx context.Context, coin currency.Code) (*CryptoLoanMaxCollateralReductionAmountResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", coin.Upper().String())
	var resp *CryptoLoanMaxCollateralReductionAmountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan-common/max-collateral-amount", params, nil, &resp, getCryptoLoanMaxCollateralReductionAmountEPL)
}

// AdjustCryptoLoanCollateralAmount adds collateral to or removes collateral from the crypto loan position through the
// funding wallet (Adjust Collateral Amount)
func (e *Exchange) AdjustCryptoLoanCollateralAmount(ctx context.Context, req *AdjustCryptoLoanCollateralAmountRequest) (*AdjustCryptoLoanCollateralAmountResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.Direction == "" {
		return nil, errCollateralAdjustmentDirectionRequired
	}
	body := make(map[string]any)
	body["currency"] = req.Currency.Upper().String()
	body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	body["direction"] = req.Direction
	var resp *AdjustCryptoLoanCollateralAmountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/crypto-loan-common/adjust-ltv", nil, body, &resp, adjustCryptoLoanCollateralAmountEPL)
}

// GetCryptoLoanCollateralAdjustmentHistory returns the account's crypto loan collateral adjustments (Get Collateral
// Adjustment History)
func (e *Exchange) GetCryptoLoanCollateralAdjustmentHistory(ctx context.Context, req *CryptoLoanCollateralAdjustmentHistoryRequest) (*CryptoLoanCollateralAdjustmentHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.AdjustID != "" {
		params.Set("adjustId", req.AdjustID)
	}
	if !req.CollateralCurrency.IsEmpty() {
		params.Set("collateralCurrency", req.CollateralCurrency.Upper().String())
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *CryptoLoanCollateralAdjustmentHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan-common/adjustment-history", params, nil, &resp, getCryptoLoanCollateralAdjustmentHistoryEPL)
}

// GetCryptoLoanPosition returns the account's crypto loan debts, collateral and supplies (Get Crypto Loan Position)
func (e *Exchange) GetCryptoLoanPosition(ctx context.Context) (*CryptoLoanPositionResponse, error) {
	var resp *CryptoLoanPositionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan-common/position", nil, nil, &resp, getCryptoLoanPositionEPL)
}

// GetCryptoLoanMaxLoanAmount returns the maximum amount of a coin the account can borrow against its current
// collateral plus any collateral given, whose amounts Bybit checks against the funding wallet (Obtain Max Loan Amount)
func (e *Exchange) GetCryptoLoanMaxLoanAmount(ctx context.Context, coin currency.Code, collateral []CryptoLoanCollateral) (*CryptoLoanMaxLoanAmountResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	for i := range collateral {
		if collateral[i].Currency.IsEmpty() {
			return nil, fmt.Errorf("%w: collateral %d", currency.ErrCurrencyCodeEmpty, i)
		}
		if collateral[i].Amount <= 0 {
			return nil, fmt.Errorf("%w: collateral %d", limits.ErrAmountBelowMin, i)
		}
	}
	body := map[string]any{"currency": coin.Upper().String()}
	if len(collateral) > 0 {
		body["collateralList"] = collateralListPayload(collateral, "ccy")
	}
	var resp *CryptoLoanMaxLoanAmountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/crypto-loan-common/max-loan", nil, body, &resp, getCryptoLoanMaxLoanAmountEPL)
}

// GetFlexibleLoanAvailableInventory returns a coin's flexible loan pool inventory (Get Available Inventory)
func (e *Exchange) GetFlexibleLoanAvailableInventory(ctx context.Context, coin currency.Code) (*FlexibleLoanAvailableInventoryResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", coin.Upper().String())
	var resp *FlexibleLoanAvailableInventoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan-flexible/available-inventory", params, nil, &resp, getFlexibleLoanAvailableInventoryEPL)
}

// BorrowFlexibleLoan borrows a flexible crypto loan into the funding wallet, taking any collateral given from the
// funding wallet (Borrow)
func (e *Exchange) BorrowFlexibleLoan(ctx context.Context, req *BorrowFlexibleLoanRequest) (*CryptoLoanOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.LoanCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.LoanAmount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	body := map[string]any{
		"loanCurrency": req.LoanCurrency.Upper().String(),
		"loanAmount":   strconv.FormatFloat(req.LoanAmount, 'f', -1, 64),
	}
	if len(req.CollateralList) > 0 {
		body["collateralList"] = collateralListPayload(req.CollateralList, "currency")
	}
	var resp *CryptoLoanOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/crypto-loan-flexible/borrow", nil, body, &resp, borrowFlexibleLoanEPL)
}

// RepayFlexibleLoan repays a flexible crypto loan in full or in part from the funding wallet, paying due interest
// first (Repay)
func (e *Exchange) RepayFlexibleLoan(ctx context.Context, loanCurrency currency.Code, amount float64) (*CryptoLoanRepayResponse, error) {
	if loanCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp *CryptoLoanRepayResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/crypto-loan-flexible/repay", nil, map[string]any{
		"loanCurrency": loanCurrency.Upper().String(),
		"amount":       strconv.FormatFloat(amount, 'f', -1, 64),
	}, &resp, repayFlexibleLoanEPL)
}

// RepayFlexibleLoanWithCollateral repays a flexible crypto loan with its collateral, paying interest first and
// charging a repayment fee (Collateral Repayment)
func (e *Exchange) RepayFlexibleLoanWithCollateral(ctx context.Context, req *RepayFlexibleLoanWithCollateralRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	body, err := collateralRepaymentBody(req.LoanCurrency, req.CollateralCoins, req.Amount)
	if err != nil {
		return err
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/crypto-loan-flexible/repay-collateral", nil, body, nil, repayFlexibleLoanWithCollateralEPL)
}

// GetFlexibleLoans returns the account's ongoing flexible crypto loans, optionally for one loan coin (Get Flexible
// Loans)
func (e *Exchange) GetFlexibleLoans(ctx context.Context, loanCurrency currency.Code) (*FlexibleLoansResponse, error) {
	params := url.Values{}
	if !loanCurrency.IsEmpty() {
		params.Set("loanCurrency", loanCurrency.Upper().String())
	}
	var resp *FlexibleLoansResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan-flexible/ongoing-coin", params, nil, &resp, getFlexibleLoansEPL)
}

// GetFlexibleLoanBorrowingHistory returns the account's flexible crypto loan borrows (Get Borrowing History)
func (e *Exchange) GetFlexibleLoanBorrowingHistory(ctx context.Context, req *FlexibleLoanBorrowingHistoryRequest) (*FlexibleLoanBorrowingHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if !req.LoanCurrency.IsEmpty() {
		params.Set("loanCurrency", req.LoanCurrency.Upper().String())
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *FlexibleLoanBorrowingHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan-flexible/borrow-history", params, nil, &resp, getFlexibleLoanBorrowingHistoryEPL)
}

// GetFlexibleLoanRepaymentHistory returns the account's flexible crypto loan repayments (Get Repayment History)
func (e *Exchange) GetFlexibleLoanRepaymentHistory(ctx context.Context, req *FlexibleLoanRepaymentHistoryRequest) (*FlexibleLoanRepaymentHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.RepayID != "" {
		params.Set("repayId", req.RepayID)
	}
	if !req.LoanCurrency.IsEmpty() {
		params.Set("loanCurrency", req.LoanCurrency.Upper().String())
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *FlexibleLoanRepaymentHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan-flexible/repayment-history", params, nil, &resp, getFlexibleLoanRepaymentHistoryEPL)
}

// GetFixedLoanLendingMarket returns the fixed term borrow orders a supplier can fill (Get Lending Market)
func (e *Exchange) GetFixedLoanLendingMarket(ctx context.Context, req *FixedLoanMarketRequest) (*FixedLoanMarketResponse, error) {
	path, err := fixedLoanMarketPath("/v5/crypto-loan-fixed/supply-order-quote", req)
	if err != nil {
		return nil, err
	}
	var resp *FixedLoanMarketResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, path, defaultEPL, &resp)
}

// GetFixedLoanBorrowingMarket returns the fixed term supply orders a borrower can fill (Get Borrowing Market)
func (e *Exchange) GetFixedLoanBorrowingMarket(ctx context.Context, req *FixedLoanMarketRequest) (*FixedLoanMarketResponse, error) {
	path, err := fixedLoanMarketPath("/v5/crypto-loan-fixed/borrow-order-quote", req)
	if err != nil {
		return nil, err
	}
	var resp *FixedLoanMarketResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, path, defaultEPL, &resp)
}

// fixedLoanMarketPath validates a fixed term market request and returns the endpoint path with its query, since the
// lending and borrowing markets take the same parameters
func fixedLoanMarketPath(path string, req *FixedLoanMarketRequest) (string, error) {
	if err := common.NilGuard(req); err != nil {
		return "", err
	}
	if req.OrderCurrency.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if req.OrderBy == "" {
		return "", errOrderByRequired
	}
	params := url.Values{}
	params.Set("orderCurrency", req.OrderCurrency.Upper().String())
	if req.Term > 0 {
		params.Set("term", strconv.FormatUint(req.Term, 10))
	}
	params.Set("orderBy", req.OrderBy)
	if req.Sort > 0 {
		params.Set("sort", strconv.FormatUint(req.Sort, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	return common.EncodeURLValues(path, params), nil
}

// GetFixedLoanAvailableInventory returns a coin's fixed term loan pool inventory at a term and annual rate (Get
// Available Inventory)
func (e *Exchange) GetFixedLoanAvailableInventory(ctx context.Context, req *FixedLoanAvailableInventoryRequest) (*FixedLoanAvailableInventoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Term == 0 {
		return nil, errLoanTermRequired
	}
	if req.AnnualRate <= 0 {
		return nil, errAnnualRateRequired
	}
	params := url.Values{}
	params.Set("currency", req.Currency.Upper().String())
	params.Set("term", strconv.FormatUint(req.Term, 10))
	params.Set("annualRate", strconv.FormatFloat(req.AnnualRate, 'f', -1, 64))
	var resp *FixedLoanAvailableInventoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan-fixed/available-inventory", params, nil, &resp, getFixedLoanAvailableInventoryEPL)
}

// CreateFixedLoanBorrowOrder places a fixed term borrow order, released into the funding wallet when filled, taking
// any collateral given from the funding wallet (Create Borrow Order)
func (e *Exchange) CreateFixedLoanBorrowOrder(ctx context.Context, req *CreateFixedLoanBorrowOrderRequest) (*CryptoLoanOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.OrderAmount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.AnnualRate <= 0 {
		return nil, errAnnualRateRequired
	}
	if req.Term == 0 {
		return nil, errLoanTermRequired
	}
	body := map[string]any{
		"orderCurrency": req.OrderCurrency.Upper().String(),
		"orderAmount":   strconv.FormatFloat(req.OrderAmount, 'f', -1, 64),
		"annualRate":    strconv.FormatFloat(req.AnnualRate, 'f', -1, 64),
		"term":          strconv.FormatUint(req.Term, 10),
	}
	if req.RepayType != "" {
		body["repayType"] = req.RepayType
	}
	if req.StrategyType != "" {
		body["strategyType"] = req.StrategyType
	}
	if len(req.CollateralList) > 0 {
		body["collateralList"] = collateralListPayload(req.CollateralList, "currency")
	}
	var resp *CryptoLoanOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/crypto-loan-fixed/borrow", nil, body, &resp, createFixedLoanBorrowOrderEPL)
}

// RenewFixedLoanBorrowOrder re-borrows the principal already repaid on a fixed term loan, taking any collateral given
// from the funding wallet (Renew Borrow Order)
func (e *Exchange) RenewFixedLoanBorrowOrder(ctx context.Context, loanID string, collateral []CryptoLoanCollateral) (*RenewFixedLoanBorrowOrderResponse, error) {
	if loanID == "" {
		return nil, errLoanIDRequired
	}
	body := map[string]any{"loanId": loanID}
	if len(collateral) > 0 {
		body["collateralList"] = collateralListPayload(collateral, "currency")
	}
	var resp *RenewFixedLoanBorrowOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/crypto-loan-fixed/renew", nil, body, &resp, renewFixedLoanBorrowOrderEPL)
}

// CreateFixedLoanSupplyOrder places a fixed term supply order (Create Supply Order)
func (e *Exchange) CreateFixedLoanSupplyOrder(ctx context.Context, req *CreateFixedLoanSupplyOrderRequest) (*CryptoLoanOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.OrderAmount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.AnnualRate <= 0 {
		return nil, errAnnualRateRequired
	}
	if req.Term == 0 {
		return nil, errLoanTermRequired
	}
	body := map[string]any{
		"orderCurrency": req.OrderCurrency.Upper().String(),
		"orderAmount":   strconv.FormatFloat(req.OrderAmount, 'f', -1, 64),
		"annualRate":    strconv.FormatFloat(req.AnnualRate, 'f', -1, 64),
		"term":          strconv.FormatUint(req.Term, 10),
	}
	if req.AvailableSource != "" {
		body["availableSource"] = req.AvailableSource
	}
	var resp *CryptoLoanOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/crypto-loan-fixed/supply", nil, body, &resp, createFixedLoanSupplyOrderEPL)
}

// CancelFixedLoanBorrowOrder cancels a fixed term borrow order (Cancel Borrow Order)
func (e *Exchange) CancelFixedLoanBorrowOrder(ctx context.Context, orderID string) error {
	if orderID == "" {
		return order.ErrOrderIDNotSet
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/crypto-loan-fixed/borrow-order-cancel", nil, map[string]any{
		"orderId": orderID,
	}, nil, cancelFixedLoanBorrowOrderEPL)
}

// CancelFixedLoanSupplyOrder cancels a fixed term supply order (Cancel Supply Order). refundedAccount is optional: "0"
// refunds the funding account, the default, and "1" refunds EasyEarn
func (e *Exchange) CancelFixedLoanSupplyOrder(ctx context.Context, orderID, refundedAccount string) error {
	if orderID == "" {
		return order.ErrOrderIDNotSet
	}
	body := map[string]any{"orderId": orderID}
	if refundedAccount != "" {
		body["refundedAccount"] = refundedAccount
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/crypto-loan-fixed/supply-order-cancel", nil, body, nil, cancelFixedLoanSupplyOrderEPL)
}

// GetFixedLoanBorrowContractInfo returns the account's fixed term borrow contracts (Get Borrow Contract Info)
func (e *Exchange) GetFixedLoanBorrowContractInfo(ctx context.Context, req *FixedLoanBorrowContractInfoRequest) (*FixedLoanBorrowContractInfoResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.LoanID != "" {
		params.Set("loanId", req.LoanID)
	}
	if !req.OrderCurrency.IsEmpty() {
		params.Set("orderCurrency", req.OrderCurrency.Upper().String())
	}
	if req.Term > 0 {
		params.Set("term", strconv.FormatUint(req.Term, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *FixedLoanBorrowContractInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan-fixed/borrow-contract-info", params, nil, &resp, getFixedLoanBorrowContractInfoEPL)
}

// GetFixedLoanSupplyContractInfo returns the account's fixed term supply contracts (Get Supply Contract Info)
func (e *Exchange) GetFixedLoanSupplyContractInfo(ctx context.Context, req *FixedLoanSupplyContractInfoRequest) (*FixedLoanSupplyContractInfoResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.SupplyID != "" {
		params.Set("supplyId", req.SupplyID)
	}
	if !req.SupplyCurrency.IsEmpty() {
		params.Set("supplyCurrency", req.SupplyCurrency.Upper().String())
	}
	if req.Term > 0 {
		params.Set("term", strconv.FormatUint(req.Term, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *FixedLoanSupplyContractInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan-fixed/supply-contract-info", params, nil, &resp, getFixedLoanSupplyContractInfoEPL)
}

// GetFixedLoanBorrowOrderInfo returns the account's fixed term borrow orders (Get Borrow Order Info)
func (e *Exchange) GetFixedLoanBorrowOrderInfo(ctx context.Context, req *FixedLoanOrderInfoRequest) (*FixedLoanBorrowOrderInfoResponse, error) {
	params, err := fixedLoanOrderInfoParams(req)
	if err != nil {
		return nil, err
	}
	var resp *FixedLoanBorrowOrderInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan-fixed/borrow-order-info", params, nil, &resp, getFixedLoanBorrowOrderInfoEPL)
}

// GetFixedLoanRenewOrderInfo returns the account's fixed term borrow renewals (Get Renew Order Info)
func (e *Exchange) GetFixedLoanRenewOrderInfo(ctx context.Context, req *FixedLoanRenewOrderInfoRequest) (*FixedLoanRenewOrderInfoResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if !req.OrderCurrency.IsEmpty() {
		params.Set("orderCurrency", req.OrderCurrency.Upper().String())
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *FixedLoanRenewOrderInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan-fixed/renew-info", params, nil, &resp, getFixedLoanRenewOrderInfoEPL)
}

// GetFixedLoanSupplyOrderInfo returns the account's fixed term supply orders (Get Supply Order Info)
func (e *Exchange) GetFixedLoanSupplyOrderInfo(ctx context.Context, req *FixedLoanOrderInfoRequest) (*FixedLoanSupplyOrderInfoResponse, error) {
	params, err := fixedLoanOrderInfoParams(req)
	if err != nil {
		return nil, err
	}
	var resp *FixedLoanSupplyOrderInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan-fixed/supply-order-info", params, nil, &resp, getFixedLoanSupplyOrderInfoEPL)
}

// fixedLoanOrderInfoParams returns the query of a fixed term borrow or supply order request, since both endpoints take
// the same parameters
func fixedLoanOrderInfoParams(req *FixedLoanOrderInfoRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if !req.OrderCurrency.IsEmpty() {
		params.Set("orderCurrency", req.OrderCurrency.Upper().String())
	}
	if req.State != "" {
		params.Set("state", req.State)
	}
	if req.Term > 0 {
		params.Set("term", strconv.FormatUint(req.Term, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	return params, nil
}

// RepayFixedLoan fully repays a fixed term loan contract, or every contract of a loan coin (Repay)
func (e *Exchange) RepayFixedLoan(ctx context.Context, loanID string, loanCurrency currency.Code) (*CryptoLoanRepayResponse, error) {
	if loanID == "" && loanCurrency.IsEmpty() {
		return nil, errLoanIDOrLoanCurrencyRequired
	}
	body := map[string]any{}
	if loanID != "" {
		body["loanId"] = loanID
	}
	if !loanCurrency.IsEmpty() {
		body["loanCurrency"] = loanCurrency.Upper().String()
	}
	var resp *CryptoLoanRepayResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/crypto-loan-fixed/fully-repay", nil, body, &resp, repayFixedLoanEPL)
}

// RepayFixedLoanWithCollateral repays fixed term loans with their collateral, paying interest first and charging a
// repayment fee (Collateral Repayment)
func (e *Exchange) RepayFixedLoanWithCollateral(ctx context.Context, req *RepayFixedLoanWithCollateralRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	body, err := collateralRepaymentBody(req.LoanCurrency, req.CollateralCoins, req.Amount)
	if err != nil {
		return err
	}
	if req.LoanID != "" {
		body["loanId"] = req.LoanID
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/crypto-loan-fixed/repay-collateral", nil, body, nil, repayFixedLoanWithCollateralEPL)
}

// GetFixedLoanRepaymentHistory returns the account's fixed term loan repayments (Get Repayment History)
func (e *Exchange) GetFixedLoanRepaymentHistory(ctx context.Context, req *FixedLoanRepaymentHistoryRequest) (*FixedLoanRepaymentHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.RepayID != "" {
		params.Set("repayId", req.RepayID)
	}
	if !req.LoanCurrency.IsEmpty() {
		params.Set("loanCurrency", req.LoanCurrency.Upper().String())
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *FixedLoanRepaymentHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/crypto-loan-fixed/repayment-history", params, nil, &resp, getFixedLoanRepaymentHistoryEPL)
}

// collateralListPayload returns the documented collateralList objects, leaving out unset fields; coinKey names the
// coin field, which Obtain Max Loan Amount calls ccy and the borrow endpoints call currency
func collateralListPayload(collateral []CryptoLoanCollateral, coinKey string) []map[string]any {
	list := make([]map[string]any, len(collateral))
	for i := range collateral {
		entry := make(map[string]any, 2)
		if !collateral[i].Currency.IsEmpty() {
			entry[coinKey] = collateral[i].Currency.Upper().String()
		}
		if collateral[i].Amount > 0 {
			entry["amount"] = strconv.FormatFloat(collateral[i].Amount, 'f', -1, 64)
		}
		list[i] = entry
	}
	return list
}

// collateralRepaymentBody validates the parameters the flexible and fixed collateral repayment endpoints share and
// returns them as a request body, with the collateral coins joined by commas
func collateralRepaymentBody(loanCurrency currency.Code, collateralCoins []currency.Code, amount float64) (map[string]any, error) {
	if loanCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: loan currency", currency.ErrCurrencyCodeEmpty)
	}
	if len(collateralCoins) == 0 {
		return nil, fmt.Errorf("%w: collateral coin", currency.ErrCurrencyCodeEmpty)
	}
	coins := make([]string, len(collateralCoins))
	for i := range collateralCoins {
		if collateralCoins[i].IsEmpty() {
			return nil, fmt.Errorf("%w: collateral coin %d", currency.ErrCurrencyCodeEmpty, i)
		}
		coins[i] = collateralCoins[i].Upper().String()
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	return map[string]any{
		"loanCurrency":   loanCurrency.Upper().String(),
		"collateralCoin": strings.Join(coins, ","),
		"amount":         strconv.FormatFloat(amount, 'f', -1, 64),
	}, nil
}
