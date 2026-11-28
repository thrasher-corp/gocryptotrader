package binanceus

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// GetMostRecentTrades returns the recent trades of a symbol as Exchange.GetMostRecentTrades does, over the WebSocket
// API
func (w WebsocketAPI) GetMostRecentTrades(ctx context.Context, req *RecentTradeRequest) ([]RecentTradeResponse, error) {
	return w.e.getMostRecentTrades(ctx, req, true)
}

// GetHistoricalTrades looks up older trades of a symbol as Exchange.GetHistoricalTrades does, over the WebSocket API
func (w WebsocketAPI) GetHistoricalTrades(ctx context.Context, symbol currency.Pair, limit, fromID uint64) ([]RecentTradeResponse, error) {
	return w.e.getHistoricalTrades(ctx, symbol, limit, fromID, true)
}

// GetAggregatedTrades returns aggregate trades as Exchange.GetAggregatedTrades does, over the WebSocket API
func (w WebsocketAPI) GetAggregatedTrades(ctx context.Context, req *AggregatedTradeRequest) ([]AggregatedTradeResponse, error) {
	return w.e.getAggregatedTrades(ctx, req, true)
}

// GetOrderBook returns the order book of a symbol as Exchange.GetOrderBook does, over the WebSocket API
func (w WebsocketAPI) GetOrderBook(ctx context.Context, req *OrderBookRequest) (*OrderBookResponse, error) {
	return w.e.getOrderBook(ctx, req, true)
}

// GetSpotKline returns the klines of a symbol as Exchange.GetSpotKline does, over the WebSocket API
func (w WebsocketAPI) GetSpotKline(ctx context.Context, req *KlinesRequest) ([]CandleStickResponse, error) {
	return w.e.getSpotKline(ctx, req, true)
}

// GetLatestSpotPrice returns the latest price of symbols as Exchange.GetLatestSpotPrice does, over the WebSocket API
func (w WebsocketAPI) GetLatestSpotPrice(ctx context.Context, symbols currency.Pairs, symbolStatus SymbolStatus) ([]SymbolPriceResponse, error) {
	return w.e.getLatestSpotPrice(ctx, symbols, symbolStatus, true)
}

// GetAveragePrice returns the current average price of a symbol as Exchange.GetAveragePrice does, over the WebSocket
// API
func (w WebsocketAPI) GetAveragePrice(ctx context.Context, symbol currency.Pair) (*AveragePriceResponse, error) {
	return w.e.getAveragePrice(ctx, symbol, true)
}

// GetBestPrice returns the best bid and ask of symbols as Exchange.GetBestPrice does, over the WebSocket API
func (w WebsocketAPI) GetBestPrice(ctx context.Context, symbols currency.Pairs, symbolStatus SymbolStatus) ([]BookTickerResponse, error) {
	return w.e.getBestPrice(ctx, symbols, symbolStatus, true)
}

// GetPriceChangeStatistics returns 24hr ticker price change statistics as Exchange.GetPriceChangeStatistics does, over
// the WebSocket API
func (w WebsocketAPI) GetPriceChangeStatistics(ctx context.Context, req *PriceChangeStatisticsRequest) ([]PriceChangeStatisticsResponse, error) {
	return w.e.getPriceChangeStatistics(ctx, req, true)
}

// GetTickerData returns rolling window price change statistics as Exchange.GetTickerData does, over the WebSocket API
func (w WebsocketAPI) GetTickerData(ctx context.Context, req *RollingWindowTickerRequest) ([]TickerStatisticsResponse, error) {
	return w.e.getTickerData(ctx, req, true)
}
