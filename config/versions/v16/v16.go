// Package v16 removes obsolete websocket orderbook buffer settings and GCTScript configuration.
package v16

import (
	"context"
	"errors"

	"github.com/buger/jsonparser"
)

var legacyGCTScriptConfig = []byte(`{"enabled":false,"timeout":30000000000,"max_virtual_machines":10,"allow_imports":false,"auto_load":null,"verbose":false}`)

// Version implements ConfigVersion and ExchangeVersion for the v16 migrations.
type Version struct{}

// UpgradeConfig removes the GCTScript configuration. Obsolete sublogger entries
// are retained because rewriting duplicate subloggers keys can alter their merged decoding.
func (*Version) UpgradeConfig(_ context.Context, config []byte) ([]byte, error) {
	return jsonparser.Delete(config, "gctscript"), nil
}

// DowngradeConfig restores the legacy GCTScript defaults expected by older releases.
func (*Version) DowngradeConfig(_ context.Context, config []byte) ([]byte, error) {
	return jsonparser.Set(config, legacyGCTScriptConfig, "gctscript")
}

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
