package kucoin

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetBasicFee returns the account's basic spot and margin fee rates (Get Basic Fee - Spot/Margin). currencyType is 0
// for cryptocurrencies, the default, or 1 for fiat currencies
func (e *Exchange) GetBasicFee(ctx context.Context, currencyType uint64) (*BasicFeeResponse, error) {
	params := url.Values{}
	if currencyType != 0 {
		params.Set("currencyType", strconv.FormatUint(currencyType, 10))
	}
	var resp *BasicFeeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getBasicFeeEPL, http.MethodGet, common.EncodeURLValues("/v1/base-fee", params), nil, &resp)
}

// GetTradingPairActualFees returns the actual fee rates of up to 10 spot or margin symbols (Get Actual Fee -
// Spot/Margin). Sub-accounts pay the master account's rates
func (e *Exchange) GetTradingPairActualFees(ctx context.Context, symbols []string) ([]TradingPairActualFee, error) {
	if len(symbols) == 0 || slices.Contains(symbols, "") {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbols", strings.Join(symbols, ","))
	var resp []TradingPairActualFee
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getTradingPairActualFeesEPL, http.MethodGet, common.EncodeURLValues("/v1/trade-fees", params), nil, &resp)
}

// GetFuturesTradingPairsActualFees returns the actual fee rates of a futures symbol (Get Actual Fee - Futures).
// Sub-accounts pay the master account's rates
func (e *Exchange) GetFuturesTradingPairsActualFees(ctx context.Context, symbol string) (*FuturesTradingPairsActualFeesResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesTradingPairsActualFeesResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesTradingPairsActualFeesEPL, http.MethodGet, common.EncodeURLValues("/v1/trade-fees", params), nil, &resp)
}
