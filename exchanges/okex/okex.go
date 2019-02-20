package okex

import (
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/okgroup"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	okExAuthRate     = 0
	okExUnauthRate   = 0
	okExAPIPath      = "api/"
	okExAPIURL       = "https://www.okex.com/" + okExAPIPath
	okExAPIVersion   = "/v3/"
	okExExchangeName = "OKEX"
	okExWebsocketURL = "wss://real.okex.com:10440/websocket/okexapi"
)

// OKEX bases all methods off okgroup implementation
type OKEX struct {
	okgroup.OKGroup
}

// SetDefaults method assignes the default values for OKEX
func (o *OKEX) SetDefaults() {
	o.SetErrorDefaults()
	o.SetCheckVarDefaults()
	o.Name = okExExchangeName
	o.Enabled = false
	o.Verbose = false
	o.RESTPollingDelay = 10
	o.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.NoFiatWithdrawals
	o.RequestCurrencyPairFormat.Delimiter = "_"
	o.RequestCurrencyPairFormat.Uppercase = false
	o.ConfigCurrencyPairFormat.Delimiter = "_"
	o.ConfigCurrencyPairFormat.Uppercase = true
	o.SupportsAutoPairUpdating = true
	o.SupportsRESTTickerBatching = false
	o.Requester = request.New(o.Name,
		request.NewRateLimit(time.Second, okExAuthRate),
		request.NewRateLimit(time.Second, okExUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	o.APIUrlDefault = okExAPIURL
	o.APIUrl = okExAPIURL
	o.AssetTypes = []string{ticker.Spot}
	o.WebsocketInit()
	o.APIVersion = okExAPIVersion
	o.Websocket.Functionality = exchange.WebsocketTickerSupported |
		exchange.WebsocketTradeDataSupported |
		exchange.WebsocketKlineSupported |
		exchange.WebsocketOrderbookSupported
}
