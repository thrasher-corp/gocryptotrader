package v12

import (
	"context"
	"encoding/json" //nolint:depguard // Config versions must retain stable standard-library JSON behaviour
	"errors"
	"strings"

	"github.com/buger/jsonparser"
)

const exmo = "EXMO"

// Version implements ConfigVersion to remove the decommissioned EXMO exchange.
type Version struct{}

// UpgradeConfig removes all EXMO configurations while preserving every other exchange.
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
		if !strings.EqualFold(exchange.Name, exmo) {
			filtered = append(filtered, exchanges[i])
		}
	}

	if len(filtered) == len(exchanges) {
		return config, nil
	}

	exchangesJSON, err = json.Marshal(filtered)
	if err != nil {
		return config, err
	}
	return jsonparser.Set(config, exchangesJSON, "exchanges")
}

// DowngradeConfig is a no-op because removed EXMO configuration and credentials cannot be reconstructed.
func (*Version) DowngradeConfig(_ context.Context, config []byte) ([]byte, error) {
	return config, nil
}
