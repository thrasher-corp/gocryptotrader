package bingx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WebsocketSubscriptionRequest is a subscription or unsubscription request
type WebsocketSubscriptionRequest struct {
	ID          string               `json:"id"`
	RequestType WebsocketRequestType `json:"reqType"`
	DataType    string               `json:"dataType"`
}

// WebsocketSubscriptionResponse is the reply to a subscription or unsubscription request. Spot and coin-M replies carry a
// timestamp, USDT-M replies an empty dataType, and coin-M error replies no ID
type WebsocketSubscriptionResponse struct {
	ID string `json:"id"`
	// Code is int64 as APIError carries it
	Code      int64      `json:"code"`
	Message   string     `json:"msg"`
	DataType  string     `json:"dataType"`
	Timestamp types.Time `json:"timestamp"`
}

// WebsocketPing is the heartbeat the spot and coin-M servers send
type WebsocketPing struct {
	Ping string `json:"ping"`
	// Time is the server's local time as text, such as 2026-11-11T17:53:38.549+0800, which the pong echoes unchanged
	Time string `json:"time"`
}

// WebsocketPongRequest answers a WebsocketPing with its ID and time
type WebsocketPongRequest struct {
	Pong string `json:"pong"`
	// Time echoes the ping's time text unchanged
	Time string `json:"time"`
}

// WebsocketSpotTrade is a spot trade stream payload
type WebsocketSpotTrade struct {
	EventType    string       `json:"e"`
	EventTime    types.Time   `json:"E"`
	Symbol       string       `json:"s"`
	TradeID      NumericID    `json:"t"`
	Price        types.Number `json:"p"`
	Quantity     types.Number `json:"q"`
	TradeTime    types.Time   `json:"T"`
	IsBuyerMaker bool         `json:"m"`
}

// WebsocketSpotKline is a spot candle stream payload
type WebsocketSpotKline struct {
	EventType string              `json:"e"`
	EventTime types.Time          `json:"E"`
	Symbol    string              `json:"s"`
	Kline     WebsocketSpotCandle `json:"K"`
}

// WebsocketSpotCandle is the candle of a spot candle stream payload
type WebsocketSpotCandle struct {
	StartTime      types.Time   `json:"t"`
	EndTime        types.Time   `json:"T"`
	Symbol         string       `json:"s"`
	Interval       string       `json:"i"`
	Open           types.Number `json:"o"`
	Close          types.Number `json:"c"`
	High           types.Number `json:"h"`
	Low            types.Number `json:"l"`
	Volume         types.Number `json:"v"`
	Turnover       types.Number `json:"q"`
	NumberOfTrades uint64       `json:"n"`
}

// WebsocketSpotOrderbook is a spot partial depth stream payload, whose asks arrive highest first
type WebsocketSpotOrderbook struct {
	Bids orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks orderbook.LevelsArrayPriceAmount `json:"asks"`
	// LastUpdateID is int64 as orderbook.Book holds update IDs
	LastUpdateID int64 `json:"lastUpdateId"`
}

// WebsocketSpotTicker is a spot 24 hour price change stream payload
type WebsocketSpotTicker struct {
	EventType             string       `json:"e"`
	EventTime             types.Time   `json:"E"`
	Symbol                string       `json:"s"`
	PriceChange           types.Number `json:"p"`
	PriceChangePercentage Percentage   `json:"P"`
	OpenPrice             types.Number `json:"o"`
	HighestPrice          types.Number `json:"h"`
	LowestPrice           types.Number `json:"l"`
	LatestTradePrice      types.Number `json:"c"`
	Volume                types.Number `json:"v"`
	Turnover              types.Number `json:"q"`
	StatisticsStartTime   types.Time   `json:"O"`
	StatisticsEndTime     types.Time   `json:"C"`
	BestBidPrice          types.Number `json:"B"`
	BestBidQuantity       types.Number `json:"b"`
	BestAskPrice          types.Number `json:"A"`
	BestAskQuantity       types.Number `json:"a"`
	NumberOfTrades        uint64       `json:"n"`
}

