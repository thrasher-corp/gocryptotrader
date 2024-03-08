package poloniex

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

const (
	poloniexFuturesAPIURL = "https://futures-api.poloniex.com/api"
)

// GetOpenContractList retrieves the info of all open contracts.
func (p *Poloniex) GetOpenContractList(ctx context.Context) (*Contracts, error) {
	var resp *Contracts
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/v1/contracts/active", &resp)
}

// GetOrderInfoOfTheContract info of the specified contract.
func (p *Poloniex) GetOrderInfoOfTheContract(ctx context.Context, symbol string) (*ContractItem, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *ContractItem
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/v1/contracts/"+symbol, &resp)
}

// GetRealTimeTicker real-time ticker 1.0 includes the last traded price, the last traded size, transaction ID,
// the side of the liquidity taker, the best bid price and size, the best ask price and size as well as the transaction time of the orders.
func (p *Poloniex) GetRealTimeTicker(ctx context.Context, symbol string) (*TickerDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *TickerDetail
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/v1/ticker?symbol="+symbol, &resp)
}

// TestGetRealTimeTickersOfSymbols retrieves real-time tickers includes tickers of all trading symbols.
func (p *Poloniex) TestGetRealTimeTickersOfSymbols(ctx context.Context) ([]TickerInfo, error) {
	var resp []TickerInfo
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/v2/tickers", &resp)
}

// GetFullOrderbookLevel2 retrieves a snapshot of aggregated open orders for a symbol.
// level 2 order book includes all bids and asks (aggregated by price). This level returns only one aggregated size for each price (as if there was only one single order for that price).
func (p *Poloniex) GetFullOrderbookLevel2(ctx context.Context, symbol string) (*Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *Orderbook
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, common.EncodeURLValues("/v1/level2/snapshot", params), &resp)
}

// GetPartialOrderbookLevel2 represents partial snapshot of aggregated open orders for a symbol.
// depth: depth5, depth10, depth20 , depth30 , depth50 or depth100
func (p *Poloniex) GetPartialOrderbookLevel2(ctx context.Context, symbol, depth string) (*Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if depth == "" {
		return nil, errors.New("depth is required")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("depth", depth)
	var resp *Orderbook
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, common.EncodeURLValues("/v1/level2/depth", params), &resp)
}

// Level2PullingMessages if the messages pushed by Websocket are not continuous, you can submit the following request and re-pull the data to ensure that the sequence is not missing.
func (p *Poloniex) Level2PullingMessages(ctx context.Context, symbol string, startSequence, endSequence int64) (*OrderbookChanges, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if startSequence <= 0 {
		return nil, errors.New("start sequence is required")
	}
	if endSequence <= 0 {
		return nil, errors.New("end sequence is required")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("start", strconv.FormatInt(startSequence, 10))
	params.Set("end", strconv.FormatInt(endSequence, 10))
	var resp *OrderbookChanges
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, common.EncodeURLValues("/v1/level2/message/query", params), &resp)
}

// GetFullOrderBookLevel3 a snapshot of all the open orders for a symbol. The Level 3 order book includes all bids and asks (the data is non-aggregated, and each item means a single order).
// To ensure your local orderbook data is the latest one, please use Websocket incremental feed after retrieving the level 3 snapshot.
func (p *Poloniex) GetFullOrderBookLevel3(ctx context.Context, symbol string) (*Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *Orderbook
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/v2/level3/snapshot", &resp)
}

// Level3PullingMessages If the messages pushed by the Websocket is not continuous, you can submit the following request and re-pull the data to ensure that the sequence is not missing.
func (p *Poloniex) Level3PullingMessages(ctx context.Context) (*Level3PullingMessageResponse, error) {
	var resp *Level3PullingMessageResponse
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/v2/level3/snapshot", &resp)
}
