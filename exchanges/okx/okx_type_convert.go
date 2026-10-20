package okx

import (
	"bytes"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UnmarshalJSON decoder for OpenInterestResponse instance.
func (a *OpenInterest) UnmarshalJSON(data []byte) error {
	type Alias OpenInterest
	chil := &struct {
		*Alias
		InstrumentType string `json:"instType"`
	}{Alias: (*Alias)(a)}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	a.InstrumentType, err = assetTypeFromInstrumentType(chil.InstrumentType)
	return err
}

// MarshalJSON serialised QuoteLeg instance into bytes
func (a *QuoteLeg) MarshalJSON() ([]byte, error) {
	type Alias QuoteLeg
	chil := &struct {
		*Alias
		Side string `json:"side"`
	}{
		Alias: (*Alias)(a),
	}
	if a.Side == order.Buy {
		chil.Side = "buy"
	} else {
		chil.Side = "sell"
	}
	return json.Marshal(chil)
}

// UnmarshalJSON decodes OKX's numeric strings while retaining explicit zero values.
func (n *OptionalNumber) UnmarshalJSON(data []byte) error {
	var value types.Number
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	data = bytes.TrimSpace(data)
	n.Value = nil
	if !bytes.Equal(data, []byte(`null`)) && !bytes.Equal(data, []byte(`""`)) {
		n.Value = &value
	}
	return nil
}
