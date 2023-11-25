package okx

import (
	"encoding/json"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
	a.InstrumentType = GetAssetTypeFromInstrumentType(chil.InstrumentType)
	return nil
}

// MarshalJSON serialized QuoteLeg instance into bytes
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

// MarshalJSON serialized CreateQuoteParams instance into bytes
func (a *CreateQuoteParams) MarshalJSON() ([]byte, error) {
	type Alias CreateQuoteParams
	chil := &struct {
		*Alias
		QuoteSide string `json:"quoteSide"`
	}{
		Alias: (*Alias)(a),
	}
	if a.QuoteSide == order.Buy {
		chil.QuoteSide = "buy"
	} else {
		chil.QuoteSide = "sell"
	}
	return json.Marshal(chil)
}

// MarshalJSON serializes the WebsocketLoginData object
func (a *WebsocketLoginData) MarshalJSON() ([]byte, error) {
	type Alias WebsocketLoginData
	return json.Marshal(struct {
		Timestamp int64 `json:"timestamp"`
		*Alias
	}{
		Timestamp: a.Timestamp.UTC().Unix(),
		Alias:     (*Alias)(a),
	})
}
