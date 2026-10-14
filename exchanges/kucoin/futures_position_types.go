package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FuturesMarginModeResponse holds a contract's margin mode
type FuturesMarginModeResponse struct {
	Symbol string `json:"symbol"`
	// MarginMode is ISOLATED or CROSS
	MarginMode string `json:"marginMode"`
}

// BatchSwitchFuturesMarginModeResponse holds the outcome of a Batch Switch Margin Mode request
type BatchSwitchFuturesMarginModeResponse struct {
	// MarginMode maps each requested symbol to its margin mode after the request, including the symbols KuCoin could
	// not switch
	MarginMode map[string]string `json:"marginMode"`
	// Errors lists the symbols KuCoin could not switch
	Errors []FuturesMarginModeError `json:"errors"`
}

// FuturesMarginModeError holds why KuCoin could not switch a symbol's margin mode
type FuturesMarginModeError struct {
	Code    string `json:"code"`
	Message string `json:"msg"`
	Symbol  string `json:"symbol"`
}

// FuturesPositionModeResponse holds the account's position mode, which applies to every contract
type FuturesPositionModeResponse struct {
	// PositionMode is 0 for one-way mode and 1 for hedge mode. Get Position Mode sends it as a JSON number and Switch
	// Position Mode as a string
	PositionMode types.Number `json:"positionMode"`
}

// FuturesMaxOpenSizeResponse holds the largest positions a contract can open at a price and leverage
type FuturesMaxOpenSizeResponse struct {
	Symbol string `json:"symbol"`
	// MaxBuyOpenSize and MaxSellOpenSize are in lots: the largest long and short in hedge mode, and the largest buy
	// and sell in one-way mode
	MaxBuyOpenSize  uint64 `json:"maxBuyOpenSize"`
	MaxSellOpenSize uint64 `json:"maxSellOpenSize"`
}

