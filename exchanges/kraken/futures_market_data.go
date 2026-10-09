package kraken

import (
	"context"
	"net/url"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// GetFuturesTickers calls Get tickers, returning the tickers of the listed markets. A nil request returns every
// futures market's ticker, which excludes options
func (e *Exchange) GetFuturesTickers(ctx context.Context, req *FuturesTickersRequest) (*FuturesTickersResponse, error) {
	var params url.Values
	if req != nil {
		params = futuresContractTypeParams(req.ContractTypes)
		for i := range req.Pairs {
			symbol, err := e.futuresMarketSymbol(req.Pairs[i])
			if err != nil {
				return nil, err
			}
			params.Add("symbol", symbol)
		}
	}
	var resp *FuturesTickersResponse
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues("/api/v3/tickers", params), &resp)
}

// GetFuturesTicker calls Get ticker by symbol, returning a market's ticker
func (e *Exchange) GetFuturesTicker(ctx context.Context, pair currency.Pair) (*FuturesTickerResponse, error) {
	symbol, err := e.futuresMarketSymbol(pair)
	if err != nil {
		return nil, err
	}
	var resp *FuturesTickerResponse
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFutures, "/api/v3/tickers/"+url.PathEscape(symbol), &resp)
}

// GetFuturesOrderbook calls Get orderbook, returning every price level of a market's order book
func (e *Exchange) GetFuturesOrderbook(ctx context.Context, pair currency.Pair) (*FuturesOrderbookResponse, error) {
	symbol, err := e.futuresMarketSymbol(pair)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesOrderbookResponse
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues("/api/v3/orderbook", params), &resp)
}

// GetFuturesTradeHistory calls Get trade history, returning up to 100 of a market's trades from the last 7 days
func (e *Exchange) GetFuturesTradeHistory(ctx context.Context, req *FuturesTradeHistoryRequest) (*FuturesTradeHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	symbol, err := e.futuresMarketSymbol(req.Pair)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !req.LastTime.IsZero() {
		params.Set("lastTime", req.LastTime.UTC().Format(time.RFC3339Nano))
	}
	if req.IncludeMTFData {
		params.Set("includeMTFData", "true")
	}
	var resp *FuturesTradeHistoryResponse
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues("/api/v3/history", params), &resp)
}

// futuresMarketSymbol returns a pair as the market symbol the Derivatives REST API takes, such as PF_XBTUSD
func (e *Exchange) futuresMarketSymbol(pair currency.Pair) (string, error) {
	if pair.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	return e.FormatSymbol(pair, asset.Futures)
}

// futuresContractTypeParams returns the contractType parameter the list endpoints take, repeated for each type
func futuresContractTypeParams(contractTypes []string) url.Values {
	params := url.Values{}
	for _, t := range contractTypes {
		params.Add("contractType", t)
	}
	return params
}
