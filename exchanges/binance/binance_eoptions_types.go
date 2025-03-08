package binance

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// EOptionExchangeInfo represents an exchange information.
type EOptionExchangeInfo struct {
	Timezone        string     `json:"timezone"`
	ServerTime      types.Time `json:"serverTime"`
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
		ContractID int64      `json:"contractId"`
		ExpiryDate types.Time `json:"expiryDate"`
		Filters    []struct {
			FilterType string       `json:"filterType"` // possible values are: PRICE_FILTER, LOT_SIZE
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
		Unit                 int64        `json:"unit"`
		MakerFeeRate         types.Number `json:"makerFeeRate"`
		TakerFeeRate         types.Number `json:"takerFeeRate"`
		MaxQty               types.Number `json:"maxQty"`
		MinQty               types.Number `json:"minQty"`
		InitialMargin        string       `json:"initialMargin"`
		MaintenanceMargin    string       `json:"maintenanceMargin"`
		MinInitialMargin     string       `json:"minInitialMargin"`
		MinMaintenanceMargin string       `json:"minMaintenanceMargin"`
		PriceScale           float64      `json:"priceScale"`
		QuantityScale        float64      `json:"quantityScale"`
		QuoteAsset           string       `json:"quoteAsset"`
	} `json:"optionSymbols"`
	RateLimits []RateLimitInfo `json:"rateLimits"`
}

// EOptionsOrderbook represents an european orderbook option information.
type EOptionsOrderbook struct {
	TransactionTime types.Time        `json:"T"`
	UpdateID        int64             `json:"u"`
	Asks            OrderbookTranches `json:"asks"`
	Bids            OrderbookTranches `json:"bids"` // [][Price, Quantity]
}

// EOptionsTradeItem represents a recent trade information
type EOptionsTradeItem struct {
	ID       int64        `json:"id"`
	Symbol   string       `json:"symbol"`
	Price    types.Number `json:"price"`
	Quantity types.Number `json:"qty"`
	QuoteQty types.Number `json:"quoteQty"`
	Side     tradeSide    `json:"side"` // Completed trade direction（-1 Sell，1 Buy）
	Time     types.Time   `json:"time"`
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
	Open        types.Number `json:"open"`
	High        types.Number `json:"high"`
	Low         types.Number `json:"low"`
	Close       types.Number `json:"close"`
	Volume      types.Number `json:"volume"`
	Amount      types.Number `json:"amount"`
	Interval    string       `json:"interval"`
	TradeCount  int64        `json:"tradeCount"`
	TakerVolume types.Number `json:"takerVolume"`
	TakerAmount types.Number `json:"takerAmount"`
	OpenTime    types.Time   `json:"openTime"`
	CloseTime   types.Time   `json:"closeTime"`
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
	Symbol             string       `json:"symbol"`
	PriceChange        types.Number `json:"priceChange"`
	PriceChangePercent types.Number `json:"priceChangePercent"`
	LastPrice          types.Number `json:"lastPrice"`
	LastQty            types.Number `json:"lastQty"`
	Open               types.Number `json:"open"`
	High               types.Number `json:"high"`
	Low                types.Number `json:"low"`
	Volume             types.Number `json:"volume"`
	Amount             types.Number `json:"amount"`
	BidPrice           types.Number `json:"bidPrice"`
	AskPrice           types.Number `json:"askPrice"`
	OpenTime           types.Time   `json:"openTime"`
	CloseTime          types.Time   `json:"closeTime"`
	FirstTradeID       int64        `json:"firstTradeId"`
	TradeCount         int64        `json:"tradeCount"`
	StrikePrice        types.Number `json:"strikePrice"`
	ExercisePrice      types.Number `json:"exercisePrice"`
}

// EOptionIndexSymbolPriceTicker represents spot index price
type EOptionIndexSymbolPriceTicker struct {
	Time       types.Time `json:"time"`
	IndexPrice string     `json:"indexPrice"`
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
	Symbol             string       `json:"symbol"`
	SumOpenInterest    types.Number `json:"sumOpenInterest"`
	SumOpenInterestUSD types.Number `json:"sumOpenInterestUsd"`
	Timestamp          types.Time   `json:"timestamp"`
}

