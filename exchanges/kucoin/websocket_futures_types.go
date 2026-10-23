package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WebsocketFuturesTickerV2 is a futures Ticker V2 push, which carries the best bid and ask
type WebsocketFuturesTickerV2 struct {
	Symbol       string       `json:"symbol"`
	Sequence     uint64       `json:"sequence"`
	BestBidSize  uint64       `json:"bestBidSize"`
	BestBidPrice types.Number `json:"bestBidPrice"`
	BestAskPrice types.Number `json:"bestAskPrice"`
	BestAskSize  uint64       `json:"bestAskSize"`
	Timestamp    types.Time   `json:"ts"`
}

// WebsocketFuturesTickerV1 is a futures Ticker V1 push, which carries the last fill
type WebsocketFuturesTickerV1 struct {
	Symbol       string       `json:"symbol"`
	Sequence     uint64       `json:"sequence"`
	Side         string       `json:"side"`
	Size         uint64       `json:"size"`
	Price        types.Number `json:"price"`
	BestBidSize  uint64       `json:"bestBidSize"`
	BestBidPrice types.Number `json:"bestBidPrice"`
	BestAskPrice types.Number `json:"bestAskPrice"`
	TradeID      string       `json:"tradeId"`
	BestAskSize  uint64       `json:"bestAskSize"`
	Timestamp    types.Time   `json:"ts"`
}

// WebsocketFuturesOrderbookDepth is a futures Orderbook - Level 5 or Orderbook - Level 50 push
type WebsocketFuturesOrderbookDepth struct {
	Sequence      uint64                           `json:"sequence"`
	Asks          orderbook.LevelsArrayPriceAmount `json:"asks"`
	Bids          orderbook.LevelsArrayPriceAmount `json:"bids"`
	Timestamp     types.Time                       `json:"timestamp"`
	PushTimestamp types.Time                       `json:"ts"`
}

// WebsocketFuturesOrderbookIncrement is a futures Orderbook - Increment push
type WebsocketFuturesOrderbookIncrement struct {
	Sequence  uint64     `json:"sequence"`
	Change    string     `json:"change"` // Price, side, size
	Timestamp types.Time `json:"timestamp"`
}

// WebsocketFuturesExecution is a futures Trade push
type WebsocketFuturesExecution struct {
	Symbol       string       `json:"symbol"`
	Sequence     uint64       `json:"sequence"`
	Side         string       `json:"side"`
	Size         uint64       `json:"size"`
	Price        types.Number `json:"price"`
	TakerOrderID string       `json:"takerOrderId"`
	MakerOrderID string       `json:"makerOrderId"`
	TradeID      string       `json:"tradeId"`
	Timestamp    types.Time   `json:"ts"`
}

// WebsocketFuturesMarkIndexPrice is an Instrument push of the mark and index price
type WebsocketFuturesMarkIndexPrice struct {
	Symbol      string     `json:"symbol"` // From the topic; the push does not carry it
	MarkPrice   float64    `json:"markPrice"`
	IndexPrice  float64    `json:"indexPrice"`
	Granularity uint64     `json:"granularity"` // Milliseconds
	Timestamp   types.Time `json:"timestamp"`
}

// WebsocketFuturesFundingRate is an Instrument push of the funding rate
type WebsocketFuturesFundingRate struct {
	Symbol      string     `json:"symbol"`      // From the topic; the push does not carry it
	Period      uint64     `json:"period"`      // Undocumented
	Granularity uint64     `json:"granularity"` // Milliseconds
	FundingRate float64    `json:"fundingRate"`
	Timestamp   types.Time `json:"timestamp"`
}

// WebsocketFuturesFundingBegin is a Funding Fee Settlement push for the start of a settlement
type WebsocketFuturesFundingBegin struct {
	Symbol      string     `json:"symbol"`
	FundingTime types.Time `json:"fundingTime"`
	FundingRate float64    `json:"fundingRate"`
	Timestamp   types.Time `json:"timestamp"`
}

// WebsocketFuturesFundingEnd is a Funding Fee Settlement push for the end of a settlement
type WebsocketFuturesFundingEnd WebsocketFuturesFundingBegin

// WebsocketFuturesSnapshot is a futures Symbol Snapshot push, which carries the 24 hour statistics
type WebsocketFuturesSnapshot struct {
	FundingRate           float64    `json:"fundingRate"`
	HighPrice             float64    `json:"highPrice"`
	LastPrice             float64    `json:"lastPrice"`
	LowPrice              float64    `json:"lowPrice"`
	Price24HoursBefore    float64    `json:"price24HoursBefore"`
	PriceChange           float64    `json:"priceChg"`
	PriceChangePercentage float64    `json:"priceChgPct"`
	Symbol                string     `json:"symbol"`
	Timestamp             types.Time `json:"ts"`
	Turnover              float64    `json:"turnover"`
	Volume                float64    `json:"volume"`
}

