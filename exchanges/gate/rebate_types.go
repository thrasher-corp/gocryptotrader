package gate

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// RebateTransactionHistoryRequest holds query parameters for retrieving agency or partner transaction history of recommended users.
type RebateTransactionHistoryRequest struct {
	CurrencyPair currency.Pair
	UserID       uint64
	From         time.Time
	To           time.Time
	Limit        uint64
	Offset       uint64
}

// RebateCommissionHistoryRequest holds query parameters for retrieving agency or partner rebate history of recommended users.
type RebateCommissionHistoryRequest struct {
	Currency currency.Code
	// CommissionType filters the rebate type and is only used by the agency commission history endpoint: 1 - Direct rebate, 2 - Indirect rebate, 3 - Self rebate.
	CommissionType uint64
	UserID         uint64
	From           time.Time
	To             time.Time
	Limit          uint64
	Offset         uint64
}

// RebateBrokerHistoryRequest holds query parameters for retrieving a broker's commission or transaction history of recommended users.
type RebateBrokerHistoryRequest struct {
	UserID uint64
	From   time.Time
	To     time.Time
	Limit  uint64
	Offset uint64
}

// PartnerSubordinateListRequest holds query parameters for retrieving a partner's subordinate list.
type PartnerSubordinateListRequest struct {
	UserID uint64
	Limit  uint64
	Offset uint64
}

// rebateUserInfoReply holds the UID of the user who invited the authenticated user
type rebateUserInfoReply struct {
	InviteUID uint64 `json:"invite_uid"`
}

// RebateTransaction holds a single transaction history record for a recommended user.
type RebateTransaction struct {
	TransactionTime types.Time    `json:"transaction_time"`
	UserID          uint64        `json:"user_id"`
	GroupName       string        `json:"group_name"`
	Fee             types.Number  `json:"fee"`
	FeeAsset        currency.Code `json:"fee_asset"`
	CurrencyPair    currency.Pair `json:"currency_pair"`
	Amount          types.Number  `json:"amount"`
	AmountAsset     currency.Code `json:"amount_asset"`
	Source          string        `json:"source"`
}

// RebateCommission holds a single rebate commission record for a recommended user.
type RebateCommission struct {
	CommissionTime   types.Time    `json:"commission_time"`
	UserID           uint64        `json:"user_id"`
	GroupName        string        `json:"group_name"`
	CommissionAmount types.Number  `json:"commission_amount"`
	CommissionAsset  currency.Code `json:"commission_asset"`
	Source           string        `json:"source"`
}

// AgencyTransactionHistoryResponse holds an agency's transaction history of recommended users.
type AgencyTransactionHistoryResponse struct {
	CurrencyPair currency.Pair        `json:"currency_pair"`
	Total        uint64               `json:"total"`
	List         []*RebateTransaction `json:"list"`
}

// AgencyCommissionHistoryResponse holds an agency's rebate history of recommended users.
type AgencyCommissionHistoryResponse struct {
	CurrencyPair currency.Pair       `json:"currency_pair"`
	Total        uint64              `json:"total"`
	List         []*RebateCommission `json:"list"`
}

// PartnerTransactionHistoryResponse holds a partner's transaction history of recommended users.
type PartnerTransactionHistoryResponse struct {
	Total uint64               `json:"total"`
	List  []*RebateTransaction `json:"list"`
}

// PartnerCommissionHistoryResponse holds a partner's rebate history of recommended users.
type PartnerCommissionHistoryResponse struct {
	Total uint64              `json:"total"`
	List  []*RebateCommission `json:"list"`
}

// PartnerSubordinate holds a single subordinate of a partner.
type PartnerSubordinate struct {
	UserID       uint64     `json:"user_id"`
	UserJoinTime types.Time `json:"user_join_time"`
	// Type identifies the subordinate kind: 1 - Sub-agent, 2 - Indirect direct customer, 3 - Direct direct customer.
	Type uint64 `json:"type"`
}

// PartnerSubordinateListResponse holds a partner's subordinate list including sub-agents, direct and indirect customers.
type PartnerSubordinateListResponse struct {
	Total uint64                `json:"total"`
	List  []*PartnerSubordinate `json:"list"`
}

// BrokerSubBrokerInfo holds the sub-broker commission rate details attached to a broker rebate record.
type BrokerSubBrokerInfo struct {
	UserID                 uint64       `json:"user_id"`
	OriginalCommissionRate types.Number `json:"original_commission_rate"`
	RelativeCommissionRate types.Number `json:"relative_commission_rate"`
	CommissionRate         types.Number `json:"commission_rate"`
}

// BrokerCommission holds a single broker rebate commission record.
type BrokerCommission struct {
	CommissionTime       types.Time           `json:"commission_time"`
	UserID               uint64               `json:"user_id"`
	GroupName            string               `json:"group_name"`
	Amount               types.Number         `json:"amount"`
	Fee                  types.Number         `json:"fee"`
	FeeAsset             currency.Code        `json:"fee_asset"`
	RebateFeeUSDT        types.Number         `json:"rebate_fee"`
	Source               string               `json:"source"`
	CurrencyPair         currency.Pair        `json:"currency_pair"`
	SubBrokerInfo        *BrokerSubBrokerInfo `json:"sub_broker_info"`
	AlphaContractAddress string               `json:"alpha_contract_addr"`
}

