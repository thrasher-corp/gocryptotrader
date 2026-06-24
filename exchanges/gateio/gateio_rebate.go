package gateio

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// rebateTransactionHistoryParams builds the shared query parameters used by the agency and partner transaction history endpoints.
func rebateTransactionHistoryParams(arg *RebateTransactionHistoryRequest) (url.Values, error) {
	params := url.Values{}
	if arg == nil {
		return params, nil
	}
	if !arg.From.IsZero() && !arg.To.IsZero() {
		if err := common.StartEndTimeCheck(arg.From, arg.To); err != nil {
			return nil, err
		}
	}
	if !arg.CurrencyPair.IsEmpty() {
		params.Set("currency_pair", arg.CurrencyPair.String())
	}
	if arg.UserID != 0 {
		params.Set("user_id", strconv.FormatUint(arg.UserID, 10))
	}
	if !arg.From.IsZero() {
		params.Set("from", strconv.FormatInt(arg.From.UTC().Unix(), 10))
	}
	if !arg.To.IsZero() {
		params.Set("to", strconv.FormatInt(arg.To.UTC().Unix(), 10))
	}
	if arg.Limit != 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Offset != 0 {
		params.Set("offset", strconv.FormatUint(arg.Offset, 10))
	}
	return params, nil
}

// rebateCommissionHistoryParams builds the shared query parameters used by the agency and partner commission history endpoints.
func rebateCommissionHistoryParams(arg *RebateCommissionHistoryRequest) (url.Values, error) {
	params := url.Values{}
	if arg == nil {
		return params, nil
	}
	if !arg.From.IsZero() && !arg.To.IsZero() {
		if err := common.StartEndTimeCheck(arg.From, arg.To); err != nil {
			return nil, err
		}
	}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.String())
	}
	if arg.CommissionType != 0 {
		params.Set("commission_type", strconv.FormatUint(arg.CommissionType, 10))
	}
	if arg.UserID != 0 {
		params.Set("user_id", strconv.FormatUint(arg.UserID, 10))
	}
	if !arg.From.IsZero() {
		params.Set("from", strconv.FormatInt(arg.From.UTC().Unix(), 10))
	}
	if !arg.To.IsZero() {
		params.Set("to", strconv.FormatInt(arg.To.UTC().Unix(), 10))
	}
	if arg.Limit != 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Offset != 0 {
		params.Set("offset", strconv.FormatUint(arg.Offset, 10))
	}
	return params, nil
}

// rebateBrokerHistoryParams builds the shared query parameters used by the broker commission and transaction history endpoints.
func rebateBrokerHistoryParams(arg *RebateBrokerHistoryRequest) (url.Values, error) {
	params := url.Values{}
	if arg == nil {
		return params, nil
	}
	if !arg.From.IsZero() && !arg.To.IsZero() {
		if err := common.StartEndTimeCheck(arg.From, arg.To); err != nil {
			return nil, err
		}
	}
	if arg.UserID != 0 {
		params.Set("user_id", strconv.FormatUint(arg.UserID, 10))
	}
	if !arg.From.IsZero() {
		params.Set("from", strconv.FormatInt(arg.From.UTC().Unix(), 10))
	}
	if !arg.To.IsZero() {
		params.Set("to", strconv.FormatInt(arg.To.UTC().Unix(), 10))
	}
	if arg.Limit != 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Offset != 0 {
		params.Set("offset", strconv.FormatUint(arg.Offset, 10))
	}
	return params, nil
}

// GetAgencyTransactionHistory retrieves a broker's transaction history of recommended users.
func (e *Exchange) GetAgencyTransactionHistory(ctx context.Context, arg *RebateTransactionHistoryRequest) (*AgencyTransactionHistoryResponse, error) {
	params, err := rebateTransactionHistoryParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *AgencyTransactionHistoryResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebateAgencyTransactionHistoryEPL, http.MethodGet, "rebate/agency/transaction_history", params, nil, &resp)
}

// GetAgencyCommissionHistory retrieves a broker's rebate history of recommended users.
func (e *Exchange) GetAgencyCommissionHistory(ctx context.Context, arg *RebateCommissionHistoryRequest) (*AgencyCommissionHistoryResponse, error) {
	params, err := rebateCommissionHistoryParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *AgencyCommissionHistoryResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebateAgencyCommissionHistoryEPL, http.MethodGet, "rebate/agency/commission_history", params, nil, &resp)
}