// WebsocketFuturesOrderChange is a futures Orders push
type WebsocketFuturesOrderChange struct {
	Symbol          string       `json:"symbol"`
	OrderType       string       `json:"orderType"`
	Side            string       `json:"side"`
	CanceledSize    types.Number `json:"canceledSize"`
	OrderID         string       `json:"orderId"`
	Liquidity       string       `json:"liquidity"`
	MarginMode      string       `json:"marginMode"`
	Type            string       `json:"type"`
	OrderTime       types.Time   `json:"orderTime"`
	Size            types.Number `json:"size"`
	FilledSize      types.Number `json:"filledSize"`
	Price           types.Number `json:"price"`
	MatchPrice      types.Number `json:"matchPrice"`
	MatchSize       types.Number `json:"matchSize"`
	RemainSize      types.Number `json:"remainSize"`
	TradeID         string       `json:"tradeId"`
	ClientOrderID   string       `json:"clientOid"`
	OldSize         types.Number `json:"oldSize"`
	AllCanceledSize types.Number `json:"allCanceledSize"`
	Status          string       `json:"status"`
	Timestamp       types.Time   `json:"ts"`
	FeeType         string       `json:"feeType"`
	TradeType       string       `json:"tradeType"`
	PositionSide    string       `json:"positionSide"`
	UserID          string       `json:"userId"`
}

// WebsocketFuturesWalletBalance is a futures Balance push of the walletBalance.change subject
type WebsocketFuturesWalletBalance struct {
	CrossPositionMargin      types.Number  `json:"crossPosMargin"`
	IsolatedOrderMargin      types.Number  `json:"isolatedOrderMargin"`
	HoldBalance              types.Number  `json:"holdBalance"`
	Equity                   types.Number  `json:"equity"`
	Version                  types.Number  `json:"version"`
	AvailableBalance         types.Number  `json:"availableBalance"`
	IsolatedPositionMargin   types.Number  `json:"isolatedPosMargin"`
	MaxWithdrawAmount        types.Number  `json:"maxWithdrawAmount"`
	WalletBalance            types.Number  `json:"walletBalance"`
	IsolatedFundingFeeMargin types.Number  `json:"isolatedFundingFeeMargin"`
	CrossUnrealisedPNL       types.Number  `json:"crossUnPnl"`
	TotalCrossMargin         types.Number  `json:"totalCrossMargin"`
	Currency                 currency.Code `json:"currency"`
	IsolatedUnrealisedPNL    types.Number  `json:"isolatedUnPnl"`
	CrossOrderMargin         types.Number  `json:"crossOrderMargin"`
	Timestamp                types.Time    `json:"timestamp"`
}

// WebsocketFuturesOrderMargin is a futures Balance push of the deprecated orderMargin.change subject
type WebsocketFuturesOrderMargin struct {
	OrderMargin float64       `json:"orderMargin"`
	Currency    currency.Code `json:"currency"`
	Timestamp   types.Time    `json:"timestamp"`
}

// WebsocketFuturesAvailableBalance is a futures Balance push of the deprecated availableBalance.change subject
type WebsocketFuturesAvailableBalance struct {
	AvailableBalance float64       `json:"availableBalance"`
	HoldBalance      float64       `json:"holdBalance"`
	Currency         currency.Code `json:"currency"`
	Timestamp        types.Time    `json:"timestamp"`
}

// WebsocketFuturesWithdrawHold is a futures Balance push of the deprecated withdrawHold.change subject
type WebsocketFuturesWithdrawHold struct {
	WithdrawHold float64       `json:"withdrawHold"`
	Currency     currency.Code `json:"currency"`
	Timestamp    types.Time    `json:"timestamp"`
}