// WebsocketSpotLastPrice is a spot latest trade price stream payload
type WebsocketSpotLastPrice struct {
	EventType string       `json:"e"`
	Symbol    string       `json:"s"`
	Price     types.Number `json:"c"`
}

// WebsocketSpotBookTicker is a spot best order book stream payload
type WebsocketSpotBookTicker struct {
	EventType       string       `json:"e"`
	EventPushTime   types.Time   `json:"E"`
	Symbol          string       `json:"s"`
	BestBidPrice    types.Number `json:"b"`
	BestBidQuantity types.Number `json:"B"`
	BestAskPrice    types.Number `json:"a"`
	BestAskQuantity types.Number `json:"A"`
}

// WebsocketSwapTrade is a trade of a USDT-M perpetual trade stream payload, which holds an array of them
type WebsocketSwapTrade struct {
	TradeID   NumericID    `json:"t"`
	TradeTime types.Time   `json:"T"`
	Symbol    string       `json:"s"`
	Price     types.Number `json:"p"`
	Quantity  types.Number `json:"q"`
	// IsBuyerMaker is documented as a string and sent as a boolean
	IsBuyerMaker types.Boolean `json:"m"`
}

// WebsocketSwapKline is a candle of a USDT-M perpetual candle stream payload, which holds an array of them
type WebsocketSwapKline struct {
	Open   types.Number `json:"o"`
	Close  types.Number `json:"c"`
	High   types.Number `json:"h"`
	Low    types.Number `json:"l"`
	Volume types.Number `json:"v"`
	// Time is undocumented; the candle's start time
	Time types.Time `json:"T"`
}

