package dydx

import (
	"encoding/json"
	"time"
)

// UnmarshalJSON deserialises the JSON info, including the timestamp
func (a *APIServerTime) UnmarshalJSON(data []byte) error {
	type Alias APIServerTime
	chil := &struct {
		*Alias
		Epoch float64 `json:"epoch"`
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, chil); err != nil {
		return err
	}
	a.Epoch = time.Unix(int64(chil.Epoch), 0)
	return nil
}
