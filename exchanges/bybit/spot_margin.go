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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Spot margin errors
var (
	errAnnualRateNotSet             = errors.New("annual rate not set")
	errTermNotSet                   = errors.New("term not set")
	errLoanIDNotSet                 = errors.New("loan ID not set")
	errFixedRateBorrowRenewalFailed = errors.New("fixed-rate borrow renewal failed")
)

// GetVIPMarginData returns the spot margin borrowing terms of each VIP level, or of one VIP level or coin
// (Get VIP Margin Data)
func (e *Exchange) GetVIPMarginData(ctx context.Context, vipLevel string, coin currency.Code) (*VIPMarginDataResponse, error) {
	params := url.Values{}
	if vipLevel != "" {
		params.Set("vipLevel", vipLevel)
	}
	if !coin.IsEmpty() {
		params.Set("currency", coin.Upper().String())
	}
	var resp *VIPMarginDataResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/spot-margin-trade/data", params), defaultEPL, &resp)
}

// GetTieredCollateralRatio returns the tiered collateral ratios of unified account loans, for every coin or one
// (Get Tiered Collateral Ratio)
func (e *Exchange) GetTieredCollateralRatio(ctx context.Context, coin currency.Code) (*TieredCollateralRatioResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("currency", coin.Upper().String())
	}
	var resp *TieredCollateralRatioResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/spot-margin-trade/collateral", params), defaultEPL, &resp)
}

// GetSpotMarginCurrencyData returns the manual borrowing settings of every coin or one. Bybit's examples sign it, but
// it answers without authentication (Get Currency Data)
func (e *Exchange) GetSpotMarginCurrencyData(ctx context.Context, coin currency.Code) (*SpotMarginCurrencyDataResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("currency", coin.Upper().String())
	}
	var resp *SpotMarginCurrencyDataResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/spot-margin-trade/currency-data", params), defaultEPL, &resp)
}

// GetSpotMarginHistoricalInterestRate returns a coin's hourly borrowing rates of up to six months ago
// (Get Historical Interest Rate)
func (e *Exchange) GetSpotMarginHistoricalInterestRate(ctx context.Context, r *SpotMarginHistoricalInterestRateRequest) (*SpotMarginHistoricalInterestRateResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if !r.StartTime.IsZero() || !r.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(r.StartTime, r.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("currency", r.Currency.Upper().String())
	if r.VIPLevel != "" {
		params.Set("vipLevel", r.VIPLevel)
	}
	if !r.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(r.StartTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(r.EndTime.UnixMilli(), 10))
	}
	var resp *SpotMarginHistoricalInterestRateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-margin-trade/interest-rate-history", params, nil, &resp, defaultEPL)
}

// ToggleMarginTrade turns spot margin trading on or off. The account must have activated spot margin trading on the
// web or app first (Toggle Margin Trade)
func (e *Exchange) ToggleMarginTrade(ctx context.Context, spotMarginMode bool) (*SpotMarginModeResponse, error) {
	body := make(map[string]any)
	body["spotMarginMode"] = "0"
	if spotMarginMode {
		body["spotMarginMode"] = "1"
	}
	var resp *SpotMarginModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-margin-trade/switch-mode", nil, body, &resp, toggleMarginTradeEPL)
}

