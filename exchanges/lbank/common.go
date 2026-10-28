package lbank

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// GetCurrencyPairs returns the symbols of every pair LBank trades
func (e *Exchange) GetCurrencyPairs(ctx context.Context) ([]string, error) {
	var resp []string
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, defaultEPL, http.MethodGet, "/v2/currencyPairs.do", nil, &resp)
}

// GetPairInfo returns the precision and minimum order sizes of a pair, or of every pair when the pair is empty
func (e *Exchange) GetPairInfo(ctx context.Context, pair currency.Pair) ([]PairInfoResponse, error) {
	params := url.Values{}
	if !pair.IsEmpty() {
		symbol, err := e.FormatSymbol(pair, asset.Spot)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
	}
	var resp []PairInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, defaultEPL, http.MethodGet, "/v2/accuracy.do", params, &resp)
}

// GetAssetConfigs returns the deposit and withdrawal settings of a coin on each chain it moves on
func (e *Exchange) GetAssetConfigs(ctx context.Context, coin currency.Code) ([]AssetConfigResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("assetCode", coin.Lower().String())
	var resp []AssetConfigResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, defaultEPL, http.MethodGet, "/v2/assetConfigs.do", params, &resp)
}

// GetTimestamp returns LBank's server time
func (e *Exchange) GetTimestamp(ctx context.Context) (time.Time, error) {
	var resp types.Time
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, defaultEPL, http.MethodGet, "/v2/timestamp.do", nil, &resp); err != nil {
		return time.Time{}, err
	}
	if resp.Time().IsZero() {
		return time.Time{}, common.ErrNoResponse
	}
	return resp.Time(), nil
}
