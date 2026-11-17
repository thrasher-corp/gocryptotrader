package bingx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Please supply your own keys here for due diligence testing
const canManipulateRealOrders = false

// apiCredentials holds the credentials used for due diligence testing; please supply your own
var apiCredentials = &accounts.Credentials{
	Key:    "",
	Secret: "",
}

var (
	e *Exchange

	// enabled and active tradable pairs used to test endpoints
	spotTradablePair, usdtMarginedTradablePair, coinMarginedTradablePair currency.Pair

	assetToTradablePairMap map[asset.Item]currency.Pair
)
