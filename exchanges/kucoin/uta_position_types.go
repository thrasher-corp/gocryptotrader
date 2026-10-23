package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UTAPositionListRequest holds the parameters of Get Position List
type UTAPositionListRequest struct {
	Symbol     string
	PageNumber uint64
	PageSize   uint64
}

// UTAPositionResponse holds an open futures position of the unified trading account
type UTAPositionResponse struct {
	Symbol     string `json:"symbol"`
	ID         string `json:"id"`
	MarginMode string `json:"marginMode"`
	// Size is in contracts: positive for a long position and negative for a short one
	Size       types.Number `json:"size"`
	EntryPrice types.Number `json:"entryPrice"`
	// PositionValue is in the settlement currency
	PositionValue types.Number `json:"positionValue"`
	MarkPrice     types.Number `json:"markPrice"`
	Leverage      types.Number `json:"leverage"`
	UnrealisedPNL types.Number `json:"unrealizedPnL"`
	// RealisedPNL accumulates partial closes, funding and fees
	RealisedPNL           types.Number `json:"realizedPnL"`
	InitialMargin         types.Number `json:"initialMargin"`
	MaintenanceMarginRate types.Number `json:"mmr"`
	MaintenanceMargin     types.Number `json:"maintenanceMargin"`
	CreationTime          types.Time   `json:"creationTime"`
	LiquidationPrice      types.Number `json:"liquidationPrice"`
	// ADLPercentage is the position's auto-deleveraging ranking percentile: 0.12 ranks it at 12%, and a higher rank
	// is deleveraged sooner
	ADLPercentage types.Number `json:"adlPercentage"`
	// RiskRatio is an isolated position's risk ratio, at which 1 liquidates it; Get Account Overview (UTA) holds the
	// cross margin one
	RiskRatio types.Number `json:"riskRatio"`
	// UpdateTime is the last change of the position's size or leverage
	UpdateTime types.Time `json:"updateTime"`
	// PositionMargin is the position value at the mark price over the leverage in cross margin mode, and the margin
	// allocated to the position in isolated margin mode
	PositionMargin types.Number `json:"positionMargin"`
}

// UTAPositionsHistoryRequest holds the parameters of Get Positions History. KuCoin keeps three months of history and
// limits each request to a 7 day window
type UTAPositionsHistoryRequest struct {
	Symbol  string
	StartAt time.Time
	EndAt   time.Time
	// LastID is the previous page's LastID, to fetch the page after it
	LastID   uint64
	PageSize uint64
}

// UTAPositionsHistoryResponse holds a page of the unified trading account's closed futures positions
type UTAPositionsHistoryResponse struct {
	Items []UTAClosedPosition `json:"items"`
	// LastID is the CloseID of the page's last position, the cursor of the next page
	LastID uint64 `json:"lastId"`
}

// UTAClosedPosition holds a closed futures position of the unified trading account
type UTAClosedPosition struct {
	CloseID    string `json:"closeId"`
	Symbol     string `json:"symbol"`
	MarginMode string `json:"marginMode"`
	// Side is LONG or SHORT
	Side              string       `json:"side"`
	EntryPrice        types.Number `json:"entryPrice"`
	ClosePrice        types.Number `json:"closePrice"`
	AverageClosePrice types.Number `json:"avgClosePrice"`
	MaxSize           types.Number `json:"maxSize"`
	Leverage          types.Number `json:"leverage"`
	// RealisedPNL is net of fees and funding
	RealisedPNL  types.Number `json:"realizedPnL"`
	Fee          types.Number `json:"fee"`
	Tax          types.Number `json:"tax"`
	FundingFee   types.Number `json:"fundingFee"`
	CreationTime types.Time   `json:"creationTime"`
	ClosingTime  types.Time   `json:"closingTime"`
}

// UTAPrivateFundingFeeHistoryRequest holds the parameters of Get Private Funding Fee History. The window between
// StartAt and EndAt cannot exceed 90 days
type UTAPrivateFundingFeeHistoryRequest struct {
	Symbol  string
	StartAt time.Time
	EndAt   time.Time
	// LastID is the previous page's LastID, to fetch the page after it; without it KuCoin returns the latest records
	LastID uint64
	// PageSize is 50 by default and 200 at most
	PageSize uint64
}

// UTAPrivateFundingFeeHistoryResponse holds a page of the unified trading account's settled funding fees
type UTAPrivateFundingFeeHistoryResponse struct {
	// LastID is the cursor of the next page
	LastID uint64                `json:"lastId"`
	Items  []UTAFundingFeeRecord `json:"items"`
}

// UTAFundingFeeRecord holds a funding fee settlement of a unified trading account futures position
type UTAFundingFeeRecord struct {
	Symbol      string       `json:"symbol"`
	MarginMode  string       `json:"marginMode"`
	FundingRate types.Number `json:"fundingRate"`
	MarkPrice   types.Number `json:"markPrice"`
	Size        types.Number `json:"size"`
	// PositionValue is at the mark price
	PositionValue types.Number `json:"positionValue"`
	// FundingFee is positive when the position received funding and negative when it paid
	FundingFee     types.Number  `json:"fundingFee"`
	SettleCurrency currency.Code `json:"settleCurrency"`
	SettlementTime types.Time    `json:"settlementTime"`
}

// ModifyUTAMarginModeResponse holds the outcome of each symbol of a Modify Margin Mode request
type ModifyUTAMarginModeResponse struct {
	Timestamp types.Time            `json:"ts"`
	Items     []UTAMarginModeResult `json:"items"`
}

// UTAMarginModeResult is the outcome of changing one symbol's margin mode: a Code other than 200000 means KuCoin did
// not change it, and Message says why
type UTAMarginModeResult struct {
	Message    string `json:"msg"`
	Code       string `json:"code"`
	Symbol     string `json:"symbol"`
	MarginMode string `json:"marginMode"`
}

// ModifyUTAPositionMarginRequest holds the parameters of Modify Position Margin
type ModifyUTAPositionMarginRequest struct {
	// Type is DEPOSIT to add margin or WITHDRAW to remove it
	Type   string
	Amount float64
	Symbol string
	// TradeType is FUTURES, the only trade type KuCoin supports here
	TradeType string
}

// ModifyUTAPositionMarginResponse holds the time KuCoin changed an isolated position's margin
type ModifyUTAPositionMarginResponse struct {
	Timestamp types.Time `json:"ts"`
}

// UTAMarginModeResponse holds the margin mode of the unified trading account's futures symbols
type UTAMarginModeResponse struct {
	Timestamp types.Time            `json:"ts"`
	Items     []UTASymbolMarginMode `json:"items"`
}

// UTASymbolMarginMode holds a futures symbol's margin mode, ISOLATED or CROSS
type UTASymbolMarginMode struct {
	Symbol     string `json:"symbol"`
	MarginMode string `json:"marginMode"`
}
