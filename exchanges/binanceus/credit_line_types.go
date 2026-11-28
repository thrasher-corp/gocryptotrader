package binanceus

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CreditLineAccountInformationResponse holds the credit line account's terms, loans and balances
type CreditLineAccountInformationResponse struct {
	CreditLineAccountID          uint64                `json:"clAccountId"`
	MasterAccountID              uint64                `json:"masterAccountId"`
	NotificationMailAddress      string                `json:"notificationMailAddr"`
	CurrentLoanToValue           types.Number          `json:"currentLTV"`
	InitialLoanToValue           types.Number          `json:"initialLTV"`
	MarginCallLoanToValue        types.Number          `json:"marginCallLTV"`
	LiquidationLoanToValue       types.Number          `json:"liquidationLTV"`
	InterestRate                 types.Number          `json:"interestRate"`
	LiquidationFeeRate           types.Number          `json:"liquidationFeeRate"`
	ContractStartTime            types.Time            `json:"contractStartTime"`
	ContractEndTime              types.Time            `json:"contractEndTime"`
	IsActive                     bool                  `json:"isActive"`
	CanTrade                     bool                  `json:"canTrade"`
	CanTransferIn                bool                  `json:"canTransferIn"`
	CanTransferOut               bool                  `json:"canTransferOut"`
	RequiredDepositAmount        types.Number          `json:"requiredDepositAmount"`
	AvailableAmountToTransferOut types.Number          `json:"availableAmountToTransferOut"`
	LoanAssets                   []CreditLineLoanAsset `json:"loanAssets"`
	Balances                     []CreditLineBalance   `json:"balances"`
}

// CreditLineLoanAsset holds a loan of the credit line
type CreditLineLoanAsset struct {
	LoanAsset           currency.Code `json:"loanAsset"`
	LoanQuantity        types.Number  `json:"loanQuantity"`
	CurrentLoanQuantity types.Number  `json:"currentLoanQuantity"`
	ReclaimSequence     uint64        `json:"reclaimSequence"`
}

// CreditLineBalance holds an asset's balance in the credit line account
type CreditLineBalance struct {
	Asset  currency.Code `json:"asset"`
	Free   types.Number  `json:"free"`
	Locked types.Number  `json:"locked"`
	Freeze types.Number  `json:"freeze"` // Sent by the alert history only
}

// CreditLineAlertHistoryRequest holds the optional filters of a credit line alert history query
type CreditLineAlertHistoryRequest struct {
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64 // 200 when zero
	AlertType CreditLineAlertType
}

// CreditLineAlertRecordResponse holds a margin call or liquidation alert with the account's state when it was sent
type CreditLineAlertRecordResponse struct {
	AlertTime              types.Time            `json:"alertTime"`
	AlertType              CreditLineAlertType   `json:"alertType"`
	CurrentLoanToValue     types.Number          `json:"currentLTV"`
	InitialLoanToValue     types.Number          `json:"initialLTV"`
	MarginCallLoanToValue  types.Number          `json:"marginCallLTV"`
	LiquidationLoanToValue types.Number          `json:"liquidationLTV"`
	TotalBalance           types.Number          `json:"totalBalance"`
	LoanAssets             []CreditLineLoanAsset `json:"loanAssets"`
	Balances               []CreditLineBalance   `json:"balances"`
}

// CreditLineTransferHistoryRequest holds the optional filters of a credit line transfer history query
type CreditLineTransferHistoryRequest struct {
	StartTime    time.Time
	EndTime      time.Time
	Limit        uint64 // 20 when zero, at most 100
	TransferType CreditLineTransferType
	Asset        currency.Code
}

// CreditLineTransferRecordResponse holds a transfer into or out of the credit line account
type CreditLineTransferRecordResponse struct {
	TransferID   string                 `json:"transferId"`
	TransferType CreditLineTransferType `json:"transferType"`
	Asset        currency.Code          `json:"asset"`
	Amount       types.Number           `json:"amount"`
	Status       string                 `json:"status"`
	TransferTime types.Time             `json:"transferTime"`
}

// CreditLineTransferRequest holds the parameters of a transfer into or out of the credit line account
type CreditLineTransferRequest struct {
	TransferType      CreditLineTransferType
	TransferAssetType currency.Code // The asset transferred
	Quantity          float64
}

// CreditLineTransferResponse holds the outcome of a credit line transfer
type CreditLineTransferResponse struct {
	TransferID uint64 `json:"transferId"`
	Status     string `json:"status"`
}
