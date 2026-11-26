package bingx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CreateSubAccountRequest holds the parameters of a sub-account creation, sent as a JSON body
type CreateSubAccountRequest struct {
	SubAccountString string `json:"subAccountString"` // Username: starts with a letter, holds a number, over 6 characters
	Note             string `json:"note,omitempty"`
}

// CreateSubAccountResponse holds a new sub-account
type CreateSubAccountResponse struct {
	SubUID           NumericID `json:"subUid"`
	SubAccountString string    `json:"subAccountString"`
	Note             string    `json:"note"`
}

// APIKeyPermissionsResponse holds the permissions of the API key making the request
type APIKeyPermissionsResponse struct {
	APIKey      string             `json:"apiKey"`
	Permissions []APIKeyPermission `json:"permissions"`
	IPAddresses []string           `json:"ipAddresses"` // IP whitelist
	Note        string             `json:"note"`
}

// AccountUIDResponse holds the UID and status of the account making the request
type AccountUIDResponse struct {
	UID           NumericID     `json:"uid"`
	AccountStatus AccountStatus `json:"accountStatus"`
}

// SubAccountListRequest holds the parameters of a sub-account list query
type SubAccountListRequest struct {
	SubUID           uint64
	SubAccountString string
	IsFeeze          *bool  // isFeeze, BingX's spelling: whether to list only frozen or only unfrozen sub-accounts
	Page             uint64 // From 1
	Limit            uint64 // Up to 1000
}

// SubAccountListResponse holds a page of sub-accounts
type SubAccountListResponse struct {
	Result []SubAccount `json:"result"`
	PageID uint64       `json:"pageId"`
	Total  uint64       `json:"total"`
}

// SubAccount holds a sub-account
type SubAccount struct {
	SubUID           NumericID  `json:"subUid"`
	SubAccountString string     `json:"subAccountString"`
	Note             string     `json:"note"`
	Freeze           bool       `json:"freeze"`
	CreateTime       types.Time `json:"createTime"`
}

// CreateSubAccountAPIKeyRequest holds the parameters of a sub-account API key creation, sent as a JSON body
type CreateSubAccountAPIKeyRequest struct {
	SubUID      uint64             `json:"subUid"`
	Note        string             `json:"note"`
	Permissions []APIKeyPermission `json:"permissions"`
	IPAddresses []string           `json:"ipAddresses,omitempty"` // IP whitelist
}

// CreateSubAccountAPIKeyResponse holds a new sub-account API key
type CreateSubAccountAPIKeyResponse struct {
	Note        string             `json:"note"`
	APIKey      string             `json:"apiKey"`
	APISecret   string             `json:"apiSecret"`
	Permissions []APIKeyPermission `json:"permissions"`
	IPAddresses []string           `json:"ipAddresses"`
}

// APIKeyInformationResponse holds the API keys of an account
type APIKeyInformationResponse struct {
	APIInfos []APIKeyInformation `json:"apiInfos"`
}

// APIKeyInformation holds an API key's settings
type APIKeyInformation struct {
	APIKey      string             `json:"apiKey"`
	Note        string             `json:"note"`
	Permissions []APIKeyPermission `json:"permissions"`
	IPAddresses []string           `json:"ipAddresses"`
	Status      uint64             `json:"status"` // Undocumented
	CreateTime  types.Time         `json:"createTime"`
	UpdateTime  types.Time         `json:"updateTime"`
}

// EditSubAccountAPIKeyRequest holds the parameters of a sub-account API key edit, sent as a JSON body
type EditSubAccountAPIKeyRequest struct {
	SubUID      uint64             `json:"subUid"`
	APIKey      string             `json:"apiKey"`
	Note        string             `json:"note"`
	Permissions []APIKeyPermission `json:"permissions"`
	IPAddresses []string           `json:"ipAddresses,omitempty"`
}

// EditSubAccountAPIKeyResponse holds an edited sub-account API key's settings
type EditSubAccountAPIKeyResponse struct {
	Note        string             `json:"note"`
	Permissions []APIKeyPermission `json:"permissions"`
	IPAddresses []string           `json:"ipAddresses"`
}

// DeleteSubAccountAPIKeyRequest holds the parameters of a sub-account API key deletion, sent as a JSON body
type DeleteSubAccountAPIKeyRequest struct {
	SubUID uint64 `json:"subUid"`
	APIKey string `json:"apiKey"`
}

// UpdateSubAccountStatusRequest holds the parameters of a sub-account freeze or unfreeze, sent as a JSON body
type UpdateSubAccountStatusRequest struct {
	SubUID uint64 `json:"subUid"`
	Freeze bool   `json:"freeze"`
}

// CreateSubAccountDepositAddressRequest holds the parameters of a sub-account deposit address creation
type CreateSubAccountDepositAddressRequest struct {
	Coin       currency.Code
	SubUID     uint64
	Network    string
	WalletType WalletType
}

// CreateSubAccountDepositAddressResponse holds a sub-account's deposit address of a coin on a network
type CreateSubAccountDepositAddressResponse struct {
	Address           string               `json:"address"`
	AddressWithPrefix string               `json:"addressWithPrefix"`
	AddressTag        string               `json:"addressTag"`
	Coin              currency.Code        `json:"coin"`
	Network           string               `json:"network"`
	Status            DepositAddressStatus `json:"status"`
	Timestamp         types.Time           `json:"ts"` // Creation time
	WalletType        WalletType           `json:"walletType"`
}

