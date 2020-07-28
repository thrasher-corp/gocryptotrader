package event

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

type Directions uint8

const (
	BUY Directions = iota
	SELL
	HOLD
	EXIT
)

type Handler interface {
	Time
	Pair
}

type Time interface {
	Time() time.Time
}

type Pair interface {
	Pair() currency.Pair
}

type Event struct {
	Timestamp time.Time
	Pair      currency.Pair
}

type Direction interface {
	Direction() Directions
	SetDirection(Directions)
}
