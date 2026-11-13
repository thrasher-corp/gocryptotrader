package bybit

import (
	"time"
	"uuid"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// InternalTransferRequest holds the Create Internal Transfer parameters, all of them required
type InternalTransferRequest struct {
	// TransferID is a UUID the caller generates
	TransferID      uuid.UUID
	Coin            currency.Code
	Amount          float64
	FromAccountType string
	ToAccountType   string
}

// TransferResponse is the outcome of a transfer
type TransferResponse struct {
	TransferID string `json:"transferId"`
	// Status is STATUS_UNKNOWN, SUCCESS, PENDING or FAILED
	Status string `json:"status"`
}

// InternalTransferRecordsRequest holds the Get Internal Transfer Records filters; a window spans at most 7 days
type InternalTransferRecordsRequest struct {
	TransferID string
	Coin       currency.Code
	Status     string
	StartTime  time.Time
	EndTime    time.Time
	Limit      uint64
	Cursor     string
}

// InternalTransferRecordsResponse is a page of transfers between account types
type InternalTransferRecordsResponse struct {
	List           []InternalTransferRecord `json:"list"`
	NextPageCursor string                   `json:"nextPageCursor"`
}

// InternalTransferRecord is one transfer between account types
type InternalTransferRecord struct {
	TransferID      string        `json:"transferId"`
	Coin            currency.Code `json:"coin"`
	Amount          types.Number  `json:"amount"`
	FromAccountType string        `json:"fromAccountType"`
	ToAccountType   string        `json:"toAccountType"`
	Timestamp       types.Time    `json:"timestamp"`
	Status          string        `json:"status"`
}

// UniversalTransferRequest holds the Create Universal Transfer parameters, all of them required
type UniversalTransferRequest struct {
	// TransferID is a UUID the caller generates
	TransferID      uuid.UUID
	Coin            currency.Code
	Amount          float64
	FromMemberID    uint64
	ToMemberID      uint64
	FromAccountType string
	ToAccountType   string
}

// UniversalTransferRecordsRequest holds the Get Universal Transfer Records filters; a window spans at most 7 days
type UniversalTransferRecordsRequest struct {
	TransferID string
	Coin       currency.Code
	// Status is SUCCESS, FAILED or PENDING
	Status       string
	StartTime    time.Time
	EndTime      time.Time
	FromMemberID string
	ToMemberID   string
	Limit        uint64
	Cursor       string
}

// UniversalTransferRecordsResponse is a page of transfers between UIDs
type UniversalTransferRecordsResponse struct {
	List           []UniversalTransferRecord `json:"list"`
	NextPageCursor string                    `json:"nextPageCursor"`
}

// UniversalTransferRecord is one transfer between UIDs
type UniversalTransferRecord struct {
	TransferID   string        `json:"transferId"`
	Coin         currency.Code `json:"coin"`
	Amount       types.Number  `json:"amount"`
	FromMemberID string        `json:"fromMemberId"`
	ToMemberID   string        `json:"toMemberId"`
	// WithdrawRequestID and DepositRequestID match the id of the debit and credit in Get Transaction Log
	WithdrawRequestID string     `json:"withdrawReqId"`
	DepositRequestID  string     `json:"depositReqId"`
	FromAccountType   string     `json:"fromAccountType"`
	ToAccountType     string     `json:"toAccountType"`
	Timestamp         types.Time `json:"timestamp"`
	Status            string     `json:"status"`
}

// TransferableCoinResponse lists the coins that can be transferred between two account types
type TransferableCoinResponse struct {
	List []currency.Code `json:"list"`
}
