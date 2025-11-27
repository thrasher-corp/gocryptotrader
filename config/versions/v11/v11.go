package v11

import (
	"context"
	"errors"

	"github.com/buger/jsonparser"
)

// Version is an ExchangeVersion to replace deprecated WS and REST endpoints for Poloniex
type Version struct{}

// Exchanges returns just Poloniex
func (v *Version) Exchanges() []string { return []string{"Poloniex"} }

// UpgradeExchange replaces deprecated WS and REST endpoints
func (v *Version) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	for _, detail := range []struct {
		key    string
		oldURL string
		newURL string
	}{
		{
			key:    "RestSpotURL",
			oldURL: "https://poloniex.com",
			newURL: "https://api.poloniex.com",
		},
		{
			key:    "WebsocketSpotURL",
			oldURL: "wss://api2.poloniex.com",
			newURL: "wss://ws.poloniex.com/ws/public",
		},
	} {
		url, err := jsonparser.GetString(e, "api", "urlEndpoints", detail.key)
		if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
			return e, err
		}
		if detail.oldURL == url {
			e = jsonparser.Delete(e, "api", "urlEndpoints", detail.key)
			return jsonparser.Set(e, []byte(`"`+detail.newURL+`"`), "api", "urlEndpoints", detail.key)
		}
	}
	return e, nil
}

// DowngradeExchange is a no-op for v11
func (v *Version) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	return e, nil
}
