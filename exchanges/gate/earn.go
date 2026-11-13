package gate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// GetDualInvestmentProductList retrieves the dual investment product list
func (e *Exchange) GetDualInvestmentProductList(ctx context.Context, arg *DualInvestmentProductListRequest) ([]*DualInvestmentPlanResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if arg.PlanID != 0 {
		params.Set("plan_id", strconv.FormatUint(arg.PlanID, 10))
	}
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.String())
	}
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	if !arg.QuoteCurrency.IsEmpty() {
		params.Set("quote_currency", arg.QuoteCurrency.String())
	}
	if arg.Sort != "" {
		params.Set("sort", arg.Sort)
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.PageSize > 0 {
		params.Set("page_size", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp []*DualInvestmentPlanResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, earnDualInvestmentPlanEPL, common.EncodeURLValues("earn/dual/investment_plan", params), &resp)
}

// GetDualInvestmentOrderList retrieves dual investment orders
func (e *Exchange) GetDualInvestmentOrderList(ctx context.Context, arg *DualInvestmentOrderListRequest) ([]*DualInvestmentOrderDetailResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	// The window filters settlement times, which open positions have in the future, so only its order is checked
	if !arg.From.IsZero() && !arg.To.IsZero() && arg.To.Before(arg.From) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if !arg.From.IsZero() {
		params.Set("from", strconv.FormatInt(arg.From.Unix(), 10))
	}
	if !arg.To.IsZero() {
		params.Set("to", strconv.FormatInt(arg.To.Unix(), 10))
	}
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	if arg.Status != "" {
		params.Set("status", arg.Status)
	}
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.String())
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []*DualInvestmentOrderDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnDualOrdersEPL, http.MethodGet, "earn/dual/orders", params, nil, &resp)
}

// PlaceDualInvestmentOrder places a dual investment order
func (e *Exchange) PlaceDualInvestmentOrder(ctx context.Context, arg *DualInvestmentOrderRequest) (*DualInvestmentOrderDetailResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.PlanID == "" {
		return nil, errPlanIDRequired
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.Text != "" && !strings.HasPrefix(arg.Text, "t-") {
		return nil, errInvalidTextPrefix
	}
	var resp *DualInvestmentOrderDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnDualPlaceOrderEPL, http.MethodPost, "earn/dual/orders", nil, arg, &resp)
}

// GetDualCurrencyEarningAssets retrieves the user's dual investment assets and interest
func (e *Exchange) GetDualCurrencyEarningAssets(ctx context.Context) (*DualCurrencyEarningAssetResponse, error) {
	var resp *DualCurrencyEarningAssetResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnDualBalanceEPL, http.MethodGet, "earn/dual/balance", nil, nil, &resp)
}

// GetDualCurrencyEarlyRedemptionPreview previews a dual investment order's early redemption; the returned request
// ID is required by RedeemDualCurrencyOrder
func (e *Exchange) GetDualCurrencyEarlyRedemptionPreview(ctx context.Context, orderID string) (*DualCurrencyEarlyRedemptionResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("order_id", orderID)
	var resp *DualCurrencyEarlyRedemptionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnDualOrderRefundPreviewEPL, http.MethodGet, "earn/dual/order-refund-preview", params, nil, &resp)
}

// RedeemDualCurrencyOrder redeems a dual investment order early using the request ID from its redemption preview
func (e *Exchange) RedeemDualCurrencyOrder(ctx context.Context, orderID, requestID string) error {
	if orderID == "" {
		return order.ErrOrderIDNotSet
	}
	if requestID == "" {
		return errOrderRefundRequestIDRequired
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnDualOrderRefundEPL, http.MethodPost, "earn/dual/order-refund", nil, map[string]string{
		"order_id": orderID,
		"req_id":   requestID,
	}, nil)
}

