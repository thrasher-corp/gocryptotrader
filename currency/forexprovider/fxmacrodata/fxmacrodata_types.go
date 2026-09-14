package fxmacrodata

import (
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	// APIURL is the default FXMacroData API endpoint.
	APIURL = "https://api.fxmacrodata.com/v1/"

	// supportedCurrencies lists the currencies the forex endpoint serves. KRW is
	// accepted by the endpoint's enum but no public reference source serves it
	// yet, and a pair with no rows fails the whole rate batch, so it is left out
	// until the endpoint returns rows for it.
	supportedCurrencies = "AUD,BRL,CAD,CHF,CNH,CNY,DKK,EUR,GBP,HUF,ILS,JPY,MYR,NGN,NOK,NZD,PEN,SEK,THB,TWD,USD"
)

// Six response fields are declared as free-form objects by the FXMacroData
// OpenAPI contract, with additionalProperties and no named properties, because
// their keys vary with the requested indicator set, view or factor
// decomposition: Indicators, DailyOHLCBasis, TechnicalIndicatorBasis and
// Coverage on ForexResponse, Data on CurveAnalyticsResponse, and Components and
// SourceObservations on FactorDataPoint. Those stay map[string]any. Every field
// with a documented shape is strongly typed.

// Date represents an ISO 8601 calendar date without a time or timezone.
type Date time.Time

// UnmarshalJSON deserialises an ISO 8601 date.
func (d *Date) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("error unmarshalling %q into date: %w", data, err)
	}
	if value == "" {
		return nil
	}

	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return fmt.Errorf("error parsing %q into date: %w", value, err)
	}
	*d = Date(parsed)
	return nil
}

// Time converts Date to time.Time.
func (d Date) Time() time.Time {
	return time.Time(d)
}

// String returns the date in ISO 8601 format.
func (d Date) String() string {
	if d.Time().IsZero() {
		return ""
	}
	return d.Time().Format(time.DateOnly)
}

// MarshalJSON serialises the date in ISO 8601 format.
func (d Date) MarshalJSON() ([]byte, error) {
	if d.Time().IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.String())
}

// UnixSeconds is a Unix timestamp in whole seconds. FXMacroData documents its
// release and observation times as epoch seconds and derives them for rows
// reaching back to the start of each series. types.Time infers the unit from
// the number of digits, which cannot represent seconds before September 2001.
type UnixSeconds time.Time

// UnmarshalJSON deserialises a Unix timestamp in seconds.
func (u *UnixSeconds) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	seconds, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return fmt.Errorf("error parsing %q into Unix seconds: %w", data, err)
	}
	*u = UnixSeconds(time.Unix(seconds, 0).UTC())
	return nil
}

// Time converts UnixSeconds to time.Time.
func (u UnixSeconds) Time() time.Time {
	return time.Time(u)
}

// MarshalJSON serialises the timestamp as Unix seconds.
func (u UnixSeconds) MarshalJSON() ([]byte, error) {
	if u.Time().IsZero() {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(u.Time().Unix(), 10)), nil
}

// SourceNames is a list of publisher names. The rate-differential contract
// declares each leg's source as either a single string or a list of strings,
// so both are accepted and a single name decodes as a one-element list.
type SourceNames []string

// UnmarshalJSON deserialises either a single publisher name or a list of them.
func (s *SourceNames) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var single string
		if err := json.Unmarshal(data, &single); err != nil {
			return fmt.Errorf("error unmarshalling %q into source names: %w", data, err)
		}
		*s = SourceNames{single}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return fmt.Errorf("error unmarshalling %q into source names: %w", data, err)
	}
	*s = SourceNames(many)
	return nil
}

// FXMacroData is an FXMacroData foreign exchange and macro data provider.
type FXMacroData struct {
	base.Base
	Requester *request.Requester
	APIURL    string
}

// ServiceStatusResponse represents a public FXMacroData service status response.
type ServiceStatusResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// PointInTimeCompleteness describes timestamp coverage for historical rows.
type PointInTimeCompleteness struct {
	RowCount                     uint64 `json:"row_count"`
	AnnouncementDatetimeCount    uint64 `json:"announcement_datetime_count"`
	MissingAnnouncementDateCount uint64 `json:"missing_announcement_datetime_count"`
	AssumedReleaseTimeCount      uint64 `json:"assumed_release_time_count"`
	BoundedReleaseTimeCount      uint64 `json:"bounded_release_time_count"`
	UnboundedReleaseTimeCount    uint64 `json:"unbounded_release_time_count"`
	UnverifiedVintageCount       uint64 `json:"unverified_vintage_count"`
	UnknownPublicationTimeCount  uint64 `json:"unknown_publication_time_count"`
	PointInTimeSafe              bool   `json:"point_in_time_safe"`
}

// Provenance identifies the upstream publisher and storage backing a series.
// Publisher, Storage, ServedBy, TimestampField and ValueField are returned by
// every endpoint; the remaining fields are indicator-specific and decode to
// their zero value when the endpoint omits them.
type Provenance struct {
	Publisher          string `json:"publisher"`
	PublisherURL       string `json:"publisher_url"`
	Storage            string `json:"storage"`
	ServedBy           string `json:"served_by"`
	TimestampField     string `json:"timestamp_field"`
	ValueField         string `json:"value_field"`
	SourceSeriesID     string `json:"source_series_id"`
	SourceSeriesName   string `json:"source_series_name"`
	SourceLocalName    string `json:"source_local_name"`
	SeasonalAdjustment string `json:"seasonal_adjustment"`
	PriceBasis         string `json:"price_basis"`
	IsProxy            bool   `json:"is_proxy"`
	ProxyNote          string `json:"proxy_note"`
}

// SourceLeg describes a single upstream pair used to build a rate, either
// directly or as one leg of a derived cross.
type SourceLeg struct {
	Pair            string `json:"pair"`
	SourceProvider  string `json:"source_provider"`
	SourceName      string `json:"source_name"`
	SourceURL       string `json:"source_url"`
	SourceFrequency string `json:"source_frequency"`
}

// PairMetadata describes how a returned FX pair was sourced or derived.
type PairMetadata struct {
	DirectAvailable       bool        `json:"direct_available"`
	InverseAvailable      bool        `json:"inverse_available"`
	DerivedFromInverse    bool        `json:"derived_from_inverse"`
	DerivedFromCross      bool        `json:"derived_from_cross"`
	DerivedFromSourceLegs bool        `json:"derived_from_source_legs"`
	IsDerived             bool        `json:"is_derived"`
	DerivationMethod      string      `json:"derivation_method"`
	SourcePair            string      `json:"source_pair"`
	AnchorCurrency        string      `json:"anchor_currency"`
	PairConsistencyCheck  string      `json:"pair_consistency_check"`
	SourceLegs            []SourceLeg `json:"source_legs"`
}

// DataPointSource describes the derivation of an individual observation.
type DataPointSource struct {
	SourcePair            string      `json:"source_pair"`
	IsDerived             bool        `json:"is_derived"`
	DerivedFromSourceLegs bool        `json:"derived_from_source_legs"`
	DerivationMethod      string      `json:"derivation_method"`
	AnchorCurrency        string      `json:"anchor_currency"`
	PairConsistencyCheck  string      `json:"pair_consistency_check"`
	SourceLegs            []SourceLeg `json:"source_legs"`
}

// FXSourcePolicy describes the redistribution policy applied to FX reference
// sources served by the public endpoints.
type FXSourcePolicy struct {
	PublicDataPolicy           string `json:"public_data_policy"`
	RequestTimeUpstreamFetches bool   `json:"request_time_upstream_fetches"`
	ValueField                 string `json:"value_field"`
	LegacyDailyForexValueField string `json:"legacy_daily_forex_value_field"`
}

// FXSource describes a single official FX reference-rate source.
type FXSource struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	AuthorityType          string   `json:"authority_type"`
	CountryOrArea          string   `json:"country_or_area"`
	URL                    string   `json:"url"`
	AttributionText        string   `json:"attribution_text"`
	Tier                   string   `json:"tier"`
	IsOfficial             bool     `json:"is_official"`
	IsAggregator           bool     `json:"is_aggregator"`
	DefaultTimezone        string   `json:"default_timezone"`
	NativePairs            []string `json:"native_pairs"`
	PublicationFrequency   string   `json:"publication_frequency"`
	NativePairCount        uint64   `json:"native_pair_count"`
	SourceUniverseEndpoint string   `json:"source_universe_endpoint"`
	// NativePairs is what the authority publishes; ServedPairs is what this API
	// will actually return for it, and the two diverge sharply -- several
	// sources publish dozens of pairs and serve none. A caller reading only
	// NativePairs is misled about what it can request.
	ServedPairs     []string `json:"served_pairs"`
	ServedPairCount uint64   `json:"served_pair_count"`
	CoverageNote    string   `json:"coverage_note"`
}

