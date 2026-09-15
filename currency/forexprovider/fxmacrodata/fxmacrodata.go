package fxmacrodata

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	errUnsupportedCurrency = errors.New("currency not supported by FXMacroData")
	errEmptyCurrency       = errors.New("currency symbol must not be empty")
	errDuplicateCurrency   = errors.New("duplicate currency symbol")
	errNoTargetCurrencies  = errors.New("at least one target currency is required")
	errNoRateAvailable     = errors.New("no FXMacroData rate available")
	errAPIKeyNotConfigured = errors.New("FXMacroData API key is required for this endpoint")
)

const (
	// requestRateLimit is the documented per-minute allowance for Trial and
	// Individual keys, which is also what anonymous USD access is held to.
	requestRateLimit = 300
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
		// Keep requests within the documented per-minute allowance. A burst of
		// one is deliberate: the request limiter does not permit outbound bursts,
		// so this spreads the allowance evenly across the minute.
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

	return f.getLatestForexRates(context.TODO(), baseCurrency, targetSymbols)
}

// getLatestForexRates fetches one rate per target. A pair whose latest row
// carries no usable value is skipped and logged rather than failing the batch,
// so one empty pair does not discard every other rate; the error is returned
// only when no pair produced a rate. Any other error, such as a transport or
// authentication failure, still fails the whole call.
func (f *FXMacroData) getLatestForexRates(ctx context.Context, baseCurrency string, targetSymbols []string) (map[string]float64, error) {
	if len(targetSymbols) == 0 {
		return nil, errNoTargetCurrencies
	}
	standardisedRates := make(map[string]float64, len(targetSymbols))

	var skipped error
	for _, quote := range targetSymbols {
		rate, err := f.GetLatestForexRate(ctx, baseCurrency, quote)
		if err != nil {
			if !errors.Is(err, errNoRateAvailable) {
				return nil, err
			}
			log.Warnf(log.Currency, "%s: skipping %s%s: %v", f.Name, baseCurrency, quote, err)
			skipped = errors.Join(skipped, err)
			continue
		}
		standardisedRates[baseCurrency+quote] = rate
	}
	if len(standardisedRates) == 0 {
		return nil, skipped
	}
	return standardisedRates, nil
}

// Forex returns historical FX observations for a pair.
func (f *FXMacroData) Forex(ctx context.Context, baseCurrency, quoteCurrency string, values url.Values) (*ForexResponse, error) {
	response := new(ForexResponse)
	return response, f.sendHTTPAuthRequest(ctx,
		"forex/"+strings.ToLower(baseCurrency)+"/"+strings.ToLower(quoteCurrency),
		values,
		response,
	)
}

// GetLatestForexRate returns the latest available FXMacroData rate for a pair.
func (f *FXMacroData) GetLatestForexRate(ctx context.Context, baseCurrency, quoteCurrency string) (float64, error) {
	response, err := f.Forex(ctx, baseCurrency, quoteCurrency, url.Values{"limit": {"1"}})
	if err != nil {
		return 0, err
	}
	if len(response.Data) == 0 {
		return 0, fmt.Errorf("%w for %s/%s", errNoRateAvailable, baseCurrency, quoteCurrency)
	}
	// val is documented as anyOf[number, null] and date is the only required
	// field, so a row can legitimately carry no value. It decodes to 0 here,
	// and ConversionRates.Update stores 1/rate, which turns a missing value
	// into +Inf on every pair that touches this currency. Guard the value, not
	// just the row count.
	if response.Data[0].Val <= 0 {
		return 0, fmt.Errorf("%w for %s/%s: %v", errNoRateAvailable, baseCurrency, quoteCurrency, response.Data[0].Val)
	}
	return response.Data[0].Val, nil
}

// Ping returns the public FXMacroData service liveness status.
func (f *FXMacroData) Ping(ctx context.Context) (*ServiceStatusResponse, error) {
	response := new(ServiceStatusResponse)
	return response, f.sendHTTPPublicRequest(ctx, "ping", nil, response)
}

// Health returns the public FXMacroData service health status.
func (f *FXMacroData) Health(ctx context.Context) (*ServiceStatusResponse, error) {
	response := new(ServiceStatusResponse)
	return response, f.sendHTTPPublicRequest(ctx, "health", nil, response)
}

