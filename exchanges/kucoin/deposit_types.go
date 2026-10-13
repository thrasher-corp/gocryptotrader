package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AddDepositAddressV3Request holds the Add Deposit Address (V3) parameters; Currency and Chain are required
type AddDepositAddressV3Request struct {
	Currency currency.Code
	// Chain is the currency's chain ID, such as eth, bech32, btc or trx
	Chain string
	// To is the account deposits are credited to: main (the default) or trade
	To string
	// Amount is only used for Lightning Network invoices
	Amount float64
}

// AddDepositAddressV3Response is the deposit address KuCoin created
type AddDepositAddressV3Response struct {
	Address string `json:"address"`
	// Memo is the address tag a deposit must carry when one is set
	Memo    string `json:"memo"`
	ChainID string `json:"chainId"`
	// To is the account deposits are credited to: MAIN or TRADE
	To string `json:"to"`
	// ExpirationDate is only set for Lightning Network invoices
	ExpirationDate types.Time    `json:"expirationDate"`
	Currency       currency.Code `json:"currency"`
	ChainName      string        `json:"chainName"`
	Remark         string        `json:"remark"`
}

// DepositAddress is a deposit address of a currency (Get Deposit Address (V3))
type DepositAddress struct {
	Address string `json:"address"`
	// Memo is the address tag a deposit must carry when one is set
	Memo    string `json:"memo"`
	ChainID string `json:"chainId"`
	// To is the account deposits are credited to: MAIN or TRADE
	To              string        `json:"to"`
	Currency        currency.Code `json:"currency"`
	ContractAddress string        `json:"contractAddress"`
	ChainName       string        `json:"chainName"`
	Remark          string        `json:"remark"`
	// ExpirationDate is only set for Lightning Network invoices
	ExpirationDate types.Time `json:"expirationDate"`
}

// DepositHistoryRequest holds the Get Deposit History parameters; Currency is required
type DepositHistoryRequest struct {
	Currency currency.Code
	// Status is PROCESSING, SUCCESS, FAILURE, WAIT_TRM_MGT or TRM_MGT_REJECTED
	Status      string
	StartAt     time.Time
	EndAt       time.Time
	CurrentPage uint64
	// PageSize is between 10 and 500
	PageSize uint64
}

// DepositHistoryResponse is a page of deposits, latest first
type DepositHistoryResponse struct {
	CurrentPage uint64          `json:"currentPage"`
	PageSize    uint64          `json:"pageSize"`
	TotalNumber uint64          `json:"totalNum"`
	TotalPage   uint64          `json:"totalPage"`
	Items       []DepositRecord `json:"items"`
}

// DepositRecord is a deposit (Get Deposit History)
type DepositRecord struct {
	ID         string        `json:"id"`
	Currency   currency.Code `json:"currency"`
	Chain      string        `json:"chain"`
	Status     string        `json:"status"`
	SubStatus  string        `json:"subStatus"`
	URL        string        `json:"url"`
	Address    string        `json:"address"`
	Memo       string        `json:"memo"`
	IsInner    bool          `json:"isInner"`
	Amount     types.Number  `json:"amount"`
	Fee        types.Number  `json:"fee"`
	WalletTxID string        `json:"walletTxId"`
	CreatedAt  types.Time    `json:"createdAt"`
	UpdatedAt  types.Time    `json:"updatedAt"`
	Remark     string        `json:"remark"`
	// Arrears is true when a deposit failed after a quick rollback and the balance must be repaid
	Arrears       bool   `json:"arrears"`
	NeedSubmitWHT string `json:"needSubmitWht"`
}