// SetSpotMarginTradeLeverage sets the account's maximum spot margin leverage, from 2 to 10, or a coin's
// (Set Leverage)
func (e *Exchange) SetSpotMarginTradeLeverage(ctx context.Context, leverage float64, coin currency.Code) error {
	if leverage <= 0 {
		return fmt.Errorf("%w: %v", errInvalidLeverage, leverage)
	}
	body := make(map[string]any)
	body["leverage"] = strconv.FormatFloat(leverage, 'f', -1, 64)
	if !coin.IsEmpty() {
		body["currency"] = coin.Upper().String()
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-margin-trade/set-leverage", nil, body, nil, setSpotMarginTradeLeverageEPL)
}

// GetSpotMarginStatusAndLeverage returns whether spot margin trading is on and its leverage
// (Get Status And Leverage)
func (e *Exchange) GetSpotMarginStatusAndLeverage(ctx context.Context) (*SpotMarginStatusResponse, error) {
	var resp *SpotMarginStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-margin-trade/state", nil, nil, &resp, getSpotMarginStatusAndLeverageEPL)
}

// GetMaxBorrowableAmount returns the most of a coin the account can borrow (Get Max Borrowable Amount)
func (e *Exchange) GetMaxBorrowableAmount(ctx context.Context, coin currency.Code) (*MaxBorrowableAmountResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", coin.Upper().String())
	var resp *MaxBorrowableAmountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-margin-trade/max-borrowable", params, nil, &resp, getMaxBorrowableAmountEPL)
}

// GetSpotMarginPositionTiers returns the borrowing risk tiers of every configured coin or one (Get Position Tiers)
func (e *Exchange) GetSpotMarginPositionTiers(ctx context.Context, coin currency.Code) (*SpotMarginPositionTiersResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("currency", coin.Upper().String())
	}
	var resp *SpotMarginPositionTiersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-margin-trade/position-tiers", params, nil, &resp, defaultEPL)
}

// GetSpotMarginCoinState returns the spot margin leverage of every coin or one (Get Coin State)
func (e *Exchange) GetSpotMarginCoinState(ctx context.Context, coin currency.Code) (*SpotMarginCoinStateResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("currency", coin.Upper().String())
	}
	var resp *SpotMarginCoinStateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-margin-trade/coinstate", params, nil, &resp, getSpotMarginCoinStateEPL)
}

// GetSpotMarginAvailableAmountToRepay returns how much of a coin the account can repay without converting other
// assets (Get Available Amount to Repay)
func (e *Exchange) GetSpotMarginAvailableAmountToRepay(ctx context.Context, coin currency.Code) (*SpotMarginAvailableAmountToRepayResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", coin.Upper().String())
	var resp *SpotMarginAvailableAmountToRepayResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-margin-trade/repayment-available-amount", params, nil, &resp, getSpotMarginAvailableAmountToRepayEPL)
}

// SetSpotMarginAutoRepayMode turns automatic repayment on or off for a coin, or for every coin when coin is empty
// (Set Auto Repay Mode)
func (e *Exchange) SetSpotMarginAutoRepayMode(ctx context.Context, coin currency.Code, autoRepayMode bool) (*SpotMarginAutoRepayModeResponse, error) {
	body := make(map[string]any)
	if !coin.IsEmpty() {
		body["currency"] = coin.Upper().String()
	}
	body["autoRepayMode"] = "0"
	if autoRepayMode {
		body["autoRepayMode"] = "1"
	}
	var resp *SpotMarginAutoRepayModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-margin-trade/set-auto-repay-mode", nil, body, &resp, setSpotMarginAutoRepayModeEPL)
}

// GetSpotMarginAutoRepayMode returns whether a coin, or every coin when coin is empty, is repaid automatically
// (Get Auto Repay Mode)
func (e *Exchange) GetSpotMarginAutoRepayMode(ctx context.Context, coin currency.Code) (*SpotMarginAutoRepayModeResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("currency", coin.Upper().String())
	}
	var resp *SpotMarginAutoRepayModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-margin-trade/get-auto-repay-mode", params, nil, &resp, getSpotMarginAutoRepayModeEPL)
}

