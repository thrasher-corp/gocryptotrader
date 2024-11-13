package versions

import (
	"bytes"
	"context"

	"github.com/buger/jsonparser"
)

// Version3 implements ExchangeVersion
type Version3 struct {
}

func init() {
	Manager.registerVersion(3, &Version3{})
}

// Exchanges returns all exchanges: "*"
func (v *Version3) Exchanges() []string { return []string{"*"} }

// UpgradeExchange sets AssetEnabed: true for any exchange missing it
func (v *Version3) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	cb := func(k []byte, v []byte, _ jsonparser.ValueType, _ int) error {
		if _, err := jsonparser.GetBoolean(v, "assetEnabled"); err != nil {
			e, err = jsonparser.Set(e, []byte(`true`), "currencyPairs", "pairs", string(k), "assetEnabled")
			return err
		}
		return nil
	}
	err := jsonparser.ObjectEach(bytes.Clone(e), cb, "currencyPairs", "pairs")
	return e, err
}

// DowngradeExchange doesn't do anything for this version, because it's a lossy downgrade to disable all assets
func (v *Version3) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	return e, nil
}
