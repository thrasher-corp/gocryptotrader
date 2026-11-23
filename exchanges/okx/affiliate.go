package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var errUserIDRequired = errors.New("uid is required")

const (
	affiliatePeriodTypeCustom = "custom"
	maxInviteeListUIDs        = 100
)

// GetAffiliatePerformanceSummary retrieves the affiliate's performance over a period, by commission category
func (e *Exchange) GetAffiliatePerformanceSummary(ctx context.Context, req *AffiliatePerformanceSummaryRequest) (*AffiliatePerformanceSummaryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setAffiliatePeriod(params, req.PeriodType, req.Begin, req.End); err != nil {
		return nil, err
	}
	var resp *AffiliatePerformanceSummaryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAffiliatePerformanceSummaryEPL, http.MethodGet, common.EncodeURLValues("affiliate/performance/summary", params), nil, &resp, request.AuthenticatedRequest)
}

// GetInviteeDetail retrieves an invitee's details, covering its master account and sub-accounts
func (e *Exchange) GetInviteeDetail(ctx context.Context, req *InviteeDetailRequest) (*InviteeDetailResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.UID == "" {
		return nil, errUserIDRequired
	}
	params := url.Values{}
	params.Set("uid", req.UID)
	if req.PeriodType != "" {
		params.Set("periodType", req.PeriodType)
	}
	var resp *InviteeDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getInviteeDetailEPL, http.MethodGet, common.EncodeURLValues("affiliate/invitee/detail", params), nil, &resp, request.AuthenticatedRequest)
}

// GetInviteeList retrieves a page of the affiliate's invitees with their trading statistics
func (e *Exchange) GetInviteeList(ctx context.Context, req *InviteeListRequest) (*InviteeListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if len(req.UIDs) > maxInviteeListUIDs {
		return nil, fmt.Errorf("%w: %d UIDs exceeds the maximum of %d", errExceedLimit, len(req.UIDs), maxInviteeListUIDs)
	}
	if req.JoinTimeBegin.IsZero() != req.JoinTimeEnd.IsZero() {
		return nil, fmt.Errorf("%w: join time begin and end must be sent together", common.ErrDateUnset)
	}
	if req.JoinTimeBegin.After(req.JoinTimeEnd) {
		return nil, fmt.Errorf("%w: join time", common.ErrStartAfterEnd)
	}
	params := url.Values{}
	if err := setAffiliatePeriod(params, req.PeriodType, req.Begin, req.End); err != nil {
		return nil, err
	}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Keyword != "" {
		params.Set("keyword", req.Keyword)
	}
	if req.CommissionCategory != "" {
		params.Set("commissionCategory", req.CommissionCategory)
	}
	if req.OrderBy != "" {
		params.Set("orderBy", req.OrderBy)
	}
	if req.OrderDirection != "" {
		params.Set("orderDir", req.OrderDirection)
	}
	if req.KYCStatus != "" {
		params.Set("kycStatus", req.KYCStatus)
	}
	if req.SubAffiliateUID != "" {
		params.Set("subAffiliateUid", req.SubAffiliateUID)
	}
	if len(req.UIDs) != 0 {
		params.Set("uid", strings.Join(req.UIDs, ","))
	}
	if !req.JoinTimeBegin.IsZero() {
		params.Set("joinTimeBegin", strconv.FormatInt(req.JoinTimeBegin.UnixMilli(), 10))
		params.Set("joinTimeEnd", strconv.FormatInt(req.JoinTimeEnd.UnixMilli(), 10))
	}
	envelope := new(RESTResponse)
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getInviteeListEPL, http.MethodGet, common.EncodeURLValues("affiliate/invitee/list", params), nil, envelope, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	var invitees []InviteeResponse
	if err := json.Unmarshal(envelope.Data, &invitees); err != nil {
		return nil, fmt.Errorf("error decoding invitee list: %w", err)
	}
	return &InviteeListResponse{TotalPage: envelope.TotalPage, Invitees: invitees}, nil
}

