package gateio

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WithdrawalRequest represents currency withdrawal request param.
type WithdrawalRequest struct {
	Currency        currency.Code `json:"currency"`
	Amount          types.Number  `json:"amount"`
	Chain           string        `json:"chain"`
	AssetClass      string        `json:"asset_class,omitempty"`
	WithdrawID      string        `json:"withdraw_id,omitempty"`
	WithdrawOrderID string        `json:"withdraw_order_id,omitempty"`
	Address         string        `json:"address,omitempty"`
	Memo            string        `json:"memo,omitempty"`
}

// UIDTransferRequest holds request parameters for transferring funds to another main account by UID
type UIDTransferRequest struct {
	RecipientUserID uint64        `json:"receive_uid"`
	Currency        currency.Code `json:"currency"`
	Amount          types.Number  `json:"amount"`
}

// UIDTransferResponse holds the order ID of an accepted UID transfer
type UIDTransferResponse struct {
	ID string `json:"id"`
}
