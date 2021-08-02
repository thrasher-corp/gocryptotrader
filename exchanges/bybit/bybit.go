package bybit

import (
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Bybit is the overarching type across this package
type Bybit struct {
	exchange.Base
}

const (
	bybitAPIURL     = "https://api.bybit.com"
	bybitAPIVersion = "v1"

	// Public endpoints
	// TODO:
	bybitSpotGetSymbols = "/spot/v1/symbols"
	bybitGetOrderBook = "/spot/quote/v1/depth"
	bybitGetMergedOrderBook = "/spot/quote/v1/depth/merged"
	bybitGetRecentTrades = "/spot/quote/v1/trades"
	bybitGetCandlestickChart = "/spot/quote/v1/kline"

	bybitGet24HrsChange = "/spot/quote/v1/ticker/24hr"
	bybitGetLastTradedPrice = "/spot/quote/v1/ticker/price"
	bybitGetBestBidAskPrice = "/spot/quote/v1/ticker/book_ticker"
	// Authenticated endpoints
)

// Start implementing public and private exchange API funcs below
