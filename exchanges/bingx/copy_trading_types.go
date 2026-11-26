package bingx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CopyTradingTakeProfitStopLossRequest holds the parameters of Traders set take profit and stop loss based on order
// numbers; at least one of the mark prices is required
type CopyTradingTakeProfitStopLossRequest struct {
	// PositionID is the number of the order the trader leads
	PositionID          uint64
	TakeProfitMarkPrice float64
	StopLossMarkPrice   float64
}

// CopyTradingTakeProfitStopLossResponse holds the take profit and stop loss mark prices set on a lead order
type CopyTradingTakeProfitStopLossResponse struct {
	PositionID          NumericID `json:"positionId"`
	TakeProfitMarkPrice float64   `json:"takeProfitMarkPrice"`
	StopLossMarkPrice   float64   `json:"stopLossMarkPrice"`
}

// CopyTradingPerpetualTraderDetailResponse holds a trader's perpetual futures copy trading overview
type CopyTradingPerpetualTraderDetailResponse struct {
	ContractType CopyTradingContractType `json:"contractType"`
	TraderID     string                  `json:"traderId"`
	// Status is whether the trader can be followed
	Status       types.Boolean           `json:"status"`
	AccountLevel CopyTradingAccountLevel `json:"accountLevel"`
	// Ratio is the profit percentage over the requested number of days
	Ratio types.Number `json:"ratio"`
	// AccountAssets is in USDT, up to 15 minutes old
	AccountAssets types.Number `json:"accountAssets"`
	CopierProfit  types.Number `json:"copierProfit"`
	CopiersNumber uint64       `json:"copiersNum"`
	// RiskLevel runs from 1 to 9, the highest risk
	RiskLevel     uint64       `json:"riskLevel"`
	WinRatio      types.Number `json:"winRatio"`
	TradeNumber   uint64       `json:"tradeNum"`
	WinNumber     uint64       `json:"winNum"`
	LoseNumber    uint64       `json:"loseNum"`
	AverageProfit types.Number `json:"averageProfit"`
	AverageLosses types.Number `json:"averageLosses"`
	PNLRatio      types.Number `json:"PnlRatio"`
	// AverageTime is the average holding time in hours
	AverageTime    float64 `json:"averageTime"`
	MarginMedian   float64 `json:"marginMedian"`
	LeverageMedian float64 `json:"leverageMedian"`
	// TradeFrequency is the number of trades a week
	TradeFrequency types.Number `json:"tradeFrequency"`
	TradeDays      uint64       `json:"tradeDays"`
	LastTrade      types.Time   `json:"lastTrade"`
}

// CopyTradingPerpetualProfitSummaryResponse holds a trader's perpetual futures profit sharing overview
type CopyTradingPerpetualProfitSummaryResponse struct {
	ContractType CopyTradingContractType `json:"contractType"`
	// Earn is the estimated profit share for this week
	Earn types.Number `json:"earn"`
	// Settled is the profit share settled to date
	Settled     types.Number `json:"settled"`
	ProfitRatio types.Number `json:"profitRatio"`
}

// CopyTradingProfitDetailRequest holds the parameters of the perpetual futures and spot Profit Details queries
type CopyTradingProfitDetailRequest struct {
	StartTime time.Time
	EndTime   time.Time
	PageIndex uint64
	// PageSize is at most 100
	PageSize uint64
}

// CopyTradingPerpetualProfitDetailResponse is a profit share a trader received from a perpetual futures copier
type CopyTradingPerpetualProfitDetailResponse struct {
	ContractType   CopyTradingContractType `json:"contractType"`
	CommissionTime types.Time              `json:"commissionTime"`
	CopiersUID     NumericID               `json:"CopiersUid"`
	// ProfitShare is in USDT
	ProfitShare types.Number `json:"profitShare"`
}

// CopyTradingPerpetualCommissionRateResponse holds a trader's new perpetual futures commission rate
type CopyTradingPerpetualCommissionRateResponse struct {
	// NewCommission is a percentage with its sign, such as 16%
	NewCommission string `json:"newCommission"`
}

