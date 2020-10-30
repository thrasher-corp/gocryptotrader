package event

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
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
