package htx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WsFundingRate contains a public derivative funding-rate update.
type WsFundingRate struct {
	Asset     asset.Item       `json:"-"`
	Pair      currency.Pair    `json:"-"`
	Channel   string           `json:"ch"`
	Timestamp types.Time       `json:"ts"`
	Tick      FundingRatesData `json:"tick"`
}
