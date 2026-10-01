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

// DowngradeConfig restores the legacy default expected by older releases, as the
// upgrade discarded any configured precision. An explicitly supplied setting is
// preserved.
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

	_, precisionType, _, err := jsonparser.Get(config, "remoteControl", "gRPC", "timeInNanoSeconds")
	switch {
	case err == nil && precisionType != jsonparser.Null:
		return config, nil
	case err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError):
		return config, fmt.Errorf("error getting gRPC timestamp precision: %w", err)
	}

	updated, err := jsonparser.Set(config, []byte("false"), "remoteControl", "gRPC", "timeInNanoSeconds")
	if err != nil {
		return config, fmt.Errorf("error restoring gRPC timestamp precision: %w", err)
	}
	return updated, nil
}
