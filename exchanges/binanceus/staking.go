package binanceus

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// GetStakingAssetInformation returns the staking terms of every asset that can be staked, or of stakingAsset when it is
// set
func (e *Exchange) GetStakingAssetInformation(ctx context.Context, stakingAsset currency.Code) ([]StakingAssetInformationResponse, error) {
	params := url.Values{}
	if !stakingAsset.IsEmpty() {
		params.Set("stakingAsset", stakingAsset.Upper().String())
	}
	var resp []StakingAssetInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/staking/asset", params, defaultRate, &resp)
}

// StakeAsset stakes an amount of an asset
func (e *Exchange) StakeAsset(ctx context.Context, req *StakeAssetRequest) (*StakeAssetResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.StakingAsset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	params := url.Values{}
	params.Set("stakingAsset", req.StakingAsset.Upper().String())
	params.Set("amount", formatFloat(req.Amount))
	if req.AutoRestake != nil {
		params.Set("autoRestake", strconv.FormatBool(*req.AutoRestake))
	}
	var resp *StakingEnvelope[StakeAssetResponse]
	if err := e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/staking/stake", params, defaultRate, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// UnstakeAsset unstakes an amount of a staked asset
func (e *Exchange) UnstakeAsset(ctx context.Context, stakingAsset currency.Code, amount float64) (*UnstakeAssetResponse, error) {
	if stakingAsset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	params := url.Values{}
	params.Set("stakingAsset", stakingAsset.Upper().String())
	params.Set("amount", formatFloat(amount))
	var resp *StakingEnvelope[UnstakeAssetResponse]
	if err := e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/staking/unstake", params, defaultRate, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetStakingBalance returns the staking balance of every staked asset, or of asset when it is set. The envelope is
// returned whole, since it adds the statuses of the balances to their data
func (e *Exchange) GetStakingBalance(ctx context.Context, asset currency.Code) (*StakingBalanceResponse, error) {
	params := url.Values{}
	if !asset.IsEmpty() {
		params.Set("asset", asset.Upper().String())
	}
	var resp *StakingBalanceResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/staking/stakingBalance", params, defaultRate, &resp)
}

// GetStakingHistory returns the staking transactions of every asset, or of the request's asset, a page at a time. A nil
// request returns the first page of every asset's transactions
func (e *Exchange) GetStakingHistory(ctx context.Context, req *StakingHistoryRequest) ([]StakingRecordResponse, error) {
	params, err := stakingHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp []StakingRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/staking/history", params, defaultRate, &resp)
}

// GetStakingRewardsHistory returns the staking rewards of every asset, or of the request's asset, a page at a time. A
// nil request returns the first page of every asset's rewards. The envelope is returned whole, since it adds the number
// of rewards the query matched to their data
func (e *Exchange) GetStakingRewardsHistory(ctx context.Context, req *StakingHistoryRequest) (*StakingRewardsHistoryResponse, error) {
	params, err := stakingHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *StakingRewardsHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/staking/stakingRewardsHistory", params, defaultRate, &resp)
}

// stakingHistoryParams builds the parameters the staking history and staking rewards history queries share
func stakingHistoryParams(req *StakingHistoryRequest) (url.Values, error) {
	params := url.Values{}
	if req == nil {
		return params, nil
	}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Page > 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	return params, nil
}
