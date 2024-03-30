package binance

import (
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// EOptionExchangeInfo represents an exchange information.
type EOptionExchangeInfo struct {
	Timezone        string               `json:"timezone"`
	ServerTime      convert.ExchangeTime `json:"serverTime"`
	OptionContracts []struct {
		ID          int64  `json:"id"`
		BaseAsset   string `json:"baseAsset"`
		QuoteAsset  string `json:"quoteAsset"`
		Underlying  string `json:"underlying"`
		SettleAsset string `json:"settleAsset"`
	} `json:"optionContracts"`
	OptionAssets []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"optionAssets"`
	OptionSymbols []struct {
		ContractID int64                `json:"contractId"`
		ExpiryDate convert.ExchangeTime `json:"expiryDate"`
		Filters    []struct {
			FilterType string       `json:"filterType"`
			MinPrice   types.Number `json:"minPrice,omitempty"`
			MaxPrice   types.Number `json:"maxPrice,omitempty"`
			TickSize   types.Number `json:"tickSize,omitempty"`
			MinQty     types.Number `json:"minQty,omitempty"`
			MaxQty     types.Number `json:"maxQty,omitempty"`
			StepSize   types.Number `json:"stepSize,omitempty"`
		} `json:"filters"`
		ID                   int64        `json:"id"`
		Symbol               string       `json:"symbol"`
		Side                 string       `json:"side"`
		StrikePrice          types.Number `json:"strikePrice"`
		Underlying           string       `json:"underlying"`
		Unit                 int          `json:"unit"`
		MakerFeeRate         string       `json:"makerFeeRate"`
		TakerFeeRate         string       `json:"takerFeeRate"`
		MinQty               string       `json:"minQty"`
		MaxQty               string       `json:"maxQty"`
		InitialMargin        string       `json:"initialMargin"`
		MaintenanceMargin    string       `json:"maintenanceMargin"`
		MinInitialMargin     string       `json:"minInitialMargin"`
		MinMaintenanceMargin string       `json:"minMaintenanceMargin"`
		PriceScale           float64      `json:"priceScale"`
		QuantityScale        float64      `json:"quantityScale"`
		QuoteAsset           string       `json:"quoteAsset"`
	} `json:"optionSymbols"`
	RateLimits []struct {
		RateLimitType string `json:"rateLimitType"`
		Interval      string `json:"interval"`
		IntervalNum   int64  `json:"intervalNum"`
		Limit         int64  `json:"limit"`
	} `json:"rateLimits"`
}

// EOptionsOrderbook represents an european orderbook option information.
type EOptionsOrderbook struct {
	TransactionTime convert.ExchangeTime `json:"T"`
	UpdateID        int64                `json:"u"`
	Asks            [][2]types.Number    `json:"asks"`
	Bids            [][2]types.Number    `json:"bids"` // [][Price, Quantity]
}

// EOptionsTradeItem represents a recent trade information
type EOptionsTradeItem struct {
	ID       string               `json:"id"`
	Symbol   string               `json:"symbol"`
	Price    types.Number         `json:"price"`
	Quantity types.Number         `json:"qty"`
	QuoteQty types.Number         `json:"quoteQty"`
	Side     tradeSide            `json:"side"` // Completed trade direction（-1 Sell，1 Buy）
	Time     convert.ExchangeTime `json:"time"`
}

type tradeSide int64

const (
	sellSide tradeSide = -1
	buySide  tradeSide = 1
)

// String returns a string representation of side value in eoptions recent trades.
func (s tradeSide) String() string {
	switch s {
	case sellSide:
		return "Sell"
	case buySide:
		return "Buy"
	default:
		return ""
	}
}

// EOptionsCandlestick represents candlestick bar for options symbols.
type EOptionsCandlestick struct {
	Open        types.Number         `json:"open"`
	High        types.Number         `json:"high"`
	Low         types.Number         `json:"low"`
	Close       types.Number         `json:"close"`
	Volume      types.Number         `json:"volume"`
	Amount      types.Number         `json:"amount"`
	Interval    string               `json:"interval"`
	TradeCount  int64                `json:"tradeCount"`
	TakerVolume types.Number         `json:"takerVolume"`
	TakerAmount types.Number         `json:"takerAmount"`
	OpenTime    convert.ExchangeTime `json:"openTime"`
	CloseTime   convert.ExchangeTime `json:"closeTime"`
}

// OptionMarkPrice represents an option mark price
type OptionMarkPrice struct {
	Symbol         string       `json:"symbol"`
	MarkPrice      types.Number `json:"markPrice"`
	BidIV          types.Number `json:"bidIV"`
	AskIV          types.Number `json:"askIV"`
	MarkIV         types.Number `json:"markIV"`
	Delta          types.Number `json:"delta"`
	Theta          types.Number `json:"theta"`
	Gamma          types.Number `json:"gamma"`
	Vega           types.Number `json:"vega"`
	HighPriceLimit types.Number `json:"highPriceLimit"`
	LowPriceLimit  types.Number `json:"lowPriceLimit"`
}

// EOptionTicker represents a ticker information for an european option instance.
type EOptionTicker struct {
	Symbol             string               `json:"symbol"`
	PriceChange        types.Number         `json:"priceChange"`
	PriceChangePercent types.Number         `json:"priceChangePercent"`
	LastPrice          types.Number         `json:"lastPrice"`
	LastQty            types.Number         `json:"lastQty"`
	Open               types.Number         `json:"open"`
	High               types.Number         `json:"high"`
	Low                types.Number         `json:"low"`
	Volume             types.Number         `json:"volume"`
	Amount             types.Number         `json:"amount"`
	BidPrice           types.Number         `json:"bidPrice"`
	AskPrice           types.Number         `json:"askPrice"`
	OpenTime           convert.ExchangeTime `json:"openTime"`
	CloseTime          convert.ExchangeTime `json:"closeTime"`
	FirstTradeID       int64                `json:"firstTradeId"`
	TradeCount         int64                `json:"tradeCount"`
	StrikePrice        types.Number         `json:"strikePrice"`
	ExercisePrice      types.Number         `json:"exercisePrice"`
}

// EOptionIndexSymbolPriceTicker represents spot index price
type EOptionIndexSymbolPriceTicker struct {
	Time       convert.ExchangeTime `json:"time"`
	IndexPrice string               `json:"indexPrice"`
}

// ExerciseHistoryItem represents an exercise history
// REALISTIC_VALUE_STRICKEN -> Exercised
// EXTRINSIC_VALUE_EXPIRED -> Expired OTM
type ExerciseHistoryItem struct {
	Symbol          string       `json:"symbol"`
	StrikePrice     types.Number `json:"strikePrice"`
	RealStrikePrice types.Number `json:"realStrikePrice"`
	ExpiryDate      int64        `json:"expiryDate"`
	StrikeResult    string       `json:"strikeResult"`
}

// OpenInterest represents an instance open interest
type OpenInterest struct {
	Symbol             string               `json:"symbol"`
	SumOpenInterest    types.Number         `json:"sumOpenInterest"`
	SumOpenInterestUSD types.Number         `json:"sumOpenInterestUsd"`
	Timestamp          convert.ExchangeTime `json:"timestamp"`
}

// EOptionsAccountInformation represents current account information.
type EOptionsAccountInformation struct {
	Asset []struct {
		AssetType      string       `json:"asset"`
		MarginBalance  types.Number `json:"marginBalance"`
		AccountEquity  types.Number `json:"equity"`
		AvailableFunds string       `json:"available"`
		Locked         types.Number `json:"locked"`        // locked balance for order and position
		UnrealizedPNL  types.Number `json:"unrealizedPNL"` // Unrealized profit/loss
	} `json:"asset"`
	Greek []struct {
		Underlying string       `json:"underlying"`
		Delta      types.Number `json:"delta"`
		Gamma      types.Number `json:"gamma"`
		Theta      types.Number `json:"theta"`
		Vega       types.Number `json:"vega"`
	} `json:"greek"`
	RiskLevel string               `json:"riskLevel"`
	Time      convert.ExchangeTime `json:"time"`
}

// OptionsOrderParams represents an options order instance.
type OptionsOrderParams struct {
	Symbol                  currency.Pair `json:"symbol"`
	Side                    string        `json:"side"`
	OrderType               string        `json:"type"`
	Amount                  float64       `json:"quantity"`
	Price                   float64       `json:"price,omitempty"`
	TimeInForce             string        `json:"timeInForce,omitempty"`
	ReduceOnly              bool          `json:"reduceOnly,omitempty"`
	PostOnly                bool          `json:"postOnly,omitempty"`
	NewOrderResponseType    string        `json:"newOrderRespType,omitempty"`
	ClientOrderID           string        `json:"clientOrderId,omitempty"`
	IsMarketMakerProtection bool          `json:"isMmp,omitempty"`
}

// OptionOrder represents an options order instance.
type OptionOrder struct {
	OrderID       int64                `json:"orderId"`
	ClientOrderID string               `json:"clientOrderId"`
	Symbol        string               `json:"symbol"`
	Price         types.Number         `json:"price"`
	Quantity      types.Number         `json:"quantity"`
	Side          string               `json:"side"`
	Type          string               `json:"type"`
	CreateDate    convert.ExchangeTime `json:"createDate,omitempty"`
	UpdateTime    convert.ExchangeTime `json:"updateTime"`
	ExecutedQty   types.Number         `json:"executedQty,omitempty"`
	Fee           types.Number         `json:"fee,omitempty"`
	TimeInForce   string               `json:"timeInForce,omitempty"`
	ReduceOnly    bool                 `json:"reduceOnly,omitempty"`
	PostOnly      bool                 `json:"postOnly,omitempty"`
	Source        string               `json:"source"`
	CreateTime    convert.ExchangeTime `json:"createTime,omitempty"`
	Status        string               `json:"status,omitempty"`
	AvgPrice      types.Number         `json:"avgPrice,omitempty"`
	PriceScale    int64                `json:"priceScale,omitempty"`
	QuantityScale int64                `json:"quantityScale,omitempty"`
	OptionSide    string               `json:"optionSide,omitempty"`
	QuoteAsset    string               `json:"quoteAsset,omitempty"`
	Mmp           bool                 `json:"mmp,omitempty"` // is market maker protection order, true/false
}

// OptionPosition represents current position position information.
type OptionPosition struct {
	AverageEntryPrice string `json:"entryPrice"`
	Symbol            string `json:"symbol"`
	Side              string `json:"side"`         // Position Direction
	Quantity          string `json:"quantity"`     // Number of positions (positive numbers represent long positions, negative number represent short positions)
	ReducibleQty      string `json:"reducibleQty"` //// Number of positions that can be reduced
	MarkValue         string `json:"markValue"`
	Ror               string `json:"ror"`
	UnrealizedPNL     string `json:"unrealizedPNL"` // Unrealized profit/loss
	MarkPrice         string `json:"markPrice"`
	StrikePrice       string `json:"strikePrice"`
	PositionCost      string `json:"positionCost"`
	ExpiryTime        int64  `json:"expiryDate"`
	PriceScale        int    `json:"priceScale"`
	QuantityScale     int    `json:"quantityScale"`
	OptionSide        string `json:"optionSide"`
	QuoteAsset        string `json:"quoteAsset"`
}

// OptionsAccountTradeItem represents an options account trade item
type OptionsAccountTradeItem struct {
	ID             int64                `json:"id"`
	TradeID        int64                `json:"tradeId"`
	OrderID        int64                `json:"orderId"`
	Symbol         string               `json:"symbol"`
	Price          types.Number         `json:"price"`
	Quantity       types.Number         `json:"quantity"`
	Fee            types.Number         `json:"fee"`
	RealizedProfit types.Number         `json:"realizedProfit"`
	Side           string               `json:"side"`
	Type           string               `json:"type"`
	Volatility     string               `json:"volatility"`
	Liquidity      string               `json:"liquidity"`
	QuoteAsset     string               `json:"quoteAsset"`
	Time           convert.ExchangeTime `json:"time"`
	PriceScale     int64                `json:"priceScale"`
	QuantityScale  int64                `json:"quantityScale"`
	OptionSide     string               `json:"optionSide"`
}

// UserOptionsExerciseRecord represents options users exercise records
type UserOptionsExerciseRecord struct {
	ID            string               `json:"id"`
	Currency      string               `json:"currency"`
	Symbol        string               `json:"symbol"`
	ExercisePrice types.Number         `json:"exercisePrice"`
	MarkPrice     types.Number         `json:"markPrice"`
	Quantity      types.Number         `json:"quantity"`
	Amount        types.Number         `json:"amount"`
	Fee           types.Number         `json:"fee"`
	CreateDate    convert.ExchangeTime `json:"createDate"`
	PriceScale    int64                `json:"priceScale"`
	QuantityScale int64                `json:"quantityScale"`
	OptionSide    string               `json:"optionSide"`
	PositionSide  string               `json:"positionSide"`
	QuoteAsset    string               `json:"quoteAsset"`
}

// AccountFunding represents account funding flow
type AccountFunding struct {
	ID         int64                `json:"id"`
	Asset      string               `json:"asset"`
	Amount     string               `json:"amount"`
	Type       string               `json:"type"`
	CreateDate convert.ExchangeTime `json:"createDate"`
}

// DownloadIDOfOptionsTransaction represents download id information for options transaction.
type DownloadIDOfOptionsTransaction struct {
	AvgCostTimestampOfLast30D int64  `json:"avgCostTimestampOfLast30d"`
	DownloadID                string `json:"downloadId"`
}

// DownloadIDTransactionHistory represents a transaction history download link information.
type DownloadIDTransactionHistory struct {
	DownloadID          string               `json:"downloadId"`
	Status              string               `json:"status"`
	URL                 string               `json:"url"`
	Notified            bool                 `json:"notified"`
	ExpirationTimestamp convert.ExchangeTime `json:"expirationTimestamp"`
	IsExpired           any                  `json:"isExpired"`
}

// OptionMarginAccountInfo represents an account information.
type OptionMarginAccountInfo struct {
	Asset []struct {
		AssetType     string       `json:"asset"`
		MarginBalance types.Number `json:"marginBalance"`
		Equity        types.Number `json:"equity"`
		Available     types.Number `json:"available"`
		InitialMargin types.Number `json:"initialMargin"`
		MaintMargin   types.Number `json:"maintMargin"`
		UnrealizedPNL types.Number `json:"unrealizedPNL"`
		LpProfit      types.Number `json:"lpProfit"` // Unrealized profit for long position
	} `json:"asset"`
	Greek []struct {
		Underlying string       `json:"underlying"`
		Delta      types.Number `json:"delta"`
		Gamma      types.Number `json:"gamma"`
		Theta      types.Number `json:"theta"`
		Vega       types.Number `json:"vega"`
	} `json:"greek"`
	RiskLevel string               `json:"riskLevel"`
	Time      convert.ExchangeTime `json:"time"`
}

// MarketMakerProtectionConfig represents a market maker protection for option market maker.
type MarketMakerProtectionConfig struct {
	Underlying               string  `json:"underlying"`
	WindowTimeInMilliseconds int64   `json:"windowTimeInMilliseconds"`
	FrozenTimeInMilliseconds int64   `json:"frozenTimeInMilliseconds"`
	QuantityLimit            float64 `json:"qtyLimit"`
	NetDeltaLimit            float64 `json:"deltaLimit"`
}

// MarketMakerProtection represents a market maker protection mechanism for option market maker.
type MarketMakerProtection struct {
	UnderlyingID             int64                `json:"underlyingId"`
	Underlying               string               `json:"underlying"`
	WindowTimeInMilliseconds convert.ExchangeTime `json:"windowTimeInMilliseconds"`
	FrozenTimeInMilliseconds convert.ExchangeTime `json:"frozenTimeInMilliseconds"`
	QtyLimit                 types.Number         `json:"qtyLimit"`
	DeltaLimit               types.Number         `json:"deltaLimit"`
	LastTriggerTime          convert.ExchangeTime `json:"lastTriggerTime"`
}

// UnderlyingCountdown represents a response for cancelling open orders.
type UnderlyingCountdown struct {
	Underlying    string `json:"underlying"`
	CountdownTime int64  `json:"countdownTime"`
}
