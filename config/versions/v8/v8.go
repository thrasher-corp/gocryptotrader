package v8

import (
	"context"
	"errors"

	"github.com/buger/jsonparser"
)

// Version is an ExchangeVersion to remove deprecated WS endpoints from user config
// Announcements:
// * https://blog.bitmex.com/api_announcement/change-of-websocket-endpoint/
// * https://blog.bitmex.com/api_announcement/api-update-remove-support-realtimemd/
type Version struct{}

// Exchanges returns just Bitmex
func (v *Version) Exchanges() []string { return []string{"Bitmex", "Poloniex"} }

// UpgradeExchange replaces deprecated WS and REST endpoints
func (v *Version) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	url, err := jsonparser.GetString(e, "api", "urlEndpoints", "WebsocketSpotURL")
	if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return e, err
	}

	switch url {
	case "wss://ws.bitmex.com/realtimemd", "wss://www.bitmex.com/realtimemd", "wss://www.bitmex.com/realtime":
		// Old defaults, just delete them
		return jsonparser.Delete(e, "api", "urlEndpoints", "WebsocketSpotURL"), nil
	case "wss://ws.testnet.bitmex.com/realtimemd", "wss://testnet.bitmex.com/realtimemd", "wss://testnet.bitmex.com/realtime":
		// User wants to use testnet
		return jsonparser.Set(e, []byte(`"wss://ws.testnet.bitmex.com/realtime"`), "api", "urlEndpoints", "WebsocketSpotURL")
	case "wss://api2.poloniex.com":
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
	switch restSpotURL {
	case "https://poloniex.com":
		e = jsonparser.Delete(e, "api", "urlEndpoints", "RestSpotURL")
		return jsonparser.Set(e, []byte(`"https://api.poloniex.com"`), "api", "urlEndpoints", "RestSpotURL")
	}
	return e, nil
}

// DowngradeExchange is a no-op for v8
func (v *Version) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	return e, nil
}