// EOptionsAccountInformation represents current account information.
type EOptionsAccountInformation struct {
	Asset []struct {
		AssetType      string       `json:"asset"`
		MarginBalance  types.Number `json:"marginBalance"`
		AccountEquity  types.Number `json:"equity"`
		AvailableFunds types.Number `json:"available"`
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
	RiskLevel string     `json:"riskLevel"`
	Time      types.Time `json:"time"`
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
	OrderID       int64             `json:"orderId"`
	ClientOrderID string            `json:"clientOrderId"`
	Symbol        string            `json:"symbol"`
	Price         types.Number      `json:"price"`
	Quantity      types.Number      `json:"quantity"`
	Side          string            `json:"side"`
	Type          string            `json:"type"`
	CreateDate    types.Time        `json:"createDate,omitempty"`
	UpdateTime    types.Time        `json:"updateTime"`
	ExecutedQty   types.Number      `json:"executedQty,omitempty"`
	Fee           types.Number      `json:"fee,omitempty"`
	TimeInForce   order.TimeInForce `json:"timeInForce,omitempty"`
	ReduceOnly    bool              `json:"reduceOnly,omitempty"`
	PostOnly      bool              `json:"postOnly,omitempty"`
	Source        string            `json:"source"`
	CreateTime    types.Time        `json:"createTime,omitempty"`
	Status        string            `json:"status,omitempty"`
	AvgPrice      types.Number      `json:"avgPrice,omitempty"`
	PriceScale    int64             `json:"priceScale,omitempty"`
	QuantityScale int64             `json:"quantityScale,omitempty"`
	OptionSide    string            `json:"optionSide,omitempty"`
	QuoteAsset    string            `json:"quoteAsset,omitempty"`
	Mmp           bool              `json:"mmp,omitempty"` // is market maker protection order, true/false
}

// OptionPosition represents current position information.
type OptionPosition struct {
	AverageEntryPrice string       `json:"entryPrice"`
	Symbol            string       `json:"symbol"`
	Side              string       `json:"side"`         // Position Direction
	Quantity          types.Number `json:"quantity"`     // Number of positions (positive numbers represent long positions, negative number represent short positions)
	ReducibleQty      string       `json:"reducibleQty"` //// Number of positions that can be reduced
	MarkValue         string       `json:"markValue"`
	Ror               string       `json:"ror"`
	UnrealizedPNL     string       `json:"unrealizedPNL"` // Unrealized profit/loss
	MarkPrice         string       `json:"markPrice"`
	StrikePrice       string       `json:"strikePrice"`
	PositionCost      string       `json:"positionCost"`
	ExpiryTime        int64        `json:"expiryDate"`
	PriceScale        int64        `json:"priceScale"`
	QuantityScale     int64        `json:"quantityScale"`
	OptionSide        string       `json:"optionSide"`
	QuoteAsset        string       `json:"quoteAsset"`
}

// OptionsAccountTradeItem represents an options account trade item
type OptionsAccountTradeItem struct {
	ID             int64        `json:"id"`
	TradeID        int64        `json:"tradeId"`
	OrderID        int64        `json:"orderId"`
	Symbol         string       `json:"symbol"`
	Price          types.Number `json:"price"`
	Quantity       types.Number `json:"quantity"`
	Fee            types.Number `json:"fee"`
	RealizedProfit types.Number `json:"realizedProfit"`
	Side           string       `json:"side"`
	Type           string       `json:"type"`
	Volatility     string       `json:"volatility"`
	Liquidity      string       `json:"liquidity"`
	QuoteAsset     string       `json:"quoteAsset"`
	Time           types.Time   `json:"time"`
	PriceScale     int64        `json:"priceScale"`
	QuantityScale  int64        `json:"quantityScale"`
	OptionSide     string       `json:"optionSide"`
}

// UserOptionsExerciseRecord represents options users exercise records
type UserOptionsExerciseRecord struct {
	ID            string       `json:"id"`
	Currency      string       `json:"currency"`
	Symbol        string       `json:"symbol"`
	ExercisePrice types.Number `json:"exercisePrice"`
	MarkPrice     types.Number `json:"markPrice"`
	Quantity      types.Number `json:"quantity"`
	Amount        types.Number `json:"amount"`
	Fee           types.Number `json:"fee"`
	CreateDate    types.Time   `json:"createDate"`
	PriceScale    int64        `json:"priceScale"`
	QuantityScale int64        `json:"quantityScale"`
	OptionSide    string       `json:"optionSide"`
	PositionSide  string       `json:"positionSide"`
	QuoteAsset    string       `json:"quoteAsset"`
}

// AccountFunding represents account funding flow
type AccountFunding struct {
	ID         int64        `json:"id"`
	Asset      string       `json:"asset"`
	Amount     types.Number `json:"amount"`
	Type       string       `json:"type"`
	CreateDate types.Time   `json:"createDate"`
}

// DownloadIDOfOptionsTransaction represents download id information for options transaction.
type DownloadIDOfOptionsTransaction struct {
	AvgCostTimestampOfLast30D int64  `json:"avgCostTimestampOfLast30d"`
	DownloadID                string `json:"downloadId"`
}

// DownloadIDTransactionHistory represents a transaction history download link information.
type DownloadIDTransactionHistory struct {
	DownloadID          string     `json:"downloadId"`
	Status              string     `json:"status"`
	URL                 string     `json:"url"`
	Notified            bool       `json:"notified"`
	ExpirationTimestamp types.Time `json:"expirationTimestamp"`
	IsExpired           any        `json:"isExpired"`
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
	RiskLevel string     `json:"riskLevel"`
	Time      types.Time `json:"time"`
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
	UnderlyingID             int64        `json:"underlyingId"`
	Underlying               string       `json:"underlying"`
	WindowTimeInMilliseconds types.Time   `json:"windowTimeInMilliseconds"`
	FrozenTimeInMilliseconds types.Time   `json:"frozenTimeInMilliseconds"`
	QtyLimit                 types.Number `json:"qtyLimit"`
	DeltaLimit               types.Number `json:"deltaLimit"`
	LastTriggerTime          types.Time   `json:"lastTriggerTime"`
}

// UnderlyingCountdown represents a response for cancelling open orders.
type UnderlyingCountdown struct {
	Underlying    string `json:"underlying"`
	CountdownTime int64  `json:"countdownTime"`
}

// EOptionsWsTrade represents an european options
type EOptionsWsTrade struct {
	EventType          string       `json:"e"`
	EventTime          types.Time   `json:"E"`
	Symbol             string       `json:"s"`
	TradeID            int64        `json:"t"`
	Price              types.Number `json:"p"`
	Quantity           types.Number `json:"q"`
	BuyOrderID         int64        `json:"b"`
	SellOrderID        int64        `json:"a"`
	TradeCompletedTime types.Time   `json:"T"`
	Direction          string       `json:"S"` // direction, -1 for taker sell, 1 for taker buy
}

// EOptionSubscriptionParam represents a subscription/unsubscription parameter used to
type EOptionSubscriptionParam struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     int64    `json:"id"`
}

