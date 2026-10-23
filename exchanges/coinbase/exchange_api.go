package coinbase

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetCurrencies returns every currency Coinbase Exchange knows of, delisted ones included, with the networks each is
// deposited and withdrawn on and their withdrawal limits; not every currency trades
func (e *Exchange) GetCurrencies(ctx context.Context) ([]ExchangeCurrencyResponse, error) {
	var resp []ExchangeCurrencyResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, exchangeAPIPublicEPL, "/currencies", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetCurrency returns a currency Coinbase Exchange knows of, with the networks it is deposited and withdrawn on and
// their withdrawal limits
func (e *Exchange) GetCurrency(ctx context.Context, code currency.Code) (*ExchangeCurrencyResponse, error) {
	if code.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *ExchangeCurrencyResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, exchangeAPIPublicEPL, "/currencies/"+code.Upper().String(), nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}
