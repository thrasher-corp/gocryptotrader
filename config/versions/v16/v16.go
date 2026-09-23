// Package v16 removes obsolete websocket orderbook buffer settings.
package v16

import (
	"context"

	"github.com/buger/jsonparser"
)

// Version implements ExchangeVersion for the removal of orderbook buffering.
type Version struct{}

// Exchanges applies this migration to every exchange.
func (*Version) Exchanges() []string { return []string{"*"} }

// UpgradeExchange removes buffer settings that are no longer used by the orderbook manager.
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	exchange = jsonparser.Delete(exchange, "orderbook", "websocketBufferLimit")
	exchange = jsonparser.Delete(exchange, "orderbook", "websocketBufferEnabled")
	return exchange, nil
}

// DowngradeExchange cannot recover the removed buffer settings, so it leaves them unset.
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return exchange, nil
}
