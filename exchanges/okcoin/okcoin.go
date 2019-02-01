package okcoin

import (
	"fmt"

	"github.com/thrasher-/gocryptotrader/exchanges/okgroup"
)

// Okex is okgroup implementation for OKEX
var OkCoin = okgroup.OKGroup{
	APIURL:       fmt.Sprintf("%v%v", "https://www.okcoin.com/", okgroup.OkGroupAPIPath),
	APIVersion:   "/v3/",
	ExchangeName: "OKCoin",
	WebsocketURL: "wss://real.okcoin.com:10440/websocket/okcoinapi",
}
