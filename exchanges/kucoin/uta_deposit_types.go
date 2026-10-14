package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UTADepositAddress holds a currency's deposit address on a chain
type UTADepositAddress struct {
	Currency currency.Code `json:"currency"`
	// Chain is the chain's identifier used in requests, such as trx
	Chain string `json:"chain"`
	// ChainName is the chain's display name, such as TRC20
	ChainName string `json:"chainName"`
	Address   string `json:"address"`
	Memo      string `json:"memo"`
	Remark    string `json:"remark"`
	// ToAccountType is the account deposits are credited to: FUNDING, UNIFIED, FUTURES, ISOLATED, CROSS or SPOT,
	// depending on the account mode, currency and chain
	ToAccountType   string `json:"toAccountType"`
	ContractAddress string `json:"contractAddress"`
	// ExpirationDate is zero when the address does not expire
	ExpirationDate types.Time `json:"expirationDate"`
}

// UTAAssetHistoryRequest holds the parameters of GetUTADepositHistory and GetUTAWithdrawalHistory; every field is
// optional
type UTAAssetHistoryRequest struct {
	Currency currency.Code
	// ID selects one record
	ID string
	// Status is FAILURE, PRE_SUCCESS, PROCESSING, RISK_MGT_REJECTED, ROLLBACK, ROLLBACKING, SUCCESS, TRM_MGT_REJECTED,
	// WAIT_RISK_MGT or WAIT_TRM_MGT
	Status      string
	StartAt     time.Time
	EndAt       time.Time
	CurrentPage uint64
	PageSize    uint64
}

// UTADepositHistoryResponse holds a page of deposit records
type UTADepositHistoryResponse struct {
	CurrentPage uint64             `json:"currentPage"`
	PageSize    uint64             `json:"pageSize"`
	TotalNumber uint64             `json:"totalNum"`
	TotalPage   uint64             `json:"totalPage"`
	Items       []UTADepositRecord `json:"items"`
}

// UTADepositRecord holds a deposit record
type UTADepositRecord struct {
	ID       string        `json:"id"`
	Currency currency.Code `json:"currency"`
	// Chain is the chain's identifier used in requests, such as trx
	Chain   string `json:"chain"`
	Status  string `json:"status"`
	Address string `json:"address"`
	Memo    string `json:"memo"`
	// IsInner is true for an internal transfer
	IsInner             bool         `json:"isInner"`
	Amount              types.Number `json:"amount"`
	Fee                 types.Number `json:"fee"`
	WalletTransactionID string       `json:"walletTxId"`
	CreatedAt           types.Time   `json:"createdAt"`
	UpdatedAt           types.Time   `json:"updatedAt"`
	Remark              string       `json:"remark"`
	// Arrears is true when an on-chain rollback may leave the account in debt
	Arrears bool `json:"arrears"`
	// URL is the deposit transaction's blockchain explorer URL
	URL          string `json:"url"`
	StatusRemark string `json:"statusRemark"`
	// PreConfirms is the number of confirmations needed to credit the deposit early
	PreConfirms uint64 `json:"preConfirms"`
	// Confirms is the number of confirmations needed to credit the deposit finally
	Confirms        uint64 `json:"confirms"`
	CurrentConfirms uint64 `json:"currentConfirms"`
}