// EOptionsOperationResponse represents response coming through the websocket stream
type EOptionsOperationResponse struct {
	Error struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Result interface{} `json:"result"`
	ID     int64       `json:"id"`
}

// OptionsTicker24Hr represents 24-hour ticker data
type OptionsTicker24Hr struct {
	EventType                  string       `json:"e"`
	EventTime                  types.Time   `json:"E"`
	TransactionTime            types.Time   `json:"T"`
	Symbol                     string       `json:"s"`
	OpeningPrice               types.Number `json:"o"`
	HightPrice                 types.Number `json:"h"`
	LowPrice                   types.Number `json:"l"`
	ClosingPrice               types.Number `json:"c"`
	TradingVolume              types.Number `json:"V"` // Trading volume in contract
	TradingAmount              types.Number `json:"A"` // Trading volume in quote asset
	PriceChangesPercent        types.Number `json:"P"`
	PriceChange                string       `json:"p"`
	VolumeOfLastCompletedTrade string       `json:"Q"` // In contract asset
	FirstTradeID               string       `json:"F"`
	LastTradeID                string       `json:"L"`
	NumberOfTrade              int64        `json:"n"`
	BestBuyPrice               types.Number `json:"bo"`
	BestSellPrice              types.Number `json:"ao"`
	BestBuyQuantity            types.Number `json:"bq"`
	BestSellQuantity           types.Number `json:"aq"`
	BuyImpliedVolatility       types.Number `json:"b"`
	SellImpliedVolatility      types.Number `json:"a"`
	Delta                      types.Number `json:"d"`
	Theta                      types.Number `json:"t"`
	Gamma                      types.Number `json:"g"`
	Vega                       types.Number `json:"v"`
	ImpliedVolatility          types.Number `json:"vo"`
	MarkPrice                  types.Number `json:"mp"`
	BuyMaximumPrice            types.Number `json:"hl"`
	SellMaximumPrice           types.Number `json:"ll"`
	EstimatedStrikePrice       types.Number `json:"eep"`
}

