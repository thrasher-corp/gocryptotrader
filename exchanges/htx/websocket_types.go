package htx

import (
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// wsSubscriptionKey distinguishes identical topics on different HTX connections while
// retaining ExactKey matching for regenerated subscription lists.
type wsSubscriptionKey struct {
	*subscription.ExactKey
	connection websocket.Connection
}
