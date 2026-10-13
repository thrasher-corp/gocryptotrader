package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WsTicker is a Ticker or All Tickers push
type WsTicker struct {
	Sequence    types.Number `json:"sequence"`
	Price       types.Number `json:"price"`
	Size        types.Number `json:"size"`
	BestAsk     types.Number `json:"bestAsk"`
	BestAskSize types.Number `json:"bestAskSize"`
	BestBid     types.Number `json:"bestBid"`
	BestBidSize types.Number `json:"bestBidSize"`
	Time        types.Time   `json:"time"` // The matching time of the latest trade
}

// WsOrderbookLevel1 is an Orderbook - Level 1 push
type WsOrderbookLevel1 struct {
	Asks      [2]types.Number `json:"asks"` // Price, size
	Bids      [2]types.Number `json:"bids"` // Price, size
	Timestamp types.Time      `json:"timestamp"`
}

// WsOrderbookDepth is an Orderbook - Level 5 or Orderbook - Level 50 push
type WsOrderbookDepth struct {
	Asks      orderbook.LevelsArrayPriceAmount `json:"asks"`
	Bids      orderbook.LevelsArrayPriceAmount `json:"bids"`
	Timestamp types.Time                       `json:"timestamp"`
}

// WsOrderbookIncrement is an Orderbook - Increment push
type WsOrderbookIncrement struct {
	Changes       WsOrderbookChanges `json:"changes"`
	SequenceEnd   int64              `json:"sequenceEnd"`
	SequenceStart int64              `json:"sequenceStart"`
	Symbol        string             `json:"symbol"`
	Time          types.Time         `json:"time"`
}

// WsOrderbookChanges are the changed levels of an Orderbook - Increment push
type WsOrderbookChanges struct {
	Asks [][3]types.Number `json:"asks"` // Price, size, sequence
	Bids [][3]types.Number `json:"bids"` // Price, size, sequence
}

// WsCallAuctionOrderbook is a Call Auction Orderbook - Level 50 push
type WsCallAuctionOrderbook struct {
	Symbol    string                           `json:"symbol"` // From the topic; the push does not carry it
	Asks      orderbook.LevelsArrayPriceAmount `json:"asks"`
	Bids      orderbook.LevelsArrayPriceAmount `json:"bids"`
	Timestamp types.Time                       `json:"timestamp"`
}

// WsCallAuctionInfo is a Call Auction Data push
type WsCallAuctionInfo struct {
	Symbol                  string       `json:"symbol"`
	EstimatedPrice          types.Number `json:"estimatedPrice"`
	EstimatedSize           types.Number `json:"estimatedSize"`
	SellOrderRangeLowPrice  types.Number `json:"sellOrderRangeLowPrice"`
	SellOrderRangeHighPrice types.Number `json:"sellOrderRangeHighPrice"`
	BuyOrderRangeLowPrice   types.Number `json:"buyOrderRangeLowPrice"`
	BuyOrderRangeHighPrice  types.Number `json:"buyOrderRangeHighPrice"`
	Time                    types.Time   `json:"time"`
}

// WsCandlestick is a Klines push
type WsCandlestick struct {
	Symbol  string     `json:"symbol"`
	Candles WsCandle   `json:"candles"`
	Time    types.Time `json:"time"`
}

// WsCandle is the candle of a spot Klines push, sent as an array
type WsCandle struct {
	StartTime         types.Time
	OpenPrice         types.Number
	ClosePrice        types.Number
	HighPrice         types.Number
	LowPrice          types.Number
	TransactionVolume types.Number
	TransactionAmount types.Number
}

// UnmarshalJSON decodes the candle array of a spot Klines push
func (c *WsCandle) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&c.StartTime, &c.OpenPrice, &c.ClosePrice, &c.HighPrice, &c.LowPrice, &c.TransactionVolume, &c.TransactionAmount})
}

// WsTrade is a spot Trade push
type WsTrade struct {
	MakerOrderID string       `json:"makerOrderId"`
	Price        types.Number `json:"price"`
	Sequence     string       `json:"sequence"` // Quoted and wider than a float64 holds exactly
	Side         string       `json:"side"`
	Size         types.Number `json:"size"`
	Symbol       string       `json:"symbol"`
	TakerOrderID string       `json:"takerOrderId"`
	Time         types.Time   `json:"time"`
	TradeID      string       `json:"tradeId"`
	Type         string       `json:"type"`
}

// WsSnapshot is a Symbol Snapshot or Market Snapshot push
type WsSnapshot struct {
	Sequence types.Number   `json:"sequence"` // Quoted on a symbol snapshot and bare on a market snapshot
	Data     WsSnapshotData `json:"data"`
}