// DataQuality describes the source and freshness characteristics of a result.
type DataQuality struct {
	IsOfficial                             bool                    `json:"is_official"`
	IsProxy                                bool                    `json:"is_proxy"`
	IsFallback                             bool                    `json:"is_fallback"`
	IsStale                                bool                    `json:"is_stale"`
	HasAnnouncementDatetime                bool                    `json:"has_announcement_datetime"`
	PointInTimeSafe                        bool                    `json:"point_in_time_safe"`
	PointInTimeBasis                       string                  `json:"point_in_time_basis"`
	SourcePermissionStatus                 string                  `json:"source_permission_status"`
	LatestAvailableDate                    Date                    `json:"latest_available_date"`
	LastUpdated                            time.Time               `json:"last_updated"`
	DataLagDays                            int64                   `json:"data_lag_days"`
	SourceName                             string                  `json:"source_name"`
	SourceType                             string                  `json:"source_type"`
	IsDerived                              bool                    `json:"is_derived"`
	RowCount                               uint64                  `json:"row_count"`
	AnnouncementDatetimeCount              uint64                  `json:"announcement_datetime_count"`
	MissingAnnouncementDateCount           uint64                  `json:"missing_announcement_datetime_count"`
	HasAssumedReleaseTimes                 bool                    `json:"has_assumed_release_times"`
	AssumedReleaseTimeCount                uint64                  `json:"assumed_release_time_count"`
	BoundedReleaseTimeCount                uint64                  `json:"bounded_release_time_count"`
	UnboundedReleaseTimeCount              uint64                  `json:"unbounded_release_time_count"`
	UnverifiedVintageCount                 uint64                  `json:"unverified_vintage_count"`
	UnknownPublicationTimeCount            uint64                  `json:"unknown_publication_time_count"`
	QualityScope                           string                  `json:"quality_scope"`
	StaleAfterDays                         uint64                  `json:"stale_after_days"`
	RequestedWindowHasData                 bool                    `json:"requested_window_has_data"`
	RequestedWindowLatestDate              Date                    `json:"requested_window_latest_date"`
	RequestedWindowIncludesLatestAvailable bool                    `json:"requested_window_includes_latest_available"`
	PageIncludesLatestAvailable            bool                    `json:"page_includes_latest_available"`
	ReturnedLatestAvailableBeforeWindow    bool                    `json:"returned_latest_available_before_window"`
	StalenessDays                          int64                   `json:"staleness_days"`
	Reason                                 string                  `json:"reason"`
	DatetimeField                          string                  `json:"datetime_field"`
	DatetimePrecision                      string                  `json:"datetime_precision"`
	DatetimeSemantics                      string                  `json:"datetime_semantics"`
	SourceLegs                             []SourceLeg             `json:"source_legs"`
	HistoricalPointInTime                  PointInTimeCompleteness `json:"historical_point_in_time"`
}

// PaginationInfo describes a paginated API result.
type PaginationInfo struct {
	Limit                       uint64 `json:"limit"`
	Offset                      uint64 `json:"offset"`
	ReturnedCount               uint64 `json:"returned_count"`
	TotalCount                  uint64 `json:"total_count"`
	HasMore                     bool   `json:"has_more"`
	NextOffset                  uint64 `json:"next_offset"`
	PageIncludesLatestAvailable bool   `json:"page_includes_latest_available"`
}

// Pagination describes offset pagination returned by list endpoints.
type Pagination struct {
	Limit         uint64 `json:"limit"`
	Offset        uint64 `json:"offset"`
	ReturnedCount uint64 `json:"returned_count"`
	TotalCount    uint64 `json:"total_count"`
	HasMore       bool   `json:"has_more"`
	NextOffset    uint64 `json:"next_offset"`
}

// DataCatalogueResponse maps indicator identifiers to their catalogue metadata.
type DataCatalogueResponse map[string]DataCatalogueItem

// DataCatalogueItem describes one advertised macroeconomic series.
type DataCatalogueItem struct {
	Name                string                   `json:"name"`
	Unit                string                   `json:"unit"`
	Frequency           string                   `json:"frequency"`
	HasOfficialForecast bool                     `json:"has_official_forecast"`
	Source              string                   `json:"source"`
	SourceSeriesID      string                   `json:"source_series_id"`
	SourceSeriesName    string                   `json:"source_series_name"`
	SeasonalAdjustment  string                   `json:"seasonal_adjustment"`
	PriceBasis          string                   `json:"price_basis"`
	Annualization       string                   `json:"annualization"`
	PeriodAggregation   string                   `json:"period_aggregation"`
	SeriesVariants      []CatalogueSeriesVariant `json:"series_variants"`
	Coverage            CatalogueCoverage        `json:"coverage"`
	SupportedOptions    map[string][]string      `json:"supported_options"`
	SourceURL           string                   `json:"source_url"`
	SourceURLScope      string                   `json:"source_url_scope"`
	IsProxy             bool                     `json:"is_proxy"`
	ProxyNote           string                   `json:"proxy_note"`
	// Aliases carries the names a series is also known by (non_farm_payrolls
	// answers to "NFP"), and RelatedIndicators the sibling slugs, so a caller
	// can resolve a user's wording without a second lookup.
	Aliases                   []string `json:"aliases"`
	RelatedIndicators         []string `json:"related_indicators"`
	StandardizationNote       string   `json:"standardization_note"`
	SourceHistoryStart        Date     `json:"source_history_start"`
	SupportedFrequencyOptions []string `json:"supported_frequency_options"`
	MaturityMonths            float64  `json:"maturity_months"`
	YieldType                 string   `json:"yield_type"`
}

// CatalogueSeriesVariant describes one selectable variant of a catalogue series.
type CatalogueSeriesVariant struct {
	SeriesID           string `json:"series_id"`
	StorageIndicator   string `json:"storage_indicator"`
	SourceSeriesID     string `json:"source_series_id"`
	SourceSeriesName   string `json:"source_series_name"`
	SeasonalAdjustment string `json:"seasonal_adjustment"`
	PriceBasis         string `json:"price_basis"`
	Annualization      string `json:"annualization"`
	PeriodAggregation  string `json:"period_aggregation"`
	FrequencySelector  string `json:"frequency_selector"`
	Unit               string `json:"unit"`
	Frequency          string `json:"frequency"`
	IsDefault          bool   `json:"is_default"`
	// A variant carries its own identity and coverage: selecting one by
	// SeriesID without reading these reports the parent series' source and
	// availability, which can differ.
	Name                string            `json:"name"`
	Source              string            `json:"source"`
	SourceURL           string            `json:"source_url"`
	SourceURLScope      string            `json:"source_url_scope"`
	StandardizationNote string            `json:"standardization_note"`
	HasOfficialForecast bool              `json:"has_official_forecast"`
	Coverage            CatalogueCoverage `json:"coverage"`
}

// CatalogueCoverage describes availability and freshness for a catalogue series.
type CatalogueCoverage struct {
	Available                  bool   `json:"available"`
	RequiresAPIKey             bool   `json:"requires_api_key"`
	EarliestAvailableDate      Date   `json:"earliest_available_date"`
	LatestAvailableDate        Date   `json:"latest_available_date"`
	RowCount                   uint64 `json:"row_count"`
	ValueAvailable             bool   `json:"value_available"`
	HasRecentData              bool   `json:"has_recent_data"`
	CoverageQuality            string `json:"coverage_quality"`
	HistoryCoverageQuality     string `json:"history_coverage_quality"`
	FreshnessQuality           string `json:"freshness_quality"`
	UsableForContext           bool   `json:"usable_for_context"`
	UsableForSignal            bool   `json:"usable_for_signal"`
	DataLagDays                int64  `json:"data_lag_days"`
	StaleAfterDays             uint64 `json:"stale_after_days"`
	RecentObservationCount     uint64 `json:"recent_observation_count"`
	HasYearOverYearTransform   bool   `json:"has_yoy_transform"`
	HasQuarterlyTransform      bool   `json:"has_qoq_transform"`
	HasMonthOverMonthTransform bool   `json:"has_mom_transform"`
	// LatestReleaseDate is present on every catalogue entry and is the date the
	// most recent observation was published, as opposed to the period it covers.
	LatestReleaseDate Date `json:"latest_release_date"`
}

// CBTargetEntry is one central-bank target effective from a date.
type CBTargetEntry struct {
	EffectiveFrom Date    `json:"effective_from"`
	Target        float64 `json:"target"`
	Lower         float64 `json:"lower"`
	Upper         float64 `json:"upper"`
	Notes         string  `json:"notes"`
}

// CBTargetInfo contains the current and historical central-bank targets.
type CBTargetInfo struct {
	Description string          `json:"description"`
	Source      string          `json:"source"`
	Current     CBTargetEntry   `json:"current"`
	History     []CBTargetEntry `json:"history"`
}

