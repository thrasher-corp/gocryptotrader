package gate

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Params returns the query parameters of the agency and partner transaction history endpoints
func (r *RebateTransactionHistoryRequest) Params() (url.Values, error) {
	if r == nil {
		return url.Values{}, nil
	}
	params, err := rebateHistoryParams(r.UserID, r.From, r.To, r.Limit, r.Offset)
	if err != nil {
		return nil, err
	}
	if !r.CurrencyPair.IsEmpty() {
		params.Set("currency_pair", r.CurrencyPair.String())
	}
	return params, nil
}

// Params returns the query parameters of the agency and partner commission history endpoints
func (r *RebateCommissionHistoryRequest) Params() (url.Values, error) {
	if r == nil {
		return url.Values{}, nil
	}
	params, err := rebateHistoryParams(r.UserID, r.From, r.To, r.Limit, r.Offset)
	if err != nil {
		return nil, err
	}
	if !r.Currency.IsEmpty() {
		params.Set("currency", r.Currency.String())
	}
	if r.CommissionType != 0 {
		params.Set("commission_type", strconv.FormatUint(r.CommissionType, 10))
	}
	return params, nil
}

// Params returns the query parameters of the broker commission and transaction history endpoints
func (r *RebateBrokerHistoryRequest) Params() (url.Values, error) {
	if r == nil {
		return url.Values{}, nil
	}
	return rebateHistoryParams(r.UserID, r.From, r.To, r.Limit, r.Offset)
}

// rebateHistoryParams returns the query parameters every rebate history endpoint shares
func rebateHistoryParams(userID uint64, from, to time.Time, limit, offset uint64) (url.Values, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if userID != 0 {
		params.Set("user_id", strconv.FormatUint(userID, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	return params, nil
}

// GetAgencyTransactionHistory retrieves a broker's transaction history of recommended users.
func (e *Exchange) GetAgencyTransactionHistory(ctx context.Context, arg *RebateTransactionHistoryRequest) (*AgencyTransactionHistoryResponse, error) {
	params, err := arg.Params()
	if err != nil {
		return nil, err
	}
	var resp *AgencyTransactionHistoryResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebateAgencyTransactionHistoryEPL, http.MethodGet, "rebate/agency/transaction_history", params, nil, &resp)
}

// GetAgencyCommissionHistory retrieves a broker's rebate history of recommended users.
func (e *Exchange) GetAgencyCommissionHistory(ctx context.Context, arg *RebateCommissionHistoryRequest) (*AgencyCommissionHistoryResponse, error) {
	params, err := arg.Params()
	if err != nil {
		return nil, err
	}
	var resp *AgencyCommissionHistoryResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebateAgencyCommissionHistoryEPL, http.MethodGet, "rebate/agency/commission_history", params, nil, &resp)
}

// GetPartnerTransactionHistory retrieves a partner's transaction history of recommended users.
func (e *Exchange) GetPartnerTransactionHistory(ctx context.Context, arg *RebateTransactionHistoryRequest) (*PartnerTransactionHistoryResponse, error) {
	params, err := arg.Params()
	if err != nil {
		return nil, err
	}
	var resp *PartnerTransactionHistoryResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebatePartnerTransactionHistoryEPL, http.MethodGet, "rebate/partner/transaction_history", params, nil, &resp)
}

// GetPartnerCommissionHistory retrieves a partner's rebate records of recommended users.
func (e *Exchange) GetPartnerCommissionHistory(ctx context.Context, arg *RebateCommissionHistoryRequest) (*PartnerCommissionHistoryResponse, error) {
	params, err := arg.Params()
	if err != nil {
		return nil, err
	}
	var resp *PartnerCommissionHistoryResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebatePartnerCommissionHistoryEPL, http.MethodGet, "rebate/partner/commission_history", params, nil, &resp)
}

// GetPartnerSubordinateList retrieves a partner's subordinate list, including sub-agents, direct customers, and indirect customers.
func (e *Exchange) GetPartnerSubordinateList(ctx context.Context, arg *PartnerSubordinateListRequest) (*PartnerSubordinateListResponse, error) {
	params := url.Values{}
	if arg != nil {
		if arg.UserID != 0 {
			params.Set("user_id", strconv.FormatUint(arg.UserID, 10))
		}
		if arg.Limit != 0 {
			params.Set("limit", strconv.FormatUint(arg.Limit, 10))
		}
		if arg.Offset != 0 {
			params.Set("offset", strconv.FormatUint(arg.Offset, 10))
		}
	}
	var resp *PartnerSubordinateListResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebatePartnerSubListEPL, http.MethodGet, "rebate/partner/sub_list", params, nil, &resp)
}

