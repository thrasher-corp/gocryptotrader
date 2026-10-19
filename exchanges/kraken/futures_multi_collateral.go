package kraken

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// GetFuturesPNLCurrencyPreferences calls Get PNL currency preferences, returning the currency each multi-collateral
// contract pays realised profit in
func (e *Exchange) GetFuturesPNLCurrencyPreferences(ctx context.Context) (*FuturesPNLCurrencyPreferencesResponse, error) {
	var resp *FuturesPNLCurrencyPreferencesResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/pnlpreferences", nil, nil, &resp)
}

// SetFuturesPNLCurrencyPreference calls Set PNL currency preference, choosing the currency a multi-collateral
// contract pays realised profit in, such as USD or BTC, or, for an empty symbol, the account-wide currency of every
// contract without a preference of its own. Kraken rejects a contract that does not exist (code 87) or is not
// multi-collateral (88), a currency that does not exist (89) or is not enabled for multi-collateral futures (90), and
// a change that would cause liquidation (41)
func (e *Exchange) SetFuturesPNLCurrencyPreference(ctx context.Context, symbol currency.Pair, pnlCurrency currency.Code) (*FuturesSetPreferenceResponse, error) {
	params, err := e.futuresPNLPreferenceParams(symbol)
	if err != nil {
		return nil, err
	}
	if pnlCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params.Set("pnlPreference", pnlCurrency.Upper().String())
	var resp *FuturesSetPreferenceResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPut, "/api/v3/pnlpreferences", params, nil, &resp)
}

// DeleteFuturesPNLCurrencyPreference calls Delete PNL currency preference, returning a multi-collateral contract to
// the account-wide preference, or, for an empty symbol, removing the account-wide preference, so every contract
// following it pays realised profit in its platform's currency. Removing a preference that is not set succeeds.
// Kraken rejects a removal when the account does not exist (code 13), or when it would cause liquidation (41) or leave
// the account short of margin (92)
func (e *Exchange) DeleteFuturesPNLCurrencyPreference(ctx context.Context, symbol currency.Pair) (*FuturesSetPreferenceResponse, error) {
	params, err := e.futuresPNLPreferenceParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp *FuturesSetPreferenceResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodDelete, "/api/v3/pnlpreferences", params, nil, &resp)
}

// futuresPNLPreferenceParams returns the symbol parameter the PNL currency preference endpoints take, which an empty
// pair leaves out to address the account-wide preference
func (e *Exchange) futuresPNLPreferenceParams(symbol currency.Pair) (url.Values, error) {
	if err := checkPairFilter(symbol); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !symbol.IsEmpty() {
		contract, err := e.FormatSymbol(symbol, asset.Futures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", contract)
	}
	return params, nil
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
	if !isFiniteNonNegative(req.MaximumLeverage) {
		return nil, fmt.Errorf("%w: maximum leverage %v", errFuturesInvalidLimit, req.MaximumLeverage)
	}
	params := url.Values{}
	params.Set("symbol", contract)
	if req.MaximumLeverage != 0 {
		params.Set("maxLeverage", strconv.FormatFloat(req.MaximumLeverage, 'f', -1, 64))
	}
	var resp *FuturesSetPreferenceResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPut, "/api/v3/leveragepreferences", params, nil, &resp)
}
