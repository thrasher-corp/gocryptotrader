package bitfinex

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Wallet is a wallet's balances
type Wallet struct {
	// Type is exchange, margin or funding
	Type     string        `tuple:"0"`
	Currency currency.Code `tuple:"1"`
	Balance  float64       `tuple:"2"`
	// UnsettledInterest is funding interest not yet paid into or out of the wallet
	UnsettledInterest float64 `tuple:"3"`
	// AvailableBalance is what orders, withdrawals and transfers can use; it is nil until Bitfinex has calculated it,
	// which a websocket wallet event can precede
	AvailableBalance *float64 `tuple:"4"`
	// LastChange describes the wallet's last ledger entry
	LastChange         string                `tuple:"5"`
	LastChangeMetadata *WalletChangeMetadata `tuple:"6"`
}

// UnmarshalJSON decodes a wallet array
func (w *Wallet) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, w) }

// WalletChangeMetadata details a wallet's last ledger entry; which fields are set depends on its reason
type WalletChangeMetadata struct {
	// Reason is the cause of the change, such as TRADE
	Reason          string       `json:"reason"`
	OrderID         uint64       `json:"order_id"`
	OppositeOrderID uint64       `json:"order_id_oppo"`
	TradePrice      types.Number `json:"trade_price"`
	TradeAmount     types.Number `json:"trade_amount"`
	OrderClientID   uint64       `json:"order_cid"`
	OrderGroupID    uint64       `json:"order_gid"`
}