// PolicyFamilyEntry describes a related policy indicator.
type PolicyFamilyEntry struct {
	Indicator string `json:"indicator"`
	ValueName string `json:"value_name"`
	Role      string `json:"role"`
}

// FreemiumWindow reports an anonymous-access history clamp. It is only
// populated when the clamp shortened the response; without it a truncated
// history is indistinguishable from a genuinely short one, because
// Pagination.TotalCount counts the clamped series.
type FreemiumWindow struct {
	Applied    bool   `json:"applied"`
	MaxDays    uint64 `json:"max_days"`
	CutoffDate Date   `json:"cutoff_date"`
	Message    string `json:"message"`
}

// SeriesFilters is the set of series-variant filters a request was resolved
// with. An empty value means the caller did not constrain that dimension.
type SeriesFilters struct {
	Seasonality       string `json:"seasonality"`
	Basis             string `json:"basis"`
	Annualization     string `json:"annualization"`
	PeriodAggregation string `json:"period_aggregation"`
	Frequency         string `json:"frequency"`
	Revisions         string `json:"revisions"`
}

// SelectedSeries identifies the series variant a request resolved to.
type SelectedSeries struct {
	SeriesID          string `json:"series_id"`
	Label             string `json:"label"`
	SourceSeriesID    string `json:"source_series_id"`
	Seasonality       string `json:"seasonality"`
	Basis             string `json:"basis"`
	Annualization     string `json:"annualization"`
	PeriodAggregation string `json:"period_aggregation"`
	Frequency         string `json:"frequency"`
	Revisions         string `json:"revisions"`
}

// AnnouncementRemap reports whether the requested indicator resolved to a
// canonical series.
type AnnouncementRemap struct {
	Applied            bool   `json:"applied"`
	RuleID             string `json:"rule_id"`
	Transform          string `json:"transform"`
	CanonicalIndicator string `json:"canonical_indicator"`
}

// ValueMetadata describes how the returned values relate to the source unit.
type ValueMetadata struct {
	NormalizationApplied bool   `json:"normalization_applied"`
	SourceUnit           string `json:"source_unit"`
}

// AnnouncementResponse contains macroeconomic announcement observations.
type AnnouncementResponse struct {
	Currency                    string              `json:"currency"`
	Indicator                   string              `json:"indicator"`
	Name                        string              `json:"name"`
	ValueName                   string              `json:"value_name"`
	Source                      string              `json:"source"`
	SourceURL                   string              `json:"source_url"`
	SourceSeriesID              string              `json:"source_series_id"`
	SourceSeriesName            string              `json:"source_series_name"`
	SourceLocalName             string              `json:"source_local_name"`
	SeasonalAdjustment          string              `json:"seasonal_adjustment"`
	PriceBasis                  string              `json:"price_basis"`
	IsProxy                     bool                `json:"is_proxy"`
	ProxyNote                   string              `json:"proxy_note"`
	Provenance                  Provenance          `json:"provenance"`
	PolicyRole                  string              `json:"policy_role"`
	PolicyStructure             string              `json:"policy_structure"`
	ComparisonCompatible        bool                `json:"comparison_compatible"`
	PolicyFamily                []PolicyFamilyEntry `json:"policy_family"`
	HasOfficialForecast         bool                `json:"has_official_forecast"`
	RequestedStartDate          Date                `json:"requested_start_date"`
	RequestedEndDate            Date                `json:"requested_end_date"`
	RequestedWindowHasData      bool                `json:"requested_window_has_data"`
	PageIncludesLatestAvailable bool                `json:"page_includes_latest_available"`
	StartDate                   Date                `json:"start_date"`
	EndDate                     Date                `json:"end_date"`
	EarliestAvailableDate       Date                `json:"earliest_available_date"`
	LatestAvailableDate         Date                `json:"latest_available_date"`
	CentralBankTarget           CBTargetInfo        `json:"cb_target"`
	Remap                       AnnouncementRemap   `json:"remap"`
	Filters                     SeriesFilters       `json:"filters"`
	SelectedSeriesID            string              `json:"selected_series_id"`
	SelectedSeries              SelectedSeries      `json:"selected_series"`
	SupportedOptions            map[string][]string `json:"supported_options"`
	DataQuality                 DataQuality         `json:"data_quality"`
	// DatasetVersion identifies the content version the page was built from
	// and is the value the dataset_version query parameter takes to pin
	// pagination to one version.
	DatasetVersion string                  `json:"dataset_version"`
	ValueMode      string                  `json:"value_mode"`
	ValueMetadata  ValueMetadata           `json:"value_metadata"`
	Pagination     PaginationInfo          `json:"pagination"`
	FreemiumWindow FreemiumWindow          `json:"freemium_window"`
	Data           []AnnouncementDataPoint `json:"data"`
}

// RevisionEntry is one previously published value for an observation.
type RevisionEntry struct {
	Epoch                    UnixSeconds `json:"epoch"`
	Val                      float64     `json:"val"`
	Change                   float64     `json:"change"`
	ObservedAtNS             types.Time  `json:"observed_at_ns"`
	ObservedAtNSString       string      `json:"observed_at_ns_string"`
	PublicationAtNS          types.Time  `json:"publication_at_ns"`
	PublicationAtNSString    string      `json:"publication_at_ns_string"`
	AnnouncementSourceURL    string      `json:"announcement_source_url"`
	SourceURL                string      `json:"source_url"`
	SourceArtifactSHA256     string      `json:"source_artifact_sha256"`
	CaptureTimeBasis         string      `json:"capture_time_basis"`
	PublicationTimeStatus    string      `json:"publication_time_status"`
	PublicationTimePrecision string      `json:"publication_time_precision"`
	VintageStatus            string      `json:"vintage_status"`
	IsFirstRelease           bool        `json:"is_first_release"`
}

// AnnouncementDataPoint is an individual macroeconomic observation.
type AnnouncementDataPoint struct {
	AnnouncementID                      string          `json:"announcement_id"`
	Date                                Date            `json:"date"`
	Val                                 float64         `json:"val"`
	Source                              string          `json:"source"`
	SourceURL                           string          `json:"source_url"`
	SourceURLScope                      string          `json:"source_url_scope"`
	PreviousValue                       float64         `json:"previous_value"`
	PreviousDate                        Date            `json:"previous_date"`
	PreviousAnnouncementDatetime        UnixSeconds     `json:"previous_announcement_datetime"`
	Change                              float64         `json:"change"`
	ChangeFromPrevious                  float64         `json:"change_from_previous"`
	PctChangeFromPrevious               float64         `json:"pct_change_from_previous"`
	OriginalVal                         float64         `json:"original_val"`
	OriginalUnit                        string          `json:"original_unit"`
	ValMonthOverMonth                   float64         `json:"val_mom"`
	ObservationID                       string          `json:"observation_id"`
	AnnouncementDatetime                UnixSeconds     `json:"announcement_datetime"`
	AnnouncementDatetimeLocal           time.Time       `json:"announcement_datetime_local"`
	ReleaseTimeAssumed                  bool            `json:"release_time_assumed"`
	ReleaseTimeUpperBound               bool            `json:"release_time_upper_bound"`
	PublicationTimeStatus               string          `json:"publication_time_status"`
	PublicationTimePrecision            string          `json:"publication_time_precision"`
	VintageStatus                       string          `json:"vintage_status"`
	IsFirstRelease                      bool            `json:"is_first_release"`
	ObservedAtNS                        types.Time      `json:"observed_at_ns"`
	ObservedAtNSString                  string          `json:"observed_at_ns_string"`
	CollectedAtNSString                 string          `json:"collected_at_ns_string"`
	PublicationAtNS                     types.Time      `json:"publication_at_ns"`
	PublicationAtNSString               string          `json:"publication_at_ns_string"`
	AnnouncementSourceURL               string          `json:"announcement_source_url"`
	SourceArtifactSHA256                string          `json:"source_artifact_sha256"`
	CaptureTimeBasis                    string          `json:"capture_time_basis"`
	Geography                           string          `json:"geography"`
	PeriodBasis                         string          `json:"period_basis"`
	MethodologyID                       string          `json:"methodology_id"`
	SourcePeriodLabelConflict           bool            `json:"source_period_label_conflict"`
	SourceMethodologyStatus             string          `json:"source_methodology_status"`
	SelectedVintageKnownAtNS            string          `json:"selected_vintage_known_at_ns"`
	ReplayVintageVerified               bool            `json:"replay_vintage_verified"`
	OfficialPlannedReleaseDatetime      UnixSeconds     `json:"official_planned_release_datetime"`
	OfficialPlannedReleaseDatetimeLocal time.Time       `json:"official_planned_release_datetime_local"`
	OfficialActualReleaseDatetime       UnixSeconds     `json:"official_actual_release_datetime"`
	OfficialActualReleaseDatetimeLocal  time.Time       `json:"official_actual_release_datetime_local"`
	CollectedAtNS                       types.Time      `json:"collected_at_ns"`
	CollectedAtISO                      time.Time       `json:"collected_at_iso"`
	IngestionLatencyMS                  float64         `json:"ingestion_latency_ms"`
	IngestionLatencyReference           string          `json:"ingestion_latency_reference"`
	PctChange                           float64         `json:"pct_change"`
	PctChangeYearOverYear               float64         `json:"pct_change_yoy"`
	PctChangeQuarterOverQuarter         float64         `json:"pct_change_qoq"`
	PctChangeMonthOverMonth             float64         `json:"pct_change_mom"`
	PctChange12Month                    float64         `json:"pct_change_12m"`
	Revisions                           []RevisionEntry `json:"revisions"`
	OutsideRequestedWindow              bool            `json:"outside_requested_window"`
	RequestedStartDate                  Date            `json:"requested_start_date"`
	RequestedEndDate                    Date            `json:"requested_end_date"`
	ReturnedReason                      string          `json:"returned_reason"`
	StalenessDays                       int64           `json:"staleness_days"`
	CanonicalIndicator                  string          `json:"canonical_indicator"`
	RawIndicator                        string          `json:"raw_indicator"`
	RemapApplied                        bool            `json:"remap_applied"`
	RemapRuleID                         string          `json:"remap_rule_id"`
	RemapSegmentID                      string          `json:"remap_segment_id"`
}

