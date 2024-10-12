package v9997

import (
	"context"
)

// Version is test fixture
type Version struct {
}

func (v *Version) Disabled() {}

func (v *Version) UpgradeConfig(_ context.Context, c []byte) ([]byte, error) {
	return c, nil
}

func (v *Version) DowngradeConfig(_ context.Context, c []byte) ([]byte, error) {
	return c, nil
}
