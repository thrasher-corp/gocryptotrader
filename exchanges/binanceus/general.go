package binanceus

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// TestConnectivity tests connectivity to the REST API
func (e *Exchange) TestConnectivity(ctx context.Context) error {
	return e.testConnectivity(ctx, false)
}

func (e *Exchange) testConnectivity(ctx context.Context, useWebsocket bool) error {
	return e.sendPublicRequest(ctx, useWebsocket, "/api/v3/ping", "ping", nil, defaultRate, nil)
}

// GetExchangeServerTime returns the server time
func (e *Exchange) GetExchangeServerTime(ctx context.Context) (*ServerTimeResponse, error) {
	return e.getExchangeServerTime(ctx, false)
}

func (e *Exchange) getExchangeServerTime(ctx context.Context, useWebsocket bool) (*ServerTimeResponse, error) {
	var resp *ServerTimeResponse
	return resp, e.sendPublicRequest(ctx, useWebsocket, "/api/v3/time", "time", nil, defaultRate, &resp)
}

// GetSystemStatus returns whether the system is normal or under maintenance
func (e *Exchange) GetSystemStatus(ctx context.Context) (*SystemStatusResponse, error) {
	var resp *SystemStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/system/status", nil, defaultRate, &resp)
}

// GetExchangeInfo returns the current exchange trading rules and symbol information. A nil request returns every symbol
// with the SPOT permission
func (e *Exchange) GetExchangeInfo(ctx context.Context, req *ExchangeInfoRequest) (*ExchangeInfoResponse, error) {
	return e.getExchangeInfo(ctx, req, false)
}

func (e *Exchange) getExchangeInfo(ctx context.Context, req *ExchangeInfoRequest, useWebsocket bool) (*ExchangeInfoResponse, error) {
	params := url.Values{}
	if req != nil {
		if len(req.Symbols) > 0 && len(req.Permissions) > 0 {
			return nil, fmt.Errorf("%w: permissions cannot be sent with symbols", errInvalidParameterCombination)
		}
		if err := e.setSymbolParams(params, req.Symbols); err != nil {
			return nil, err
		}
		switch len(req.Permissions) {
		case 0:
		case 1:
			params.Set("permissions", string(req.Permissions[0]))
		default:
			permissions, err := json.Marshal(req.Permissions)
			if err != nil {
				return nil, err
			}
			params.Set("permissions", string(permissions))
		}
		if req.ShowPermissionSets != nil {
			params.Set("showPermissionSets", strconv.FormatBool(*req.ShowPermissionSets))
		}
	}
	var resp *ExchangeInfoResponse
	return resp, e.sendPublicRequest(ctx, useWebsocket, "/api/v3/exchangeInfo", "exchangeInfo", params, exchangeInfoRate, &resp)
}