// LatestAnnouncementsResponse contains the latest observation for each indicator.
type LatestAnnouncementsResponse struct {
	Currency   string                   `json:"currency"`
	Source     string                   `json:"source"`
	Provenance Provenance               `json:"provenance"`
	AsOf       Date                     `json:"as_of"`
	Count      uint64                   `json:"count"`
	Data       []LatestAnnouncementItem `json:"data"`
}

// LatestAnnouncementItem describes the latest available row for an indicator.
type LatestAnnouncementItem struct {
	Indicator           string                  `json:"indicator"`
	Name                string                  `json:"name"`
	Source              string                  `json:"source"`
	SourceURL           string                  `json:"source_url"`
	SourceSeriesID      string                  `json:"source_series_id"`
	SourceSeriesName    string                  `json:"source_series_name"`
	SourceLocalName     string                  `json:"source_local_name"`
	SeasonalAdjustment  string                  `json:"seasonal_adjustment"`
	PriceBasis          string                  `json:"price_basis"`
	IsProxy             bool                    `json:"is_proxy"`
	ProxyNote           string                  `json:"proxy_note"`
	Provenance          Provenance              `json:"provenance"`
	Unit                string                  `json:"unit"`
	Frequency           string                  `json:"frequency"`
	HasOfficialForecast bool                    `json:"has_official_forecast"`
	Latest              LatestAnnouncementValue `json:"latest"`
	// Previous carries the same shape as Latest and is what a caller needs to
	// compute a change without a second request.
	Previous                    LatestAnnouncementValue `json:"previous"`
	PctChangeYearOverYear       float64                 `json:"pct_change_yoy"`
	PctChangeQuarterOverQuarter float64                 `json:"pct_change_qoq"`
	PctChangeMonthOverMonth     float64                 `json:"pct_change_mom"`
	PctDiffPrev                 float64                 `json:"pct_diff_prev"`
}

// LatestAnnouncementValue contains the latest value and release timestamp.
type LatestAnnouncementValue struct {
	Date                 Date        `json:"date"`
	Val                  float64     `json:"val"`
	AnnouncementDatetime UnixSeconds `json:"announcement_datetime"`
	Source               string      `json:"source"`
	SourceURL            string      `json:"source_url"`
	// SourceURLScope says whether SourceURL points at the series, the dataset
	// or the release. OriginalVal/OriginalUnit are the publisher's own figure
	// before standardisation, which is what a reconciliation against the
	// source has to compare with.
	SourceURLScope string  `json:"source_url_scope"`
	OriginalVal    float64 `json:"original_val"`
	OriginalUnit   string  `json:"original_unit"`
}

// AnnouncementChangesScope is the filter set a changes response was produced
// under. Currencies and Indicators are empty when no filter was applied.
type AnnouncementChangesScope struct {
	Currencies []string `json:"currencies"`
	Indicators []string `json:"indicators"`
	Payload    string   `json:"payload"`
}

// AnnouncementChangesResponse contains changed announcement events.
type AnnouncementChangesResponse struct {
	Data             []AnnouncementChangeEvent `json:"data"`
	Count            uint64                    `json:"count"`
	NextCursor       string                    `json:"next_cursor"`
	HasMore          bool                      `json:"has_more"`
	RetentionSeconds uint64                    `json:"retention_seconds"`
	Scope            AnnouncementChangesScope  `json:"scope"`
}

// ReleaseDeliveryAnnouncement is the announcement row carried by a delivery
// event. The contract allows extra keys here so that a field added upstream
// keeps reaching clients; only the documented ones are decoded.
type ReleaseDeliveryAnnouncement struct {
	Date                 Date       `json:"date"`
	Val                  float64    `json:"val"`
	OriginalVal          float64    `json:"original_val"`
	OriginalUnit         string     `json:"original_unit"`
	AnnouncementDatetime types.Time `json:"announcement_datetime"`
	Source               string     `json:"source"`
	SourceURL            string     `json:"source_url"`
	SourceURLScope       string     `json:"source_url_scope"`
	SourceRelease        string     `json:"source_release"`
	ReleaseURL           string     `json:"release_url"`
	DatasetURL           string     `json:"dataset_url"`
	SeriesURL            string     `json:"series_url"`
}

// AnnouncementChangeEvent describes one announcement change notification. The
// contract types every count and duration here as integer or number, so they
// decode as float64; the *AtNS timestamps are epoch nanoseconds and change
// events are always recent, so types.Time reads them correctly.
type AnnouncementChangeEvent struct {
	EventID                                   string                      `json:"event_id"`
	Currency                                  string                      `json:"currency"`
	Indicator                                 string                      `json:"indicator"`
	RecordsWritten                            float64                     `json:"records_written"`
	Timestamp                                 types.Time                  `json:"timestamp"`
	ReleaseTimestamp                          types.Time                  `json:"release_timestamp"`
	LatestAnnouncement                        ReleaseDeliveryAnnouncement `json:"latest_announcement"`
	DeliveryMode                              string                      `json:"delivery_mode"`
	OriginDeliveryMode                        string                      `json:"origin_delivery_mode"`
	SourceFreshAtNS                           types.Time                  `json:"source_fresh_at_ns"`
	StreamReadyAtNS                           types.Time                  `json:"stream_ready_at_ns"`
	ServerSentAtNS                            types.Time                  `json:"server_sent_at_ns"`
	AcknowledgementEndpoint                   string                      `json:"acknowledgement_endpoint"`
	AgeMS                                     float64                     `json:"age_ms"`
	StaleAfterMS                              float64                     `json:"stale_after_ms"`
	Stale                                     bool                        `json:"stale"`
	LateDelivery                              bool                        `json:"late_delivery"`
	LateEventsAreDelivered                    bool                        `json:"late_events_are_delivered"`
	ScheduledReleaseAtNS                      types.Time                  `json:"scheduled_release_at_ns"`
	ScheduledToServerSendMS                   float64                     `json:"scheduled_to_server_send_ms"`
	SourceLate                                bool                        `json:"source_late"`
	SourceLateByMS                            float64                     `json:"source_late_by_ms"`
	SourceDelayProven                         bool                        `json:"source_delay_proven"`
	PlatformDeliveryAfterSourceMS             float64                     `json:"platform_delivery_after_source_ms"`
	PollingStartedAtNS                        types.Time                  `json:"polling_started_at_ns"`
	PollingStartedLagMS                       float64                     `json:"polling_started_lag_ms"`
	PollingStartedBeforeScheduledRelease      bool                        `json:"polling_started_before_scheduled_release"`
	PreFreshStalePollCount                    float64                     `json:"pre_fresh_stale_poll_count"`
	LastStaleFetchCompletedAtNS               types.Time                  `json:"last_stale_fetch_completed_at_ns"`
	FirstFreshResponseCompletedAtNS           types.Time                  `json:"first_fresh_response_completed_at_ns"`
	OfficialSourceStaleAfterScheduledRelease  bool                        `json:"official_source_stale_after_scheduled_release"`
	ScheduledToFirstFreshResponseMS           float64                     `json:"scheduled_to_first_fresh_response_ms"`
	SourceFreshnessObservationWindowStartAtNS types.Time                  `json:"source_freshness_observation_window_start_at_ns"`
	SourceFreshnessObservationWindowEndAtNS   types.Time                  `json:"source_freshness_observation_window_end_at_ns"`
	SourceFreshnessObservationWindowMS        float64                     `json:"source_freshness_observation_window_ms"`
	SourceFreshnessObservationBasis           string                      `json:"source_freshness_observation_basis"`
	SourceDelayAttribution                    string                      `json:"source_delay_attribution"`
	FXMDDeliveryAttribution                   string                      `json:"fxmd_delivery_attribution"`
	FXMDAfterSourceMS                         float64                     `json:"fxmd_after_source_ms"`
	FXMDAfterSourceSLOMet                     bool                        `json:"fxmd_after_source_slo_met"`
	CatchupSource                             string                      `json:"catchup_source"`
	CatchupDelayMS                            float64                     `json:"catchup_delay_ms"`
	RecoveryClass                             string                      `json:"recovery_class"`
	SubsecondStatus                           string                      `json:"subsecond_status"`
	SubsecondOperationalStatus                string                      `json:"subsecond_operational_status"`
	SubsecondGuaranteeActive                  bool                        `json:"subsecond_guarantee_active"`
	SubsecondContractOutcome                  string                      `json:"subsecond_contract_outcome"`
	SubsecondContractBreached                 bool                        `json:"subsecond_contract_breached"`
}

