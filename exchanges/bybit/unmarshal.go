package bybit

import (
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// UnmarshalJSON implements the json.Unmarshaler interface for KlineItem
func (k *KlineItem) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&k.StartTime, &k.Open, &k.High, &k.Low, &k.Close, &k.TradeVolume, &k.Turnover})
}