// GetSpotMarginFixedRateBorrowOrderQuote returns the fixed-rate lending offers of a coin
// (Get Fixed-Rate Borrow Order Quote)
func (e *Exchange) GetSpotMarginFixedRateBorrowOrderQuote(ctx context.Context, r *SpotMarginFixedRateBorrowOrderQuoteRequest) (*SpotMarginFixedRateBorrowOrderQuoteResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.OrderCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("orderCurrency", r.OrderCurrency.Upper().String())
	if r.Term != 0 {
		params.Set("term", strconv.FormatUint(r.Term, 10))
	}
	if r.OrderBy != "" {
		params.Set("orderBy", r.OrderBy)
	}
	if r.Sort != 0 {
		params.Set("sort", strconv.FormatUint(r.Sort, 10))
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	var resp *SpotMarginFixedRateBorrowOrderQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-margin-trade/fixedborrow-order-quote", params, nil, &resp, getSpotMarginFixedRateBorrowOrderQuoteEPL)
}

// SpotMarginFixedRateBorrow places an order to borrow a coin for a term at a fixed interest rate (Fixed-Rate Borrow)
func (e *Exchange) SpotMarginFixedRateBorrow(ctx context.Context, r *SpotMarginFixedRateBorrowRequest) (*SpotMarginFixedRateBorrowResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.OrderCurrency.IsEmpty():
		return nil, currency.ErrCurrencyCodeEmpty
	case r.OrderAmount <= 0:
		return nil, limits.ErrAmountBelowMin
	case r.AnnualRate <= 0:
		return nil, errAnnualRateNotSet
	case r.Term == 0:
		return nil, errTermNotSet
	}
	body := make(map[string]any)
	body["orderCurrency"] = r.OrderCurrency.Upper().String()
	body["orderAmount"] = strconv.FormatFloat(r.OrderAmount, 'f', -1, 64)
	body["annualRate"] = strconv.FormatFloat(r.AnnualRate, 'f', -1, 64)
	body["term"] = strconv.FormatUint(r.Term, 10)
	if r.RepayType != "" {
		body["repayType"] = r.RepayType
	}
	if r.StrategyType != "" {
		body["strategyType"] = r.StrategyType
	}
	var resp *SpotMarginFixedRateBorrowResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-margin-trade/fixedborrow", nil, body, &resp, spotMarginFixedRateBorrowEPL)
}

// RenewSpotMarginFixedRateBorrow renews a fixed-rate loan, all of its remaining amount when quantity is zero. Bybit
// reports a failed renewal in the result rather than as an error, so it is returned as one (Renew Fixed-Rate Borrow)
func (e *Exchange) RenewSpotMarginFixedRateBorrow(ctx context.Context, loanID string, quantity float64) error {
	if loanID == "" {
		return errLoanIDNotSet
	}
	body := make(map[string]any)
	body["loanId"] = loanID
	if quantity != 0 {
		body["qty"] = strconv.FormatFloat(quantity, 'f', -1, 64)
	}
	var result string
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-margin-trade/fixedborrow-renew", nil, body, &result, renewSpotMarginFixedRateBorrowEPL); err != nil {
		return err
	}
	if result != "Success" {
		return fmt.Errorf("%w: loan %s: %s", errFixedRateBorrowRenewalFailed, loanID, result)
	}
	return nil
}

