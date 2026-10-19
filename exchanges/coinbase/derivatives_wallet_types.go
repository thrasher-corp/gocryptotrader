package coinbase

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// DerivativesSubmitTransferBetweenSubaccountsRequest transfers funds between subaccounts, or a subaccount and the main
// account, such as collateral topping up an isolated margin subaccount
type DerivativesSubmitTransferBetweenSubaccountsRequest struct {
	// Currency is BTC, ETH, USDC, USDT or EURR
	Currency currency.Code
	Amount   float64
	// Destination and Source are account user IDs; Source defaults to the requesting account, and only a request with
	// the main account's scope may name another subaccount
	Destination uint64
	Source      uint64
	// Nonce, of 8 to 128 characters, makes a repeated request return the transfer it created instead of another
	Nonce string
}

// derivativesSubmitTransferBetweenSubaccountsParams are private/submit_transfer_between_subaccounts' params
type derivativesSubmitTransferBetweenSubaccountsParams struct {
	Currency    string  `json:"currency"`
	Amount      float64 `json:"amount"`
	Destination uint64  `json:"destination"`
	Source      uint64  `json:"source,omitempty"`
	Nonce       string  `json:"nonce,omitempty"`
}

// DerivativesTransfer is a transfer between the account and a subaccount or another user
type DerivativesTransfer struct {
	ID uint64 `json:"id"`
	// Type is user for a transfer to or from another user, and subaccount for one to or from a subaccount
	Type     string        `json:"type"`
	Currency currency.Code `json:"currency"`
	Amount   float64       `json:"amount"`
	// OtherSide is the subaccount's name for a subaccount transfer, the destination address for a transfer to another
	// user and the sender's username for one from another user
	OtherSide string `json:"other_side"`
	// State is prepared, confirmed, cancelled, waiting_for_admin, insufficient_funds, withdrawal_limit or otherwise the
	// reason the transfer was rejected
	State string `json:"state"`
	// Direction is payment or income
	Direction        string     `json:"direction"`
	CreatedTimestamp types.Time `json:"created_timestamp"`
	UpdatedTimestamp types.Time `json:"updated_timestamp"`
	// Nonce is the idempotency nonce the transfer was requested with
	Nonce string `json:"nonce"`
}
