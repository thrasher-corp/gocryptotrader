package v11

import (
	"context"
	"errors"

	"github.com/buger/jsonparser"
)

// Version is an ExchangeVersion to replaces deprecated WS and REST endpoints from user config
type Version struct{}

// Exchanges returns just Poloniex
func (v *Version) Exchanges() []string { return []string{"Poloniex"} }

// UpgradeExchange replaces deprecated WS and REST endpoints
func (v *Version) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	url, err := jsonparser.GetString(e, "api", "urlEndpoints", "WebsocketSpotURL")
	if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return e, err
	}

	if url == "wss://api2.poloniex.com" {
		e = jsonparser.Delete(e, "api", "urlEndpoints", "WebsocketSpotURL")
		e, err = jsonparser.Set(e, []byte(`"wss://ws.poloniex.com/ws/public"`), "api", "urlEndpoints", "WebsocketSpotURL")
		if err != nil {
			return e, err
		}
	}
	restSpotURL, err := jsonparser.GetString(e, "api", "urlEndpoints", "RestSpotURL")
	if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return e, err
	}
	if restSpotURL == "https://poloniex.com" {
		e = jsonparser.Delete(e, "api", "urlEndpoints", "RestSpotURL")
		return jsonparser.Set(e, []byte(`"https://api.poloniex.com"`), "api", "urlEndpoints", "RestSpotURL")
	}
	return e, nil
}

// DowngradeExchange is a no-op for v11
func (v *Version) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	return e, nil
}
