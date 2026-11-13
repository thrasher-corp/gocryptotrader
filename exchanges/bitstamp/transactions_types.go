package bitstamp

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// User transaction types
const (
	TransactionTypeDeposit                       TransactionType = 0
	TransactionTypeWithdrawal                    TransactionType = 1
	TransactionTypeMarketTrade                   TransactionType = 2
	TransactionTypeSubAccountTransfer            TransactionType = 14
	TransactionTypeStakingCredit                 TransactionType = 25
	TransactionTypeSentToStaking                 TransactionType = 26
	TransactionTypeStakingReward                 TransactionType = 27
	TransactionTypeReferralReward                TransactionType = 32
	TransactionTypeSettlementTransfer            TransactionType = 33
	TransactionTypeInterAccountTransfer          TransactionType = 35
	TransactionTypeSimpleBuy                     TransactionType = 36
	TransactionTypeSmallBalanceConversion        TransactionType = 53
	TransactionTypeSmallBalanceConversionAlt     TransactionType = 55
	TransactionTypeDerivativesPeriodicSettlement TransactionType = 58
	TransactionTypeInsuranceFundClaim            TransactionType = 59
	TransactionTypeInsuranceFundPremium          TransactionType = 60
	TransactionTypeCollateralLiquidation         TransactionType = 61
)

// Cryptocurrency deposit review statuses and the reasons a review is pending
const (
	DepositStatusPending                          = "PENDING"
	DepositStatusFinalised                        = "FINALIZED"
	DepositStatusInProcessing                     = "IN_PROCESSING"
	DepositStatusReverted                         = "REVERTED"
	DepositStatusRejected                         = "REJECTED"
	DepositPendingReasonAddressVerificationNeeded = "ADDRESS_VERIFICATION_NEEDED"
	DepositPendingReasonAdditionalInfoMissing     = "ADDITIONAL_INFORMATION_MISSING"
)

// UserTransactionsRequest holds the parameters for fetching user transactions
// Pair is optional and restricts results to a single market
type UserTransactionsRequest struct {
	Pair    currency.Pair
	Offset  uint64
	Limit   uint64
	Sort    string
	Since   time.Time
	Until   time.Time
	SinceID uint64
}

// UserTransactionResponse holds a user transaction
// Amounts holds the signed amount of each currency involved in the transaction and ExchangeRates holds the
// execution rate for each market, both keyed by the names Bitstamp returns
type UserTransactionResponse struct {
	ID               uint64
	DateTime         time.Time
	Type             TransactionType
	Fee              float64
	OrderID          uint64
	SelfTrade        bool
	SelfTradeOrderID uint64
	Amounts          map[currency.Code]float64
	ExchangeRates    map[currency.Pair]float64
}

// CryptoTransactionsRequest holds the parameters for fetching cryptocurrency transactions
type CryptoTransactionsRequest struct {
	Limit       uint64
	Offset      uint64
	IncludeIOUs bool
	Since       time.Time
	Until       time.Time
}

// CryptoTransactionsResponse holds cryptocurrency deposits and withdrawals
type CryptoTransactionsResponse struct {
	RippleIOUTransactions []CryptoTransaction `json:"ripple_iou_transactions"`
	Deposits              []CryptoDeposit     `json:"deposits"`
	Withdrawals           []CryptoTransaction `json:"withdrawals"`
}

// CryptoTransaction holds a cryptocurrency withdrawal or Ripple IOU transaction
type CryptoTransaction struct {
	Currency           currency.Code `json:"currency"`
	Network            string        `json:"network"`
	DestinationAddress string        `json:"destinationAddress"`
	TxID               string        `json:"txid"`
	Amount             types.Number  `json:"amount"`
	DateTime           types.Time    `json:"datetime"`
}

// CryptoDeposit holds a cryptocurrency deposit
type CryptoDeposit struct {
	ID                 uint64        `json:"id"`
	Network            string        `json:"network"`
	Currency           currency.Code `json:"currency"`
	OriginatorAddress  string        `json:"originator_address"`
	TxID               string        `json:"txid"`
	Amount             types.Number  `json:"amount"`
	DateTime           types.Time    `json:"datetime"`
	Status             string        `json:"status"`
	PendingReason      string        `json:"pending_reason"`
	DestinationAddress string        `json:"destinationAddress"`
}

// CryptoDepositsRequest holds the parameters for fetching cryptocurrency deposits with their review status
type CryptoDepositsRequest struct {
	Offset uint64
	Limit  uint64
	Since  time.Time
	Until  time.Time
	Status string
}

// CryptoDepositResponse holds a cryptocurrency deposit and its review status
type CryptoDepositResponse struct {
	ID                 uint64        `json:"id"`
	Network            string        `json:"network"`
	Currency           currency.Code `json:"currency"`
	DestinationAddress string        `json:"destination_address"`
	OriginatorAddress  string        `json:"originator_address"`
	TxID               string        `json:"txid"`
	Amount             types.Number  `json:"amount"`
	DateTime           types.Time    `json:"datetime"`
	Status             string        `json:"status"`
	PendingReason      string        `json:"pending_reason"`
}

// DepositOriginatorRequest holds Travel Rule originator details for a pending cryptocurrency deposit
// OriginatorThirdParty is sent when set, and must be false when the deposit came from an address in the account holder's name
type DepositOriginatorRequest struct {
	OriginatorThirdParty            *bool         `json:"originator_thirdparty,omitempty"`
	OriginatorID                    string        `json:"originator_id,omitempty"`
	OriginatorInfo                  *CustomerInfo `json:"originator_info,omitempty"`
	BeneficiaryInfo                 *CustomerInfo `json:"beneficiary_info,omitempty"`
	VirtualAssetServiceProviderUUID string        `json:"vasp_uuid,omitempty"`
}