// CalendarResponse contains scheduled macroeconomic releases.
type CalendarResponse struct {
	Currency          string               `json:"currency"`
	Timezone          string               `json:"timezone"`
	RequestedTimezone string               `json:"requested_timezone"`
	Indicator         string               `json:"indicator"`
	StartDate         Date                 `json:"start_date"`
	EndDate           Date                 `json:"end_date"`
	DataQuality       DataQuality          `json:"data_quality"`
	Data              []CalendarReleaseRow `json:"data"`
}

// CalendarReleaseRow is one scheduled macroeconomic release.
type CalendarReleaseRow struct {
	AnnouncementDatetime                  UnixSeconds `json:"announcement_datetime"`
	Release                               string      `json:"release"`
	CalendarEventID                       string      `json:"calendar_event_id"`
	AnnouncementDatetimeUTC               time.Time   `json:"announcement_datetime_utc"`
	AnnouncementDatetimeLocal             time.Time   `json:"announcement_datetime_local"`
	AnnouncementDatetimeRequestedTimezone time.Time   `json:"announcement_datetime_requested_timezone"`
	ReleaseDateConfirmed                  bool        `json:"release_date_confirmed"`
	ReleaseTimeAssumed                    bool        `json:"release_time_assumed"`
	ReleaseTimeStatus                     string      `json:"release_time_status"`
	ReleaseTimeAssumption                 string      `json:"release_time_assumption"`
	Name                                  string      `json:"name"`
	Source                                string      `json:"source"`
	SourceURL                             string      `json:"source_url"`
	SourceRelease                         string      `json:"source_release"`
	ScheduleURL                           string      `json:"schedule_url"`
	ReleaseStage                          string      `json:"release_stage"`
	ReferencePeriod                       string      `json:"reference_period"`
	DatasetCodes                          any         `json:"dataset_codes"`
	Date                                  Date        `json:"date"`
	Domain                                string      `json:"domain"`
	DataCurrency                          string      `json:"data_currency"`
	EndpointFamily                        string      `json:"endpoint_family"`
	EndpointPath                          string      `json:"endpoint_path"`
	RequiresAPIKey                        bool        `json:"requires_api_key"`
	EventImportance                       string      `json:"event_importance"`
	MarketTier                            uint64      `json:"market_tier"`
	TopTierForCurrency                    bool        `json:"top_tier_for_currency"`
}

// PredictionsResponse contains model and consensus forecasts for announcements.
type PredictionsResponse struct {
	Currency         string                    `json:"currency"`
	Indicator        string                    `json:"indicator"`
	Filters          SeriesFilters             `json:"filters"`
	SelectedSeriesID string                    `json:"selected_series_id"`
	SelectedSeries   SelectedSeries            `json:"selected_series"`
	SupportedOptions map[string][]string       `json:"supported_options"`
	PredictionType   string                    `json:"prediction_type"`
	PredictionSource string                    `json:"prediction_source"`
	PreReleaseOnly   bool                      `json:"pre_release_only"`
	StartDate        Date                      `json:"start_date"`
	EndDate          Date                      `json:"end_date"`
	NextCursor       string                    `json:"next_cursor"`
	HasMore          bool                      `json:"has_more"`
	Count            uint64                    `json:"count"`
	PredictionCount  uint64                    `json:"prediction_count"`
	DataQuality      DataQuality               `json:"data_quality"`
	Data             []AnnouncementPredictions `json:"data"`
}

// AnnouncementPredictions groups forecasts for a scheduled observation.
type AnnouncementPredictions struct {
	AnnouncementID            string           `json:"announcement_id"`
	ObservationID             string           `json:"observation_id"`
	SelectedSeriesID          string           `json:"selected_series_id"`
	Currency                  string           `json:"currency"`
	Indicator                 string           `json:"indicator"`
	Date                      Date             `json:"date"`
	AnnouncementDatetime      UnixSeconds      `json:"announcement_datetime"`
	AnnouncementDatetimeLocal time.Time        `json:"announcement_datetime_local"`
	AnnouncementTiming        string           `json:"announcement_timing"`
	Predictions               []PredictionItem `json:"predictions"`
}

// PredictionProvenance describes how a forecast value was captured.
type PredictionProvenance struct {
	ValueOrigin                     string `json:"value_origin"`
	CaptureMode                     string `json:"capture_mode"`
	ArchivedInRealTimeByFXMacroData bool   `json:"archived_in_real_time_by_fxmacrodata"`
	GeneratedAtBasis                string `json:"generated_at_basis"`
	GeneratedAtPrecision            string `json:"generated_at_precision"`
	ReconstructedWithLaterData      bool   `json:"reconstructed_with_later_data"`
	VintageSelection                string `json:"vintage_selection"`
}

// PredictionItem is one forecast value.
type PredictionItem struct {
	PredictedValue              float64              `json:"predicted_value"`
	EventCompatible             bool                 `json:"event_compatible"`
	EventCompatibilityReason    string               `json:"event_compatibility_reason"`
	SourceURL                   string               `json:"source_url"`
	TargetPeriod                string               `json:"target_period"`
	PeriodBasis                 string               `json:"period_basis"`
	Unit                        string               `json:"unit"`
	Statistic                   string               `json:"statistic"`
	PublicationDate             string               `json:"publication_date"`
	PublicationDatetime         string               `json:"publication_datetime"`
	PublicationPrecision        string               `json:"publication_precision"`
	ForecastObservationID       string               `json:"forecast_observation_id"`
	SeasonalAdjustment          string               `json:"seasonal_adjustment"`
	Geography                   string               `json:"geography"`
	AnnouncementMappingVerified bool                 `json:"announcement_mapping_verified"`
	PredictionClass             string               `json:"prediction_class"`
	PredictionClassLabel        string               `json:"prediction_class_label"`
	PredictionType              string               `json:"prediction_type"`
	PredictionSource            string               `json:"prediction_source"`
	PredictionSourceLabel       string               `json:"prediction_source_label"`
	GeneratedAt                 UnixSeconds          `json:"generated_at"`
	Provenance                  PredictionProvenance `json:"provenance"`
	IsPreRelease                bool                 `json:"is_pre_release"`
	Confidence                  float64              `json:"confidence"`
	PredictionReason            string               `json:"prediction_reason"`
}

// COTProvenance describes the origin and storage contract for COT rows.
type COTProvenance struct {
	Publisher      string `json:"publisher"`
	PublisherURL   string `json:"publisher_url"`
	Storage        string `json:"storage"`
	ServedBy       string `json:"served_by"`
	TimestampField string `json:"timestamp_field"`
	ValueField     string `json:"value_field"`
}

// COTFXOverlay describes the related FX pair for COT positioning.
type COTFXOverlay struct {
	Pair string `json:"pair"`
}

