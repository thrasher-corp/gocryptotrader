package binance

import (
	"context"
	"net/http"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetFuturesLeadTraderStatus calls Get Futures Lead Trader Status
func (e *Exchange) GetFuturesLeadTraderStatus(ctx context.Context) (*LeadTraderStatusResponse, error) {
	var resp *LeadTraderStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/copyTrading/futures/userStatus", nil, sapiDefaultRate, &resp)
}

// GetFuturesLeadTradingSymbolWhitelist calls Get Futures Lead Trading Symbol Whitelist
func (e *Exchange) GetFuturesLeadTradingSymbolWhitelist(ctx context.Context) (*LeadTradingSymbolWhitelistResponse, error) {
	var resp *LeadTradingSymbolWhitelistResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/copyTrading/futures/leadSymbol", nil, sapiDefaultRate, &resp)
}
