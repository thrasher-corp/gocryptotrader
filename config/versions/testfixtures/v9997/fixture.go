package v9997

import (
	"context"
)

// Version is test fixture
type Version struct {
}

// Disabled implements the DisabledVersion interface
func (v *Version) Disabled() {}

// UpgradeConfig implements the ConfigdVersion interface
func (v *Version) UpgradeConfig(_ context.Context, c []byte) ([]byte, error) {
	return c, nil
}

// DowngradeConfig implements the ConfigdVersion interface
func (v *Version) DowngradeConfig(_ context.Context, c []byte) ([]byte, error) {
	return c, nil
}
