package versions

import (
	"context"
)

// Version0 is a baseline version with no changes, so we can downgrade back to nothing
// It does not implement any upgrade interfaces
type Version0 struct{}

func init() {
	Manager.registerVersion(0, &Version0{})
}

// UpgradeConfig is an empty stub
func (v *Version0) UpgradeConfig(_ context.Context, j []byte) ([]byte, error) {
	return j, nil
}

// DowngradeConfig is an empty stub
func (v *Version0) DowngradeConfig(_ context.Context, j []byte) ([]byte, error) {
	return j, nil
}
