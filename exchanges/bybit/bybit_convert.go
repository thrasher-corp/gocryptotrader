package bybit

import "github.com/thrasher-corp/gocryptotrader/encoding/json"

// UnmarshalJSON deserializes incoming data into orderbookResponse instance.
func (a *orderbookResponse) UnmarshalJSON(data []byte) error {
	type Alias orderbookResponse
	child := &struct {
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	err := json.Unmarshal(data, child)
	if err != nil {
		var resp []any
		err = json.Unmarshal(data, &resp)
		if err != nil {
			return err
		}
	}
	return nil
}
