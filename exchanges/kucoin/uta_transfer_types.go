package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UTATransferQuotasRequest holds the parameters of GetUTATransferQuotas
type UTATransferQuotasRequest struct {
	// AccountType is required: FUNDING, SPOT, FUTURES, CROSS, ISOLATED or UNIFIED
	AccountType string
	// Currency is required
	Currency currency.Code
	// Symbol is required for an isolated margin account, such as BTC-USDT
	Symbol string
}

// UTATransferQuotasResponse holds how much of a currency can be transferred out of an account type
type UTATransferQuotasResponse struct {
	AccountType string        `json:"accountType"`
	Currency    currency.Code `json:"currency"`
	// Transferable is the amount that can be transferred out now, which can be less than the balance
	Transferable types.Number `json:"transferable"`
}

// UTAFlexTransferRequest holds the parameters of UTAFlexTransfer
type UTAFlexTransferRequest struct {
	// ClientOrderID is required: a unique ID for the transfer
	ClientOrderID string
	// TransferType is required: INTERNAL, PARENT_TO_SUB, SUB_TO_PARENT or SUB_TO_SUB
	TransferType string
	// Currency is required
	Currency currency.Code
	// Amount is required
	Amount float64
	// FromUID is required for transfer types with a source sub-account
	FromUID string
	// FromAccountType is required: FUNDING, UNIFIED, FUTURES, ISOLATED, CROSS or SPOT
	FromAccountType string
	// FromAccountTag is required: the isolated margin symbol, such as BTC-USDT, when FromAccountType is ISOLATED, and
	// otherwise DEFAULT unless a special account tag applies
	FromAccountTag string
	// ToUID is required for transfer types with a destination sub-account
	ToUID string
	// ToAccountType is required: FUNDING, UNIFIED, FUTURES, ISOLATED, CROSS or SPOT
	ToAccountType string
	// ToAccountTag is required: the isolated margin symbol, such as BTC-USDT, when ToAccountType is ISOLATED, and
	// otherwise DEFAULT unless a special account tag applies
	ToAccountTag string
}

// UTAFlexTransferResponse holds a transfer's order ID
type UTAFlexTransferResponse struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOid"`
}
