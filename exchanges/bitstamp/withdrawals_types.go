package bitstamp

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Bank withdrawal types
const (
	BankWithdrawalTypeSEPA          = "sepa"
	BankWithdrawalTypeInternational = "international"
)

// WithdrawalType is the type of a withdrawal request
// Cryptocurrency withdrawals use a type number per currency, so only the bank withdrawal types are named
type WithdrawalType uint64

// Bank withdrawal request types
const (
	WithdrawalTypeSEPA WithdrawalType = 0
	WithdrawalTypeWire WithdrawalType = 2
)

// WithdrawalStatus is the status of a withdrawal request
type WithdrawalStatus uint64

// Withdrawal request statuses
const (
	WithdrawalStatusOpen      WithdrawalStatus = 0
	WithdrawalStatusInProcess WithdrawalStatus = 1
	WithdrawalStatusFinished  WithdrawalStatus = 2
	WithdrawalStatusCancelled WithdrawalStatus = 3
	WithdrawalStatusFailed    WithdrawalStatus = 4
	WithdrawalStatusReversed  WithdrawalStatus = 11
)

// WithdrawalRequestsRequest holds the parameters for fetching withdrawal requests
// TimeDelta returns withdrawal requests created within the supplied duration, between 1 and 50000000 seconds
type WithdrawalRequestsRequest struct {
	ID        uint64
	TimeDelta time.Duration
	Limit     uint64
	Offset    uint64
}

// WithdrawalRequestResponse holds a withdrawal request
// BitstampTransactionID is Bitstamp's own transaction ID, while TransactionID is the blockchain transaction ID of a
// cryptocurrency withdrawal and is empty until the withdrawal has been processed
type WithdrawalRequestResponse struct {
	ID                    uint64           `json:"id"`
	DateTime              types.DateTime   `json:"datetime"`
	Type                  WithdrawalType   `json:"type"`
	Currency              currency.Code    `json:"currency"`
	Network               string           `json:"network"`
	Amount                types.Number     `json:"amount"`
	Status                WithdrawalStatus `json:"status"`
	BitstampTransactionID uint64           `json:"txid"`
	Address               string           `json:"address"`
	TransactionID         string           `json:"transaction_id"`
}

// BankWithdrawalRequest holds the parameters for a SEPA or international bank withdrawal
// Name, Address, PostalCode, City and Country describe the account holder, and the Bank fields describe the target bank
type BankWithdrawalRequest struct {
	Amount          float64
	AccountCurrency currency.Code
	Name            string
	IBAN            string
	BIC             string
	Address         string
	PostalCode      string
	City            string
	Country         string
	Type            string
	Comment         string

	// The following fields only apply to international withdrawals
	BankName                       string
	BankAddress                    string
	BankPostalCode                 string
	BankCity                       string
	BankCountry                    string
	Currency                       currency.Code
	IntermediaryRoutingNumberOrBIC string
}

// BankWithdrawalResponse holds the ID of a bank withdrawal request
type BankWithdrawalResponse struct {
	WithdrawalID uint64 `json:"withdrawal_id"`
}

// CancelWithdrawalResponse holds the details of a cancelled withdrawal request
type CancelWithdrawalResponse struct {
	ID              uint64        `json:"id,string"`
	Amount          types.Number  `json:"amount"`
	Currency        currency.Code `json:"currency"`
	AccountCurrency currency.Code `json:"account_currency"`
	Type            string        `json:"type"`
}

// WithdrawalStatusResponse holds the status of a bank withdrawal request
type WithdrawalStatusResponse struct {
	Status string `json:"status"`
}

// CryptoWithdrawalRequest holds the parameters for a cryptocurrency withdrawal
type CryptoWithdrawalRequest struct {
	Currency        currency.Code
	Network         string
	Amount          float64
	Address         string
	MemoID          string
	DestinationTag  string
	TransferID      uint64
	OriginatorInfo  *CustomerInfo
	BeneficiaryInfo *CustomerInfo
	// BeneficiaryThirdParty is sent when set, and must be false when withdrawing to an address in the account holder's name
	BeneficiaryThirdParty           *bool
	BeneficiaryID                   string
	VirtualAssetServiceProviderUUID string
}

// RippleIOUWithdrawalRequest holds the parameters for a Ripple IOU withdrawal
type RippleIOUWithdrawalRequest struct {
	Currency        currency.Code
	Amount          float64
	Address         string
	OriginatorInfo  *CustomerInfo
	BeneficiaryInfo *CustomerInfo
	// BeneficiaryThirdParty is sent when set, and must be false when withdrawing to an address in the account holder's name
	BeneficiaryThirdParty           *bool
	BeneficiaryID                   string
	VirtualAssetServiceProviderUUID string
}

// CryptoWithdrawalResponse holds the ID of a cryptocurrency withdrawal request
type CryptoWithdrawalResponse struct {
	ID uint64 `json:"id"`
}

// travelRuleParams holds the optional Travel Rule parameters shared by cryptocurrency withdrawals
type travelRuleParams struct {
	OriginatorInfo                  *CustomerInfo
	BeneficiaryInfo                 *CustomerInfo
	BeneficiaryThirdParty           *bool
	BeneficiaryID                   string
	VirtualAssetServiceProviderUUID string
}
