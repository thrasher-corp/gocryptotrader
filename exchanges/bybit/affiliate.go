package bybit

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var (
	errClientUIDRequired   = errors.New("client UID required")
	errDateRangeIncomplete = errors.New("start and end dates must be set together")
)

// GetAffiliateUserList returns a page of an affiliate's users, with an API key that only has the Affiliate
// permission (Get Affiliate User List)
func (e *Exchange) GetAffiliateUserList(ctx context.Context, req *AffiliateUserListRequest) (*AffiliateUserListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartDate.IsZero() && !req.EndDate.IsZero() && req.StartDate.After(req.EndDate) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if req.Size != 0 {
		params.Set("size", strconv.FormatUint(req.Size, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	if req.NeedDeposit {
		params.Set("needDeposit", "true")
	}
	if req.Need30Day {
		params.Set("need30", "true")
	}
	if req.Need365Day {
		params.Set("need365", "true")
	}
	if !req.StartDate.IsZero() {
		params.Set("startDate", req.StartDate.UTC().Format(time.DateOnly))
	}
	if !req.EndDate.IsZero() {
		params.Set("endDate", req.EndDate.UTC().Format(time.DateOnly))
	}
	var resp *AffiliateUserListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/affiliate/aff-user-list", params, nil, &resp, defaultEPL)
}

// GetAffiliateUserInfo returns the trading volume, deposits and commissions of one of an affiliate's users, with an
// API key that only has the Affiliate permission (Get Affiliate User Info)
func (e *Exchange) GetAffiliateUserInfo(ctx context.Context, req *AffiliateUserInfoRequest) (*AffiliateUserInfoResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.UID == "" {
		return nil, errClientUIDRequired
	}
	params := url.Values{}
	params.Set("uid", req.UID)
	if !req.Coin.IsEmpty() {
		params.Set("coin", req.Coin.Upper().String())
	}
	if req.Business != "" {
		params.Set("business", req.Business)
	}
	var resp *AffiliateUserInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/aff-customer-info", params, nil, &resp, getAffiliateUserInfoEPL)
}

// GetAffiliateSubAffiliateList returns a page of an affiliate's sub-affiliates and their commissions, with an API key
// that only has the Affiliate permission (Get Affiliate Sub-Affiliate List)
func (e *Exchange) GetAffiliateSubAffiliateList(ctx context.Context, req *AffiliateSubAffiliateListRequest) (*AffiliateSubAffiliateListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.StartDate.IsZero() != req.EndDate.IsZero() {
		return nil, errDateRangeIncomplete
	}
	if req.StartDate.After(req.EndDate) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	if req.Size != 0 {
		params.Set("size", strconv.FormatUint(req.Size, 10))
	}
	if !req.StartDate.IsZero() {
		params.Set("startDate", req.StartDate.UTC().Format(time.DateOnly))
		params.Set("endDate", req.EndDate.UTC().Format(time.DateOnly))
	}
	if req.SubAffiliateID != 0 {
		params.Set("subAffId", strconv.FormatUint(req.SubAffiliateID, 10))
	}
	var resp *AffiliateSubAffiliateListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/affiliate/affiliate-sub-list", params, nil, &resp, defaultEPL)
}
