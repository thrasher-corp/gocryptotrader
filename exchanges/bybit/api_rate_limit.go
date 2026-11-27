package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var (
	errNoAPIRateLimits      = errors.New("no API rate limits provided")
	errBusinessTypeRequired = errors.New("business type required")
	errRateRequired         = errors.New("rate required")
)

// SetAPIRateLimit sets the per second API rate limit of UIDs for a business type, within the institution's cap. Only
// institutional users can set limits: a master account for its subaccounts, and a subaccount for itself (Set Rate
// Limit)
func (e *Exchange) SetAPIRateLimit(ctx context.Context, limits []SetAPIRateLimitRequest) (*SetAPIRateLimitResponse, error) {
	if len(limits) == 0 {
		return nil, errNoAPIRateLimits
	}
	list := make([]map[string]any, len(limits))
	for i := range limits {
		switch {
		case len(limits[i].UIDs) == 0:
			return nil, fmt.Errorf("%w: limit %d", errMembersIDsNotSet, i)
		case limits[i].BusinessType == "":
			return nil, fmt.Errorf("%w: limit %d", errBusinessTypeRequired, i)
		case limits[i].Rate == 0:
			return nil, fmt.Errorf("%w: limit %d", errRateRequired, i)
		}
		list[i] = map[string]any{
			"uids":    strings.Join(limits[i].UIDs, ","),
			"bizType": limits[i].BusinessType,
			"rate":    limits[i].Rate,
		}
	}
	body := map[string]any{"list": list}
	var resp *SetAPIRateLimitResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/apilimit/set", nil, body, &resp, setAPIRateLimitEPL)
}

// GetAPIRateLimit returns the per second API rate limits of UIDs: a master account can query itself and its
// subaccounts, and a subaccount only itself (Get Rate Limit)
func (e *Exchange) GetAPIRateLimit(ctx context.Context, uids ...string) (*APIRateLimitResponse, error) {
	if len(uids) == 0 {
		return nil, errMembersIDsNotSet
	}
	params := url.Values{}
	params.Set("uids", strings.Join(uids, ","))
	var resp *APIRateLimitResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/apilimit/query", params, nil, &resp, getAPIRateLimitEPL)
}

// GetAPIRateLimitCap returns the institution's API rate limit caps and their usage per business type (Get Rate Limit
// Cap)
func (e *Exchange) GetAPIRateLimitCap(ctx context.Context) (*APIRateLimitCapResponse, error) {
	var resp *APIRateLimitCapResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/apilimit/query-cap", nil, nil, &resp, getAPIRateLimitCapEPL)
}

// GetAllAPIRateLimits returns a page of the per second API rate limits of every master account and subaccount (Get
// All Rate Limits)
func (e *Exchange) GetAllAPIRateLimits(ctx context.Context, req *AllAPIRateLimitsRequest) (*AllAPIRateLimitsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	if len(req.UIDs) != 0 {
		params.Set("uids", strings.Join(req.UIDs, ","))
	}
	var resp *AllAPIRateLimitsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/apilimit/query-all", params, nil, &resp, getAllAPIRateLimitsEPL)
}
