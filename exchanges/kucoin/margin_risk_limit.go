package kucoin

import (
	"context"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetMarginRiskLimit returns the cross margin risk limits and configuration by currency, or the isolated margin ones
// by symbol (Get Margin Risk Limit)
func (e *Exchange) GetMarginRiskLimit(ctx context.Context, req *MarginRiskLimitRequest) ([]MarginRiskLimitResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.IsIsolated {
		params.Set("isIsolated", "true")
	}
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	var resp []MarginRiskLimitResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getMarginRiskLimitEPL, http.MethodGet, common.EncodeURLValues("/v3/margin/currencies", params), nil, &resp)
}
