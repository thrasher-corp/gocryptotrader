package bingx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var (
	errPageIndexRequired                   = errors.New("page index is required")
	errPageSizeRequired                    = errors.New("page size is required")
	errUIDRequired                         = errors.New("UID is required")
	errAgentDepositBusinessTypeRequired    = errors.New("deposit business type is required")
	errAgentCommissionBusinessTypeRequired = errors.New("commission business type is required")
)

// GetAgentInvitedUsers returns a page of the users the agent invited, from Query Invited Users
func (e *Exchange) GetAgentInvitedUsers(ctx context.Context, arg *AgentInvitedUsersRequest) (*AgentInvitedUsersResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := agentPageParams(params, arg.PageIndex, arg.PageSize); err != nil {
		return nil, err
	}
	if err := setTimeRange(params, "startTime", "endTime", arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.LastUID > 0 {
		params.Set("lastUid", strconv.FormatUint(arg.LastUID, 10))
	}
	var resp *AgentInvitedUsersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/agent/v1/account/inviteAccountList", params, agentInvitedUsersRate, &resp)
}

// GetAgentDailyCommissionDetails returns a page of the agent's daily commission from the users it invited, from Daily
// commission details
func (e *Exchange) GetAgentDailyCommissionDetails(ctx context.Context, arg *AgentDailyCommissionDetailsRequest) (*AgentDailyCommissionDetailsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := agentRequiredTimeRange(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if err := agentPageParams(params, arg.PageIndex, arg.PageSize); err != nil {
		return nil, err
	}
	if arg.UID > 0 {
		params.Set("uid", strconv.FormatUint(arg.UID, 10))
	}
	if arg.InvitationCode != "" {
		params.Set("invitationCode", arg.InvitationCode)
	}
	if arg.BusinessType != "" {
		params.Set("businessType", string(arg.BusinessType))
	}
	if arg.DistanceType > 0 {
		params.Set("distanceType", strconv.FormatUint(uint64(arg.DistanceType), 10))
	}
	var resp *AgentDailyCommissionDetailsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/agent/v2/reward/commissionDataList", params, agentDailyCommissionRate, &resp)
}

// GetAgentUserInformation returns an invited user's relationship to the agent and activity, from Query agent user
// information
func (e *Exchange) GetAgentUserInformation(ctx context.Context, uid uint64) (*AgentUserInformationResponse, error) {
	if uid == 0 {
		return nil, errUIDRequired
	}
	params := url.Values{}
	params.Set("uid", strconv.FormatUint(uid, 10))
	var resp *AgentUserInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/agent/v1/account/inviteRelationCheck", params, agentUserInformationRate, &resp)
}

// GetAgentInvitedUserDepositDetails returns a page of the deposits of the users the agent invited, from Query the
// deposit details of invited users
func (e *Exchange) GetAgentInvitedUserDepositDetails(ctx context.Context, arg *AgentInvitedUserDepositDetailsRequest) (*AgentInvitedUserDepositDetailsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.UID == 0 {
		return nil, errUIDRequired
	}
	if arg.BusinessType == 0 {
		return nil, errAgentDepositBusinessTypeRequired
	}
	params := url.Values{}
	if err := agentRequiredTimeRange(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if err := agentPageParams(params, arg.PageIndex, arg.PageSize); err != nil {
		return nil, err
	}
	params.Set("uid", strconv.FormatUint(arg.UID, 10))
	params.Set("bizType", strconv.FormatUint(uint64(arg.BusinessType), 10))
	var resp *AgentInvitedUserDepositDetailsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/agent/v1/asset/depositDetailList", params, agentInvitedUserDepositsRate, &resp)
}

// GetAgentAPITransactionCommission returns a page of the agent's commission on the API trades of users it did not
// invite, from Query API transaction commission (non-invitation relationship)
func (e *Exchange) GetAgentAPITransactionCommission(ctx context.Context, arg *AgentAPITransactionCommissionRequest) (*AgentAPITransactionCommissionResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CommissionBusinessType == 0 {
		return nil, errAgentCommissionBusinessTypeRequired
	}
	params := url.Values{}
	if err := agentRequiredTimeRange(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if err := agentPageParams(params, arg.PageIndex, arg.PageSize); err != nil {
		return nil, err
	}
	params.Set("commissionBizType", strconv.FormatUint(uint64(arg.CommissionBusinessType), 10))
	if arg.UID > 0 {
		params.Set("uid", strconv.FormatUint(arg.UID, 10))
	}
	var resp *AgentAPITransactionCommissionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/agent/v1/reward/third/commissionDataList", params, agentAPICommissionRate, &resp)
}

// GetAgentPartnerInformation returns a page of the agent's partners and their channel activity, from Query partner
// information
func (e *Exchange) GetAgentPartnerInformation(ctx context.Context, arg *AgentPartnerInformationRequest) (*AgentPartnerInformationResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRange(params, "startTime", "endTime", arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.UID > 0 {
		params.Set("uid", strconv.FormatUint(arg.UID, 10))
	}
	if arg.PageIndex > 0 {
		params.Set("pageIndex", strconv.FormatUint(arg.PageIndex, 10))
	}
	if arg.PageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp *AgentPartnerInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/agent/v1/asset/partnerData", params, agentPartnerInformationRate, &resp)
}

// GetAgentReferralCodeCommission returns the invitees and commission of the agent's invitation codes, from Invitation
// code data
func (e *Exchange) GetAgentReferralCodeCommission(ctx context.Context, arg *AgentReferralCodeCommissionRequest) (*AgentReferralCodeCommissionResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRange(params, "startTime", "endTime", arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	params.Set("directInvitation", strconv.FormatBool(arg.DirectInvitation))
	if arg.ReferralCode != "" {
		params.Set("referralCode", arg.ReferralCode)
	}
	if arg.PageIndex > 0 {
		params.Set("pageIndex", strconv.FormatUint(arg.PageIndex, 10))
	}
	if arg.PageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp *AgentReferralCodeCommissionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/agent/v1/commissionDataList/referralCode", params, agentReferralCodeCommissionRate, &resp)
}

// GetAgentSuperiorVerification returns whether the account and the superior with the UID given are in an invitation
// relationship, from Superior verification
func (e *Exchange) GetAgentSuperiorVerification(ctx context.Context, superiorUID uint64) (*AgentSuperiorVerificationResponse, error) {
	if superiorUID == 0 {
		return nil, errUIDRequired
	}
	params := url.Values{}
	params.Set("uid", strconv.FormatUint(superiorUID, 10))
	var resp *AgentSuperiorVerificationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/agent/v1/account/superiorCheck", params, agentSuperiorCheckRate, &resp)
}

// GetAgentChannelCommissionDetails returns a page of the agent's commission from its subordinate channel users, from
// Channel Commission Details. The agent is the account of the API key
func (e *Exchange) GetAgentChannelCommissionDetails(ctx context.Context, arg *AgentChannelCommissionDetailsRequest) (*AgentChannelCommissionDetailsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := agentRequiredTimeRange(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if err := agentPageParams(params, arg.PageIndex, arg.PageSize); err != nil {
		return nil, err
	}
	if arg.UID > 0 {
		params.Set("uid", strconv.FormatUint(arg.UID, 10))
	}
	if arg.BusinessType != "" {
		params.Set("businessType", string(arg.BusinessType))
	}
	if arg.AssetType > 0 {
		params.Set("assetType", strconv.FormatUint(uint64(arg.AssetType), 10))
	}
	var resp *AgentChannelCommissionDetailsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/openApi/agent/v1/reward/channelCommissionList", params, agentChannelCommissionRate, &resp)
}

// agentPageParams sets the page parameters of an agent query that requires them
func agentPageParams(params url.Values, pageIndex, pageSize uint64) error {
	if pageIndex == 0 {
		return errPageIndexRequired
	}
	if pageSize == 0 {
		return errPageSizeRequired
	}
	params.Set("pageIndex", strconv.FormatUint(pageIndex, 10))
	params.Set("pageSize", strconv.FormatUint(pageSize, 10))
	return nil
}

// agentRequiredTimeRange sets the time range of an agent query that requires both bounds
func agentRequiredTimeRange(params url.Values, start, end time.Time) error {
	if start.IsZero() {
		return fmt.Errorf("%w: start time", common.ErrDateUnset)
	}
	if end.IsZero() {
		return fmt.Errorf("%w: end time", common.ErrDateUnset)
	}
	return setTimeRange(params, "startTime", "endTime", start, end)
}
