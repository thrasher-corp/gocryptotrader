package bingx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AssetBalancesResponse holds an account's balance of each asset
type AssetBalancesResponse struct {
	Balances []AssetBalance `json:"balances"`
}

// AssetBalance holds the available and locked balance of an asset
type AssetBalance struct {
	Asset       currency.Code `json:"asset"`
	DisplayName string        `json:"displayName"`
	Free        types.Number  `json:"free"`
	Locked      types.Number  `json:"locked"`
}

// AssetTransferRecordsRequest holds the parameters of an asset transfer records query, which needs a transfer type or a
// transaction ID
type AssetTransferRecordsRequest struct {
	Type          AssetTransferType
	TransactionID uint64
	StartTime     time.Time
	EndTime       time.Time
	Current       uint64 // Page number, from 1
	Size          uint64 // Page size, up to 100; 10 when unset
}

// AssetTransferRecordsResponse holds a page of asset transfer records
type AssetTransferRecordsResponse struct {
	Total uint64                `json:"total"`
	Rows  []AssetTransferRecord `json:"rows"`
}

// AssetTransferRecord holds an asset transfer between the account's own accounts
type AssetTransferRecord struct {
	Asset         currency.Code     `json:"asset"`
	Amount        types.Number      `json:"amount"`
	Type          AssetTransferType `json:"type"`
	Status        string            `json:"status"`
	TransactionID NumericID         `json:"tranId"`
	Timestamp     types.Time        `json:"timestamp"`
}

// InternalTransferRequest holds the parameters of an internal transfer to another BingX account, sent as a JSON body
type InternalTransferRequest struct {
	Coin             currency.Code   `json:"coin"`
	UserAccountType  UserAccountType `json:"userAccountType"`
	UserAccount      string          `json:"userAccount"`           // The recipient's UID, phone number or email address
	Amount           float64         `json:"amount"`                // Amount to transfer
	CallingCode      string          `json:"callingCode,omitempty"` // Telephone area code, required for a phone number
	WalletType       WalletType      `json:"walletType"`            // Account the transfer is paid from
	TransferClientID string          `json:"transferClientId,omitempty"`
}

// InternalTransferResponse holds the IDs of an internal transfer
type InternalTransferResponse struct {
	ID               string `json:"id"`
	TransferClientID string `json:"transferClientId"`
}

// AssetTransferRequest holds the parameters of a transfer between the account's own accounts
type AssetTransferRequest struct {
	FromAccount TransferAccount
	ToAccount   TransferAccount
	Asset       currency.Code
	Amount      float64
}

// AssetTransferResponse holds the ID of an asset transfer
type AssetTransferResponse struct {
	TransferID string `json:"transferId"`
}

// TransferableCurrenciesResponse holds the coins that can be transferred between two accounts
type TransferableCurrenciesResponse struct {
	Coins []TransferableCoin `json:"coins"`
}

// TransferableCoin holds the amount of a coin that can be transferred
type TransferableCoin struct {
	Asset  currency.Code `json:"asset"`
	Amount types.Number  `json:"amount"`
}

// AssetTransferRecordsNewRequest holds the parameters of an asset transfer records new query, which needs both accounts
// or a transaction ID
type AssetTransferRecordsNewRequest struct {
	FromAccount   TransferAccount
	ToAccount     TransferAccount
	TransactionID string // A record's TransferID
	StartTime     time.Time
	EndTime       time.Time
	PageIndex     uint64 // From 1
	PageSize      uint64 // Up to 100; 10 when unset
}

// AssetTransferRecordsNewResponse holds a page of asset transfer records
type AssetTransferRecordsNewResponse struct {
	Total uint64                   `json:"total"`
	Rows  []AssetTransferRecordNew `json:"rows"`
}

// AssetTransferRecordNew holds an asset transfer between the account's own accounts
type AssetTransferRecordNew struct {
	Asset       currency.Code   `json:"asset"`
	Amount      types.Number    `json:"amount"`
	Status      string          `json:"status"`
	TransferID  string          `json:"transferId"`
	Timestamp   types.Time      `json:"timestamp"`
	FromAccount TransferAccount `json:"fromAccount"`
	ToAccount   TransferAccount `json:"toAccount"`
}

// InternalTransferRecordsRequest holds the parameters of an internal transfer records query
type InternalTransferRecordsRequest struct {
	Coin             currency.Code
	ID               string // Platform transfer ID, which takes precedence over TransferClientID
	TransferClientID string
	StartTime        time.Time
	EndTime          time.Time
	Offset           uint64
	Limit            uint64 // Up to 1000; 100 when unset
}

// InternalTransferRecordsResponse holds a page of internal transfer records
type InternalTransferRecordsResponse struct {
	Data  []InternalTransferRecord `json:"data"`
	Total uint64                   `json:"total"`
}

// InternalTransferRecord holds an internal transfer
type InternalTransferRecord struct {
	ID               NumericID                  `json:"id"`
	Coin             currency.Code              `json:"coin"`
	Receiver         NumericID                  `json:"receiver"` // Recipient's UID
	Amount           float64                    `json:"amount"`
	Time             types.Time                 `json:"time"`
	Status           WithdrawStatus             `json:"status"`
	TransferClientID string                     `json:"transferClientId"`
	FromUID          NumericID                  `json:"fromUid"`
	RecordType       InternalTransferRecordType `json:"recordType"`
}

// AssetOverviewResponse holds the USDT value of each of the account's accounts
type AssetOverviewResponse struct {
	Result []AccountBalanceOverview `json:"result"`
	PageID uint64                   `json:"pageId"`
	Total  uint64                   `json:"total"`
}

// AccountBalanceOverview holds the USDT value of an account
type AccountBalanceOverview struct {
	AccountType AccountType  `json:"accountType"`
	USDTBalance types.Number `json:"usdtBalance"`
}
