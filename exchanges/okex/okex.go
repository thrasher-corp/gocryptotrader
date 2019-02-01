package okex

import (
	"fmt"

	"github.com/thrasher-/gocryptotrader/exchanges/okgroup"
)

// Okex is okgroup implementation for OKEX
var Okex = okgroup.OKGroup{
	APIURL:       fmt.Sprintf("%v%v", "https://www.okex.com/", okgroup.OkGroupAPIPath),
	APIVersion:   "/v3/",
	ExchangeName: "OKEX",
	WebsocketURL: "wss://real.okex.com:10440/websocket/okexapi",
}
