package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/types"
)

// AffiliatePerformanceSummaryRequest holds the statistics window of an affiliate's performance summary
type AffiliatePerformanceSummaryRequest struct {
	PeriodType string
	// Begin and End bound a custom period, inclusively
	Begin time.Time
	End   time.Time
}

// AffiliatePerformanceSummaryResponse holds an affiliate's aggregated performance
type AffiliatePerformanceSummaryResponse struct {
	UpdateTime    types.Time                    `json:"uTime"`
	InviteeCount  types.Number                  `json:"inviteeCnt"`
	DepositAmount types.Number                  `json:"depAmt"`
	Details       []AffiliateCommissionCategory `json:"details"`
}

// AffiliateCommissionCategory holds an affiliate's performance in a commission category
type AffiliateCommissionCategory struct {
	CommissionCategory string       `json:"commissionCategory"`
	FirstTraderCount   types.Number `json:"firstTraderCnt"`
	TraderCount        types.Number `json:"traderCnt"`
	Volume             types.Number `json:"vol"`
	Commission         types.Number `json:"commission"`
}

// InviteeDetailRequest identifies an invitee and the period of its trading volume
type InviteeDetailRequest struct {
	UID string
	// PeriodType sets the window of PeriodVolume, which OKX omits without it
	PeriodType string
}

// InviteeDetailResponse holds an invitee's details
type InviteeDetailResponse struct {
	InviteeLevel      types.Number `json:"inviteeLevel"`
	JoinTime          types.Time   `json:"joinTime"`
	InviteeRebateRate types.Number `json:"inviteeRebateRate"`
	TotalCommission   types.Number `json:"totalCommission"`
	FirstTradeTime    types.Time   `json:"firstTradeTime"`
	Level             string       `json:"level"`
	DepositAmount     types.Number `json:"depAmt"`
	WithdrawalAmount  types.Number `json:"wdAmt"`
	MonthVolume       types.Number `json:"volMonth"`
	TotalVolume       types.Number `json:"totalVol"`
	PeriodVolume      types.Number `json:"volPeriod"`
	AccumulatedFee    types.Number `json:"accFee"`
	KYCTime           types.Time   `json:"kycTime"`
	Region            string       `json:"region"`
	AffiliateCode     string       `json:"affiliateCode"`
}

// InviteeListRequest holds the filters of an affiliate's invitee list
type InviteeListRequest struct {
	Page       uint64
	Limit      uint64
	PeriodType string
	// Begin and End bound a custom period, inclusively
	Begin              time.Time
	End                time.Time
	Keyword            string
	CommissionCategory string
	OrderBy            string
	OrderDirection     string
	KYCStatus          string
	SubAffiliateUID    string
	// UIDs takes up to 100 invitees
	UIDs []string
	// JoinTimeBegin and JoinTimeEnd bound the join time, inclusively, and are sent together
	JoinTimeBegin time.Time
	JoinTimeEnd   time.Time
}

// InviteeListResponse holds a page of an affiliate's invitees
type InviteeListResponse struct {
	TotalPage types.Number
	Invitees  []InviteeResponse
}

// InviteeResponse holds an invitee with its trading statistics
type InviteeResponse struct {
	UID             string       `json:"uid"`
	Country         string       `json:"country"`
	JoinTime        types.Time   `json:"joinTime"`
	FirstTradeTime  types.Time   `json:"firstTradeTime"`
	ChannelName     string       `json:"channelName"`
	RebateRate      types.Number `json:"rebateRate"`
	FeeTierRank     types.Number `json:"feeTierRank"`
	KYCStatus       string       `json:"kycStatus"`
	KYCTime         types.Time   `json:"kycTime"`
	DepositAmount   types.Number `json:"depAmt"`
	TotalVolume     types.Number `json:"totalVol"`
	TotalFee        types.Number `json:"totalFee"`
	TotalCommission types.Number `json:"totalCommission"`
	IsCompliant     bool         `json:"isCompliant"`
}

// AffiliateLinkListRequest holds the filters of an affiliate's invite links
type AffiliateLinkListRequest struct {
	Page       uint64
	Limit      uint64
	LinkType   string
	LinkStatus string
}

// AffiliateLinkListResponse holds a page of an affiliate's invite links
type AffiliateLinkListResponse struct {
	TotalPage types.Number
	Links     []AffiliateLinkResponse
}

