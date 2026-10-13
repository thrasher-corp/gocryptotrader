package bybit

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetInstitutionWhitelistIP returns the account's dedicated institution hostnames and the IP addresses whitelisted for
// them (Get Institution Whitelist IP). It needs a main account API key
func (e *Exchange) GetInstitutionWhitelistIP(ctx context.Context) ([]InstitutionWhitelistIP, error) {
	var resp []InstitutionWhitelistIP
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins/whitelist/ip", nil, nil, &resp, defaultEPL)
}

// GetInstitutionIPChangeLog returns changes to the IP addresses whitelisted for the account's dedicated institution
// hostnames (Get Institution IP Change Log). It needs a main account API key
func (e *Exchange) GetInstitutionIPChangeLog(ctx context.Context, req *InstitutionIPChangeLogRequest) (*InstitutionIPChangeLogResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *InstitutionIPChangeLogResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins/ip/changelog", params, nil, &resp, defaultEPL)
}
