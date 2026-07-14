package fxmacrodata

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errUnsupportedCurrency = errors.New("currency not supported by FXMacroData")
	errEmptyCurrency       = errors.New("currency symbol must not be empty")
	errDuplicateCurrency   = errors.New("duplicate currency symbol")
	errNoTargetCurrencies  = errors.New("at least one target currency is required")
)

const (
	requestRateLimit = 60
)

// Setup sets appropriate values for the FXMacroData provider.
func (f *FXMacroData) Setup(config base.Settings) error {
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
		// Keep requests within the documented 60-per-minute allowance. A burst of
		// one is deliberate: the request limiter does not permit outbound bursts.
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
	return f.GetRatesContext(context.Background(), baseCurrency, symbols)
}

// GetRatesContext returns latest FX conversion rates and honours ctx while
// waiting for rate-limit capacity and while sending each request.
func (f *FXMacroData) GetRatesContext(ctx context.Context, baseCurrency, symbols string) (map[string]float64, error) {
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	supportedCurrencies, err := f.GetSupportedCurrencies()
	if err != nil {
		return nil, err
	}
	supported := make(map[string]struct{}, len(supportedCurrencies))
	for _, currency := range supportedCurrencies {
		supported[strings.ToUpper(currency)] = struct{}{}
	}
	if _, ok := supported[baseCurrency]; !ok {
		return nil, fmt.Errorf("%w: %q", errUnsupportedCurrency, baseCurrency)
	}

	targets := splitSymbols(symbols)
	if len(targets) == 0 {
		targets = supportedCurrencies
	}

	targetSymbols := make([]string, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	var unsupported []string
	for _, symbol := range targets {
		symbol = strings.ToUpper(symbol)
		if symbol == "" {
			return nil, errEmptyCurrency
		}
		if symbol == baseCurrency {
			continue
		}
		if _, ok := supported[symbol]; !ok {
			unsupported = append(unsupported, symbol)
			continue
		}
		if _, ok := seen[symbol]; ok {
			return nil, fmt.Errorf("%w: %s", errDuplicateCurrency, symbol)
		}
		seen[symbol] = struct{}{}
		targetSymbols = append(targetSymbols, symbol)
	}
	if len(targetSymbols) == 0 && len(unsupported) != 0 {
		return nil, fmt.Errorf("%w: %s", errUnsupportedCurrency, strings.Join(unsupported, ","))
	}

	standardisedRates, err := f.getLatestForexRates(ctx, baseCurrency, targetSymbols)
	if err != nil {
		return nil, err
	}
	if len(standardisedRates) == 0 && len(unsupported) != 0 {
		return nil, fmt.Errorf("%w: %s", errUnsupportedCurrency, strings.Join(unsupported, ","))
	}
	return standardisedRates, nil
}

func (f *FXMacroData) getLatestForexRates(ctx context.Context, baseCurrency string, targetSymbols []string) (map[string]float64, error) {
	if len(targetSymbols) == 0 {
		return nil, errNoTargetCurrencies
	}
	standardisedRates := make(map[string]float64, len(targetSymbols))

	for _, quote := range targetSymbols {
		rate, err := f.GetLatestForexRateContext(ctx, baseCurrency, quote)
		if err != nil {
			return nil, err
		}
		standardisedRates[baseCurrency+quote] = rate
	}
	return standardisedRates, nil
}

// GetLatestForexRate returns the latest available FXMacroData rate for a pair.
func (f *FXMacroData) GetLatestForexRate(baseCurrency, quoteCurrency string) (float64, error) {
	return f.GetLatestForexRateContext(context.Background(), baseCurrency, quoteCurrency)
}

// GetLatestForexRateContext returns the latest available FXMacroData rate for a
// pair and honours ctx while sending the request.
func (f *FXMacroData) GetLatestForexRateContext(ctx context.Context, baseCurrency, quoteCurrency string) (float64, error) {
	var resp forexResponse
	values := url.Values{}
	values.Set("limit", "1")
	err := f.SendHTTPRequestContext(ctx,
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

// Health returns the public FXMacroData service health status.
func (f *FXMacroData) Health() (*ServiceStatusResponse, error) {
	return f.HealthContext(context.Background())
}

// HealthContext returns the public FXMacroData service health status.
func (f *FXMacroData) HealthContext(ctx context.Context) (*ServiceStatusResponse, error) {
	response := new(ServiceStatusResponse)
	return response, f.sendPublic(ctx, "health", response)
}

// Ping returns the public FXMacroData service liveness status.
func (f *FXMacroData) Ping() (*ServiceStatusResponse, error) {
	return f.PingContext(context.Background())
}

// PingContext returns the public FXMacroData service liveness status.
func (f *FXMacroData) PingContext(ctx context.Context) (*ServiceStatusResponse, error) {
	response := new(ServiceStatusResponse)
	return response, f.sendPublic(ctx, "ping", response)
}

// DataCatalogue returns the available FXMacroData indicators for a currency.
func (f *FXMacroData) DataCatalogue(currency string) (*DataCatalogueResponse, error) {
	response := new(DataCatalogueResponse)
	return response, f.getResponse("data_catalogue/"+strings.ToLower(currency), nil, response)
}

// Announcements returns historical macro announcement rows.
func (f *FXMacroData) Announcements(currency, indicator string, values url.Values) (*AnnouncementResponse, error) {
	response := new(AnnouncementResponse)
	return response, f.getResponse("announcements/"+strings.ToLower(currency)+"/"+indicator, values, response)
}

// LatestAnnouncements returns latest announcements for a currency.
func (f *FXMacroData) LatestAnnouncements(currency string, values url.Values) (*AnnouncementResponse, error) {
	response := new(AnnouncementResponse)
	return response, f.getResponse("announcements/"+strings.ToLower(currency)+"/latest", values, response)
}

// AnnouncementChanges returns recently changed announcement rows.
func (f *FXMacroData) AnnouncementChanges(values url.Values) (*AnnouncementChangesResponse, error) {
	response := new(AnnouncementChangesResponse)
	return response, f.getResponse("announcements/changes", values, response)
}

// Calendar returns the release calendar for a currency.
func (f *FXMacroData) Calendar(currency string, values url.Values) (*CalendarResponse, error) {
	response := new(CalendarResponse)
	return response, f.getResponse("calendar/"+strings.ToLower(currency), values, response)
}

// Predictions returns consensus/model prediction rows.
func (f *FXMacroData) Predictions(currency, indicator string, values url.Values) (*PredictionsResponse, error) {
	response := new(PredictionsResponse)
	return response, f.getResponse("predictions/"+strings.ToLower(currency)+"/"+indicator, values, response)
}

// COT returns CFTC positioning data for a currency.
func (f *FXMacroData) COT(currency string, values url.Values) (*COTResponse, error) {
	response := new(COTResponse)
	return response, f.getResponse("cot/"+strings.ToLower(currency), values, response)
}

// Commodity returns a commodity time series.
func (f *FXMacroData) Commodity(indicator string, values url.Values) (*CommodityResponse, error) {
	response := new(CommodityResponse)
	return response, f.getResponse("commodities/"+indicator, values, response)
}

// CommoditiesLatest returns latest commodity points.
func (f *FXMacroData) CommoditiesLatest(values url.Values) (*CommodityResponse, error) {
	response := new(CommodityResponse)
	return response, f.getResponse("commodities/latest", values, response)
}

// Curves returns yield curve data for a currency.
func (f *FXMacroData) Curves(currency string, values url.Values) (*CurveSnapshotResponse, error) {
	response := new(CurveSnapshotResponse)
	return response, f.getResponse("curves/"+strings.ToLower(currency), values, response)
}

// CurveProxies returns curve proxy data for a currency.
func (f *FXMacroData) CurveProxies(currency string, values url.Values) (*CurveProxyResponse, error) {
	response := new(CurveProxyResponse)
	return response, f.getResponse("curve_proxies/"+strings.ToLower(currency), values, response)
}

// ForwardCurves returns forward curve data for a currency.
func (f *FXMacroData) ForwardCurves(currency string, values url.Values) (*ForwardCurveResponse, error) {
	response := new(ForwardCurveResponse)
	return response, f.getResponse("forward_curves/"+strings.ToLower(currency), values, response)
}

// RateDifferentials returns rate differentials for a pair.
func (f *FXMacroData) RateDifferentials(baseCurrency, quoteCurrency string, values url.Values) (*RateDifferentialResponse, error) {
	response := new(RateDifferentialResponse)
	return response, f.getResponse("rate_differentials/"+strings.ToLower(baseCurrency)+"/"+strings.ToLower(quoteCurrency), values, response)
}

// ForwardDifferentials returns forward differentials for a pair.
func (f *FXMacroData) ForwardDifferentials(baseCurrency, quoteCurrency string, values url.Values) (*ForwardDifferentialResponse, error) {
	response := new(ForwardDifferentialResponse)
	return response, f.getResponse("forward_differentials/"+strings.ToLower(baseCurrency)+"/"+strings.ToLower(quoteCurrency), values, response)
}

// MarketSessions returns FX market-session state.
func (f *FXMacroData) MarketSessions(values url.Values) (*MarketSessionsResponse, error) {
	response := new(MarketSessionsResponse)
	return response, f.getResponse("market_sessions", values, response)
}

// RiskSentiment returns risk sentiment data.
func (f *FXMacroData) RiskSentiment(values url.Values) (*RiskSentimentResponse, error) {
	response := new(RiskSentimentResponse)
	return response, f.getResponse("risk_sentiment", values, response)
}

// News returns macro news for a currency.
func (f *FXMacroData) News(currency string, values url.Values) (*NewsResponse, error) {
	response := new(NewsResponse)
	return response, f.getResponse("news/"+strings.ToLower(currency), values, response)
}

// PressReleases returns central-bank and official press releases.
func (f *FXMacroData) PressReleases(currency string, values url.Values) (*PressReleasesResponse, error) {
	response := new(PressReleasesResponse)
	return response, f.getResponse("press-releases/"+strings.ToLower(currency), values, response)
}

// GraphQL executes an FXMacroData GraphQL request.
func (f *FXMacroData) GraphQL(payload string, result any) error {
	return f.GraphQLContext(context.Background(), payload, result)
}

// GraphQLContext executes an FXMacroData GraphQL request and honours ctx.
func (f *FXMacroData) GraphQLContext(ctx context.Context, payload string, result any) error {
	return f.send(ctx, "graphql", nil, strings.NewReader(payload), http.MethodPost, result)
}

func (f *FXMacroData) getResponse(endpoint string, values url.Values, result any) error {
	return f.SendHTTPRequest(endpoint, values, result)
}

// SendHTTPRequest sends an authenticated FXMacroData GET request.
func (f *FXMacroData) SendHTTPRequest(endpoint string, values url.Values, result any) error {
	return f.SendHTTPRequestContext(context.Background(), endpoint, values, result)
}

// SendHTTPRequestContext sends an FXMacroData GET request and honours ctx.
// If no API key is configured, it sends an unauthenticated request so public
// FXMacroData endpoints remain usable; protected endpoints report their normal
// server-side authorisation error.
func (f *FXMacroData) SendHTTPRequestContext(ctx context.Context, endpoint string, values url.Values, result any) error {
	return f.send(ctx, endpoint, values, nil, http.MethodGet, result)
}

func (f *FXMacroData) send(ctx context.Context, endpoint string, values url.Values, body io.Reader, method string, result any) error {
	query := make(url.Values, len(values))
	for k, v := range values {
		query[k] = append([]string(nil), v...)
	}

	baseURL := strings.TrimRight(f.APIURL, "/") + "/"
	path := common.EncodeURLValues(baseURL+strings.TrimLeft(endpoint, "/"), query)
	headers := map[string]string{
		"Accept": "application/json",
	}
	var requestType request.AuthType = request.UnauthenticatedRequest
	if f.APIKey != "" {
		headers["X-API-Key"] = f.APIKey
		requestType = request.AuthenticatedRequest
	}
	if method == http.MethodPost {
		headers["Content-Type"] = "application/json"
	}

	item := &request.Item{
		Method:  method,
		Path:    path,
		Headers: headers,
		Body:    body,
		Result:  result,
		Verbose: f.Verbose,
	}
	return f.Requester.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, requestType)
}

func (f *FXMacroData) sendPublic(ctx context.Context, endpoint string, result any) error {
	item := &request.Item{
		Method: http.MethodGet,
		Path:   strings.TrimRight(f.APIURL, "/") + "/" + strings.TrimLeft(endpoint, "/"),
		Headers: map[string]string{
			"Accept": "application/json",
		},
		Result:  result,
		Verbose: f.Verbose,
	}
	return f.Requester.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

func splitSymbols(symbols string) []string {
	symbols = strings.TrimSpace(symbols)
	if symbols == "" {
		return nil
	}
	values := strings.Split(symbols, ",")
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	return values
}
