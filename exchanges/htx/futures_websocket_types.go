package htx

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/types"
)

type wsFuturesAuthRequest struct {
	Operation        string `json:"op"`
	AuthType         string `json:"type"`
	AccessKeyID      string `json:"AccessKeyId"`
	SignatureMethod  string `json:"SignatureMethod"`
	SignatureVersion string `json:"SignatureVersion"`
	Timestamp        string `json:"Timestamp"`
	Signature        string `json:"Signature"`
}

type wsFuturesSubscriptionRequest struct {
	Operation string `json:"op"`
	Topic     string `json:"topic"`
}

type wsV5FuturesSubscriptionRequest struct {
	Operation    string `json:"op"`
	Topic        string `json:"topic"`
	ContractCode string `json:"contract_code,omitempty"`
}

// wsFuturesTimestamp validates the heartbeat while preserving its JSON representation for the reply.
type wsFuturesTimestamp string

// UnmarshalJSON accepts HTX's numeric and quoted Unix timestamps.
func (t *wsFuturesTimestamp) UnmarshalJSON(data []byte) error {
	var parsed types.Time
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	if parsed.Time().IsZero() {
		return fmt.Errorf("%w: missing futures websocket timestamp", common.ErrParsingWSField)
	}
	*t = wsFuturesTimestamp(data)
	return nil
}

// MarshalJSON echoes the validated timestamp without converting its JSON type.
func (t wsFuturesTimestamp) MarshalJSON() ([]byte, error) {
	return []byte(t), nil
}

type wsFuturesPong struct {
	Operation string             `json:"op"`
	Timestamp wsFuturesTimestamp `json:"ts"`
}

// WsFundingRate contains a public derivative funding-rate update.
type WsFundingRate struct {
	Asset     asset.Item       `json:"-"`
	Pair      currency.Pair    `json:"-"`
	Channel   string           `json:"ch"`
	Timestamp types.Time       `json:"ts"`
	Tick      FundingRatesData `json:"tick"`
}
