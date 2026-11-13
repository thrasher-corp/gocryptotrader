package bitstamp

import (
	"github.com/thrasher-corp/gocryptotrader/types"
)

// DepositAddressResponse holds a cryptocurrency deposit address
type DepositAddressResponse struct {
	Address        string `json:"address"`
	MemoID         string `json:"memo_id"`
	DestinationTag uint64 `json:"destination_tag"`
	TransferID     uint64 `json:"transfer_id"`
}

// UnconfirmedDepositResponse holds an unconfirmed bitcoin deposit
type UnconfirmedDepositResponse struct {
	Amount         types.Number `json:"amount"`
	Address        string       `json:"address"`
	Confirmations  uint64       `json:"confirmations"`
	MemoID         string       `json:"memo_id"`
	DestinationTag uint64       `json:"destination_tag"`
	TransferID     uint64       `json:"transfer_id"`
}

// RippleIOUDepositAddressResponse holds a Ripple IOU deposit address
type RippleIOUDepositAddressResponse struct {
	Address        string `json:"address"`
	DestinationTag uint64 `json:"destination_tag"`
}

// unconfirmedDeposits decodes unconfirmed deposits sent as either a list or a single object
type unconfirmedDeposits []UnconfirmedDepositResponse
