package bingx

import (
	"context"
	"errors"
	"net/http"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errSignalSideRequired       = errors.New("signal side is required")
	errSignalEntryPriceRequired = errors.New("signal entry price is required")
	errSignalIDEmpty            = errors.New("signal ID empty")
)

// CreateSignal publishes a USDT-M perpetual futures signal and returns its sharing link, from Create Signal. It needs
// a KOL account with signal publishing permission, and BingX takes at most 100 signals a day
func (e *Exchange) CreateSignal(ctx context.Context, arg *CreateSignalRequest) (*CreateSignalResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, errSignalSideRequired
	}
	if arg.EntryPrice <= 0 {
		return nil, errSignalEntryPriceRequired
	}
	body := *arg
	var err error
	if body.Symbol, err = e.FormatExchangeCurrency(arg.Symbol, asset.USDTMarginedFutures); err != nil {
		return nil, err
	}
	var resp *CreateSignalResponse
	if err := e.SendAuthJSONBodyHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpot, http.MethodPost, "/openApi/social/v1/swap/signal", &body, createSignalRate, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.SignalID == "" {
		return nil, errSignalIDEmpty
	}
	return resp, nil
}
