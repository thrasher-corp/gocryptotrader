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
	for _, key := range []string{"WebsocketSpotURL", "RestSpotURL"} {
		url, err := jsonparser.GetString(e, "api", "urlEndpoints", key)
		if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
			return e, err
		}
		switch url {
		case "wss://api2.poloniex.com", "https://poloniex.com":
			e = jsonparser.Delete(e, "api", "urlEndpoints", key)
		}
	}
	return e, nil
}

// DowngradeExchange is a no-op for v11
func (v *Version) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	return e, nil
}