// COTResponse contains CFTC positioning observations.
type COTResponse struct {
	Currency                               string         `json:"currency"`
	Instrument                             string         `json:"instrument"`
	Source                                 string         `json:"source"`
	SourceURL                              string         `json:"source_url"`
	Provenance                             COTProvenance  `json:"provenance"`
	FXOverlay                              COTFXOverlay   `json:"fx_overlay"`
	StartDate                              Date           `json:"start_date"`
	EndDate                                Date           `json:"end_date"`
	LatestAvailableDate                    Date           `json:"latest_available_date"`
	LatestAvailableAnnouncementDatetime    UnixSeconds    `json:"latest_available_announcement_datetime"`
	ExpectedNextRelease                    time.Time      `json:"expected_next_release"`
	ExpectedNextReleaseEpoch               UnixSeconds    `json:"expected_next_release_epoch"`
	LastSyncStatus                         string         `json:"last_sync_status"`
	LastSyncDatetime                       time.Time      `json:"last_sync_datetime"`
	DataLagDays                            int64          `json:"data_lag_days"`
	NextExpectedCFTCReportDate             Date           `json:"next_expected_cftc_report_date"`
	NextExpectedCFTCReleaseDate            Date           `json:"next_expected_cftc_release_date"`
	ExpectedNextReleaseHolidayAdjusted     bool           `json:"expected_next_release_holiday_adjusted"`
	ExpectedNextReleaseSource              string         `json:"expected_next_release_source"`
	ExpectedNextReleaseSourceURL           string         `json:"expected_next_release_source_url"`
	ExpectedNextReleaseScheduleStorage     string         `json:"expected_next_release_schedule_storage"`
	ExpectedNextReleaseScheduleLastUpdated time.Time      `json:"expected_next_release_schedule_last_updated"`
	RequestedWindowHasData                 bool           `json:"requested_window_has_data"`
	RequestedWindowLatestDate              Date           `json:"requested_window_latest_date"`
	RequestedWindowIncludesLatestAvailable bool           `json:"requested_window_includes_latest_available"`
	PageIncludesLatestAvailable            bool           `json:"page_includes_latest_available"`
	LastUpdated                            time.Time      `json:"last_updated"`
	DataQuality                            DataQuality    `json:"data_quality"`
	Pagination                             Pagination     `json:"pagination"`
	Data                                   []COTDataPoint `json:"data"`
}

// COTDataPoint is one CFTC positioning observation. Position sizes are
// contract counts and cannot be negative; the net fields are long minus short
// and are negative whenever shorts dominate.
type COTDataPoint struct {
	Date                       Date        `json:"date"`
	AnnouncementDatetime       UnixSeconds `json:"announcement_datetime"`
	OpenInterest               uint64      `json:"open_interest"`
	NonCommercialLong          uint64      `json:"noncommercial_long"`
	NonCommercialShort         uint64      `json:"noncommercial_short"`
	NonCommercialNet           int64       `json:"noncommercial_net"`
	NonCommercialSpread        uint64      `json:"noncommercial_spread"`
	CommercialLong             uint64      `json:"commercial_long"`
	CommercialShort            uint64      `json:"commercial_short"`
	CommercialNet              int64       `json:"commercial_net"`
	TotalReportableLong        uint64      `json:"total_reportable_long"`
	TotalReportableShort       uint64      `json:"total_reportable_short"`
	NonReportableLong          uint64      `json:"nonreportable_long"`
	NonReportableShort         uint64      `json:"nonreportable_short"`
	OpenInterestZScore         float64     `json:"open_interest_zscore"`
	NonCommercialLongZScore    float64     `json:"noncommercial_long_zscore"`
	NonCommercialShortZScore   float64     `json:"noncommercial_short_zscore"`
	NonCommercialNetZScore     float64     `json:"noncommercial_net_zscore"`
	NonCommercialSpreadZScore  float64     `json:"noncommercial_spread_zscore"`
	CommercialLongZScore       float64     `json:"commercial_long_zscore"`
	CommercialShortZScore      float64     `json:"commercial_short_zscore"`
	CommercialNetZScore        float64     `json:"commercial_net_zscore"`
	TotalReportableLongZScore  float64     `json:"total_reportable_long_zscore"`
	TotalReportableShortZScore float64     `json:"total_reportable_short_zscore"`
	NonReportableLongZScore    float64     `json:"nonreportable_long_zscore"`
	NonReportableShortZScore   float64     `json:"nonreportable_short_zscore"`
	ReportDate                 Date        `json:"report_date"`
	CutoffDate                 Date        `json:"cutoff_date"`
	ReleaseDate                Date        `json:"release_date"`
	ReleaseDatetime            time.Time   `json:"release_datetime"`
	ReleaseDateConfirmed       bool        `json:"release_date_confirmed"`
	ReleaseTimeAssumed         bool        `json:"release_time_assumed"`
	ReleaseSource              string      `json:"release_source"`
	ReleaseSourceURL           string      `json:"release_source_url"`
	HolidayAdjustedRelease     bool        `json:"holiday_adjusted_release"`
}

// CommodityResponse contains commodity observations.
type CommodityResponse struct {
	Currency            string               `json:"currency"`
	Indicator           string               `json:"indicator"`
	Source              string               `json:"source"`
	SourceURL           string               `json:"source_url"`
	Provenance          Provenance           `json:"provenance"`
	HasOfficialForecast bool                 `json:"has_official_forecast"`
	LastUpdated         time.Time            `json:"last_updated"`
	LatestAvailableDate Date                 `json:"latest_available_date"`
	DataQuality         DataQuality          `json:"data_quality"`
	StartDate           Date                 `json:"start_date"`
	EndDate             Date                 `json:"end_date"`
	Pagination          PaginationInfo       `json:"pagination"`
	Data                []CommodityDataPoint `json:"data"`
}

// CommodityDataPoint is one commodity observation.
type CommodityDataPoint struct {
	Date                   Date        `json:"date"`
	Val                    float64     `json:"val"`
	AnnouncementDatetime   UnixSeconds `json:"announcement_datetime"`
	Source                 string      `json:"source"`
	SourceURL              string      `json:"source_url"`
	SourceType             string      `json:"source_type"`
	SourcePermissionStatus string      `json:"source_permission_status"`
	SourceChartTimestamp   UnixSeconds `json:"source_chart_timestamp"`
	SourceChartTimestampMS types.Time  `json:"source_chart_timestamp_ms"`
	SourceChartPeriod      string      `json:"source_chart_period"`
	QuoteTimeStatus        string      `json:"quote_time_status"`
	SamplingMethod         string      `json:"sampling_method"`
	PublicationTimeStatus  string      `json:"publication_time_status"`
	PointInTimeSafe        bool        `json:"point_in_time_safe"`
	ProvenanceVersion      uint64      `json:"provenance_version"`
	PctChange              float64     `json:"pct_change"`
	PctChange12Month       float64     `json:"pct_change_12m"`
}

// CommodityObservation is one dated commodity value as carried by the latest
// commodities envelope.
type CommodityObservation struct {
	Date                   Date        `json:"date"`
	Val                    float64     `json:"val"`
	AnnouncementDatetime   UnixSeconds `json:"announcement_datetime"`
	Source                 string      `json:"source"`
	SourceURL              string      `json:"source_url"`
	SourceType             string      `json:"source_type"`
	SourcePermissionStatus string      `json:"source_permission_status"`
	SourceChartTimestamp   UnixSeconds `json:"source_chart_timestamp"`
	SourceChartTimestampMS types.Time  `json:"source_chart_timestamp_ms"`
	SourceChartPeriod      string      `json:"source_chart_period"`
	QuoteTimeStatus        string      `json:"quote_time_status"`
	SamplingMethod         string      `json:"sampling_method"`
	PublicationTimeStatus  string      `json:"publication_time_status"`
	PointInTimeSafe        bool        `json:"point_in_time_safe"`
	ProvenanceVersion      uint64      `json:"provenance_version"`
}

// CommodityLatestItem is the latest and previous observation for one commodity.
type CommodityLatestItem struct {
	Indicator           string               `json:"indicator"`
	Unit                string               `json:"unit"`
	Frequency           string               `json:"frequency"`
	HasOfficialForecast bool                 `json:"has_official_forecast"`
	LastUpdated         time.Time            `json:"last_updated"`
	DataQuality         DataQuality          `json:"data_quality"`
	Latest              CommodityObservation `json:"latest"`
	Previous            CommodityObservation `json:"previous"`
	PctDiffPrev         float64              `json:"pct_diff_prev"`
}

// CommoditiesLatestResponse contains the latest observation for each commodity.
type CommoditiesLatestResponse struct {
	Currency string                `json:"currency"`
	Source   string                `json:"source"`
	AsOf     Date                  `json:"as_of"`
	Count    uint64                `json:"count"`
	Data     []CommodityLatestItem `json:"data"`
}

// OfficialForwardSourceSupport describes how the official publisher makes
// forward rates available, if at all.
type OfficialForwardSourceSupport struct {
	SourceType string `json:"source_type"`
	SourceNote string `json:"source_note"`
}

// CurveAnalyticsResponse contains the selected yield-curve analytics view.
type CurveAnalyticsResponse struct {
	Currency                     string                       `json:"currency"`
	CurveFamily                  string                       `json:"curve_family"`
	View                         string                       `json:"view"`
	Metric                       string                       `json:"metric"`
	Method                       string                       `json:"method"`
	RequestedDate                Date                         `json:"requested_date"`
	AsOf                         Date                         `json:"as_of"`
	NodeCount                    uint64                       `json:"node_count"`
	SlopeCount                   uint64                       `json:"slope_count"`
	InvertedCount                uint64                       `json:"inverted_count"`
	SegmentCount                 uint64                       `json:"segment_count"`
	Sources                      []string                     `json:"sources"`
	OfficialForwardSourceSupport OfficialForwardSourceSupport `json:"official_forward_source_support"`
	DataQuality                  DataQuality                  `json:"data_quality"`
	Data                         []map[string]any             `json:"data"`
}

