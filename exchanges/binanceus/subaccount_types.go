package binanceus

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SubAccountInformationRequest holds the optional filters of a sub-account information request
type SubAccountInformationRequest struct {
	Email  string
	Status string // enabled or disabled
	Page   uint64 // 1 when zero
	Limit  uint64 // 500 when zero
}

// SubAccountInformationResponse holds the master account's sub-accounts
type SubAccountInformationResponse struct {
	Success     bool         `json:"success"`
	SubAccounts []SubAccount `json:"subAccounts"`
}

// SubAccount holds a sub-account
type SubAccount struct {
	Email                      string     `json:"email"`
	Status                     string     `json:"status"`
	Activated                  bool       `json:"activated"`
	Mobile                     string     `json:"mobile"`
	GoogleAuthenticatorEnabled bool       `json:"gAuth"`
	CreateTime                 types.Time `json:"createTime"`
}

// SubAccountTransferHistoryRequest holds the optional filters of a sub-account transfer history request
type SubAccountTransferHistoryRequest struct {
	Email     string
	StartTime time.Time
	EndTime   time.Time
	Page      uint64 // Batch of at most 500 transfers
	Limit     uint64 // 500 when zero
}

// SubAccountTransferHistoryResponse holds sub-account transfers
type SubAccountTransferHistoryResponse struct {
	Message   string               `json:"msg"`
	Success   bool                 `json:"success"`
	Transfers []SubAccountTransfer `json:"transfers"`
}

// SubAccountTransfer holds a transfer between the master account and a sub-account
type SubAccountTransfer struct {
	Asset    currency.Code `json:"asset"`
	From     string        `json:"from"`
	Quantity types.Number  `json:"qty"`
	Time     types.Time    `json:"time"`
	To       string        `json:"to"`
}

// SubAccountTransferRequest holds the parameters of a transfer between the master account and a sub-account
type SubAccountTransferRequest struct {
	FromEmail string
	ToEmail   string
	Asset     currency.Code
	Amount    float64
}

// SubAccountTransferResponse holds an executed sub-account transfer
type SubAccountTransferResponse struct {
	Message       string `json:"msg"`
	Success       bool   `json:"success"`
	TransactionID string `json:"txnId"`
}

// SubAccountAssetsResponse holds a sub-account's balances
type SubAccountAssetsResponse struct {
	Balances []SubAccountBalance `json:"balances"`
	Success  bool                `json:"success"`
}

// SubAccountBalance holds a sub-account's free and locked balance of an asset
type SubAccountBalance struct {
	Asset  currency.Code `json:"asset"`
	Free   types.Number  `json:"free"`
	Locked types.Number  `json:"locked"`
}

// MasterAccountTotalUSDValueRequest holds the optional filters of a master account total USD value request
type MasterAccountTotalUSDValueRequest struct {
	Email string
	Page  uint64
	Size  uint64
}

// MasterAccountTotalUSDValueResponse holds the master account's total asset value and each sub-account's
type MasterAccountTotalUSDValueResponse struct {
	TotalCount                uint64                 `json:"totalCount"`
	MasterAccountTotalAsset   float64                `json:"masterAccountTotalAsset"`
	SpotSubUserAssetBTCVOList []SubAccountTotalAsset `json:"spotSubUserAssetBtcVoList"`
}

// SubAccountTotalAsset holds a sub-account's total asset value
type SubAccountTotalAsset struct {
	Email      string  `json:"email"`
	TotalAsset float64 `json:"totalAsset"`
}

// SubAccountStatusResponse holds a sub-account's status
type SubAccountStatusResponse struct {
	Email            string     `json:"email"`
	InsertTime       types.Time `json:"insertTime"`
	Mobile           string     `json:"mobile"`
	IsUserActive     bool       `json:"isUserActive"`
	IsMarginEnabled  bool       `json:"isMarginEnabled"`
	IsSubUserEnabled bool       `json:"isSubUserEnabled"`
	IsFutureEnabled  bool       `json:"isFutureEnabled"`
}
