package binance

import (
	"context"
	"errors"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

const (
	// Without subscriptions
	binanceEOptionWebsocketURL       = "wss://nbstream.binance.com/eoptions/ws"
	binanceEOptionWebsocketURLStream = "wss://nbstream.binance.com/eoptions/stream"
)

var (
	errUnderlyingIsRequired = errors.New("underlying is required")
)

// CheckEOptionsServerTime retrieves the server time.
func (b *Binance) CheckEOptionsServerTime(ctx context.Context) (convert.ExchangeTime, error) {
	resp := &struct {
		ServerTime convert.ExchangeTime `json:"serverTime"`
	}{}
	return resp.ServerTime, b.SendHTTPRequest(ctx, exchange.RestOptions, "/eapi/v1/time", spotDefaultRate, &resp)
}

// GetOptionsExchangeInformation retrieves an exchange information through the options endpoint.
func (b *Binance) GetOptionsExchangeInformation(ctx context.Context) (*EOptionExchangeInfo, error) {
	var resp *EOptionExchangeInfo
	return resp, b.SendHTTPRequest(ctx, exchange.RestOptions, "/eapi/v1/exchangeInfo", spotDefaultRate, &resp)
}