// ModifyDualCurrencyOrderReinvest switches a dual investment order's reinvestment on or off
func (e *Exchange) ModifyDualCurrencyOrderReinvest(ctx context.Context, arg *DualModifyOrderReinvestRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.OrderID == 0 {
		return order.ErrOrderIDNotSet
	}
	if arg.Status > 1 {
		return fmt.Errorf("%w: 0 switches reinvestment off and 1 switches it on", errInvalidReinvestStatus)
	}
	if err := checkWholeSeconds(arg.EffectiveDuration, "effective duration"); err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnDualModifyOrderReinvestEPL, http.MethodPost, "earn/dual/modify-order-reinvest", nil, arg, nil)
}

// GetDualCurrencyRecommendedProjects retrieves recommended dual investment products
// mode is normal, senior, apy_up, ep_down, ep_up, dt_down or dt_up; orderType is call or put; excludeProductIDs
// lists previously recommended products to leave out
func (e *Exchange) GetDualCurrencyRecommendedProjects(ctx context.Context, mode string, coin currency.Code, orderType string, excludeProductIDs []uint64) ([]*DualRecommendedProjectResponse, error) {
	params := url.Values{}
	if mode != "" {
		params.Set("mode", mode)
	}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if len(excludeProductIDs) > 0 {
		productIDs := make([]string, len(excludeProductIDs))
		for i, productID := range excludeProductIDs {
			productIDs[i] = strconv.FormatUint(productID, 10)
		}
		params.Set("history_pids", strings.Join(productIDs, ","))
	}
	var resp []*DualRecommendedProjectResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnDualProjectRecommendEPL, http.MethodGet, "earn/dual/project-recommend", params, nil, &resp)
}

// CreateAutoInvestPlan creates an auto invest plan
func (e *Exchange) CreateAutoInvestPlan(ctx context.Context, arg *CreateAutoInvestPlanRequest) (*AutoInvestPlanResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.PlanMoney.IsEmpty() {
		return nil, fmt.Errorf("%w: plan pricing currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.PlanAmount <= 0 {
		return nil, fmt.Errorf("%w: plan amount is required", limits.ErrAmountBelowMin)
	}
	// The period day only applies to monthly and weekly cycles; daily and hourly cycles ignore it
	switch arg.PlanPeriodType {
	case "":
		return nil, errPeriodTypeRequired
	case "monthly":
		if arg.PlanPeriodDay < 1 || arg.PlanPeriodDay > 30 {
			return nil, fmt.Errorf("%w: monthly plans run on a day from 1 to 30", errInvalidPlanPeriodDay)
		}
	case "weekly", "biweekly":
		if arg.PlanPeriodDay < 1 || arg.PlanPeriodDay > 7 {
			return nil, fmt.Errorf("%w: weekly plans run on a weekday from 1 to 7", errInvalidPlanPeriodDay)
		}
	}
	if arg.PlanPeriodHour > 23 {
		return nil, fmt.Errorf("%w: must be an hour from 0 to 23", errInvalidPlanPeriodHour)
	}
	if err := validateAutoInvestPlanItems(arg.Items); err != nil {
		return nil, err
	}
	var resp *AutoInvestPlanResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestCreatePlanEPL, http.MethodPost, "earn/autoinvest/plans/create", nil, arg, &resp)
}

// validateAutoInvestPlanItems checks each portfolio item names a currency and its share of the portfolio
func validateAutoInvestPlanItems(items []*AutoInvestPlanItem) error {
	if len(items) == 0 {
		return fmt.Errorf("%w: at least one portfolio item is required", errNoValidParameterPassed)
	}
	for _, item := range items {
		if item == nil {
			return fmt.Errorf("%w: portfolio item", common.ErrNilPointer)
		}
		if item.Asset.IsEmpty() {
			return currency.ErrCurrencyCodeEmpty
		}
		if item.Ratio <= 0 {
			return fmt.Errorf("%w: proportion of this currency in the portfolio is required", order.ErrAmountMustBeSet)
		}
	}
	return nil
}

