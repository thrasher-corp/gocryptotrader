package okcoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/okgroup"
)

const (
	okCoinRateInterval        = time.Second
	okCoinStandardRequestRate = 6
	okCoinAPIPath             = "api/"
	okCoinAPIURL              = "https://www.okcoin.com/" + okCoinAPIPath
	okCoinAPIVersion          = "/v3/"
	okCoinExchangeName        = "OKCOIN International"
	okCoinWebsocketURL        = "wss://real.okcoin.com:10442/ws/v3"
)

// OKCoin bases all methods off okgroup implementation
type OKCoin struct {
	okgroup.OKGroup
}

// GetHistoricCandles returns rangesize number of candles for the given granularity and pair starting from the latest available
func (o *OKCoin) GetHistoricCandles(pair currency.Pair, rangesize, granularity int64) ([]exchange.Candle, error) {
	return nil, common.ErrFunctionNotSupported
}
