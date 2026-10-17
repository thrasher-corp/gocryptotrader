package bitfinex

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// thalexProvider is the provider the Thalex endpoints name Thalex by
const thalexProvider = "thalex"

// ThalexTransferRequest holds a transfer between the account and its Thalex derivatives account
type ThalexTransferRequest struct {
	// Currency is USE, which is USDt on Ethereum, ETH or BTC
	Currency currency.Code
	Amount   float64
	// TwoFactorToken validates the transfer, which needs it
	TwoFactorToken *TwoFactorToken
}

// thalexTransferBody is the body of a Thalex Deposit Request or Thalex Withdrawal Request
type thalexTransferBody struct {
	Provider       string          `json:"provider"`
	Amount         string          `json:"amount"`
	Currency       string          `json:"ccy"`
	TwoFactorToken *TwoFactorToken `json:"tfaToken"`
}

// ThalexTransfer is a transfer between the account and its Thalex derivatives account
type ThalexTransfer struct {
	// Type is deposit for a transfer to Thalex and withdraw for one from it
	Type               string        `json:"type"`
	DestinationAddress string        `json:"addressDest"`
	Amount             types.Number  `json:"amount"`
	Currency           currency.Code `json:"ccy"`
	CreatedAt          types.Time    `json:"createdAt"`
	UpdatedAt          types.Time    `json:"updatedAt"`
	ID                 string        `json:"id"`
	// Status is such as PENDING
	Status string       `json:"status"`
	Fee    types.Number `json:"fee"`
}

// thalexProviderBody is the body of a Thalex Free Transfer Count request
type thalexProviderBody struct {
	Provider string `json:"provider"`
}

// ThalexFreeTransferCountResponse is how many free transfers to and from Thalex the account has left
type ThalexFreeTransferCountResponse struct {
	Deposits    ThalexFreeTransfers `json:"deposits"`
	Withdrawals ThalexFreeTransfers `json:"withdrawals"`
}

// ThalexFreeTransfers is how many free transfers of a kind the account has left, and when they renew
type ThalexFreeTransfers struct {
	Available uint64 `json:"available"`
	// ResetsAt is unset when Bitfinex gives no time
	ResetsAt types.Time `json:"resetsAt"`
}
