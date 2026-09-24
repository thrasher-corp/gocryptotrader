// Package v16 corrects Gemini's legacy public websocket endpoint override.
package v16

import (
	"context"
	"errors"

	"github.com/buger/jsonparser"
)

// Version migrates the bare Gemini host which cannot serve market-data upgrades.
type Version struct{}

// Exchanges limits the migration to Gemini configurations.
func (*Version) Exchanges() []string { return []string{"Gemini"} }

// UpgradeExchange preserves custom endpoints and updates only the old default.
func (*Version) UpgradeExchange(_ context.Context, data []byte) ([]byte, error) {
	url, err := jsonparser.GetString(data, "api", "urlEndpoints", "WebsocketSpotURL")
	if errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return data, nil
	}
	if err != nil {
		return data, err
	}
	if url != "wss://api.gemini.com" {
		return data, nil
	}
	return jsonparser.Set(data, []byte(`"wss://api.gemini.com/v2/marketdata"`), "api", "urlEndpoints", "WebsocketSpotURL")
}

// DowngradeExchange retains the working URL because the bare host is unusable.
func (*Version) DowngradeExchange(_ context.Context, data []byte) ([]byte, error) {
	return data, nil
}