// RateDifferentialSources names the publisher of each leg's rate. The spot
// path reports one publisher per leg; the curve and forward paths report the
// list of curve sources each leg was built from.
type RateDifferentialSources struct {
	Base  SourceNames `json:"base"`
	Quote SourceNames `json:"quote"`
}

// RateDifferentialResponse contains historical spot or forward rate differentials.
type RateDifferentialResponse struct {
	Base                         string                       `json:"base"`
	Quote                        string                       `json:"quote"`
	RateType                     string                       `json:"rate_type"`
	MeasureRequested             string                       `json:"measure_requested"`
	MeasureUsed                  string                       `json:"measure_used"`
	BaseIndicator                string                       `json:"base_indicator"`
	QuoteIndicator               string                       `json:"quote_indicator"`
	CurveFamily                  string                       `json:"curve_family"`
	StartTenor                   string                       `json:"start_tenor"`
	EndTenor                     string                       `json:"end_tenor"`
	ForwardLabel                 string                       `json:"forward_label"`
	StartDate                    Date                         `json:"start_date"`
	EndDate                      Date                         `json:"end_date"`
	MatchedPoints                uint64                       `json:"matched_points"`
	Unit                         string                       `json:"unit"`
	LatestSpread                 float64                      `json:"latest_spread"`
	LatestSpreadBPS              float64                      `json:"latest_spread_bps"`
	LatestDifferential           float64                      `json:"latest_differential"`
	LatestDifferentialBPS        float64                      `json:"latest_differential_bps"`
	BaseLatest                   float64                      `json:"base_latest"`
	QuoteLatest                  float64                      `json:"quote_latest"`
	Sources                      RateDifferentialSources      `json:"sources"`
	OfficialForwardSourceSupport OfficialForwardSourceSupport `json:"official_forward_source_support"`
	DataQuality                  DataQuality                  `json:"data_quality"`
	Pagination                   PaginationInfo               `json:"pagination"`
	Data                         []RateDifferentialPoint      `json:"data"`
}

// RateDifferentialPoint is one matched rate differential observation.
type RateDifferentialPoint struct {
	Date                           Date        `json:"date"`
	BaseVal                        float64     `json:"base_val"`
	QuoteVal                       float64     `json:"quote_val"`
	Spread                         float64     `json:"spread"`
	SpreadBPS                      float64     `json:"spread_bps"`
	Differential                   float64     `json:"differential"`
	DifferentialBPS                float64     `json:"differential_bps"`
	BaseForwardVal                 float64     `json:"base_forward_val"`
	QuoteForwardVal                float64     `json:"quote_forward_val"`
	BaseStartVal                   float64     `json:"base_start_val"`
	BaseEndVal                     float64     `json:"base_end_val"`
	QuoteStartVal                  float64     `json:"quote_start_val"`
	QuoteEndVal                    float64     `json:"quote_end_val"`
	BaseAnnouncementDatetime       UnixSeconds `json:"base_announcement_datetime"`
	QuoteAnnouncementDatetime      UnixSeconds `json:"quote_announcement_datetime"`
	BaseStartAnnouncementDatetime  UnixSeconds `json:"base_start_announcement_datetime"`
	BaseEndAnnouncementDatetime    UnixSeconds `json:"base_end_announcement_datetime"`
	QuoteStartAnnouncementDatetime UnixSeconds `json:"quote_start_announcement_datetime"`
	QuoteEndAnnouncementDatetime   UnixSeconds `json:"quote_end_announcement_datetime"`
}

// ForexResponse contains daily FX observations and optional technical fields.
type ForexResponse struct {
	Base         string         `json:"base"`
	Quote        string         `json:"quote"`
	Source       string         `json:"source"`
	Provenance   Provenance     `json:"provenance"`
	PairMetadata PairMetadata   `json:"pair_metadata"`
	DataQuality  DataQuality    `json:"data_quality"`
	StartDate    Date           `json:"start_date"`
	EndDate      Date           `json:"end_date"`
	Pagination   PaginationInfo `json:"pagination"`
	// DatasetVersion identifies the content version the page was built from
	// and is the value the dataset_version query parameter takes to pin
	// pagination to one version.
	DatasetVersion          string           `json:"dataset_version"`
	Coverage                map[string]any   `json:"coverage"`
	Data                    []ForexDataPoint `json:"data"`
	Indicators              map[string]any   `json:"indicators"`
	DailyOHLCBasis          map[string]any   `json:"daily_ohlc_basis"`
	TechnicalIndicatorBasis map[string]any   `json:"technical_indicator_basis"`
}

// ForexDataPoint is one FX observation.
type ForexDataPoint struct {
	Date                         Date            `json:"date"`
	Val                          float64         `json:"val"`
	Open                         float64         `json:"open"`
	High                         float64         `json:"high"`
	Low                          float64         `json:"low"`
	Close                        float64         `json:"close"`
	OHLCPointCount               uint64          `json:"ohlc_point_count"`
	OHLCSourceCount              uint64          `json:"ohlc_source_count"`
	OHLCTimestampStartUTC        time.Time       `json:"ohlc_timestamp_start_utc"`
	OHLCTimestampEndUTC          time.Time       `json:"ohlc_timestamp_end_utc"`
	OHLCType                     string          `json:"ohlc_type"`
	AnnouncementDatetime         UnixSeconds     `json:"announcement_datetime"`
	ObservationDatetime          UnixSeconds     `json:"observation_datetime"`
	ObservationDatetimeISO       time.Time       `json:"observation_datetime_iso"`
	ObservationDatetimePrecision string          `json:"observation_datetime_precision"`
	SourceType                   string          `json:"source_type"`
	SourcePermissionStatus       string          `json:"source_permission_status"`
	SourceChartTimestamp         UnixSeconds     `json:"source_chart_timestamp"`
	SourceChartTimestampMS       types.Time      `json:"source_chart_timestamp_ms"`
	SourceChartPeriod            string          `json:"source_chart_period"`
	QuoteTimeStatus              string          `json:"quote_time_status"`
	SamplingMethod               string          `json:"sampling_method"`
	PublicationTimeStatus        string          `json:"publication_time_status"`
	PointInTimeSafe              bool            `json:"point_in_time_safe"`
	ProvenanceVersion            uint64          `json:"provenance_version"`
	Source                       DataPointSource `json:"source"`
	SMA20                        float64         `json:"sma_20"`
	SMA50                        float64         `json:"sma_50"`
	SMA200                       float64         `json:"sma_200"`
	EMA12                        float64         `json:"ema_12"`
	EMA20                        float64         `json:"ema_20"`
	EMA26                        float64         `json:"ema_26"`
	EMA50                        float64         `json:"ema_50"`
	EMA200                       float64         `json:"ema_200"`
	RSI14                        float64         `json:"rsi_14"`
	ATR14                        float64         `json:"atr_14"`
	ADX14                        float64         `json:"adx_14"`
	StochasticK14                float64         `json:"stoch_k_14"`
	StochasticD3                 float64         `json:"stoch_d_3"`
	WilliamsR14                  float64         `json:"williams_r_14"`
	CCI20                        float64         `json:"cci_20"`
	DonchianUpper20              float64         `json:"donchian_upper_20"`
	DonchianMiddle20             float64         `json:"donchian_middle_20"`
	DonchianLower20              float64         `json:"donchian_lower_20"`
	MACD                         float64         `json:"macd"`
	MACDSignal                   float64         `json:"macd_signal"`
	MACDHistogram                float64         `json:"macd_histogram"`
	BollingerUpper               float64         `json:"bb_upper"`
	BollingerMiddle              float64         `json:"bb_middle"`
	BollingerLower               float64         `json:"bb_lower"`
}

// FXIntradayReferenceRatesResponse contains subscriber intraday reference rates.
type FXIntradayReferenceRatesResponse struct {
	Pair      string                         `json:"pair"`
	StartTime time.Time                      `json:"start_time"`
	EndTime   time.Time                      `json:"end_time"`
	Data      []FXIntradayReferenceRatePoint `json:"data"`
}

// FXIntradayReferenceRatePoint is one intraday reference-rate observation.
type FXIntradayReferenceRatePoint struct {
	Timestamp        time.Time `json:"timestamp"`
	Price            float64   `json:"price"`
	ReferenceDate    Date      `json:"reference_date"`
	TimestampType    string    `json:"timestamp_type"`
	Source           FXSource  `json:"source"`
	SourcePair       string    `json:"source_pair"`
	DerivationMethod string    `json:"derivation_method"`
}

