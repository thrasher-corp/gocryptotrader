package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// maxStakingAPYHistoryDays is the longest APY history the ETH and SOL staking endpoints return
const maxStakingAPYHistoryDays = 365

var errAPYHistoryDaysOutOfRange = errors.New("APY history days out of range")

// GetETHStakingProductInfo returns the ETH staking product's limits and yield
func (e *Exchange) GetETHStakingProductInfo(ctx context.Context) (*ETHStakingProductInfoResponse, error) {
	var resp *ETHStakingProductInfoResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getProductInfoEPL, http.MethodGet, "finance/staking-defi/eth/product-info", nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// PurchaseETHStaking stakes ETH from the funding account for BETH
func (e *Exchange) PurchaseETHStaking(ctx context.Context, amount float64) error {
	if amount <= 0 {
		return limits.ErrAmountBelowMin
	}
	var resp []json.RawMessage
	return e.SendHTTPRequest(ctx, exchange.RestSpot, purchaseETHStakingEPL, http.MethodPost, "finance/staking-defi/eth/purchase", &StakingAmountRequest{Amount: amount}, &resp, request.AuthenticatedRequest)
}

// RedeemETHStaking redeems BETH held in the funding account for ETH
func (e *Exchange) RedeemETHStaking(ctx context.Context, amount float64) error {
	if amount <= 0 {
		return limits.ErrAmountBelowMin
	}
	var resp []json.RawMessage
	return e.SendHTTPRequest(ctx, exchange.RestSpot, redeemETHStakingEPL, http.MethodPost, "finance/staking-defi/eth/redeem", &StakingAmountRequest{Amount: amount}, &resp, request.AuthenticatedRequest)
}

// CancelETHStakingRedemption cancels a pending ETH staking redemption
func (e *Exchange) CancelETHStakingRedemption(ctx context.Context, orderID string) (*ETHStakingOrderIDResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *ETHStakingOrderIDResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, cancelETHStakingRedemptionEPL, http.MethodPost, "finance/staking-defi/eth/cancel-redeem", &ETHStakingCancelRedemptionRequest{OrderID: orderID}, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetETHStakingBalance returns the account's BETH holdings across the trading and funding accounts, redemptions in
// progress included
func (e *Exchange) GetETHStakingBalance(ctx context.Context) (*ETHStakingBalanceResponse, error) {
	var resp *ETHStakingBalanceResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getBETHBalanceEPL, http.MethodGet, "finance/staking-defi/eth/balance", nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetETHStakingPurchaseRedeemHistory returns the account's ETH staking purchases and redemptions
func (e *Exchange) GetETHStakingPurchaseRedeemHistory(ctx context.Context, arg *StakingPurchaseRedeemHistoryRequest) ([]ETHStakingPurchaseRedeemHistoryResponse, error) {
	params, err := arg.values()
	if err != nil {
		return nil, err
	}
	var resp []ETHStakingPurchaseRedeemHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getPurchaseRedeemHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/staking-defi/eth/purchase-redeem-history", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetETHStakingAPYHistory returns the daily BETH APY over the past days, at most 365
func (e *Exchange) GetETHStakingAPYHistory(ctx context.Context, days uint64) ([]APYHistoryResponse, error) {
	if days == 0 || days > maxStakingAPYHistoryDays {
		return nil, fmt.Errorf("%w: %d days must be between 1 and %d", errAPYHistoryDaysOutOfRange, days, maxStakingAPYHistoryDays)
	}
	params := url.Values{}
	params.Set("days", strconv.FormatUint(days, 10))
	var resp []APYHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getAPYHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/staking-defi/eth/apy-history", params), nil, &resp, request.UnauthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// values validates the purchase and redemption history filters and encodes them as query parameters
func (r *StakingPurchaseRedeemHistoryRequest) values() (url.Values, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	if !r.After.IsZero() && !r.Before.IsZero() && r.Before.After(r.After) {
		return nil, fmt.Errorf("%w: Before %s must not be later than After %s", common.ErrStartAfterEnd, r.Before, r.After)
	}
	params := url.Values{}
	if r.Type != "" {
		params.Set("type", r.Type)
	}
	if r.Status != "" {
		params.Set("status", r.Status)
	}
	if !r.After.IsZero() {
		params.Set("after", strconv.FormatInt(r.After.UnixMilli(), 10))
	}
	if !r.Before.IsZero() {
		params.Set("before", strconv.FormatInt(r.Before.UnixMilli(), 10))
	}
	if r.Limit > 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	return params, nil
}
