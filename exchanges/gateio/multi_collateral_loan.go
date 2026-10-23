package gateio

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// PlaceMultiCollateralLoanOrder places a multi-currency collateral loan order
func (e *Exchange) PlaceMultiCollateralLoanOrder(ctx context.Context, arg *MultiCollateralLoanOrderRequest) (*OrderIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.BorrowCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.BorrowAmount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.OrderType == "fixed" {
		if arg.FixedType == "" {
			return nil, errFixedRateTermRequired
		}
		if arg.FixedRate <= 0 {
			return nil, errFixedRateRequired
		}
	}
	var resp *OrderIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralCreateOrderEPL, http.MethodPost, "loan/multi_collateral/orders", nil, arg, &resp)
}

// GetMultiCollateralLoanOrders retrieves multi-currency collateral loan orders
// sortType is time_desc, ltv_asc or ltv_desc and orderType is current or fixed
func (e *Exchange) GetMultiCollateralLoanOrders(ctx context.Context, page, limit uint64, sortType, orderType string) ([]*MultiCollateralLoanOrderDetailResponse, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if sortType != "" {
		params.Set("sort", sortType)
	}
	if orderType != "" {
		params.Set("order_type", orderType)
	}
	var resp []*MultiCollateralLoanOrderDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralOrdersEPL, http.MethodGet, "loan/multi_collateral/orders", params, nil, &resp)
}

// GetMultiCollateralLoanOrderDetail retrieves a multi-currency collateral loan order
func (e *Exchange) GetMultiCollateralLoanOrderDetail(ctx context.Context, orderID string) (*MultiCollateralLoanOrderDetailResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *MultiCollateralLoanOrderDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralOrderDetailEPL, http.MethodGet, "loan/multi_collateral/orders/"+orderID, nil, nil, &resp)
}

// RepayMultiCollateralLoan repays a multi-currency collateral loan
func (e *Exchange) RepayMultiCollateralLoan(ctx context.Context, arg *MultiCollateralLoanRepaymentRequest) (*MultiCollateralLoanRepaymentResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if len(arg.RepayItems) == 0 {
		return nil, fmt.Errorf("%w: at least one repay item is required", errNoValidParameterPassed)
	}
	for _, item := range arg.RepayItems {
		if item.Currency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		if !item.RepaidAll && item.Amount <= 0 {
			return nil, fmt.Errorf("%w: partial repayments require an amount", order.ErrAmountIsInvalid)
		}
	}
	var resp *MultiCollateralLoanRepaymentResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralRepayEPL, http.MethodPost, "loan/multi_collateral/repay", nil, arg, &resp)
}

// GetMultiCurrencyCollateralRepaymentRecords retrieves multi-currency collateral repayment records
func (e *Exchange) GetMultiCurrencyCollateralRepaymentRecords(ctx context.Context, arg *MultiCollateralRepaymentRecordsRequest) ([]*MultiCurrencyCollateralRepaymentResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Type == "" {
		return nil, errLoanTypeIsRequired
	}
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
		return nil, err
	}
	params.Set("type", arg.Type)
	if !arg.BorrowCurrency.IsEmpty() {
		params.Set("borrow_currency", arg.BorrowCurrency.String())
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []*MultiCurrencyCollateralRepaymentResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralRepayRecordsEPL, http.MethodGet, "loan/multi_collateral/repay", params, nil, &resp)
}

// AddOrWithdrawCollateral adds collateral to or withdraws collateral from a multi-currency collateral loan
func (e *Exchange) AddOrWithdrawCollateral(ctx context.Context, arg *AddOrWithdrawCollateralRequest) (*CollateralAddOrRemoveResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.OperationType == "" {
		return nil, errLoanTypeIsRequired
	}
	var resp *CollateralAddOrRemoveResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralOperateEPL, http.MethodPost, "loan/multi_collateral/mortgage", nil, arg, &resp)
}

// GetMultiCollateralAdjustmentRecords retrieves multi-currency collateral adjustment records
func (e *Exchange) GetMultiCollateralAdjustmentRecords(ctx context.Context, arg *MultiCollateralAdjustmentRecordsRequest) ([]*MultiCollateralAdjustmentRecordResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
		return nil, err
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if !arg.CollateralCurrency.IsEmpty() {
		params.Set("collateral_currency", arg.CollateralCurrency.String())
	}
	var resp []*MultiCollateralAdjustmentRecordResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralMortgageEPL, http.MethodGet, "loan/multi_collateral/mortgage", params, nil, &resp)
}

// GetMultiCollateralCurrencyQuota retrieves the user's collateral or borrowing quota per currency
// quotaType is collateral or borrow; borrow quotas accept a single currency
func (e *Exchange) GetMultiCollateralCurrencyQuota(ctx context.Context, quotaType string, currencies []currency.Code) ([]*MultiCollateralCurrencyQuotaResponse, error) {
	if quotaType == "" {
		return nil, errCurrencyTypeRequired
	}
	if len(currencies) == 0 {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if quotaType == "borrow" && len(currencies) > 1 {
		return nil, fmt.Errorf("%w: borrow quotas accept a single currency", errTooManyCurrencyCodes)
	}
	params := url.Values{}
	params.Set("type", quotaType)
	params.Set("currency", currency.Currencies(currencies).Join())
	var resp []*MultiCollateralCurrencyQuotaResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralCurrencyQuotaEPL, http.MethodGet, "loan/multi_collateral/currency_quota", params, nil, &resp)
}

// GetMultiCollateralSupportedCurrencies retrieves the borrowing and collateral currencies supported by multi-currency collateral loans
func (e *Exchange) GetMultiCollateralSupportedCurrencies(ctx context.Context) (*MultiCollateralSupportedCurrenciesResponse, error) {
	var resp *MultiCollateralSupportedCurrenciesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralCurrenciesEPL, "loan/multi_collateral/currencies", &resp)
}

// GetMultiCollateralisationRatio retrieves the multi-currency collateral ratios, which are the same for every currency
func (e *Exchange) GetMultiCollateralisationRatio(ctx context.Context) (*MultiCollateralisationRatioResponse, error) {
	var resp *MultiCollateralisationRatioResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralLtvEPL, "loan/multi_collateral/ltv", &resp)
}

// GetMultiCollateralFixedRates retrieves each currency's 7-day and 30-day fixed interest rates
func (e *Exchange) GetMultiCollateralFixedRates(ctx context.Context) ([]*CollateralFixRateResponse, error) {
	var resp []*CollateralFixRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralFixedRateEPL, "loan/multi_collateral/fixed_rate", &resp)
}

// GetMultiCollateralCurrentRates retrieves the previous hour's interest rate for up to 100 currencies
// vipLevel defaults to 0 when empty
func (e *Exchange) GetMultiCollateralCurrentRates(ctx context.Context, currencies []currency.Code, vipLevel string) ([]*CollateralCurrentRateResponse, error) {
	if len(currencies) == 0 {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if len(currencies) > 100 {
		return nil, fmt.Errorf("%w: maximum 100", errTooManyCurrencyCodes)
	}
	params := url.Values{}
	params.Set("currencies", currency.Currencies(currencies).Join())
	if vipLevel != "" {
		params.Set("vip_level", vipLevel)
	}
	var resp []*CollateralCurrentRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralCurrentRateEPL, common.EncodeURLValues("loan/multi_collateral/current_rate", params), &resp)
}
