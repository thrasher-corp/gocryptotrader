package binanceus

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// ConvertDustResponse holds the result of a dust conversion
type ConvertDustResponse struct {
	TotalTransferred   types.Number          `json:"totalTransferred"`
	TotalServiceCharge types.Number          `json:"totalServiceCharge"`
	TransferResult     []DustAssetConversion `json:"transferResult"`
}

// DustAssetConversion holds the conversion of one asset's dust
type DustAssetConversion struct {
	TransactionID       uint64        `json:"tranId"`
	FromAsset           currency.Code `json:"fromAsset"`
	ToAsset             currency.Code `json:"toAsset"`
	Amount              types.Number  `json:"amount"`
	TransferredAmount   types.Number  `json:"transferredAmount"`
	ServiceChargeAmount types.Number  `json:"serviceChargeAmount"`
	OperateTime         types.Time    `json:"operateTime"`
	Result              string        `json:"result"` // Sent by ConvertDust only
}

// ConvertDustHistoryResponse holds the account's dust conversions
type ConvertDustHistoryResponse struct {
	Total                  uint64           `json:"total"`
	UserDustConvertHistory []DustConversion `json:"userDustConvertHistory"`
}

// DustConversion holds a dust conversion and the conversion of each asset in it
type DustConversion struct {
	OperateTime              types.Time            `json:"operateTime"`
	TotalServiceChargeAmount types.Number          `json:"totalServiceChargeAmount"`
	TotalTransferredAmount   types.Number          `json:"totalTransferredAmount"`
	UserAssetDribbletDetails []DustAssetConversion `json:"userAssetDribbletDetails"`
}

// ConvertibleAssetsResponse holds the account's dust that can be converted to an asset, and what converting it would
// receive
type ConvertibleAssetsResponse struct {
	ConvertibleAssets           []ConvertibleAsset `json:"convertibleAssets"`
	TotalConvertedAsset         types.Number       `json:"totalConvertedAsset"`
	USDValueTotalConvertedAsset types.Number       `json:"usdValueTotalConvertedAsset"`
	TotalConversionFee          types.Number       `json:"totalConversionFee"`
	TotalReceivedAsset          types.Number       `json:"totalReceivedAsset"`
	FeeRate                     types.Number       `json:"feeRate"`
	WithinRestrictedTime        bool               `json:"withinRestrictedTime"`
}

// ConvertibleAsset holds an asset's dust that can be converted, and what converting it would receive
type ConvertibleAsset struct {
	FromAsset        currency.Code `json:"fromAsset"`
	ToAsset          currency.Code `json:"toAsset"`
	AvailableBalance types.Number  `json:"availableBalance"`
	// ConvertedAsset is the amount of ToAsset the dust converts to before the conversion fee, and ReceivedAsset the
	// amount after it
	ConvertedAsset         types.Number `json:"convertedAsset"`
	USDValueConvertedAsset types.Number `json:"usdValueConvertedAsset"`
	ConversionFee          types.Number `json:"conversionFee"`
	ReceivedAsset          types.Number `json:"receivedAsset"`
}
