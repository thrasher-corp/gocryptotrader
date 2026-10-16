package binance

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetETHStakingAccount returns the ETH staking holdings and 30-day profit of the account
func (e *Exchange) GetETHStakingAccount(ctx context.Context) (*ETHStakingAccountResponse, error) {
	var resp *ETHStakingAccountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v2/eth-staking/account", nil, ethStakingAccountRate, &resp)
}

// GetBETHRewardsDistributionHistory returns BETH rewards distributions. Binance dropped the endpoint from its API
// reference without a retirement notice and it still answers, so it follows the last published reference (2025-11)
func (e *Exchange) GetBETHRewardsDistributionHistory(ctx context.Context, req *EarnHistoryRequest) (*BETHRewardsDistributionHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	var resp *BETHRewardsDistributionHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/eth-staking/eth/history/rewardsHistory", params, bethRewardDistributionHistoryRate, &resp)
}

// GetCurrentETHStakingQuota returns the ETH staking and redemption quotas, limits and redemption period
func (e *Exchange) GetCurrentETHStakingQuota(ctx context.Context) (*ETHStakingQuotaResponse, error) {
	var resp *ETHStakingQuotaResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/eth-staking/eth/quota", nil, currentETHStakingQuotaRate, &resp)
}

// GetETHRedemptionHistory returns ETH staking redemptions
func (e *Exchange) GetETHRedemptionHistory(ctx context.Context, req *StakingRedemptionHistoryRequest) (*ETHRedemptionHistoryResponse, error) {
	params, err := stakingRedemptionHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *ETHRedemptionHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/eth-staking/eth/history/redemptionHistory", params, ethRedemptionHistoryRate, &resp)
}

// GetETHStakingHistory returns ETH staking subscriptions
func (e *Exchange) GetETHStakingHistory(ctx context.Context, req *StakingSubscriptionHistoryRequest) (*ETHStakingHistoryResponse, error) {
	params, err := stakingSubscriptionHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *ETHStakingHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/eth-staking/eth/history/stakingHistory", params, ethStakingHistoryRate, &resp)
}

// GetWBETHRateHistory returns the WBETH annual percentage rate and exchange rate history
func (e *Exchange) GetWBETHRateHistory(ctx context.Context, req *EarnHistoryRequest) (*WBETHRateHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	var resp *WBETHRateHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/eth-staking/eth/history/rateHistory", params, getWBETHRateHistoryRate, &resp)
}

// GetWBETHRewardHistory returns the rewards accrued within held WBETH
func (e *Exchange) GetWBETHRewardHistory(ctx context.Context, req *EarnHistoryRequest) (*WBETHRewardHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	var resp *WBETHRewardHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/eth-staking/eth/history/wbethRewardsHistory", params, wbethRewardsHistoryRate, &resp)
}

// GetWBETHUnwrapHistory returns WBETH to BETH unwraps
func (e *Exchange) GetWBETHUnwrapHistory(ctx context.Context, req *EarnHistoryRequest) (*WBETHWrapHistoryResponse, error) {
	return e.getWBETHWrapOrUnwrapHistory(ctx, req, "/sapi/v1/eth-staking/wbeth/history/unwrapHistory")
}

// GetWBETHWrapHistory returns BETH to WBETH wraps
func (e *Exchange) GetWBETHWrapHistory(ctx context.Context, req *EarnHistoryRequest) (*WBETHWrapHistoryResponse, error) {
	return e.getWBETHWrapOrUnwrapHistory(ctx, req, "/sapi/v1/eth-staking/wbeth/history/wrapHistory")
}

func (e *Exchange) getWBETHWrapOrUnwrapHistory(ctx context.Context, req *EarnHistoryRequest, path string) (*WBETHWrapHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	var resp *WBETHWrapHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, path, params, wbethWrapOrUnwrapHistoryRate, &resp)
}

// RedeemETH redeems WBETH or BETH for ETH; the asset defaults to BETH when empty
func (e *Exchange) RedeemETH(ctx context.Context, amount float64, asset currency.Code) (*ETHRedemptionResponse, error) {
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if !asset.IsEmpty() {
		params.Set("asset", asset.Upper().String())
	}
	var resp *ETHRedemptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/eth-staking/eth/redeem", params, ethereumStakingRedemptionRate, &resp)
}

// SubscribeETHStaking stakes ETH for WBETH
func (e *Exchange) SubscribeETHStaking(ctx context.Context, amount float64) (*ETHStakingSubscriptionResponse, error) {
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *ETHStakingSubscriptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v2/eth-staking/eth/stake", params, subscribeETHStakingRate, &resp)
}

// WrapBETH wraps BETH into WBETH
func (e *Exchange) WrapBETH(ctx context.Context, amount float64) (*WrapBETHResponse, error) {
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *WrapBETHResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/eth-staking/wbeth/wrap", params, wrapBETHRate, &resp)
}

