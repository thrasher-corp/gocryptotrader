// Package v16 removes configuration for the decommissioned GCTScript feature.
package v16

import (
	"context"

	"github.com/buger/jsonparser"
)

var legacyGCTScriptConfig = []byte(`{"enabled":false,"timeout":30000000000,"max_virtual_machines":10,"allow_imports":false,"auto_load":null,"verbose":false}`)

// Version implements ConfigVersion to remove decommissioned GCTScript configuration.
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
