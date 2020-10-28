package direction

import "github.com/thrasher-corp/gocryptotrader/exchanges/order"

// Directioner dictates the side of an order
type Directioner interface {
	SetDirection(side order.Side)
	GetDirection() order.Side
}
