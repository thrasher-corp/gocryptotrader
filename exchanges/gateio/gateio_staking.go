package gateio

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// SwapETH2 swaps ETH2
// 1-Forward Swap (ETH -> ETH2), 2-Reverse Swap (ETH2 -> ETH
func (e *Exchange) SwapETH2(ctx context.Context, arg *SwapETHParam) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.Side == "" {
		return order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return order.ErrAmountIsInvalid
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodPost, "earn/staking/eth2/swap", nil, arg, nil)
}

// GetETH2HistoricalReturnRate gets ETH2 historical return rate
// Query ETH earnings rate records for the last 31 days
func (e *Exchange) GetETH2HistoricalReturnRate(ctx context.Context) ([]*ETH2ReturnRate, error) {
	var resp []*ETH2ReturnRate
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "earn/staking/eth2/rate_records", nil, nil, &resp)
}

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
func (e *Exchange) GetStakingOrders(ctx context.Context, pid, orderType int64, ccy currency.Code, page int32) (*StakingOrdersResponse, error) {
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
		params.Set("page", strconv.FormatInt(int64(page), 10))
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
		params.Set("page", strconv.FormatInt(int64(page), 10))
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
