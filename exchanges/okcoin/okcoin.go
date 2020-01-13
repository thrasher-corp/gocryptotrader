package okcoin

import (
	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/okgroup"
)

const (
	okCoinAuthRate     = 600
	okCoinUnauthRate   = 600
	okCoinAPIPath      = "api/"
	okCoinAPIURL       = "https://www.okcoin.com/" + okCoinAPIPath
	okCoinAPIVersion   = "/v3/"
	okCoinExchangeName = "OKCOIN International"
	okCoinWebsocketURL = "wss://real.okcoin.com:10442/ws/v3"
)

// OKCoin bases all methods off okgroup implementation
type OKCoin struct {
	okgroup.OKGroup
}

func (o *OKCoin) GetHistoricCandles(rangesize int, granularity int) ([]exchange.Candle, error) {
	return nil, common.ErrFunctionNotSupported
}
