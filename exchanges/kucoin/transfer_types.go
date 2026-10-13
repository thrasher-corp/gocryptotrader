package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// TransferQuotasRequest holds the Get Transfer Quotas parameters; Currency and AccountType are required
type TransferQuotasRequest struct {
	Currency currency.Code
	// AccountType is MAIN, TRADE, MARGIN, ISOLATED, MARGIN_V2 or ISOLATED_V2; the V2 types are being phased out
	AccountType string
	// Tag is the symbol of an ISOLATED account, such as BTC-USDT
	Tag string
}

// TransferQuotasResponse is the transferable balance of an account
type TransferQuotasResponse struct {
	Currency currency.Code `json:"currency"`
	Balance  types.Number  `json:"balance"`
	// Available is being deprecated and should be ignored; Balance holds the available quota
	Available    types.Number `json:"available"`
	Holds        types.Number `json:"holds"`
	Transferable types.Number `json:"transferable"`
}

// FlexTransferRequest holds the Flex Transfer parameters; ClientOrderID, Currency, Amount, FromAccountType, Type and
// ToAccountType are required
type FlexTransferRequest struct {
	ClientOrderID string
	Currency      currency.Code
	// Amount is a multiple of the currency's precision
	Amount float64
	// FromUserID is required to transfer from a sub-account to the master account
	FromUserID string
	// FromAccountType is MAIN, TRADE, CONTRACT, MARGIN, ISOLATED, MARGIN_V2, ISOLATED_V2 or UNIFIED; the V2 types are
	// only accepted for INTERNAL transfers
	FromAccountType string
	// FromAccountTag is the symbol of an ISOLATED or ISOLATED_V2 account, such as BTC-USDT
	FromAccountTag string
	// Type is INTERNAL, PARENT_TO_SUB, SUB_TO_PARENT or SUB_TO_SUB
	Type string
	// ToUserID is required to transfer from the master account to a sub-account
	ToUserID string
	// ToAccountType takes the values of FromAccountType
	ToAccountType string
	// ToAccountTag is the symbol of an ISOLATED or ISOLATED_V2 account, such as BTC-USDT
	ToAccountTag string
}

// FlexTransferResponse identifies the transfer KuCoin accepted
type FlexTransferResponse struct {
	OrderID string `json:"orderId"`
}
