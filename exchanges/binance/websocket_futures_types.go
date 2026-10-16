package binance

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// DerivativesStreamRequest is a SUBSCRIBE, UNSUBSCRIBE, LIST_SUBSCRIPTIONS, SET_PROPERTY or GET_PROPERTY request on a
// USDⓈ-M, COIN-M or options stream connection
type DerivativesStreamRequest struct {
	Method string `json:"method"`
	Params []any  `json:"params,omitempty"`
	ID     int64  `json:"id"`
}

// DerivativesStreamResponse is the reply to a DerivativesStreamRequest
type DerivativesStreamResponse struct {
	Result json.RawMessage         `json:"result"`
	Error  *DerivativesStreamError `json:"error"`
	ID     int64                   `json:"id"`
}

// DerivativesStreamError is the error a stream connection returns for a rejected request, before closing the
// connection
type DerivativesStreamError struct {
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// FuturesBookTicker is a best bid or ask update from the USDⓈ-M and COIN-M <symbol>@bookTicker and !bookTicker
// streams
type FuturesBookTicker struct {
	EventType       string       `json:"e"`
	UpdateID        uint64       `json:"u"`
	EventTime       types.Time   `json:"E"`
	TransactionTime types.Time   `json:"T"`
	Symbol          string       `json:"s"`
	Pair            string       `json:"ps"`
	BestBidPrice    types.Number `json:"b"`
	BestBidQuantity types.Number `json:"B"`
	BestAskPrice    types.Number `json:"a"`
	BestAskQuantity types.Number `json:"A"`
	SymbolType      uint64       `json:"st"` // 1 USDⓈ-M, 2 COIN-M
}

// FuturesDepthUpdate is an order book update from the USDⓈ-M and COIN-M diff, partial and RPI book depth streams.
// Update IDs are int64 to match the order book API they feed
type FuturesDepthUpdate struct {
	EventType             string                           `json:"e"`
	EventTime             types.Time                       `json:"E"`
	TransactionTime       types.Time                       `json:"T"`
	Symbol                string                           `json:"s"`
	Pair                  string                           `json:"ps"`
	FirstUpdateID         int64                            `json:"U"`
	FinalUpdateID         int64                            `json:"u"`
	PreviousFinalUpdateID int64                            `json:"pu"`
	Bids                  orderbook.LevelsArrayPriceAmount `json:"b"`
	Asks                  orderbook.LevelsArrayPriceAmount `json:"a"`
	SymbolType            uint64                           `json:"st"`
}

// FuturesAggregateTrade is a trade from the USDⓈ-M and COIN-M <symbol>@aggTrade streams. COIN-M counts Quantity in
// contracts
type FuturesAggregateTrade struct {
	EventType        string       `json:"e"`
	EventTime        types.Time   `json:"E"`
	Symbol           string       `json:"s"`
	AggregateTradeID uint64       `json:"a"`
	Price            types.Number `json:"p"`
	Quantity         types.Number `json:"q"`
	NormalQuantity   types.Number `json:"nq"` // USDⓈ-M only: the quantity without trades involving RPI orders
	FirstTradeID     uint64       `json:"f"`
	LastTradeID      uint64       `json:"l"`
	TradeTime        types.Time   `json:"T"`
	IsBuyerMaker     bool         `json:"m"`
	SymbolType       uint64       `json:"st"`
}

// FuturesLiquidationOrder is a market liquidation from the USDⓈ-M and COIN-M <symbol>@forceOrder and
// !forceOrder@arr streams
type FuturesLiquidationOrder struct {
	EventType string                        `json:"e"`
	EventTime types.Time                    `json:"E"`
	Order     FuturesLiquidationOrderDetail `json:"o"`
}

// FuturesLiquidationOrderDetail is the liquidated order of a FuturesLiquidationOrder. The docs place pair and symbol
// type beside the order for USDⓈ-M, but both hosts send them inside it
type FuturesLiquidationOrderDetail struct {
	Symbol                         string       `json:"s"`
	Pair                           string       `json:"ps"`
	Side                           string       `json:"S"`
	OrderType                      string       `json:"o"`
	TimeInForce                    string       `json:"f"`
	OriginalQuantity               types.Number `json:"q"`
	Price                          types.Number `json:"p"`
	AveragePrice                   types.Number `json:"ap"`
	OrderStatus                    string       `json:"X"`
	OrderLastFilledQuantity        types.Number `json:"l"`
	OrderFilledAccumulatedQuantity types.Number `json:"z"`
	OrderTradeTime                 types.Time   `json:"T"`
	SymbolType                     uint64       `json:"st"`
}

// FuturesMiniTicker is a 24 hour rolling window mini ticker from the USDⓈ-M and COIN-M <symbol>@miniTicker and
// !miniTicker@arr streams. COIN-M counts TotalTradedBaseAssetVolume in contracts and sends its base asset volume as
// TotalTradedQuoteAssetVolume
type FuturesMiniTicker struct {
	EventType                   string       `json:"e"`
	EventTime                   types.Time   `json:"E"`
	Symbol                      string       `json:"s"`
	Pair                        string       `json:"ps"`
	ClosePrice                  types.Number `json:"c"`
	OpenPrice                   types.Number `json:"o"`
	HighPrice                   types.Number `json:"h"`
	LowPrice                    types.Number `json:"l"`
	TotalTradedBaseAssetVolume  types.Number `json:"v"`
	TotalTradedQuoteAssetVolume types.Number `json:"q"`
	SymbolType                  uint64       `json:"st"`
}

// FuturesTicker is a 24 hour rolling window ticker from the USDⓈ-M and COIN-M <symbol>@ticker and !ticker@arr
// streams. COIN-M counts TotalTradedBaseAssetVolume in contracts and sends its base asset volume as
// TotalTradedQuoteAssetVolume. PriceChangePercent is a percentage
type FuturesTicker struct {
	EventType                   string       `json:"e"`
	EventTime                   types.Time   `json:"E"`
	Symbol                      string       `json:"s"`
	Pair                        string       `json:"ps"`
	PriceChange                 types.Number `json:"p"`
	PriceChangePercent          types.Number `json:"P"`
	WeightedAveragePrice        types.Number `json:"w"`
	LastPrice                   types.Number `json:"c"`
	LastQuantity                types.Number `json:"Q"`
	OpenPrice                   types.Number `json:"o"`
	HighPrice                   types.Number `json:"h"`
	LowPrice                    types.Number `json:"l"`
	TotalTradedBaseAssetVolume  types.Number `json:"v"`
	TotalTradedQuoteAssetVolume types.Number `json:"q"`
	StatisticsOpenTime          types.Time   `json:"O"`
	StatisticsCloseTime         types.Time   `json:"C"`
	FirstTradeID                int64        `json:"F"` // -1 when no trade happened in the window
	LastTradeID                 int64        `json:"L"`
	TotalNumberOfTrades         uint64       `json:"n"`
	SymbolType                  uint64       `json:"st"`
}

// FuturesCompositeIndex is the composite index information of a USDⓈ-M index symbol from the
// <symbol>@compositeIndex stream
type FuturesCompositeIndex struct {
	EventType         string                             `json:"e"`
	EventTime         types.Time                         `json:"E"`
	Symbol            string                             `json:"s"`
	Price             types.Number                       `json:"p"`
	BaseAssetCategory string                             `json:"C"`
	Composition       []FuturesCompositeIndexComposition `json:"c"`
}

// FuturesCompositeIndexComposition is one constituent of a FuturesCompositeIndex
type FuturesCompositeIndexComposition struct {
	BaseAsset          currency.Code `json:"b"`
	QuoteAsset         currency.Code `json:"q"`
	WeightInQuantity   types.Number  `json:"w"`
	WeightInPercentage types.Number  `json:"W"`
	IndexPrice         types.Number  `json:"i"`
}

// FuturesContinuousKline is a candle of a USDⓈ-M or COIN-M pair and contract type from the
// <pair>_<contractType>@continuousKline_<interval> stream
type FuturesContinuousKline struct {
	EventType    string                     `json:"e"`
	EventTime    types.Time                 `json:"E"`
	Pair         string                     `json:"ps"`
	ContractType string                     `json:"ct"`
	Kline        FuturesContinuousKlineData `json:"k"`
}

// FuturesContinuousKlineData is the candle of a FuturesContinuousKline
type FuturesContinuousKlineData struct {
	StartTime                types.Time   `json:"t"`
	CloseTime                types.Time   `json:"T"`
	Interval                 string       `json:"i"`
	FirstUpdateID            int64        `json:"f"`
	LastUpdateID             int64        `json:"L"`
	OpenPrice                types.Number `json:"o"`
	ClosePrice               types.Number `json:"c"`
	HighPrice                types.Number `json:"h"`
	LowPrice                 types.Number `json:"l"`
	Volume                   types.Number `json:"v"`
	NumberOfTrades           uint64       `json:"n"`
	IsClosed                 bool         `json:"x"`
	QuoteAssetVolume         types.Number `json:"q"`
	TakerBuyVolume           types.Number `json:"V"`
	TakerBuyQuoteAssetVolume types.Number `json:"Q"`
	Ignore                   types.Number `json:"B"`
}

// FuturesContractInfo is a contract listing, settlement or bracket update from the USDⓈ-M and COIN-M !contractInfo
// stream. NotionalBrackets is only sent when the brackets change
type FuturesContractInfo struct {
	EventType        string                       `json:"e"`
	EventTime        types.Time                   `json:"E"`
	Symbol           string                       `json:"s"`
	Pair             string                       `json:"ps"` // COIN-M only
	ContractType     string                       `json:"ct"`
	DeliveryDateTime types.Time                   `json:"dt"`
	OnboardDateTime  types.Time                   `json:"ot"`
	ContractStatus   string                       `json:"cs"`
	NotionalBrackets []FuturesContractInfoBracket `json:"bks"`
	SymbolType       uint64                       `json:"st"`
}

// FuturesContractInfoBracket is a notional bracket of a FuturesContractInfo
type FuturesContractInfoBracket struct {
	NotionalBracket      uint64  `json:"bs"`
	BracketFloorNotional float64 `json:"bnf"`
	BracketNotionalCap   float64 `json:"bnc"`
	MaintenanceRatio     float64 `json:"mmr"`
	AuxiliaryNumber      float64 `json:"cf"`
	MinLeverage          uint64  `json:"mi"`
	MaxLeverage          uint64  `json:"ma"`
}

// FuturesKline is a candle from the USDⓈ-M and COIN-M <symbol>@kline_<interval> streams. COIN-M counts Volume and
// TakerBuyBaseAssetVolume in contracts and sends its base asset volumes in the quote volume fields
type FuturesKline struct {
	EventType string           `json:"e"`
	EventTime types.Time       `json:"E"`
	Symbol    string           `json:"s"`
	Kline     FuturesKlineData `json:"k"`
}

// FuturesKlineData is the candle of a FuturesKline
type FuturesKlineData struct {
	StartTime                types.Time   `json:"t"`
	CloseTime                types.Time   `json:"T"`
	Symbol                   string       `json:"s"`
	Interval                 string       `json:"i"`
	FirstTradeID             int64        `json:"f"` // -1 when the candle has no trades
	LastTradeID              int64        `json:"L"`
	OpenPrice                types.Number `json:"o"`
	ClosePrice               types.Number `json:"c"`
	HighPrice                types.Number `json:"h"`
	LowPrice                 types.Number `json:"l"`
	Volume                   types.Number `json:"v"`
	NumberOfTrades           uint64       `json:"n"`
	IsClosed                 bool         `json:"x"`
	QuoteAssetVolume         types.Number `json:"q"`
	TakerBuyBaseAssetVolume  types.Number `json:"V"`
	TakerBuyQuoteAssetVolume types.Number `json:"Q"`
	Ignore                   types.Number `json:"B"`
}

// FuturesMarkPrice is a mark price and funding rate update from the USDⓈ-M and COIN-M <symbol>@markPrice,
// <pair>@markPrice and !markPrice@arr streams. COIN-M delivery contracts send no funding rate or next funding time
type FuturesMarkPrice struct {
	EventType              string       `json:"e"`
	EventTime              types.Time   `json:"E"`
	Symbol                 string       `json:"s"`
	MarkPrice              types.Number `json:"p"`
	IndexPrice             types.Number `json:"i"`
	EstimatedSettlePrice   types.Number `json:"P"` // only useful in the last hour before the settlement starts
	FundingRate            types.Number `json:"r"`
	MarkPriceMovingAverage types.Number `json:"ap"`
	NextFundingTime        types.Time   `json:"T"`
	SymbolType             uint64       `json:"st"`
}

// FuturesAssetIndex is an asset index update from the USDⓈ-M !assetIndex@arr and <assetSymbol>@assetIndex streams.
// Since the COIN-M integration it also carries COIN-M settlement asset indices such as BTCUSD
type FuturesAssetIndex struct {
	EventType             string       `json:"e"`
	EventTime             types.Time   `json:"E"`
	Symbol                string       `json:"s"`
	IndexPrice            types.Number `json:"i"`
	BidBuffer             types.Number `json:"b"`
	AskBuffer             types.Number `json:"a"`
	BidRate               types.Number `json:"B"`
	AskRate               types.Number `json:"A"`
	AutoExchangeBidBuffer types.Number `json:"q"`
	AutoExchangeAskBuffer types.Number `json:"g"`
	AutoExchangeBidRate   types.Number `json:"Q"`
	AutoExchangeAskRate   types.Number `json:"G"`
}

// FuturesTradingSession is a trading session update for the underlying market of USDⓈ-M TradFi perpetual contracts
// from the tradingSession stream
type FuturesTradingSession struct {
	EventType        string     `json:"e"`
	EventTime        types.Time `json:"E"`
	SessionStartTime types.Time `json:"t"`
	SessionEndTime   types.Time `json:"T"`
	SessionType      string     `json:"S"`
}

// CFuturesIndexPrice is an index price update from the COIN-M <pair>@indexPrice stream
type CFuturesIndexPrice struct {
	EventType  string       `json:"e"`
	EventTime  types.Time   `json:"E"`
	Pair       string       `json:"s"`
	IndexPrice types.Number `json:"p"`
}

// CFuturesPriceKline is an index or mark price candle from the COIN-M <pair>@indexPriceKline_<interval> and
// <symbol>@markPriceKline_<interval> streams
type CFuturesPriceKline struct {
	EventType string                 `json:"e"`
	EventTime types.Time             `json:"E"`
	Pair      string                 `json:"ps"`
	Kline     CFuturesPriceKlineData `json:"k"`
}

// CFuturesPriceKlineData is the candle of a CFuturesPriceKline. Binance documents f, L, v, q, V, Q and B as ignored
// for price candles; they are decoded under their regular candle names so that, for example, L cannot be decoded
// into l by case-insensitive matching
type CFuturesPriceKlineData struct {
	StartTime                types.Time   `json:"t"`
	CloseTime                types.Time   `json:"T"`
	Symbol                   string       `json:"s"` // "0" for index price candles
	Interval                 string       `json:"i"`
	FirstTradeID             int64        `json:"f"`
	LastTradeID              int64        `json:"L"`
	OpenPrice                types.Number `json:"o"`
	ClosePrice               types.Number `json:"c"`
	HighPrice                types.Number `json:"h"`
	LowPrice                 types.Number `json:"l"`
	Volume                   types.Number `json:"v"`
	NumberOfBasicData        uint64       `json:"n"`
	IsClosed                 bool         `json:"x"`
	QuoteAssetVolume         types.Number `json:"q"`
	TakerBuyBaseAssetVolume  types.Number `json:"V"`
	TakerBuyQuoteAssetVolume types.Number `json:"Q"`
	Ignore                   types.Number `json:"B"`
}

// FuturesMarginCall is a USDⓈ-M or COIN-M MARGIN_CALL user data event
type FuturesMarginCall struct {
	EventType          string                      `json:"e"`
	EventTime          types.Time                  `json:"E"`
	AccountAlias       string                      `json:"i"` // COIN-M only
	CrossWalletBalance types.Number                `json:"cw"`
	Positions          []FuturesMarginCallPosition `json:"p"`
}

// FuturesMarginCallPosition is a position of a FuturesMarginCall
type FuturesMarginCallPosition struct {
	Symbol                    string       `json:"s"`
	PositionSide              string       `json:"ps"`
	PositionAmount            types.Number `json:"pa"`
	MarginType                string       `json:"mt"`
	IsolatedWallet            types.Number `json:"iw"`
	MarkPrice                 types.Number `json:"mp"`
	UnrealizedPNL             types.Number `json:"up"`
	MaintenanceMarginRequired types.Number `json:"mm"`
}

// FuturesAccountUpdate is a USDⓈ-M or COIN-M ACCOUNT_UPDATE user data event
type FuturesAccountUpdate struct {
	EventType       string                   `json:"e"`
	EventTime       types.Time               `json:"E"`
	TransactionTime types.Time               `json:"T"`
	AccountAlias    string                   `json:"i"` // COIN-M only
	UpdateData      FuturesAccountUpdateData `json:"a"`
}

// FuturesAccountUpdateData holds the changed balances and positions of a FuturesAccountUpdate
type FuturesAccountUpdateData struct {
	EventReasonType string                         `json:"m"`
	Balances        []FuturesAccountUpdateBalance  `json:"B"`
	Positions       []FuturesAccountUpdatePosition `json:"P"`
	Symbol          string                         `json:"S"` // only sent with the FUNDING_FEE reason type
}

// FuturesAccountUpdateBalance is a changed balance of a FuturesAccountUpdate
type FuturesAccountUpdateBalance struct {
	Asset              currency.Code `json:"a"`
	WalletBalance      types.Number  `json:"wb"`
	CrossWalletBalance types.Number  `json:"cw"`
	BalanceChange      types.Number  `json:"bc"` // excludes PNL and commission
}

// FuturesAccountUpdatePosition is a changed position of a FuturesAccountUpdate
type FuturesAccountUpdatePosition struct {
	Symbol              string       `json:"s"`
	PositionAmount      types.Number `json:"pa"`
	EntryPrice          types.Number `json:"ep"`
	BreakevenPrice      types.Number `json:"bep"`
	AccumulatedRealized types.Number `json:"cr"` // before fees
	UnrealizedPNL       types.Number `json:"up"`
	MarginType          string       `json:"mt"`
	IsolatedWallet      types.Number `json:"iw"`
	PositionSide        string       `json:"ps"`
}

// FuturesOrderTradeUpdate is a USDⓈ-M or COIN-M ORDER_TRADE_UPDATE user data event
type FuturesOrderTradeUpdate struct {
	EventType       string                       `json:"e"`
	EventTime       types.Time                   `json:"E"`
	TransactionTime types.Time                   `json:"T"`
	AccountAlias    string                       `json:"i"` // COIN-M only
	Order           FuturesOrderTradeUpdateOrder `json:"o"`
}

// FuturesOrderTradeUpdateOrder is the order of a FuturesOrderTradeUpdate. COIN-M sends BidsNotional and AskNotional
// as base asset quantities
type FuturesOrderTradeUpdateOrder struct {
	Symbol                         string        `json:"s"`
	ClientOrderID                  string        `json:"c"`
	Side                           string        `json:"S"`
	OrderType                      string        `json:"o"`
	TimeInForce                    string        `json:"f"`
	OriginalQuantity               types.Number  `json:"q"`
	OriginalPrice                  types.Number  `json:"p"`
	AveragePrice                   types.Number  `json:"ap"`
	StopPrice                      types.Number  `json:"sp"`
	ExecutionType                  string        `json:"x"`
	OrderStatus                    string        `json:"X"`
	OrderID                        uint64        `json:"i"`
	ModifyID                       string        `json:"M"` // only sent for AMENDMENT events of a modification with a modifyId
	OrderLastFilledQuantity        types.Number  `json:"l"`
	OrderFilledAccumulatedQuantity types.Number  `json:"z"`
	LastFilledPrice                types.Number  `json:"L"`
	MarginAsset                    currency.Code `json:"ma"` // COIN-M only
	CommissionAsset                currency.Code `json:"N"`
	Commission                     types.Number  `json:"n"`
	OrderTradeTime                 types.Time    `json:"T"`
	TradeID                        uint64        `json:"t"`
	BidsNotional                   types.Number  `json:"b"`
	AskNotional                    types.Number  `json:"a"`
	IsMaker                        bool          `json:"m"`
	IsReduceOnly                   bool          `json:"R"`
	StopPriceWorkingType           string        `json:"wt"`
	OriginalOrderType              string        `json:"ot"`
	PositionSide                   string        `json:"ps"`
	IsCloseAll                     bool          `json:"cp"`
	ActivationPrice                types.Number  `json:"AP"`
	CallbackRate                   types.Number  `json:"cr"`
	IsPriceProtected               bool          `json:"pP"`
	IgnoreSI                       int64         `json:"si"` // USDⓈ-M only, documented as ignore
	IgnoreSS                       int64         `json:"ss"` // USDⓈ-M only, documented as ignore
	RealizedProfit                 types.Number  `json:"rp"`
	SelfTradePreventionMode        string        `json:"V"`
	PriceMatchMode                 string        `json:"pm"`
	GoodTillDate                   types.Time    `json:"gtd"` // USDⓈ-M only
	ExpiryReason                   string        `json:"er"`
}

// FuturesAccountConfigUpdate is a USDⓈ-M or COIN-M ACCOUNT_CONFIG_UPDATE user data event
type FuturesAccountConfigUpdate struct {
	EventType                string                           `json:"e"`
	EventTime                types.Time                       `json:"E"`
	TransactionTime          types.Time                       `json:"T"`
	AccountConfiguration     *FuturesAccountConfiguration     `json:"ac"`
	UserAccountConfiguration *FuturesUserAccountConfiguration `json:"ai"` // USDⓈ-M only
}

// FuturesAccountConfiguration is the changed leverage of a trading pair in a FuturesAccountConfigUpdate
type FuturesAccountConfiguration struct {
	Symbol   string `json:"s"`
	Leverage uint64 `json:"l"`
}

// FuturesUserAccountConfiguration is the changed multi-assets mode in a FuturesAccountConfigUpdate
type FuturesUserAccountConfiguration struct {
	MultiAssetsMode bool `json:"j"`
}

// FuturesTradeLite is a USDⓈ-M TRADE_LITE user data event, a faster but smaller report of a fill
type FuturesTradeLite struct {
	EventType               string       `json:"e"`
	EventTime               types.Time   `json:"E"`
	TransactionTime         types.Time   `json:"T"`
	Symbol                  string       `json:"s"`
	OriginalQuantity        types.Number `json:"q"`
	OriginalPrice           types.Number `json:"p"`
	IsMaker                 bool         `json:"m"`
	ClientOrderID           string       `json:"c"`
	Side                    string       `json:"S"`
	LastFilledPrice         types.Number `json:"L"`
	OrderLastFilledQuantity types.Number `json:"l"`
	TradeID                 uint64       `json:"t"`
	OrderID                 uint64       `json:"i"`
}

// FuturesConditionalOrderTriggerReject is a USDⓈ-M CONDITIONAL_ORDER_TRIGGER_REJECT user data event, deprecated
// since 2025-12-15 in favour of the rejection reason in ALGO_UPDATE
type FuturesConditionalOrderTriggerReject struct {
	EventType       string                        `json:"e"`
	EventTime       types.Time                    `json:"E"`
	MessageSendTime types.Time                    `json:"T"`
	OrderReject     FuturesConditionalOrderReject `json:"or"`
}

// FuturesConditionalOrderReject is the rejected order of a FuturesConditionalOrderTriggerReject
type FuturesConditionalOrderReject struct {
	Symbol       string `json:"s"`
	OrderID      uint64 `json:"i"`
	RejectReason string `json:"r"`
}

// FuturesStrategyUpdate is a USDⓈ-M or COIN-M STRATEGY_UPDATE user data event
type FuturesStrategyUpdate struct {
	EventType       string                    `json:"e"`
	TransactionTime types.Time                `json:"T"`
	EventTime       types.Time                `json:"E"`
	StrategyUpdate  FuturesStrategyUpdateData `json:"su"`
}

// FuturesStrategyUpdateData is the strategy of a FuturesStrategyUpdate
type FuturesStrategyUpdateData struct {
	StrategyID     uint64     `json:"si"`
	StrategyType   string     `json:"st"`
	StrategyStatus string     `json:"ss"`
	Symbol         string     `json:"s"`
	UpdateTime     types.Time `json:"ut"`
	OpCode         uint64     `json:"c"`
}

// FuturesGridUpdate is a USDⓈ-M or COIN-M GRID_UPDATE user data event, which Binance has deprecated
type FuturesGridUpdate struct {
	EventType       string                `json:"e"`
	TransactionTime types.Time            `json:"T"`
	EventTime       types.Time            `json:"E"`
	GridUpdate      FuturesGridUpdateData `json:"gu"`
}

// FuturesGridUpdateData is the grid strategy of a FuturesGridUpdate
type FuturesGridUpdateData struct {
	StrategyID            uint64       `json:"si"`
	StrategyType          string       `json:"st"`
	StrategyStatus        string       `json:"ss"`
	Symbol                string       `json:"s"`
	RealizedPNL           types.Number `json:"r"`
	UnmatchedAveragePrice types.Number `json:"up"`
	UnmatchedQuantity     types.Number `json:"uq"`
	UnmatchedFee          types.Number `json:"uf"`
	MatchedPNL            types.Number `json:"mp"`
	UpdateTime            types.Time   `json:"ut"`
}

// FuturesAlgoUpdate is a USDⓈ-M ALGO_UPDATE user data event
type FuturesAlgoUpdate struct {
	EventType       string                 `json:"e"`
	TransactionTime types.Time             `json:"T"`
	EventTime       types.Time             `json:"E"`
	Order           FuturesAlgoUpdateOrder `json:"o"`
}

// FuturesAlgoUpdateOrder is the algo order of a FuturesAlgoUpdate. The matching engine fields are only sent once the
// order has been triggered
type FuturesAlgoUpdateOrder struct {
	ClientAlgoID            string       `json:"caid"`
	AlgoID                  uint64       `json:"aid"`
	AlgoType                string       `json:"at"`
	OrderType               string       `json:"o"`
	Symbol                  string       `json:"s"`
	Side                    string       `json:"S"`
	PositionSide            string       `json:"ps"`
	TimeInForce             string       `json:"f"`
	Quantity                types.Number `json:"q"`
	AlgoStatus              string       `json:"X"`
	OrderID                 string       `json:"ai"`
	AverageFillPrice        types.Number `json:"ap"`
	ExecutedQuantity        types.Number `json:"aq"`
	ActualOrderType         string       `json:"act"`
	TriggerPrice            types.Number `json:"tp"`
	OrderPrice              types.Number `json:"p"`
	SelfTradePreventionMode string       `json:"V"`
	WorkingType             string       `json:"wt"`
	PriceMatchMode          string       `json:"pm"`
	IsCloseAll              bool         `json:"cp"`
	IsPriceProtected        bool         `json:"pP"`
	IsReduceOnly            bool         `json:"R"`
	TriggerTime             types.Time   `json:"tt"`
	GoodTillDate            types.Time   `json:"gtd"`
	FailedReason            string       `json:"rm"`
	IsActivated             bool         `json:"ia"` // only meaningful for trailing stop orders
}

// ListenKeyExpired is the listenKeyExpired user data event, after which no user data arrives until a valid listen key
// is subscribed. Options sends the event time as a string
type ListenKeyExpired struct {
	EventType string     `json:"e"`
	EventTime types.Time `json:"E"`
	ListenKey string     `json:"listenKey"`
}

// OptionsDepthUpdate is an order book update from the options diff and partial book depth streams. Update IDs are
// int64 to match the order book API they feed
type OptionsDepthUpdate struct {
	EventType             string                           `json:"e"`
	EventTime             types.Time                       `json:"E"`
	TransactionTime       types.Time                       `json:"T"`
	Symbol                string                           `json:"s"`
	FirstUpdateID         int64                            `json:"U"`
	FinalUpdateID         int64                            `json:"u"`
	PreviousFinalUpdateID int64                            `json:"pu"`
	Bids                  orderbook.LevelsArrayPriceAmount `json:"b"`
	Asks                  orderbook.LevelsArrayPriceAmount `json:"a"`
}

// OptionsBookTicker is a best bid or ask update from the options <symbol>@bookTicker stream
type OptionsBookTicker struct {
	EventType       string       `json:"e"`
	UpdateID        uint64       `json:"u"`
	Symbol          string       `json:"s"`
	BestBidPrice    types.Number `json:"b"`
	BestBidQuantity types.Number `json:"B"`
	BestAskPrice    types.Number `json:"a"`
	BestAskQuantity types.Number `json:"A"`
	TransactionTime types.Time   `json:"T"`
	EventTime       types.Time   `json:"E"`
}

// OptionsTicker is a 24 hour ticker from the options <symbol>@optionTicker stream, which only pushes symbols whose
// ticker changed. PriceChangePercent is a fraction, unlike the futures tickers
type OptionsTicker struct {
	EventType            string       `json:"e"`
	EventTime            types.Time   `json:"E"`
	Symbol               string       `json:"s"`
	PriceChange          types.Number `json:"p"`
	PriceChangePercent   types.Number `json:"P"`
	WeightedAveragePrice types.Number `json:"w"`
	LastPrice            types.Number `json:"c"`
	LastQuantity         types.Number `json:"Q"`
	OpenPrice            types.Number `json:"o"`
	HighPrice            types.Number `json:"h"`
	LowPrice             types.Number `json:"l"`
	TradingVolume        types.Number `json:"v"` // in contracts
	TradeAmount          types.Number `json:"q"` // in the quote asset
	StatisticsOpenTime   types.Time   `json:"O"`
	StatisticsCloseTime  types.Time   `json:"C"`
	FirstTradeID         int64        `json:"F"`
	LastTradeID          int64        `json:"L"`
	TotalNumberOfTrades  uint64       `json:"n"`
}

// WsOptionsTrade is a trade from the options <symbol>@optionTrade stream
type WsOptionsTrade struct {
	EventType          string       `json:"e"`
	EventTime          types.Time   `json:"E"`
	TradeCompletedTime types.Time   `json:"T"`
	Symbol             string       `json:"s"`
	TradeID            uint64       `json:"t"`
	Price              types.Number `json:"p"`
	Quantity           types.Number `json:"q"`
	TradeType          string       `json:"X"` // MARKET for order book trades, BLOCK for block trades
	Direction          string       `json:"S"`
	IsBuyerMaker       bool         `json:"m"`
}

// OptionsIndexPrice is an underlying index price from the options !index@arr stream
type OptionsIndexPrice struct {
	EventType        string       `json:"e"`
	EventTime        types.Time   `json:"E"`
	UnderlyingSymbol string       `json:"s"`
	IndexPrice       types.Number `json:"p"`
}

// OptionsKline is a candle from the options <symbol>@kline_<interval> stream
type OptionsKline struct {
	EventType string           `json:"e"`
	EventTime types.Time       `json:"E"`
	Symbol    string           `json:"s"`
	Kline     OptionsKlineData `json:"k"`
}

// OptionsKlineData is the candle of an OptionsKline
type OptionsKlineData struct {
	StartTime                 types.Time   `json:"t"`
	EndTime                   types.Time   `json:"T"`
	Symbol                    string       `json:"s"`
	CandlePeriod              string       `json:"i"`
	FirstTradeID              int64        `json:"f"` // -1 when the candle has no trades
	LastTradeID               int64        `json:"L"`
	Open                      types.Number `json:"o"`
	Close                     types.Number `json:"c"`
	High                      types.Number `json:"h"`
	Low                       types.Number `json:"l"`
	Volume                    types.Number `json:"v"` // in contracts
	NumberOfTrades            uint64       `json:"n"`
	IsCompleted               bool         `json:"x"`
	CompletedTradeAmount      types.Number `json:"q"` // in the quote asset
	TakerCompletedTradeVolume types.Number `json:"V"` // in contracts
	TakerTradeAmount          types.Number `json:"Q"` // in the quote asset
}

// OptionsMarkPrice is the mark price and greeks of an option symbol from the options <underlying>@optionMarkPrice
// stream. An implied volatility of -1 means none is available
type OptionsMarkPrice struct {
	Symbol                string       `json:"s"`
	MarkPrice             types.Number `json:"mp"`
	EventTime             types.Time   `json:"E"`
	EventType             string       `json:"e"`
	IndexPrice            types.Number `json:"i"`
	EstimatedSettlePrice  types.Number `json:"P"` // only useful in the half hour before the settlement starts
	BestBuyPrice          types.Number `json:"bo"`
	BestSellPrice         types.Number `json:"ao"`
	BestBuyQuantity       types.Number `json:"bq"`
	BestSellQuantity      types.Number `json:"aq"`
	BuyImpliedVolatility  types.Number `json:"b"`
	SellImpliedVolatility types.Number `json:"a"`
	BuyMaximumPrice       types.Number `json:"hl"`
	SellMinimumPrice      types.Number `json:"ll"`
	Volatility            types.Number `json:"vo"`
	RiskFreeRate          types.Number `json:"rf"`
	Delta                 types.Number `json:"d"`
	Theta                 types.Number `json:"t"`
	Gamma                 types.Number `json:"g"`
	Vega                  types.Number `json:"v"`
}

// OptionsNewSymbol is a new option listing from the options !optionSymbol stream
type OptionsNewSymbol struct {
	EventType        string        `json:"e"`
	EventTime        types.Time    `json:"E"`
	Symbol           string        `json:"s"`
	Underlying       string        `json:"ps"`
	QuotationAsset   currency.Code `json:"qa"`
	OptionType       string        `json:"d"`
	StrikePrice      types.Number  `json:"sp"`
	DeliveryDateTime types.Time    `json:"dt"`
	Unit             uint64        `json:"u"` // quantity of the underlying asset one contract represents
	OnboardDateTime  types.Time    `json:"ot"`
	ContractStatus   string        `json:"cs"`
}

// WsOptionsOpenInterest is the open interest of an option symbol from the options
// <underlying>@openInterest@<expirationDate> stream
type WsOptionsOpenInterest struct {
	EventType        string       `json:"e"`
	EventTime        types.Time   `json:"E"`
	Symbol           string       `json:"s"`
	OpenInterest     types.Number `json:"o"` // in contracts
	OpenInterestUSDT types.Number `json:"h"`
}

// OptionsAccountUpdate is an options ACCOUNT_UPDATE user data event, pushed on balance or position changes and every
// 10 seconds while a position is open. Amounts are in USDT
type OptionsAccountUpdate struct {
	EventType         string       `json:"e"`
	EventTime         types.Time   `json:"E"`
	TransactionTime   types.Time   `json:"T"`
	Equity            types.Number `json:"eq"`
	AdjustedEquity    types.Number `json:"aeq"`
	WalletBalance     types.Number `json:"b"`
	PositionValue     types.Number `json:"m"`
	UnrealizedPNL     types.Number `json:"u"`
	InitialMargin     types.Number `json:"i"`
	MaintenanceMargin types.Number `json:"M"`
}

// OptionsBalancePositionUpdate is an options BALANCE_POSITION_UPDATE user data event
type OptionsBalancePositionUpdate struct {
	EventType       string                  `json:"e"`
	EventTime       types.Time              `json:"E"`
	TransactionTime types.Time              `json:"T"`
	EventReasonType string                  `json:"m"`
	Balances        []OptionsBalanceUpdate  `json:"B"`
	Positions       []OptionsPositionUpdate `json:"P"`
}

// OptionsBalanceUpdate is a changed balance of an OptionsBalancePositionUpdate
type OptionsBalanceUpdate struct {
	MarginAsset    currency.Code `json:"a"`
	AccountBalance types.Number  `json:"b"`
	BalanceChange  types.Number  `json:"bc"` // excludes PNL and commission
}

// OptionsPositionUpdate is a changed position of an OptionsBalancePositionUpdate
type OptionsPositionUpdate struct {
	Symbol            string       `json:"s"`
	PositionQuantity  types.Number `json:"c"`
	PositionValue     types.Number `json:"p"`
	AverageEntryPrice types.Number `json:"a"`
}

// OptionsOrderTradeUpdate is an options ORDER_TRADE_UPDATE user data event
type OptionsOrderTradeUpdate struct {
	EventType       string                       `json:"e"`
	EventTime       types.Time                   `json:"E"`
	TransactionTime types.Time                   `json:"T"`
	Order           OptionsOrderTradeUpdateOrder `json:"o"`
}

// OptionsOrderTradeUpdateOrder is the order of an OptionsOrderTradeUpdate
type OptionsOrderTradeUpdateOrder struct {
	Symbol                         string        `json:"s"`
	ClientOrderID                  string        `json:"c"`
	Side                           string        `json:"S"`
	OrderType                      string        `json:"o"`
	TimeInForce                    string        `json:"f"`
	OriginalQuantity               types.Number  `json:"q"`
	OriginalPrice                  types.Number  `json:"p"`
	AveragePrice                   types.Number  `json:"ap"`
	ExecutionType                  string        `json:"x"`
	OrderStatus                    string        `json:"X"`
	OrderID                        uint64        `json:"i"`
	OrderLastFilledQuantity        types.Number  `json:"l"`
	OrderFilledAccumulatedQuantity types.Number  `json:"z"`
	LastFilledPrice                types.Number  `json:"L"`
	CommissionAsset                currency.Code `json:"N"`
	Commission                     types.Number  `json:"n"` // negative means a fee was charged
	OrderTradeTime                 types.Time    `json:"T"`
	TradeID                        uint64        `json:"t"`
	BidsQuantity                   types.Number  `json:"b"`
	AskQuantity                    types.Number  `json:"a"`
	IsMaker                        bool          `json:"m"`
	IsReduceOnly                   bool          `json:"R"`
	OriginalOrderType              string        `json:"ot"`
	RealizedProfit                 types.Number  `json:"rp"`
	SelfTradePreventionMode        string        `json:"V"`
}

// OptionsGreekUpdate is an options GREEK_UPDATE user data event, pushed on position changes and every 10 seconds while
// a position is open
type OptionsGreekUpdate struct {
	EventType       string                   `json:"e"`
	EventTime       types.Time               `json:"E"`
	TransactionTime types.Time               `json:"T"`
	Greeks          []OptionsGreekUpdateItem `json:"G"`
}

// OptionsGreekUpdateItem holds the greeks of an underlying in an OptionsGreekUpdate
type OptionsGreekUpdateItem struct {
	Underlying string       `json:"u"`
	Delta      types.Number `json:"d"`
	Gamma      types.Number `json:"g"`
	Theta      types.Number `json:"t"`
	Vega       types.Number `json:"v"`
}

// OptionsRiskLevelChange is an options RISK_LEVEL_CHANGE user data event, which only VIP and market maker accounts
// receive
type OptionsRiskLevelChange struct {
	EventType         string       `json:"e"`
	EventTime         types.Time   `json:"E"`
	RiskLevel         string       `json:"s"`
	MarginBalance     types.Number `json:"mb"`
	MaintenanceMargin types.Number `json:"mm"`
}
