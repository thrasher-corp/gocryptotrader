package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/types"
)

// BasicFeeResponse holds the account's basic spot and margin fee rates
type BasicFeeResponse struct {
	TakerFeeRate types.Number `json:"takerFeeRate"`
	MakerFeeRate types.Number `json:"makerFeeRate"`
}

// TradingPairActualFeeResponse holds the actual fee and tax rates of a spot or margin symbol
type TradingPairActualFeeResponse struct {
	Symbol       string       `json:"symbol"`
	TakerFeeRate types.Number `json:"takerFeeRate"`
	MakerFeeRate types.Number `json:"makerFeeRate"`
	SellTaxRate  types.Number `json:"sellTaxRate"` // Only shown to users in some countries
	BuyTaxRate   types.Number `json:"buyTaxRate"`  // Only shown to users in some countries
}

// FuturesTradingPairsActualFeesResponse holds the actual fee rates of a futures symbol
type FuturesTradingPairsActualFeesResponse struct {
	Symbol       string       `json:"symbol"`
	TakerFeeRate types.Number `json:"takerFeeRate"`
	MakerFeeRate types.Number `json:"makerFeeRate"`
}
