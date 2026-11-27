package okx

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// GetSOLStakingProductInfo returns the SOL staking product's limits and yield
func (e *Exchange) GetSOLStakingProductInfo(ctx context.Context) (*SOLStakingProductInfoResponse, error) {
	var resp *SOLStakingProductInfoResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getSOLStakingProductInfoEPL, http.MethodGet, "finance/staking-defi/sol/product-info", nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// PurchaseSOLStaking stakes SOL from the funding account for OKSOL
func (e *Exchange) PurchaseSOLStaking(ctx context.Context, amount float64) error {
	if amount <= 0 {
		return limits.ErrAmountBelowMin
	}
	var resp []json.RawMessage
	return e.SendHTTPRequest(ctx, exchange.RestSpot, purchaseSOLStakingEPL, http.MethodPost, "finance/staking-defi/sol/purchase", &StakingAmountRequest{Amount: amount}, &resp, request.AuthenticatedRequest)
}

// RedeemSOLStaking redeems OKSOL held in the funding account for SOL
func (e *Exchange) RedeemSOLStaking(ctx context.Context, amount float64) error {
	if amount <= 0 {
		return limits.ErrAmountBelowMin
	}
	var resp []json.RawMessage
	return e.SendHTTPRequest(ctx, exchange.RestSpot, redeemSOLStakingEPL, http.MethodPost, "finance/staking-defi/sol/redeem", &StakingAmountRequest{Amount: amount}, &resp, request.AuthenticatedRequest)
}

// GetSOLStakingBalance returns the account's OKSOL holdings across the trading and funding accounts, redemptions in
// progress included
func (e *Exchange) GetSOLStakingBalance(ctx context.Context) (*SOLStakingBalanceResponse, error) {
	var resp *SOLStakingBalanceResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getSOLStakingBalanceEPL, http.MethodGet, "finance/staking-defi/sol/balance", nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetSOLStakingPurchaseRedeemHistory returns the account's SOL staking purchases and redemptions
func (e *Exchange) GetSOLStakingPurchaseRedeemHistory(ctx context.Context, arg *StakingPurchaseRedeemHistoryRequest) ([]SOLStakingPurchaseRedeemHistoryResponse, error) {
	params, err := arg.values()
	if err != nil {
		return nil, err
	}
	var resp []SOLStakingPurchaseRedeemHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getSOLStakingPurchaseRedeemHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/staking-defi/sol/purchase-redeem-history", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetSOLStakingAPYHistory returns the daily OKSOL APY over the past days, at most 365
func (e *Exchange) GetSOLStakingAPYHistory(ctx context.Context, days uint64) ([]APYHistoryResponse, error) {
	if days == 0 || days > maxStakingAPYHistoryDays {
		return nil, fmt.Errorf("%w: %d days must be between 1 and %d", errAPYHistoryDaysOutOfRange, days, maxStakingAPYHistoryDays)
	}
	params := url.Values{}
	params.Set("days", strconv.FormatUint(days, 10))
	var resp []APYHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getSOLStakingAPYHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/staking-defi/sol/apy-history", params), nil, &resp, request.UnauthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}
