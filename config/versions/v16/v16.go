// Package v16 removes configuration for the decommissioned GCTScript feature.
package v16

import (
	"context"
	"encoding/json" //nolint:depguard // Config versions must retain stable standard-library JSON behaviour
	"errors"
	"strings"

	"github.com/buger/jsonparser"
)

var legacyGCTScriptConfig = []byte(`{"enabled":false,"timeout":30000000000,"max_virtual_machines":10,"allow_imports":false,"auto_load":null,"verbose":false}`)

// Version implements ConfigVersion to remove decommissioned GCTScript configuration.
type Version struct{}

// UpgradeConfig removes the GCTScript configuration and any obsolete GCTSCRIPT sublogger.
func (*Version) UpgradeConfig(_ context.Context, config []byte) ([]byte, error) {
	config = jsonparser.Delete(config, "gctscript")

	subloggersJSON, valueType, _, err := jsonparser.Get(config, "logging", "subloggers")
	if errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return config, nil
	}
	if err != nil {
		return config, err
	}
	if valueType != jsonparser.Array {
		return config, nil
	}

	var subloggers []json.RawMessage
	if err := json.Unmarshal(subloggersJSON, &subloggers); err != nil {
		return config, err
	}

	filtered := subloggers[:0]
	for i := range subloggers {
		var sublogger struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(subloggers[i], &sublogger); err != nil {
			return config, err
		}
		if !strings.EqualFold(sublogger.Name, "GCTSCRIPT") {
			filtered = append(filtered, subloggers[i])
		}
	}
	if len(filtered) == len(subloggers) {
		return config, nil
	}

	subloggersJSON, err = json.Marshal(filtered)
	if err != nil {
		return config, err
	}
	return jsonparser.Set(config, subloggersJSON, "logging", "subloggers")
}

// DowngradeConfig restores the legacy GCTScript defaults expected by older releases.
func (*Version) DowngradeConfig(_ context.Context, config []byte) ([]byte, error) {
	return jsonparser.Set(config, legacyGCTScriptConfig, "gctscript")
}
