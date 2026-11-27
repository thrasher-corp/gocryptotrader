package okx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// TickersRequest holds the parameters for GetTickers
type TickersRequest struct {
	InstrumentType   string
	InstrumentFamily string
}

// TickerResponse represents the detailed data from the market ticker endpoint
type TickerResponse struct {
	InstrumentType                string        `json:"instType"`
	InstrumentID                  currency.Pair `json:"instId"`
	LastTradePrice                types.Number  `json:"last"`
	LastTradeSize                 types.Number  `json:"lastSz"`
	BestAskPrice                  types.Number  `json:"askPx"`
	BestAskSize                   types.Number  `json:"askSz"`
	BestBidPrice                  types.Number  `json:"bidPx"`
	BestBidSize                   types.Number  `json:"bidSz"`
	OpenPrice24Hour               types.Number  `json:"open24h"`
	HighestPrice24Hour            types.Number  `json:"high24h"`
	LowestPrice24Hour             types.Number  `json:"low24h"`
	TradingVolume24HourInCurrency types.Number  `json:"volCcy24h"` // number of base currency on a derivative, quantity in quote currency on spot and margin
	TradingVolume24HourInContract types.Number  `json:"vol24h"`    // number of contracts on a derivative, quantity in base currency on spot and margin
	OpenPriceUTC0                 types.Number  `json:"sodUtc0"`
	OpenPriceUTC8                 types.Number  `json:"sodUtc8"`
	TickerDataGenerationTime      types.Time    `json:"ts"`
}

// OrderBookRequest holds the parameters for the order book endpoints
type OrderBookRequest struct {
	InstrumentID string
	Depth        uint64 // levels per side
}

// OrderBookResponse holds an order book snapshot
type OrderBookResponse struct {
	Asks       []OrderBookLevel `json:"asks"`
	Bids       []OrderBookLevel `json:"bids"`
	Timestamp  types.Time       `json:"ts"`
	SequenceID uint64           `json:"seqId"`
}

// OrderBookLevel holds one price level of an order book
type OrderBookLevel struct {
	Price             types.Number
	Size              types.Number
	LiquidationOrders types.Number // deprecated by OKX and always "0"
	OrderCount        types.Number
}

// UnmarshalJSON decodes an order book level from its array form
func (o *OrderBookLevel) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[4]any{&o.Price, &o.Size, &o.LiquidationOrders, &o.OrderCount})
}

// RPIOrderBookResponse holds an order book snapshot combining organic and tradeable retail price improvement depth
type RPIOrderBookResponse struct {
	Asks       []RPIOrderBookLevel `json:"asks"`
	Bids       []RPIOrderBookLevel `json:"bids"`
	Timestamp  types.Time          `json:"ts"`
	SequenceID uint64              `json:"seqId"`
}

// RPIOrderBookLevel holds one price level of a retail price improvement order book
type RPIOrderBookLevel struct {
	Price      types.Number
	TotalSize  types.Number
	NonRPISize types.Number
	OrderCount types.Number
}

// UnmarshalJSON decodes a retail price improvement order book level from its array form
func (r *RPIOrderBookLevel) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[4]any{&r.Price, &r.TotalSize, &r.NonRPISize, &r.OrderCount})
}

// FullOrderBookResponse holds a full depth order book snapshot
type FullOrderBookResponse struct {
	Asks       []FullOrderBookLevel `json:"asks"`
	Bids       []FullOrderBookLevel `json:"bids"`
	Timestamp  types.Time           `json:"ts"`
	SequenceID uint64               `json:"seqId"`
}

// FullOrderBookLevel holds one price level of a full depth order book
type FullOrderBookLevel struct {
	Price      types.Number
	Size       types.Number
	OrderCount types.Number
}

// UnmarshalJSON decodes a full depth order book level from its array form
func (f *FullOrderBookLevel) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[3]any{&f.Price, &f.Size, &f.OrderCount})
}

// CandlesticksRequest holds the parameters for the candlestick endpoints
type CandlesticksRequest struct {
	InstrumentID string
	Bar          kline.Interval
	After        time.Time // returns records earlier than this time
	Before       time.Time // returns records newer than this time
	Limit        uint64
	Adjust       string // price adjustment for equity perpetual contracts, e.g. forward
}

