package bybit

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// ReferralCommissionRecordsRequest holds the parameters of GetReferralCommissionRecords; every field is optional
type ReferralCommissionRecordsRequest struct {
	// UserIDList limits the records to these invitees
	UserIDList []uint64
	// StartTime and EndTime are sent in UTC, to the second; Bybit applies no default window
	StartTime time.Time
	EndTime   time.Time
	// Page is one-based; 0 is taken as 1
	Page uint64
	// PageSize is at most 200; 0, or more than 200, is taken as 10
	PageSize uint64
	// ProductLine is 1 spot, 2 contracts, 3 options, 4 USDC perpetual, 6 TradFi or 9 Alpha
	ProductLine uint64
}

// ReferralCommissionRecordsResponse holds a page of referral commission records. Total and the payment times decode
// quoted or bare, since the docs leave unverified how the gateway sends int64 values
type ReferralCommissionRecordsResponse struct {
	Records []ReferralCommissionRecord `json:"records"`
	Total   types.Number               `json:"total"`
}

// ReferralCommissionRecord holds a commission the account earned on an invitee's trading
type ReferralCommissionRecord struct {
	// UserID is the invitee's masked ID: its first character, *** and its last character
	UserID      string       `json:"userId"`
	PaymentTime types.Time   `json:"paymentTime"`
	Rate        types.Number `json:"rate"`
	// BusinessUnit is the product line: 1 spot, 2 contracts, 3 options, 4 USDC perpetual, 6 TradFi, 9 Alpha, or 0 when
	// Bybit does not recognise it
	BusinessUnit           uint64        `json:"businessUnit"`
	CommissionOriginCoin   currency.Code `json:"commissionOriginCoin"`
	CommissionOriginAmount types.Number  `json:"commissionOriginAmount"`
	CommissionUSDTAmount   types.Number  `json:"commissionUsdtAmount"`
	// CommissionStatus is 1 awaiting credit, 2 settled, or 0 when Bybit does not recognise it
	CommissionStatus uint64 `json:"commissionStatus"`
	// IsBoostM2 reports whether the invitee is a Boost user; Bybit may leave it out
	IsBoostM2 bool `json:"isBoostM2"`
}

// ReferralCampaignRewardsResponse holds a site's referral rules and the tasks and rewards of its active campaigns
type ReferralCampaignRewardsResponse struct {
	// RebateRule is the site's configuration, not the tier the account has reached
	RebateRule ReferralRebateRule `json:"rebateRule"`
	// Tasks holds the active campaigns: base first, then fiat, then earn
	Tasks []ReferralCampaign `json:"tasks"`
}

// ReferralRebateRule holds a site's referral commission rules. Its integers decode quoted or bare, since the docs leave
// unverified how the gateway sends int64 values
type ReferralRebateRule struct {
	// DepositAmount and TradeAmount are thresholds whose currency and precision Bybit does not state
	DepositAmount    types.Number              `json:"depositAmount"`
	TradeAmount      types.Number              `json:"tradeAmount"`
	RebateRatioRules []ReferralRebateRatioRule `json:"rebateRatioRules"`
	// RebateDurationDays is how many days a referral earns commission for
	RebateDurationDays types.Number `json:"rebateDurationDays"`
	// Bybit does not populate TradFiDepositAmount and TradFiTradeAmount, so 0 does not mean there is no threshold
	TradFiDepositAmount types.Number `json:"tradefiDepositAmount"`
	TradFiTradeAmount   types.Number `json:"tradefiTradeAmount"`
}

// ReferralRebateRatioRule holds a tier of referral commission
type ReferralRebateRatioRule struct {
	// RebateRatio is the commission rate, from 0 to 100
	RebateRatio types.Number `json:"rebateRatio"`
	// ReferralCount is the number of qualified invitees the tier needs
	ReferralCount        types.Number `json:"referralCount"`
	HasTradeCondition    bool         `json:"hasTradeCondition"`
	TradeAmountCondition types.Number `json:"tradeAmountCondition"`
}

