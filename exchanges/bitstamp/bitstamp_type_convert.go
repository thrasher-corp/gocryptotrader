package bitstamp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// datetime provides an internal conversion helper
type datetime time.Time

func (d *datetime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	t, err := convert.UnixTimestampStrToTime(s)
	if err != nil {
		return err
	}

	*d = datetime(t)

	return nil
}

// Time returns datetime cast directly as time.Time
func (d datetime) Time() time.Time {
	return time.Time(d)
}

// microTimestamp provides an internal conversion helper
type microTimestamp time.Time

func (t *microTimestamp) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	if strconv.IntSize == 32 && len(s) >= 10 {
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		*t = microTimestamp(time.UnixMicro(i))
		return nil
	}

	// Has Fast path optimisation when int == 64
	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	*t = microTimestamp(time.UnixMicro(int64(i)))
	return nil
}

// Time returns datetime cast directly as time.Time
func (t microTimestamp) Time() time.Time {
	return time.Time(t)
}

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
	minOrderStr := t.MinimumOrder
	if prefix, _, found := strings.Cut(t.MinimumOrder, " "); found {
		minOrderStr = prefix
	}
	p.MinimumOrder, err = strconv.ParseFloat(minOrderStr, 64)
	return err
}

type orderSide order.Side

func (s *orderSide) UnmarshalJSON(data []byte) error {
	var i int64
	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}
	switch i {
	case 0:
		*s = orderSide(order.Buy)
	case 1:
		*s = orderSide(order.Sell)
	default:
		return fmt.Errorf("invalid value for order side: %v", i)
	}

	return nil
}

func (s *orderSide) Side() order.Side {
	return order.Side(*s)
}
