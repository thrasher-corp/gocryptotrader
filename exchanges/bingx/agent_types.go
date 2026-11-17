package bingx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AgentInvitedUsersRequest holds the parameters of Query Invited Users. A time window spans at most 30 days, and
// without one every invited user is listed
type AgentInvitedUsersRequest struct {
	StartTime time.Time
	EndTime   time.Time
	// LastUID is the UID of the last user on the previous page, which BingX requires once a query passes 10,000 users
	LastUID   uint64
	PageIndex uint64
	// PageSize is at most 200
	PageSize uint64
}

// AgentInvitedUsersResponse holds a page of an agent's invited users
type AgentInvitedUsersResponse struct {
	List            []AgentInvitedUser `json:"list"`
	Total           uint64             `json:"total"`
	CurrentAgentUID NumericID          `json:"currentAgentUid"`
}

// AgentInvitedUser is a user an agent invited
type AgentInvitedUser struct {
	UID           NumericID `json:"uid"`
	OwnInviteCode string    `json:"ownInviteCode"`
	// SuperiorsUID is the inviter's UID
	SuperiorsUID uint64 `json:"superiorsUid"`
	// InvitationCode is the inviter's invitation code
	InvitationCode   string     `json:"InvitationCode"`
	RegisterDateTime types.Time `json:"registerDateTime"`
	// DirectInvitation is false for a user invited through another invitee
	DirectInvitation bool          `json:"directInvitation"`
	KYCResult        types.Boolean `json:"kycResult"`
	Deposit          bool          `json:"deposit"`
	// BalanceVolume is the user's net assets in USDT
	BalanceVolume types.Number `json:"balanceVolume"`
	// Trade is whether the user traded, leaving out trades with trial or bonus funds
	Trade     bool   `json:"trade"`
	UserLevel uint64 `json:"userLevel"`
	// CommissionRatio is a percentage
	CommissionRatio float64      `json:"commissionRatio"`
	CurrentBenefit  AgentBenefit `json:"currentBenefit"`
	// BenefitRatio is the fee reduction percentage
	BenefitRatio      float64    `json:"benefitRatio"`
	BenefitExpiration types.Time `json:"benefitExpiration"`
}

// AgentDailyCommissionDetailsRequest holds the parameters of Daily commission details. The time window is required, may
// span at most seven days and must lie within the last 365 days
type AgentDailyCommissionDetailsRequest struct {
	UID            uint64
	InvitationCode string
	// BusinessType is every business type when empty
	BusinessType AgentBusinessType
	// DistanceType is every invitation type when zero
	DistanceType AgentInvitationType
	StartTime    time.Time
	EndTime      time.Time
	PageIndex    uint64
	// PageSize is at most 100
	PageSize uint64
}

// AgentDailyCommissionDetailsResponse holds a page of an agent's daily commission from invited users
type AgentDailyCommissionDetailsResponse struct {
	List            []AgentDailyCommission `json:"list"`
	Total           uint64                 `json:"total"`
	CurrentAgentUID NumericID              `json:"currentAgentUid"`
}

// AgentDailyCommission is an agent's commission from an invited user on a day
type AgentDailyCommission struct {
	UID uint64 `json:"uid"`
	// CommissionTime is the day the commission is for
	CommissionTime types.Time `json:"commissionTime"`
	// TradingVolume is in USDT
	TradingVolume       types.Number `json:"tradingVolume"`
	Commission          types.Number `json:"commission"`
	ExpectedTradingFees types.Number `json:"expectedTradingFees"`
	OffsetTradingFees   types.Number `json:"offsetTradingFees"`
	// CollectedTradingFees is ExpectedTradingFees less OffsetTradingFees
	CollectedTradingFees types.Number `json:"collectedTradingFees"`
	RebateRatio          types.Number `json:"rebateRatio"`
	// TradingRebate is the fee rebate paid to the user, who pays CollectedTradingFees less TradingRebate
	TradingRebate types.Number `json:"tradingRebate"`
}

