package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// NFTTransactionHistoryRequest holds the parameters of Get NFT Transaction History
type NFTTransactionHistoryRequest struct {
	// OrderType is 0 purchase order, 1 sell order, 2 royalty income, 3 primary market order or 4 mint fee; it is
	// always sent
	OrderType uint64
	StartTime time.Time
	EndTime   time.Time
	// Limit is at most 50, the default
	Limit uint64
	Page  uint64
}

// NFTTransactionHistoryResponse is a page of NFT transactions
type NFTTransactionHistoryResponse struct {
	Total uint64           `json:"total"`
	List  []NFTTransaction `json:"list"`
}

// NFTTransaction is an NFT purchase, sale, royalty income, primary market order or mint fee
type NFTTransaction struct {
	OrderNumber   string        `json:"orderNo"`
	Tokens        []NFTToken    `json:"tokens"`
	TradeTime     types.Time    `json:"tradeTime"`
	TradeAmount   types.Number  `json:"tradeAmount"`
	TradeCurrency currency.Code `json:"tradeCurrency"`
}

// NFTToken identifies an NFT
type NFTToken struct {
	Network         string `json:"network"`
	TokenID         string `json:"tokenId"`
	ContractAddress string `json:"contractAddress"`
}

// NFTHistoryRequest holds the parameters of Get NFT Deposit History and Get NFT Withdraw History
type NFTHistoryRequest struct {
	StartTime time.Time
	EndTime   time.Time
	// Limit is at most 50, the default
	Limit uint64
	Page  uint64
}

// NFTDepositHistoryResponse is a page of NFT deposits
type NFTDepositHistoryResponse struct {
	Total uint64       `json:"total"`
	List  []NFTDeposit `json:"list"`
}

// NFTDeposit is an NFT deposit; ContractAdrress keeps the API's spelling
type NFTDeposit struct {
	Network         string     `json:"network"`
	TransactionID   string     `json:"txID"`
	ContractAdrress string     `json:"contractAdrress"`
	TokenID         string     `json:"tokenId"`
	Timestamp       types.Time `json:"timestamp"`
}

// NFTWithdrawalHistoryResponse is a page of NFT withdrawals
type NFTWithdrawalHistoryResponse struct {
	Total uint64          `json:"total"`
	List  []NFTWithdrawal `json:"list"`
}

// NFTWithdrawal is an NFT withdrawal; ContractAdrress keeps the API's spelling
type NFTWithdrawal struct {
	Network         string        `json:"network"`
	TransactionID   string        `json:"txID"`
	ContractAdrress string        `json:"contractAdrress"`
	TokenID         string        `json:"tokenId"`
	Timestamp       types.Time    `json:"timestamp"`
	Fee             float64       `json:"fee"`
	FeeAsset        currency.Code `json:"feeAsset"`
}

// NFTAssetsResponse is a page of the account's NFTs
type NFTAssetsResponse struct {
	Total uint64     `json:"total"`
	List  []NFTToken `json:"list"`
}
