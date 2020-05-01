package okcoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/okgroup"
)

const (
	okCoinRateInterval        = time.Second
	okCoinStandardRequestRate = 6
	okCoinAPIPath             = "api/"
	okCoinAPIURL              = "https://www.okcoin.com/" + okCoinAPIPath
	okCoinAPIVersion          = "/v3/"
	okCoinExchangeName        = "OKCOIN International"
	okCoinWebsocketURL        = "wss://real.okcoin.com:8443/ws/v3"
)

// OKCoin bases all methods off okgroup implementation
type OKCoin struct {
	okgroup.OKGroup
}
