package bingx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CoinMarginedContractResponse holds a coin-M perpetual contract's information
type CoinMarginedContractResponse struct {
	Symbol         string `json:"symbol"`
	PricePrecision uint64 `json:"pricePrecision"`
	// MinTickSize holds the value of one contract in USD, such as 100 for BTC-USD, despite its name
	MinTickSize   types.Number `json:"minTickSize"`
	MinTradeValue types.Number `json:"minTradeValue"`
	// MinQuantity is the smallest order quantity, in contracts
	MinQuantity types.Number `json:"minQty"`
	Status      uint64       `json:"status"`
	TimeOnline  types.Time   `json:"timeOnline"`
	// DisplayName is the name BingX shows for the contract, which differs from Symbol for some contracts, such as
	// TRUMP-USD for TRUMPSOL-USD
	DisplayName string `json:"displayName"`
}

// CoinMarginedPriceAndFundingRateResponse holds a coin-M perpetual's mark and index prices and its current funding rate
type CoinMarginedPriceAndFundingRateResponse struct {
	Symbol          string       `json:"symbol"`
	MarkPrice       types.Number `json:"markPrice"`
	IndexPrice      types.Number `json:"indexPrice"`
	LastFundingRate types.Number `json:"lastFundingRate"`
	NextFundingTime types.Time   `json:"nextFundingTime"`
}

// CoinMarginedOpenInterestResponse holds a coin-M perpetual's open interest
type CoinMarginedOpenInterestResponse struct {
	Symbol       string       `json:"symbol"`
	OpenInterest types.Number `json:"openInterest"`
	Timestamp    types.Time   `json:"timestamp"`
}

// CoinMarginedKlineDataRequest holds the parameters of a coin-M perpetual candle query
type CoinMarginedKlineDataRequest struct {
	Symbol   currency.Pair
	Interval kline.Interval
	// StartTime includes the candle opening at it; EndTime excludes the candle opening at it
	StartTime time.Time
	EndTime   time.Time
	// TimeZone is the documented UTC offset in hours, 0 or 8; live coin-M candles follow UTC periods whichever is sent
	TimeZone uint64
	// Limit is at most 1000 and 500 by default
	Limit uint64
}

// CoinMarginedKlineResponse holds a coin-M perpetual candle
type CoinMarginedKlineResponse struct {
	Open  types.Number `json:"open"`
	Close types.Number `json:"close"`
	High  types.Number `json:"high"`
	Low   types.Number `json:"low"`
	// Volume counts contracts
	Volume types.Number `json:"volume"`
	// Time is the candle's open time
	Time types.Time `json:"time"`
}

// CoinMarginedOrderBookResponse holds a coin-M perpetual order book, its quantities counting contracts. BingX sends the
// asks highest price first
type CoinMarginedOrderBookResponse struct {
	Time types.Time                       `json:"T"`
	Bids orderbook.LevelsArrayPriceAmount `json:"bids"`
	Asks orderbook.LevelsArrayPriceAmount `json:"asks"`
}

// CoinMargined24HourPriceChangeResponse holds a coin-M perpetual's price change and trading over the last 24 hours. Its
// quantities and Volume count contracts
type CoinMargined24HourPriceChangeResponse struct {
	Symbol      string       `json:"symbol"`
	PriceChange types.Number `json:"priceChange"`
	// PriceChangePercent arrives as text with a percent sign, such as -1.0300%
	PriceChangePercent Percentage   `json:"priceChangePercent"`
	LastPrice          types.Number `json:"lastPrice"`
	LastQuantity       types.Number `json:"lastQty"`
	HighPrice          types.Number `json:"highPrice"`
	LowPrice           types.Number `json:"lowPrice"`
	Volume             types.Number `json:"volume"`
	// QuoteVolume is the volume in the base coin, such as BTC for BTC-USD, despite its name
	QuoteVolume types.Number `json:"quoteVolume"`
	OpenPrice   types.Number `json:"openPrice"`
	CloseTime   types.Time   `json:"closeTime"`
	BidPrice    types.Number `json:"bidPrice"`
	BidQuantity types.Number `json:"bidQty"`
	AskPrice    types.Number `json:"askPrice"`
	AskQuantity types.Number `json:"askQty"`
}
