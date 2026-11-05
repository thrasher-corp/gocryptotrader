package lbank

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// MarketDepthResponse holds a pair's order book
type MarketDepthResponse struct {
	Asks      orderbook.LevelsArrayPriceAmount `json:"asks"`
	Bids      orderbook.LevelsArrayPriceAmount `json:"bids"`
	Timestamp types.Time                       `json:"timestamp"`
}

// LatestPriceResponse holds a pair's latest price
type LatestPriceResponse struct {
	Symbol string       `json:"symbol"`
	Price  types.Number `json:"price"`
}

// BookTickerResponse holds a pair's best bid and ask
type BookTickerResponse struct {
	Symbol      string       `json:"symbol"`
	BidPrice    types.Number `json:"bidPrice"`
	BidQuantity types.Number `json:"bidQty"`
	AskPrice    types.Number `json:"askPrice"`
	AskQuantity types.Number `json:"askQty"`
}

// TickerResponse holds a pair's 24 hour statistics
type TickerResponse struct {
	Symbol string           `json:"symbol"`
	Ticker TickerStatistics `json:"ticker"`
	// Timestamp is the time of the pair's latest trade
	Timestamp types.Time `json:"timestamp"`
}

// TickerStatistics holds a pair's latest price and its statistics over the last 24 hours
type TickerStatistics struct {
	LastPrice              types.Number `json:"latest"`
	High24Hour             types.Number `json:"high"`
	Low24Hour              types.Number `json:"low"`
	Volume24Hour           types.Number `json:"vol"`
	Turnover24Hour         types.Number `json:"turnover"`
	ChangePercentage24Hour types.Number `json:"change"`
}

// TradeResponse holds a public trade
type TradeResponse struct {
	ID            string     `json:"id"`
	Price         float64    `json:"price"`
	Quantity      float64    `json:"qty"`
	QuoteQuantity float64    `json:"quoteQty"`
	Time          types.Time `json:"time"`
	// IsBuyerMaker is true when the buyer took liquidity, despite its name: live trades at the ask carry true, and
	// LBank's websocket and older trades endpoint report those same trades as buys
	IsBuyerMaker bool `json:"isBuyerMaker"`
}

// KlineRequest holds a candle request's parameters
type KlineRequest struct {
	Pair     currency.Pair
	Interval kline.Interval
	Start    time.Time
	Size     uint64
}

// KlineResponse holds a candle
type KlineResponse struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// UnmarshalJSON decodes a candle from LBank's [time, open, high, low, close, volume] list, its time in Unix seconds
func (k *KlineResponse) UnmarshalJSON(data []byte) error {
	var fields []types.Number
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if len(fields) < 6 {
		return fmt.Errorf("%w: %d fields, expected 6", errMalformedKline, len(fields))
	}
	*k = KlineResponse{
		Time:   time.Unix(fields[0].Int64(), 0),
		Open:   fields[1].Float64(),
		High:   fields[2].Float64(),
		Low:    fields[3].Float64(),
		Close:  fields[4].Float64(),
		Volume: fields[5].Float64(),
	}
	return nil
}
