package binanceus

import (
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WebsocketPayloadRequest is a WebSocket Streams subscription request: SUBSCRIBE or UNSUBSCRIBE with the stream names
type WebsocketPayloadRequest struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     string   `json:"id"`
}

// TradeStream is a Trade Data Stream (<symbol>@trade) payload
type TradeStream struct {
	EventType    string       `json:"e"`
	EventTime    types.Time   `json:"E"`
	Symbol       string       `json:"s"`
	TradeID      uint64       `json:"t"`
	Price        types.Number `json:"p"`
	Quantity     types.Number `json:"q"`
	TradeTime    types.Time   `json:"T"`
	IsBuyerMaker bool         `json:"m"`
	Ignore       bool         `json:"M"`
}

// TickerStream is an Individual Symbol 24h Change Stream (<symbol>@ticker) payload: 24hr rolling window statistics
type TickerStream struct {
	EventType          string       `json:"e"`
	EventTime          types.Time   `json:"E"`
	Symbol             string       `json:"s"`
	PriceChange        types.Number `json:"p"`
	PriceChangePercent types.Number `json:"P"`
	// WeightedAveragePrice is the volume weighted average price of the window
	WeightedAveragePrice types.Number `json:"w"`
	// PreviousClosePrice is the price of the last trade before the window, which the REST ticker calls prevClosePrice
	PreviousClosePrice          types.Number `json:"x"`
	LastPrice                   types.Number `json:"c"`
	LastQuantity                types.Number `json:"Q"`
	BestBidPrice                types.Number `json:"b"`
	BestBidQuantity             types.Number `json:"B"`
	BestAskPrice                types.Number `json:"a"`
	BestAskQuantity             types.Number `json:"A"`
	OpenPrice                   types.Number `json:"o"`
	HighPrice                   types.Number `json:"h"`
	LowPrice                    types.Number `json:"l"`
	TotalTradedBaseAssetVolume  types.Number `json:"v"`
	TotalTradedQuoteAssetVolume types.Number `json:"q"`
	StatisticsOpenTime          types.Time   `json:"O"`
	StatisticsCloseTime         types.Time   `json:"C"`
	FirstTradeID                int64        `json:"F"` // -1 when the window has no trades
	LastTradeID                 int64        `json:"L"` // -1 when the window has no trades
	TotalNumberOfTrades         uint64       `json:"n"`
}

// KlineStream is a Candlestick Data Stream (<symbol>@kline_<interval>) payload
type KlineStream struct {
	EventType string     `json:"e"`
	EventTime types.Time `json:"E"`
	Symbol    string     `json:"s"`
	Kline     SpotCandle `json:"k"`
}

// SpotCandle is the candle of a kline stream payload
type SpotCandle struct {
	StartTime                types.Time   `json:"t"`
	CloseTime                types.Time   `json:"T"`
	Symbol                   string       `json:"s"`
	Interval                 string       `json:"i"`
	FirstTradeID             int64        `json:"f"` // -1 when the candle has no trades
	LastTradeID              int64        `json:"L"` // -1 when the candle has no trades
	OpenPrice                types.Number `json:"o"`
	ClosePrice               types.Number `json:"c"`
	HighPrice                types.Number `json:"h"`
	LowPrice                 types.Number `json:"l"`
	BaseAssetVolume          types.Number `json:"v"`
	NumberOfTrades           uint64       `json:"n"`
	IsClosed                 bool         `json:"x"`
	QuoteAssetVolume         types.Number `json:"q"`
	TakerBuyBaseAssetVolume  types.Number `json:"V"`
	TakerBuyQuoteAssetVolume types.Number `json:"Q"`
	Ignore                   types.Number `json:"B"`
}

// DiffDepthStream is an Order Book Depth Diff Stream (<symbol>@depth) payload: the new quantity of each changed price
// level
type DiffDepthStream struct {
	EventType     string                           `json:"e"`
	EventTime     types.Time                       `json:"E"`
	Symbol        string                           `json:"s"`
	FirstUpdateID int64                            `json:"U"` // int64 to match the order book API it feeds
	FinalUpdateID int64                            `json:"u"` // int64 to match the order book API it feeds
	Bids          orderbook.LevelsArrayPriceAmount `json:"b"`
	Asks          orderbook.LevelsArrayPriceAmount `json:"a"`
}

// WebsocketPartialDepth is a Partial Order Book Depth Stream (<symbol>@depth<levels>) payload: the top levels of the
// book
type WebsocketPartialDepth struct {
	// LastUpdateID is signed like the order book's update IDs it is loaded into
	LastUpdateID int64                            `json:"lastUpdateId"`
	Bids         orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks         orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// WebsocketServerShutdown is the serverShutdown event, sent on the stream and WebSocket API connections when the server
// is about to shut down
type WebsocketServerShutdown struct {
	EventType string     `json:"e"`
	EventTime types.Time `json:"E"`
}