// CopyTradingPerpetualTradingPairResponse is a perpetual futures pair a trader can lead copy trading in
type CopyTradingPerpetualTradingPairResponse struct {
	ContractType CopyTradingContractType `json:"contractType"`
	// Symbol is the pair without a delimiter, such as BTCUSDT
	Symbol     string                `json:"symbol"`
	OpenTrader CopyTradingOpenTrader `json:"openTrader"`
	// MinOpenCount is the smallest quantity a trader can open
	MinOpenCount types.Number `json:"minOpenCount"`
	MaxLeverage  types.Number `json:"maxLeverage"`
	// StopSurplusRatio is the take profit ratio
	StopSurplusRatio types.Number `json:"stopSurplusRatio"`
	StopLossRatio    types.Number `json:"stopLossRatio"`
}

// CopyTradingSpotSellOrderResponse holds the order that sells the spot assets a trader's buy order bought
type CopyTradingSpotSellOrderResponse struct {
	OrderNumber       NumericID     `json:"orderNo"`
	CoinName          currency.Code `json:"coinName"`
	ValuationCoinName currency.Code `json:"valuationCoinName"`
	// Side is undocumented; the documented sell order reports 2
	Side uint64 `json:"side"`
	// Price is the price the trader gave
	Price types.Number `json:"price"`
	// DealPrice is the average fill price
	DealPrice     types.Number               `json:"dealPrice"`
	DelegatePrice types.Number               `json:"delegatePrice"`
	Status        CopyTradingSpotOrderStatus `json:"status"`
}

// CopyTradingSpotTraderDetailResponse holds a trader's spot copy trading overview
type CopyTradingSpotTraderDetailResponse struct {
	TraderID string `json:"traderId"`
	// Status is whether the trader can be followed
	Status types.Boolean `json:"status"`
	// Ratio is the profit percentage
	Ratio types.Number `json:"ratio"`
	// AccountAssets is in USDT, up to 15 minutes old
	AccountAssets types.Number `json:"accountAssets"`
	// PNL is the trader's cumulative profit and loss in USDT
	PNL types.Number `json:"PnL"`
	// CopierProfit is the copiers' profit and loss in USDT
	CopierProfit  types.Number `json:"copierProfit"`
	CopiersNumber uint64       `json:"copiersNum"`
	WinRatio      types.Number `json:"winRatio"`
	TradeNumber   uint64       `json:"tradeNum"`
	// TradeFrequency is the number of trades a week
	TradeFrequency types.Number `json:"tradeFrequency"`
	TradeDays      uint64       `json:"tradeDays"`
	// TradingPairs are the trader's preferred pairs over the last 30 days, without a delimiter, such as BTCUSDT
	TradingPairs []string   `json:"tradingPairs"`
	LastTrade    types.Time `json:"lastTrade"`
}

// CopyTradingSpotProfitSummaryResponse holds a trader's spot profit sharing overview
type CopyTradingSpotProfitSummaryResponse struct {
	// Earn is the estimated profit share for this week
	Earn types.Number `json:"earn"`
	// Settled is the profit share settled to date
	Settled     types.Number `json:"settled"`
	ProfitRatio types.Number `json:"profitRatio"`
}

// CopyTradingSpotProfitDetailResponse is a profit share a trader received from a spot copier
type CopyTradingSpotProfitDetailResponse struct {
	CommissionTime types.Time `json:"commissionTime"`
	CopiersUID     NumericID  `json:"CopiersUid"`
	// ProfitShare is in USDT
	ProfitShare types.Number `json:"profitShare"`
}

// CopyTradingSpotOrderHistoryRequest holds the parameters of Query Historical Orders
type CopyTradingSpotOrderHistoryRequest struct {
	// Symbol is every pair when empty
	Symbol    currency.Pair
	StartTime time.Time
	EndTime   time.Time
	PageIndex uint64
	// PageSize is at most 100
	PageSize uint64
}

// CopyTradingSpotOrderHistoryResponse is a trader's completed spot lead trade, its buy order and the order that sold it
type CopyTradingSpotOrderHistoryResponse struct {
	OpenOrderID  NumericID `json:"openOrderId"`
	CloseOrderID NumericID `json:"closeOrderId"`
	// Symbol is the pair without a delimiter, such as BTCUSDT
	Symbol   string       `json:"symbol"`
	Quantity types.Number `json:"quantity"`
	Amount   types.Number `json:"Amount"`
	// Profit is the realised profit and loss
	Profit types.Number `json:"profit"`
	// OpenPrice is the average buy price
	OpenPrice types.Number `json:"openPrice"`
	CloseTime types.Time   `json:"closeTime"`
	// ClosePrice is the average sell price
	ClosePrice types.Number `json:"closePrice"`
}