// FuturesPositionDetail holds a position, as Get Position Details returns it. Fields documented as cross margin only
// or isolated margin only are absent for positions of the other margin mode
type FuturesPositionDetail struct {
	ID                           string       `json:"id"`
	Symbol                       string       `json:"symbol"`
	MaintenanceMarginRequirement types.Number `json:"maintMarginReq"`
	CrossMode                    bool         `json:"crossMode"`
	// DeleveragePercentage is the auto-deleveraging ranking percentile
	DeleveragePercentage types.Number `json:"delevPercentage"`
	OpeningTimestamp     types.Time   `json:"openingTimestamp"`
	CurrentTimestamp     types.Time   `json:"currentTimestamp"`
	// CurrentQuantity is in lots, negative for a short position
	CurrentQuantity   int64        `json:"currentQty"`
	CurrentCost       types.Number `json:"currentCost"`
	CurrentCommission types.Number `json:"currentComm"`
	UnrealisedCost    types.Number `json:"unrealisedCost"`
	RealisedGrossCost types.Number `json:"realisedGrossCost"`
	RealisedCost      types.Number `json:"realisedCost"`
	IsOpen            bool         `json:"isOpen"`
	MarkPrice         types.Number `json:"markPrice"`
	MarkValue         types.Number `json:"markValue"`
	PositionCost      types.Number `json:"posCost"`
	// PositionInitialMargin is the opening value over the cross leverage for cross margin, and the sum of each fill's
	// initial margin for isolated margin
	PositionInitialMargin types.Number `json:"posInit"`
	// PositionMargin is the mark value times the initial margin rate for cross margin, and the initial and added
	// margin plus funding fees for isolated margin
	PositionMargin            types.Number  `json:"posMargin"`
	PositionMaintenanceMargin types.Number  `json:"posMaint"`
	RealisedGrossPNL          types.Number  `json:"realisedGrossPnl"`
	RealisedPNL               types.Number  `json:"realisedPnl"`
	UnrealisedPNL             types.Number  `json:"unrealisedPnl"`
	UnrealisedPNLPercent      types.Number  `json:"unrealisedPnlPcnt"`
	UnrealisedROEPercent      types.Number  `json:"unrealisedRoePcnt"`
	AverageEntryPrice         types.Number  `json:"avgEntryPrice"`
	LiquidationPrice          types.Number  `json:"liquidationPrice"`
	BankruptPrice             types.Number  `json:"bankruptPrice"`
	SettleCurrency            currency.Code `json:"settleCurrency"`
	IsInverse                 bool          `json:"isInverse"`
	// MaintainMargin is the maintenance margin rate, for isolated margin
	MaintainMargin types.Number `json:"maintainMargin"`
	// MarginMode is CROSS or ISOLATED
	MarginMode string `json:"marginMode"`
	// PositionSide is BOTH in one-way mode, and LONG or SHORT in hedge mode
	PositionSide         string       `json:"positionSide"`
	Leverage             types.Number `json:"leverage"`
	CumulativeTradeFee   types.Number `json:"cumulativeTradeFee"`
	CumulativeFundingFee types.Number `json:"cumulativeFundingFee"`
	CumulativeTax        types.Number `json:"cumulativeTax"`
	// InitialMarginRate is for cross margin
	InitialMarginRate types.Number `json:"imr"`
	// AMR is for cross margin; KuCoin documents it only with a description copied from another field
	AMR types.Number `json:"amr"`
	// AutoDeposit and the fields that follow are for isolated margin
	AutoDeposit  bool         `json:"autoDeposit"`
	RiskLimit    uint64       `json:"riskLimit"`
	RealLeverage types.Number `json:"realLeverage"`
	// PositionCross is the margin added to the position
	PositionCross types.Number `json:"posCross"`
	// PositionCrossMargin is the margin added automatically, manually or by a risk limit change
	PositionCrossMargin types.Number `json:"posCrossMargin"`
	// PositionProfit is the funding fees the position received
	PositionProfit types.Number `json:"posProfit"`
	// PositionCommission is the bankruptcy cost
	PositionCommission types.Number `json:"posComm"`
	// PositionCommissionCommon is the part of the bankruptcy cost from opening and adding margin
	PositionCommissionCommon types.Number `json:"posCommCommon"`
	// PositionCommissionFunding is the part of received funding fees held as bankruptcy cost, for older positions
	PositionCommissionFunding types.Number `json:"posCommFunding"`
	// PositionLoss is the funding fees the position paid
	PositionLoss types.Number `json:"posLoss"`
	// PositionFunding is the funding fee not yet settled
	PositionFunding   types.Number `json:"posFunding"`
	MaintenanceMargin types.Number `json:"maintMargin"`
	RiskLimitLevel    uint64       `json:"riskLimitLevel"`
	// WithdrawPNL is the unrealised profit withdrawn from the position
	WithdrawPNL types.Number `json:"withdrawPnl"`
	// AggregateRate is the maintenance margin rate plus the taker fee rate
	AggregateRate types.Number `json:"aggRate"`
}

