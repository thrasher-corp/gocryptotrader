package bingx

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// spotKlineFields is the number of elements in a spot candle row
const spotKlineFields = 8

// SpotSymbolsResponse holds the spot trading symbols
type SpotSymbolsResponse struct {
	Symbols []SpotSymbolResponse `json:"symbols"`
}

// SpotSymbolResponse holds a spot trading symbol and its trading rules
type SpotSymbolResponse struct {
	Symbol   string  `json:"symbol"`
	TickSize float64 `json:"tickSize"`
	StepSize float64 `json:"stepSize"`
	// MinQuantity and MaxQuantity are deprecated: BingX derives them from the notional limits and the price
	MinQuantity       float64          `json:"minQty"`
	MaxQuantity       float64          `json:"maxQty"`
	MinNotional       float64          `json:"minNotional"`
	MaxNotional       float64          `json:"maxNotional"`
	MaxMarketNotional float64          `json:"maxMarketNotional"` // Some symbols, mostly offline ones, report -1
	Status            SpotSymbolStatus `json:"status"`
	APIStateBuy       bool             `json:"apiStateBuy"`
	APIStateSell      bool             `json:"apiStateSell"`
	TimeOnline        types.Time       `json:"timeOnline"`
	OffTime           types.Time       `json:"offTime"`
	MaintainTime      types.Time       `json:"maintainTime"` // When trading is suspended
	DisplayName       string           `json:"displayName"`
}

// SpotRecentTradeResponse holds a recent spot trade
type SpotRecentTradeResponse struct {
	ID       uint64     `json:"id"`
	Price    float64    `json:"price"`
	Quantity float64    `json:"qty"`
	Time     types.Time `json:"time"`
	// BuyerMaker is true when the taker bought: live, it is the opposite of the trade stream's m, which reports whether
	// the buyer was the maker
	BuyerMaker bool `json:"buyerMaker"`
}

// SpotOrderBookResponse holds a spot order book
type SpotOrderBookResponse struct {
	Bids         orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks         orderbook.LevelsArrayPriceAmount `json:"asks"`
	Timestamp    types.Time                       `json:"ts"`
	LastUpdateID uint64                           `json:"lastUpdateId"`
}

// SpotKlinesRequest holds the parameters of a spot candle request
type SpotKlinesRequest struct {
	Symbol    currency.Pair
	Interval  kline.Interval
	StartTime time.Time
	EndTime   time.Time
	TimeZone  SpotKlineTimeZone
	Limit     uint64
}

// SpotKlineResponse holds a spot candle, which BingX sends as an array of [openTime, open, high, low, close, volume,
// closeTime, quoteVolume]
type SpotKlineResponse struct {
	OpenTime    types.Time
	Open        float64
	High        float64
	Low         float64
	Close       float64
	Volume      float64
	CloseTime   types.Time
	QuoteVolume float64
}

// UnmarshalJSON decodes a candle from its array, which must hold at least its eight documented elements
func (k *SpotKlineResponse) UnmarshalJSON(data []byte) error {
	var row []json.RawMessage
	if err := json.Unmarshal(data, &row); err != nil {
		return err
	}
	if len(row) < spotKlineFields {
		return fmt.Errorf("%w: %d elements, want %d", errSpotKlineRowLength, len(row), spotKlineFields)
	}
	for i, field := range []any{&k.OpenTime, &k.Open, &k.High, &k.Low, &k.Close, &k.Volume, &k.CloseTime, &k.QuoteVolume} {
		if err := json.Unmarshal(row[i], field); err != nil {
			return fmt.Errorf("error decoding kline element %d: %w", i, err)
		}
	}
	return nil
}

// SpotTickerResponse holds a spot symbol's price change statistics over the last 24 hours
type SpotTickerResponse struct {
	Symbol             string     `json:"symbol"`
	OpenPrice          float64    `json:"openPrice"`
	HighPrice          float64    `json:"highPrice"`
	LowPrice           float64    `json:"lowPrice"`
	LastPrice          float64    `json:"lastPrice"`
	PriceChange        float64    `json:"priceChange"`
	Volume             float64    `json:"volume"`
	QuoteVolume        float64    `json:"quoteVolume"`
	OpenTime           types.Time `json:"openTime"`
	CloseTime          types.Time `json:"closeTime"`
	BidPrice           float64    `json:"bidPrice"`
	BidQuantity        float64    `json:"bidQty"`
	AskPrice           float64    `json:"askPrice"`
	AskQuantity        float64    `json:"askQty"`
	PriceChangePercent Percentage `json:"priceChangePercent"`
}

// SpotAggregatedOrderBookRequest holds the parameters of a spot aggregated order book request
type SpotAggregatedOrderBookRequest struct {
	Symbol currency.Pair
	Limit  uint64
	Type   OrderBookAggregation
}

// SpotAggregatedOrderBookResponse holds a spot order book with its levels merged to a coarser price precision
type SpotAggregatedOrderBookResponse struct {
	Bids      orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks      orderbook.LevelsArrayPriceAmount `json:"asks"`
	Timestamp types.Time                       `json:"ts"`
}

// SpotPriceTickerResponse holds a spot symbol's latest price
type SpotPriceTickerResponse struct {
	Price     types.Number `json:"price"`
	Symbol    string       `json:"symbol"`
	Timestamp types.Time   `json:"timestamp"` // Zero for every symbol when every symbol is requested
}

// SpotBookTickerResponse holds a spot symbol's best bid and ask
type SpotBookTickerResponse struct {
	Time        types.Time   `json:"time"`
	Symbol      string       `json:"symbol"`
	BidPrice    types.Number `json:"bidPrice"`
	BidQuantity types.Number `json:"bidQty"`
	AskPrice    types.Number `json:"askPrice"`
	AskQuantity types.Number `json:"askQty"`
}

// SpotHistoricalKlinesRequest holds the parameters of a spot historical candle request
type SpotHistoricalKlinesRequest struct {
	Symbol    currency.Pair
	Interval  kline.Interval
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// SpotHistoricalTradesRequest holds the parameters of a spot historical trade request
type SpotHistoricalTradesRequest struct {
	Symbol currency.Pair
	Limit  uint64
	// FromID is the trade ID of the oldest trade a previous reply held, to return the trades before it
	FromID string
}

// SpotHistoricalTradeResponse holds a historical spot trade
type SpotHistoricalTradeResponse struct {
	TradeID   string        `json:"tid"` // Too large for uint64
	Time      types.Time    `json:"t"`
	TradeSide SpotTradeSide `json:"ms"`
	Symbol    string        `json:"s"`
	Price     float64       `json:"p"`
	Quantity  float64       `json:"v"`
}
