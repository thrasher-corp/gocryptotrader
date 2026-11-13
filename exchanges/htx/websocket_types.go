package htx

import (
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// wsSubscriptionKey distinguishes identical topics on different HTX connections while
// retaining ExactKey matching for regenerated subscription lists.
type wsSubscriptionKey struct {
	*subscription.ExactKey
	connection websocket.Connection
}

// websocketEndpoint associates each managed connection with its HTX API role.
type websocketEndpoint struct {
	endpoint exchange.URL
	asset    asset.Item
	private  bool
	trade    bool
}
