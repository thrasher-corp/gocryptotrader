package bitstamp

import (
	"context"
	"net/http"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetAccountBalances returns the balances of all currencies
func (e *Exchange) GetAccountBalances(ctx context.Context) ([]AccountBalanceResponse, error) {
	var resp []AccountBalanceResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/account_balances/", nil, nil, &resp)
}

// GetAccountBalance returns the balance of a currency
func (e *Exchange) GetAccountBalance(ctx context.Context, c currency.Code) (*AccountBalanceResponse, error) {
	if c.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *AccountBalanceResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/account_balances/"+c.Lower().String()+"/", nil, nil, &resp)
}
