// Package v16 removes obsolete websocket orderbook buffer settings.
package v16

import (
	"context"
	"errors"

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

// DowngradeExchange restores the v15 buffer defaults when the settings are absent.
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	_, valueType, _, err := jsonparser.Get(exchange, "orderbook")
	if errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return exchange, nil
	}
	if err != nil {
		return nil, err
	}
	if valueType != jsonparser.Object {
		return exchange, nil
	}

	for _, setting := range []struct {
		key   string
		value []byte
	}{
		{"websocketBufferLimit", []byte("5")},
		{"websocketBufferEnabled", []byte("false")},
	} {
		_, _, _, err = jsonparser.Get(exchange, "orderbook", setting.key)
		if err == nil {
			continue
		}
		if !errors.Is(err, jsonparser.KeyPathNotFoundError) {
			return nil, err
		}
		exchange, err = jsonparser.Set(exchange, setting.value, "orderbook", setting.key)
		if err != nil {
			return nil, err
		}
	}
	return exchange, nil
}
