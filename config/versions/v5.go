package versions

import (
	"context"
	"errors"
	"strconv"

	"github.com/buger/jsonparser"
	v5 "github.com/thrasher-corp/gocryptotrader/config/versions/v5"
)

// Version5 implements ConfigVersion
type Version5 struct{}

func init() {
	Manager.registerVersion(5, &Version5{})
}

// UpgradeConfig handles upgrading config for OrderManager:
// * Sets OrderManager config to defaults if it doesn't exist or enabled is null
// * Sets respectOrderHistoryLimits to true if it doesn't exist or is null
// * Sets futuresTrackingSeekDuration to positive if it's negative
func (v *Version5) UpgradeConfig(_ context.Context, e []byte) ([]byte, error) {
	_, valueType, _, err := jsonparser.Get(e, "orderManager", "enabled")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError), valueType == jsonparser.Null:
		return jsonparser.Set(e, v5.DefaultOrderbookConfig, "orderManager")
	case err != nil:
		return e, err
	}

	_, valueType, _, err = jsonparser.Get(e, "orderManager", "respectOrderHistoryLimits")
	if errors.Is(err, jsonparser.KeyPathNotFoundError) || valueType == jsonparser.Null {
		if e, err = jsonparser.Set(e, []byte(`true`), "orderManager", "respectOrderHistoryLimits"); err != nil {
			return e, err
		}
	}

	if i, err := jsonparser.GetInt(e, "orderManager", "futuresTrackingSeekDuration"); err != nil {
		if e, err = jsonparser.Set(e, []byte(v5.DefaultFuturesTrackingSeekDuration), "orderManager", "futuresTrackingSeekDuration"); err != nil {
			return e, err
		}
	} else if i < 0 {
		if e, err = jsonparser.Set(e, []byte(strconv.FormatInt(-i, 10)), "orderManager", "futuresTrackingSeekDuration"); err != nil {
			return e, err
		}
	}
	return e, nil
}

// DowngradeConfig just reverses the futuresTrackingSeekDuration to negative, and leaves everything else alone
func (v *Version5) DowngradeConfig(_ context.Context, e []byte) ([]byte, error) {
	if i, err := jsonparser.GetInt(e, "orderManager", "futuresTrackingSeekDuration"); err == nil && i > 0 {
		if e, err = jsonparser.Set(e, []byte(strconv.FormatInt(-i, 10)), "orderManager", "futuresTrackingSeekDuration"); err != nil {
			return e, err
		}
	}
	return e, nil
}