// GetBrokerCommissionHistory retrieves a broker's rebate records for users.
func (e *Exchange) GetBrokerCommissionHistory(ctx context.Context, arg *RebateBrokerHistoryRequest) (*BrokerCommissionHistoryResponse, error) {
	params, err := arg.Params()
	if err != nil {
		return nil, err
	}
	var resp *BrokerCommissionHistoryResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebateBrokerCommissionHistoryEPL, http.MethodGet, "rebate/broker/commission_history", params, nil, &resp)
}

// GetBrokerTransactionHistory retrieves a broker's trading history for users.
func (e *Exchange) GetBrokerTransactionHistory(ctx context.Context, arg *RebateBrokerHistoryRequest) (*BrokerTransactionHistoryResponse, error) {
	params, err := arg.Params()
	if err != nil {
		return nil, err
	}
	var resp *BrokerTransactionHistoryResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebateBrokerTransactionHistoryEPL, http.MethodGet, "rebate/broker/transaction_history", params, nil, &resp)
}

// GetUserRebateInformation retrieves the authenticated user's rebate information, returning the inviter's UID.
func (e *Exchange) GetUserRebateInformation(ctx context.Context) (uint64, error) {
	var resp rebateUserInfoReply
	return resp.InviteUID, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebateUserInfoEPL, http.MethodGet, "rebate/user/info", nil, nil, &resp)
}

// GetUserSubordinateRelationship queries whether the specified users are within the system. Gate only reports the first
// 100 users
func (e *Exchange) GetUserSubordinateRelationship(ctx context.Context, userIDs []uint64) (*UserSubordinateRelationResponse, error) {
	if len(userIDs) == 0 || slices.Contains(userIDs, 0) {
		return nil, errUserIDRequired
	}
	ids := make([]string, len(userIDs))
	for i := range userIDs {
		ids[i] = strconv.FormatUint(userIDs[i], 10)
	}
	params := url.Values{}
	params.Set("user_id_list", strings.Join(ids, ","))
	var resp *UserSubordinateRelationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebateUserSubRelationEPL, http.MethodGet, "rebate/user/sub_relation", params, nil, &resp)
}

// GetRecentPartnerApplicationRecords retrieves the current user's most recent partner application within the last 30
// days
func (e *Exchange) GetRecentPartnerApplicationRecords(ctx context.Context) (*PartnerApplicationRecordResponse, error) {
	var resp gateAPIResponse[*PartnerApplicationRecordResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebatePartnerApplicationsRecentEPL, http.MethodGet, "rebate/partner/applications/recent", nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResults
	}
	return resp.Data, nil
}

// CheckPartnerApplicationEligibility checks whether the current user is eligible to apply as a partner
func (e *Exchange) CheckPartnerApplicationEligibility(ctx context.Context) (*PartnerEligibilityResponse, error) {
	var resp gateAPIResponse[*PartnerEligibilityResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebatePartnerEligibilityEPL, http.MethodGet, "rebate/partner/eligibility", nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResults
	}
	return resp.Data, nil
}

// partnerStatisticsTimezone is the UTC+8 zone the aggregated partner statistics endpoint reads its dates in
var partnerStatisticsTimezone = time.FixedZone("UTC+8", 8*60*60)

// GetAggregatedPartnerAgentStatistics retrieves aggregated partner agent statistics, which default to the last 7 days.
// Business type filter: - 0: All (default) - 1: Spot - 2: Futures - 3: Alpha - 4: Web3 - 5: Perps (DEX) - 6: Exchange All - 7: Web3 All - 8: TradFi
func (e *Exchange) GetAggregatedPartnerAgentStatistics(ctx context.Context, startTime, endTime time.Time, businessType uint64) (*RebateAgentStatisticsResponse, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("start_date", startTime.In(partnerStatisticsTimezone).Format(time.DateTime))
	}
	if !endTime.IsZero() {
		params.Set("end_date", endTime.In(partnerStatisticsTimezone).Format(time.DateTime))
	}
	if businessType > 0 {
		params.Set("business_type", strconv.FormatUint(businessType, 10))
	}
	var resp gateAPIResponse[*RebateAgentStatisticsResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebatePartnerDataAggregatedEPL, http.MethodGet, "rebate/partner/data/aggregated", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResults
	}
	return resp.Data, nil
}
