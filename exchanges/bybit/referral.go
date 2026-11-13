package bybit

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var errMalformedReferralCampaign = errors.New("malformed referral campaign")

// GetReferralCommissionRecords returns a page of the commissions the account earned as an inviter (Get Referral
// Commission Records)
func (e *Exchange) GetReferralCommissionRecords(ctx context.Context, req *ReferralCommissionRecordsRequest) (*ReferralCommissionRecordsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body, err := referralRecordsBody(req.UserIDList, req.StartTime, req.EndTime, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	if req.ProductLine != 0 {
		body["productLine"] = req.ProductLine
	}
	var resp *ReferralCommissionRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/referral/commission/records", nil, body, &resp, getReferralCommissionRecordsEPL)
}

// GetReferralCampaignRewards returns the site's referral rules and the tasks and rewards its active campaigns
// configure, but not the account's progress or rewards (Get Referral Campaign Rewards)
func (e *Exchange) GetReferralCampaignRewards(ctx context.Context) (*ReferralCampaignRewardsResponse, error) {
	var resp *ReferralCampaignRewardsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/referral/campaign/rewards", nil, nil, &resp, getReferralCampaignRewardsEPL)
}

// GetReferralUserIdentityProfile returns whether the account is an affiliate or an M1 inviter, which referral
// relationships it has and, for an M1 inviter, its commission figures (Get Referral User Identity Profile)
func (e *Exchange) GetReferralUserIdentityProfile(ctx context.Context) (*ReferralUserIdentityProfileResponse, error) {
	var resp *ReferralUserIdentityProfileResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/referral/user/identity-profile", nil, nil, &resp, getReferralUserIdentityProfileEPL)
}

// GetReferralUserInvitations returns a page of the users the account invited (Get Referral User Invitations)
func (e *Exchange) GetReferralUserInvitations(ctx context.Context, req *ReferralUserInvitationsRequest) (*ReferralUserInvitationsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body, err := referralRecordsBody(req.UserIDList, req.StartTime, req.EndTime, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	if req.FriendStatus != 0 {
		body["friendStatus"] = req.FriendStatus
	}
	var resp *ReferralUserInvitationsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/referral/user/invitations", nil, body, &resp, getReferralUserInvitationsEPL)
}

// referralRecordsBody returns the body of the filters Get Referral Commission Records and Get Referral User
// Invitations share. Bybit reads the times as UTC date times and passes on one it cannot parse as unset, so they are
// sent in its layout
func referralRecordsBody(userIDList []uint64, startTime, endTime time.Time, page, pageSize uint64) (map[string]any, error) {
	if !startTime.IsZero() && !endTime.IsZero() && startTime.After(endTime) {
		return nil, common.ErrStartAfterEnd
	}
	body := make(map[string]any)
	if len(userIDList) != 0 {
		users := make([]map[string]any, len(userIDList))
		for i, userID := range userIDList {
			users[i] = map[string]any{"userId": userID}
		}
		body["userIdList"] = users
	}
	if !startTime.IsZero() {
		body["startTime"] = startTime.UTC().Format(time.DateTime)
	}
	if !endTime.IsZero() {
		body["endTime"] = endTime.UTC().Format(time.DateTime)
	}
	if page != 0 {
		body["page"] = page
	}
	if pageSize != 0 {
		body["pageSize"] = pageSize
	}
	return body, nil
}
