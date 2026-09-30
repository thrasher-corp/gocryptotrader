// Package v17 removes the obsolete configurable gRPC timestamp precision.
package v17

import (
	"context"
	"errors"
	"fmt"

	"github.com/buger/jsonparser"
)

// Version implements ConfigVersion for the gRPC timestamp migration.
type Version struct{}

// UpgradeConfig removes the timestamp precision setting because protobuf
// timestamps now preserve nanoseconds without configuration.
func (*Version) UpgradeConfig(_ context.Context, config []byte) ([]byte, error) {
	return jsonparser.Delete(config, "remoteControl", "gRPC", "timeInNanoSeconds"), nil
}

// DowngradeConfig enables nanosecond precision when a gRPC configuration
// exists so the standardised timestamp behaviour survives a downgrade.
func (*Version) DowngradeConfig(_ context.Context, config []byte) ([]byte, error) {
	_, valueType, _, err := jsonparser.Get(config, "remoteControl", "gRPC")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError):
		return config, nil
	case err != nil:
		return config, fmt.Errorf("error getting gRPC configuration: %w", err)
	case valueType != jsonparser.Object:
		return config, nil
	}

	updated, err := jsonparser.Set(config, []byte("true"), "remoteControl", "gRPC", "timeInNanoSeconds")
	if err != nil {
		return config, fmt.Errorf("error restoring gRPC timestamp precision: %w", err)
	}
	return updated, nil
}
