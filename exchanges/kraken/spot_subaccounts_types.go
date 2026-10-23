package kraken

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	errSubaccountUsernameRequired  = errors.New("subaccount username required")
	errSubaccountEmailRequired     = errors.New("subaccount email address required")
	errSubaccountAccountIDRequired = errors.New("source and destination account IDs required")
)

// SubaccountTransferRequest holds the parameters of Account Transfer
type SubaccountTransferRequest struct {
	Asset currency.Code
	// AssetClass is currency, the default, or tokenized_asset for xStocks
	AssetClass string
	Amount     float64
	// FromAccountID and ToAccountID are the public IDs of the source and destination accounts, such as ABCD 1234 EFGH
	// 5678
	FromAccountID string
	ToAccountID   string
}

// SubaccountTransferResponse holds an account transfer's ID and status
type SubaccountTransferResponse struct {
	TransferID string `json:"transfer_id"`
	// Status is pending or complete
	Status string `json:"status"`
}
