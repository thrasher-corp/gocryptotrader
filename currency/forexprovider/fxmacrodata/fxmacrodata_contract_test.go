package fxmacrodata

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
)

func newContractProvider(t *testing.T, path, fixture string, authenticated bool) (provider *FXMacroData, closeServer func()) {
	t.Helper()
	return newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, path, r.URL.Path, "request path should match the documented endpoint")
		if authenticated {
			assert.Equal(t, "placeholder", r.Header.Get("X-API-Key"), "authenticated request should include the configured key")
		} else {
			assert.Empty(t, r.Header.Get("X-API-Key"), "public request should not include an API key")
		}
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(fixture))
		assert.NoError(t, err, "fixture response should write successfully")
	}))
}

func TestForex(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/forex/usd/aud", `{
		"base":"USD","quote":"AUD","source":"official_reference_rates",
		"provenance":{"publisher":"official"},"data_quality":{"row_count":1},
		"start_date":"2026-08-01","end_date":"2026-08-01",
		"pagination":{"returned_count":1},
		"data":[{"date":"2026-08-01","val":1.53,"announcement_datetime":1785542400,"rsi_14":55.2}]
	}`, true)
	defer closeServer()

	response, err := provider.Forex(t.Context(), "USD", "AUD", url.Values{"limit": {"1"}})
	require.NoError(t, err, "Forex must decode a documented response")
	require.Len(t, response.Data, 1, "Forex must decode one data point")
	require.NotNil(t, response.Data[0].Val, "Forex value must be present")
	assert.Equal(t, 1.53, response.Data[0].Val, "Forex should decode the rate")
	require.NotNil(t, response.Data[0].RSI14, "Forex RSI must be present")
	assert.Equal(t, 55.2, response.Data[0].RSI14, "Forex should decode technical fields")
}

func TestDataCatalogue(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/data_catalogue/usd", `{
		"inflation":{"name":"Inflation (CPI)","unit":"%YoY","frequency":"Monthly",
		"has_official_forecast":true,"source":"BLS",
		"coverage":{"available":true,"requires_api_key":false,"row_count":42},
		"supported_options":{"seasonality":["nsa","sa"]}}
	}`, false)
	defer closeServer()

	response, err := provider.DataCatalogue(t.Context(), "USD")
	require.NoError(t, err, "DataCatalogue must decode a documented response")
	item, ok := (*response)["inflation"]
	require.True(t, ok, "DataCatalogue must contain the fixture indicator")
	assert.Equal(t, "Inflation (CPI)", item.Name, "DataCatalogue should decode indicator metadata")
	require.NotNil(t, item.Coverage, "DataCatalogue coverage must be present")
	assert.Equal(t, 42, item.Coverage.RowCount, "DataCatalogue should decode coverage")
}

func TestAnnouncements(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/announcements/usd/inflation", `{
		"currency":"USD","indicator":"inflation","name":"Inflation (CPI)",
		"source_series_name":"Consumer Price Index","policy_role":"inflation_target",
		"has_official_forecast":true,"start_date":"2026-07-31","end_date":"2026-07-31",
		"data_quality":{"row_count":1,"reason":"current"},
		"pagination":{"limit":1,"returned_count":1},
		"data":[{"announcement_id":"usd_inflation_2026-07-31","date":"2026-07-31","val":2.7,
		"previous_value":2.6,"announcement_datetime":1786105800,"pct_change_yoy":2.7,
		"revisions":[{"epoch":1786100000,"val":2.6}],"remap_applied":false}]
	}`, true)
	defer closeServer()

	response, err := provider.Announcements(t.Context(), "USD", "inflation", url.Values{"limit": {"1"}})
	require.NoError(t, err, "Announcements must decode a documented response")
	require.Len(t, response.Data, 1, "Announcements must decode one data point")
	require.NotNil(t, response.Data[0].PreviousValue, "Announcements previous value must be present")
	assert.Equal(t, 2.6, response.Data[0].PreviousValue, "Announcements should preserve previous values")
	assert.Equal(t, "2026-07-31", response.Data[0].Date.String(), "Announcements should decode ISO dates")
	assert.Equal(t, int64(1786105800), response.Data[0].AnnouncementDatetime.Time().Unix(),
		"Announcements should decode Unix timestamps")
	assert.Equal(t, "current", response.DataQuality.Reason, "Announcements should decode quality context")
}

func TestLatestAnnouncements(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/announcements/usd/latest", `{
		"currency":"USD","source":"FXMacroData","as_of":"2026-08-12","count":1,
		"data":[{"indicator":"inflation","name":"Inflation (CPI)","source":"BLS",
		"unit":"%YoY","frequency":"Monthly","has_official_forecast":true,
		"latest":{"date":"2026-07-31","val":2.7,"announcement_datetime":1786105800}}]
	}`, true)
	defer closeServer()

	response, err := provider.LatestAnnouncements(t.Context(), "USD", nil)
	require.NoError(t, err, "LatestAnnouncements must decode a documented response")
	require.Len(t, response.Data, 1, "LatestAnnouncements must decode one indicator")
	assert.Equal(t, "inflation", response.Data[0].Indicator, "LatestAnnouncements should decode the indicator")
	require.NotNil(t, response.Data[0].Latest.Val, "LatestAnnouncements value must be present")
	assert.Equal(t, 2.7, response.Data[0].Latest.Val, "LatestAnnouncements should decode the latest value")
}

func TestAnnouncementChanges(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/announcements/changes", `{
		"count":1,"next_cursor":"cursor-2","has_more":true,"retention_seconds":86400,
		"scope":{"currency":"USD"},"data":[{"event_id":"event-1","currency":"USD",
		"indicator":"inflation","records_written":2,"timestamp":1786105800,
		"latest_announcement":{"date":"2026-07-31","val":2.7}}]
	}`, true)
	defer closeServer()

	response, err := provider.AnnouncementChanges(t.Context(), nil)
	require.NoError(t, err, "AnnouncementChanges must decode a documented response")
	require.Len(t, response.Data, 1, "AnnouncementChanges must decode one event")
	assert.Equal(t, "event-1", response.Data[0].EventID, "AnnouncementChanges should decode event identifiers")
	assert.Equal(t, 2, response.Data[0].RecordsWritten, "AnnouncementChanges should decode write counts")
}

func TestCalendar(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/calendar/usd", `{
		"currency":"USD","timezone":"America/New_York","requested_timezone":"UTC",
		"data_quality":{"row_count":1},"data":[{"announcement_datetime":1786105800,
		"release":"inflation","announcement_datetime_utc":"2026-08-07T12:30:00Z",
		"release_date_confirmed":true,"release_time_assumed":false,"source":"BLS",
		"source_url":"https://www.bls.gov/schedule/","event_importance":"high","market_tier":1}]
	}`, true)
	defer closeServer()

	response, err := provider.Calendar(t.Context(), "USD", nil)
	require.NoError(t, err, "Calendar must decode a documented response")
	require.Len(t, response.Data, 1, "Calendar must decode one release")
	assert.True(t, response.Data[0].ReleaseDateConfirmed, "Calendar should preserve confirmed release state")
	assert.False(t, response.Data[0].ReleaseTimeAssumed, "Calendar should preserve assumed-time state")
}

func TestPredictions(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/predictions/usd/inflation", `{
		"currency":"USD","indicator":"inflation","pre_release_only":true,"count":1,
		"prediction_count":1,"has_more":false,"data_quality":{"row_count":1},
		"data":[{"announcement_id":"usd_inflation_2026-08-31","currency":"USD",
		"indicator":"inflation","date":"2026-08-31","announcement_datetime":1788134400,
		"predictions":[{"predicted_value":2.5,"prediction_type":"market_consensus",
		"prediction_source":"survey","generated_at":1788000000,"is_pre_release":true,"confidence":0.8}]}]
	}`, true)
	defer closeServer()

	response, err := provider.Predictions(t.Context(), "USD", "inflation", nil)
	require.NoError(t, err, "Predictions must decode a documented response")
	require.Len(t, response.Data, 1, "Predictions must decode one announcement group")
	require.Len(t, response.Data[0].Predictions, 1, "Predictions must decode one source")
	assert.True(t, response.Data[0].Predictions[0].IsPreRelease, "Predictions should preserve pre-release state")
	assert.Equal(t, 2.5, response.Data[0].Predictions[0].PredictedValue, "Predictions should decode forecast values")
}

func TestCOT(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/cot/jpy", `{
		"currency":"JPY","instrument":"JAPANESE YEN","source":"CFTC","source_url":"https://www.cftc.gov/",
		"provenance":{"publisher":"CFTC","publisher_url":"https://www.cftc.gov/","storage":"Firestore",
		"served_by":"/v1/cot/{currency}","timestamp_field":"announcement_datetime","value_field":"noncommercial_net"},
		"fx_overlay":{"pair":"USD/JPY"},"start_date":"2026-08-01","end_date":"2026-08-08",
		"last_sync_status":"ok","expected_next_release":"2026-08-14","data_quality":{"row_count":1},
		"pagination":{"limit":1,"returned_count":1,"total_count":1,"has_more":false},
		"data":[{"date":"2026-08-04","announcement_datetime":1786132800,"open_interest":250000,
		"noncommercial_long":80000,"noncommercial_short":120000,"noncommercial_net":-40000,
		"commercial_long":140000,"commercial_short":90000,"release_date_confirmed":true,
		"release_source":"CFTC","release_source_url":"https://www.cftc.gov/"}]
	}`, true)
	defer closeServer()

	response, err := provider.COT(t.Context(), "JPY", nil)
	require.NoError(t, err, "COT must decode a documented response")
	require.Len(t, response.Data, 1, "COT must decode one positioning row")
	assert.Equal(t, int64(-40000), response.Data[0].NonCommercialNet, "COT should preserve net positioning")
	assert.True(t, response.Data[0].ReleaseDateConfirmed, "COT should preserve release confirmation")
}

func TestCommodity(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/commodities/brent", `{
		"currency":"USD","indicator":"brent","source":"EIA","source_url":"https://www.eia.gov/",
		"has_official_forecast":false,"latest_available_date":"2026-08-11","data_quality":{"row_count":1},
		"start_date":"2026-08-11","end_date":"2026-08-11","pagination":{"returned_count":1},
		"data":[{"date":"2026-08-11","val":68.4,"announcement_datetime":1786406400,"pct_change":1.2}]
	}`, true)
	defer closeServer()

	response, err := provider.Commodity(t.Context(), "brent", nil)
	require.NoError(t, err, "Commodity must decode a documented response")
	require.Len(t, response.Data, 1, "Commodity must decode one data point")
	assert.Equal(t, 68.4, response.Data[0].Val, "Commodity should decode values")
	assert.Equal(t, "2026-08-11", response.LatestAvailableDate.String(), "Commodity should decode availability metadata")
}

func TestCommoditiesLatest(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/commodities/latest", `{
		"brent":{"date":"2026-08-11","val":68.4}
	}`, true)
	defer closeServer()

	response, err := provider.CommoditiesLatest(t.Context(), nil)
	require.NoError(t, err, "CommoditiesLatest must decode a documented dynamic response")
	brent, ok := (*response)["brent"].(map[string]any)
	require.True(t, ok, "CommoditiesLatest must decode indicator metadata")
	assert.Equal(t, 68.4, brent["val"], "CommoditiesLatest should decode values")
}

func TestCurves(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/curves/usd", `{
		"currency":"USD","curve_family":"government","view":"nodes","metric":"yield",
		"requested_date":"2026-08-12","as_of":"2026-08-12","node_count":1,"sources":["Treasury"],
		"official_forward_source_support":{"supported":false},"data_quality":{"row_count":1},
		"data":[{"indicator":"bond_yield_10y","maturity":"10Y","val":4.21}]
	}`, true)
	defer closeServer()

	response, err := provider.Curves(t.Context(), "USD", nil)
	require.NoError(t, err, "Curves must decode a documented response")
	require.Len(t, response.Data, 1, "Curves must decode one selected-view row")
	assert.Equal(t, "10Y", response.Data[0]["maturity"], "Curves should retain view-specific fields")
	assert.Equal(t, 1, response.NodeCount, "Curves should decode node counts")
}

func TestFactor(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/factors/usd/inflation_pressure", `{
		"currency":"USD","factor":"inflation_pressure","methodology":"standardised",
		"as_of":"2026-08-12","score":0.7,"label":"elevated","start_date":"2026-08-12",
		"end_date":"2026-08-12","data_quality":{"row_count":1},"pagination":{"returned_count":1},
		"data":[{"date":"2026-08-12","val":0.7,"score":0.7,"point_in_time_safe":true,
		"components":{"inflation":0.8},"source_endpoints":["announcements/usd/inflation"]}]
	}`, true)
	defer closeServer()

	response, err := provider.Factor(t.Context(), "USD", "inflation_pressure", nil)
	require.NoError(t, err, "Factor must decode a documented response")
	require.Len(t, response.Data, 1, "Factor must decode one data point")
	assert.Equal(t, 0.7, response.Data[0].Score, "Factor should decode scores")
	assert.True(t, response.Data[0].PointInTimeSafe, "Factor should preserve point-in-time state")
}

func TestRateDifferentials(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/rate_differentials/eur/usd", `{
		"base":"EUR","quote":"USD","rate_type":"spot","measure_requested":"policy_rate",
		"measure_used":"policy_rate","base_indicator":"policy_rate","quote_indicator":"policy_rate",
		"start_date":"2026-08-01","end_date":"2026-08-12","matched_points":1,"unit":"percentage_points",
		"latest_spread":-1.75,"latest_spread_bps":-175,"sources":{"base":"ECB","quote":"Federal Reserve"},
		"data_quality":{"row_count":1},"pagination":{"returned_count":1},
		"data":[{"date":"2026-08-12","base_val":2.0,"quote_val":3.75,"spread":-1.75,
		"spread_bps":-175,"base_announcement_datetime":1786000000}]
	}`, true)
	defer closeServer()

	response, err := provider.RateDifferentials(t.Context(), "EUR", "USD", nil)
	require.NoError(t, err, "RateDifferentials must decode a documented response")
	require.Len(t, response.Data, 1, "RateDifferentials must decode one data point")
	assert.Equal(t, -175.0, response.Data[0].SpreadBPS, "RateDifferentials should decode basis-point spreads")
	assert.Equal(t, "spot", response.RateType, "RateDifferentials should decode rate type")
}

func TestIntradayReferenceRates(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/fx/intraday-reference-rates/usd/aud", `{
		"pair":"USD/AUD","start_time":"2026-08-12T00:00:00Z","end_time":"2026-08-12T23:59:59Z",
		"data":[{"timestamp":"2026-08-12T16:00:00Z","price":1.53,"reference_date":"2026-08-12",
		"timestamp_type":"official_fixing","source":{"id":"official"},"source_pair":"AUD/USD",
		"derivation_method":"inverse"}]
	}`, true)
	defer closeServer()

	response, err := provider.IntradayReferenceRates(t.Context(), "USD", "AUD", nil)
	require.NoError(t, err, "IntradayReferenceRates must decode a documented response")
	require.Len(t, response.Data, 1, "IntradayReferenceRates must decode one data point")
	assert.Equal(t, 1.53, response.Data[0].Price, "IntradayReferenceRates should decode prices")
	assert.Equal(t, "inverse", response.Data[0].DerivationMethod, "IntradayReferenceRates should decode derivation metadata")
}

func TestFXSources(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/fx/sources", `{
		"source_policy":{"request_time_upstream_fetches":false},
		"sources":[{"id":"official-source","name":"Official Source","is_official":true,"native_pairs":["USD/AUD"]}]
	}`, false)
	defer closeServer()

	response, err := provider.FXSources(t.Context(), nil)
	require.NoError(t, err, "FXSources must decode a documented response")
	require.Len(t, response.Sources, 1, "FXSources must decode one source")
	assert.Equal(t, "official-source", response.Sources[0].ID, "FXSources should decode source identifiers")
	assert.True(t, response.Sources[0].IsOfficial, "FXSources should decode the official flag")
	assert.Equal(t, []string{"USD/AUD"}, response.Sources[0].NativePairs, "FXSources should decode native pairs")
	assert.False(t, response.SourcePolicy.RequestTimeUpstreamFetches, "FXSources should decode the source policy")
}

func TestFXSourceUniverse(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/fx/source-universe", `{
		"source_policy":{"request_time_upstream_fetches":false},"currency":"USD",
		"data":[{"source":{"id":"official-source"},"pair_count":1,"pairs":[{"pair":"USD/AUD","availability":"native"}]}]
	}`, false)
	defer closeServer()

	response, err := provider.FXSourceUniverse(t.Context(), nil)
	require.NoError(t, err, "FXSourceUniverse must decode a documented response")
	require.Len(t, response.Data, 1, "FXSourceUniverse must decode one source group")
	assert.Equal(t, "USD", response.Currency, "FXSourceUniverse should decode currency filters")
	assert.Equal(t, float64(1), response.Data[0]["pair_count"], "FXSourceUniverse should decode pair counts")
}

func TestMarketSessions(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/market_sessions", `{
		"now_utc":"2026-08-13T10:00:00Z","now_unix":1786615200,"is_market_day":true,
		"sessions":[{"name":"London","currencies":["GBP","EUR"],"is_open":true}],
		"overlaps":[{"name":"London / New York","sessions":["London","New York"],"duration_hours":4}]
	}`, false)
	defer closeServer()

	response, err := provider.MarketSessions(t.Context(), nil)
	require.NoError(t, err, "MarketSessions must decode a documented response")
	require.Len(t, response.Sessions, 1, "MarketSessions must decode one session")
	assert.True(t, response.Sessions[0].IsOpen, "MarketSessions should decode open state")
	assert.Equal(t, 4.0, response.Overlaps[0].DurationHours, "MarketSessions should decode overlap duration")
}

func TestRiskSentiment(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/risk_sentiment", `{
		"start_date":"2026-08-01","end_date":"2026-08-12","latest_available_date":"2026-08-12",
		"last_updated":"2026-08-13T00:00:00Z","data_quality":{"row_count":1},
		"component_metadata":{"aliases":{"score":"alias for val"}},
		"pagination":{"limit":1,"returned_count":1,"total_count":1,"has_more":false},
		"data":[{"components":{"ofr_fsi":0.5},"val":0.5,"date":"2026-08-12","regime":"risk_on",
		"component_coverage":{"ofr_fsi":true},"stored_component_count":1}]
	}`, false)
	defer closeServer()

	response, err := provider.RiskSentiment(t.Context(), nil)
	require.NoError(t, err, "RiskSentiment must decode a documented response")
	require.Len(t, response.Data, 1, "RiskSentiment must decode one data point")
	assert.Equal(t, 0.5, response.Data[0].Components["ofr_fsi"], "RiskSentiment should decode components")
	assert.Equal(t, "alias for val", response.ComponentMetadata.Aliases["score"], "RiskSentiment should decode aliases")
}

func TestPressReleases(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/press-releases/usd", `{
		"currency":"USD","source":"Federal Reserve","source_url":"https://www.federalreserve.gov",
		"limit":1,"offset":0,"count":1,"pagination":{"limit":1,"returned_count":1,"total_count":1,"has_more":false},
		"data":[{"title":"Policy statement","url":"https://example.test/release","date":"2026-07-20",
		"summary":"Held rates","sentiment":0,"topics":["policy"],"category":"monetary_policy","relevance":0.9,
		"rate_path":{"score":0,"label":"Neutral","bias_action":"hold","confidence":"low","raw_score":0,"matches":""}}]
	}`, true)
	defer closeServer()

	response, err := provider.PressReleases(t.Context(), "USD", nil)
	require.NoError(t, err, "PressReleases must decode a documented response")
	require.Len(t, response.Data, 1, "PressReleases must decode one release")
	assert.Equal(t, "Policy statement", response.Data[0].Title, "PressReleases should decode titles")
	assert.Equal(t, "hold", response.Data[0].RatePath.BiasAction, "PressReleases should decode rate-path context")
}

func TestAuthenticatedEndpointsLive(t *testing.T) {
	if !authTestsEnabled() {
		t.Skip("set testAuth = true or GCT_RUN_FXMACRODATA_AUTH_TESTS=true to run the authenticated FXMacroData smoke test")
	}

	apiKey := liveTestAPIKey()
	if apiKey == "" {
		t.Skip("set testAPIKey, FXMACRODATA_API_KEY or FXMD_API_KEY to run the authenticated smoke test")
	}

	provider := new(FXMacroData)
	require.NoError(t, provider.Setup(base.Settings{Name: "FXMacroData", APIKey: apiKey}),
		"Setup must configure the authenticated endpoint client")
	rate, err := provider.GetLatestForexRate(t.Context(), "USD", "AUD")
	require.NoError(t, err, "GetLatestForexRate must not error in the opt-in authenticated smoke test")
	assert.Positive(t, rate, "GetLatestForexRate should return a positive rate")
}