// AffiliateLinkResponse holds an affiliate invite link with its commission rates and statistics
type AffiliateLinkResponse struct {
	ChannelID               string       `json:"channelId"`
	ChannelName             string       `json:"channelName"`
	JoinLink                string       `json:"joinLink"`
	LinkType                string       `json:"linkType"`
	InviterCommissionRate   types.Number `json:"inviterCommissionRate"`
	CoInviterCommissionRate types.Number `json:"coInviterCommissionRate"`
	InviteeDiscountRate     types.Number `json:"inviteeDiscountRate"`
	InviteeCount            types.Number `json:"inviteeCnt"`
	TraderCount             types.Number `json:"traderCnt"`
	TotalCommission         types.Number `json:"totalCommission"`
	Commission24Hour        types.Number `json:"commission24h"`
	CreationTime            types.Time   `json:"cTime"`
	IsDefault               bool         `json:"isDefault"`
	LinkStatus              string       `json:"linkStatus"`
}

// CoInviterLinkListRequest holds the filters of the links on which the account is the co-inviter
type CoInviterLinkListRequest struct {
	Page       uint64
	Limit      uint64
	LinkStatus string
}

// CoInviterLinkListResponse holds a page of the links on which the account is the co-inviter
type CoInviterLinkListResponse struct {
	TotalPage types.Number
	Links     []CoInviterLinkResponse
}

// CoInviterLinkResponse holds a link on which the account is the co-inviter
type CoInviterLinkResponse struct {
	ChannelID               string       `json:"channelId"`
	ChannelName             string       `json:"channelName"`
	JoinLink                string       `json:"joinLink"`
	InviterCommissionRate   types.Number `json:"inviterCommissionRate"`
	CoInviterCommissionRate types.Number `json:"coInviterCommissionRate"`
	InviteeDiscountRate     types.Number `json:"inviteeDiscountRate"`
	// PartnerUserName is partially masked
	PartnerUserName string `json:"parUserName"`
	// CoInviterUserName is always fully masked
	CoInviterUserName       string       `json:"coUserName"`
	IsCompliant             bool         `json:"isCompliant"`
	Note                    string       `json:"note"`
	IsDefault               bool         `json:"isDefault"`
	TotalCommission         types.Number `json:"totalCommission"`
	Commission24Hour        types.Number `json:"commission24h"`
	InviteeCount            types.Number `json:"inviteeCnt"`
	TraderCount             types.Number `json:"traderCnt"`
	ClickCount              types.Number `json:"clickCnt"`
	TotalFee                types.Number `json:"totalFee"`
	CreationTime            types.Time   `json:"cTime"`
	ChannelAssessmentStatus string       `json:"channelAssessmentStatus"`
	InviterChannelStatus    string       `json:"inviterChannelStatus"`
	CoInviterChannelStatus  string       `json:"coInviterChannelStatus"`
	LinkStatus              string       `json:"linkStatus"`
}

// SubAffiliateListRequest holds the filters of an affiliate's sub-affiliates
type SubAffiliateListRequest struct {
	Page               uint64
	Limit              uint64
	Keyword            string
	CommissionCategory string
	OrderBy            string
	OrderDirection     string
}

// SubAffiliateListResponse holds a page of an affiliate's sub-affiliates
type SubAffiliateListResponse struct {
	TotalPage     types.Number
	SubAffiliates []SubAffiliateResponse
}

// SubAffiliateResponse holds a sub-affiliate with the statistics of its invitees
type SubAffiliateResponse struct {
	SubAffiliateUID string     `json:"subAffiliateUid"`
	Country         string     `json:"country"`
	JoinTime        types.Time `json:"joinTime"`
	// SubAffiliateLevel is 2 for a direct sub-affiliate and 3 for one a level below
	SubAffiliateLevel types.Number `json:"subAffiliateLevel"`
	CommissionRate    types.Number `json:"commissionRate"`
	IsCompliant       bool         `json:"isCompliant"`
	InviteeCount      types.Number `json:"inviteeCnt"`
	TraderCount       types.Number `json:"traderCnt"`
	DepositAmount     types.Number `json:"depAmt"`
	TotalVolume       types.Number `json:"totalVol"`
	TotalFee          types.Number `json:"totalFee"`
	TotalCommission   types.Number `json:"totalCommission"`
}

// AffiliateRebateInformationResponse holds whether a user was invited by the affiliate and why any rebate is withheld
type AffiliateRebateInformationResponse struct {
	Result bool   `json:"result"`
	Type   string `json:"type"`
}