// CandlestickResponse holds one candlestick
type CandlestickResponse struct {
	OpenTime            types.Time
	OpenPrice           types.Number
	HighestPrice        types.Number
	LowestPrice         types.Number
	ClosePrice          types.Number
	Volume              types.Number // number of contracts on a derivative, quantity in base currency on spot and margin
	VolumeCurrency      types.Number // number of base currency on a derivative, quantity in quote currency on spot and margin
	VolumeQuoteCurrency types.Number
	Confirmed           bool
}

// UnmarshalJSON decodes a candlestick from its array form
func (c *CandlestickResponse) UnmarshalJSON(data []byte) error {
	var confirm string
	if err := json.Unmarshal(data, &[9]any{&c.OpenTime, &c.OpenPrice, &c.HighestPrice, &c.LowestPrice, &c.ClosePrice, &c.Volume, &c.VolumeCurrency, &c.VolumeQuoteCurrency, &confirm}); err != nil {
		return err
	}
	c.Confirmed = confirm == "1"
	return nil
}

// TradesRequest holds the parameters for GetTrades
type TradesRequest struct {
	InstrumentID string
	Limit        uint64
}

// TradesHistoryRequest holds the parameters for GetTradesHistory
type TradesHistoryRequest struct {
	InstrumentID   string
	PaginationType string // 1: trade ID, the default; 2: timestamp
	After          string // trade ID or Unix millisecond timestamp, as PaginationType selects
	Before         string // trade ID
	Limit          uint64
}

// TradeResponse represents a public trade
type TradeResponse struct {
	InstrumentID string       `json:"instId"`
	TradeID      string       `json:"tradeId"`
	Price        types.Number `json:"px"`
	Size         types.Number `json:"sz"`
	Side         order.Side   `json:"side"`
	Source       string       `json:"source"`
	SequenceID   uint64       `json:"seqId"`
	Count        string       `json:"count"`
	Timestamp    types.Time   `json:"ts"`
}

// OptionFamilyTradesResponse holds an option family's recent trades of one option type
type OptionFamilyTradesResponse struct {
	Volume24Hour types.Number        `json:"vol24h"` // in contracts
	OptionType   string              `json:"optType"`
	TradeInfo    []OptionFamilyTrade `json:"tradeInfo"`
}

// OptionFamilyTrade holds one trade of an option family
type OptionFamilyTrade struct {
	InstrumentID string       `json:"instId"`
	TradeID      string       `json:"tradeId"`
	Price        types.Number `json:"px"`
	Size         types.Number `json:"sz"` // in contracts
	Side         order.Side   `json:"side"`
	Timestamp    types.Time   `json:"ts"`
}

// OptionTradesRequest holds the parameters for GetOptionTrades
type OptionTradesRequest struct {
	InstrumentID     string
	InstrumentFamily string
	OptionType       string // C: call, P: put
}

// OptionTradeResponse holds one option trade
type OptionTradeResponse struct {
	InstrumentID     string       `json:"instId"`
	InstrumentFamily string       `json:"instFamily"`
	TradeID          string       `json:"tradeId"`
	Price            types.Number `json:"px"`
	Size             types.Number `json:"sz"` // in contracts
	Side             order.Side   `json:"side"`
	OptionType       string       `json:"optType"`
	FillVolatility   types.Number `json:"fillVol"`
	ForwardPrice     types.Number `json:"fwdPx"`
	IndexPrice       types.Number `json:"idxPx"`
	MarkPrice        types.Number `json:"markPx"`
	Timestamp        types.Time   `json:"ts"`
}

// TotalVolume24HourResponse holds the platform's rolling 24 hour order book trading volume
type TotalVolume24HourResponse struct {
	VolumeUSD types.Number `json:"volUsd"`
	VolumeCNY types.Number `json:"volCny"`
	Timestamp types.Time   `json:"ts"`
}

// CallAuctionDetailsResponse holds an instrument's call auction details
type CallAuctionDetailsResponse struct {
	InstrumentID     string       `json:"instId"`
	EquilibriumPrice types.Number `json:"eqPx"`
	MatchedSize      types.Number `json:"matchedSz"`
	UnmatchedSize    types.Number `json:"unmatchedSz"`
	AuctionEndTime   types.Time   `json:"auctionEndTime"`
	State            string       `json:"state"`
	Timestamp        types.Time   `json:"ts"`
}
