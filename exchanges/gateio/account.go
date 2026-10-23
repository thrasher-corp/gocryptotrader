package gateio

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

// CreateSelfTradePreventionUserGroup creates an STP user group. Only the main account can create groups
func (e *Exchange) CreateSelfTradePreventionUserGroup(ctx context.Context, name string) (*STPUserGroupResponse, error) {
	if name == "" {
		return nil, errSTPGroupNameRequired
	}
	var resp *STPUserGroupResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountCreateSTPGroupEPL, http.MethodPost, "account/stp_groups", nil, map[string]string{"name": name}, &resp)
}

// GetUserSelfTradePreventionGroups retrieves the STP user groups created by the main account, fuzzy matched by name
// when it is set
func (e *Exchange) GetUserSelfTradePreventionGroups(ctx context.Context, name string) ([]*STPUserGroupResponse, error) {
	params := url.Values{}
	if name != "" {
		params.Set("name", name)
	}
	var resp []*STPUserGroupResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountListSTPGroupsEPL, http.MethodGet, "account/stp_groups", params, nil, &resp)
}

// GetUsersInSTPUserGroup retrieves the users in an STP user group created by the main account
func (e *Exchange) GetUsersInSTPUserGroup(ctx context.Context, stpID uint64) ([]*STPUserGroupMemberResponse, error) {
	if stpID == 0 {
		return nil, errSTPGroupIDRequired
	}
	var resp []*STPUserGroupMemberResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountListSTPGroupUsersEPL, http.MethodGet, "account/stp_groups/"+strconv.FormatUint(stpID, 10)+"/users", nil, nil, &resp)
}

// AddUsersToSTPUserGroup adds accounts under the main account to an STP user group and returns the group's users
func (e *Exchange) AddUsersToSTPUserGroup(ctx context.Context, stpID uint64, userIDs []uint64) ([]*STPUserGroupMemberResponse, error) {
	if stpID == 0 {
		return nil, errSTPGroupIDRequired
	}
	if len(userIDs) == 0 || slices.Contains(userIDs, 0) {
		return nil, errUserIDRequired
	}
	var resp []*STPUserGroupMemberResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountAddSTPGroupUsersEPL, http.MethodPost, "account/stp_groups/"+strconv.FormatUint(stpID, 10)+"/users", nil, userIDs, &resp)
}

// DeleteUserFromSTPUserGroup removes accounts under the main account from an STP user group and returns the group's
// remaining users
func (e *Exchange) DeleteUserFromSTPUserGroup(ctx context.Context, stpID uint64, userIDs []uint64) ([]*STPUserGroupMemberResponse, error) {
	if stpID == 0 {
		return nil, errSTPGroupIDRequired
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
	var resp []*STPUserGroupMemberResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountDeleteSTPGroupUsersEPL, http.MethodDelete, "account/stp_groups/"+strconv.FormatUint(stpID, 10)+"/users", params, nil, &resp)
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
