package htx

import (
	"bytes"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// LegacyOrderPriceType accepts numeric V3 order-price codes and legacy names.
type LegacyOrderPriceType string

// UnmarshalJSON normalises HTX's numeric history codes to the names used by order classification.
func (p *LegacyOrderPriceType) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) != 0 && data[0] == '"' {
		var name string
		if err := json.Unmarshal(data, &name); err != nil {
			return fmt.Errorf("%w: %w", errInvalidOrderPriceType, err)
		}
		*p = LegacyOrderPriceType(name)
		return nil
	}
	var code uint64
	if err := json.Unmarshal(data, &code); err != nil {
		return fmt.Errorf("%w: %w", errInvalidOrderPriceType, err)
	}
	names := [...]string{
		"", "limit", "market", "opponent", "lightning", "trigger", "post_only",
		orderPriceTypeOptimal5, orderPriceTypeOptimal10, orderPriceTypeOptimal20, "fok", "ioc", "opponent_ioc",
		orderPriceTypeLightningIOC, "optimal_5_ioc", "optimal_10_ioc", orderPriceTypeOptimal20IOC,
		"opponent_fok", "lightning_fok", "optimal_5_fok",
	}
	switch {
	case code > 0 && code < uint64(len(names)):
		*p = LegacyOrderPriceType(names[code])
	case code == 40:
		*p = "optimal_10_fok"
	case code == 41:
		*p = orderPriceTypeOptimal20FOK
	default:
		return fmt.Errorf("%w: numeric code %d", errInvalidOrderPriceType, code)
	}
	return nil
}
