package v9998

import (
	"context"
	"errors"
)

// Version is test fixture
type Version struct {
	ConfigErr bool
	ExchErr   bool
}

// Public Errors
var (
	ErrUpgrade   = errors.New("do you expect me to talk?")
	ErrDowngrade = errors.New("no, I expect you to die")
)

// UpgradeConfig errors if v.ConfigErr is true
func (v *Version) UpgradeConfig(_ context.Context, c []byte) ([]byte, error) {
	if v.ConfigErr {
		return c, ErrUpgrade
	}
	return c, nil
}

// DowngradeConfig errors if v.ConfigErr is true
func (v *Version) DowngradeConfig(_ context.Context, c []byte) ([]byte, error) {
	if v.ConfigErr {
		return c, ErrDowngrade
	}
	return c, nil
}

// Exchanges returns just Juan
func (v *Version) Exchanges() []string {
	return []string{"Juan"}
}

// UpgradeExchange errors if ExchErr is true
func (v *Version) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if v.ExchErr {
		return e, ErrUpgrade
	}
	return e, nil
}

// DowngradeExchange errors if ExchErr is true
func (v *Version) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	if v.ExchErr {
		return e, ErrDowngrade
	}
	return e, nil
}
