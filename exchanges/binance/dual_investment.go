package binance

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

var errAutoCompoundPlanRequired = errors.New("auto-compound plan is required")

// GetDualInvestmentProductList returns the Dual Investment products for an option type and asset pair. The schema
// lists this endpoint without security, but Binance rejects it without an API key and earlier docs titled it
// USER_DATA, so it is sent signed
func (e *Exchange) GetDualInvestmentProductList(ctx context.Context, req *DualInvestmentProductListRequest) (*DualInvestmentProductListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OptionType == "" {
		return nil, errOptionTypeRequired
	}
	if req.ExercisedCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: exercised coin", currency.ErrCurrencyCodeEmpty)
	}
	if req.InvestCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: invest coin", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("optionType", req.OptionType)
	params.Set("exercisedCoin", req.ExercisedCoin.Upper().String())
	params.Set("investCoin", req.InvestCoin.Upper().String())
	if req.PageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	if req.PageIndex > 0 {
		params.Set("pageIndex", strconv.FormatUint(req.PageIndex, 10))
	}
	var resp *DualInvestmentProductListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/dci/product/list", params, sapiDefaultRate, &resp)
}

// ChangeAutoCompoundStatus changes the auto-compound plan of a Dual Investment position
func (e *Exchange) ChangeAutoCompoundStatus(ctx context.Context, positionID, autoCompoundPlan string) (*DualInvestmentAutoCompoundStatusResponse, error) {
	if positionID == "" {
		return nil, errPositionIDRequired
	}
	if autoCompoundPlan == "" {
		return nil, errAutoCompoundPlanRequired
	}
	params := url.Values{}
	params.Set("positionId", positionID)
	params.Set("autoCompoundPlan", autoCompoundPlan)
	var resp *DualInvestmentAutoCompoundStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/dci/product/auto_compound/edit-status", params, sapiDefaultRate, &resp)
}

// CheckDualInvestmentAccounts returns the total value held in Dual Investment
func (e *Exchange) CheckDualInvestmentAccounts(ctx context.Context) (*DualInvestmentAccountsResponse, error) {
	var resp *DualInvestmentAccountsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/dci/product/accounts", nil, sapiDefaultRate, &resp)
}

// GetDualInvestmentPositions returns Dual Investment positions, filtered by purchase status when one is given
func (e *Exchange) GetDualInvestmentPositions(ctx context.Context, status string, pageSize, pageIndex uint64) (*DualInvestmentPositionsResponse, error) {
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(pageSize, 10))
	}
	if pageIndex > 0 {
		params.Set("pageIndex", strconv.FormatUint(pageIndex, 10))
	}
	var resp *DualInvestmentPositionsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/dci/product/positions", params, sapiDefaultRate, &resp)
}

// SubscribeDualInvestmentProducts subscribes to a Dual Investment product listed by GetDualInvestmentProductList
func (e *Exchange) SubscribeDualInvestmentProducts(ctx context.Context, req *SubscribeDualInvestmentProductsRequest) (*DualInvestmentSubscriptionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ID == "" {
		return nil, errProductIDRequired
	}
	if req.OrderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if req.DepositAmount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.AutoCompoundPlan == "" {
		return nil, errAutoCompoundPlanRequired
	}
	params := url.Values{}
	params.Set("id", req.ID)
	params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	params.Set("depositAmount", strconv.FormatFloat(req.DepositAmount, 'f', -1, 64))
	params.Set("autoCompoundPlan", req.AutoCompoundPlan)
	var resp *DualInvestmentSubscriptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/dci/product/subscribe", params, sapiDefaultRate, &resp)
}
