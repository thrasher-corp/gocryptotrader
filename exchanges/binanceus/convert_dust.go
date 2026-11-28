package binanceus

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// ConvertDust converts the dust balances of fromAssets to toAsset, such as BNB, BTC, ETH or USDT
func (e *Exchange) ConvertDust(ctx context.Context, fromAssets []currency.Code, toAsset currency.Code) (*ConvertDustResponse, error) {
	if len(fromAssets) == 0 {
		return nil, fmt.Errorf("%w: fromAsset", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	for i := range fromAssets {
		if fromAssets[i].IsEmpty() {
			return nil, fmt.Errorf("%w: fromAsset %d", currency.ErrCurrencyCodeEmpty, i)
		}
		params.Add("fromAsset", fromAssets[i].Upper().String())
	}
	if toAsset.IsEmpty() {
		return nil, fmt.Errorf("%w: toAsset", currency.ErrCurrencyCodeEmpty)
	}
	params.Set("toAsset", toAsset.Upper().String())
	var resp *ConvertDustResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/asset/dust", params, convertDustRate, &resp)
}

// GetConvertDustHistory returns the account's dust conversions between startTime and endTime
func (e *Exchange) GetConvertDustHistory(ctx context.Context, startTime, endTime time.Time) (*ConvertDustHistoryResponse, error) {
	if startTime.IsZero() {
		return nil, fmt.Errorf("%w: startTime", common.ErrDateUnset)
	}
	if endTime.IsZero() {
		return nil, fmt.Errorf("%w: endTime", common.ErrDateUnset)
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, startTime, endTime); err != nil {
		return nil, err
	}
	var resp *ConvertDustHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/asset/query/dust-logs", params, defaultRate, &resp)
}

// GetAssetsThatCanBeConverted returns the account's dust that can be converted to toAsset, such as BNB, BTC, ETH or
// USDT, and what converting it would receive
func (e *Exchange) GetAssetsThatCanBeConverted(ctx context.Context, toAsset currency.Code) (*ConvertibleAssetsResponse, error) {
	if toAsset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("toAsset", toAsset.Upper().String())
	var resp *ConvertibleAssetsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/asset/query/dust-assets", params, defaultRate, &resp)
}
