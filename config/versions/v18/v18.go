// Package v18 migrates the legacy OKX websocket defaults to TLS port 443.
package v18

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/buger/jsonparser"
)

// Version implements ExchangeVersion for the retirement of OKX port 8443.
type Version struct{}

// Exchanges limits the endpoint migration to OKX.
func (*Version) Exchanges() []string { return []string{"Okx"} }

// UpgradeExchange replaces only the previous defaults, preserving custom endpoints.
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	for _, endpoint := range []struct{ key, channel string }{
		{"WebsocketSpotURL", "public"},
		{"WebsocketPrivateURL", "private"},
		{"WebsocketSpotSupplementaryURL", "business"},
	} {
		value, kind, _, err := jsonparser.Get(exchange, "api", "urlEndpoints", endpoint.key)
		if errors.Is(err, jsonparser.KeyPathNotFoundError) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("getting OKX endpoint %s: %w", endpoint.key, err)
		}
		if kind != jsonparser.String {
			continue
		}
		decoded, err := jsonparser.ParseString(value)
		if err != nil {
			return nil, fmt.Errorf("decoding OKX endpoint %s: %w", endpoint.key, err)
		}
		if decoded != "wss://ws.okx.com:8443/ws/v5/"+endpoint.channel {
			continue
		}
		exchange, err = jsonparser.Set(exchange, []byte(strconv.Quote("wss://ws.okx.com/ws/v5/"+endpoint.channel)), "api", "urlEndpoints", endpoint.key)
		if err != nil {
			return nil, fmt.Errorf("setting OKX endpoint %s: %w", endpoint.key, err)
		}
	}
	return exchange, nil
}

// DowngradeExchange retains port 443, which is also supported by older clients.
// Restoring the retired port would make an otherwise compatible configuration fail.
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return exchange, nil
}