// ReferralCampaign holds a campaign's configured tasks, which Bybit sends as a JSON-encoded string
type ReferralCampaign struct {
	// SchemaVersion is 2
	SchemaVersion uint64 `json:"schemaVersion"`
	// Source is base, fiat or earn
	Source string `json:"source"`
	// CampaignID identifies the campaign's configuration, not a reward record
	CampaignID string `json:"campaignId"`
	// Tasks holds the configured tasks and their one-hop M1 and M2 related tasks; it is empty when Bybit cannot assemble
	// them
	Tasks []ReferralCampaignTask `json:"tasks"`
}

// UnmarshalJSON decodes a campaign from the JSON-encoded string Bybit sends it as, and leaves it unchanged for null, as
// encoding/json does
func (c *ReferralCampaign) UnmarshalJSON(data []byte) error {
	var encoded *string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return fmt.Errorf("%w: %w", errMalformedReferralCampaign, err)
	}
	if encoded == nil {
		return nil
	}
	type alias ReferralCampaign
	if err := json.Unmarshal([]byte(*encoded), (*alias)(c)); err != nil {
		return fmt.Errorf("%w: %w", errMalformedReferralCampaign, err)
	}
	return nil
}

// ReferralCampaignTask holds a campaign task and the rewards for completing it
type ReferralCampaignTask struct {
	ID uint64 `json:"id"`
	// AwardType is a value of the awardType enum, such as 1 for a trial funds card or 100 for a reward packet; Bybit
	// assembles no reward details for 95 or 97, and the type does not mean a reward was claimed
	AwardType uint64 `json:"awardType"`
	// RuleTitle, RuleDescription and RuleParam hold the English title, description and first parameter of each valid
	// rule, matched by index; they are nil when the task has no valid rules
	RuleTitle       []string                `json:"ruleTitle"`
	RuleDescription []string                `json:"ruleDescription"`
	RuleParam       []string                `json:"ruleParam"`
	TaskAwards      []ReferralCampaignAward `json:"taskAwards"`
	// M1Awards holds the M1 rewards of the REFERRAL user relation
	M1Awards []ReferralCampaignAward `json:"m1Awards"`
}

// ReferralCampaignAward holds a reward or reward packet of a campaign task
type ReferralCampaignAward struct {
	// ID is the reward's or the reward packet's ID
	ID uint64 `json:"id"`
	// Name is in English; a reward packet's is UP TO or EARN and its value in USD
	Name string `json:"name"`
	// Type is the English reward type, Packet for a reward packet
	Type string `json:"type"`
	// ProductLine is the English product type, empty for a reward packet
	ProductLine string `json:"productLine"`
	// StartTime is zero for a reward packet, and either time is zero when Bybit has none
	StartTime ReferralAwardTime `json:"startTime"`
	EndTime   ReferralAwardTime `json:"endTime"`
}

// ReferralAwardTime is a campaign reward's start or end time, which Bybit sends in RFC 3339 format, or empty when it
// has none
type ReferralAwardTime time.Time

// UnmarshalJSON decodes an RFC 3339 time, or an empty string or null as the zero time
func (t *ReferralAwardTime) UnmarshalJSON(data []byte) error {
	var parsed time.Time
	if string(data) != `""` {
		if err := parsed.UnmarshalJSON(data); err != nil {
			return fmt.Errorf("%w: %s: %w", types.ErrInvalidTimestampFormat, data, err)
		}
	}
	*t = ReferralAwardTime(parsed)
	return nil
}

// MarshalJSON encodes the time in RFC 3339 format, as time.Time does
func (t ReferralAwardTime) MarshalJSON() ([]byte, error) {
	return time.Time(t).MarshalJSON()
}

// Time returns the reward's time, the zero time when it has none
func (t ReferralAwardTime) Time() time.Time {
	return time.Time(t)
}

// ReferralUserIdentityProfileResponse holds the account's referral identity and relationships. Its integers decode
// quoted or bare, since the docs leave unverified how the gateway sends int64 values
type ReferralUserIdentityProfileResponse struct {
	IsAffiliate bool `json:"isAffiliate"`
	// IsM1 reports whether the account is an M1 inviter, which Bybit checks only for an account that is not an affiliate
	IsM1 bool `json:"isM1"`
	// InviterAffiliateID is the affiliate whose direct customer the account is, or 0 when there is none
	InviterAffiliateID types.Number `json:"inviterAffiliateId"`
	// InviterAffiliateUserID is 1 when the account is an affiliate's direct customer and InviterUserID is 1 when it is an
	// invitee, each 0 otherwise; neither is an ID
	InviterAffiliateUserID types.Number `json:"inviterAffiliateUserId"`
	InviterUserID          types.Number `json:"inviterUserId"`
	// CommissionInfo is nil unless the account is an M1 inviter whose commission Bybit could look up
	CommissionInfo *ReferralCommissionInfo `json:"commissionInfo"`
}

