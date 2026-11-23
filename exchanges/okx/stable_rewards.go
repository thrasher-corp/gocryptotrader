package okx

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// maxStableRewardsAPYHistoryDays is the longest Stable Rewards APY history one request returns
const maxStableRewardsAPYHistoryDays = 100

// GetStableRewardsProductInfo returns the settlement currencies a stablecoin subscribes and redeems through, with their
// fee rates, amount limits and remaining daily quotas
func (e *Exchange) GetStableRewardsProductInfo(ctx context.Context, ccy currency.Code) (*StableRewardsProductInfoResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("ccy", ccy.String())
	var resp *StableRewardsProductInfoResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getStableRewardsProductInfoEPL, http.MethodGet, common.EncodeURLValues("finance/stable-rewards/product-info", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetStableRewardsBalance returns the account's Stable Rewards balances, every supported stablecoin when ccy is empty
func (e *Exchange) GetStableRewardsBalance(ctx context.Context, ccy currency.Code) (*StableRewardsBalanceResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp *StableRewardsBalanceResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getStableRewardsBalanceEPL, http.MethodGet, common.EncodeURLValues("finance/stable-rewards/balance", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetStableRewardsAPYHistory returns a stablecoin's daily APY at the account's VIP level
func (e *Exchange) GetStableRewardsAPYHistory(ctx context.Context, arg *StableRewardsAPYHistoryRequest) ([]APYHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Days > maxStableRewardsAPYHistoryDays {
		return nil, fmt.Errorf("%w: %d days exceeds %d", errAPYHistoryDaysOutOfRange, arg.Days, maxStableRewardsAPYHistoryDays)
	}
	params := url.Values{}
	params.Set("ccy", arg.Currency.String())
	if arg.Days > 0 {
		params.Set("days", strconv.FormatUint(arg.Days, 10))
	}
	var resp []APYHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getStableRewardsAPYHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/stable-rewards/apy-history", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}