// FuturesPosition holds a position, as Get Position List returns it. Fields documented as isolated margin only are
// absent for cross margin positions
type FuturesPosition struct {
	ID        string `json:"id"`
	Symbol    string `json:"symbol"`
	CrossMode bool   `json:"crossMode"`
	// DeleveragePercentage is the auto-deleveraging ranking percentile
	DeleveragePercentage float64    `json:"delevPercentage"`
	OpeningTimestamp     types.Time `json:"openingTimestamp"`
	CurrentTimestamp     types.Time `json:"currentTimestamp"`
	// CurrentQuantity is in lots, negative for a short position
	CurrentQuantity   int64   `json:"currentQty"`
	CurrentCost       float64 `json:"currentCost"`
	CurrentCommission float64 `json:"currentComm"`
	UnrealisedCost    float64 `json:"unrealisedCost"`
	RealisedGrossCost float64 `json:"realisedGrossCost"`
	RealisedCost      float64 `json:"realisedCost"`
	IsOpen            bool    `json:"isOpen"`
	MarkPrice         float64 `json:"markPrice"`
	MarkValue         float64 `json:"markValue"`
	PositionCost      float64 `json:"posCost"`
	// PositionInitialMargin is the opening value over the cross leverage for cross margin, and the sum of each fill's
	// initial margin for isolated margin
	PositionInitialMargin float64 `json:"posInit"`
	// PositionMargin is the mark value times the initial margin rate for cross margin, and the initial and added
	// margin plus funding fees for isolated margin
	PositionMargin       float64       `json:"posMargin"`
	RealisedGrossPNL     float64       `json:"realisedGrossPnl"`
	RealisedPNL          float64       `json:"realisedPnl"`
	UnrealisedPNL        float64       `json:"unrealisedPnl"`
	UnrealisedPNLPercent float64       `json:"unrealisedPnlPcnt"`
	UnrealisedROEPercent float64       `json:"unrealisedRoePcnt"`
	AverageEntryPrice    float64       `json:"avgEntryPrice"`
	LiquidationPrice     float64       `json:"liquidationPrice"`
	BankruptPrice        float64       `json:"bankruptPrice"`
	SettleCurrency       currency.Code `json:"settleCurrency"`
	IsInverse            bool          `json:"isInverse"`
	// MarginMode is CROSS or ISOLATED
	MarginMode string `json:"marginMode"`
	// PositionSide is BOTH in one-way mode, and LONG or SHORT in hedge mode
	PositionSide string  `json:"positionSide"`
	Leverage     float64 `json:"leverage"`
	// AutoDeposit is for isolated margin
	AutoDeposit                  bool    `json:"autoDeposit"`
	MaintenanceMarginRequirement float64 `json:"maintMarginReq"`
	// RiskLimit, RealLeverage and the position fields that follow up to PositionFunding are for isolated margin
	RiskLimit    float64 `json:"riskLimit"`
	RealLeverage float64 `json:"realLeverage"`
	// PositionCross is the margin added to the position
	PositionCross float64 `json:"posCross"`
	// PositionCrossMargin is the margin added automatically, manually or by a risk limit change
	PositionCrossMargin float64 `json:"posCrossMargin"`
	// PositionCommission is the bankruptcy cost
	PositionCommission float64 `json:"posComm"`
	// PositionCommissionCommon is the part of the bankruptcy cost from opening and adding margin
	PositionCommissionCommon float64 `json:"posCommCommon"`
	// PositionLoss is the funding fees the position paid
	PositionLoss float64 `json:"posLoss"`
	// PositionFunding is the funding fee not yet settled
	PositionFunding           float64 `json:"posFunding"`
	PositionMaintenanceMargin float64 `json:"posMaint"`
	// MaintenanceMargin is the position margin, for isolated margin
	MaintenanceMargin float64 `json:"maintMargin"`
	// MaintainMargin is the maintenance margin rate
	MaintainMargin float64 `json:"maintainMargin"`
	// DealCommission is the trading fees charged opening and closing the position
	DealCommission float64 `json:"dealComm"`
	// FundingFee is the funding fees received, negative when paid
	FundingFee float64 `json:"fundingFee"`
	Tax        float64 `json:"tax"`
	// WithdrawPNL is the unrealised profit withdrawn from the position, for isolated margin
	WithdrawPNL float64 `json:"withdrawPnl"`
}

// GetPositionHistoryRequest holds the parameters of Get Positions History. KuCoin keeps three months of closed
// positions and lists a window of up to seven days per request
type GetPositionHistoryRequest struct {
	Symbol string
	// From and To bound the closing time
	From time.Time
	To   time.Time
	// Limit is at most 200; KuCoin defaults to 10
	Limit  uint64
	PageID uint64
}

// FuturesPositionHistoryResponse holds a page of closed positions
type FuturesPositionHistoryResponse struct {
	CurrentPage uint64                       `json:"currentPage"`
	PageSize    uint64                       `json:"pageSize"`
	TotalNumber uint64                       `json:"totalNum"`
	TotalPage   uint64                       `json:"totalPage"`
	Items       []FuturesPositionHistoryItem `json:"items"`
}

