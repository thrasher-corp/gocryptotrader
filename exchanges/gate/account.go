package gate

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetAccountDetails retrieves account details
func (e *Exchange) GetAccountDetails(ctx context.Context) (*AccountDetailsResponse, error) {
	var resp *AccountDetailsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountDetailEPL, http.MethodGet, "account/detail", nil, nil, &resp)
}

// GetAccountMainKeys retrieves every API key of the main account
func (e *Exchange) GetAccountMainKeys(ctx context.Context) ([]*AccountMainKeyResponse, error) {
	var resp []*AccountMainKeyResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountMainKeysEPL, http.MethodGet, "account/main_keys", nil, nil, &resp)
}

// GetUserTransactionRateLimitInfo retrieves user transaction rate limit info. Gate notes the endpoint is not yet open
// for use
func (e *Exchange) GetUserTransactionRateLimitInfo(ctx context.Context) ([]*UserTransactionRateLimitInfoResponse, error) {
	var resp []*UserTransactionRateLimitInfoResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountRateLimitEPL, http.MethodGet, "account/rate_limit", nil, nil, &resp)
}

// CreateSelfTradePreventionUserGroup creates a self-trade prevention user group, which only the main account can do
func (e *Exchange) CreateSelfTradePreventionUserGroup(ctx context.Context, name string) (*SelfTradePreventionUserGroupResponse, error) {
	if name == "" {
		return nil, errSelfTradePreventionGroupNameRequired
	}
	var resp *SelfTradePreventionUserGroupResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountCreateSelfTradePreventionGroupEPL, http.MethodPost, "account/stp_groups", nil, map[string]string{"name": name}, &resp)
}

// GetUserSelfTradePreventionGroups retrieves the self-trade prevention user groups created by the main account, fuzzy
// matched by name when it is set
func (e *Exchange) GetUserSelfTradePreventionGroups(ctx context.Context, name string) ([]*SelfTradePreventionUserGroupResponse, error) {
	params := url.Values{}
	if name != "" {
		params.Set("name", name)
	}
	var resp []*SelfTradePreventionUserGroupResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountListSelfTradePreventionGroupsEPL, http.MethodGet, "account/stp_groups", params, nil, &resp)
}

// GetUsersInSelfTradePreventionUserGroup retrieves the users in a self-trade prevention user group created by the main
// account
func (e *Exchange) GetUsersInSelfTradePreventionUserGroup(ctx context.Context, groupID uint64) ([]*SelfTradePreventionUserGroupMemberResponse, error) {
	if groupID == 0 {
		return nil, errSelfTradePreventionGroupIDRequired
	}
	var resp []*SelfTradePreventionUserGroupMemberResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountListSelfTradePreventionGroupUsersEPL, http.MethodGet, "account/stp_groups/"+strconv.FormatUint(groupID, 10)+"/users", nil, nil, &resp)
}

// AddUsersToSelfTradePreventionUserGroup adds accounts under the main account to a self-trade prevention user group and
// returns the group's users
func (e *Exchange) AddUsersToSelfTradePreventionUserGroup(ctx context.Context, groupID uint64, userIDs []uint64) ([]*SelfTradePreventionUserGroupMemberResponse, error) {
	if groupID == 0 {
		return nil, errSelfTradePreventionGroupIDRequired
	}
	if len(userIDs) == 0 || slices.Contains(userIDs, 0) {
		return nil, errUserIDRequired
	}
	var resp []*SelfTradePreventionUserGroupMemberResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountAddSelfTradePreventionGroupUsersEPL, http.MethodPost, "account/stp_groups/"+strconv.FormatUint(groupID, 10)+"/users", nil, userIDs, &resp)
}

// DeleteUserFromSelfTradePreventionUserGroup removes accounts under the main account from a self-trade prevention user
// group and returns the group's remaining users
func (e *Exchange) DeleteUserFromSelfTradePreventionUserGroup(ctx context.Context, groupID uint64, userIDs []uint64) ([]*SelfTradePreventionUserGroupMemberResponse, error) {
	if groupID == 0 {
		return nil, errSelfTradePreventionGroupIDRequired
	}
	if len(userIDs) == 0 || slices.Contains(userIDs, 0) {
		return nil, errUserIDRequired
	}
	ids := make([]string, len(userIDs))
	for i := range userIDs {
		ids[i] = strconv.FormatUint(userIDs[i], 10)
	}
	params := url.Values{}
	params.Set("user_id", strings.Join(ids, ","))
	var resp []*SelfTradePreventionUserGroupMemberResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountDeleteSelfTradePreventionGroupUsersEPL, http.MethodDelete, "account/stp_groups/"+strconv.FormatUint(groupID, 10)+"/users", params, nil, &resp)
}

// ConfigureGTFeeDeduction enables or disables GT fee deduction for the current account
func (e *Exchange) ConfigureGTFeeDeduction(ctx context.Context, setEnabled bool) error {
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountSetDebitFeeEPL, http.MethodPost, "account/debit_fee", nil, &map[string]bool{"enabled": setEnabled}, nil)
}

// GetGTFeeDeductionConfiguration reports whether GT fee deduction is enabled for the current account
func (e *Exchange) GetGTFeeDeductionConfiguration(ctx context.Context) (bool, error) {
	var resp gtFeeDeductionReply
	return resp.Enabled, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountGetDebitFeeEPL, http.MethodGet, "account/debit_fee", nil, nil, &resp)
}
