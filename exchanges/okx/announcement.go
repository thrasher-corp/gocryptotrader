package okx

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// GetAnnouncements retrieves a page of 20 announcements, newest first. OKX limits them by the request IP, or by the
// account's country of residence when the request is authenticated
func (e *Exchange) GetAnnouncements(ctx context.Context, req *AnnouncementsRequest) (*AnnouncementsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.AnnouncementType != "" {
		params.Set("annType", req.AnnouncementType)
	}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	var authType request.AuthType = request.UnauthenticatedRequest
	if e.AreCredentialsValid(ctx) {
		authType = request.AuthenticatedRequest
	}
	var resp *AnnouncementsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAnnouncementsEPL, http.MethodGet, common.EncodeURLValues("support/announcements", params), nil, &resp, authType)
}

// GetAnnouncementTypes retrieves the announcement types, limited by the request IP
func (e *Exchange) GetAnnouncementTypes(ctx context.Context) ([]AnnouncementTypeResponse, error) {
	var resp []AnnouncementTypeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAnnouncementTypeEPL, http.MethodGet, "support/announcement-types", nil, &resp, request.UnauthenticatedRequest)
}
