package binanceus

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var errPartnerIDRequired = errors.New("API partner ID is required")

// CheckUserEligibility returns whether the user is eligible for the rebates of the API partner with partnerID, its 8
// character ID
func (e *Exchange) CheckUserEligibility(ctx context.Context, partnerID string) (*UserEligibilityResponse, error) {
	if partnerID == "" {
		return nil, errPartnerIDRequired
	}
	params := url.Values{}
	params.Set("partnerId", partnerID)
	var resp *UserEligibilityResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/apipartner/checkEligibility", params, defaultRate, &resp)
}

// GetRebateHistory returns the rebates an API partner paid the user
func (e *Exchange) GetRebateHistory(ctx context.Context, req *RebateHistoryRequest) ([]RebateRecordResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.PartnerID == "" {
		return nil, errPartnerIDRequired
	}
	params := url.Values{}
	params.Set("partnerId", req.PartnerID)
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Page > 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	var resp []RebateRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/apipartner/rebateHistory", params, defaultRate, &resp)
}