// UpdateAutoInvestPlan updates an auto invest plan's fund source and fund flow
func (e *Exchange) UpdateAutoInvestPlan(ctx context.Context, arg *AutoInvestPlanUpdateRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.PlanID == 0 {
		return errPlanIDRequired
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestUpdatePlanEPL, http.MethodPost, "earn/autoinvest/plans/update", nil, arg, nil)
}

// StopAutoInvestPlan stops an auto invest plan
func (e *Exchange) StopAutoInvestPlan(ctx context.Context, planID uint64) error {
	if planID == 0 {
		return errPlanIDRequired
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestStopPlanEPL, http.MethodPost, "earn/autoinvest/plans/stop", nil, &AutoInvestPlanStopRequest{PlanID: planID}, nil)
}

// AddAutoInvestPlanPosition invests an amount into an auto invest plan immediately
func (e *Exchange) AddAutoInvestPlanPosition(ctx context.Context, planID uint64, amount float64) error {
	if planID == 0 {
		return errPlanIDRequired
	}
	if amount <= 0 {
		return order.ErrAmountIsInvalid
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestAddPositionEPL, http.MethodPost, "earn/autoinvest/plans/add_position", nil, &AutoInvestPlanAddPositionRequest{
		PlanID: planID,
		Amount: types.Number(amount),
	}, nil)
}

// GetAutoInvestSupportedCoins retrieves the currencies supported by auto invest for a pricing currency of USDT or BTC
// The pricing currency defaults to USDT when empty
func (e *Exchange) GetAutoInvestSupportedCoins(ctx context.Context, planMoney currency.Code) ([]*AutoInvestCoinItemResponse, error) {
	params := url.Values{}
	if !planMoney.IsEmpty() {
		params.Set("plan_money", planMoney.String())
	}
	var resp []*AutoInvestCoinItemResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestCoinsEPL, http.MethodGet, "earn/autoinvest/coins", params, nil, &resp)
}

// GetAutoInvestMinimumAmount retrieves the minimum investment amount for a portfolio
func (e *Exchange) GetAutoInvestMinimumAmount(ctx context.Context, arg *AutoInvestMinAmountRequest) (*AutoInvestMinAmountResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Money.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateAutoInvestPlanItems(arg.Items); err != nil {
		return nil, err
	}
	var resp *AutoInvestMinAmountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestMinAmountEPL, http.MethodPost, "earn/autoinvest/min_invest_amount", nil, arg, &resp)
}

// GetAutoInvestPlanExecutionRecords retrieves an auto invest plan's execution records; pageSize is at most 100
func (e *Exchange) GetAutoInvestPlanExecutionRecords(ctx context.Context, planID, page, pageSize uint64) (*AutoInvestPlanExecutionRecordsResponse, error) {
	if planID == 0 {
		return nil, errPlanIDRequired
	}
	params := url.Values{}
	params.Set("plan_id", strconv.FormatUint(planID, 10))
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatUint(pageSize, 10))
	}
	var resp *AutoInvestPlanExecutionRecordsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestPlanRecordsEPL, http.MethodGet, "earn/autoinvest/plans/records", params, nil, &resp)
}

// GetAutoInvestPlanOrderDetails retrieves the orders placed by an auto invest plan execution record
func (e *Exchange) GetAutoInvestPlanOrderDetails(ctx context.Context, planID, recordID uint64) ([]*AutoInvestOrderItemResponse, error) {
	if planID == 0 {
		return nil, errPlanIDRequired
	}
	if recordID == 0 {
		return nil, errRecordIDRequired
	}
	params := url.Values{}
	params.Set("plan_id", strconv.FormatUint(planID, 10))
	params.Set("record_id", strconv.FormatUint(recordID, 10))
	var resp []*AutoInvestOrderItemResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestOrdersEPL, http.MethodGet, "earn/autoinvest/orders", params, nil, &resp)
}