// DataCatalogue returns the available FXMacroData indicators for a currency.
func (f *FXMacroData) DataCatalogue(ctx context.Context, currency string) (*DataCatalogueResponse, error) {
	response := new(DataCatalogueResponse)
	return response, f.sendHTTPPublicRequest(ctx, "data_catalogue/"+strings.ToLower(currency), nil, response)
}

// Announcements returns historical macro announcement rows.
func (f *FXMacroData) Announcements(ctx context.Context, currency, indicator string, values url.Values) (*AnnouncementResponse, error) {
	response := new(AnnouncementResponse)
	return response, f.sendCurrencyScopedRequest(ctx, currency, "announcements/"+strings.ToLower(currency)+"/"+indicator, values, response)
}

// LatestAnnouncements returns latest announcements for a currency.
func (f *FXMacroData) LatestAnnouncements(ctx context.Context, currency string, values url.Values) (*LatestAnnouncementsResponse, error) {
	response := new(LatestAnnouncementsResponse)
	return response, f.sendCurrencyScopedRequest(ctx, currency, "announcements/"+strings.ToLower(currency)+"/latest", values, response)
}

// AnnouncementChanges returns recently changed announcement rows. USD events
// are served without a key and the server scopes an anonymous request to USD
// itself, so a keyless caller is sent through as a public USD request.
func (f *FXMacroData) AnnouncementChanges(ctx context.Context, values url.Values) (*AnnouncementChangesResponse, error) {
	response := new(AnnouncementChangesResponse)
	return response, f.sendCurrencyScopedRequest(ctx, "USD", "announcements/changes", values, response)
}

// Calendar returns the release calendar for a currency.
func (f *FXMacroData) Calendar(ctx context.Context, currency string, values url.Values) (*CalendarResponse, error) {
	response := new(CalendarResponse)
	return response, f.sendCurrencyScopedRequest(ctx, currency, "calendar/"+strings.ToLower(currency), values, response)
}

// Predictions returns consensus/model prediction rows. Forecast values require
// an API key on every currency, USD included.
func (f *FXMacroData) Predictions(ctx context.Context, currency, indicator string, values url.Values) (*PredictionsResponse, error) {
	response := new(PredictionsResponse)
	return response, f.sendHTTPAuthRequest(ctx, "predictions/"+strings.ToLower(currency)+"/"+indicator, values, response)
}

// COT returns CFTC positioning data for a currency.
func (f *FXMacroData) COT(ctx context.Context, currency string, values url.Values) (*COTResponse, error) {
	response := new(COTResponse)
	return response, f.sendCurrencyScopedRequest(ctx, currency, "cot/"+strings.ToLower(currency), values, response)
}

// Commodity returns a commodity time series.
func (f *FXMacroData) Commodity(ctx context.Context, indicator string, values url.Values) (*CommodityResponse, error) {
	response := new(CommodityResponse)
	return response, f.sendHTTPAuthRequest(ctx, "commodities/"+indicator, values, response)
}

// CommoditiesLatest returns latest commodity points.
func (f *FXMacroData) CommoditiesLatest(ctx context.Context, values url.Values) (*CommoditiesLatestResponse, error) {
	response := new(CommoditiesLatestResponse)
	return response, f.sendHTTPAuthRequest(ctx, "commodities/latest", values, response)
}

// Curves returns yield curve data for a currency.
func (f *FXMacroData) Curves(ctx context.Context, currency string, values url.Values) (*CurveAnalyticsResponse, error) {
	response := new(CurveAnalyticsResponse)
	return response, f.sendHTTPAuthRequest(ctx, "curves/"+strings.ToLower(currency), values, response)
}

// Factor returns one precomputed currency factor. USD factors are public.
func (f *FXMacroData) Factor(ctx context.Context, currency, factor string, values url.Values) (*FactorResponse, error) {
	response := new(FactorResponse)
	return response, f.sendCurrencyScopedRequest(ctx, currency, "factors/"+strings.ToLower(currency)+"/"+factor, values, response)
}