// ClaimBoostRewards claims SOL staking boost APR airdrop rewards
func (e *Exchange) ClaimBoostRewards(ctx context.Context) (*EarnSuccessResponse, error) {
	var resp *EarnSuccessResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/sol-staking/sol/claim", nil, claimBoostRewardsRate, &resp)
}

// GetBNSOLRateHistory returns the BNSOL annual percentage rate, exchange rate and boost reward history
func (e *Exchange) GetBNSOLRateHistory(ctx context.Context, req *EarnHistoryRequest) (*BNSOLRateHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	var resp *BNSOLRateHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sol-staking/sol/history/rateHistory", params, bnsolRateHistoryRate, &resp)
}

// GetBNSOLRewardsHistory returns the rewards accrued within held BNSOL
func (e *Exchange) GetBNSOLRewardsHistory(ctx context.Context, req *EarnHistoryRequest) (*BNSOLRewardsHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	var resp *BNSOLRewardsHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sol-staking/sol/history/bnsolRewardsHistory", params, bnsolRewardsHistoryRate, &resp)
}

// GetBoostRewardsHistory returns SOL staking boost APR airdrop rewards that were claimed or distributed
func (e *Exchange) GetBoostRewardsHistory(ctx context.Context, req *BoostRewardsHistoryRequest) (*BoostRewardsHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Type == "" {
		return nil, errRewardTypeMissing
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	params.Set("type", req.Type)
	var resp *BoostRewardsHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sol-staking/sol/history/boostRewardsHistory", params, boostRewardsHistoryRate, &resp)
}

// GetSOLRedemptionHistory returns SOL staking redemptions
func (e *Exchange) GetSOLRedemptionHistory(ctx context.Context, req *StakingRedemptionHistoryRequest) (*SOLRedemptionHistoryResponse, error) {
	params, err := stakingRedemptionHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *SOLRedemptionHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sol-staking/sol/history/redemptionHistory", params, solRedemptionHistoryRate, &resp)
}

// GetSOLStakingHistory returns SOL staking subscriptions
func (e *Exchange) GetSOLStakingHistory(ctx context.Context, req *StakingSubscriptionHistoryRequest) (*SOLStakingHistoryResponse, error) {
	params, err := stakingSubscriptionHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp *SOLStakingHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sol-staking/sol/history/stakingHistory", params, solStakingHistoryRate, &resp)
}

// GetSOLStakingQuotaDetails returns the SOL staking and redemption quotas, limits and redemption period
func (e *Exchange) GetSOLStakingQuotaDetails(ctx context.Context) (*SOLStakingQuotaResponse, error) {
	var resp *SOLStakingQuotaResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sol-staking/sol/quota", nil, solStakingQuotaDetailsRate, &resp)
}

// GetUnclaimedRewards returns unclaimed SOL staking boost APR airdrop rewards
func (e *Exchange) GetUnclaimedRewards(ctx context.Context) ([]SOLStakingUnclaimedReward, error) {
	var resp []SOLStakingUnclaimedReward
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sol-staking/sol/history/unclaimedRewards", nil, unclaimedRewardsRate, &resp)
}

// RedeemSOL redeems BNSOL for SOL
func (e *Exchange) RedeemSOL(ctx context.Context, amount float64) (*SOLRedemptionResponse, error) {
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *SOLRedemptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/sol-staking/sol/redeem", params, redeemSOLRate, &resp)
}

// GetSOLStakingAccount returns the BNSOL holdings and 30-day profit of the account
func (e *Exchange) GetSOLStakingAccount(ctx context.Context) (*SOLStakingAccountResponse, error) {
	var resp *SOLStakingAccountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/sol-staking/account", nil, solStakingAccountRate, &resp)
}

// SubscribeToSOLStaking stakes SOL for BNSOL
func (e *Exchange) SubscribeToSOLStaking(ctx context.Context, amount float64) (*SOLStakingSubscriptionResponse, error) {
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *SOLStakingSubscriptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/sol-staking/sol/stake", params, subscribeSOLStakingRate, &resp)
}

// stakingRedemptionHistoryParams builds the query shared by the ETH and SOL redemption history endpoints
func stakingRedemptionHistoryParams(req *StakingRedemptionHistoryRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	if req.RedeemID > 0 {
		params.Set("redeemId", strconv.FormatUint(req.RedeemID, 10))
	}
	return params, nil
}

// stakingSubscriptionHistoryParams builds the query shared by the ETH and SOL staking history endpoints
func stakingSubscriptionHistoryParams(req *StakingSubscriptionHistoryRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := historyPageParams(req.StartTime, req.EndTime, req.Current, req.Size)
	if err != nil {
		return nil, err
	}
	if req.PurchaseID > 0 {
		params.Set("purchaseId", strconv.FormatUint(req.PurchaseID, 10))
	}
	return params, nil
}
