package poloniex

import (
	"context"

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
func (p *Poloniex) GetRealTimeTicker(ctx context.Context, symbol string) (*TickerInfo, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *TickerInfo
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/v1/ticker?symbol="+symbol, &resp)
}