// RateDifferentials returns rate differentials for a pair.
func (f *FXMacroData) RateDifferentials(ctx context.Context, baseCurrency, quoteCurrency string, values url.Values) (*RateDifferentialResponse, error) {
	response := new(RateDifferentialResponse)
	return response, f.sendHTTPAuthRequest(ctx, "rate_differentials/"+strings.ToLower(baseCurrency)+"/"+strings.ToLower(quoteCurrency), values, response)
}

// IntradayReferenceRates returns subscriber intraday reference-rate observations.
func (f *FXMacroData) IntradayReferenceRates(ctx context.Context, baseCurrency, quoteCurrency string, values url.Values) (*FXIntradayReferenceRatesResponse, error) {
	response := new(FXIntradayReferenceRatesResponse)
	return response, f.sendHTTPAuthRequest(ctx, "fx/intraday-reference-rates/"+strings.ToLower(baseCurrency)+"/"+strings.ToLower(quoteCurrency), values, response)
}

// FXSources returns public metadata for supported FX reference-rate sources.
func (f *FXMacroData) FXSources(ctx context.Context, values url.Values) (*FXSourcesResponse, error) {
	response := new(FXSourcesResponse)
	return response, f.sendHTTPPublicRequest(ctx, "fx/sources", values, response)
}

// FXSourceUniverse returns the pair universe available from public FX sources.
func (f *FXMacroData) FXSourceUniverse(ctx context.Context, values url.Values) (*FXSourceUniverseResponse, error) {
	response := new(FXSourceUniverseResponse)
	return response, f.sendHTTPPublicRequest(ctx, "fx/source-universe", values, response)
}

// MarketSessions returns FX market-session state.
func (f *FXMacroData) MarketSessions(ctx context.Context, values url.Values) (*MarketSessionsResponse, error) {
	response := new(MarketSessionsResponse)
	return response, f.sendHTTPPublicRequest(ctx, "market_sessions", values, response)
}

// RiskSentiment returns risk sentiment data.
func (f *FXMacroData) RiskSentiment(ctx context.Context, values url.Values) (*RiskSentimentResponse, error) {
	response := new(RiskSentimentResponse)
	return response, f.sendHTTPPublicRequest(ctx, "risk_sentiment", values, response)
}

// PressReleases returns central-bank and official press releases.
func (f *FXMacroData) PressReleases(ctx context.Context, currency string, values url.Values) (*PressReleasesResponse, error) {
	response := new(PressReleasesResponse)
	return response, f.sendCurrencyScopedRequest(ctx, currency, "press-releases/"+strings.ToLower(currency), values, response)
}

func (f *FXMacroData) sendCurrencyScopedRequest(ctx context.Context, currency, endpoint string, values url.Values, result any) error {
	if strings.EqualFold(currency, "USD") && f.APIKey == "" {
		return f.sendHTTPPublicRequest(ctx, endpoint, values, result)
	}
	return f.sendHTTPAuthRequest(ctx, endpoint, values, result)
}

// sendHTTPAuthRequest sends an API-key authenticated FXMacroData request.
func (f *FXMacroData) sendHTTPAuthRequest(ctx context.Context, endpoint string, values url.Values, result any) error {
	if f.APIKey == "" {
		return errAPIKeyNotConfigured
	}
	return f.send(ctx, endpoint, values, result, request.AuthenticatedRequest)
}

// sendHTTPPublicRequest sends an unauthenticated FXMacroData request.
func (f *FXMacroData) sendHTTPPublicRequest(ctx context.Context, endpoint string, values url.Values, result any) error {
	return f.send(ctx, endpoint, values, result, request.UnauthenticatedRequest)
}

func (f *FXMacroData) send(ctx context.Context, endpoint string, values url.Values, result any, auth request.AuthType) error {
	baseURL := strings.TrimRight(f.APIURL, "/") + "/"
	path := common.EncodeURLValues(baseURL+strings.TrimLeft(endpoint, "/"), values)
	headers := map[string]string{
		"Accept": "application/json",
	}
	if auth == request.AuthenticatedRequest {
		headers["X-API-Key"] = f.APIKey
	}
	item := &request.Item{
		Method:  http.MethodGet,
		Path:    path,
		Headers: headers,
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
