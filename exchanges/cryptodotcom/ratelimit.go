package cryptodotcom

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Endpoint rate limits. Crypto.com limits each method separately, public methods per IP and private methods per API
// key; the methods sharing a limiter here are held to one method's limit together, which is stricter than documented
// but keeps the limiters few
const (
	publicEPL request.EndpointLimit = iota
	stakingEPL
	createOrderEPL
	cancelOrderEPL
	cancelAllOrdersEPL
	getOrderDetailEPL
	getTradesEPL
	getOrderHistoryEPL
	privateEPL
)

// Websocket rate limits, per connection
const (
	websocketMarketDataRate = 100
	websocketUserRate       = 150
)

// rateLimits returns the documented limits of each endpoint limit: public methods 100 requests a second, staking
// methods 50, create-order, cancel-order and cancel-all-orders 15 every 100 milliseconds, get-order-detail 30 every
// 100 milliseconds, get-trades and get-order-history one a second, and every other private method 3 every 100
// milliseconds
func rateLimits() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		publicEPL:          request.NewRateLimitWithWeight(time.Second, 100, 1),
		stakingEPL:         request.NewRateLimitWithWeight(time.Second, 50, 1),
		createOrderEPL:     request.NewRateLimitWithWeight(100*time.Millisecond, 15, 1),
		cancelOrderEPL:     request.NewRateLimitWithWeight(100*time.Millisecond, 15, 1),
		cancelAllOrdersEPL: request.NewRateLimitWithWeight(100*time.Millisecond, 15, 1),
		getOrderDetailEPL:  request.NewRateLimitWithWeight(100*time.Millisecond, 30, 1),
		getTradesEPL:       request.NewRateLimitWithWeight(time.Second, 1, 1),
		getOrderHistoryEPL: request.NewRateLimitWithWeight(time.Second, 1, 1),
		privateEPL:         request.NewRateLimitWithWeight(100*time.Millisecond, 3, 1),
	}
}
