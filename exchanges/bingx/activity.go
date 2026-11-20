package bingx

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetActivityList returns a page of the activities the account can see, from Query Activity List
func (e *Exchange) GetActivityList(ctx context.Context, arg *ActivityListRequest) (*ActivityListResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	var resp *ActivityListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/activity/v1/getActivityList", activityListParams(arg), activityListRate, &resp)
}

// GetPublicActivityList returns a page of the public activities, from Query Public Activity List
func (e *Exchange) GetPublicActivityList(ctx context.Context, arg *ActivityListRequest) (*ActivityListResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	var resp *ActivityListResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/openApi/activity/v1/getPublicActivityList", activityListParams(arg)), publicActivityListRate, &resp)
}

// activityListParams returns the query parameters of the activity list queries
func activityListParams(arg *ActivityListRequest) url.Values {
	params := url.Values{}
	if arg.ActivityName != "" {
		params.Set("activityName", arg.ActivityName)
	}
	if arg.Status > 0 {
		params.Set("status", strconv.FormatUint(uint64(arg.Status), 10))
	}
	if arg.TagURLKey != "" {
		params.Set("tagUrlKey", arg.TagURLKey)
	}
	if arg.PageIndex > 0 {
		params.Set("pageIndex", strconv.FormatUint(arg.PageIndex, 10))
	}
	if arg.PageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	return params
}
