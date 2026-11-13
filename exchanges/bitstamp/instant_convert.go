package bitstamp

import (
	"context"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// CreateInstantConvertAddress creates an address which automatically sells deposited bitcoin for the liquidation currency
// addressFormat is optional and may be P2SHP2WSH or BECH32
func (e *Exchange) CreateInstantConvertAddress(ctx context.Context, liquidationCurrency currency.Code, addressFormat string) (*InstantConvertAddressResponse, error) {
	if liquidationCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("liquidation_currency", liquidationCurrency.Upper().String())
	if addressFormat != "" {
		params.Set("address_format", addressFormat)
	}
	var resp *InstantConvertAddressResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/instant_convert_address/new/", nil, params, &resp)
}

// GetInstantConvertAddressInfo returns the transactions of the account's instant convert addresses
// address is optional and restricts the results to a single address
func (e *Exchange) GetInstantConvertAddressInfo(ctx context.Context, address string) ([]InstantConvertAddressInfoResponse, error) {
	params := url.Values{}
	if address != "" {
		params.Set("address", address)
	}
	var resp []InstantConvertAddressInfoResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/instant_convert_address/info/", nil, params, &resp)
}
