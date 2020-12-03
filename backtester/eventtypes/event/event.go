package event

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func (e *Event) IsEvent() bool {
	return true
}

func (e *Event) GetTime() time.Time {
	return e.Time
}

func (e *Event) Pair() currency.Pair {
	return e.CurrencyPair
}

func (e *Event) GetExchange() string {
	return e.Exchange
}

func (e *Event) GetAssetType() asset.Item {
	return e.AssetType
}

func (e *Event) GetInterval() kline.Interval {
	return e.Interval
}

func (e *Event) AppendWhy(y string) {
	if e.Why == "" {
		e.Why = y
	} else {
		e.Why = y + ". " + e.Why
	}
}

func (e *Event) GetWhy() string {
	return e.Why
}
