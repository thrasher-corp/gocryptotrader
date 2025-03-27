package versions

import (
	"context"
	"errors"

	"github.com/buger/jsonparser"
	v6 "github.com/thrasher-corp/gocryptotrader/config/versions/v6"
)

// Version6 implements ConfigVersion
type Version6 struct{}

func init() {
	Manager.registerVersion(6, &Version6{})
}

// UpgradeConfig checks and upgrades the portfolioAddresses.providers field
func (v *Version6) UpgradeConfig(_ context.Context, e []byte) ([]byte, error) {
	_, valueType, _, err := jsonparser.Get(e, "portfolioAddresses", "providers")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError), valueType == jsonparser.Null:
		return jsonparser.Set(e, v6.DefaultConfig, "portfolioAddresses", "providers")
	case err != nil:
		return e, err
	}
	return e, nil
}

// DowngradeConfig removes the portfolioAddresses.providers field
func (v *Version6) DowngradeConfig(_ context.Context, e []byte) ([]byte, error) {
	e = jsonparser.Delete(e, "portfolioAddresses", "providers")
	return e, nil
}
