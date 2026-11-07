package coinbase

import (
	"context"
	"net/http"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// GetAPIKeyPermissions returns what the CDP API key may do and the portfolio it is permissioned for
func (e *Exchange) GetAPIKeyPermissions(ctx context.Context) (*APIKeyPermissionsResponse, error) {
	var resp *APIKeyPermissionsResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"key_permissions", nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}