// FXSourcesResponse contains public FX source metadata.
type FXSourcesResponse struct {
	SourcePolicy FXSourcePolicy `json:"source_policy"`
	Sources      []FXSource     `json:"sources"`
}

// FXSourceUniversePair is one pair a source serves, either natively or through
// an anchor currency.
type FXSourceUniversePair struct {
	Pair         string `json:"pair"`
	Availability string `json:"availability"`
	Anchor       string `json:"anchor"`
}

// FXSourceUniverseEntry is the served pair universe for one FX source.
type FXSourceUniverseEntry struct {
	Source    FXSource               `json:"source"`
	PairCount uint64                 `json:"pair_count"`
	Pairs     []FXSourceUniversePair `json:"pairs"`
}

// FXSourceUniverseResponse contains the pair universe available from public FX sources.
type FXSourceUniverseResponse struct {
	SourcePolicy FXSourcePolicy          `json:"source_policy"`
	Currency     string                  `json:"currency"`
	Source       string                  `json:"source"`
	Data         []FXSourceUniverseEntry `json:"data"`
}

// FactorResponse contains one precomputed currency factor.
type FactorResponse struct {
	Currency            string            `json:"currency"`
	Factor              string            `json:"factor"`
	Methodology         string            `json:"methodology"`
	AsOf                Date              `json:"as_of"`
	Score               float64           `json:"score"`
	LevelScore          float64           `json:"level_score"`
	ImpulseScore        float64           `json:"impulse_score"`
	RateRepricingScore  float64           `json:"rate_repricing_score"`
	MacroPressureScore  float64           `json:"macro_pressure_score"`
	Label               string            `json:"label"`
	StanceContext       string            `json:"stance_context"`
	LatestAvailableDate Date              `json:"latest_available_date"`
	LastUpdated         time.Time         `json:"last_updated"`
	StartDate           Date              `json:"start_date"`
	EndDate             Date              `json:"end_date"`
	DataQuality         DataQuality       `json:"data_quality"`
	Pagination          PaginationInfo    `json:"pagination"`
	Data                []FactorDataPoint `json:"data"`
}

// FactorDataPoint is one dated factor observation.
type FactorDataPoint struct {
	Date                     Date           `json:"date"`
	Val                      float64        `json:"val"`
	Score                    float64        `json:"score"`
	LevelScore               float64        `json:"level_score"`
	ImpulseScore             float64        `json:"impulse_score"`
	RateRepricingScore       float64        `json:"rate_repricing_score"`
	MacroPressureScore       float64        `json:"macro_pressure_score"`
	Label                    string         `json:"label"`
	StanceContext            string         `json:"stance_context"`
	AnnouncementDatetime     UnixSeconds    `json:"announcement_datetime"`
	CoverageRatio            float64        `json:"coverage_ratio"`
	ComponentCount           uint64         `json:"component_count"`
	PointInTimeSafe          bool           `json:"point_in_time_safe"`
	ReleaseTimeAssumed       bool           `json:"release_time_assumed"`
	AssumedReleaseTimeInputs uint64         `json:"assumed_release_time_inputs"`
	Components               map[string]any `json:"components"`
	SourceObservations       map[string]any `json:"source_observations"`
	SourceEndpoints          []string       `json:"source_endpoints"`
}

// MarketSessionsResponse contains the FX market-session snapshot.
type MarketSessionsResponse struct {
	NowUTC      time.Time              `json:"now_utc"`
	NowUnix     UnixSeconds            `json:"now_unix"`
	IsMarketDay bool                   `json:"is_market_day"`
	Sessions    []MarketSession        `json:"sessions"`
	Overlaps    []MarketSessionOverlap `json:"overlaps"`
}

// MarketSession is one major FX market session.
type MarketSession struct {
	Name           string      `json:"name"`
	DisplayName    string      `json:"display_name"`
	Description    string      `json:"description"`
	Currencies     []string    `json:"currencies"`
	Timezone       string      `json:"timezone"`
	OpenUTC        time.Time   `json:"open_utc"`
	CloseUTC       time.Time   `json:"close_utc"`
	OpenUnix       UnixSeconds `json:"open_unix"`
	CloseUnix      UnixSeconds `json:"close_unix"`
	IsOpen         bool        `json:"is_open"`
	SecondsToOpen  int64       `json:"seconds_to_open"`
	SecondsToClose int64       `json:"seconds_to_close"`
}

// MarketSessionOverlap is a named overlap between major FX sessions.
type MarketSessionOverlap struct {
	Name           string      `json:"name"`
	Sessions       []string    `json:"sessions"`
	Description    string      `json:"description"`
	Priority       string      `json:"priority"`
	NotablePairs   []string    `json:"notable_pairs"`
	StartUTC       time.Time   `json:"start_utc"`
	EndUTC         time.Time   `json:"end_utc"`
	StartUnix      UnixSeconds `json:"start_unix"`
	EndUnix        UnixSeconds `json:"end_unix"`
	IsActive       bool        `json:"is_active"`
	SecondsToStart int64       `json:"seconds_to_start"`
	SecondsToEnd   int64       `json:"seconds_to_end"`
	DurationHours  float64     `json:"duration_hours"`
}

// RiskSentimentResponse contains global daily risk-on/risk-off observations.
type RiskSentimentResponse struct {
	StartDate           Date                           `json:"start_date"`
	EndDate             Date                           `json:"end_date"`
	LatestAvailableDate Date                           `json:"latest_available_date"`
	LastUpdated         time.Time                      `json:"last_updated"`
	DataQuality         DataQuality                    `json:"data_quality"`
	ComponentMetadata   RiskSentimentComponentMetadata `json:"component_metadata"`
	Pagination          Pagination                     `json:"pagination"`
	Data                []RiskSentimentPoint           `json:"data"`
}

// RiskSentimentComponentMetadata describes the components used in the score.
type RiskSentimentComponentMetadata struct {
	StoredComponents                      []string          `json:"stored_components"`
	ComponentCoverageFields               []string          `json:"component_coverage_fields"`
	Aliases                               map[string]string `json:"aliases"`
	UnavailableComponentsAreReportedFalse bool              `json:"unavailable_components_are_reported_false"`
}

// RiskSentimentPoint is one daily risk-sentiment observation.
type RiskSentimentPoint struct {
	Components           map[string]float64 `json:"components"`
	Val                  float64            `json:"val"`
	Date                 Date               `json:"date"`
	Regime               string             `json:"regime"`
	Score                float64            `json:"score"`
	RiskRegime           string             `json:"risk_regime"`
	Sentiment            string             `json:"sentiment"`
	ComponentCoverage    map[string]bool    `json:"component_coverage"`
	StoredComponentCount uint64             `json:"stored_component_count"`
	FinancialStressScore float64            `json:"financial_stress_score"`
	CommodityBetaScore   float64            `json:"commodity_beta_score"`
	SafeHavenScore       float64            `json:"safe_haven_score"`
}

// PressReleasesResponse contains official central-bank release items.
type PressReleasesResponse struct {
	Currency   string             `json:"currency"`
	Source     string             `json:"source"`
	SourceURL  string             `json:"source_url"`
	Limit      uint64             `json:"limit"`
	Offset     uint64             `json:"offset"`
	Count      uint64             `json:"count"`
	Pagination Pagination         `json:"pagination"`
	Data       []PressReleaseItem `json:"data"`
}

// PressReleaseItem is one official central-bank announcement or news item.
type PressReleaseItem struct {
	Title                    string         `json:"title"`
	URL                      string         `json:"url"`
	Date                     Date           `json:"date"`
	Summary                  string         `json:"summary"`
	Sentiment                float64        `json:"sentiment"`
	Topics                   []string       `json:"topics"`
	Category                 string         `json:"category"`
	Relevance                float64        `json:"relevance"`
	AISummary                string         `json:"ai_summary"`
	AIStance                 string         `json:"ai_stance"`
	AIStanceScore            float64        `json:"ai_stance_score"`
	AINextMeetingAction      string         `json:"ai_next_meeting_action"`
	AINextMeetingProbability float64        `json:"ai_next_meeting_probability"`
	AIRationale              string         `json:"ai_rationale"`
	RatePath                 RatePathSignal `json:"rate_path"`
}

// RatePathSignal is the hawkish/dovish interpretation supplied with a release.
type RatePathSignal struct {
	Score      float64 `json:"score"`
	Label      string  `json:"label"`
	BiasAction string  `json:"bias_action"`
	Confidence string  `json:"confidence"`
	RawScore   float64 `json:"raw_score"`
	// Matches stays untyped until the contract settles: the schema declares a
	// list of strings, while the endpoint's own response example carries
	// phrase/weight objects and the live endpoint sends an empty list.
	Matches any `json:"matches"`
}