// AgentUserInformationResponse holds an invited user's relationship to the agent and activity
type AgentUserInformationResponse struct {
	UID          uint64        `json:"uid"`
	ExistInviter types.Boolean `json:"existInviter"`
	// InviteResult is whether the user is in an invitation relationship with the agent
	InviteResult     bool       `json:"inviteResult"`
	RegisterDateTime types.Time `json:"registerDateTime"`
	// DirectInvitation is false for a user invited through another invitee
	DirectInvitation bool `json:"directInvitation"`
	// SuperiorsUID is the inviter's UID
	SuperiorsUID uint64        `json:"superiorsUid"`
	Deposit      bool          `json:"deposit"`
	KYCResult    types.Boolean `json:"kycResult"`
	// BalanceVolume is the user's net assets in USDT
	BalanceVolume types.Number `json:"balanceVolume"`
	// Trade is whether the user traded, leaving out trades with trial or bonus funds
	Trade     bool   `json:"trade"`
	UserLevel uint64 `json:"userLevel"`
	// CommissionRatio is a percentage
	CommissionRatio float64      `json:"commissionRatio"`
	CurrentBenefit  AgentBenefit `json:"currentBenefit"`
	// BenefitRatio is the fee reduction percentage
	BenefitRatio      float64    `json:"benefitRatio"`
	BenefitExpiration types.Time `json:"benefitExpiration"`
}

// AgentInvitedUserDepositDetailsRequest holds the parameters of Query the deposit details of invited users. Every
// parameter is required, and only the last 90 days are kept
type AgentInvitedUserDepositDetailsRequest struct {
	// UID is the inviting user's UID, which must be a parent account's
	UID          uint64
	BusinessType AgentDepositBusinessType
	StartTime    time.Time
	EndTime      time.Time
	PageIndex    uint64
	// PageSize is at most 100
	PageSize uint64
}

// AgentInvitedUserDepositDetailsResponse holds a page of invited users' deposits
type AgentInvitedUserDepositDetailsResponse struct {
	List  []AgentInvitedUserDeposit `json:"list"`
	Total uint64                    `json:"total"`
}

// AgentInvitedUserDeposit is an invited user's deposit
type AgentInvitedUserDeposit struct {
	UID uint64 `json:"uid"`
	// InviteResult is whether the user is in an invitation relationship with the agent
	InviteResult bool `json:"inviteResult"`
	// DirectInvitation is false for a user invited through another invitee
	DirectInvitation bool                     `json:"directInvitation"`
	BusinessType     AgentDepositBusinessType `json:"bizType"`
	BusinessTime     types.Time               `json:"bizTime"`
	// AssetType is the operation type, which AssetTypeName names
	AssetType            uint64        `json:"assetType"`
	AssetTypeName        string        `json:"assetTypeName"`
	CurrencyName         currency.Code `json:"currencyName"`
	CurrencyAmountVolume types.Number  `json:"currencyAmountVolume"`
}

// AgentAPITransactionCommissionRequest holds the parameters of Query API transaction commission (non-invitation
// relationship). The time window is required, and only data from 1 December 2023 is kept
type AgentAPITransactionCommissionRequest struct {
	// UID filters the commission to a trading user's when set
	UID                    uint64
	CommissionBusinessType AgentCommissionBusinessType
	StartTime              time.Time
	EndTime                time.Time
	PageIndex              uint64
	// PageSize is at most 100
	PageSize uint64
}

// AgentAPITransactionCommissionResponse holds a page of an agent's commission on API trades of users it did not invite
type AgentAPITransactionCommissionResponse struct {
	List  []AgentAPITransactionCommission `json:"list"`
	Total uint64                          `json:"total"`
}

// AgentAPITransactionCommission is an agent's commission on a user's API trades
type AgentAPITransactionCommission struct {
	UID            uint64     `json:"uid"`
	CommissionTime types.Time `json:"commissionTime"`
	// TradeVolume is in USDT
	TradeVolume types.Number `json:"tradeVolume"`
	// CommissionVolume is in USDT
	CommissionVolume       types.Number                `json:"commissionVolume"`
	CommissionBusinessType AgentCommissionBusinessType `json:"commissionBizType"`
}

// AgentPartnerInformationRequest holds the parameters of Query partner information. Only the last three months are
// kept, and without a time window the current month is returned
type AgentPartnerInformationRequest struct {
	UID       uint64
	StartTime time.Time
	EndTime   time.Time
	PageIndex uint64
	// PageSize is at most 200
	PageSize uint64
}

// AgentPartnerInformationResponse holds a page of an agent's partners
type AgentPartnerInformationResponse struct {
	List  []AgentPartner `json:"list"`
	Total uint64         `json:"total"`
}

