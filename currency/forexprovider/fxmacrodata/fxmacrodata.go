package fxmacrodata

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errAPIKeyNotSet        = errors.New("api key must be set")
	errUnsupportedCurrency = errors.New("currency not supported by FXMacroData")
)

const (
	requestRateLimit = 60
	maxRateWorkers   = 4
)

// Setup sets appropriate values for the FXMacroData provider.
func (f *FXMacroData) Setup(config base.Settings) error {
	if config.APIKey == "" {
		return errAPIKeyNotSet
	}
	f.APIKey = config.APIKey
	f.APIKeyLvl = config.APIKeyLvl
	f.Enabled = config.Enabled
	f.Name = config.Name
	f.Verbose = config.Verbose
	f.PrimaryProvider = config.PrimaryProvider
	f.APIURL = APIURL

	var err error
	f.Requester, err = request.New(
		f.Name,
		common.NewHTTPClientWithTimeout(base.DefaultTimeOut),
		request.WithLimiter(request.NewBasicRateLimit(time.Minute, requestRateLimit, 1)),
	)
	return err
}

// GetSupportedCurrencies returns currencies covered by FXMacroData FX endpoints.
func (f *FXMacroData) GetSupportedCurrencies() ([]string, error) {
	return strings.Split(supportedCurrencies, ","), nil
}

