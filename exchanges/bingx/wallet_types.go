package bingx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// DepositRecordsRequest holds the parameters of a deposit records query
type DepositRecordsRequest struct {
	Coin      currency.Code
	Status    *DepositStatus
	StartTime time.Time
	EndTime   time.Time
	Offset    uint64
	Limit     uint64 // Up to 1000; 1000 when unset
	TxID      string
}

// DepositRecordResponse holds a deposit
type DepositRecordResponse struct {
	Amount        types.Number  `json:"amount"`
	Coin          currency.Code `json:"coin"`
	Network       string        `json:"network"`
	Status        DepositStatus `json:"status"`
	Address       string        `json:"address"`
	AddressTag    string        `json:"addressTag"`
	TxID          string        `json:"txId"`
	InsertTime    types.Time    `json:"insertTime"`
	TransferType  TransferType  `json:"transferType"`
	UnlockConfirm string        `json:"unlockConfirm"` // Confirmations to unlock the deposit, such as 2/2
	ConfirmTimes  string        `json:"confirmTimes"`  // Network confirmations, such as 2/2
	SourceAddress string        `json:"sourceAddress"`
}

// WithdrawRecordsRequest holds the parameters of a withdrawal records query
type WithdrawRecordsRequest struct {
	ID              string // Platform withdrawal ID, which takes precedence over WithdrawOrderID
	Coin            currency.Code
	WithdrawOrderID string
	Status          WithdrawStatus
	StartTime       time.Time
	EndTime         time.Time
	Offset          uint64
	Limit           uint64 // Up to 1000; 1000 when unset
	TxID            string
}

// WithdrawRecordResponse holds a withdrawal
type WithdrawRecordResponse struct {
	Address         string         `json:"address"`
	Amount          types.Number   `json:"amount"`
	ApplyTime       time.Time      `json:"applyTime"`
	Coin            currency.Code  `json:"coin"`
	ID              string         `json:"id"`
	WithdrawOrderID string         `json:"withdrawOrderId"`
	Network         string         `json:"network"`
	Status          WithdrawStatus `json:"status"`
	TransactionFee  types.Number   `json:"transactionFee"`
	ConfirmNumber   uint64         `json:"confirmNo"`
	Info            string         `json:"info"` // Reason for a failed withdrawal
	TxID            string         `json:"txId"`
	SourceAddress   string         `json:"sourceAddress"`
	TransferType    TransferType   `json:"transferType"`
	AddressTag      string         `json:"addressTag"`
}

// CurrencyConfigResponse holds a coin's deposit and withdrawal configuration on each of its networks
type CurrencyConfigResponse struct {
	Coin        currency.Code     `json:"coin"`
	DisplayName string            `json:"displayName"`
	Name        string            `json:"name"`
	NetworkList []CurrencyNetwork `json:"networkList"`
}

// CurrencyNetwork holds a coin's deposit and withdrawal configuration on a network
type CurrencyNetwork struct {
	Name                string        `json:"name"`
	Network             string        `json:"network"`
	IsDefault           bool          `json:"isDefault"`
	MinimumConfirm      uint64        `json:"minConfirm"` // Block confirmations a deposit needs
	WithdrawEnable      bool          `json:"withdrawEnable"`
	DepositEnable       bool          `json:"depositEnable"`
	WithdrawFee         types.Number  `json:"withdrawFee"`
	WithdrawMaximum     types.Number  `json:"withdrawMax"`
	WithdrawMinimum     types.Number  `json:"withdrawMin"`
	DepositMinimum      types.Number  `json:"depositMin"`
	WithdrawDescription string        `json:"withdrawDesc"`
	WithdrawPrecision   uint64        `json:"withdrawPrecision"`
	DepositPrecision    uint64        `json:"depositPrecision"`
	ContractAddress     string        `json:"contractAddress"`
	NeedTagOrMemo       types.Boolean `json:"needTagOrMemo"`
}

// WithdrawRequest holds the parameters of a withdrawal. Korean individual users must also set the VASP entity and,
// unless it is VASPEntityIDOthers, the recipient's names and date of birth
type WithdrawRequest struct {
	Coin               currency.Code
	Network            string // The coin's default network when unset
	Address            string
	AddressTag         string // Tag or memo, for coins that take one
	Amount             float64
	WalletType         WalletType // Account the withdrawal is paid from
	WithdrawOrderID    string
	VASPEntityID       VASPEntityID
	RecipientLastName  string
	RecipientFirstName string
	DateOfBirth        time.Time
}

// WithdrawResponse holds the IDs of a withdrawal
type WithdrawResponse struct {
	ID              string `json:"id"`
	WithdrawOrderID string `json:"withdrawOrderId"`
}

// MainAccountDepositAddressRequest holds the parameters of a main account deposit address query
type MainAccountDepositAddressRequest struct {
	Coin   currency.Code
	Offset uint64
	Limit  uint64 // Up to 1000; 100 when unset
}

// DepositAddressListResponse holds a page of deposit addresses
type DepositAddressListResponse struct {
	Data  []DepositAddress `json:"data"`
	Total uint64           `json:"total"`
}

// DepositAddress holds a deposit address of a coin on a network
type DepositAddress struct {
	CoinID            uint64               `json:"coinId"`
	Coin              currency.Code        `json:"coin"`
	Network           string               `json:"network"`
	Address           string               `json:"address"`
	AddressWithPrefix string               `json:"addressWithPrefix"`
	Tag               string               `json:"tag"`
	Status            DepositAddressStatus `json:"status"`
	WalletType        WalletType           `json:"walletType"`
}

// DepositRiskControlRecordResponse holds a deposit held by risk control
type DepositRiskControlRecordResponse struct {
	UID           NumericID     `json:"uid"`
	Coin          currency.Code `json:"coin"`
	Amount        types.Number  `json:"amount"`
	SourceAddress string        `json:"sourceAddress"`
	Address       string        `json:"address"`
	InsetTime     string        `json:"insetTime"` // Creation time, whose format the documentation does not show
}
