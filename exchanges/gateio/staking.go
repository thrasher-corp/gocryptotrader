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

// GetStakingCoins retrieves on-chain staking products; coinType is swap for vouchers, lock for locked positions or
// debt for US Treasury bonds, and all products are returned when empty
func (e *Exchange) GetStakingCoins(ctx context.Context, coinType string) ([]*StakingCoinResponse, error) {
	params := url.Values{}
	if coinType != "" {
		params.Set("cointype", coinType)
	}
	var resp []*StakingCoinResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnStakingCoinsEPL, http.MethodGet, "earn/staking/coins", params, nil, &resp)
}

// SwapStakingCoins stakes or redeems an on-chain staking coin
func (e *Exchange) SwapStakingCoins(ctx context.Context, arg *StakingSwapRequest) (*StakingSwapResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Side > 1 {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	var resp *StakingSwapResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnStakingSwapEPL, http.MethodPost, "earn/staking/swap", nil, arg, &resp)
}

// GetStakingOrders retrieves on-chain staking orders; orderType is "0" for staking or "1" for redemption, and both are
// returned when empty
func (e *Exchange) GetStakingOrders(ctx context.Context, ccy currency.Code, productID uint64, orderType string, page uint64) (*StakingOrdersResponse, error) {
	params := url.Values{}
	if productID > 0 {
		params.Set("pid", strconv.FormatUint(productID, 10))
	}
	if !ccy.IsEmpty() {
		params.Set("coin", ccy.String())
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	var resp *StakingOrdersResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnStakingOrderListEPL, http.MethodGet, "earn/staking/order_list", params, nil, &resp)
}

// GetStakingDividendRecords retrieves on-chain staking dividend records
func (e *Exchange) GetStakingDividendRecords(ctx context.Context, ccy currency.Code, productID, page uint64) (*StakingDividendRecordsResponse, error) {
	params := url.Values{}
	if productID > 0 {
		params.Set("pid", strconv.FormatUint(productID, 10))
	}
	if !ccy.IsEmpty() {
		params.Set("coin", ccy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	var resp *StakingDividendRecordsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnStakingAwardListEPL, http.MethodGet, "earn/staking/award_list", params, nil, &resp)
}

// GetStakingAssets retrieves on-chain staking positions
func (e *Exchange) GetStakingAssets(ctx context.Context, ccy currency.Code) ([]*StakingAssetItemResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("coin", ccy.String())
	}
	var resp []*StakingAssetItemResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnStakingAssetsEPL, http.MethodGet, "earn/staking/assets", params, nil, &resp)
}
