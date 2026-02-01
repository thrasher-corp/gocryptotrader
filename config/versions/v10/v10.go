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

// DowngradeConfig is a no-op. It does not restore deprecatedRPC or websocketRPC on downgrade as their removal is permanent
func (*Version) DowngradeConfig(_ context.Context, e []byte) ([]byte, error) {
	return e, nil
}
