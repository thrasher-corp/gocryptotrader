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
	bybitOrderBook = "/spot/quote/v1/depth"
	bybitMergedOrderBook = "/spot/quote/v1/depth/merged"
	bybitRecentTrades = "/spot/quote/v1/trades"
	bybitCandlestickChart = "/spot/quote/v1/kline"
	bybit24HrsChange = "/spot/quote/v1/ticker/24hr"
	bybitLastTradedPrice = "/spot/quote/v1/ticker/price"
	bybitBestBidAskPrice = "/spot/quote/v1/ticker/book_ticker"

	// Authenticated endpoints
	// TODO:
	bybitAuthenticatedSpotOrder = "/spot/v1/order" // create, query, cancel
	bybitBatchCancelSpotOrder = "/spot/order/batch-cancel"
	bybitOpenOrder = "/spot/v1/open-orders"
	bybitPastOrder = "/spot/v1/history-orders"
	bybitTradeHistory = "/spot/v1/myTrades"

	bybitWalletBalance = "/spot/v1/account"
	bybitServerTime = "/spot/v1/time"
)

// Start implementing public and private exchange API funcs below
