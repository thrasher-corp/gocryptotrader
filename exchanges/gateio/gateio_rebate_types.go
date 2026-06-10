package gateio

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

// RebateTransaction holds a single transaction history record for a recommended user.
type RebateTransaction struct {
	TransactionTime types.Time    `json:"transaction_time"`
	UserID          uint64        `json:"user_id"`
	GroupName       string        `json:"group_name"`
	Fee             types.Number  `json:"fee"`
	FeeAsset        string        `json:"fee_asset"`
	CurrencyPair    currency.Pair `json:"currency_pair"`
	Amount          types.Number  `json:"amount"`
	AmountAsset     string        `json:"amount_asset"`
	Source          string        `json:"source"`
}

// RebateCommission holds a single rebate commission record for a recommended user.
type RebateCommission struct {
	CommissionTime   types.Time   `json:"commission_time"`
	UserID           uint64       `json:"user_id"`
	GroupName        string       `json:"group_name"`
	CommissionAmount types.Number `json:"commission_amount"`
	CommissionAsset  string       `json:"commission_asset"`
	Source           string       `json:"source"`
}

// AgencyTransactionHistoryResponse holds an agency's transaction history of recommended users.
type AgencyTransactionHistoryResponse struct {
	CurrencyPair currency.Pair        `json:"currency_pair"`
	Total        int64                `json:"total"`
	List         []*RebateTransaction `json:"list"`
}

// AgencyCommissionHistoryResponse holds an agency's rebate history of recommended users.
type AgencyCommissionHistoryResponse struct {
	CurrencyPair currency.Pair       `json:"currency_pair"`
	Total        int64               `json:"total"`
	List         []*RebateCommission `json:"list"`
}

// PartnerTransactionHistoryResponse holds a partner's transaction history of recommended users.
type PartnerTransactionHistoryResponse struct {
	Total int64                `json:"total"`
	List  []*RebateTransaction `json:"list"`
}

// PartnerCommissionHistoryResponse holds a partner's rebate history of recommended users.
type PartnerCommissionHistoryResponse struct {
	Total int64               `json:"total"`
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
	Total int64                 `json:"total"`
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
	CommissionTime    types.Time           `json:"commission_time"`
	UserID            uint64               `json:"user_id"`
	GroupName         string               `json:"group_name"`
	Amount            types.Number         `json:"amount"`
	Fee               types.Number         `json:"fee"`
	FeeAsset          string               `json:"fee_asset"`
	RebateFee         types.Number         `json:"rebate_fee"`
	Source            string               `json:"source"`
	CurrencyPair      currency.Pair        `json:"currency_pair"`
	SubBrokerInfo     *BrokerSubBrokerInfo `json:"sub_broker_info"`
	AlphaContractAddr string               `json:"alpha_contract_addr"`
}

// BrokerTransaction holds a single broker trading history record.
type BrokerTransaction struct {
	TransactionTime   types.Time           `json:"transaction_time"`
	UserID            uint64               `json:"user_id"`
	GroupName         string               `json:"group_name"`
	Fee               types.Number         `json:"fee"`
	CurrencyPair      currency.Pair        `json:"currency_pair"`
	Amount            types.Number         `json:"amount"`
	FeeAsset          string               `json:"fee_asset"`
	Source            string               `json:"source"`
	SubBrokerInfo     *BrokerSubBrokerInfo `json:"sub_broker_info"`
	AlphaContractAddr string               `json:"alpha_contract_addr"`
}

// BrokerCommissionHistoryResponse holds a broker's rebate records for users.
type BrokerCommissionHistoryResponse struct {
	Total int64               `json:"total"`
	List  []*BrokerCommission `json:"list"`
}

// BrokerTransactionHistoryResponse holds a broker's trading history for users.
type BrokerTransactionHistoryResponse struct {
	Total int64                `json:"total"`
	List  []*BrokerTransaction `json:"list"`
}

// UserSubordinateRelation holds a single user subordinate relationship record.
type UserSubordinateRelation struct {
	UID    uint64 `json:"uid"`
	Belong string `json:"belong"`
	// Type identifies the relationship: 0 - Not in system, 1 - Direct subordinate agent, 2 - Indirect subordinate agent, 3 - Direct direct customer, 4 - Indirect direct customer, 5 - Regular user.
	Type   uint64 `json:"type"`
	RefUID uint64 `json:"ref_uid"`
}

// UserSubordinateRelationResponse holds user subordinate relationship records.
type UserSubordinateRelationResponse struct {
	List []*UserSubordinateRelation `json:"list"`
}