// GetPartnerTransactionHistory retrieves a partner's transaction history of recommended users.
func (e *Exchange) GetPartnerTransactionHistory(ctx context.Context, arg *RebateTransactionHistoryRequest) (*PartnerTransactionHistoryResponse, error) {
	params, err := rebateTransactionHistoryParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *PartnerTransactionHistoryResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebatePartnerTransactionHistoryEPL, http.MethodGet, "rebate/partner/transaction_history", params, nil, &resp)
}

// GetPartnerCommissionHistory retrieves a partner's rebate records of recommended users.
func (e *Exchange) GetPartnerCommissionHistory(ctx context.Context, arg *RebateCommissionHistoryRequest) (*PartnerCommissionHistoryResponse, error) {
	params, err := rebateCommissionHistoryParams(arg)
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
	params, err := rebateBrokerHistoryParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *BrokerCommissionHistoryResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebateBrokerCommissionHistoryEPL, http.MethodGet, "rebate/broker/commission_history", params, nil, &resp)
}

// GetBrokerTransactionHistory retrieves a broker's trading history for users.
func (e *Exchange) GetBrokerTransactionHistory(ctx context.Context, arg *RebateBrokerHistoryRequest) (*BrokerTransactionHistoryResponse, error) {
	params, err := rebateBrokerHistoryParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *BrokerTransactionHistoryResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebateBrokerTransactionHistoryEPL, http.MethodGet, "rebate/broker/transaction_history", params, nil, &resp)
}

// GetUserRebateInformation retrieves the authenticated user's rebate information, returning the inviter's UID.
func (e *Exchange) GetUserRebateInformation(ctx context.Context) (uint64, error) {
	var resp struct {
		InviteUID uint64 `json:"invite_uid"`
	}
	return resp.InviteUID, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebateUserInfoEPL, http.MethodGet, "rebate/user/info", nil, nil, &resp)
}

// GetUserSubordinateRelationship queries whether the specified users are within the system.
func (e *Exchange) GetUserSubordinateRelationship(ctx context.Context, userIDList []string) (*UserSubordinateRelationResponse, error) {
	if len(userIDList) == 0 {
		return nil, errUserIDRequired
	}
	params := url.Values{}
	params.Set("user_id_list", strings.Join(userIDList, ","))
	var resp *UserSubordinateRelationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebateUserSubRelationEPL, http.MethodGet, "rebate/user/sub_relation", params, nil, &resp)
}

// GetRecentPartnerApplicationRecords retrieves the current user's recent partner application records.
func (e *Exchange) GetRecentPartnerApplicationRecords(ctx context.Context) (*RecentPartnerApplicationRecords, error) {
	var resp *RecentPartnerApplicationRecords
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebatePartnerApplicationsRecentEPL, http.MethodGet, "rebate/partner/applications/recent", nil, nil, &resp)
}

// CheckPartnerApplicationEligibility check partner application eligibility
func (e *Exchange) CheckPartnerApplicationEligibility(ctx context.Context) (*RebaseEligibilityResponse, error) {
	var resp *RebaseEligibilityResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebatePartnerEligibilityEPL, http.MethodGet, "rebate/partner/eligibility", nil, nil, &resp)
}

// GetAggregatedPartnerAgentStatistics retrieves aggregated partner agent statistics
// Business type filter: - 0: All (default) - 1: Spot - 2: Futures - 3: Alpha - 4: Web3 - 5: Perps (DEX) - 6: Exchange All - 7: Web3 All - 8: TradFi
func (e *Exchange) GetAggregatedPartnerAgentStatistics(ctx context.Context, startTime, endTime time.Time, businessType uint64) (*RebateAgentStatisticsResponse, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("start_date", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end_date", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if businessType > 0 {
		params.Set("business_type", strconv.FormatUint(businessType, 10))
	}
	var resp *RebateAgentStatisticsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rebatePartnerDataAggregatedEPL, http.MethodGet, "rebate/partner/data/aggregated", params, nil, &resp)
}