// GetAutoInvestCurrencyConfig retrieves the maximum investment amount of each auto invest pricing currency
func (e *Exchange) GetAutoInvestCurrencyConfig(ctx context.Context) ([]*AutoInvestConfigItemResponse, error) {
	var resp []*AutoInvestConfigItemResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestConfigEPL, http.MethodGet, "earn/autoinvest/config", nil, nil, &resp)
}

// GetAutoInvestPlanDetails retrieves an auto invest plan
func (e *Exchange) GetAutoInvestPlanDetails(ctx context.Context, planID uint64) (*AutoInvestPlanDetailsResponse, error) {
	if planID == 0 {
		return nil, errPlanIDRequired
	}
	params := url.Values{}
	params.Set("plan_id", strconv.FormatUint(planID, 10))
	var resp *AutoInvestPlanDetailsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestPlanDetailEPL, http.MethodGet, "earn/autoinvest/plans/detail", params, nil, &resp)
}

// GetAutoInvestPlanList retrieves auto invest plans; status is active or history and pageSize is at most 100
func (e *Exchange) GetAutoInvestPlanList(ctx context.Context, status string, page, pageSize uint64) (*AutoInvestPlanListResponse, error) {
	if status == "" {
		return nil, errPlanStatusRequired
	}
	params := url.Values{}
	params.Set("status", status)
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatUint(pageSize, 10))
	}
	var resp *AutoInvestPlanListResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestPlanListEPL, http.MethodGet, "earn/autoinvest/plans/list_info", params, nil, &resp)
}

// GetFixedTermProducts retrieves fixed-term earn products; productType is 1 for regular or 2 for VIP products and is
// not filtered on when 0
func (e *Exchange) GetFixedTermProducts(ctx context.Context, ccy currency.Code, productType uint8, page, limit uint64) (*FixedTermProductsResponse, error) {
	if page == 0 {
		return nil, errPageRequired
	}
	if limit == 0 {
		return nil, fmt.Errorf("%w: limit is required", errInvalidLimit)
	}
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.String())
	}
	if productType != 0 {
		params.Set("type", strconv.FormatUint(uint64(productType), 10))
	}
	params.Set("page", strconv.FormatUint(page, 10))
	params.Set("limit", strconv.FormatUint(limit, 10))
	var resp gateAPIResponse[*FixedTermProductsResponse]
	return resp.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, earnFixedTermProductEPL, common.EncodeURLValues("earn/fixed-term/product", params), &resp)
}

// GetFixedTermProductsByAsset retrieves a currency's fixed-term earn products in ascending order of term
// productType is empty or "1" for regular products, "2" for VIP products and "0" for all products
func (e *Exchange) GetFixedTermProductsByAsset(ctx context.Context, ccy currency.Code, productType string) (*FixedTermProductsByAssetResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if productType != "" {
		params.Set("type", productType)
	}
	var resp gateAPIResponse[*FixedTermProductsByAssetResponse]
	return resp.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, earnFixedTermProductListEPL, common.EncodeURLValues("earn/fixed-term/product/"+ccy.String()+"/list", params), &resp)
}

