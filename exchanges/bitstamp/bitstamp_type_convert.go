package bitstamp

import (
	"encoding/json"
	"regexp"
	"strconv"
)

var currRE = regexp.MustCompile(`^[\d.]+`)

// UnmarshalJSON deserializes JSON, and timestamp information.
func (p *TradingPair) UnmarshalJSON(data []byte) error {
	type Alias TradingPair
	t := &struct {
		*Alias
		MinimumOrder string `json:"minimum_order"`
	}{
		Alias: (*Alias)(p),
	}

	err := json.Unmarshal(data, t)
	if err == nil {
		if m := currRE.FindString(t.MinimumOrder); len(m) > 0 {
			p.MinimumOrder, err = strconv.ParseFloat(m, 64)
		}
	}

	return err
}
