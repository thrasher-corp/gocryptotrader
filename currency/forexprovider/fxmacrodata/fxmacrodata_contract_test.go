package fxmacrodata

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/types"
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

// newKeylessContractProvider builds a provider with no API key configured, which
// is how a caller uses the public USD endpoints. Every other test here supplies
// one, so without this the headline keyless path went untested.
func newKeylessContractProvider(t *testing.T, path, fixture string) (provider *FXMacroData, closeServer func()) {
	t.Helper()
	provider, closeServer = newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, path, r.URL.Path, "request path should match the documented endpoint")
		assert.Empty(t, r.Header.Get("X-API-Key"), "keyless request should not send an API key header")
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(fixture))
		assert.NoError(t, err, "fixture response should write successfully")
	}))
	provider.APIKey = ""
	return provider, closeServer
}

// utcTime parses an RFC 3339 fixture timestamp the same way time.Time decodes it.
func utcTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	require.NoError(t, err, "fixture timestamp must be RFC 3339")
	return parsed
}

func calendarDay(year int, month time.Month, day int) Date {
	return Date(time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
}

func unixSeconds(seconds int64) UnixSeconds {
	return UnixSeconds(time.Unix(seconds, 0).UTC())
}

func TestPublicEndpointsWorkWithoutAnAPIKey(t *testing.T) {
	provider, closeServer := newKeylessContractProvider(t, "/api/v1/data_catalogue/usd", `{
		"inflation":{"name":"Inflation (CPI)","unit":"%YoY","frequency":"Monthly",
		"coverage":{"available":true,"requires_api_key":false,"row_count":42,"latest_release_date":"2026-08-12"},
		"aliases":["CPI"],"source_url":"https://www.bls.gov/","source_url_scope":"dataset"}
	}`)
	defer closeServer()

	response, err := provider.DataCatalogue(t.Context(), usd)
	require.NoError(t, err, "a public endpoint must work with no API key configured")
	exp := &DataCatalogueResponse{
		inflation: {
			Name:      "Inflation (CPI)",
			Unit:      "%YoY",
			Frequency: "Monthly",
			Coverage: CatalogueCoverage{
				Available:         true,
				RowCount:          42,
				LatestReleaseDate: calendarDay(2026, time.August, 12),
			},
			Aliases:        []string{"CPI"},
			SourceURL:      "https://www.bls.gov/",
			SourceURLScope: "dataset",
		},
	}
	assert.Equal(t, exp, response, "keyless DataCatalogue should decode every fixture field")
}

func TestCurrencyScopedEndpointsWithoutAnAPIKey(t *testing.T) {
	t.Run("calendar", func(t *testing.T) {
		provider, closeServer := newKeylessContractProvider(t, "/api/v1/calendar/usd", `{
			"currency":"USD","data":[{"announcement_datetime":1786105800,"release":"inflation"}]
		}`)
		defer closeServer()

		response, err := provider.Calendar(t.Context(), usd, nil)
		require.NoError(t, err, "a USD calendar request must work with no API key configured")
		exp := &CalendarResponse{
			Currency: usd,
			Data:     []CalendarReleaseRow{{AnnouncementDatetime: unixSeconds(1786105800), Release: inflation}},
		}
		assert.Equal(t, exp, response, "keyless Calendar should decode every fixture field")

		_, err = provider.Calendar(t.Context(), "AUD", nil)
		assert.ErrorIs(t, err, errAPIKeyNotConfigured, "a non-USD Calendar request should require an API key")
	})

	t.Run("factor", func(t *testing.T) {
		provider, closeServer := newKeylessContractProvider(t, "/api/v1/factors/usd/monetary_stance", `{
			"currency":"USD","factor":"monetary_stance","methodology":"monetary_stance","label":"neutral",
			"data":[{"date":"2026-09-01","val":0.155,"label":"neutral"}]
		}`)
		defer closeServer()

		response, err := provider.Factor(t.Context(), usd, "monetary_stance", nil)
		require.NoError(t, err, "a USD factor request must work with no API key configured")
		exp := &FactorResponse{
			Currency:    usd,
			Factor:      "monetary_stance",
			Methodology: "monetary_stance",
			Label:       "neutral",
			Data:        []FactorDataPoint{{Date: calendarDay(2026, time.September, 1), Val: 0.155, Label: "neutral"}},
		}
		assert.Equal(t, exp, response, "keyless Factor should decode every fixture field")

		_, err = provider.Factor(t.Context(), "AUD", "monetary_stance", nil)
		assert.ErrorIs(t, err, errAPIKeyNotConfigured, "a non-USD Factor request should require an API key")
	})

	t.Run("announcement changes", func(t *testing.T) {
		provider, closeServer := newKeylessContractProvider(t, "/api/v1/announcements/changes", `{
			"count":0,"next_cursor":"","has_more":false,"retention_seconds":604800,
			"scope":{"currencies":["usd"],"indicators":null,"payload":"compact"},"data":[]
		}`)
		defer closeServer()

		response, err := provider.AnnouncementChanges(t.Context(), nil)
		require.NoError(t, err, "a keyless changes request must be sent, since the server scopes it to USD")
		exp := &AnnouncementChangesResponse{
			Data:             []AnnouncementChangeEvent{},
			RetentionSeconds: 604800,
			Scope:            AnnouncementChangesScope{Currencies: []string{"usd"}, Payload: "compact"},
		}
		assert.Equal(t, exp, response, "keyless AnnouncementChanges should decode every fixture field")
	})
}

func TestAuthenticatedEndpointsRefuseWithoutAnAPIKey(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("an unconfigured key must fail before any request is issued")
	}))
	defer closeServer()
	provider.APIKey = ""

	_, err := provider.Forex(t.Context(), usd, "AUD", nil)
	assert.ErrorIs(t, err, errAPIKeyNotConfigured, "an authenticated endpoint should refuse without a key")
	_, err = provider.Predictions(t.Context(), usd, inflation, nil)
	assert.ErrorIs(t, err, errAPIKeyNotConfigured, "Predictions should require a key on every currency, USD included")
}

