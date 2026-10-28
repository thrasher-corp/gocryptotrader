package lbank

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// PairInfoResponse holds a pair's precision and minimum order sizes
type PairInfoResponse struct {
	Symbol string `json:"symbol"`
	// QuantityDecimals and PriceDecimals are the number of decimal places an order's quantity and price may have
	QuantityDecimals   types.Number `json:"quantityAccuracy"`
	PriceDecimals      types.Number `json:"priceAccuracy"`
	MinimumQuantity    types.Number `json:"minTranQua"`
	MinimumOrderAmount types.Number `json:"minOrderAmount"`
}

// AssetConfigResponse holds a coin's deposit and withdrawal settings on one chain
type AssetConfigResponse struct {
	AssetCode    currency.Code `json:"assetCode"`
	ChainName    string        `json:"chainName"`
	CanWithdraw  bool          `json:"canDraw"`
	CanTransfer  bool          `json:"canStationDraw"`
	CanDeposit   bool          `json:"canDeposit"`
	HasMemo      bool          `json:"hasMemo"`
	ContractInfo string        `json:"contractInfo"`
	AssetFee     AssetFee      `json:"assetFee"`
}

// AssetFee holds a coin's withdrawal, transfer and deposit fees and limits on one chain
type AssetFee struct {
	// Type is 1 for a fixed fee, 2 for a rate and 3 for a fixed fee plus a rate
	Type    uint64        `json:"type"`
	FeeCode currency.Code `json:"feeCode"`
	// WithdrawalDecimals and TransferDecimals are the number of decimal places a withdrawal's and an internal transfer's
	// amount may have
	WithdrawalDecimals    uint64       `json:"scale"`
	MinimumWithdrawal     types.Number `json:"minAmt"`
	FeeAmount             types.Number `json:"feeAmt"`
	FeeRate               types.Number `json:"feeRate"`
	TransferFeeAmount     types.Number `json:"stationFeeAmt"`
	TransferDecimals      uint64       `json:"stationScale"`
	MinimumTransferAmount types.Number `json:"stationMinAmt"`
	MinimumDepositAmount  types.Number `json:"minDepositAmt"`
	DepositFee            types.Number `json:"depositFee"`
}
