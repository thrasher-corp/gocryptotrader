package bitstamp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UnmarshalJSON deserialises JSON and parses the minimum order value
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
	minOrderStr := t.MinimumOrder
	if prefix, _, found := strings.Cut(t.MinimumOrder, " "); found {
		minOrderStr = prefix
	}
	p.MinimumOrder, err = strconv.ParseFloat(minOrderStr, 64)
	return err
}

type orderSide order.Side

func (s *orderSide) UnmarshalJSON(data []byte) error {
	// The REST ticker quotes the side whereas the websocket order feed sends a bare number
	var n types.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	switch n.Int64() {
	case 0:
		*s = orderSide(order.Buy)
	case 1:
		*s = orderSide(order.Sell)
	default:
		return fmt.Errorf("invalid value for order side: %v", n)
	}

	return nil
}

func (s *orderSide) Side() order.Side {
	return order.Side(*s)
}
