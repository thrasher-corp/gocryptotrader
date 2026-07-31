package htx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/types"
)

// V5SetAssetModeRequest defines an account asset-mode change.
type V5SetAssetModeRequest struct {
	AssetMode uint64 `json:"asset_mode"`
}

// V5AssetModeResponse stores the current account asset mode.
type V5AssetModeResponse struct {
	V5Response
	Data struct {
		AssetMode uint64 `json:"asset_mode"`
	} `json:"data"`
}

// V5AccountBillsRequest defines account transaction-record filters.
type V5AccountBillsRequest struct {
	ContractCode string
	MarginMode   string
	StartTime    time.Time
	Type         string
	EndTime      time.Time
	From         uint64
	Limit        uint64
	Direction    string
}

// V5AccountBillsResponse stores account transaction records.
type V5AccountBillsResponse struct {
	V5Response
	Data []V5AccountBill `json:"data"`
}

// V5AccountBill stores an account transaction record.
type V5AccountBill struct {
	ID           string       `json:"id"`
	Type         string       `json:"type"`
	ContractCode string       `json:"contract_code"`
	MarginMode   string       `json:"margin_mode"`
	Currency     string       `json:"currency"`
	Amount       types.Number `json:"amount"`
	CreatedTime  types.Time   `json:"created_time"`
}

// V5SetFeeDeductionCurrencyRequest defines a fee-deduction currency change.
type V5SetFeeDeductionCurrencyRequest struct {
	FeeOption         uint64 `json:"fee_option"`
	DeductionCurrency string `json:"deduction_currency"`
}

// V5FeeDeductionCurrencyResponse stores the configured fee-deduction currency.
type V5FeeDeductionCurrencyResponse struct {
	V5Response
	Data struct {
		FeeOption         uint64 `json:"fee_option"`
		DeductionCurrency string `json:"deduction_currency"`
	} `json:"data"`
}

// V5UniversalTransferRequest defines a transfer between supported HTX account types.
type V5UniversalTransferRequest struct {
	Amount          uint64 `json:"amount"`
	Currency        string `json:"currency"`
	FromAccountType string `json:"from_account_type"`
	ToAccountType   string `json:"to_account_type"`
	FromAssetType   string `json:"from_asset_type,omitempty"`
	ToAssetType     string `json:"to_asset_type,omitempty"`
}

// V5UniversalTransferResponse stores a universal-transfer acknowledgement.
type V5UniversalTransferResponse struct {
	V5Response
	Data struct {
		TransferID uint64 `json:"transfer_id"`
	} `json:"data"`
}

// V5UniversalTransferRecordsRequest defines universal-transfer record filters.
type V5UniversalTransferRecordsRequest struct {
	TransferID uint64
	Currency   string
	Status     string
	StartTime  time.Time
	EndTime    time.Time
	From       uint64
	Limit      uint64
	Direction  string
}

// V5UniversalTransferRecordsResponse stores universal-transfer records.
type V5UniversalTransferRecordsResponse struct {
	V5Response
	Data []V5UniversalTransferRecord `json:"data"`
}

// V5UniversalTransferRecord stores a universal-transfer record.
type V5UniversalTransferRecord struct {
	ID              uint64       `json:"id"`
	TransferID      types.Number `json:"transfer_id"`
	Amount          types.Number `json:"amount"`
	Currency        string       `json:"currency"`
	Status          string       `json:"status"`
	FromAccountType string       `json:"from_account_type"`
	ToAccountType   string       `json:"to_account_type"`
	FromAssetType   string       `json:"from_asset_type"`
	ToAssetType     string       `json:"to_asset_type"`
	TransferTime    types.Time   `json:"transfer_time"`
}
