package kucoin

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetAffiliateInvitees returns a page of the users an affiliate invited (Get Invited)
func (e *Exchange) GetAffiliateInvitees(ctx context.Context, req *AffiliateInviteesRequest) (*AffiliateInviteesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.RegistrationStartAt.IsZero() && !req.RegistrationEndAt.IsZero() {
		if err := common.StartEndTimeCheck(req.RegistrationStartAt, req.RegistrationEndAt); err != nil {
			return nil, err
		}
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
	if !req.RegistrationStartAt.IsZero() {
		params.Set("registrationStartAt", strconv.FormatInt(req.RegistrationStartAt.UnixMilli(), 10))
	}
	if !req.RegistrationEndAt.IsZero() {
		params.Set("registrationEndAt", strconv.FormatInt(req.RegistrationEndAt.UnixMilli(), 10))
	}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp *AffiliateInviteesResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getAffiliateInviteesEPL, http.MethodGet, common.EncodeURLValues("/v2/affiliate/queryInvitees", params), nil, &resp)
}

// GetAffiliateCommission returns a page of an affiliate's commission payouts (Get Commission)
func (e *Exchange) GetAffiliateCommission(ctx context.Context, req *AffiliateCommissionRequest) (*AffiliateCommissionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.RebateStartAt.IsZero() && !req.RebateEndAt.IsZero() {
		if err := common.StartEndTimeCheck(req.RebateStartAt, req.RebateEndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.SiteType != "" {
		params.Set("siteType", req.SiteType)
	}
	if req.RebateType != 0 {
		params.Set("rebateType", strconv.FormatUint(req.RebateType, 10))
	}
	if !req.RebateStartAt.IsZero() {
		params.Set("rebateStartAt", strconv.FormatInt(req.RebateStartAt.UnixMilli(), 10))
	}
	if !req.RebateEndAt.IsZero() {
		params.Set("rebateEndAt", strconv.FormatInt(req.RebateEndAt.UnixMilli(), 10))
	}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	if req.UserID != "" {
		params.Set("userId", req.UserID)
	}
	if req.DataType != "" {
		params.Set("dataType", req.DataType)
	}
	var resp *AffiliateCommissionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getAffiliateCommissionEPL, http.MethodGet, common.EncodeURLValues("/v2/affiliate/queryMyCommission", params), nil, &resp)
}

// GetAffiliateTradeHistory returns a page of an invitee's trades (Get Trade History)
func (e *Exchange) GetAffiliateTradeHistory(ctx context.Context, req *AffiliateTradeHistoryRequest) (*AffiliateTradeHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.UID == "" {
		return nil, errUIDRequired
	}
	if !req.TradeStartAt.IsZero() && !req.TradeEndAt.IsZero() {
		if err := common.StartEndTimeCheck(req.TradeStartAt, req.TradeEndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("uid", req.UID)
	if req.TradeType != "" {
		params.Set("tradeType", req.TradeType)
	}
	if !req.TradeStartAt.IsZero() {
		params.Set("tradeStartAt", strconv.FormatInt(req.TradeStartAt.UnixMilli(), 10))
	}
	if !req.TradeEndAt.IsZero() {
		params.Set("tradeEndAt", strconv.FormatInt(req.TradeEndAt.UnixMilli(), 10))
	}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp *AffiliateTradeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getAffiliateTradeHistoryEPL, http.MethodGet, common.EncodeURLValues("/v2/affiliate/queryTransactionByUid", params), nil, &resp)
}

// GetAffiliateTransactions returns the invitees' trades in a time window, paged by the last record's ID (Get
// Transaction)
func (e *Exchange) GetAffiliateTransactions(ctx context.Context, req *AffiliateTransactionsRequest) ([]AffiliateTransaction, error) {
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
	var resp []AffiliateTransaction
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getAffiliateTransactionsEPL, http.MethodGet, common.EncodeURLValues("/v2/affiliate/queryTransactionByTime", params), nil, &resp)
}

// GetAffiliateKumining returns KuMining purchase records, paged by the last record's ID (Get Kumining)
func (e *Exchange) GetAffiliateKumining(ctx context.Context, req *AffiliateKuminingRequest) ([]AffiliateKumining, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartAt.IsZero() && !req.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(req.StartAt, req.EndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.UID != "" {
		params.Set("uid", req.UID)
	}
	if !req.StartAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(req.StartAt.UnixMilli(), 10))
	}
	if !req.EndAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(req.EndAt.UnixMilli(), 10))
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
	var resp []AffiliateKumining
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getAffiliateKuminingEPL, http.MethodGet, common.EncodeURLValues("/v2/affiliate/queryKumining", params), nil, &resp)
}