// AgentPartner is an agent's partner and the partner's channel activity over the queried period
type AgentPartner struct {
	UID uint64 `json:"uid"`
	// Email is masked
	Email string `json:"email"`
	// Phone is masked
	Phone        string              `json:"phone"`
	ReferralType AgentInvitationType `json:"referralType"`
	Remarks      string              `json:"remarks"`
	ReferrerUID  uint64              `json:"referrerUid"`
	Language     string              `json:"language"`
	NewReferees  types.Number        `json:"newReferees"`
	// FirstTrade is the number of users who traded for the first time
	FirstTrade     types.Number `json:"firstTrade"`
	BranchDeposits types.Number `json:"branchDeposits"`
	// BranchTrading is the number of trades in the partner's channel
	BranchTrading       types.Number `json:"branchTrading"`
	BranchTradingVolume types.Number `json:"branchTradingVol"`
	Level               types.Number `json:"level"`
	CommissionRatio     types.Number `json:"commissionRatio"`
}

// AgentReferralCodeCommissionRequest holds the parameters of Invitation code data. A time window spans at most 92 days
// within the last year, and without one the last seven days are returned
type AgentReferralCodeCommissionRequest struct {
	// ReferralCode is every code when empty
	ReferralCode string
	// DirectInvitation selects direct invitations when true and indirect invitations when false
	DirectInvitation bool
	StartTime        time.Time
	EndTime          time.Time
	PageIndex        uint64
	// PageSize is at most 200
	PageSize uint64
}

// AgentReferralCodeCommissionResponse holds the invitees and commission of an agent's invitation codes
type AgentReferralCodeCommissionResponse struct {
	List []AgentReferralCodeCommission `json:"list"`
}

// AgentReferralCodeCommission is the invitees and commission of an invitation code
type AgentReferralCodeCommission struct {
	ReferralCode string `json:"referralCode"`
	// CommissionRatio is a percentage
	CommissionRatio types.Number `json:"commissionRatio"`
	// FriendEarn is the invitees' cashback percentage
	FriendEarn types.Number `json:"friendEarn"`
	// SwapFeeDiscount is the invitees' perpetual futures fee discount percentage
	SwapFeeDiscount    types.Number `json:"swapFeeDiscount"`
	ReferralNumber     types.Number `json:"referralNum"`
	Deposited          types.Number `json:"deposited"`
	Traded             types.Number `json:"traded"`
	TradingVolume      types.Number `json:"tradingVolume"`
	Fee                types.Number `json:"fee"`
	OffsetTradingFees  types.Number `json:"offsetTradingFees"`
	PayableTradingFees types.Number `json:"payableTradingFees"`
	Commission         types.Number `json:"commission"`
}

// AgentSuperiorVerificationResponse holds whether the account and a superior are in an invitation relationship
type AgentSuperiorVerificationResponse struct {
	// InviteResult is whether the user is in an invitation relationship with the agent
	InviteResult bool `json:"inviteResult"`
	// DirectInvitation is false for a user invited through another invitee
	DirectInvitation bool `json:"directInvitation"`
}

// AgentChannelCommissionDetailsRequest holds the parameters of Channel Commission Details. The time window is required
type AgentChannelCommissionDetailsRequest struct {
	// UID filters the records to a subordinate channel user's when set
	UID uint64
	// BusinessType is every business type when empty
	BusinessType AgentBusinessType
	// AssetType is every asset type when zero
	AssetType AgentAssetType
	StartTime time.Time
	EndTime   time.Time
	PageIndex uint64
	// PageSize is at most 100
	PageSize uint64
}

// AgentChannelCommissionDetailsResponse holds a page of the agent's commission from its subordinate channel users
type AgentChannelCommissionDetailsResponse struct {
	List            []AgentChannelCommission `json:"list"`
	Total           uint64                   `json:"total"`
	CurrentAgentUID NumericID                `json:"currentAgentUid"`
}

// AgentChannelCommission is the agent's commission from a subordinate channel user
type AgentChannelCommission struct {
	UID                 uint64       `json:"uid"`
	TradingVolume       types.Number `json:"tradingVolume"`
	ExpectedTradingFees types.Number `json:"expectedTradingFees"`
	OffsetTradingFees   types.Number `json:"offsetTradingFees"`
	// CollectedTradingFees is ExpectedTradingFees less OffsetTradingFees
	CollectedTradingFees  types.Number `json:"collectedTradingFees"`
	Commission            types.Number `json:"commission"`
	PartnerCommissionRate types.Number `json:"partnerCommissionRate"`
}
