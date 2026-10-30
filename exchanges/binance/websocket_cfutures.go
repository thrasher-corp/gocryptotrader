package binance

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// COIN-M serves market data and user data streams from one unrouted host
const (
	binanceCoinMarginedFuturesWebsocketURL = "wss://dstream.binance.com/stream"

	coinmFilter        = "coin-m"
	coinmPrivateFilter = "coin-m-private"
)

// generateCoinMarginedFuturesSubscriptions returns the COIN-M stream subscriptions
func (e *Exchange) generateCoinMarginedFuturesSubscriptions() (subscription.List, error) {
	return e.generateDerivativesSubscriptions(asset.CoinMarginedFutures, func(*subscription.Subscription) bool { return true })
}

// generateCoinMarginedFuturesUserDataSubscriptions returns the COIN-M user data stream subscription when it can be used
func (e *Exchange) generateCoinMarginedFuturesUserDataSubscriptions() (subscription.List, error) {
	return e.generateUserDataSubscriptions(asset.CoinMarginedFutures)
}

// wsHandleCoinMarginedFuturesData routes COIN-M market data stream frames
func (e *Exchange) wsHandleCoinMarginedFuturesData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	return e.handleFuturesStream(ctx, conn, respRaw, asset.CoinMarginedFutures)
}

// wsHandleCoinMarginedFuturesUserData routes COIN-M user data stream frames
func (e *Exchange) wsHandleCoinMarginedFuturesUserData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	return e.handleFuturesUserData(ctx, conn, respRaw, asset.CoinMarginedFutures)
}
