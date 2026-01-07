package v11

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
)

// Version is an ExchangeVersion to set any Binance subscriptions without an asset to spot
type Version struct{}

// Exchanges returns just Binance
func (v *Version) Exchanges() []string { return []string{"Binance"} }

// UpgradeExchange sets any subscriptions without an asset to spot
func (v *Version) UpgradeExchange(_ context.Context, e []byte) ([]byte, error) {
	var errs error
	newSubs := [][]byte{}
	subsFn := func(sub []byte, valueType jsonparser.ValueType, _ int, _ error) {
		if valueType == jsonparser.Object {
			assetType, err := jsonparser.GetString(sub, "asset")
			if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
				errs = common.AppendError(errs, err)
				return
			}
			if assetType == "" {
				if sub, err = jsonparser.Set(sub, []byte(`"spot"`), "asset"); err != nil {
					errs = common.AppendError(errs, err)
					return
				}
			}
		}
		newSubs = append(newSubs, sub)
	}
	_, err := jsonparser.ArrayEach(e, subsFn, "features", "subscriptions")
	if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return e, fmt.Errorf("error upgrading subscription assets: %w", err)
	}
	for i, s := range newSubs {
		e, err = jsonparser.Set(e, s, "features", "subscriptions", "["+strconv.Itoa(i)+"]")
		errs = common.AppendError(errs, err)
	}
	return e, errs
}

// DowngradeExchange is a no-op for v11
func (v *Version) DowngradeExchange(_ context.Context, e []byte) ([]byte, error) {
	return e, nil
}
