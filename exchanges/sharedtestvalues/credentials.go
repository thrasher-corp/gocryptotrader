package sharedtestvalues

import (
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// SetTestCredentials enables an exchange's authenticated REST and websocket support and sets the credentials given
func SetTestCredentials(exch interface{ GetBase() *exchange.Base }, creds *accounts.Credentials) {
	b := exch.GetBase()
	b.API.AuthenticatedSupport = true
	b.API.AuthenticatedWebsocketSupport = true
	b.SetCredentials(creds)
}
