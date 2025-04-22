package v0

import (
	"context"
)

// Version is a baseline version with no changes, so we can downgrade back to nothing
// It does not implement any upgrade interfaces
type Version struct{}

// UpgradeConfig is an empty stub
func (*Version) UpgradeConfig(_ context.Context, j []byte) ([]byte, error) {
	return j, nil
}

// DowngradeConfig is an empty stub
func (*Version) DowngradeConfig(_ context.Context, j []byte) ([]byte, error) {
	return j, nil
}