// SubAccountDepositAddressRequest holds the parameters of a sub-account deposit address query
type SubAccountDepositAddressRequest struct {
	Coin   currency.Code
	SubUID uint64
	Offset uint64
	Limit  uint64 // Up to 1000; 100 when unset
}

// SubAccountDepositRecordsRequest holds the parameters of a sub-account deposit records query
type SubAccountDepositRecordsRequest struct {
	Coin      currency.Code
	SubUID    uint64 // Every sub-account's deposits when unset
	Status    *DepositStatus
	StartTime time.Time
	EndTime   time.Time
	Offset    uint64
	Limit     uint64 // Up to 1000; 100 when unset
	TxID      string
}

// SubAccountDepositRecordsResponse holds a page of sub-account deposits
type SubAccountDepositRecordsResponse struct {
	Data  []SubAccountDepositRecord `json:"data"`
	Total uint64                    `json:"total"`
}

// SubAccountDepositRecord holds a sub-account deposit
type SubAccountDepositRecord struct {
	SubUID       NumericID     `json:"subUid"`
	Amount       types.Number  `json:"amount"`
	Coin         currency.Code `json:"coin"`
	Network      string        `json:"network"`
	Status       DepositStatus `json:"status"`
	Address      string        `json:"address"`
	AddressTag   string        `json:"addressTag"`
	TxID         string        `json:"txId"`
	InsertTime   types.Time    `json:"insertTime"`
	TransferType TransferType  `json:"transferType"`
}

// SubMotherAccountTransferHistoryRequest holds the parameters of a sub-mother account transfer history query
type SubMotherAccountTransferHistoryRequest struct {
	UID           uint64
	Type          SubMotherAccountTransferType
	TransactionID string
	StartTime     time.Time
	EndTime       time.Time
	PageID        uint64 // From 1
	PagingSize    uint64 // Up to 100; 10 when unset
}

// SubMotherAccountTransferHistoryResponse holds a page of transfers between the parent account and its sub-accounts
type SubMotherAccountTransferHistoryResponse struct {
	Total uint64                     `json:"total"`
	Rows  []SubMotherAccountTransfer `json:"rows"`
}

// SubMotherAccountTransfer holds a transfer between the parent account and a sub-account
type SubMotherAccountTransfer struct {
	Asset         currency.Code                `json:"asset"`
	Amount        types.Number                 `json:"amount"`
	Type          SubMotherAccountTransferType `json:"type"`
	Status        string                       `json:"status"`
	TransactionID string                       `json:"tranId"`
	Timestamp     types.Time                   `json:"timestamp"`
	FromUID       NumericID                    `json:"fromUid"`
	ToUID         NumericID                    `json:"toUid"`
}

// SubMotherAccountTransferableAmountRequest holds the parameters of a sub-mother account transferable amount query,
// sent as a JSON body
type SubMotherAccountTransferableAmountRequest struct {
	FromUID         uint64     `json:"fromUid"`
	FromAccountType WalletType `json:"fromAccountType"`
	ToUID           uint64     `json:"toUid"`
	ToAccountType   WalletType `json:"toAccountType"`
}

// SubMotherAccountTransferableAmountResponse holds the coins that can be transferred between two accounts
type SubMotherAccountTransferableAmountResponse struct {
	Coins []SubMotherAccountTransferableCoin `json:"coins"`
}

// SubMotherAccountTransferableCoin holds the amount of a coin that can be transferred
type SubMotherAccountTransferableCoin struct {
	ID              uint64        `json:"id"`
	Name            currency.Code `json:"name"`
	ShowName        string        `json:"showName"`
	Icon            CoinIcon      `json:"icon"`
	Type            uint64        `json:"type"` // Undocumented
	FiatSymbol      string        `json:"fiatSymbol"`
	AvailableAmount types.Number  `json:"availableAmount"`
	USDTRate        types.Number  `json:"usdtRate"`
}

// CoinIcon holds a coin's icon
type CoinIcon struct {
	ID  uint64 `json:"id"`
	URI string `json:"uri"`
}

// SubMotherAccountAssetTransferRequest holds the parameters of a transfer between the parent account and a
// sub-account, sent as a JSON body
type SubMotherAccountAssetTransferRequest struct {
	AssetName       currency.Code  `json:"assetName"`
	TransferAmount  float64        `json:"transferAmount"`
	FromUID         uint64         `json:"fromUid"`
	FromType        SubAccountType `json:"fromType"`
	FromAccountType WalletType     `json:"fromAccountType"`
	ToUID           uint64         `json:"toUid"`
	ToType          SubAccountType `json:"toType"`
	ToAccountType   WalletType     `json:"toAccountType"`
	Remark          string         `json:"remark"`
}

// SubMotherAccountAssetTransferResponse holds the ID of a transfer between the parent account and a sub-account
type SubMotherAccountAssetTransferResponse struct {
	TransactionID string `json:"tranId"`
}

// SubAccountAssetOverviewRequest holds the parameters of a sub-account asset overview query
type SubAccountAssetOverviewRequest struct {
	SubUID      uint64
	AccountType AccountType
	PageIndex   uint64 // From 1
	PageSize    uint64 // Up to 10
}

// SubAccountAssetOverviewResponse holds a page of sub-account asset overviews
type SubAccountAssetOverviewResponse struct {
	Result []SubAccountAssetOverview `json:"result"`
	PageID uint64                    `json:"pageId"`
	Total  uint64                    `json:"total"`
}

// SubAccountAssetOverview holds the USDT value of each of a sub-account's accounts
type SubAccountAssetOverview struct {
	SubUID          NumericID                `json:"subUid"`
	AccountBalances []AccountBalanceOverview `json:"accountBalances"`
}
