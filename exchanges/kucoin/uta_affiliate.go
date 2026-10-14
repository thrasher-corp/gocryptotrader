package kucoin

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// errUTAInviteeUIDRequired is returned when an affiliate invitee request names no invitee UID
var errUTAInviteeUIDRequired = errors.New("invitee UID is required")

// GetUTAAffiliateInvitees returns a page of the affiliate's invitees (Get Invited)
func (e *Exchange) GetUTAAffiliateInvitees(ctx context.Context, req *UTAAffiliateInviteesRequest) (*UTAAffiliateInviteesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.UserType != "" {
		params.Set("userType", req.UserType)
	}
	if req.ReferralCode != "" {
		params.Set("referralCode", req.ReferralCode)
	}
	if len(req.UIDs) != 0 {
		params.Set("uid", strings.Join(req.UIDs, ","))
	}
	if err := setTimeRange(params, "registrationStartAt", "registrationEndAt", req.RegistrationStartAt, req.RegistrationEndAt); err != nil {
		return nil, err
	}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp *UTAAffiliateInviteesResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAAffiliateInviteesEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/affiliate/queryInvitees", params), nil, &resp)
}

// GetUTAAffiliateCommission returns a page of the affiliate's commission (Get Commission)
func (e *Exchange) GetUTAAffiliateCommission(ctx context.Context, req *UTAAffiliateCommissionRequest) (*UTAAffiliateCommissionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.SiteType != "" {
		params.Set("siteType", req.SiteType)
	}
	if req.RebateType != 0 {
		params.Set("rebateType", strconv.FormatUint(req.RebateType, 10))
	}
	if err := setTimeRange(params, "rebateStartAt", "rebateEndAt", req.RebateStartAt, req.RebateEndAt); err != nil {
		return nil, err
	}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	if req.DataType != "" {
		params.Set("dataType", req.DataType)
	}
	var resp *UTAAffiliateCommissionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAAffiliateCommissionEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/affiliate/queryMyCommission", params), nil, &resp)
}

// GetUTAAffiliateTradeHistory returns a page of an invitee's trades and the commission each earned (Get Trade History)
func (e *Exchange) GetUTAAffiliateTradeHistory(ctx context.Context, req *UTAAffiliateTradeHistoryRequest) (*UTAAffiliateTradeHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.UID == "" {
		return nil, errUTAInviteeUIDRequired
	}
	params := url.Values{}
	params.Set("uid", req.UID)
	if req.TradeType != "" {
		params.Set("tradeType", req.TradeType)
	}
	if err := setTimeRange(params, "tradeStartAt", "tradeEndAt", req.TradeStartAt, req.TradeEndAt); err != nil {
		return nil, err
	}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp *UTAAffiliateTradeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAAffiliateTradeHistoryEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/affiliate/queryTransactionByUid", params), nil, &resp)
}

// GetUTAAffiliateTransactions returns the invitees' trades between two times and the commission each earned (Get
// Transaction)
func (e *Exchange) GetUTAAffiliateTransactions(ctx context.Context, req *UTAAffiliateTransactionsRequest) ([]UTAAffiliateTransaction, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := common.StartEndTimeCheck(req.TradeStartAt, req.TradeEndAt); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.UID != "" {
		params.Set("uid", req.UID)
	}
	if req.TradeType != "" {
		params.Set("tradeType", req.TradeType)
	}
	params.Set("tradeStartAt", strconv.FormatInt(req.TradeStartAt.UnixMilli(), 10))
	params.Set("tradeEndAt", strconv.FormatInt(req.TradeEndAt.UnixMilli(), 10))
	if req.LastID != "" {
		params.Set("lastId", req.LastID)
	}
	if req.Direction != "" {
		params.Set("direction", req.Direction)
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp []UTAAffiliateTransaction
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAAffiliateTransactionsEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/affiliate/queryTransactionByTime", params), nil, &resp)
}

// GetUTAAffiliateKumining returns the invitees' KuMining purchases (Get Kumining)
func (e *Exchange) GetUTAAffiliateKumining(ctx context.Context, req *UTAAffiliateKuminingRequest) ([]UTAAffiliateKumining, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if len(req.UIDs) != 0 {
		params.Set("uid", strings.Join(req.UIDs, ","))
	}
	if err := setTimeRange(params, "startAt", "endAt", req.StartAt, req.EndAt); err != nil {
		return nil, err
	}
	if req.LastID != "" {
		params.Set("lastId", req.LastID)
	}
	if req.Direction != "" {
		params.Set("direction", req.Direction)
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp []UTAAffiliateKumining
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAAffiliateKuminingEPL, http.MethodGet, common.EncodeURLValues("/ua/v2/affiliate/queryKumining", params), nil, &resp)
}
