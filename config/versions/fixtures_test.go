package versions

import (
	"context"
	"errors"
)

// TestVersion1 is an empty and incompatible Version for testing
type TestVersion1 struct{}

// TestVersion2 is test fixture
type TestVersion2 struct {
	ConfigErr bool
	ExchErr   bool
}

var (
	errUpgrade   = errors.New("do you expect me to talk?")
	errDowngrade = errors.New("no, I expect you to die")
)

// UpgradeConfig errors if v.ConfigErr is true
func (v *TestVersion2) UpgradeConfig(_ context.Context, c []byte) ([]byte, error) {
	if v.ConfigErr {
		return c, errUpgrade
	}
	return c, nil
}

// DowngradeConfig errors if v.ConfigErr is true
func (v *TestVersion2) DowngradeConfig(_ context.Context, c []byte) ([]byte, error) {
	if v.ConfigErr {
		return c, errDowngrade
	}
	return c, nil
}

// Exchanges returns just Juan
func (v *TestVersion2) Exchanges() []string {
	return []string{"Juan"}
}

// UpgradeExchange errors if ExchErr is true
func (v *TestVersion2) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if v.ExchErr {
		return e, errUpgrade
	}
	return e, nil
}

// DowngradeExchange errors if ExchErr is true
func (v *TestVersion2) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if v.ExchErr {
		return e, errDowngrade
	}
	return e, nil
}