func TestForex(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/forex/usd/aud", `{
		"base":"USD","quote":"AUD","source":"official_reference_rates",
		"provenance":{"publisher":"official"},"data_quality":{"row_count":1},
		"start_date":"2026-08-01","end_date":"2026-08-01","dataset_version":"v1",
		"coverage":{"scope":"requested_range"},"pagination":{"returned_count":1},
		"data":[{"date":"2026-08-01","val":1.53,"announcement_datetime":1785542400,
		"observation_datetime":1785542400,"observation_datetime_iso":"2026-08-01T00:00:00Z",
		"source":{"source_pair":"AUD/USD","is_derived":true,"derivation_method":"inverse"},"rsi_14":55.2}]
	}`, true)
	defer closeServer()

	response, err := provider.Forex(t.Context(), usd, "AUD", url.Values{"limit": {"1"}})
	require.NoError(t, err, "Forex must decode a documented response")
	day := calendarDay(2026, time.August, 1)
	exp := &ForexResponse{
		Base:           usd,
		Quote:          "AUD",
		Source:         "official_reference_rates",
		Provenance:     Provenance{Publisher: "official"},
		DataQuality:    DataQuality{RowCount: 1},
		StartDate:      day,
		EndDate:        day,
		DatasetVersion: "v1",
		Coverage:       map[string]any{"scope": "requested_range"},
		Pagination:     PaginationInfo{ReturnedCount: 1},
		Data: []ForexDataPoint{{
			Date:                   day,
			Val:                    1.53,
			AnnouncementDatetime:   unixSeconds(1785542400),
			ObservationDatetime:    unixSeconds(1785542400),
			ObservationDatetimeISO: utcTime(t, "2026-08-01T00:00:00Z"),
			Source:                 DataPointSource{SourcePair: "AUD/USD", IsDerived: true, DerivationMethod: "inverse"},
			RSI14:                  55.2,
		}},
	}
	assert.Equal(t, exp, response, "Forex should decode every fixture field")
}

func TestDataCatalogue(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/data_catalogue/usd", `{
		"inflation":{"name":"Inflation (CPI)","unit":"%YoY","frequency":"Monthly",
		"has_official_forecast":true,"source":"BLS","proxy_note":null,
		"coverage":{"available":true,"requires_api_key":false,"row_count":42,"data_lag_days":2,"stale_after_days":95},
		"series_variants":[{"series_id":"usd_inflation_cpi_yoy_nsa","is_default":true,"frequency_selector":"yoy"}],
		"supported_options":{"seasonality":["nsa","sa"]}}
	}`, false)
	defer closeServer()

	response, err := provider.DataCatalogue(t.Context(), usd)
	require.NoError(t, err, "DataCatalogue must decode a documented response")
	exp := &DataCatalogueResponse{
		inflation: {
			Name:                "Inflation (CPI)",
			Unit:                "%YoY",
			Frequency:           "Monthly",
			HasOfficialForecast: true,
			Source:              "BLS",
			Coverage:            CatalogueCoverage{Available: true, RowCount: 42, DataLagDays: 2, StaleAfterDays: 95},
			SeriesVariants:      []CatalogueSeriesVariant{{SeriesID: "usd_inflation_cpi_yoy_nsa", IsDefault: true, FrequencySelector: "yoy"}},
			SupportedOptions:    map[string][]string{"seasonality": {"nsa", "sa"}},
		},
	}
	assert.Equal(t, exp, response, "DataCatalogue should decode every fixture field")
}

func TestAnnouncements(t *testing.T) {
	// The second row carries a 1999 release time, which epoch seconds encode in
	// nine digits; the first carries a 1914 one, which needs a sign and ten.
	provider, closeServer := newContractProvider(t, "/api/v1/announcements/usd/inflation", `{
		"currency":"USD","indicator":"inflation","name":"Inflation (CPI)",
		"source_series_name":"Consumer Price Index","policy_role":"inflation_target",
		"has_official_forecast":true,"start_date":"2026-07-31","end_date":"2026-07-31",
		"remap":{"applied":false,"rule_id":null,"transform":"raw","canonical_indicator":"inflation"},
		"filters":{"basis":null,"frequency":null,"revisions":"latest","seasonality":null,"annualization":null,"period_aggregation":null},
		"selected_series_id":"usd_inflation_canonical_yoy_nsa_not_annualized_monthly",
		"selected_series":{"series_id":"usd_inflation_canonical_yoy_nsa_not_annualized_monthly","label":"Inflation, YOY, NSA",
		"source_series_id":"BLS:CUUR0000SA0","seasonality":"nsa","frequency":"yoy","revisions":"latest"},
		"data_quality":{"row_count":3,"reason":"current","unverified_vintage_count":3,
		"historical_point_in_time":{"row_count":1349,"assumed_release_time_count":997,"point_in_time_safe":false}},
		"dataset_version":"d67eee6b","value_mode":"normalized",
		"value_metadata":{"source_unit":"%YoY","normalization_applied":true},
		"pagination":{"limit":3,"returned_count":3},
		"freemium_window":{"applied":true,"max_days":90,"cutoff_date":"2026-05-14",
		"message":"Anonymous access returns the most recent 90 days."},
		"data":[{"announcement_id":"usd_inflation_2026-07-31","date":"2026-07-31","val":2.7,
		"previous_value":2.6,"announcement_datetime":1786105800,"pct_change_yoy":2.7,"val_mom":0.4,
		"announcement_datetime_local":"2026-08-12T12:30:00Z","collected_at_ns":1786105806674741407,
		"collected_at_iso":"2026-08-12T12:30:06.674742Z","publication_time_status":"unverified",
		"vintage_status":"captured_snapshot","observed_at_ns":1786105806674741407,
		"revisions":[{"epoch":1786100000,"val":2.6,"is_first_release":true}],"remap_applied":false},
		{"announcement_id":"usd_inflation_1999-01-15","date":"1998-12-31","val":1.6,
		"announcement_datetime":916407000,"release_time_assumed":true},
		{"announcement_id":"usd_inflation_1914-02-13","date":"1914-01-31","val":2.0,
		"announcement_datetime":-1763461800,"release_time_assumed":true}]
	}`, true)
	defer closeServer()

	response, err := provider.Announcements(t.Context(), usd, inflation, url.Values{"limit": {"3"}})
	require.NoError(t, err, "Announcements must decode a documented response")
	exp := &AnnouncementResponse{
		Currency:            usd,
		Indicator:           inflation,
		Name:                "Inflation (CPI)",
		SourceSeriesName:    "Consumer Price Index",
		PolicyRole:          "inflation_target",
		HasOfficialForecast: true,
		StartDate:           calendarDay(2026, time.July, 31),
		EndDate:             calendarDay(2026, time.July, 31),
		Remap:               AnnouncementRemap{Transform: "raw", CanonicalIndicator: inflation},
		Filters:             SeriesFilters{Revisions: "latest"},
		SelectedSeriesID:    "usd_inflation_canonical_yoy_nsa_not_annualized_monthly",
		SelectedSeries: SelectedSeries{
			SeriesID:       "usd_inflation_canonical_yoy_nsa_not_annualized_monthly",
			Label:          "Inflation, YOY, NSA",
			SourceSeriesID: "BLS:CUUR0000SA0",
			Seasonality:    "nsa",
			Frequency:      "yoy",
			Revisions:      "latest",
		},
		DataQuality: DataQuality{
			RowCount:               3,
			Reason:                 "current",
			UnverifiedVintageCount: 3,
			HistoricalPointInTime:  PointInTimeCompleteness{RowCount: 1349, AssumedReleaseTimeCount: 997},
		},
		DatasetVersion: "d67eee6b",
		ValueMode:      "normalized",
		ValueMetadata:  ValueMetadata{NormalizationApplied: true, SourceUnit: "%YoY"},
		Pagination:     PaginationInfo{Limit: 3, ReturnedCount: 3},
		FreemiumWindow: FreemiumWindow{
			Applied:    true,
			MaxDays:    90,
			CutoffDate: calendarDay(2026, time.May, 14),
			Message:    "Anonymous access returns the most recent 90 days.",
		},
		Data: []AnnouncementDataPoint{
			{
				AnnouncementID:            "usd_inflation_2026-07-31",
				Date:                      calendarDay(2026, time.July, 31),
				Val:                       2.7,
				PreviousValue:             2.6,
				ValMonthOverMonth:         0.4,
				AnnouncementDatetime:      unixSeconds(1786105800),
				AnnouncementDatetimeLocal: utcTime(t, "2026-08-12T12:30:00Z"),
				PublicationTimeStatus:     "unverified",
				VintageStatus:             "captured_snapshot",
				ObservedAtNS:              types.Time(time.Unix(0, 1786105806674741407)),
				CollectedAtNS:             types.Time(time.Unix(0, 1786105806674741407)),
				CollectedAtISO:            utcTime(t, "2026-08-12T12:30:06.674742Z"),
				PctChangeYearOverYear:     2.7,
				Revisions:                 []RevisionEntry{{Epoch: unixSeconds(1786100000), Val: 2.6, IsFirstRelease: true}},
			},
			{
				AnnouncementID:       "usd_inflation_1999-01-15",
				Date:                 calendarDay(1998, time.December, 31),
				Val:                  1.6,
				AnnouncementDatetime: unixSeconds(916407000),
				ReleaseTimeAssumed:   true,
			},
			{
				AnnouncementID:       "usd_inflation_1914-02-13",
				Date:                 calendarDay(1914, time.January, 31),
				Val:                  2.0,
				AnnouncementDatetime: unixSeconds(-1763461800),
				ReleaseTimeAssumed:   true,
			},
		},
	}
	assert.Equal(t, exp, response, "Announcements should decode every fixture field")
	assert.Equal(t, "1999-01-15T13:30:00Z", response.Data[1].AnnouncementDatetime.Time().Format(time.RFC3339),
		"Announcements should decode a nine-digit epoch as seconds")
	assert.Equal(t, "1914-02-13T13:30:00Z", response.Data[2].AnnouncementDatetime.Time().Format(time.RFC3339),
		"Announcements should decode a negative epoch as seconds")
}

func TestAnnouncementsWithoutFreemiumWindow(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/announcements/usd/inflation", `{
		"currency":"USD","indicator":"inflation","freemium_window":null,
		"pagination":{"limit":1,"returned_count":1},
		"data":[{"announcement_id":"usd_inflation_2026-07-31","date":"2026-07-31","val":2.7}]
	}`, true)
	defer closeServer()

	response, err := provider.Announcements(t.Context(), usd, inflation, nil)
	require.NoError(t, err, "Announcements must decode a response without a freemium window")
	exp := &AnnouncementResponse{
		Currency:   usd,
		Indicator:  inflation,
		Pagination: PaginationInfo{Limit: 1, ReturnedCount: 1},
		Data:       []AnnouncementDataPoint{{AnnouncementID: "usd_inflation_2026-07-31", Date: calendarDay(2026, time.July, 31), Val: 2.7}},
	}
	assert.Equal(t, exp, response, "Announcements should leave the freemium window at its zero value when the clamp did not apply")
}

func TestLatestAnnouncements(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/announcements/usd/latest", `{
		"currency":"USD","source":"FXMacroData","as_of":"2026-08-12","count":1,
		"data":[{"indicator":"inflation","name":"Inflation (CPI)","source":"BLS",
		"unit":"%YoY","frequency":"Monthly","has_official_forecast":true,
		"latest":{"date":"2026-07-31","val":2.7,"announcement_datetime":1786105800},
		"previous":{"date":"2026-06-30","val":2.6,"announcement_datetime":1783427400},
		"pct_change_yoy":2.7,"pct_diff_prev":3.8}]
	}`, true)
	defer closeServer()

	response, err := provider.LatestAnnouncements(t.Context(), usd, nil)
	require.NoError(t, err, "LatestAnnouncements must decode a documented response")
	exp := &LatestAnnouncementsResponse{
		Currency: usd,
		Source:   providerName,
		AsOf:     calendarDay(2026, time.August, 12),
		Count:    1,
		Data: []LatestAnnouncementItem{{
			Indicator:             inflation,
			Name:                  "Inflation (CPI)",
			Source:                "BLS",
			Unit:                  "%YoY",
			Frequency:             "Monthly",
			HasOfficialForecast:   true,
			Latest:                LatestAnnouncementValue{Date: calendarDay(2026, time.July, 31), Val: 2.7, AnnouncementDatetime: unixSeconds(1786105800)},
			Previous:              LatestAnnouncementValue{Date: calendarDay(2026, time.June, 30), Val: 2.6, AnnouncementDatetime: unixSeconds(1783427400)},
			PctChangeYearOverYear: 2.7,
			PctDiffPrev:           3.8,
		}},
	}
	assert.Equal(t, exp, response, "LatestAnnouncements should decode every fixture field")
}

func TestAnnouncementChanges(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/announcements/changes", `{
		"count":1,"next_cursor":"cursor-2","has_more":true,"retention_seconds":86400,
		"scope":{"currencies":["usd"],"indicators":["inflation"],"payload":"compact"},
		"data":[{"event_id":"event-1","currency":"usd","indicator":"inflation","records_written":2.0,
		"timestamp":1786105800,"release_timestamp":1786105800,"delivery_mode":"live",
		"polling_started_at_ns":1786105795952048238,"polling_started_lag_ms":-4047.952,
		"polling_started_before_scheduled_release":true,"pre_fresh_stale_poll_count":273,
		"subsecond_contract_outcome":"met","subsecond_contract_breached":false,
		"latest_announcement":{"date":"2026-07-31","val":2.7,"original_val":2.7,"original_unit":"%YoY",
		"announcement_datetime":null,"source":"BLS","source_url":"https://www.bls.gov/","source_url_scope":"release"}}]
	}`, true)
	defer closeServer()

	response, err := provider.AnnouncementChanges(t.Context(), nil)
	require.NoError(t, err, "AnnouncementChanges must decode a documented response")
	exp := &AnnouncementChangesResponse{
		Count:            1,
		NextCursor:       "cursor-2",
		HasMore:          true,
		RetentionSeconds: 86400,
		Scope:            AnnouncementChangesScope{Currencies: []string{"usd"}, Indicators: []string{inflation}, Payload: "compact"},
		Data: []AnnouncementChangeEvent{{
			EventID:                              "event-1",
			Currency:                             "usd",
			Indicator:                            inflation,
			RecordsWritten:                       2.0,
			Timestamp:                            types.Time(time.Unix(1786105800, 0)),
			ReleaseTimestamp:                     types.Time(time.Unix(1786105800, 0)),
			DeliveryMode:                         "live",
			PollingStartedAtNS:                   types.Time(time.Unix(0, 1786105795952048238)),
			PollingStartedLagMS:                  -4047.952,
			PollingStartedBeforeScheduledRelease: true,
			PreFreshStalePollCount:               273,
			SubsecondContractOutcome:             "met",
			LatestAnnouncement: ReleaseDeliveryAnnouncement{
				Date:           calendarDay(2026, time.July, 31),
				Val:            2.7,
				OriginalVal:    2.7,
				OriginalUnit:   "%YoY",
				Source:         "BLS",
				SourceURL:      "https://www.bls.gov/",
				SourceURLScope: "release",
			},
		}},
	}
	assert.Equal(t, exp, response, "AnnouncementChanges should decode every fixture field")
}

func TestCalendar(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/calendar/usd", `{
		"currency":"USD","timezone":"America/New_York","requested_timezone":"UTC",
		"data_quality":{"row_count":1},"data":[{"announcement_datetime":1786105800,
		"release":"inflation","calendar_event_id":"usd_inflation_2026-08-07",
		"announcement_datetime_utc":"2026-08-07T12:30:00Z","announcement_datetime_local":"2026-08-07T12:30:00Z",
		"release_date_confirmed":true,"release_time_assumed":false,"source":"BLS",
		"source_url":"https://www.bls.gov/schedule/","date":"2026-07-31","event_importance":"high",
		"market_tier":1,"top_tier_for_currency":true}]
	}`, true)
	defer closeServer()

	response, err := provider.Calendar(t.Context(), usd, nil)
	require.NoError(t, err, "Calendar must decode a documented response")
	exp := &CalendarResponse{
		Currency:          usd,
		Timezone:          "America/New_York",
		RequestedTimezone: "UTC",
		DataQuality:       DataQuality{RowCount: 1},
		Data: []CalendarReleaseRow{{
			AnnouncementDatetime:      unixSeconds(1786105800),
			Release:                   inflation,
			CalendarEventID:           "usd_inflation_2026-08-07",
			AnnouncementDatetimeUTC:   utcTime(t, "2026-08-07T12:30:00Z"),
			AnnouncementDatetimeLocal: utcTime(t, "2026-08-07T12:30:00Z"),
			ReleaseDateConfirmed:      true,
			Source:                    "BLS",
			SourceURL:                 "https://www.bls.gov/schedule/",
			Date:                      calendarDay(2026, time.July, 31),
			EventImportance:           "high",
			MarketTier:                1,
			TopTierForCurrency:        true,
		}},
	}
	assert.Equal(t, exp, response, "Calendar should decode every fixture field")
}

func TestPredictions(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/predictions/usd/inflation", `{
		"currency":"USD","indicator":"inflation","pre_release_only":true,"count":1,
		"filters":{"revisions":"latest"},"selected_series":{"series_id":"usd_inflation_cpi_yoy_nsa"},
		"prediction_count":1,"has_more":false,"data_quality":{"row_count":1},
		"data":[{"announcement_id":"usd_inflation_2026-08-31","currency":"USD",
		"indicator":"inflation","date":"2026-08-31","announcement_datetime":1788134400,
		"predictions":[{"predicted_value":2.5,"prediction_type":"market_consensus",
		"prediction_source":"survey","generated_at":1788000000,"is_pre_release":true,"confidence":0.8,
		"provenance":{"value_origin":"published_forecast","capture_mode":"archived","archived_in_real_time_by_fxmacrodata":true}}]}]
	}`, true)
	defer closeServer()

	response, err := provider.Predictions(t.Context(), usd, inflation, nil)
	require.NoError(t, err, "Predictions must decode a documented response")
	exp := &PredictionsResponse{
		Currency:        usd,
		Indicator:       inflation,
		Filters:         SeriesFilters{Revisions: "latest"},
		SelectedSeries:  SelectedSeries{SeriesID: "usd_inflation_cpi_yoy_nsa"},
		PreReleaseOnly:  true,
		Count:           1,
		PredictionCount: 1,
		DataQuality:     DataQuality{RowCount: 1},
		Data: []AnnouncementPredictions{{
			AnnouncementID:       "usd_inflation_2026-08-31",
			Currency:             usd,
			Indicator:            inflation,
			Date:                 calendarDay(2026, time.August, 31),
			AnnouncementDatetime: unixSeconds(1788134400),
			Predictions: []PredictionItem{{
				PredictedValue:   2.5,
				PredictionType:   "market_consensus",
				PredictionSource: "survey",
				GeneratedAt:      unixSeconds(1788000000),
				IsPreRelease:     true,
				Confidence:       0.8,
				Provenance: PredictionProvenance{
					ValueOrigin:                     "published_forecast",
					CaptureMode:                     "archived",
					ArchivedInRealTimeByFXMacroData: true,
				},
			}},
		}},
	}
	assert.Equal(t, exp, response, "Predictions should decode every fixture field")
}

func TestCOT(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/cot/jpy", `{
		"currency":"JPY","instrument":"JAPANESE YEN","source":"CFTC","source_url":"https://www.cftc.gov/",
		"provenance":{"publisher":"CFTC","publisher_url":"https://www.cftc.gov/","storage":"official-archive",
		"served_by":"/v1/cot/{currency}","timestamp_field":"announcement_datetime","value_field":"noncommercial_net"},
		"fx_overlay":{"pair":"USD/JPY"},"start_date":"2026-08-01","end_date":"2026-08-08",
		"last_sync_status":"ok","expected_next_release":"2026-08-14T19:30:00Z","expected_next_release_epoch":1786735800,
		"last_updated":"2026-08-07T20:45:36.226631Z","data_lag_days":3,"data_quality":{"row_count":1},
		"pagination":{"limit":1,"returned_count":1,"total_count":1,"has_more":false},
		"data":[{"date":"2026-08-04","announcement_datetime":1786132800,"open_interest":250000,
		"noncommercial_long":80000,"noncommercial_short":120000,"noncommercial_net":-40000,
		"commercial_long":140000,"commercial_short":90000,"commercial_net":50000,
		"release_datetime":"2026-08-07T19:30:00Z","release_date_confirmed":true,
		"release_source":"CFTC","release_source_url":"https://www.cftc.gov/"}]
	}`, true)
	defer closeServer()

	response, err := provider.COT(t.Context(), "JPY", nil)
	require.NoError(t, err, "COT must decode a documented response")
	exp := &COTResponse{
		Currency:   "JPY",
		Instrument: "JAPANESE YEN",
		Source:     "CFTC",
		SourceURL:  "https://www.cftc.gov/",
		Provenance: COTProvenance{
			Publisher:      "CFTC",
			PublisherURL:   "https://www.cftc.gov/",
			Storage:        "official-archive",
			ServedBy:       "/v1/cot/{currency}",
			TimestampField: "announcement_datetime",
			ValueField:     "noncommercial_net",
		},
		FXOverlay:                COTFXOverlay{Pair: "USD/JPY"},
		StartDate:                calendarDay(2026, time.August, 1),
		EndDate:                  calendarDay(2026, time.August, 8),
		LastSyncStatus:           "ok",
		ExpectedNextRelease:      utcTime(t, "2026-08-14T19:30:00Z"),
		ExpectedNextReleaseEpoch: unixSeconds(1786735800),
		LastUpdated:              utcTime(t, "2026-08-07T20:45:36.226631Z"),
		DataLagDays:              3,
		DataQuality:              DataQuality{RowCount: 1},
		Pagination:               Pagination{Limit: 1, ReturnedCount: 1, TotalCount: 1},
		Data: []COTDataPoint{{
			Date:                 calendarDay(2026, time.August, 4),
			AnnouncementDatetime: unixSeconds(1786132800),
			OpenInterest:         250000,
			NonCommercialLong:    80000,
			NonCommercialShort:   120000,
			NonCommercialNet:     -40000,
			CommercialLong:       140000,
			CommercialShort:      90000,
			CommercialNet:        50000,
			ReleaseDatetime:      utcTime(t, "2026-08-07T19:30:00Z"),
			ReleaseDateConfirmed: true,
			ReleaseSource:        "CFTC",
			ReleaseSourceURL:     "https://www.cftc.gov/",
		}},
	}
	assert.Equal(t, exp, response, "COT should decode every fixture field")
}

func TestCommodity(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/commodities/brent", `{
		"currency":"USD","indicator":"brent","source":"EIA","source_url":"https://www.eia.gov/",
		"has_official_forecast":false,"last_updated":"2026-08-12T00:00:00Z","latest_available_date":"2026-08-11",
		"data_quality":{"row_count":1},"start_date":"2026-08-11","end_date":"2026-08-11","pagination":{"returned_count":1},
		"data":[{"date":"2026-08-11","val":68.4,"announcement_datetime":1786406400,"pct_change":1.2,"pct_change_12m":-4.5}]
	}`, true)
	defer closeServer()

	response, err := provider.Commodity(t.Context(), "brent", nil)
	require.NoError(t, err, "Commodity must decode a documented response")
	day := calendarDay(2026, time.August, 11)
	exp := &CommodityResponse{
		Currency:            usd,
		Indicator:           "brent",
		Source:              "EIA",
		SourceURL:           "https://www.eia.gov/",
		LastUpdated:         utcTime(t, "2026-08-12T00:00:00Z"),
		LatestAvailableDate: day,
		DataQuality:         DataQuality{RowCount: 1},
		StartDate:           day,
		EndDate:             day,
		Pagination:          PaginationInfo{ReturnedCount: 1},
		Data: []CommodityDataPoint{{
			Date:                 day,
			Val:                  68.4,
			AnnouncementDatetime: unixSeconds(1786406400),
			PctChange:            1.2,
			PctChange12Month:     -4.5,
		}},
	}
	assert.Equal(t, exp, response, "Commodity should decode every fixture field")
}

func TestCommoditiesLatest(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/commodities/latest", `{
		"currency":"USD","source":"EIA","as_of":"2026-08-12","count":1,
		"data":[{"indicator":"brent","unit":"USD/bbl","frequency":"Daily","has_official_forecast":false,
		"last_updated":"2026-08-12T00:00:00Z","data_quality":{"row_count":2},
		"latest":{"date":"2026-08-11","val":68.4,"announcement_datetime":1786406400},
		"previous":{"date":"2026-08-10","val":67.9,"announcement_datetime":1786320000},"pct_diff_prev":0.74}]
	}`, true)
	defer closeServer()

	response, err := provider.CommoditiesLatest(t.Context(), nil)
	require.NoError(t, err, "CommoditiesLatest must decode a documented response")
	exp := &CommoditiesLatestResponse{
		Currency: usd,
		Source:   "EIA",
		AsOf:     calendarDay(2026, time.August, 12),
		Count:    1,
		Data: []CommodityLatestItem{{
			Indicator:   "brent",
			Unit:        "USD/bbl",
			Frequency:   "Daily",
			LastUpdated: utcTime(t, "2026-08-12T00:00:00Z"),
			DataQuality: DataQuality{RowCount: 2},
			Latest:      CommodityObservation{Date: calendarDay(2026, time.August, 11), Val: 68.4, AnnouncementDatetime: unixSeconds(1786406400)},
			Previous:    CommodityObservation{Date: calendarDay(2026, time.August, 10), Val: 67.9, AnnouncementDatetime: unixSeconds(1786320000)},
			PctDiffPrev: 0.74,
		}},
	}
	assert.Equal(t, exp, response, "CommoditiesLatest should decode every fixture field")
}

func TestCurves(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/curves/usd", `{
		"currency":"USD","curve_family":"government","view":"nodes","metric":"yield",
		"requested_date":"2026-08-12","as_of":"2026-08-12","node_count":1,"sources":["Treasury"],
		"official_forward_source_support":{"source_type":"none","source_note":"No official forward curve is published."},
		"data_quality":{"row_count":1},
		"data":[{"indicator":"bond_yield_10y","maturity":"10Y","val":4.21}]
	}`, true)
	defer closeServer()

	response, err := provider.Curves(t.Context(), usd, nil)
	require.NoError(t, err, "Curves must decode a documented response")
	exp := &CurveAnalyticsResponse{
		Currency:                     usd,
		CurveFamily:                  "government",
		View:                         "nodes",
		Metric:                       "yield",
		RequestedDate:                calendarDay(2026, time.August, 12),
		AsOf:                         calendarDay(2026, time.August, 12),
		NodeCount:                    1,
		Sources:                      []string{"Treasury"},
		OfficialForwardSourceSupport: OfficialForwardSourceSupport{SourceType: "none", SourceNote: "No official forward curve is published."},
		DataQuality:                  DataQuality{RowCount: 1},
		Data:                         []map[string]any{{"indicator": "bond_yield_10y", "maturity": "10Y", "val": 4.21}},
	}
	assert.Equal(t, exp, response, "Curves should decode every fixture field")
}

func TestFactor(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/factors/usd/inflation_pressure", `{
		"currency":"USD","factor":"inflation_pressure","methodology":"standardised",
		"as_of":"2026-08-12","score":0.7,"label":"elevated","last_updated":"2026-08-13T04:40:19Z",
		"start_date":"2026-08-12","end_date":"2026-08-12","data_quality":{"row_count":1},"pagination":{"returned_count":1},
		"data":[{"date":"2026-08-12","val":0.7,"score":0.7,"point_in_time_safe":true,"component_count":1,
		"components":{"inflation":0.8},"source_endpoints":["announcements/usd/inflation"]}]
	}`, true)
	defer closeServer()

	response, err := provider.Factor(t.Context(), usd, "inflation_pressure", nil)
	require.NoError(t, err, "Factor must decode a documented response")
	day := calendarDay(2026, time.August, 12)
	exp := &FactorResponse{
		Currency:    usd,
		Factor:      "inflation_pressure",
		Methodology: "standardised",
		AsOf:        day,
		Score:       0.7,
		Label:       "elevated",
		LastUpdated: utcTime(t, "2026-08-13T04:40:19Z"),
		StartDate:   day,
		EndDate:     day,
		DataQuality: DataQuality{RowCount: 1},
		Pagination:  PaginationInfo{ReturnedCount: 1},
		Data: []FactorDataPoint{{
			Date:            day,
			Val:             0.7,
			Score:           0.7,
			PointInTimeSafe: true,
			ComponentCount:  1,
			Components:      map[string]any{inflation: 0.8},
			SourceEndpoints: []string{"announcements/usd/inflation"},
		}},
	}
	assert.Equal(t, exp, response, "Factor should decode every fixture field")
}

func TestRateDifferentials(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/rate_differentials/eur/usd", `{
		"base":"EUR","quote":"USD","rate_type":"spot","measure_requested":"policy_rate",
		"measure_used":"policy_rate","base_indicator":"policy_rate","quote_indicator":"policy_rate",
		"start_date":"2026-08-01","end_date":"2026-08-12","matched_points":1,"unit":"percentage_points",
		"latest_spread":-1.75,"latest_spread_bps":-175,"sources":{"base":"ECB","quote":["Federal Reserve","Treasury"]},
		"official_forward_source_support":{"source_type":"none"},
		"data_quality":{"row_count":1},"pagination":{"returned_count":1},
		"data":[{"date":"2026-08-12","base_val":2.0,"quote_val":3.75,"spread":-1.75,
		"spread_bps":-175,"base_announcement_datetime":1786000000}]
	}`, true)
	defer closeServer()

	response, err := provider.RateDifferentials(t.Context(), "EUR", usd, nil)
	require.NoError(t, err, "RateDifferentials must decode a documented response")
	exp := &RateDifferentialResponse{
		Base:                         "EUR",
		Quote:                        usd,
		RateType:                     "spot",
		MeasureRequested:             "policy_rate",
		MeasureUsed:                  "policy_rate",
		BaseIndicator:                "policy_rate",
		QuoteIndicator:               "policy_rate",
		StartDate:                    calendarDay(2026, time.August, 1),
		EndDate:                      calendarDay(2026, time.August, 12),
		MatchedPoints:                1,
		Unit:                         "percentage_points",
		LatestSpread:                 -1.75,
		LatestSpreadBPS:              -175,
		Sources:                      RateDifferentialSources{Base: SourceNames{"ECB"}, Quote: SourceNames{"Federal Reserve", "Treasury"}},
		OfficialForwardSourceSupport: OfficialForwardSourceSupport{SourceType: "none"},
		DataQuality:                  DataQuality{RowCount: 1},
		Pagination:                   PaginationInfo{ReturnedCount: 1},
		Data: []RateDifferentialPoint{{
			Date:                     calendarDay(2026, time.August, 12),
			BaseVal:                  2.0,
			QuoteVal:                 3.75,
			Spread:                   -1.75,
			SpreadBPS:                -175,
			BaseAnnouncementDatetime: unixSeconds(1786000000),
		}},
	}
	assert.Equal(t, exp, response, "RateDifferentials should decode every fixture field")
}

func TestIntradayReferenceRates(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/fx/intraday-reference-rates/usd/aud", `{
		"pair":"USD/AUD","start_time":"2026-08-12T00:00:00Z","end_time":"2026-08-12T23:59:59Z",
		"data":[{"timestamp":"2026-08-12T16:00:00Z","price":1.53,"reference_date":"2026-08-12",
		"timestamp_type":"official_fixing","source":{"id":"official","name":"Official Source","is_official":true},
		"source_pair":"AUD/USD","derivation_method":"inverse"}]
	}`, true)
	defer closeServer()

	response, err := provider.IntradayReferenceRates(t.Context(), usd, "AUD", nil)
	require.NoError(t, err, "IntradayReferenceRates must decode a documented response")
	exp := &FXIntradayReferenceRatesResponse{
		Pair:      "USD/AUD",
		StartTime: utcTime(t, "2026-08-12T00:00:00Z"),
		EndTime:   utcTime(t, "2026-08-12T23:59:59Z"),
		Data: []FXIntradayReferenceRatePoint{{
			Timestamp:        utcTime(t, "2026-08-12T16:00:00Z"),
			Price:            1.53,
			ReferenceDate:    calendarDay(2026, time.August, 12),
			TimestampType:    "official_fixing",
			Source:           FXSource{ID: "official", Name: "Official Source", IsOfficial: true},
			SourcePair:       "AUD/USD",
			DerivationMethod: "inverse",
		}},
	}
	assert.Equal(t, exp, response, "IntradayReferenceRates should decode every fixture field")
	assert.Equal(t, "official", response.Data[0].Source.ID, "IntradayReferenceRates should decode the reference-rate source")
}

func TestFXSources(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/fx/sources", `{
		"source_policy":{"request_time_upstream_fetches":false,"value_field":"rate"},
		"sources":[{"id":"official-source","name":"Official Source","is_official":true,"native_pairs":["USD/AUD"],
		"native_pair_count":1,"served_pairs":["USD/AUD"],"served_pair_count":1}]
	}`, false)
	defer closeServer()

	response, err := provider.FXSources(t.Context(), nil)
	require.NoError(t, err, "FXSources must decode a documented response")
	exp := &FXSourcesResponse{
		SourcePolicy: FXSourcePolicy{ValueField: "rate"},
		Sources: []FXSource{{
			ID:              "official-source",
			Name:            "Official Source",
			IsOfficial:      true,
			NativePairs:     []string{"USD/AUD"},
			NativePairCount: 1,
			ServedPairs:     []string{"USD/AUD"},
			ServedPairCount: 1,
		}},
	}
	assert.Equal(t, exp, response, "FXSources should decode every fixture field")
}

func TestFXSourceUniverse(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/fx/source-universe", `{
		"source_policy":{"request_time_upstream_fetches":false},"currency":"USD",
		"data":[{"source":{"id":"official-source"},"pair_count":1,"pairs":[{"pair":"USD/AUD","availability":"native"}]}]
	}`, false)
	defer closeServer()

	response, err := provider.FXSourceUniverse(t.Context(), nil)
	require.NoError(t, err, "FXSourceUniverse must decode a documented response")
	exp := &FXSourceUniverseResponse{
		Currency: usd,
		Data: []FXSourceUniverseEntry{{
			Source:    FXSource{ID: "official-source"},
			PairCount: 1,
			Pairs:     []FXSourceUniversePair{{Pair: "USD/AUD", Availability: "native"}},
		}},
	}
	assert.Equal(t, exp, response, "FXSourceUniverse should decode every fixture field")
}

func TestMarketSessions(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/market_sessions", `{
		"now_utc":"2026-08-13T10:00:00Z","now_unix":1786615200,"is_market_day":true,
		"sessions":[{"name":"London","currencies":["GBP","EUR"],"open_utc":"2026-08-13T07:00:00Z",
		"close_utc":"2026-08-13T16:00:00Z","open_unix":1786604400,"close_unix":1786636800,"is_open":true,
		"seconds_to_open":null,"seconds_to_close":21600}],
		"overlaps":[{"name":"London / New York","sessions":["London","New York"],"start_utc":"2026-08-13T12:00:00Z",
		"end_utc":"2026-08-13T16:00:00Z","seconds_to_start":7200,"duration_hours":4}]
	}`, false)
	defer closeServer()

	response, err := provider.MarketSessions(t.Context(), nil)
	require.NoError(t, err, "MarketSessions must decode a documented response")
	exp := &MarketSessionsResponse{
		NowUTC:      utcTime(t, "2026-08-13T10:00:00Z"),
		NowUnix:     unixSeconds(1786615200),
		IsMarketDay: true,
		Sessions: []MarketSession{{
			Name:           "London",
			Currencies:     []string{"GBP", "EUR"},
			OpenUTC:        utcTime(t, "2026-08-13T07:00:00Z"),
			CloseUTC:       utcTime(t, "2026-08-13T16:00:00Z"),
			OpenUnix:       unixSeconds(1786604400),
			CloseUnix:      unixSeconds(1786636800),
			IsOpen:         true,
			SecondsToClose: 21600,
		}},
		Overlaps: []MarketSessionOverlap{{
			Name:           "London / New York",
			Sessions:       []string{"London", "New York"},
			StartUTC:       utcTime(t, "2026-08-13T12:00:00Z"),
			EndUTC:         utcTime(t, "2026-08-13T16:00:00Z"),
			SecondsToStart: 7200,
			DurationHours:  4,
		}},
	}
	assert.Equal(t, exp, response, "MarketSessions should decode every fixture field")
}

func TestRiskSentiment(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/risk_sentiment", `{
		"start_date":"2026-08-01","end_date":"2026-08-12","latest_available_date":"2026-08-12",
		"last_updated":"2026-08-13T00:00:00Z","data_quality":{"row_count":1},
		"component_metadata":{"stored_components":["ofr_fsi"],"aliases":{"score":"alias for val"}},
		"pagination":{"limit":1,"returned_count":1,"total_count":1,"has_more":false},
		"data":[{"components":{"ofr_fsi":0.5},"val":0.5,"date":"2026-08-12","regime":"risk_on",
		"component_coverage":{"ofr_fsi":true},"stored_component_count":1}]
	}`, false)
	defer closeServer()

	response, err := provider.RiskSentiment(t.Context(), nil)
	require.NoError(t, err, "RiskSentiment must decode a documented response")
	exp := &RiskSentimentResponse{
		StartDate:           calendarDay(2026, time.August, 1),
		EndDate:             calendarDay(2026, time.August, 12),
		LatestAvailableDate: calendarDay(2026, time.August, 12),
		LastUpdated:         utcTime(t, "2026-08-13T00:00:00Z"),
		DataQuality:         DataQuality{RowCount: 1},
		ComponentMetadata: RiskSentimentComponentMetadata{
			StoredComponents: []string{"ofr_fsi"},
			Aliases:          map[string]string{"score": "alias for val"},
		},
		Pagination: Pagination{Limit: 1, ReturnedCount: 1, TotalCount: 1},
		Data: []RiskSentimentPoint{{
			Components:           map[string]float64{"ofr_fsi": 0.5},
			Val:                  0.5,
			Date:                 calendarDay(2026, time.August, 12),
			Regime:               "risk_on",
			ComponentCoverage:    map[string]bool{"ofr_fsi": true},
			StoredComponentCount: 1,
		}},
	}
	assert.Equal(t, exp, response, "RiskSentiment should decode every fixture field")
}

func TestPressReleases(t *testing.T) {
	provider, closeServer := newContractProvider(t, "/api/v1/press-releases/usd", `{
		"currency":"USD","source":"Federal Reserve","source_url":"https://www.federalreserve.gov",
		"limit":1,"offset":0,"count":1,"pagination":{"limit":1,"returned_count":1,"total_count":1,"has_more":false},
		"data":[{"title":"Policy statement","url":"https://example.test/release","date":"2026-07-20",
		"summary":"Held rates","sentiment":0,"topics":["policy"],"category":"monetary_policy","relevance":0.9,
		"rate_path":{"score":0,"label":"Neutral","bias_action":"hold","confidence":"low","raw_score":0,"matches":[]}}]
	}`, true)
	defer closeServer()

	response, err := provider.PressReleases(t.Context(), usd, nil)
	require.NoError(t, err, "PressReleases must decode a documented response")
	exp := &PressReleasesResponse{
		Currency:   usd,
		Source:     "Federal Reserve",
		SourceURL:  "https://www.federalreserve.gov",
		Limit:      1,
		Count:      1,
		Pagination: Pagination{Limit: 1, ReturnedCount: 1, TotalCount: 1},
		Data: []PressReleaseItem{{
			Title:     "Policy statement",
			URL:       "https://example.test/release",
			Date:      calendarDay(2026, time.July, 20),
			Summary:   "Held rates",
			Topics:    []string{"policy"},
			Category:  "monetary_policy",
			Relevance: 0.9,
			RatePath:  RatePathSignal{Label: "Neutral", BiasAction: "hold", Confidence: "low", Matches: []any{}},
		}},
	}
	assert.Equal(t, exp, response, "PressReleases should decode every fixture field")
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
	require.NoError(t, provider.Setup(base.Settings{Name: providerName, APIKey: apiKey}),
		"Setup must configure the authenticated endpoint client")
	rate, err := provider.GetLatestForexRate(t.Context(), usd, "AUD")
	require.NoError(t, err, "GetLatestForexRate must not error in the opt-in authenticated smoke test")
	assert.Positive(t, rate, "GetLatestForexRate should return a positive rate")
}