// FuturesPositionHistoryItem holds a closed position
type FuturesPositionHistoryItem struct {
	CloseID        string        `json:"closeId"`
	UserID         string        `json:"userId"`
	Symbol         string        `json:"symbol"`
	SettleCurrency currency.Code `json:"settleCurrency"`
	Leverage       types.Number  `json:"leverage"`
	// Type is how the position closed, such as CLOSE_LONG
	Type string `json:"type"`
	// PNL is net of trading and funding fees
	PNL               types.Number `json:"pnl"`
	RealisedGrossCost types.Number `json:"realisedGrossCost"`
	WithdrawPNL       types.Number `json:"withdrawPnl"`
	TradeFee          types.Number `json:"tradeFee"`
	FundingFee        types.Number `json:"fundingFee"`
	OpenTime          types.Time   `json:"openTime"`
	CloseTime         types.Time   `json:"closeTime"`
	OpenPrice         types.Number `json:"openPrice"`
	ClosePrice        types.Number `json:"closePrice"`
	// MarginMode is CROSS or ISOLATED
	MarginMode           string       `json:"marginMode"`
	RealisedGrossCostNew types.Number `json:"realisedGrossCostNew"`
	Tax                  types.Number `json:"tax"`
	// ROE, LiquidAmount and LiquidPrice are null when KuCoin does not set them
	ROE          types.Number `json:"roe"`
	LiquidAmount types.Number `json:"liquidAmount"`
	LiquidPrice  types.Number `json:"liquidPrice"`
	// Side is LONG or SHORT
	Side         string `json:"side"`
	PositionSide string `json:"positionSide"`
}

// FuturesCrossMarginLeverageResponse holds a contract's cross margin leverage
type FuturesCrossMarginLeverageResponse struct {
	Symbol   string       `json:"symbol"`
	Leverage types.Number `json:"leverage"`
}

// AddMarginRequest holds the parameters of Add Isolated Margin
type AddMarginRequest struct {
	Symbol string
	// Margin is at least 0.00001667 XBT or its equivalent
	Margin float64
	// BusinessNumber is a unique ID of at most 36 characters, so that KuCoin adds the margin only once
	BusinessNumber string
	// PositionSide is BOTH, the one-way mode default, LONG or SHORT; hedge mode requires LONG or SHORT
	PositionSide string
}

// AddMarginResponse holds the isolated margin position Add Isolated Margin changed
type AddMarginResponse struct {
	ID                           string  `json:"id"`
	Symbol                       string  `json:"symbol"`
	AutoDeposit                  bool    `json:"autoDeposit"`
	MaintenanceMarginRequirement float64 `json:"maintMarginReq"`
	RiskLimit                    uint64  `json:"riskLimit"`
	RealLeverage                 float64 `json:"realLeverage"`
	CrossMode                    bool    `json:"crossMode"`
	// DeleveragePercentage is the auto-deleveraging ranking percentile
	DeleveragePercentage float64    `json:"delevPercentage"`
	OpeningTimestamp     types.Time `json:"openingTimestamp"`
	CurrentTimestamp     types.Time `json:"currentTimestamp"`
	// CurrentQuantity is in lots, negative for a short position
	CurrentQuantity   int64   `json:"currentQty"`
	CurrentCost       float64 `json:"currentCost"`
	CurrentCommission float64 `json:"currentComm"`
	UnrealisedCost    float64 `json:"unrealisedCost"`
	RealisedGrossCost float64 `json:"realisedGrossCost"`
	RealisedCost      float64 `json:"realisedCost"`
	IsOpen            bool    `json:"isOpen"`
	MarkPrice         float64 `json:"markPrice"`
	MarkValue         float64 `json:"markValue"`
	PositionCost      float64 `json:"posCost"`
	// PositionCross is the margin added to the position
	PositionCross         float64 `json:"posCross"`
	PositionInitialMargin float64 `json:"posInit"`
	// PositionCommission is the bankruptcy cost
	PositionCommission float64 `json:"posComm"`
	// PositionLoss is the funding fees the position paid
	PositionLoss              float64 `json:"posLoss"`
	PositionMargin            float64 `json:"posMargin"`
	PositionMaintenanceMargin float64 `json:"posMaint"`
	// MaintenanceMargin is the position margin
	MaintenanceMargin    float64       `json:"maintMargin"`
	RealisedGrossPNL     float64       `json:"realisedGrossPnl"`
	RealisedPNL          float64       `json:"realisedPnl"`
	UnrealisedPNL        float64       `json:"unrealisedPnl"`
	UnrealisedPNLPercent float64       `json:"unrealisedPnlPcnt"`
	UnrealisedROEPercent float64       `json:"unrealisedRoePcnt"`
	AverageEntryPrice    float64       `json:"avgEntryPrice"`
	LiquidationPrice     float64       `json:"liquidationPrice"`
	BankruptPrice        float64       `json:"bankruptPrice"`
	UserID               uint64        `json:"userId"`
	SettleCurrency       currency.Code `json:"settleCurrency"`
}