// WsSnapshotData is the snapshot of a symbol
type WsSnapshotData struct {
	AskSize            float64                `json:"askSize"`
	AveragePrice       float64                `json:"averagePrice"`
	BaseCurrency       currency.Code          `json:"baseCurrency"`
	BidSize            float64                `json:"bidSize"`
	Board              uint64                 `json:"board"` // Trading pair partition: 0 primary, 1 KuCoin Plus
	Buy                float64                `json:"buy"`
	ChangePrice        float64                `json:"changePrice"`
	ChangeRate         float64                `json:"changeRate"`
	Close              float64                `json:"close"`
	Datetime           types.Time             `json:"datetime"`
	High               float64                `json:"high"`
	LastSize           float64                `json:"lastSize"`
	LastTradedPrice    float64                `json:"lastTradedPrice"`
	Low                float64                `json:"low"`
	MakerCoefficient   float64                `json:"makerCoefficient"`
	MakerFeeRate       float64                `json:"makerFeeRate"`
	MarginTrade        bool                   `json:"marginTrade"`
	Mark               uint64                 `json:"mark"` // Trading pair mark: 0 default, 1 ST, 2 NEW
	Market             string                 `json:"market"`
	MarketChange1Hour  WsSnapshotMarketChange `json:"marketChange1h"`
	MarketChange24Hour WsSnapshotMarketChange `json:"marketChange24h"`
	MarketChange4Hour  WsSnapshotMarketChange `json:"marketChange4h"`
	Markets            []string               `json:"markets"`
	Open               float64                `json:"open"`
	QuoteCurrency      currency.Code          `json:"quoteCurrency"`
	Sell               float64                `json:"sell"`
	Sequence           types.Number           `json:"sequence"`
	SiteTypes          []string               `json:"siteTypes"`
	Sort               uint64                 `json:"sort"`
	Symbol             string                 `json:"symbol"`
	SymbolCode         string                 `json:"symbolCode"`
	TakerCoefficient   float64                `json:"takerCoefficient"`
	TakerFeeRate       float64                `json:"takerFeeRate"`
	Trading            bool                   `json:"trading"`
	Volume             float64                `json:"vol"`
	VolumeValue        float64                `json:"volValue"`
}

// WsSnapshotMarketChange is a snapshot's statistics over a period
type WsSnapshotMarketChange struct {
	ChangePrice float64 `json:"changePrice"`
	ChangeRate  float64 `json:"changeRate"`
	High        float64 `json:"high"`
	Low         float64 `json:"low"`
	Open        float64 `json:"open"`
	Volume      float64 `json:"vol"`
	VolumeValue float64 `json:"volValue"`
}

// WsSpotOrderChange is an Order V2 or Order V1 push, for spot and margin orders alike
type WsSpotOrderChange struct {
	CanceledSize  types.Number `json:"canceledSize"`
	ClientOrderID string       `json:"clientOid"`
	FilledSize    types.Number `json:"filledSize"`
	OrderID       string       `json:"orderId"`
	OrderTime     types.Time   `json:"orderTime"`
	OrderType     string       `json:"orderType"`
	OriginSize    types.Number `json:"originSize"`
	Price         types.Number `json:"price"`
	RemainFunds   types.Number `json:"remainFunds"`
	RemainSize    types.Number `json:"remainSize"`
	Side          string       `json:"side"`
	Size          types.Number `json:"size"`
	Status        string       `json:"status"`
	Symbol        string       `json:"symbol"`
	Timestamp     types.Time   `json:"ts"`
	Type          string       `json:"type"`
	OldSize       types.Number `json:"oldSize"`
	FeeType       string       `json:"feeType"`
	Liquidity     string       `json:"liquidity"`
	MatchPrice    types.Number `json:"matchPrice"`
	MatchSize     types.Number `json:"matchSize"`
	TradeID       string       `json:"tradeId"`
}

// WsAccountBalance is a Balance push
type WsAccountBalance struct {
	AccountID       string                   `json:"accountId"`
	Currency        currency.Code            `json:"currency"`
	Total           types.Number             `json:"total"`
	Available       types.Number             `json:"available"`
	Hold            types.Number             `json:"hold"`
	AvailableChange types.Number             `json:"availableChange"`
	HoldChange      types.Number             `json:"holdChange"`
	RelationContext WsBalanceRelationContext `json:"relationContext"`
	RelationEvent   string                   `json:"relationEvent"`
	RelationEventID string                   `json:"relationEventId"`
	Time            types.Time               `json:"time"`
}

// WsBalanceRelationContext is the trade a balance change relates to
type WsBalanceRelationContext struct {
	Symbol  string `json:"symbol"`
	OrderID string `json:"orderId"`
	TradeID string `json:"tradeId"`
}

// WsStopOrder is a spot or margin Stop Order push
type WsStopOrder struct {
	OrderID    string       `json:"orderId"`
	OrderPrice types.Number `json:"orderPrice"`
	OrderType  string       `json:"orderType"`
	Side       string       `json:"side"`
	Size       types.Number `json:"size"`
	Stop       string       `json:"stop"`
	StopPrice  types.Number `json:"stopPrice"`
	Symbol     string       `json:"symbol"`
	TradeType  string       `json:"tradeType"`
	Type       string       `json:"type"`
	CreatedAt  types.Time   `json:"createdAt"`
	Timestamp  types.Time   `json:"ts"`
}