// WebsocketSwapOrderbook is a USDT-M perpetual partial depth stream payload
type WebsocketSwapOrderbook struct {
	Bids orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// WebsocketSwapTicker is a USDT-M perpetual 24 hour price change stream payload
type WebsocketSwapTicker struct {
	EventType               string       `json:"e"`
	EventTime               types.Time   `json:"E"`
	Symbol                  string       `json:"s"`
	PriceChange             types.Number `json:"p"`
	PriceChangePercentage   Percentage   `json:"P"`
	FirstPrice              types.Number `json:"o"`
	HighPrice               types.Number `json:"h"`
	LowPrice                types.Number `json:"l"`
	LatestTransactionPrice  types.Number `json:"c"`
	LatestTransactionAmount types.Number `json:"L"`
	Volume                  types.Number `json:"v"`
	// Turnover is in USDT
	Turnover             types.Number `json:"q"`
	FirstTransactionTime types.Time   `json:"O"`
	LastTransactionTime  types.Time   `json:"C"`
	BestBidPrice         types.Number `json:"B"`
	BestBidQuantity      types.Number `json:"b"`
	BestAskPrice         types.Number `json:"A"`
	BestAskQuantity      types.Number `json:"a"`
}

// WebsocketSwapLastPrice is a USDT-M perpetual latest price stream payload
type WebsocketSwapLastPrice struct {
	EventType string       `json:"e"`
	EventTime types.Time   `json:"E"`
	Symbol    string       `json:"s"`
	LastPrice types.Number `json:"c"`
}

// WebsocketMarkPrice is a USDT-M or coin-M perpetual mark price stream payload
type WebsocketMarkPrice struct {
	EventType string     `json:"e"`
	EventTime types.Time `json:"E"`
	Symbol    string     `json:"s"`
	// Price is the mark price, which the docs call the last price
	Price types.Number `json:"p"`
}

// WebsocketSwapBookTicker is a USDT-M perpetual best bid and ask stream payload
type WebsocketSwapBookTicker struct {
	EventType       string       `json:"e"`
	UpdateID        uint64       `json:"u"`
	EventPushTime   types.Time   `json:"E"`
	MatchTime       types.Time   `json:"T"`
	Symbol          string       `json:"s"`
	BestBidPrice    types.Number `json:"b"`
	BestBidQuantity types.Number `json:"B"`
	BestAskPrice    types.Number `json:"a"`
	BestAskQuantity types.Number `json:"A"`
}

// WebsocketCoinMarginedTrade is a coin-M perpetual trade stream payload
type WebsocketCoinMarginedTrade struct {
	EventType string       `json:"e"`
	EventTime types.Time   `json:"E"`
	Symbol    string       `json:"s"`
	TradeID   NumericID    `json:"t"`
	Price     types.Number `json:"p"`
	// Quantity counts contracts
	Quantity     types.Number `json:"q"`
	TradeTime    types.Time   `json:"T"`
	IsBuyerMaker bool         `json:"m"`
}

// WebsocketCoinMarginedLastPrice is a coin-M perpetual latest transaction price stream payload
type WebsocketCoinMarginedLastPrice struct {
	EventType   string       `json:"e"`
	EventTime   types.Time   `json:"E"`
	Symbol      string       `json:"s"`
	LatestPrice types.Number `json:"p"`
}

// WebsocketCoinMarginedOrderbook is a coin-M perpetual limited depth stream payload
type WebsocketCoinMarginedOrderbook struct {
	Symbol               string                                `json:"symbol"`
	Bids                 []WebsocketCoinMarginedOrderbookLevel `json:"bids"`
	Asks                 []WebsocketCoinMarginedOrderbookLevel `json:"asks"`
	AggregationPrecision types.Number                          `json:"aggPrecision"`
	Timestamp            types.Time                            `json:"timestamp"`
}

// WebsocketCoinMarginedOrderbookLevel is a price level of a coin-M perpetual limited depth stream payload
type WebsocketCoinMarginedOrderbookLevel struct {
	Price types.Number `json:"p"`
	// Amount is undocumented; the level's size in the base coin
	Amount types.Number `json:"a"`
	// Volume is undocumented; the level's size in contracts
	Volume types.Number `json:"v"`
}

// WebsocketCoinMarginedBookTicker is a coin-M perpetual best bid and ask stream payload
type WebsocketCoinMarginedBookTicker struct {
	EventType            string     `json:"e"`
	DepthUpdateTime      types.Time `json:"E"`
	Symbol               string     `json:"s"`
	LatestTradeTimestamp types.Time `json:"T"`
	// LatestTradePrice is the price of the trade TradeID identifies
	LatestTradePrice types.Number `json:"p"`
	TradeID          NumericID    `json:"u"`
	BestBidPrice     types.Number `json:"b"`
	// BestBidQuantity counts contracts
	BestBidQuantity types.Number `json:"B"`
	BestAskPrice    types.Number `json:"a"`
	// BestAskQuantity counts contracts
	BestAskQuantity types.Number `json:"A"`
}

// WebsocketCoinMarginedKline is a coin-M perpetual candle stream payload
type WebsocketCoinMarginedKline struct {
	StartTime types.Time   `json:"t"`
	Symbol    string       `json:"s"`
	Open      types.Number `json:"o"`
	Close     types.Number `json:"c"`
	High      types.Number `json:"h"`
	Low       types.Number `json:"l"`
	// Amount is the quantity traded in the base coin
	Amount types.Number `json:"a"`
	// Volume is the number of contracts traded
	Volume         types.Number `json:"v"`
	NumberOfTrades uint64       `json:"u"`
}

// WebsocketCoinMarginedTicker is a coin-M perpetual 24 hour price change stream payload
type WebsocketCoinMarginedTicker struct {
	Symbol                string       `json:"s"`
	EventTime             types.Time   `json:"E"`
	OpenPrice             types.Number `json:"o"`
	HighestPrice          types.Number `json:"h"`
	LowestPrice           types.Number `json:"l"`
	PriceChange           types.Number `json:"p"`
	PriceChangePercentage Percentage   `json:"P"`
	LatestPrice           types.Number `json:"c"`
	LatestTradeTime       types.Time   `json:"C"`
	// LatestTradeQuantity counts contracts
	LatestTradeQuantity types.Number `json:"L"`
	// Volume counts contracts
	Volume types.Number `json:"v"`
	// TradeAmount is in the base coin
	TradeAmount types.Number `json:"m"`
	// QuoteTradeAmount is in USD
	QuoteTradeAmount types.Number `json:"q"`
	BestBidPrice     types.Number `json:"B"`
	// BestBidQuantity counts contracts
	BestBidQuantity types.Number `json:"b"`
	BestAskPrice    types.Number `json:"A"`
	// BestAskQuantity counts contracts
	BestAskQuantity types.Number `json:"a"`
}

// WebsocketSpotOrderUpdate is a spot order update event, executionReport
type WebsocketSpotOrderUpdate struct {
	UserUID                  uint64        `json:"u"`
	EventType                string        `json:"e"`
	EventTime                types.Time    `json:"E"`
	Symbol                   string        `json:"s"`
	Side                     Side          `json:"S"`
	OrderType                OrderType     `json:"o"`
	OriginalQuantity         types.Number  `json:"q"`
	OriginalPrice            types.Number  `json:"p"`
	ExecutionType            ExecutionType `json:"x"`
	OrderStatus              string        `json:"X"`
	OrderID                  NumericID     `json:"i"`
	LastFilledQuantity       types.Number  `json:"l"`
	CumulativeFilledQuantity types.Number  `json:"z"`
	LastFilledPrice          types.Number  `json:"L"`
	// FeeAmount is negative when the fee is charged
	FeeAmount              types.Number  `json:"n"`
	Reserved               string        `json:"U"`
	MainAccountUID         string        `json:"mu"`
	ClientOrderID          string        `json:"C"`
	FeeAsset               currency.Code `json:"N"`
	TradeTime              types.Time    `json:"T"`
	TradeID                NumericID     `json:"t"`
	OrderCreationTime      types.Time    `json:"O"`
	CumulativeFilledAmount types.Number  `json:"Z"`
	LastFilledAmount       types.Number  `json:"Y"`
	OriginalOrderAmount    types.Number  `json:"Q"`
	IsMaker                bool          `json:"m"`
	// CumulativeFilledAmountAlias is documented as another cumulative filled amount, usually equal to
	// CumulativeFilledAmount
	CumulativeFilledAmountAlias types.Number `json:"A"`
	WebsocketPushTime           types.Time   `json:"ws"`
}

// WebsocketSpotAccountUpdate is a spot account balance event, ACCOUNT_UPDATE
type WebsocketSpotAccountUpdate struct {
	EventType string               `json:"e"`
	EventTime types.Time           `json:"E"`
	Account   WebsocketSpotAccount `json:"a"`
}

// WebsocketSpotAccount is the content of a spot account balance event
type WebsocketSpotAccount struct {
	Reason   AccountUpdateReason    `json:"m"`
	Balances []WebsocketSpotBalance `json:"B"`
}

// WebsocketSpotBalance is a balance of a spot account balance event. The docs describe both cw and wb as the asset's
// total after the change
type WebsocketSpotBalance struct {
	Asset              currency.Code `json:"a"`
	BalanceChange      types.Number  `json:"bc"`
	CrossWalletBalance types.Number  `json:"cw"`
	WalletBalance      types.Number  `json:"wb"`
	FrozenAssets       types.Number  `json:"lk"`
}

// WebsocketPerpetualOrderUpdate is a USDT-M or coin-M perpetual order event: ORDER_TRADE_UPDATE, or TRADE_UPDATE as the
// USDT-M docs example names it
type WebsocketPerpetualOrderUpdate struct {
	EventType string     `json:"e"`
	EventTime types.Time `json:"E"`
	// PushTime is undocumented and sent by the USDT-M docs example alone
	PushTime types.Time              `json:"T"`
	Order    WebsocketPerpetualOrder `json:"o"`
}

// WebsocketPerpetualOrder is the order of a perpetual order event. It reports a fill's fee, but not the fill's own
// quantity or price
type WebsocketPerpetualOrder struct {
	Symbol        string        `json:"s"`
	ClientOrderID string        `json:"c"`
	OrderID       NumericID     `json:"i"`
	Side          Side          `json:"S"`
	OrderType     OrderType     `json:"o"`
	Quantity      types.Number  `json:"q"`
	Price         types.Number  `json:"p"`
	TriggerPrice  types.Number  `json:"sp"`
	AverageFilled types.Number  `json:"ap"`
	ExecutionType ExecutionType `json:"x"`
	OrderStatus   string        `json:"X"`
	FeeAsset      currency.Code `json:"N"`
	// Fee is negative when the fee is charged
	Fee                      types.Number   `json:"n"`
	TradeTime                types.Time     `json:"T"`
	TriggerPriceType         WorkingType    `json:"wt"`
	PositionSide             PositionSide   `json:"ps"`
	RealisedPNL              types.Number   `json:"rp"`
	CumulativeFilledQuantity types.Number   `json:"z"`
	GuaranteedStopLoss       StopGuaranteed `json:"sg"`
	ConditionalOrderID       NumericID      `json:"ti"`
	ParentOrderID            string         `json:"li"`
	// ReduceOnly is documented as a boolean; types.Boolean also decodes it should it arrive as a string, as sg does
	ReduceOnly types.Boolean `json:"ro"`
	TradeID    NumericID     `json:"td"`
	TradeValue types.Number  `json:"tv"`
}

// WebsocketPerpetualAccountUpdate is a USDT-M or coin-M perpetual account event, ACCOUNT_UPDATE
type WebsocketPerpetualAccountUpdate struct {
	EventType string                    `json:"e"`
	EventTime types.Time                `json:"E"`
	PushTime  types.Time                `json:"T"`
	Account   WebsocketPerpetualAccount `json:"a"`
}

// WebsocketPerpetualAccount is the content of a perpetual account event. Positions are pushed only when
// they change
type WebsocketPerpetualAccount struct {
	Reason    AccountUpdateReason          `json:"m"`
	Balances  []WebsocketPerpetualBalance  `json:"B"`
	Positions []WebsocketPerpetualPosition `json:"P"`
}

// WebsocketPerpetualBalance is a balance of a perpetual account event
type WebsocketPerpetualBalance struct {
	Asset         currency.Code `json:"a"`
	WalletBalance types.Number  `json:"wb"`
	// CrossWalletBalance is the wallet balance excluding isolated margin; coin-M events leave it out
	CrossWalletBalance types.Number `json:"cw"`
	BalanceChange      types.Number `json:"bc"`
}

// WebsocketPerpetualPosition is a position of a perpetual account event
type WebsocketPerpetualPosition struct {
	Symbol         string       `json:"s"`
	PositionAmount types.Number `json:"pa"`
	EntryPrice     types.Number `json:"ep"`
	UnrealisedPNL  types.Number `json:"up"`
	// MarginMode is cross or isolated, in either case
	MarginMode             string       `json:"mt"`
	IsolatedPositionMargin types.Number `json:"iw"`
	PositionSide           PositionSide `json:"ps"`
	RealisedPNL            types.Number `json:"cr"`
}

// WebsocketPerpetualConfigUpdate is a USDT-M or coin-M perpetual leverage and margin mode event, ACCOUNT_CONFIG_UPDATE
type WebsocketPerpetualConfigUpdate struct {
	EventType     string                   `json:"e"`
	EventTime     types.Time               `json:"E"`
	Configuration WebsocketPerpetualConfig `json:"ac"`
}

// WebsocketPerpetualConfig is a symbol's configuration in a perpetual configuration event
type WebsocketPerpetualConfig struct {
	Symbol        string `json:"s"`
	LongLeverage  uint64 `json:"l"`
	ShortLeverage uint64 `json:"S"`
	// MarginMode is cross or isolated, in either case
	MarginMode string `json:"mt"`
}

// WebsocketListenKeyExpired is the event an account connection may receive when its listen key expires. It is
// undocumented, so only its event type and time are read
type WebsocketListenKeyExpired struct {
	EventType string     `json:"e"`
	EventTime types.Time `json:"E"`
}
