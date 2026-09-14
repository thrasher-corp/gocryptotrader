package htx

import (
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

type wsFuturesPong struct {
	Operation string          `json:"op"`
	Timestamp json.RawMessage `json:"ts"`
}

// WsFundingRate contains a public derivative funding-rate update.
type WsFundingRate struct {
	Asset     asset.Item       `json:"-"`
	Pair      currency.Pair    `json:"-"`
	Channel   string           `json:"ch"`
	Timestamp types.Time       `json:"ts"`
	Tick      FundingRatesData `json:"tick"`
}
