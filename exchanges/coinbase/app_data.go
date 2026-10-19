package coinbase

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// GetFiatCurrencies returns the fiat currencies the Coinbase App knows of; it needs no authentication
func (a AppAPI) GetFiatCurrencies(ctx context.Context) ([]AppFiatCurrency, error) {
	resp, err := sendPublicAppRequest[[]AppFiatCurrency](ctx, a, "currencies", nil)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetCryptocurrencies returns the cryptocurrencies the Coinbase App knows of; it needs no authentication. The docs
// show a bare list, but Coinbase sends it in the data envelope as its other responses
func (a AppAPI) GetCryptocurrencies(ctx context.Context) ([]AppCurrency, error) {
	resp, err := sendPublicAppRequest[[]AppCurrency](ctx, a, "currencies/crypto", nil)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetExchangeRates returns what one unit of a base currency, USD when it is empty, is worth in each currency Coinbase
// prices; it needs no authentication
func (a AppAPI) GetExchangeRates(ctx context.Context, base currency.Code) (*AppExchangeRates, error) {
	params := url.Values{}
	if !base.IsEmpty() {
		params.Set("currency", base.Upper().String())
	}
	resp, err := sendPublicAppRequest[*AppExchangeRates](ctx, a, "exchange-rates", params)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// GetBuyPrice returns the total price of buying one unit of a pair's base currency, including Coinbase's standard 1%
// fee but not a payment method's fees; it holds for seconds and needs no authentication
func (a AppAPI) GetBuyPrice(ctx context.Context, pair currency.Pair) (*AppPrice, error) {
	return a.getPrice(ctx, pair, "buy", nil)
}

// GetSellPrice returns the total price of selling one unit of a pair's base currency, including Coinbase's standard
// 1% fee but not a payment method's fees; it holds for seconds and needs no authentication
func (a AppAPI) GetSellPrice(ctx context.Context, pair currency.Pair) (*AppPrice, error) {
	return a.getPrice(ctx, pair, "sell", nil)
}

// GetSpotPrice returns a pair's market price, usually between its buy and sell prices, now or on a past UTC day; it
// needs no authentication
func (a AppAPI) GetSpotPrice(ctx context.Context, req *AppSpotPriceRequest) (*AppPrice, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Date.IsZero() {
		params.Set("date", req.Date.UTC().Format(time.DateOnly))
	}
	return a.getPrice(ctx, req.Pair, "spot", params)
}

// GetCurrentTime returns the Coinbase App API's time; it needs no authentication
func (a AppAPI) GetCurrentTime(ctx context.Context) (*AppServerTime, error) {
	resp, err := sendPublicAppRequest[*AppServerTime](ctx, a, "time", nil)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// getPrice returns a pair's buy, sell or spot price
func (a AppAPI) getPrice(ctx context.Context, pair currency.Pair, kind string, params url.Values) (*AppPrice, error) {
	if pair.Base.IsEmpty() || pair.Quote.IsEmpty() {
		return nil, fmt.Errorf("%w: %q", currency.ErrCurrencyPairEmpty, pair)
	}
	resp, err := sendPublicAppRequest[*AppPrice](ctx, a, "prices/"+pair.Base.Upper().String()+"-"+pair.Quote.Upper().String()+"/"+kind, params)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}