// GetAffiliateLinkList retrieves a page of the affiliate's invite links with their commission rates and statistics
func (e *Exchange) GetAffiliateLinkList(ctx context.Context, req *AffiliateLinkListRequest) (*AffiliateLinkListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.LinkType != "" {
		params.Set("linkType", req.LinkType)
	}
	if req.LinkStatus != "" {
		params.Set("linkStatus", req.LinkStatus)
	}
	envelope := new(RESTResponse)
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getAffiliateLinkListEPL, http.MethodGet, common.EncodeURLValues("affiliate/link/list", params), nil, envelope, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	var links []AffiliateLinkResponse
	if err := json.Unmarshal(envelope.Data, &links); err != nil {
		return nil, fmt.Errorf("error decoding affiliate link list: %w", err)
	}
	return &AffiliateLinkListResponse{TotalPage: envelope.TotalPage, Links: links}, nil
}

// GetCoInviterLinkList retrieves a page of the links on which the account is the co-inviter
func (e *Exchange) GetCoInviterLinkList(ctx context.Context, req *CoInviterLinkListRequest) (*CoInviterLinkListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.LinkStatus != "" {
		params.Set("linkStatus", req.LinkStatus)
	}
	envelope := new(RESTResponse)
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getCoInviterLinkListEPL, http.MethodGet, common.EncodeURLValues("affiliate/co-inviter/list", params), nil, envelope, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	var links []CoInviterLinkResponse
	if err := json.Unmarshal(envelope.Data, &links); err != nil {
		return nil, fmt.Errorf("error decoding co-inviter link list: %w", err)
	}
	return &CoInviterLinkListResponse{TotalPage: envelope.TotalPage, Links: links}, nil
}

// GetSubAffiliateList retrieves a page of the affiliate's sub-affiliates with the statistics of their invitees
func (e *Exchange) GetSubAffiliateList(ctx context.Context, req *SubAffiliateListRequest) (*SubAffiliateListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Keyword != "" {
		params.Set("keyword", req.Keyword)
	}
	if req.CommissionCategory != "" {
		params.Set("commissionCategory", req.CommissionCategory)
	}
	if req.OrderBy != "" {
		params.Set("orderBy", req.OrderBy)
	}
	if req.OrderDirection != "" {
		params.Set("orderDir", req.OrderDirection)
	}
	envelope := new(RESTResponse)
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getSubAffiliateListEPL, http.MethodGet, common.EncodeURLValues("affiliate/sub-affiliate/list", params), nil, envelope, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	var subAffiliates []SubAffiliateResponse
	if err := json.Unmarshal(envelope.Data, &subAffiliates); err != nil {
		return nil, fmt.Errorf("error decoding sub-affiliate list: %w", err)
	}
	return &SubAffiliateListResponse{TotalPage: envelope.TotalPage, SubAffiliates: subAffiliates}, nil
}

// GetUserAffiliateRebateInformation retrieves whether the user owning an API key was invited by the affiliate and
// receives a rebate; the endpoint is undocumented and superseded by GetInviteeDetail
func (e *Exchange) GetUserAffiliateRebateInformation(ctx context.Context, apiKey string) (*AffiliateRebateInformationResponse, error) {
	if apiKey == "" {
		return nil, errInvalidAPIKey
	}
	params := url.Values{}
	params.Set("apiKey", apiKey)
	var resp *AffiliateRebateInformationResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUserAffiliateRebateInformationEPL, http.MethodGet, common.EncodeURLValues("users/partner/if-rebate", params), nil, &resp, request.AuthenticatedRequest)
}

// setAffiliatePeriod adds an affiliate statistics window to params
func setAffiliatePeriod(params url.Values, periodType string, begin, end time.Time) error {
	if periodType == affiliatePeriodTypeCustom && (begin.IsZero() || end.IsZero()) {
		return fmt.Errorf("%w: a custom period needs begin and end", common.ErrDateUnset)
	}
	if !end.IsZero() && begin.After(end) {
		return common.ErrStartAfterEnd
	}
	if periodType != "" {
		params.Set("periodType", periodType)
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
	return nil
}
