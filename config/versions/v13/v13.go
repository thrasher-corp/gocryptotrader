package v13

import (
	"context"
	"encoding/json" //nolint:depguard // Config versions must retain stable standard-library JSON behaviour
	"errors"
	"strings"

	"github.com/buger/jsonparser"
)

const bitmex = "Bitmex"

// Version implements ConfigVersion to remove the decommissioned BitMEX exchange.
type Version struct{}

// UpgradeConfig removes all BitMEX configurations while preserving every other exchange.
func (*Version) UpgradeConfig(_ context.Context, config []byte) ([]byte, error) {
	exchangesJSON, valueType, _, err := jsonparser.Get(config, "exchanges")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError):
		return config, nil
	case err != nil:
		return config, err
	case valueType != jsonparser.Array:
		return config, nil
	}

	var exchanges []json.RawMessage
	if err := json.Unmarshal(exchangesJSON, &exchanges); err != nil {
		return config, err
	}

	filtered := exchanges[:0]
	for i := range exchanges {
		var exchange struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(exchanges[i], &exchange); err != nil {
			return config, err
		}
		if !strings.EqualFold(exchange.Name, bitmex) {
			filtered = append(filtered, exchanges[i])
		}
	}

	if len(filtered) == len(exchanges) {
		return config, nil
	}

	exchangesJSON = []byte{'['}
	for i := range filtered {
		if i != 0 {
			exchangesJSON = append(exchangesJSON, ',')
		}
		exchangesJSON = append(exchangesJSON, filtered[i]...)
	}
	exchangesJSON = append(exchangesJSON, ']')
	return jsonparser.Set(config, exchangesJSON, "exchanges")
}

// DowngradeConfig is a no-op because removed BitMEX configuration and credentials cannot be reconstructed.
func (*Version) DowngradeConfig(_ context.Context, config []byte) ([]byte, error) {
	return config, nil
}
