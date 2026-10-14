package kucoin

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetFuturesCurrentFundingRate returns a contract's current funding rate (Get Current Funding Rate)
func (e *Exchange) GetFuturesCurrentFundingRate(ctx context.Context, symbol string) (*FuturesFundingRateResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *FuturesFundingRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getFuturesCurrentFundingRateEPL, "/v1/funding-rate/"+symbol+"/current", &resp)
}

// GetPublicFundingRate returns the funding rate a contract settled at each funding time between from and to (Get
// Public Funding History)
func (e *Exchange) GetPublicFundingRate(ctx context.Context, symbol string, from, to time.Time) ([]FuturesPublicFundingHistory, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if err := common.StartEndTimeCheck(from, to); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	var resp []FuturesPublicFundingHistory
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getPublicFundingRateEPL, common.EncodeURLValues("/v1/contract/funding-rates", params), &resp)
}

// GetFuturesFundingHistory returns a page of the account's funding fee settlements on a contract; a window may span
// at most 3 months of the last 6 (Get Private Funding History)
func (e *Exchange) GetFuturesFundingHistory(ctx context.Context, r *FuturesHistoryRequest) (*FuturesPrivateFundingHistoryResponse, error) {
	params, err := r.params()
	if err != nil {
		return nil, err
	}
	var resp *FuturesPrivateFundingHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesFundingHistoryEPL, http.MethodGet, common.EncodeURLValues("/v1/funding-history", params), nil, &resp)
}
