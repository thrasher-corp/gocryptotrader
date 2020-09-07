package btse

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	btseRateInterval = time.Second
	btseQueryLimit   = 15
	btseOrdersLimit  = 75

	queryFunc request.EndpointLimit = iota
	orderFunc
)

type RateLimit struct {
	Query  *rate.Limiter
	Orders *rate.Limiter
}

func (r *RateLimit) Limit(f request.EndpointLimit) error {
	switch f {
	case orderFunc:
		time.Sleep(r.Orders.Reserve().Delay())
	default:
		time.Sleep(r.Query.Reserve().Delay())
	}
	return nil
}

func SetRateLimit() *RateLimit {
	return &RateLimit{
		Orders: request.NewRateLimit(btseRateInterval, btseOrdersLimit),
		Query:  request.NewRateLimit(btseRateInterval, btseQueryLimit),
	}
}