// GetRates returns latest FX conversion rates for GoCryptoTrader's currency store.
func (f *FXMacroData) GetRates(baseCurrency, symbols string) (map[string]float64, error) {
	baseCurrency = strings.ToUpper(baseCurrency)
	supportedCurrencies, err := f.GetSupportedCurrencies()
	if err != nil {
		return nil, err
	}
	supported := make(map[string]struct{}, len(supportedCurrencies))
	for _, currency := range supportedCurrencies {
		supported[strings.ToUpper(currency)] = struct{}{}
	}
	if _, ok := supported[baseCurrency]; !ok {
		return nil, fmt.Errorf("%w: %s", errUnsupportedCurrency, baseCurrency)
	}

	targets := splitSymbols(symbols)
	if len(targets) == 0 {
		targets = supportedCurrencies
	}

	targetSymbols := make([]string, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	var unsupported []string
	for _, symbol := range targets {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		if symbol == "" || symbol == baseCurrency {
			continue
		}
		if _, ok := supported[symbol]; !ok {
			unsupported = append(unsupported, symbol)
			continue
		}
		if _, ok := seen[symbol]; ok {
			continue
		}
		seen[symbol] = struct{}{}
		targetSymbols = append(targetSymbols, symbol)
	}

	standardisedRates, err := f.getLatestForexRates(baseCurrency, targetSymbols)
	if err != nil {
		return nil, err
	}
	if len(standardisedRates) == 0 && len(unsupported) != 0 {
		return nil, fmt.Errorf("%w: %s", errUnsupportedCurrency, strings.Join(unsupported, ","))
	}
	return standardisedRates, nil
}

func (f *FXMacroData) getLatestForexRates(baseCurrency string, targetSymbols []string) (map[string]float64, error) {
	standardisedRates := make(map[string]float64, len(targetSymbols))
	if len(targetSymbols) == 0 {
		return standardisedRates, nil
	}

	workerCount := min(maxRateWorkers, len(targetSymbols))

	type rateResult struct {
		symbol string
		rate   float64
		err    error
	}

	jobs := make(chan string)
	results := make(chan rateResult, len(targetSymbols))
	var wg sync.WaitGroup
	wg.Add(workerCount)
	for range workerCount {
		go func() {
			defer wg.Done()
			for symbol := range jobs {
				rate, err := f.GetLatestForexRate(baseCurrency, symbol)
				results <- rateResult{
					symbol: symbol,
					rate:   rate,
					err:    err,
				}
			}
		}()
	}

	for _, symbol := range targetSymbols {
		jobs <- symbol
	}
	close(jobs)
	wg.Wait()
	close(results)

	for result := range results {
		if result.err != nil {
			return nil, result.err
		}
		standardisedRates[baseCurrency+result.symbol] = result.rate
	}
	return standardisedRates, nil
}

// GetLatestForexRate returns the latest available FXMacroData rate for a pair.
func (f *FXMacroData) GetLatestForexRate(baseCurrency, quoteCurrency string) (float64, error) {
	var resp forexResponse
	values := url.Values{}
	values.Set("limit", "1")
	err := f.SendHTTPRequest(
		"forex/"+strings.ToLower(baseCurrency)+"/"+strings.ToLower(quoteCurrency),
		values,
		&resp,
	)
	if err != nil {
		return 0, err
	}
	if len(resp.Data) == 0 {
		return 0, fmt.Errorf("no FXMacroData rate returned for %s/%s", baseCurrency, quoteCurrency)
	}
	return resp.Data[0].Val, nil
}

// DataCatalogue returns the available FXMacroData indicators for a currency.
func (f *FXMacroData) DataCatalogue(currency string) (map[string]any, error) {
	return f.getMap("data_catalogue/"+strings.ToLower(currency), nil)
}

// Announcements returns historical macro announcement rows.
func (f *FXMacroData) Announcements(currency, indicator string, values url.Values) (map[string]any, error) {
	return f.getMap("announcements/"+strings.ToLower(currency)+"/"+indicator, values)
}

// LatestAnnouncements returns latest announcements for a currency.
func (f *FXMacroData) LatestAnnouncements(currency string, values url.Values) (map[string]any, error) {
	return f.getMap("announcements/"+strings.ToLower(currency)+"/latest", values)
}

// AnnouncementChanges returns recently changed announcement rows.
func (f *FXMacroData) AnnouncementChanges(values url.Values) (map[string]any, error) {
	return f.getMap("announcements/changes", values)
}

// Calendar returns the release calendar for a currency.
func (f *FXMacroData) Calendar(currency string, values url.Values) (map[string]any, error) {
	return f.getMap("calendar/"+strings.ToLower(currency), values)
}

// Predictions returns consensus/model prediction rows.
func (f *FXMacroData) Predictions(currency, indicator string, values url.Values) (map[string]any, error) {
	return f.getMap("predictions/"+strings.ToLower(currency)+"/"+indicator, values)
}

// COT returns CFTC positioning data for a currency.
func (f *FXMacroData) COT(currency string, values url.Values) (map[string]any, error) {
	return f.getMap("cot/"+strings.ToLower(currency), values)
}

// Commodity returns a commodity time series.
func (f *FXMacroData) Commodity(indicator string, values url.Values) (map[string]any, error) {
	return f.getMap("commodities/"+indicator, values)
}

// CommoditiesLatest returns latest commodity points.
func (f *FXMacroData) CommoditiesLatest(values url.Values) (map[string]any, error) {
	return f.getMap("commodities/latest", values)
}

// Curves returns yield curve data for a currency.
func (f *FXMacroData) Curves(currency string, values url.Values) (map[string]any, error) {
	return f.getMap("curves/"+strings.ToLower(currency), values)
}

// CurveProxies returns curve proxy data for a currency.
func (f *FXMacroData) CurveProxies(currency string, values url.Values) (map[string]any, error) {
	return f.getMap("curve_proxies/"+strings.ToLower(currency), values)
}

// ForwardCurves returns forward curve data for a currency.
func (f *FXMacroData) ForwardCurves(currency string, values url.Values) (map[string]any, error) {
	return f.getMap("forward_curves/"+strings.ToLower(currency), values)
}

// RateDifferentials returns rate differentials for a pair.
func (f *FXMacroData) RateDifferentials(baseCurrency, quoteCurrency string, values url.Values) (map[string]any, error) {
	return f.getMap("rate_differentials/"+strings.ToLower(baseCurrency)+"/"+strings.ToLower(quoteCurrency), values)
}

// ForwardDifferentials returns forward differentials for a pair.
func (f *FXMacroData) ForwardDifferentials(baseCurrency, quoteCurrency string, values url.Values) (map[string]any, error) {
	return f.getMap("forward_differentials/"+strings.ToLower(baseCurrency)+"/"+strings.ToLower(quoteCurrency), values)
}

// MarketSessions returns FX market-session state.
func (f *FXMacroData) MarketSessions(values url.Values) (map[string]any, error) {
	return f.getMap("market_sessions", values)
}

// RiskSentiment returns risk sentiment data.
func (f *FXMacroData) RiskSentiment(values url.Values) (map[string]any, error) {
	return f.getMap("risk_sentiment", values)
}

// News returns macro news for a currency.
func (f *FXMacroData) News(currency string, values url.Values) (map[string]any, error) {
	return f.getMap("news/"+strings.ToLower(currency), values)
}

// PressReleases returns central-bank and official press releases.
func (f *FXMacroData) PressReleases(currency string, values url.Values) (map[string]any, error) {
	return f.getMap("press-releases/"+strings.ToLower(currency), values)
}

// GraphQL executes an FXMacroData GraphQL request.
func (f *FXMacroData) GraphQL(payload string, result any) error {
	return f.send("graphql", nil, strings.NewReader(payload), http.MethodPost, result)
}

func (f *FXMacroData) getMap(endpoint string, values url.Values) (map[string]any, error) {
	resp := make(map[string]any)
	return resp, f.SendHTTPRequest(endpoint, values, &resp)
}

// SendHTTPRequest sends an authenticated FXMacroData GET request.
func (f *FXMacroData) SendHTTPRequest(endpoint string, values url.Values, result any) error {
	return f.send(endpoint, values, nil, http.MethodGet, result)
}

func (f *FXMacroData) send(endpoint string, values url.Values, body io.Reader, method string, result any) error {
	if f.APIKey == "" {
		return errAPIKeyNotSet
	}
	query := make(url.Values, len(values)+1)
	for k, v := range values {
		query[k] = append([]string(nil), v...)
	}
	query.Set("api_key", f.APIKey)

	baseURL := strings.TrimRight(f.APIURL, "/") + "/"
	path := common.EncodeURLValues(baseURL+strings.TrimLeft(endpoint, "/"), query)
	headers := map[string]string{"Accept": "application/json"}
	if method == http.MethodPost {
		headers["Content-Type"] = "application/json"
	}

	item := &request.Item{
		Method:  method,
		Path:    path,
		Headers: headers,
		Result:  result,
		Verbose: f.Verbose,
	}
	if body != nil {
		item.Body = body
	}
	return f.Requester.SendPayload(context.Background(), request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
}

func splitSymbols(symbols string) []string {
	if strings.TrimSpace(symbols) == "" {
		return nil
	}
	parts := strings.Split(symbols, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
