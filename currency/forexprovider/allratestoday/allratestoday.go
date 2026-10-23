// Package allratestoday provides current exchange rates for 160+ fiat currencies
// through a JSON API. Every rate for a base currency is returned in a single
// request. An API key is required (free trial, paid plans).
package allratestoday

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errAPIKeyNotSet       = errors.New("AllRatesToday API key is not set")
	errEmptyBase          = errors.New("base currency must not be empty")
	errNoRatesReturn      = errors.New("no AllRatesToday rates returned")
	errNoSupportedTargets = errors.New("no supported AllRatesToday target currencies requested")
	errInvalidTimestamp   = errors.New("invalid AllRatesToday timestamp")
)

// Setup sets appropriate values for the AllRatesToday provider
func (a *AllRatesToday) Setup(config base.Settings) error {
	a.APIKey = config.APIKey
	a.APIKeyLvl = config.APIKeyLvl
	a.Enabled = config.Enabled
	a.Name = config.Name
	a.Verbose = config.Verbose
	a.PrimaryProvider = config.PrimaryProvider
	a.APIURL = APIURL
	var err error
	a.Requester, err = request.New(a.Name,
		common.NewHTTPClientWithTimeout(base.DefaultTimeOut))
	return err
}

// GetSupportedCurrencies returns a list of supported currencies
func (a *AllRatesToday) GetSupportedCurrencies() ([]string, error) {
	return strings.Split(supportedCurrencies, ","), nil
}

// GetRates is a wrapper function to return rates keyed by base and target,
// for example "USDEUR". An empty symbols string returns every available rate.
// Targets outside the supported currency list are dropped, as the API fails the
// whole request on an unlisted code; if none remain an error is returned.
// baseCurrency is a string because package currency imports the forex providers.
func (a *AllRatesToday) GetRates(baseCurrency, symbols string) (map[string]float64, error) {
	if symbols = strings.ToUpper(strings.ReplaceAll(symbols, " ", "")); symbols != "" {
		supported := strings.Split(supportedCurrencies, ",")
		targets := slices.DeleteFunc(strings.Split(symbols, ","), func(target string) bool {
			return !slices.Contains(supported, target)
		})
		if len(targets) == 0 {
			return nil, errNoSupportedTargets
		}
		symbols = strings.Join(targets, ",")
	}

	rates, err := a.GetLatestRates(context.TODO(), baseCurrency, symbols)
	if err != nil {
		return nil, err
	}

	standardisedRates := make(map[string]float64, len(rates))
	for i := range rates {
		standardisedRates[strings.ToUpper(rates[i].Source+rates[i].Target)] = rates[i].Rate
	}
	return standardisedRates, nil
}

// GetLatestRates returns the latest rates for a base currency. Symbols is an
// optional comma separated list of target currencies.
// baseCurrency is a string because package currency imports the forex providers.
func (a *AllRatesToday) GetLatestRates(ctx context.Context, baseCurrency, symbols string) ([]RateResponse, error) {
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	if baseCurrency == "" {
		return nil, errEmptyBase
	}
	if a.APIKey == "" {
		return nil, errAPIKeyNotSet
	}

	v := url.Values{}
	v.Set("source", baseCurrency)
	if symbols = strings.ToUpper(strings.ReplaceAll(symbols, " ", "")); symbols != "" {
		v.Set("target", symbols)
	}

	var resp []RateResponse
	if err := a.sendHTTPRequest(ctx, "rates", v, &resp); err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoRatesReturn
	}
	return resp, nil
}

// sendHTTPRequest sends an authenticated request to the AllRatesToday API
func (a *AllRatesToday) sendHTTPRequest(ctx context.Context, endpoint string, values url.Values, result any) error {
	path := common.EncodeURLValues(strings.TrimRight(a.APIURL, "/")+"/"+endpoint, values)
	item := &request.Item{
		Method: http.MethodGet,
		Path:   path,
		Headers: map[string]string{
			"Accept":        "application/json",
			"Authorization": "Bearer " + a.APIKey,
		},
		Result:  result,
		Verbose: a.Verbose,
	}
	return a.Requester.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
}
