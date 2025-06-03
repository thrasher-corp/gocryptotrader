package v6

import (
	"context"
	"errors"

	"github.com/buger/jsonparser"
)

// Version implements ConfigVersion
type Version struct{}

// UpgradeConfig checks and upgrades the portfolioAddresses.providers field
func (*Version) UpgradeConfig(_ context.Context, e []byte) ([]byte, error) {
	_, valueType, _, err := jsonparser.Get(e, "portfolioAddresses", "providers")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError), valueType == jsonparser.Null:
		return jsonparser.Set(e, DefaultConfig, "portfolioAddresses", "providers")
	case err != nil:
		return e, err
	}
	return e, nil
}

// DowngradeConfig removes the portfolioAddresses.providers field
func (*Version) DowngradeConfig(_ context.Context, e []byte) ([]byte, error) {
	e = jsonparser.Delete(e, "portfolioAddresses", "providers")
	return e, nil
}