// OptionsIndexInfo represents options index price information.
type OptionsIndexInfo struct {
	EventType        string       `json:"e"`
	EventTime        types.Time   `json:"E"`
	UnderlyingSymbol string       `json:"s"`
	Price            types.Number `json:"p"`
}

// WsOptionsMarkPrice represents a push data from options mark price.
type WsOptionsMarkPrice struct {
	EventType string       `json:"e"`
	EventTime types.Time   `json:"E"`
	Symbol    string       `json:"s"`
	MarkPrice types.Number `json:"mp"`
}

// WsOptionsKlineData represents an options kline push data
type WsOptionsKlineData struct {
	EventType string     `json:"e"`
	EventTime types.Time `json:"E"`
	Symbol    string     `json:"s"`
	KlineData struct {
		StartTime                 types.Time   `json:"t"`
		EndTime                   types.Time   `json:"T"`
		Symbol                    string       `json:"s"`
		CandlePeriod              string       `json:"i"`
		FirstTradeID              int64        `json:"F"`
		LastID                    int64        `json:"L"`
		Open                      types.Number `json:"o"`
		Close                     types.Number `json:"c"`
		High                      types.Number `json:"h"`
		Low                       types.Number `json:"l"`
		ContractVolume            types.Number `json:"v"` // Contract or Base
		NumberOfTrades            int64        `json:"n"`
		ContractCompleted         bool         `json:"x"`
		CompletedTradeAmount      string       `json:"q"` // In quote asset
		TakerCompletedTradeVolume types.Number `json:"V"`
		TakerTradeAmount          types.Number `json:"Q"`
	} `json:"k"`
}

// WsOptionIncomingResp used by wsHandleEOptionsData
type WsOptionIncomingResp struct {
	ID        int64           `json:"id"`
	EventType string          `json:"e"`
	Result    json.RawMessage `json:"result"`
	Stream    string          `json:"stream"`
	Data      json.RawMessage `json:"data"`
}

// WsOptionIncomingResps list of WsOptionIncomingResp
type WsOptionIncomingResps struct {
	Instances []WsOptionIncomingResp

	// To record the information about whether the incoming data was a slice or sing object instance.
	// Reason: Some slices may have a single element, which creates uncertainty about whether the incoming data is slice or object instance.
	IsSlice bool
}

// WsOpenInterest represents a single open interest instance.
type WsOpenInterest struct {
	EventType              string     `json:"e"`
	EventTime              types.Time `json:"E"`
	Symbol                 string     `json:"s"`
	OpenInterestInContract string     `json:"o"` // Base
	OpenInterestInUSDT     string     `json:"h"`
}

// WsOptionsNewPair represents a new options pair update information
type WsOptionsNewPair struct {
	ID                        int64        `json:"id"`
	EventType                 string       `json:"e"`
	EventTime                 types.Time   `json:"E"`
	UnderlyingAssetID         int64        `json:"cid"`
	UnderlyingIndexOfContract string       `json:"u"`
	QuotationAsset            string       `json:"qa"`
	TradingPairName           string       `json:"s"`
	Unit                      int64        `json:"unit"` // Conversion ratio, the quantity of the underlying asset represented by a single contract
	MinimumTradeVolume        string       `json:"mq"`   // Minimum trade volume of the underlying asset
	OptionType                string       `json:"d"`
	StrikePrice               types.Number `json:"sp"`
	ExpirationTime            types.Time   `json:"ed"`
}

// WsOptionsOrderbook represents a partial orderbook websocket stream data
type WsOptionsOrderbook struct {
	EventType       string            `json:"e"`
	EventTime       types.Time        `json:"E"`
	TransactionTime types.Time        `json:"T"`
	OptionSymbol    string            `json:"symbol"`
	UpdateID        int64             `json:"u"`  // update id in event
	PUpdateID       int64             `json:"pu"` // same as update id in event
	Bids            OrderbookTranches `json:"b"`  // 0: Price 1: Quantity
	Asks            OrderbookTranches `json:"a"`
}