// WebsocketFuturesPosition is a Positions push of the position.change subject
type WebsocketFuturesPosition struct {
	Symbol                       string        `json:"symbol"`
	MaintenanceMarginRequirement float64       `json:"maintMarginReq"`
	RiskLimit                    uint64        `json:"riskLimit"`
	RealLeverage                 float64       `json:"realLeverage"`
	CrossMode                    bool          `json:"crossMode"`
	DeleveragePercentage         float64       `json:"delevPercentage"`
	OpeningTimestamp             types.Time    `json:"openingTimestamp"`
	AutoDeposit                  bool          `json:"autoDeposit"`
	CurrentTimestamp             types.Time    `json:"currentTimestamp"`
	CurrentQuantity              int64         `json:"currentQty"` // Negative for a short position
	CurrentCost                  float64       `json:"currentCost"`
	CurrentCommission            float64       `json:"currentComm"`
	UnrealisedCost               float64       `json:"unrealisedCost"`
	RealisedCost                 float64       `json:"realisedCost"`
	IsOpen                       bool          `json:"isOpen"`
	MarkPrice                    float64       `json:"markPrice"`
	MarkValue                    float64       `json:"markValue"`
	PositionCost                 float64       `json:"posCost"`
	PositionCross                float64       `json:"posCross"`
	PositionInitial              float64       `json:"posInit"`
	PositionCommission           float64       `json:"posComm"`
	PositionLoss                 float64       `json:"posLoss"`
	PositionMargin               float64       `json:"posMargin"`
	PositionFunding              float64       `json:"posFunding"`
	PositionMaintenance          float64       `json:"posMaint"`
	MaintenanceMargin            float64       `json:"maintMargin"`
	AverageEntryPrice            float64       `json:"avgEntryPrice"`
	LiquidationPrice             float64       `json:"liquidationPrice"`
	BankruptPrice                float64       `json:"bankruptPrice"`
	SettleCurrency               currency.Code `json:"settleCurrency"`
	ChangeReason                 string        `json:"changeReason"`
	RiskLimitLevel               uint64        `json:"riskLimitLevel"`
	RealisedGrossCost            float64       `json:"realisedGrossCost"`
	RealisedGrossPNL             float64       `json:"realisedGrossPnl"`
	RealisedPNL                  float64       `json:"realisedPnl"`
	UnrealisedPNL                float64       `json:"unrealisedPnl"`
	UnrealisedPNLPercentage      float64       `json:"unrealisedPnlPcnt"`
	UnrealisedROEPercentage      float64       `json:"unrealisedRoePcnt"`
	Leverage                     float64       `json:"leverage"`
	MarginMode                   string        `json:"marginMode"`
	PositionSide                 string        `json:"positionSide"`
	Tax                          float64       `json:"tax"`
	DealCommission               float64       `json:"dealComm"`
	FundingFee                   float64       `json:"fundingFee"`
	AggregateRate                float64       `json:"aggRate"` // Undocumented
}

// WebsocketFuturesPositionSettlement is a Positions push of the position.settlement subject
type WebsocketFuturesPositionSettlement struct {
	Symbol         string        `json:"symbol"`
	MarkPrice      float64       `json:"markPrice"`
	Quantity       int64         `json:"qty"` // Negative for a short position
	PositionSide   string        `json:"positionSide"`
	SettleCurrency currency.Code `json:"settleCurrency"`
	FundingTime    types.Time    `json:"fundingTime"`
	MarginMode     string        `json:"marginMode"`
	FundingFee     float64       `json:"fundingFee"`
	FundingRate    float64       `json:"fundingRate"`
	Timestamp      types.Time    `json:"ts"`
}

// WebsocketFuturesRiskLimitAdjustment is a Positions push of the position.adjustRiskLimit subject
type WebsocketFuturesRiskLimitAdjustment struct {
	Message        string `json:"msg"`
	Success        bool   `json:"success"`
	RiskLimitLevel uint64 `json:"riskLimitLevel"`
}

// WebsocketFuturesMarginMode is a Margin Mode push: the margin mode, CROSS or ISOLATED, by symbol
type WebsocketFuturesMarginMode map[string]string

// WebsocketFuturesCrossLeverage is a Cross Margin Leverage push: the leverage by symbol
type WebsocketFuturesCrossLeverage map[string]WebsocketFuturesLeverage

// WebsocketFuturesLeverage is a symbol's leverage in a Cross Margin Leverage push
type WebsocketFuturesLeverage struct {
	Leverage types.Number `json:"leverage"`
}

// WebsocketFuturesStopOrder is a futures Stop Orders push
type WebsocketFuturesStopOrder struct {
	MarginMode    string       `json:"marginMode"`
	OrderID       string       `json:"orderId"`
	OrderPrice    types.Number `json:"orderPrice"`
	OrderType     string       `json:"orderType"`
	Side          string       `json:"side"`
	Size          uint64       `json:"size"`
	Stop          string       `json:"stop"`
	StopPrice     types.Number `json:"stopPrice"`
	StopPriceType string       `json:"stopPriceType"`
	Symbol        string       `json:"symbol"`
	Type          string       `json:"type"`
	CreatedAt     types.Time   `json:"createdAt"`
	Timestamp     types.Time   `json:"ts"`
}