// BrokerTransaction holds a single broker trading history record.
type BrokerTransaction struct {
	TransactionTime      types.Time           `json:"transaction_time"`
	UserID               uint64               `json:"user_id"`
	GroupName            string               `json:"group_name"`
	FeeUSDT              types.Number         `json:"fee"`
	CurrencyPair         currency.Pair        `json:"currency_pair"`
	Amount               types.Number         `json:"amount"`
	FeeAsset             currency.Code        `json:"fee_asset"`
	Source               string               `json:"source"`
	SubBrokerInfo        *BrokerSubBrokerInfo `json:"sub_broker_info"`
	AlphaContractAddress string               `json:"alpha_contract_addr"`
}

// BrokerCommissionHistoryResponse holds a broker's rebate records for users.
type BrokerCommissionHistoryResponse struct {
	Total uint64              `json:"total"`
	List  []*BrokerCommission `json:"list"`
}

// BrokerTransactionHistoryResponse holds a broker's trading history for users.
type BrokerTransactionHistoryResponse struct {
	Total uint64               `json:"total"`
	List  []*BrokerTransaction `json:"list"`
}

// UserSubordinateRelation holds a single user subordinate relationship record.
type UserSubordinateRelation struct {
	UserID uint64 `json:"uid"`
	// Affiliation is partner or referral, or empty when the user belongs to neither system
	Affiliation string `json:"belong"`
	// Type identifies the relationship: 0 - Not in system, 1 - Direct subordinate agent, 2 - Indirect subordinate agent, 3 - Direct direct customer, 4 - Indirect direct customer, 5 - Regular user.
	Type          uint64 `json:"type"`
	InviterUserID uint64 `json:"ref_uid"`
}

// UserSubordinateRelationResponse holds user subordinate relationship records.
type UserSubordinateRelationResponse struct {
	List []*UserSubordinateRelation `json:"list"`
}

// PartnerApplicationRecordResponse holds a partner application record
type PartnerApplicationRecordResponse struct {
	ID                 uint64                    `json:"id"`
	UserID             uint64                    `json:"uid"`
	Language           string                    `json:"language"`
	CountryID          uint64                    `json:"country_id"`
	FirstName          string                    `json:"firstname"`
	LastName           string                    `json:"lastname"`
	Email              string                    `json:"email"`
	JoinUserID         uint64                    `json:"join_uid"`
	JoinCountryID      uint64                    `json:"join_country_id"`
	IdentityComment    string                    `json:"identity_comment"`
	PromotionChannels  string                    `json:"promotion_channels"` // Channel IDs separated by |
	ContactDetails     string                    `json:"contact_details"`    // JSON object encoded as a string
	KnowDetails        string                    `json:"know_details"`
	QuestionLanguage   string                    `json:"question_lang"`
	CreateTime         string                    `json:"create_timest"` // Formatted as 2006-01-02 15:04:05 without a time zone
	UpdateTime         string                    `json:"update_timest"` // Formatted as 2006-01-02 15:04:05 without a time zone
	ApplyType          uint64                    `json:"apply_type"`
	AuditStatus        uint64                    `json:"audit_status"` // 0 - Pending, 1 - Approved, 2 - Rejected
	EditCounts         uint64                    `json:"edit_counts"`
	ProofImages        string                    `json:"proof_images"`
	ProofVideos        string                    `json:"proof_videos"`
	ProofURL           string                    `json:"proof_url"`
	AuditReason        uint64                    `json:"audit_reason"`
	ChannelType        uint64                    `json:"channel_type"`
	Region             string                    `json:"region"`
	Phone              string                    `json:"phone"`
	Telegram           string                    `json:"telegram"`
	OtherContact       PartnerApplicationContact `json:"other_contact"`
	ProofImagesURLList []string                  `json:"proof_images_url_list"`
	ProofVideosURLList []string                  `json:"proof_videos_url_list"`
	ApplyMessage       string                    `json:"apply_msg"`
	JumpURL            string                    `json:"jump_url"`
}

// PartnerApplicationContact holds an additional contact method of a partner application
type PartnerApplicationContact struct {
	Type  uint64 `json:"type"`
	Value string `json:"value"`
}

// PartnerEligibilityResponse holds whether the user can apply as a partner, and why not when ineligible
type PartnerEligibilityResponse struct {
	Eligible         bool     `json:"eligible"`
	BlockReasons     []string `json:"block_reasons"`
	BlockReasonCodes []string `json:"block_reason_codes"`
}

// RebateAgentStatisticsResponse represents rebate agent statistics
type RebateAgentStatisticsResponse struct {
	RebateAmount     types.Number `json:"rebate_amount"`
	TradeVolume      types.Number `json:"trade_volume"`
	NetFee           types.Number `json:"net_fee"`
	CustomerCount    uint64       `json:"customer_count"`
	TradingUserCount types.Number `json:"trading_user_count"` // Only reported when all business types are queried
	TimeRangeDesc    string       `json:"time_range_desc"`
	BusinessType     uint64       `json:"business_type"`
	BusinessTypeDesc string       `json:"business_type_desc"`
}
