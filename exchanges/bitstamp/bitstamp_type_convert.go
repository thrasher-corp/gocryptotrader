package bitstamp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

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
	if err != nil {
		return err
	}
	before, _, found := strings.Cut(t.MinimumOrder, " ")
	if !found {
		return fmt.Errorf("unhandled minimum order string: %s", t.MinimumOrder)
	}
	p.MinimumOrder, err = strconv.ParseFloat(before, 64)
	return err
}
