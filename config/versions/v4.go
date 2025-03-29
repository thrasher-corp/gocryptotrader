package versions

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/buger/jsonparser"
)

// Version4 is an Exchange upgrade to move currencyPairs.assetTypes to currencyPairs.pairs.*.assetEnabled
type Version4 struct{}

func init() {
	Manager.registerVersion(4, &Version4{})
}

// Exchanges returns all exchanges: "*"
func (v *Version4) Exchanges() []string { return []string{"*"} }

// UpgradeExchange sets AssetEnabled: true for all assets listed in assetTypes, and false for any with no field
func (v *Version4) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	toEnable := map[string]bool{}

	assetTypesFn := func(asset []byte, valueType jsonparser.ValueType, _ int, _ error) {
		if valueType == jsonparser.String {
			toEnable[string(asset)] = true
		}
	}
	_, err := jsonparser.ArrayEach(e, assetTypesFn, "currencyPairs", "assetTypes")
	if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return e, fmt.Errorf("%w assetTypes: %w", errUpgrading, err)
	}

	assetEnabledFn := func(assetBytes, v []byte, _ jsonparser.ValueType, _ int) (err error) {
		asset := string(assetBytes)
		if toEnable[asset] {
			e, err = jsonparser.Set(e, []byte(`true`), "currencyPairs", "pairs", asset, "assetEnabled")
		} else {
			var vT jsonparser.ValueType
			_, vT, _, err = jsonparser.Get(v, "assetEnabled")
			switch {
			case vT == jsonparser.Null, errors.Is(err, jsonparser.KeyPathNotFoundError):
				e, err = jsonparser.Set(e, []byte(`false`), "currencyPairs", "pairs", asset, "assetEnabled")
			case err == nil && vT != jsonparser.Boolean:
				err = fmt.Errorf("assetEnabled: %w (`%s`)", jsonparser.UnknownValueTypeError, vT)
			}
		}
		if err != nil {
			err = fmt.Errorf("%w for asset `%s`", err, asset)
		}
		return err
	}
	if err = jsonparser.ObjectEach(bytes.Clone(e), assetEnabledFn, "currencyPairs", "pairs"); err != nil {
		return e, fmt.Errorf("%w currencyPairs.pairs: %w", errUpgrading, err)
	}
	e = jsonparser.Delete(e, "currencyPairs", "assetTypes")
	return e, err
}

// DowngradeExchange moves AssetEnabled assets into AssetType field
func (v *Version4) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	assetTypes := []string{}

	assetEnabledFn := func(asset, v []byte, _ jsonparser.ValueType, _ int) error {
		if b, err := jsonparser.GetBoolean(v, "assetEnabled"); err == nil {
			if b {
				assetTypes = append(assetTypes, fmt.Sprintf("%q", asset))
			}
			e = jsonparser.Delete(e, "currencyPairs", "pairs", string(asset), "assetEnabled")
		}
		return nil
	}
	if err := jsonparser.ObjectEach(bytes.Clone(e), assetEnabledFn, "currencyPairs", "pairs"); err != nil {
		return e, err
	}
	return jsonparser.Set(e, []byte(`[`+strings.Join(assetTypes, ",")+`]`), "currencyPairs", "assetTypes")
}
