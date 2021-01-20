package event

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// IsEvent returns whether the event is an event
func (e *Event) IsEvent() bool {
	return true
}

// GetTime returns the time
func (e *Event) GetTime() time.Time {
	return e.Time
}

// Pair returns the currency pair
func (e *Event) Pair() currency.Pair {
	return e.CurrencyPair
}

// GetExchange returns the exchange
func (e *Event) GetExchange() string {
	return e.Exchange
}

// GetAssetType returns the asset type
func (e *Event) GetAssetType() asset.Item {
	return e.AssetType
}

// GetInterval returns the interval
func (e *Event) GetInterval() kline.Interval {
	return e.Interval
}

// AppendWhy adds reasoning for a decision being made
func (e *Event) AppendWhy(y string) {
	if e.Why == "" {
		e.Why = y
	} else {
		e.Why = y + ". " + e.Why
	}
}

// GetWhy returns the why
func (e *Event) GetWhy() string {
	return e.Why
}
