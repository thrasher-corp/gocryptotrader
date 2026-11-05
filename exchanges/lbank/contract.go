package lbank

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// ProductGroupUSDTSwap is the product group of LBank's USDT margined perpetual contracts, its only contracts
const ProductGroupUSDTSwap = "SwapU"

// GetContractServerTime returns the contract API's server time
func (e *Exchange) GetContractServerTime(ctx context.Context) (time.Time, error) {
	var resp types.Time
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, contractEPL, http.MethodGet, "/cfd/openApi/v1/pub/getTime", nil, &resp); err != nil {
		return time.Time{}, err
	}
	if resp.Time().IsZero() {
		return time.Time{}, common.ErrNoResponse
	}
	return resp.Time(), nil
}

// GetContractInstruments returns the contracts of a product group
func (e *Exchange) GetContractInstruments(ctx context.Context, productGroup string) ([]ContractInstrumentResponse, error) {
	if productGroup == "" {
		return nil, errProductGroupRequired
	}
	params := url.Values{}
	params.Set("productGroup", productGroup)
	var resp []ContractInstrumentResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, contractEPL, http.MethodGet, "/cfd/openApi/v1/pub/instrument", params, &resp)
}

// GetContractMarketData returns the 24 hour statistics, mark price and funding rate of each contract in a product group
func (e *Exchange) GetContractMarketData(ctx context.Context, productGroup string) ([]ContractMarketDataResponse, error) {
	if productGroup == "" {
		return nil, errProductGroupRequired
	}
	params := url.Values{}
	params.Set("productGroup", productGroup)
	var resp []ContractMarketDataResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, contractEPL, http.MethodGet, "/cfd/openApi/v1/pub/marketData", params, &resp)
}

// GetContractOrderbook returns a contract's order book to the depth given
func (e *Exchange) GetContractOrderbook(ctx context.Context, pair currency.Pair, depth uint64) (*ContractOrderbookResponse, error) {
	if depth == 0 {
		return nil, fmt.Errorf("%w: depth must be at least 1", errLimitOutOfRange)
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbol, err := e.FormatSymbol(pair, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("depth", strconv.FormatUint(depth, 10))
	var resp *ContractOrderbookResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, contractEPL, http.MethodGet, "/cfd/openApi/v1/pub/marketOrder", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}
