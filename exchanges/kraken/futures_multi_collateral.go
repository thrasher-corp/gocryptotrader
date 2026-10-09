package kraken

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetFuturesPNLCurrencyPreferences calls Get PNL currency preferences, returning the currency each multi-collateral
// contract pays realised profit in
func (e *Exchange) GetFuturesPNLCurrencyPreferences(ctx context.Context) (*FuturesPNLCurrencyPreferencesResponse, error) {
	var resp *FuturesPNLCurrencyPreferencesResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/pnlpreferences", nil, nil, &resp)
}

// SetFuturesPNLCurrencyPreference calls Set PNL currency preference, choosing the currency a multi-collateral
// contract pays realised profit in, such as USD or BTC. Kraken rejects a contract that does not exist (code 87) or is
// not multi-collateral (88), a currency that does not exist (89) or is not enabled for multi-collateral futures (90),
// and a change that would cause liquidation (41)
func (e *Exchange) SetFuturesPNLCurrencyPreference(ctx context.Context, symbol currency.Pair, pnlCurrency currency.Code) (*FuturesSetPreferenceResponse, error) {
	contract, err := e.futuresMarketSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if pnlCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("symbol", contract)
	params.Set("pnlPreference", pnlCurrency.Upper().String())
	var resp *FuturesSetPreferenceResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPut, "/api/v3/pnlpreferences", params, nil, &resp)
}

// GetFuturesLeverageSettings calls Get leverage settings, returning the maximum leverage configured for each contract
func (e *Exchange) GetFuturesLeverageSettings(ctx context.Context) (*FuturesLeverageSettingsResponse, error) {
	var resp *FuturesLeverageSettingsResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/leveragepreferences", nil, nil, &resp)
}

// SetFuturesLeverageSetting calls Set leverage settings, setting a contract to isolated margin at a maximum leverage,
// or to cross margin without one. Kraken rejects a contract that does not exist (code 87) or is not multi-collateral
// (88), a change that would cause liquidation (41), and any change while maker protection holds one of the account's
// requests
func (e *Exchange) SetFuturesLeverageSetting(ctx context.Context, req *FuturesLeverageSettingRequest) (*FuturesSetPreferenceResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	contract, err := e.futuresMarketSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", contract)
	if req.MaximumLeverage != 0 {
		params.Set("maxLeverage", strconv.FormatFloat(req.MaximumLeverage, 'f', -1, 64))
	}
	var resp *FuturesSetPreferenceResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPut, "/api/v3/leveragepreferences", params, nil, &resp)
}
