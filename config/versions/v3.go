package versions

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/buger/jsonparser"
)

// Version3 is an Exchange upgrade to move currencyPairs.assetTypes to currencyPairs.pairs.*.assetEnabled
type Version3 struct {
}

func init() {
	Manager.registerVersion(3, &Version3{})
}

// Exchanges returns all exchanges: "*"
func (v *Version3) Exchanges() []string { return []string{"*"} }

// UpgradeExchange sets AssetEnabled: true for all assets listed in assetTypes, and false for any with no field
func (v *Version3) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	toEnable := map[string]bool{}

	assetTypesFn := func(v []byte, vT jsonparser.ValueType, _ int, _ error) {
		if vT == jsonparser.String {
			toEnable[string(v)] = true
		}
	}
	_, err := jsonparser.ArrayEach(e, assetTypesFn, "currencyPairs", "assetTypes")
	if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return e, err
	}

	assetEnabledFn := func(kBytes []byte, v []byte, _ jsonparser.ValueType, _ int) error {
		k := string(kBytes)
		if toEnable[k] {
			e, err = jsonparser.Set(e, []byte(`true`), "currencyPairs", "pairs", k, "assetEnabled")
			return err
		}
		_, err = jsonparser.GetBoolean(v, "assetEnabled")
		if errors.Is(err, jsonparser.KeyPathNotFoundError) {
			e, err = jsonparser.Set(e, []byte(`false`), "currencyPairs", "pairs", k, "assetEnabled")
		}
		return err
	}
	err = jsonparser.ObjectEach(bytes.Clone(e), assetEnabledFn, "currencyPairs", "pairs")
	if err == nil {
		e = jsonparser.Delete(e, "currencyPairs", "assetTypes")
	}
	return e, err
}

// DowngradeExchange moves AssetEnabled assets into AssetType field
func (v *Version3) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	assetTypes := []string{}

	assetEnabledFn := func(k []byte, v []byte, _ jsonparser.ValueType, _ int) error {
		if b, err := jsonparser.GetBoolean(v, "assetEnabled"); err == nil {
			if b {
				assetTypes = append(assetTypes, fmt.Sprintf("%q", k))
			}
			e = jsonparser.Delete(e, "currencyPairs", "pairs", string(k), "assetEnabled")
		}
		return nil
	}
	err := jsonparser.ObjectEach(bytes.Clone(e), assetEnabledFn, "currencyPairs", "pairs")
	if err == nil {
		e, err = jsonparser.Set(e, []byte(`[`+strings.Join(assetTypes, ",")+`]`), "currencyPairs", "assetTypes")
	}
	return e, err
}