// ReferralCommissionInfo holds an M1 inviter's commission figures. Bybit copies them from its commission records, so a
// figure the records leave unset is 0, such as a WithdrawalSwitch of 0
type ReferralCommissionInfo struct {
	TotalCommission     types.Number `json:"totalCommission"`
	YesterdayCommission types.Number `json:"yesterdayCommission"`
	TotalFriends        types.Number `json:"totalFriends"`
	// QualifiedFriends is the number of invitees that earn commission
	QualifiedFriends types.Number `json:"qualifiedFriends"`
	// WalletBalance is the commission wallet's balance
	WalletBalance types.Number `json:"walletBalance"`
	// QualifiedFriendsQuarterTotalTradeVolume is the qualified invitees' trading volume this quarter
	QualifiedFriendsQuarterTotalTradeVolume types.Number `json:"qualifiedFriendsQuarterTotalTradeVolume"`
	CurrentQuarterActiveFriendsCount        types.Number `json:"curQuarterActiveFriendsCount"`
	// HasOldReferral indicates the legacy referral commission programme
	HasOldReferral types.Number `json:"hasOldReferral"`
	// DegradeType is 1 when the commission was downgraded for too few active invitees, 2 for too little invitee trading
	// volume, and 0 otherwise
	DegradeType    types.Number `json:"degradeType"`
	CommissionRate types.Number `json:"commissionRate"`
	// UpgradeUnifiedAccount is 1 upgraded, 2 upgrading, 3 not upgraded or 0 unknown
	UpgradeUnifiedAccount types.Number `json:"upgradeUnifiedAccount"`
	// WithdrawalSwitch is 1 when the commission can be withdrawn and 2 when it cannot
	WithdrawalSwitch                 types.Number `json:"withdrawalSwitch"`
	LastQuarterQualifiedFriendsCount types.Number `json:"lastQuarterQualifiedFriendsCount"`
}

// ReferralUserInvitationsRequest holds the parameters of GetReferralUserInvitations; every field is optional
type ReferralUserInvitationsRequest struct {
	// UserIDList limits the records to these invitees
	UserIDList []uint64
	// StartTime and EndTime are sent in UTC, to the second
	StartTime time.Time
	EndTime   time.Time
	// Page is one-based; 0 is taken as 1
	Page uint64
	// PageSize is at most 200; 0, or more than 200, is taken as 10
	PageSize uint64
	// FriendStatus is a value of the friendStatus enum, such as 1 for signed up or 4 for qualified
	FriendStatus uint64
}

// ReferralUserInvitationsResponse holds a page of the account's invitation records. Total and the sign times decode
// quoted or bare, since the docs leave unverified how the gateway sends int64 values
type ReferralUserInvitationsResponse struct {
	Records []ReferralUserInvitation `json:"records"`
	Total   types.Number             `json:"total"`
}

// ReferralUserInvitation holds a user the account invited
type ReferralUserInvitation struct {
	UserIDMask string `json:"userIdMask"`
	// SignTime is when the invitee registered
	SignTime types.Time `json:"signTime"`
	// UserStatus is a value of the friendStatus enum, such as 1 for signed up or 4 for qualified
	UserStatus uint64 `json:"userStatus"`
	// TotalCommission has at most eight decimal places
	TotalCommission types.Number `json:"totalCommission"`
	// UserID is the invitee's unmasked user ID
	UserID       string `json:"userId"`
	TaskProgress string `json:"taskProgress"`
	// NotShowCommission is true for some new invitees of the EU site and non-Boost invitees of the JPN site
	NotShowCommission bool `json:"notShowCommission"`
	// TotalBaseRewards is empty, as Bybit does not populate it
	TotalBaseRewards types.Number `json:"totalBaseRewards"`
}