// RemoveMarginManuallyRequest holds the parameters of Remove Isolated Margin
type RemoveMarginManuallyRequest struct {
	Symbol string
	// WithdrawAmount is in USDT for USDT-margined contracts, and in coins for coin-margined contracts
	WithdrawAmount float64
	// PositionSide is BOTH, the one-way mode default, LONG or SHORT; hedge mode requires LONG or SHORT
	PositionSide string
}

// GetFuturesCrossMarginRiskLimitRequest holds the parameters of Get Cross Margin Risk Limit
type GetFuturesCrossMarginRiskLimitRequest struct {
	// Symbols are up to 50 contracts
	Symbols []string
	// TotalMargin is the opening margin in the settlement currency; KuCoin defaults to the equivalent of 10000 USDT
	TotalMargin float64
	// Leverage defaults to the contract's largest cross leverage
	Leverage uint64
}

// FuturesCrossMarginRiskLimit holds the largest cross margin position a contract can open with a margin and leverage
type FuturesCrossMarginRiskLimit struct {
	Symbol string `json:"symbol"`
	// MaxOpenSize is in lots
	MaxOpenSize uint64 `json:"maxOpenSize"`
	// MaxOpenValue is in the quote currency
	MaxOpenValue types.Number `json:"maxOpenValue"`
	TotalMargin  types.Number `json:"totalMargin"`
	// Price defaults to the last traded price
	Price                 types.Number  `json:"price"`
	Leverage              types.Number  `json:"leverage"`
	MaintenanceMarginRate types.Number  `json:"mmr"`
	InitialMarginRate     types.Number  `json:"imr"`
	Currency              currency.Code `json:"currency"`
}

// GetFuturesCrossMarginRequirementRequest holds the parameters of Get Cross Margin Requirement
type GetFuturesCrossMarginRequirementRequest struct {
	// Symbols are up to 50 contracts
	Symbols []string
	// PositionValue is in USDT for USDT-margined contracts, USDC for USDC-margined contracts and USD for
	// coin-margined contracts
	PositionValue float64
	// Leverage defaults to the contract's largest leverage
	Leverage float64
}

// FuturesCrossMarginRequirement holds the cross margin rates of a contract's position of a value
type FuturesCrossMarginRequirement struct {
	Symbol string `json:"symbol"`
	// Size is in lots. It is documented as a number, and the copy trading twin of the endpoint sends it as a string
	Size          types.Number `json:"size"`
	PositionValue types.Number `json:"positionValue"`
	// Price is the last traded price
	Price                 types.Number `json:"price"`
	InitialMarginRate     types.Number `json:"imr"`
	MaintenanceMarginRate types.Number `json:"mmr"`
}

// FuturesRiskLimitLevel holds an isolated margin risk limit level of a contract
type FuturesRiskLimitLevel struct {
	Symbol string `json:"symbol"`
	Level  uint64 `json:"level"`
	// MaxRiskLimit and MinRiskLimit bound the level's position value in the contract's settlement currency, MaxRiskLimit
	// included
	MaxRiskLimit   uint64  `json:"maxRiskLimit"`
	MinRiskLimit   uint64  `json:"minRiskLimit"`
	MaxLeverage    uint64  `json:"maxLeverage"`
	InitialMargin  float64 `json:"initialMargin"`
	MaintainMargin float64 `json:"maintainMargin"`
}
