package starkex

import (
	"errors"
	"math/big"
)

var (
	ErrExpirationTimeRequired         = errors.New("expiration time is required")
	ErrContractNotFound               = errors.New("contract not found")
	ErrSettlementCurrencyInfoNotFound = errors.New("settlement currency information not found")
	ErrInvalidAssetID                 = errors.New("invalid asset ID provided")
	ErrInvalidPositionIDMissing       = errors.New("invalid position or account ID")
)

// TransferParams represents a starkex asset transfer parameters. Type value: 4
type TransferParams struct {
	AssetID              *big.Int `json:"asset_id"`
	AssetIDFee           *big.Int `json:"asset_id_fee"` // asset ID of
	SenderPositionID     *big.Int `json:"sender_position_id"`
	ReceiverPositionID   *big.Int `json:"receiver_position_id"`
	Nonce                *big.Int `json:"nonce"`
	QuantumsAmount       *big.Int `json:"amount"`
	ExpirationEpochHours *big.Int `json:"expiration_timestamp"`
	ReceiverPublicKey    *big.Int `json:"receiver_public_key"`
	MaxAmountFee         *big.Int `json:"max_amount_fee"`
	SrcFeePositionID     *big.Int `json:"src_fee_position_id"`
}

// ConditionalTransferParams represents a conditional transfer parameters. Type value: 5
type ConditionalTransferParams struct {
	QuantumsAmount     *big.Int `json:"amount"`
	AssetID            *big.Int `json:"asset_id"`
	ExpTimestampHrs    *big.Int `json:"expiration_timestamp"`
	Nonce              *big.Int `json:"nonce"`
	ReceiverPositionID *big.Int `json:"receiver_position_id"`
	ReceiverPublicKey  *big.Int `json:"receiver_public_key"`
	SenderPositionID   *big.Int `json:"sender_position_id"`
	SenderPublicKey    *big.Int `json:"sender_public_key"`
	MaxAmountFee       *big.Int `json:"max_amount_fee"`
	AssetIDFee         *big.Int `json:"asset_id_fee"`
	SrcFeePositionID   *big.Int `json:"src_fee_position_id"`
	Condition          *big.Int `json:"condition"`
}

// CreateOrderWithFeeParams represents a starkex create order parameters. Order Prefix: 3
type CreateOrderWithFeeParams struct {
	OrderType               string
	AssetIDSynthetic        *big.Int
	AssetIDCollateral       *big.Int
	AssetIDFee              *big.Int
	QuantumAmountSynthetic  *big.Int
	QuantumAmountCollateral *big.Int
	QuantumAmountFee        *big.Int
	IsBuyingSynthetic       bool
	PositionID              *big.Int
	Nonce                   *big.Int
	ExpirationEpochHours    *big.Int
}

// WithdrawalToAddressParams represents a starkex withdrawal to address parameters. Type value: 7.
type WithdrawalToAddressParams struct {
	NetworkID            int64    `json:"-"`
	AssetIDCollateral    *big.Int `json:"asset_id_collateral"`
	EthAddress           *big.Int `json:"eth_address"`
	PositionID           *big.Int `json:"position_id"`
	Amount               *big.Int `json:"amount"`
	Nonce                *big.Int `json:"nonce"`
	ExpirationEpochHours *big.Int `json:"expiration_timestamp"`
}

// WithdrawalParams represents a starkex withdrawal parameters. Type value: 6.
type WithdrawalParams struct {
	NetworkID            int64    `json:"-"`
	AssetIDCollateral    *big.Int `json:"asset_id_collateral"`
	PositionID           *big.Int `json:"position_id"`
	Amount               *big.Int `json:"amount"`
	Nonce                *big.Int `json:"nonce"`
	ExpirationEpochHours *big.Int `json:"expiration_timestamp"`
}
