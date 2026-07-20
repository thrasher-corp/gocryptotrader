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
	errAPIKeyNotConfigured = errors.New("FXMacroData API key is required for this endpoint")
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
func (f *FXMacroData) GetRates(ctx context.Context, baseCurrency, symbols string) (map[string]float64, error) {
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
		rate, err := f.GetLatestForexRate(ctx, baseCurrency, quote)
		if err != nil {
			return nil, err
		}
		standardisedRates[baseCurrency+quote] = rate
	}
	return standardisedRates, nil
}

// GetLatestForexRate returns the latest available FXMacroData rate for a pair
// and honours ctx while sending the request.
func (f *FXMacroData) GetLatestForexRate(ctx context.Context, baseCurrency, quoteCurrency string) (float64, error) {
	var resp forexResponse
	values := url.Values{}
	values.Set("limit", "1")
	err := f.sendHTTPAuthRequest(ctx,
		"forex/"+strings.ToLower(baseCurrency)+"/"+strings.ToLower(quoteCurrency),
		values,
		nil,
		http.MethodGet,
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
func (f *FXMacroData) Health(ctx context.Context) (*ServiceStatusResponse, error) {
	response := new(ServiceStatusResponse)
	return response, f.sendHTTPPublicRequest(ctx, "health", nil, nil, http.MethodGet, response)
}

// Ping returns the public FXMacroData service liveness status.
func (f *FXMacroData) Ping(ctx context.Context) (*ServiceStatusResponse, error) {
	response := new(ServiceStatusResponse)
	return response, f.sendHTTPPublicRequest(ctx, "ping", nil, nil, http.MethodGet, response)
}

// DataCatalogue returns the available FXMacroData indicators for a currency.
func (f *FXMacroData) DataCatalogue(ctx context.Context, currency string) (*DataCatalogueResponse, error) {
	response := new(DataCatalogueResponse)
	return response, f.sendHTTPPublicRequest(ctx, "data_catalogue/"+strings.ToLower(currency), nil, nil, http.MethodGet, response)
}

// Announcements returns historical macro announcement rows.
func (f *FXMacroData) Announcements(ctx context.Context, currency, indicator string, values url.Values) (*AnnouncementResponse, error) {
	response := new(AnnouncementResponse)
	return response, f.sendCurrencyScopedRequest(ctx, currency, "announcements/"+strings.ToLower(currency)+"/"+indicator, values, response)
}

// LatestAnnouncements returns latest announcements for a currency.
func (f *FXMacroData) LatestAnnouncements(ctx context.Context, currency string, values url.Values) (*AnnouncementResponse, error) {
	response := new(AnnouncementResponse)
	return response, f.sendCurrencyScopedRequest(ctx, currency, "announcements/"+strings.ToLower(currency)+"/latest", values, response)
}

// AnnouncementChanges returns recently changed announcement rows.
func (f *FXMacroData) AnnouncementChanges(ctx context.Context, values url.Values) (*AnnouncementChangesResponse, error) {
	response := new(AnnouncementChangesResponse)
	return response, f.sendHTTPAuthRequest(ctx, "announcements/changes", values, nil, http.MethodGet, response)
}

// Calendar returns the release calendar for a currency.
func (f *FXMacroData) Calendar(ctx context.Context, currency string, values url.Values) (*CalendarResponse, error) {
	response := new(CalendarResponse)
	return response, f.sendCurrencyScopedRequest(ctx, currency, "calendar/"+strings.ToLower(currency), values, response)
}

// Predictions returns consensus/model prediction rows.
func (f *FXMacroData) Predictions(ctx context.Context, currency, indicator string, values url.Values) (*PredictionsResponse, error) {
	response := new(PredictionsResponse)
	return response, f.sendCurrencyScopedRequest(ctx, currency, "predictions/"+strings.ToLower(currency)+"/"+indicator, values, response)
}

// COT returns CFTC positioning data for a currency.
func (f *FXMacroData) COT(ctx context.Context, currency string, values url.Values) (*COTResponse, error) {
	response := new(COTResponse)
	return response, f.sendCurrencyScopedRequest(ctx, currency, "cot/"+strings.ToLower(currency), values, response)
}

// Commodity returns a commodity time series.
func (f *FXMacroData) Commodity(ctx context.Context, indicator string, values url.Values) (*CommodityResponse, error) {
	response := new(CommodityResponse)
	return response, f.sendHTTPAuthRequest(ctx, "commodities/"+indicator, values, nil, http.MethodGet, response)
}

// CommoditiesLatest returns latest commodity points.
func (f *FXMacroData) CommoditiesLatest(ctx context.Context, values url.Values) (*CommodityResponse, error) {
	response := new(CommodityResponse)
	return response, f.sendHTTPAuthRequest(ctx, "commodities/latest", values, nil, http.MethodGet, response)
}

// Curves returns yield curve data for a currency.
func (f *FXMacroData) Curves(ctx context.Context, currency string, values url.Values) (*CurveSnapshotResponse, error) {
	response := new(CurveSnapshotResponse)
	return response, f.sendHTTPAuthRequest(ctx, "curves/"+strings.ToLower(currency), values, nil, http.MethodGet, response)
}

// CurveProxies returns curve proxy data for a currency.
func (f *FXMacroData) CurveProxies(ctx context.Context, currency string, values url.Values) (*CurveProxyResponse, error) {
	response := new(CurveProxyResponse)
	return response, f.sendHTTPAuthRequest(ctx, "curve_proxies/"+strings.ToLower(currency), values, nil, http.MethodGet, response)
}

// ForwardCurves returns forward curve data for a currency.
func (f *FXMacroData) ForwardCurves(ctx context.Context, currency string, values url.Values) (*ForwardCurveResponse, error) {
	response := new(ForwardCurveResponse)
	return response, f.sendHTTPAuthRequest(ctx, "forward_curves/"+strings.ToLower(currency), values, nil, http.MethodGet, response)
}

// RateDifferentials returns rate differentials for a pair.
func (f *FXMacroData) RateDifferentials(ctx context.Context, baseCurrency, quoteCurrency string, values url.Values) (*RateDifferentialResponse, error) {
	response := new(RateDifferentialResponse)
	return response, f.sendHTTPAuthRequest(ctx, "rate_differentials/"+strings.ToLower(baseCurrency)+"/"+strings.ToLower(quoteCurrency), values, nil, http.MethodGet, response)
}

// ForwardDifferentials returns forward differentials for a pair.
func (f *FXMacroData) ForwardDifferentials(ctx context.Context, baseCurrency, quoteCurrency string, values url.Values) (*ForwardDifferentialResponse, error) {
	response := new(ForwardDifferentialResponse)
	return response, f.sendHTTPAuthRequest(ctx, "forward_differentials/"+strings.ToLower(baseCurrency)+"/"+strings.ToLower(quoteCurrency), values, nil, http.MethodGet, response)
}

// MarketSessions returns FX market-session state.
func (f *FXMacroData) MarketSessions(ctx context.Context, values url.Values) (*MarketSessionsResponse, error) {
	response := new(MarketSessionsResponse)
	return response, f.sendHTTPPublicRequest(ctx, "market_sessions", values, nil, http.MethodGet, response)
}

// RiskSentiment returns risk sentiment data.
func (f *FXMacroData) RiskSentiment(ctx context.Context, values url.Values) (*RiskSentimentResponse, error) {
	response := new(RiskSentimentResponse)
	return response, f.sendHTTPPublicRequest(ctx, "risk_sentiment", values, nil, http.MethodGet, response)
}

// News returns macro news for a currency.
func (f *FXMacroData) News(ctx context.Context, currency string, values url.Values) (*NewsResponse, error) {
	response := new(NewsResponse)
	return response, f.sendCurrencyScopedRequest(ctx, currency, "news/"+strings.ToLower(currency), values, response)
}

// PressReleases returns central-bank and official press releases.
func (f *FXMacroData) PressReleases(ctx context.Context, currency string, values url.Values) (*PressReleasesResponse, error) {
	response := new(PressReleasesResponse)
	return response, f.sendCurrencyScopedRequest(ctx, currency, "press-releases/"+strings.ToLower(currency), values, response)
}

// GraphQL executes an FXMacroData GraphQL request and honours ctx.
func (f *FXMacroData) GraphQL(ctx context.Context, payload string, result any) error {
	return f.sendHTTPAuthRequest(ctx, "graphql", nil, strings.NewReader(payload), http.MethodPost, result)
}

func (f *FXMacroData) sendCurrencyScopedRequest(ctx context.Context, currency, endpoint string, values url.Values, result any) error {
	if strings.EqualFold(currency, "USD") && f.APIKey == "" {
		return f.sendHTTPPublicRequest(ctx, endpoint, values, nil, http.MethodGet, result)
	}
	return f.sendHTTPAuthRequest(ctx, endpoint, values, nil, http.MethodGet, result)
}

// sendHTTPAuthRequest sends an API-key authenticated FXMacroData request.
func (f *FXMacroData) sendHTTPAuthRequest(ctx context.Context, endpoint string, values url.Values, body io.Reader, method string, result any) error {
	if f.APIKey == "" {
		return errAPIKeyNotConfigured
	}
	return f.send(ctx, endpoint, values, body, method, result, request.AuthenticatedRequest)
}

// sendHTTPPublicRequest sends an unauthenticated FXMacroData request.
func (f *FXMacroData) sendHTTPPublicRequest(ctx context.Context, endpoint string, values url.Values, body io.Reader, method string, result any) error {
	return f.send(ctx, endpoint, values, body, method, result, request.UnauthenticatedRequest)
}

func (f *FXMacroData) send(ctx context.Context, endpoint string, values url.Values, body io.Reader, method string, result any, auth request.AuthType) error {
	query := make(url.Values, len(values))
	for k, v := range values {
		query[k] = append([]string(nil), v...)
	}

	baseURL := strings.TrimRight(f.APIURL, "/") + "/"
	path := common.EncodeURLValues(baseURL+strings.TrimLeft(endpoint, "/"), query)
	headers := map[string]string{
		"Accept": "application/json",
	}
	if auth == request.AuthenticatedRequest {
		headers["X-API-Key"] = f.APIKey
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
	}, auth)
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
