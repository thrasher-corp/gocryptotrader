package gateio

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetStakingCoins retrieves a list of on-chain staking coin products.
func (e *Exchange) GetStakingCoins(ctx context.Context, coinType string) ([]*StakingCoin, error) {
	params := url.Values{}
	if coinType != "" {
		params.Set("cointype", coinType)
	}
	var resp []*StakingCoin
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "earn/staking/coins", params, nil, &resp)
}

// SwapStakingCoins performs an on-chain token swap for earned coins (stake or redeem).
// side: 0=Stake, 1=Redeem
func (e *Exchange) SwapStakingCoins(ctx context.Context, arg *StakingSwapRequest) (*StakingSwapResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Coin == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *StakingSwapResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodPost, "earn/staking/swap", nil, arg, &resp)
}

// GetStakingOrders retrieves a list of on-chain coin-earning orders.
func (e *Exchange) GetStakingOrders(ctx context.Context, ccy currency.Code, pid, orderType, page int64) (*StakingOrdersResponse, error) {
	params := url.Values{}
	if pid > 0 {
		params.Set("pid", strconv.FormatInt(pid, 10))
	}
	if !ccy.IsEmpty() {
		params.Set("coin", ccy.String())
	}
	if orderType >= 0 {
		params.Set("type", strconv.FormatInt(orderType, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	var resp *StakingOrdersResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "earn/staking/order_list", params, nil, &resp)
}

// GetStakingDividendRecords retrieves on-chain coin-earning dividend records.
func (e *Exchange) GetStakingDividendRecords(ctx context.Context, ccy currency.Code, page, pageID int64) (*StakingDividendRecordsResponse, error) {
	params := url.Values{}
	if pageID > 0 {
		params.Set("pid", strconv.FormatInt(pageID, 10))
	}
	if !ccy.IsEmpty() {
		params.Set("coin", ccy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	var resp *StakingDividendRecordsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "earn/staking/award_list", params, nil, &resp)
}

// GetStakingAssets retrieves on-chain coin-earning assets.
func (e *Exchange) GetStakingAssets(ctx context.Context, ccy currency.Code) ([]*StakingAssetItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("coin", ccy.String())
	}
	var resp []*StakingAssetItem
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "earn/staking/assets", params, nil, &resp)
}