// GetSpotMarginFixedRateBorrowOrderInfo returns the account's fixed-rate loan orders, newest first
// (Get Fixed-Rate Borrow Order Info)
func (e *Exchange) GetSpotMarginFixedRateBorrowOrderInfo(ctx context.Context, r *SpotMarginFixedRateBorrowOrderInfoRequest) (*SpotMarginFixedRateBorrowOrderInfoResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	params := url.Values{}
	if r.OrderID != "" {
		params.Set("orderId", r.OrderID)
	}
	if !r.OrderCurrency.IsEmpty() {
		params.Set("orderCurrency", r.OrderCurrency.Upper().String())
	}
	if r.State != 0 {
		params.Set("state", strconv.FormatUint(r.State, 10))
	}
	if r.Term != 0 {
		params.Set("term", strconv.FormatUint(r.Term, 10))
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	if r.Cursor != "" {
		params.Set("cursor", r.Cursor)
	}
	var resp *SpotMarginFixedRateBorrowOrderInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-margin-trade/fixedborrow-order-info", params, nil, &resp, getSpotMarginFixedRateBorrowOrderInfoEPL)
}

// GetSpotMarginFixedRateBorrowContractInfo returns the account's fixed-rate loans, newest first
// (Get Fixed-Rate Borrow Contract Info)
func (e *Exchange) GetSpotMarginFixedRateBorrowContractInfo(ctx context.Context, r *SpotMarginFixedRateBorrowContractInfoRequest) (*SpotMarginFixedRateBorrowContractInfoResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	params := url.Values{}
	if r.OrderID != "" {
		params.Set("orderId", r.OrderID)
	}
	if !r.OrderCurrency.IsEmpty() {
		params.Set("orderCurrency", r.OrderCurrency.Upper().String())
	}
	if r.Term != 0 {
		params.Set("term", strconv.FormatUint(r.Term, 10))
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	if r.Cursor != "" {
		params.Set("cursor", r.Cursor)
	}
	var resp *SpotMarginFixedRateBorrowContractInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-margin-trade/fixedborrow-contract-info", params, nil, &resp, getSpotMarginFixedRateBorrowContractInfoEPL)
}

// GetSpotMarginLiability returns the account's fixed-rate, flexible-rate, spot and derivatives liability in a coin
// (Get Liability Info)
func (e *Exchange) GetSpotMarginLiability(ctx context.Context, coin currency.Code) (*SpotMarginLiabilityResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", coin.Upper().String())
	var resp *SpotMarginLiabilityResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-margin-trade/liability", params, nil, &resp, getSpotMarginLiabilityEPL)
}

// GetSpotMarginFlexibleAvailableInventory returns how much of a coin the account can borrow at a flexible rate
// (Get Flexible Available Inventory)
func (e *Exchange) GetSpotMarginFlexibleAvailableInventory(ctx context.Context, coin currency.Code) (*SpotMarginFlexibleAvailableInventoryResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", coin.Upper().String())
	var resp *SpotMarginFlexibleAvailableInventoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-margin-trade/flexible-available-inventory", params, nil, &resp, defaultEPL)
}

// GetSpotMarginFixedRateAvailableInventory returns how much of a coin the account can borrow for a term at an
// interest rate (Get Fixed-Rate Available Inventory)
func (e *Exchange) GetSpotMarginFixedRateAvailableInventory(ctx context.Context, r *SpotMarginFixedRateAvailableInventoryRequest) (*SpotMarginFixedRateAvailableInventoryResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.Currency.IsEmpty():
		return nil, currency.ErrCurrencyCodeEmpty
	case r.Term == 0:
		return nil, errTermNotSet
	case r.AnnualRate <= 0:
		return nil, errAnnualRateNotSet
	}
	params := url.Values{}
	params.Set("currency", r.Currency.Upper().String())
	params.Set("term", strconv.FormatUint(r.Term, 10))
	params.Set("annualRate", strconv.FormatFloat(r.AnnualRate, 'f', -1, 64))
	var resp *SpotMarginFixedRateAvailableInventoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-margin-trade/fixed-available-inventory", params, nil, &resp, defaultEPL)
}

// GetSpotMarginFixedRateBorrowPenaltyInterest returns the penalty interest charged on the account's fixed-rate loans,
// newest first; without a cursor it returns the last 30 days' records (Query Fixed-Rate Borrow Penalty Interest)
func (e *Exchange) GetSpotMarginFixedRateBorrowPenaltyInterest(ctx context.Context, r *SpotMarginFixedRateBorrowPenaltyInterestRequest) (*SpotMarginFixedRateBorrowPenaltyInterestResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	params := url.Values{}
	if r.OrderID != "" {
		params.Set("orderId", r.OrderID)
	}
	if !r.OrderCurrency.IsEmpty() {
		params.Set("orderCurrency", r.OrderCurrency.Upper().String())
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	if r.Cursor != "" {
		params.Set("cursor", r.Cursor)
	}
	var resp *SpotMarginFixedRateBorrowPenaltyInterestResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-margin-trade/fixedborrow-interest-accrued", params, nil, &resp, getSpotMarginFixedRateBorrowPenaltyInterestEPL)
}

// CancelSpotMarginFixedRateBorrowOrder asks Bybit to cancel a fixed-rate borrow order that is still cancellable. Its
// reply only confirms the request was accepted, so the order's state shows whether it was cancelled (Cancel
// Fixed-Rate Borrow Order)
func (e *Exchange) CancelSpotMarginFixedRateBorrowOrder(ctx context.Context, orderID string) error {
	if orderID == "" {
		return order.ErrOrderIDNotSet
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-margin-trade/fixedborrow-order-cancel", nil, map[string]any{
		"orderId": orderID,
	}, nil, cancelSpotMarginFixedRateBorrowOrderEPL)
}