// SetFixedTermSubscriptionOrder subscribes to a fixed-term earn product
func (e *Exchange) SetFixedTermSubscriptionOrder(ctx context.Context, arg *FixedTermSubscriptionRequest) (*OrderIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.ProductID == 0 {
		return nil, errProductIDRequired
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp gateAPIResponse[*OrderIDResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnFixedTermLendEPL, http.MethodPost, "earn/fixed-term/user/lend", nil, arg, &resp)
}

// GetFixedTermSubscriptionOrders retrieves fixed-term earn subscription orders
func (e *Exchange) GetFixedTermSubscriptionOrders(ctx context.Context, arg *FixedTermSubscriptionOrdersRequest) (*FixedTermSubscriptionOrdersResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderType != 1 && arg.OrderType != 2 {
		return nil, fmt.Errorf("%w: 1 for current orders or 2 for historical orders", order.ErrTypeIsInvalid)
	}
	if arg.Page == 0 {
		return nil, errPageRequired
	}
	if arg.Limit == 0 {
		return nil, fmt.Errorf("%w: limit is required", errInvalidLimit)
	}
	params := url.Values{}
	if err := setFixedTermBusinessFilter(params, arg.BusinessFilter); err != nil {
		return nil, err
	}
	if arg.ProductID != 0 {
		params.Set("product_id", strconv.FormatUint(arg.ProductID, 10))
	}
	if arg.OrderID != 0 {
		params.Set("order_id", strconv.FormatUint(arg.OrderID, 10))
	}
	if !arg.Asset.IsEmpty() {
		params.Set("asset", arg.Asset.String())
	}
	params.Set("order_type", strconv.FormatUint(uint64(arg.OrderType), 10))
	params.Set("page", strconv.FormatUint(arg.Page, 10))
	params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	if arg.SubBusiness != 0 {
		params.Set("sub_business", strconv.FormatUint(arg.SubBusiness, 10))
	}
	var resp gateAPIResponse[*FixedTermSubscriptionOrdersResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnFixedTermLendsEPL, http.MethodGet, "earn/fixed-term/user/lend", params, nil, &resp)
}

// RedeemFixedTermOrder redeems a fixed-term earn order before it matures
func (e *Exchange) RedeemFixedTermOrder(ctx context.Context, orderID string) error {
	if orderID == "" {
		return order.ErrOrderIDNotSet
	}
	var resp gateAPIResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnFixedTermPreRedeemEPL, http.MethodPost, "earn/fixed-term/user/pre-redeem", nil, map[string]string{"order_id": orderID}, &resp)
}

// GetFixedTermSubscriptionHistory retrieves fixed-term earn subscription, redemption, interest or bonus history
func (e *Exchange) GetFixedTermSubscriptionHistory(ctx context.Context, arg *FixedTermHistoryRequest) (*FixedTermHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Type == 0 {
		return nil, errHistoryTypeRequired
	}
	if arg.Type > 4 {
		return nil, fmt.Errorf("%w: 1 subscription, 2 redemption, 3 interest or 4 bonus reward", errHistoryTypeRequired)
	}
	if arg.Page == 0 {
		return nil, errPageRequired
	}
	if arg.Limit == 0 {
		return nil, fmt.Errorf("%w: limit is required", errInvalidLimit)
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if err := setFixedTermBusinessFilter(params, arg.BusinessFilter); err != nil {
		return nil, err
	}
	if arg.ProductID != 0 {
		params.Set("product_id", strconv.FormatUint(arg.ProductID, 10))
	}
	if arg.OrderID != "" {
		params.Set("order_id", arg.OrderID)
	}
	if !arg.Asset.IsEmpty() {
		params.Set("asset", arg.Asset.String())
	}
	params.Set("type", strconv.FormatUint(uint64(arg.Type), 10))
	params.Set("page", strconv.FormatUint(arg.Page, 10))
	params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	if !arg.StartTime.IsZero() {
		params.Set("start_at", strconv.FormatInt(arg.StartTime.Unix(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("end_at", strconv.FormatInt(arg.EndTime.Unix(), 10))
	}
	if arg.SubBusiness != 0 {
		params.Set("sub_business", strconv.FormatUint(arg.SubBusiness, 10))
	}
	var resp gateAPIResponse[*FixedTermHistoryResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnFixedTermHistoryEPL, http.MethodGet, "earn/fixed-term/user/history", params, nil, &resp)
}

// setFixedTermBusinessFilter encodes fixed-term business filters as the JSON array the business_filter parameter expects
func setFixedTermBusinessFilter(params url.Values, filters []FixedTermBusinessFilterRequest) error {
	if len(filters) == 0 {
		return nil
	}
	filter, err := json.Marshal(filters)
	if err != nil {
		return fmt.Errorf("error encoding fixed-term business filter: %w", err)
	}
	params.Set("business_filter", string(filter))
	return nil
}
