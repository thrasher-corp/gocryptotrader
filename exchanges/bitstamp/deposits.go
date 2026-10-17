package bitstamp

import (
	"context"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetCryptoDepositAddress returns a deposit address for a cryptocurrency
// network is optional and selects the currency network of the address
func (e *Exchange) GetCryptoDepositAddress(ctx context.Context, c currency.Code, network string) (*DepositAddressResponse, error) {
	if c.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if network != "" {
		params.Set("network", network)
	}
	var resp *DepositAddressResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/"+c.Lower().String()+"_address/", nil, params, &resp)
}

// GetUnconfirmedBitcoinDeposits returns the account's unconfirmed bitcoin deposits
func (e *Exchange) GetUnconfirmedBitcoinDeposits(ctx context.Context) ([]UnconfirmedDepositResponse, error) {
	var resp unconfirmedDeposits
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/btc_unconfirmed/", nil, nil, &resp)
}

// GetRippleIOUDepositAddress returns the account's Ripple IOU deposit address
func (e *Exchange) GetRippleIOUDepositAddress(ctx context.Context) (*RippleIOUDepositAddressResponse, error) {
	var resp *RippleIOUDepositAddressResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/ripple_address/", nil, nil, &resp)
}
