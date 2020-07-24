package event

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

type Handler struct {
	Time time.Time
	Symbol currency.Pair
}