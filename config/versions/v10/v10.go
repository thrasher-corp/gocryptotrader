package v10

import (
	"context"

	"github.com/buger/jsonparser"
)

// Version implements ConfigVersion
type Version struct{}

// UpgradeConfig checks and removes the deprecatedRPC and websocketRPC fields from the remoteControl config
func (*Version) UpgradeConfig(_ context.Context, e []byte) ([]byte, error) {
	e = jsonparser.Delete(e, "remoteControl", "deprecatedRPC")
	e = jsonparser.Delete(e, "remoteControl", "websocketRPC")
	return e, nil
}

// DowngradeConfig is a no-op
func (*Version) DowngradeConfig(_ context.Context, e []byte) ([]byte, error) {
	// Note: We do NOT restore deprecatedRPC or websocketRPC on downgrade.
	// Their removal is permanent as those subsystems have been eliminated
	return e, nil
}
