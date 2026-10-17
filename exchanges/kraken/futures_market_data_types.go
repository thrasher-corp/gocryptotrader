package kraken

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FuturesTickersRequest holds the parameters of Get tickers
type FuturesTickersRequest struct {
	// ContractTypes limits the tickers to futures_inverse, futures_vanilla, flexible_futures, options or all. Every
	// futures type, which excludes options, is returned when it is empty
	ContractTypes []string
	// Pairs limits the tickers to these markets, such as PF_XBTUSD
	Pairs currency.Pairs
}

// FuturesTickersResponse holds the tickers of the listed markets, in no particular order
type FuturesTickersResponse struct {
	Tickers    []FuturesTicker `json:"tickers"`
	ServerTime time.Time       `json:"serverTime"`
}

// FuturesTickerResponse holds a market's ticker
type FuturesTickerResponse struct {
	Ticker     FuturesTicker `json:"ticker"`
	ServerTime time.Time     `json:"serverTime"`
}

// FuturesTicker holds a market's ticker, or an index's, which holds only its symbol and last value and time. Values
// over the last 24 hours are of the fills observed in that window
type FuturesTicker struct {
	Symbol   string    `json:"symbol"`
	Last     float64   `json:"last"`
	LastTime time.Time `json:"lastTime"`
	LastSize float64   `json:"lastSize"`
	// Tag groups the market by expiry, such as perpetual, week, month, quarter or semiannual
	Tag string `json:"tag"`
	// Pair is the market's base and quote, such as XBT:USD
	Pair string `json:"pair"`
	// MarkPrice is the price positions are margined at
	MarkPrice    float64 `json:"markPrice"`
	Bid          float64 `json:"bid"`
	BidSize      float64 `json:"bidSize"`
	Ask          float64 `json:"ask"`
	AskSize      float64 `json:"askSize"`
	Volume24Hour float64 `json:"vol24h"`
	// QuoteVolume24Hour is the sum of each fill's size times its price
	QuoteVolume24Hour float64 `json:"volumeQuote"`
	// VolumeWeightedAveragePrice24Hour is sent, though Kraken does not document it
	VolumeWeightedAveragePrice24Hour float64 `json:"vwap24h"`
	OpenInterest                     float64 `json:"openInterest"`
	Open24Hour                       float64 `json:"open24h"`
	High24Hour                       float64 `json:"high24h"`
	Low24Hour                        float64 `json:"low24h"`
	// ExtrinsicValue is an option's mark price less what exercising it now would be worth
	ExtrinsicValue float64 `json:"extrinsicValue"`
	// MarkImpliedVolatility is an option's implied volatility at its mark price, which Kraken sends without documenting
	MarkImpliedVolatility float64 `json:"markIv"`
	// FundingRate and FundingRatePrediction are a perpetual's current and estimated next absolute funding rates; the
	// relative rates express them as a fraction of the price
	FundingRate                   float64 `json:"fundingRate"`
	FundingRatePrediction         float64 `json:"fundingRatePrediction"`
	RelativeFundingRate           float64 `json:"relativeFundingRate"`
	RelativeFundingRatePrediction float64 `json:"relativeFundingRatePrediction"`
	Suspended                     bool    `json:"suspended"`
	IndexPrice                    float64 `json:"indexPrice"`
	PostOnly                      bool    `json:"postOnly"`
	// Change24Hour is the price change in percent
	Change24Hour float64             `json:"change24h"`
	Greeks       FuturesOptionGreeks `json:"greeks"`
	// IsUnderlyingMarketClosed is sent for a traditional finance market only
	IsUnderlyingMarketClosed bool `json:"isUnderlyingMarketClosed"`
}

// FuturesOptionGreeks holds an option's greeks. ImpliedVolatility is -1 when it cannot be calculated, and a greek
// Kraken sends as null decodes as 0
type FuturesOptionGreeks struct {
	ImpliedVolatility float64 `json:"iv"`
	Delta             float64 `json:"delta"`
	Gamma             float64 `json:"gamma"`
	Vega              float64 `json:"vega"`
	Theta             float64 `json:"theta"`
	Rho               float64 `json:"rho"`
}

// FuturesOrderbookResponse holds a market's order book
type FuturesOrderbookResponse struct {
	OrderBook  FuturesOrderbook `json:"orderBook"`
	ServerTime time.Time        `json:"serverTime"`
}

// FuturesOrderbook holds every non-cumulative price level of a market. Kraken documents the bids in descending price
// order, but sends both sides in ascending price order, so the best bid is the last
type FuturesOrderbook struct {
	Asks []FuturesOrderbookLevel `json:"asks"`
	Bids []FuturesOrderbookLevel `json:"bids"`
}

// FuturesOrderbookLevel is an order book price level
type FuturesOrderbookLevel struct {
	Price float64
	Size  float64
}

// UnmarshalJSON decodes a [price, size] array
func (f *FuturesOrderbookLevel) UnmarshalJSON(data []byte) error {
	fields := [2]any{&f.Price, &f.Size}
	return unmarshalFixedArray(data, fields[:])
}

// FuturesTradeHistoryRequest holds the parameters of Get trade history
type FuturesTradeHistoryRequest struct {
	Pair currency.Pair
	// LastTime returns the 100 trades before it, for paging back with the oldest trade's Time; the 100 most recent
	// trades are returned when it is zero
	LastTime time.Time
	// IncludeMTFData adds the transparency fields of an MTF market's trades. Kraken documents them as included by
	// default, but omits them unless they are requested
	IncludeMTFData bool
}

// FuturesTradeHistoryResponse holds a market's trades from the last 7 days, or since the trading engine last
// restarted if that is sooner. Kraken documents them in descending time order, but sends them in ascending order
type FuturesTradeHistoryResponse struct {
	History    []FuturesTrade `json:"history"`
	ServerTime time.Time      `json:"serverTime"`
}

// FuturesTrade is a market's trade, or an index's computation, which holds only its price and time
type FuturesTrade struct {
	// Price is a trade's price or an index's computed value
	Price float64 `json:"price"`
	// Side is the taker's side, buy or sell
	Side string `json:"side"`
	// Size is documented as a string but sent as a number, so either is accepted
	Size types.Number `json:"size"`
	Time time.Time    `json:"time"`
	// TradeID numbers the response's trades from 1, the most recent, rather than numbering the market's fills as
	// Kraken documents, so UID or SequenceID identifies a trade
	TradeID uint64 `json:"trade_id"`
	// Type is fill, liquidation, partial liquidation, assignment, unwind or rfq
	Type string `json:"type"`
	UID  string `json:"uid"`
	// SequenceID numbers the market's trades from 1 without gaps, and is 0 for a trade made before it was introduced
	SequenceID uint64 `json:"sequence_id,string"`
	// The remaining fields are the transparency fields of an MTF market's trade
	InstrumentIdentificationType  string        `json:"instrument_identification_type"`
	ISIN                          string        `json:"isin"`
	ExecutionVenue                string        `json:"execution_venue"`
	PriceNotation                 string        `json:"price_notation"`
	PriceCurrency                 currency.Code `json:"price_currency"`
	NotionalAmount                float64       `json:"notional_amount"`
	NotionalCurrency              currency.Code `json:"notional_currency"`
	PublicationTime               time.Time     `json:"publication_time"`
	PublicationVenue              string        `json:"publication_venue"`
	TransactionIdentificationCode string        `json:"transaction_identification_code"`
	ToBeCleared                   bool          `json:"to_be_cleared"`
}
