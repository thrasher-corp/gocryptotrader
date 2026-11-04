package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WebsocketMarginIndexPrice is a margin Index Price push
type WebsocketMarginIndexPrice struct {
	Symbol      string     `json:"symbol"`
	Granularity uint64     `json:"granularity"` // Milliseconds
	Timestamp   types.Time `json:"timestamp"`
	Value       float64    `json:"value"`
}

// WebsocketMarginMarkPrice is a margin Mark Price push
type WebsocketMarginMarkPrice WebsocketMarginIndexPrice

// WebsocketCrossMarginPosition is a Cross Margin Position push: a debt.ratio push carries the debt and the assets, a
// position.status push the event type
type WebsocketCrossMarginPosition struct {
	DebtRatio                   float64                                     `json:"debtRatio"`
	TotalAsset                  float64                                     `json:"totalAsset"`
	MarginCoefficientTotalAsset types.Number                                `json:"marginCoefficientTotalAsset"`
	TotalDebt                   types.Number                                `json:"totalDebt"`
	AssetList                   map[currency.Code]WebsocketCrossMarginAsset `json:"assetList"`
	DebtList                    map[currency.Code]types.Number              `json:"debtList"`
	Timestamp                   types.Time                                  `json:"timestamp"`
	Type                        string                                      `json:"type"`
}

// WebsocketCrossMarginAsset is a currency's balance in a Cross Margin Position push
type WebsocketCrossMarginAsset struct {
	Total     types.Number `json:"total"`
	Available types.Number `json:"available"`
	Hold      types.Number `json:"hold"`
}

// WebsocketIsolatedMarginPosition is an Isolated Margin Position push
type WebsocketIsolatedMarginPosition struct {
	Tag                  string                                         `json:"tag"`
	Status               string                                         `json:"status"`
	StatusBizType        string                                         `json:"statusBizType"`
	AccumulatedPrincipal types.Number                                   `json:"accumulatedPrincipal"`
	ChangeAssets         map[currency.Code]WebsocketIsolatedMarginAsset `json:"changeAssets"`
	Timestamp            types.Time                                     `json:"timestamp"`
}

// WebsocketIsolatedMarginAsset is a currency's balance in an Isolated Margin Position push
type WebsocketIsolatedMarginAsset struct {
	Total              types.Number `json:"total"`
	Hold               types.Number `json:"hold"`
	LiabilityPrincipal types.Number `json:"liabilityPrincipal"`
	LiabilityInterest  types.Number `json:"liabilityInterest"`
}
