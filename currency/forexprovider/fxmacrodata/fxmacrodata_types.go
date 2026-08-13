package fxmacrodata

import (
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// APIURL is the default FXMacroData API endpoint.
	APIURL = "https://api.fxmacrodata.com/v1/"

	supportedCurrencies = "AUD,BRL,CAD,CHF,CNH,CNY,DKK,EUR,GBP,ILS,JPY,NGN,NOK,NZD,PEN,SEK,THB,USD"
)

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
	RowCount                     int  `json:"row_count"`
	AnnouncementDatetimeCount    int  `json:"announcement_datetime_count"`
	MissingAnnouncementDateCount int  `json:"missing_announcement_datetime_count"`
	PointInTimeSafe              bool `json:"point_in_time_safe"`
}

// DataQuality describes the source and freshness characteristics of a result.
type DataQuality struct {
	IsOfficial                             bool                     `json:"is_official"`
	IsProxy                                bool                     `json:"is_proxy"`
	IsFallback                             bool                     `json:"is_fallback"`
	IsStale                                bool                     `json:"is_stale"`
	HasAnnouncementDatetime                bool                     `json:"has_announcement_datetime"`
	PointInTimeSafe                        bool                     `json:"point_in_time_safe"`
	LatestAvailableDate                    *string                  `json:"latest_available_date"`
	LastUpdated                            *string                  `json:"last_updated"`
	DataLagDays                            *int                     `json:"data_lag_days"`
	SourceName                             *string                  `json:"source_name"`
	SourceType                             string                   `json:"source_type"`
	IsDerived                              bool                     `json:"is_derived"`
	RowCount                               int                      `json:"row_count"`
	AnnouncementDatetimeCount              int                      `json:"announcement_datetime_count"`
	MissingAnnouncementDateCount           int                      `json:"missing_announcement_datetime_count"`
	HasAssumedReleaseTimes                 bool                     `json:"has_assumed_release_times"`
	AssumedReleaseTimeCount                int                      `json:"assumed_release_time_count"`
	QualityScope                           string                   `json:"quality_scope"`
	StaleAfterDays                         *int                     `json:"stale_after_days"`
	RequestedWindowHasData                 *bool                    `json:"requested_window_has_data"`
	RequestedWindowLatestDate              *string                  `json:"requested_window_latest_date"`
	RequestedWindowIncludesLatestAvailable *bool                    `json:"requested_window_includes_latest_available"`
	PageIncludesLatestAvailable            *bool                    `json:"page_includes_latest_available"`
	ReturnedLatestAvailableBeforeWindow    *bool                    `json:"returned_latest_available_before_window"`
	StalenessDays                          *int                     `json:"staleness_days"`
	Reason                                 *string                  `json:"reason"`
	DatetimeField                          *string                  `json:"datetime_field"`
	DatetimePrecision                      *string                  `json:"datetime_precision"`
	DatetimeSemantics                      *string                  `json:"datetime_semantics"`
	SourceLegs                             []map[string]any         `json:"source_legs"`
	HistoricalPointInTime                  *PointInTimeCompleteness `json:"historical_point_in_time"`
}

// PaginationInfo describes a paginated API result.
type PaginationInfo struct {
	Limit                       *int  `json:"limit"`
	Offset                      int   `json:"offset"`
	ReturnedCount               int   `json:"returned_count"`
	TotalCount                  int   `json:"total_count"`
	HasMore                     bool  `json:"has_more"`
	NextOffset                  *int  `json:"next_offset"`
	PageIncludesLatestAvailable *bool `json:"page_includes_latest_available"`
}

// Pagination describes offset pagination returned by list endpoints.
type Pagination struct {
	Limit         int  `json:"limit"`
	Offset        int  `json:"offset"`
	ReturnedCount int  `json:"returned_count"`
	TotalCount    int  `json:"total_count"`
	HasMore       bool `json:"has_more"`
	NextOffset    *int `json:"next_offset"`
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
	SourceSeriesID      *string                  `json:"source_series_id"`
	SourceSeriesName    *string                  `json:"source_series_name"`
	SeasonalAdjustment  *string                  `json:"seasonal_adjustment"`
	PriceBasis          *string                  `json:"price_basis"`
	Annualization       *string                  `json:"annualization"`
	PeriodAggregation   *string                  `json:"period_aggregation"`
	SeriesVariants      []CatalogueSeriesVariant `json:"series_variants"`
	Coverage            *CatalogueCoverage       `json:"coverage"`
	SupportedOptions    map[string][]string      `json:"supported_options"`
}

// CatalogueSeriesVariant describes one selectable variant of a catalogue series.
type CatalogueSeriesVariant struct {
	SeriesID           string  `json:"series_id"`
	StorageIndicator   string  `json:"storage_indicator"`
	SourceSeriesID     *string `json:"source_series_id"`
	SourceSeriesName   *string `json:"source_series_name"`
	SeasonalAdjustment *string `json:"seasonal_adjustment"`
	PriceBasis         *string `json:"price_basis"`
	Annualization      *string `json:"annualization"`
	PeriodAggregation  *string `json:"period_aggregation"`
	FrequencySelector  *string `json:"frequency_selector"`
	Unit               string  `json:"unit"`
	Frequency          string  `json:"frequency"`
	IsDefault          bool    `json:"is_default"`
}

// CatalogueCoverage describes availability and freshness for a catalogue series.
type CatalogueCoverage struct {
	Available                  bool    `json:"available"`
	RequiresAPIKey             bool    `json:"requires_api_key"`
	EarliestAvailableDate      *string `json:"earliest_available_date"`
	LatestAvailableDate        *string `json:"latest_available_date"`
	RowCount                   int     `json:"row_count"`
	ValueAvailable             bool    `json:"value_available"`
	HasRecentData              bool    `json:"has_recent_data"`
	CoverageQuality            string  `json:"coverage_quality"`
	HistoryCoverageQuality     string  `json:"history_coverage_quality"`
	FreshnessQuality           string  `json:"freshness_quality"`
	UsableForContext           bool    `json:"usable_for_context"`
	UsableForSignal            bool    `json:"usable_for_signal"`
	DataLagDays                *int    `json:"data_lag_days"`
	StaleAfterDays             *int    `json:"stale_after_days"`
	RecentObservationCount     int     `json:"recent_observation_count"`
	HasYearOverYearTransform   *bool   `json:"has_yoy_transform"`
	HasQuarterlyTransform      *bool   `json:"has_qoq_transform"`
	HasMonthOverMonthTransform *bool   `json:"has_mom_transform"`
}

// CBTargetEntry is one central-bank target effective from a date.
type CBTargetEntry struct {
	EffectiveFrom string   `json:"effective_from"`
	Target        *float64 `json:"target"`
	Lower         *float64 `json:"lower"`
	Upper         *float64 `json:"upper"`
	Notes         *string  `json:"notes"`
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

// AnnouncementResponse contains macroeconomic announcement observations.
type AnnouncementResponse struct {
	Currency                    string                  `json:"currency"`
	Indicator                   string                  `json:"indicator"`
	Name                        *string                 `json:"name"`
	ValueName                   *string                 `json:"value_name"`
	Source                      *string                 `json:"source"`
	SourceURL                   *string                 `json:"source_url"`
	SourceSeriesID              *string                 `json:"source_series_id"`
	SourceSeriesName            *string                 `json:"source_series_name"`
	SourceLocalName             *string                 `json:"source_local_name"`
	SeasonalAdjustment          *string                 `json:"seasonal_adjustment"`
	PriceBasis                  *string                 `json:"price_basis"`
	IsProxy                     bool                    `json:"is_proxy"`
	ProxyNote                   *string                 `json:"proxy_note"`
	Provenance                  map[string]any          `json:"provenance"`
	PolicyRole                  *string                 `json:"policy_role"`
	PolicyStructure             *string                 `json:"policy_structure"`
	ComparisonCompatible        *bool                   `json:"comparison_compatible"`
	PolicyFamily                []PolicyFamilyEntry     `json:"policy_family"`
	HasOfficialForecast         bool                    `json:"has_official_forecast"`
	RequestedStartDate          *string                 `json:"requested_start_date"`
	RequestedEndDate            *string                 `json:"requested_end_date"`
	RequestedWindowHasData      *bool                   `json:"requested_window_has_data"`
	PageIncludesLatestAvailable *bool                   `json:"page_includes_latest_available"`
	StartDate                   string                  `json:"start_date"`
	EndDate                     string                  `json:"end_date"`
	EarliestAvailableDate       *string                 `json:"earliest_available_date"`
	LatestAvailableDate         *string                 `json:"latest_available_date"`
	CentralBankTarget           *CBTargetInfo           `json:"cb_target"`
	Remap                       map[string]any          `json:"remap"`
	Filters                     map[string]any          `json:"filters"`
	SelectedSeriesID            *string                 `json:"selected_series_id"`
	SelectedSeries              map[string]any          `json:"selected_series"`
	SupportedOptions            map[string][]string     `json:"supported_options"`
	DataQuality                 DataQuality             `json:"data_quality"`
	Pagination                  PaginationInfo          `json:"pagination"`
	Data                        []AnnouncementDataPoint `json:"data"`
}

// RevisionEntry is one previously published value for an observation.
type RevisionEntry struct {
	Epoch int64    `json:"epoch"`
	Val   *float64 `json:"val"`
}

// AnnouncementDataPoint is an individual macroeconomic observation.
type AnnouncementDataPoint struct {
	AnnouncementID                      *string         `json:"announcement_id"`
	Date                                string          `json:"date"`
	Val                                 *float64        `json:"val"`
	Source                              *string         `json:"source"`
	SourceURL                           *string         `json:"source_url"`
	SourceURLScope                      *string         `json:"source_url_scope"`
	PreviousValue                       *float64        `json:"previous_value"`
	PreviousDate                        *string         `json:"previous_date"`
	PreviousAnnouncementDatetime        *int64          `json:"previous_announcement_datetime"`
	Change                              *float64        `json:"change"`
	ChangeFromPrevious                  *float64        `json:"change_from_previous"`
	PctChangeFromPrevious               *float64        `json:"pct_change_from_previous"`
	OriginalVal                         *float64        `json:"original_val"`
	OriginalUnit                        *string         `json:"original_unit"`
	ValMOM                              *float64        `json:"val_mom"`
	ObservationID                       *string         `json:"observation_id"`
	AnnouncementDatetime                *int64          `json:"announcement_datetime"`
	AnnouncementDatetimeLocal           *string         `json:"announcement_datetime_local"`
	OfficialPlannedReleaseDatetime      *int64          `json:"official_planned_release_datetime"`
	OfficialPlannedReleaseDatetimeLocal *string         `json:"official_planned_release_datetime_local"`
	OfficialActualReleaseDatetime       *int64          `json:"official_actual_release_datetime"`
	OfficialActualReleaseDatetimeLocal  *string         `json:"official_actual_release_datetime_local"`
	CollectedAtNS                       *int64          `json:"collected_at_ns"`
	CollectedAtISO                      *string         `json:"collected_at_iso"`
	IngestionLatencyMS                  *float64        `json:"ingestion_latency_ms"`
	IngestionLatencyReference           *string         `json:"ingestion_latency_reference"`
	PctChange                           *float64        `json:"pct_change"`
	PctChangeYearOverYear               *float64        `json:"pct_change_yoy"`
	PctChangeQuarterOverQuarter         *float64        `json:"pct_change_qoq"`
	PctChangeMonthOverMonth             *float64        `json:"pct_change_mom"`
	PctChange12M                        *float64        `json:"pct_change_12m"`
	Revisions                           []RevisionEntry `json:"revisions"`
	OutsideRequestedWindow              *bool           `json:"outside_requested_window"`
	RequestedStartDate                  *string         `json:"requested_start_date"`
	RequestedEndDate                    *string         `json:"requested_end_date"`
	ReturnedReason                      *string         `json:"returned_reason"`
	StalenessDays                       *int            `json:"staleness_days"`
	CanonicalIndicator                  *string         `json:"canonical_indicator"`
	RawIndicator                        *string         `json:"raw_indicator"`
	RemapApplied                        *bool           `json:"remap_applied"`
	RemapRuleID                         *string         `json:"remap_rule_id"`
	RemapSegmentID                      *string         `json:"remap_segment_id"`
}

// LatestAnnouncementsResponse contains the latest observation for each indicator.
type LatestAnnouncementsResponse struct {
	Currency   string                   `json:"currency"`
	Source     string                   `json:"source"`
	Provenance map[string]any           `json:"provenance"`
	AsOf       string                   `json:"as_of"`
	Count      int                      `json:"count"`
	Data       []LatestAnnouncementItem `json:"data"`
}

// LatestAnnouncementItem describes the latest available row for an indicator.
type LatestAnnouncementItem struct {
	Indicator           string                  `json:"indicator"`
	Name                string                  `json:"name"`
	Source              string                  `json:"source"`
	SourceURL           *string                 `json:"source_url"`
	SourceSeriesID      *string                 `json:"source_series_id"`
	SourceSeriesName    *string                 `json:"source_series_name"`
	SourceLocalName     *string                 `json:"source_local_name"`
	SeasonalAdjustment  *string                 `json:"seasonal_adjustment"`
	PriceBasis          *string                 `json:"price_basis"`
	IsProxy             bool                    `json:"is_proxy"`
	ProxyNote           *string                 `json:"proxy_note"`
	Provenance          map[string]any          `json:"provenance"`
	Unit                string                  `json:"unit"`
	Frequency           string                  `json:"frequency"`
	HasOfficialForecast bool                    `json:"has_official_forecast"`
	Latest              LatestAnnouncementValue `json:"latest"`
}

// LatestAnnouncementValue contains the latest value and release timestamp.
type LatestAnnouncementValue struct {
	Date                 string   `json:"date"`
	Val                  *float64 `json:"val"`
	AnnouncementDatetime *int64   `json:"announcement_datetime"`
	Source               *string  `json:"source"`
	SourceURL            *string  `json:"source_url"`
}

// AnnouncementChangesResponse contains changed announcement events.
type AnnouncementChangesResponse struct {
	Data             []AnnouncementChangeEvent `json:"data"`
	Count            int                       `json:"count"`
	NextCursor       string                    `json:"next_cursor"`
	HasMore          bool                      `json:"has_more"`
	RetentionSeconds int                       `json:"retention_seconds"`
	Scope            map[string]any            `json:"scope"`
}

// AnnouncementChangeEvent describes one announcement change notification.
type AnnouncementChangeEvent struct {
	EventID            string         `json:"event_id"`
	Currency           string         `json:"currency"`
	Indicator          string         `json:"indicator"`
	RecordsWritten     *int           `json:"records_written"`
	Timestamp          *int64         `json:"timestamp"`
	LatestAnnouncement map[string]any `json:"latest_announcement"`
}

// CalendarResponse contains scheduled macroeconomic releases.
type CalendarResponse struct {
	Currency          string               `json:"currency"`
	Timezone          *string              `json:"timezone"`
	RequestedTimezone *string              `json:"requested_timezone"`
	Indicator         *string              `json:"indicator"`
	StartDate         *string              `json:"start_date"`
	EndDate           *string              `json:"end_date"`
	DataQuality       DataQuality          `json:"data_quality"`
	Data              []CalendarReleaseRow `json:"data"`
}

// CalendarReleaseRow is one scheduled macroeconomic release.
type CalendarReleaseRow struct {
	AnnouncementDatetime                  int64   `json:"announcement_datetime"`
	Release                               string  `json:"release"`
	AnnouncementDatetimeUTC               *string `json:"announcement_datetime_utc"`
	AnnouncementDatetimeLocal             *string `json:"announcement_datetime_local"`
	AnnouncementDatetimeRequestedTimezone *string `json:"announcement_datetime_requested_timezone"`
	ReleaseDateConfirmed                  *bool   `json:"release_date_confirmed"`
	ReleaseTimeAssumed                    *bool   `json:"release_time_assumed"`
	ReleaseTimeStatus                     *string `json:"release_time_status"`
	ReleaseTimeAssumption                 *string `json:"release_time_assumption"`
	Name                                  *string `json:"name"`
	Source                                *string `json:"source"`
	SourceURL                             *string `json:"source_url"`
	SourceRelease                         *string `json:"source_release"`
	ScheduleURL                           *string `json:"schedule_url"`
	ReleaseStage                          *string `json:"release_stage"`
	ReferencePeriod                       *string `json:"reference_period"`
	DatasetCodes                          any     `json:"dataset_codes"`
	Date                                  *string `json:"date"`
	Domain                                *string `json:"domain"`
	DataCurrency                          *string `json:"data_currency"`
	EndpointFamily                        *string `json:"endpoint_family"`
	EndpointPath                          *string `json:"endpoint_path"`
	RequiresAPIKey                        *bool   `json:"requires_api_key"`
	EventImportance                       *string `json:"event_importance"`
	MarketTier                            *int    `json:"market_tier"`
	TopTierForCurrency                    *bool   `json:"top_tier_for_currency"`
}

// PredictionsResponse contains model and consensus forecasts for announcements.
type PredictionsResponse struct {
	Currency         string                    `json:"currency"`
	Indicator        *string                   `json:"indicator"`
	Filters          map[string]*string        `json:"filters"`
	SelectedSeriesID *string                   `json:"selected_series_id"`
	SelectedSeries   map[string]any            `json:"selected_series"`
	SupportedOptions map[string][]string       `json:"supported_options"`
	PredictionType   *string                   `json:"prediction_type"`
	PredictionSource *string                   `json:"prediction_source"`
	PreReleaseOnly   bool                      `json:"pre_release_only"`
	StartDate        *string                   `json:"start_date"`
	EndDate          *string                   `json:"end_date"`
	NextCursor       *string                   `json:"next_cursor"`
	HasMore          bool                      `json:"has_more"`
	Count            int                       `json:"count"`
	PredictionCount  int                       `json:"prediction_count"`
	DataQuality      DataQuality               `json:"data_quality"`
	Data             []AnnouncementPredictions `json:"data"`
}

// AnnouncementPredictions groups forecasts for a scheduled observation.
type AnnouncementPredictions struct {
	AnnouncementID            string           `json:"announcement_id"`
	ObservationID             *string          `json:"observation_id"`
	SelectedSeriesID          *string          `json:"selected_series_id"`
	Currency                  string           `json:"currency"`
	Indicator                 string           `json:"indicator"`
	Date                      string           `json:"date"`
	AnnouncementDatetime      int64            `json:"announcement_datetime"`
	AnnouncementDatetimeLocal *string          `json:"announcement_datetime_local"`
	AnnouncementTiming        *string          `json:"announcement_timing"`
	Predictions               []PredictionItem `json:"predictions"`
}

// PredictionItem is one forecast value.
type PredictionItem struct {
	PredictedValue        float64  `json:"predicted_value"`
	PredictionType        *string  `json:"prediction_type"`
	PredictionSource      *string  `json:"prediction_source"`
	PredictionSourceLabel *string  `json:"prediction_source_label"`
	GeneratedAt           *int64   `json:"generated_at"`
	IsPreRelease          *bool    `json:"is_pre_release"`
	Confidence            *float64 `json:"confidence"`
	PredictionReason      *string  `json:"prediction_reason"`
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

// COTPaginationInfo describes pagination returned by the COT endpoint.
type COTPaginationInfo struct {
	Limit         int  `json:"limit"`
	Offset        int  `json:"offset"`
	ReturnedCount int  `json:"returned_count"`
	TotalCount    int  `json:"total_count"`
	HasMore       bool `json:"has_more"`
	NextOffset    *int `json:"next_offset"`
}

// COTResponse contains CFTC positioning observations.
type COTResponse struct {
	Currency                               string            `json:"currency"`
	Instrument                             string            `json:"instrument"`
	Source                                 string            `json:"source"`
	SourceURL                              string            `json:"source_url"`
	Provenance                             COTProvenance     `json:"provenance"`
	FXOverlay                              COTFXOverlay      `json:"fx_overlay"`
	StartDate                              string            `json:"start_date"`
	EndDate                                string            `json:"end_date"`
	LatestAvailableDate                    *string           `json:"latest_available_date"`
	LatestAvailableAnnouncementDatetime    *int64            `json:"latest_available_announcement_datetime"`
	ExpectedNextRelease                    *string           `json:"expected_next_release"`
	ExpectedNextReleaseEpoch               *int64            `json:"expected_next_release_epoch"`
	LastSyncStatus                         string            `json:"last_sync_status"`
	LastSyncDatetime                       *string           `json:"last_sync_datetime"`
	DataLagDays                            *int              `json:"data_lag_days"`
	NextExpectedCFTCReportDate             *string           `json:"next_expected_cftc_report_date"`
	NextExpectedCFTCReleaseDate            *string           `json:"next_expected_cftc_release_date"`
	ExpectedNextReleaseHolidayAdjusted     *bool             `json:"expected_next_release_holiday_adjusted"`
	ExpectedNextReleaseSource              *string           `json:"expected_next_release_source"`
	ExpectedNextReleaseSourceURL           *string           `json:"expected_next_release_source_url"`
	ExpectedNextReleaseScheduleStorage     *string           `json:"expected_next_release_schedule_storage"`
	ExpectedNextReleaseScheduleLastUpdated *string           `json:"expected_next_release_schedule_last_updated"`
	RequestedWindowHasData                 *bool             `json:"requested_window_has_data"`
	RequestedWindowLatestDate              *string           `json:"requested_window_latest_date"`
	RequestedWindowIncludesLatestAvailable *bool             `json:"requested_window_includes_latest_available"`
	PageIncludesLatestAvailable            *bool             `json:"page_includes_latest_available"`
	LastUpdated                            *string           `json:"last_updated"`
	DataQuality                            DataQuality       `json:"data_quality"`
	Pagination                             COTPaginationInfo `json:"pagination"`
	Data                                   []COTDataPoint    `json:"data"`
}

// COTDataPoint is one CFTC positioning observation.
type COTDataPoint struct {
	Date                       string   `json:"date"`
	AnnouncementDatetime       *int64   `json:"announcement_datetime"`
	OpenInterest               *int64   `json:"open_interest"`
	NonCommercialLong          *int64   `json:"noncommercial_long"`
	NonCommercialShort         *int64   `json:"noncommercial_short"`
	NonCommercialNet           *int64   `json:"noncommercial_net"`
	NonCommercialSpread        *int64   `json:"noncommercial_spread"`
	CommercialLong             *int64   `json:"commercial_long"`
	CommercialShort            *int64   `json:"commercial_short"`
	CommercialNet              *int64   `json:"commercial_net"`
	TotalReportableLong        *int64   `json:"total_reportable_long"`
	TotalReportableShort       *int64   `json:"total_reportable_short"`
	NonReportableLong          *int64   `json:"nonreportable_long"`
	NonReportableShort         *int64   `json:"nonreportable_short"`
	OpenInterestZScore         *float64 `json:"open_interest_zscore"`
	NonCommercialLongZScore    *float64 `json:"noncommercial_long_zscore"`
	NonCommercialShortZScore   *float64 `json:"noncommercial_short_zscore"`
	NonCommercialNetZScore     *float64 `json:"noncommercial_net_zscore"`
	NonCommercialSpreadZScore  *float64 `json:"noncommercial_spread_zscore"`
	CommercialLongZScore       *float64 `json:"commercial_long_zscore"`
	CommercialShortZScore      *float64 `json:"commercial_short_zscore"`
	CommercialNetZScore        *float64 `json:"commercial_net_zscore"`
	TotalReportableLongZScore  *float64 `json:"total_reportable_long_zscore"`
	TotalReportableShortZScore *float64 `json:"total_reportable_short_zscore"`
	NonReportableLongZScore    *float64 `json:"nonreportable_long_zscore"`
	NonReportableShortZScore   *float64 `json:"nonreportable_short_zscore"`
	ReportDate                 *string  `json:"report_date"`
	CutoffDate                 *string  `json:"cutoff_date"`
	ReleaseDate                *string  `json:"release_date"`
	ReleaseDatetime            *string  `json:"release_datetime"`
	ReleaseDateConfirmed       *bool    `json:"release_date_confirmed"`
	ReleaseSource              *string  `json:"release_source"`
	ReleaseSourceURL           *string  `json:"release_source_url"`
	HolidayAdjustedRelease     *bool    `json:"holiday_adjusted_release"`
}

// CommodityResponse contains commodity observations.
type CommodityResponse struct {
	Currency            string               `json:"currency"`
	Indicator           string               `json:"indicator"`
	Source              *string              `json:"source"`
	SourceURL           *string              `json:"source_url"`
	Provenance          map[string]any       `json:"provenance"`
	HasOfficialForecast bool                 `json:"has_official_forecast"`
	LastUpdated         *string              `json:"last_updated"`
	LatestAvailableDate *string              `json:"latest_available_date"`
	DataQuality         DataQuality          `json:"data_quality"`
	StartDate           string               `json:"start_date"`
	EndDate             string               `json:"end_date"`
	Pagination          map[string]any       `json:"pagination"`
	Data                []CommodityDataPoint `json:"data"`
}

// CommodityDataPoint is one commodity observation.
type CommodityDataPoint struct {
	Date                 string   `json:"date"`
	Val                  *float64 `json:"val"`
	AnnouncementDatetime *int64   `json:"announcement_datetime"`
	PctChange            *float64 `json:"pct_change"`
	PctChange12M         *float64 `json:"pct_change_12m"`
}

// CommoditiesLatestResponse contains the documented dynamic latest-value map.
type CommoditiesLatestResponse map[string]any

// CurveAnalyticsResponse contains the selected yield-curve analytics view.
type CurveAnalyticsResponse struct {
	Currency                     string           `json:"currency"`
	CurveFamily                  string           `json:"curve_family"`
	View                         string           `json:"view"`
	Metric                       *string          `json:"metric"`
	Method                       *string          `json:"method"`
	RequestedDate                string           `json:"requested_date"`
	AsOf                         *string          `json:"as_of"`
	NodeCount                    int              `json:"node_count"`
	SlopeCount                   *int             `json:"slope_count"`
	InvertedCount                *int             `json:"inverted_count"`
	SegmentCount                 *int             `json:"segment_count"`
	Sources                      []string         `json:"sources"`
	OfficialForwardSourceSupport map[string]any   `json:"official_forward_source_support"`
	DataQuality                  DataQuality      `json:"data_quality"`
	Data                         []map[string]any `json:"data"`
}

// RateDifferentialResponse contains historical spot or forward rate differentials.
type RateDifferentialResponse struct {
	Base                         string                  `json:"base"`
	Quote                        string                  `json:"quote"`
	RateType                     string                  `json:"rate_type"`
	MeasureRequested             string                  `json:"measure_requested"`
	MeasureUsed                  string                  `json:"measure_used"`
	BaseIndicator                *string                 `json:"base_indicator"`
	QuoteIndicator               *string                 `json:"quote_indicator"`
	CurveFamily                  *string                 `json:"curve_family"`
	StartTenor                   *string                 `json:"start_tenor"`
	EndTenor                     *string                 `json:"end_tenor"`
	ForwardLabel                 *string                 `json:"forward_label"`
	StartDate                    string                  `json:"start_date"`
	EndDate                      string                  `json:"end_date"`
	MatchedPoints                int                     `json:"matched_points"`
	Unit                         string                  `json:"unit"`
	LatestSpread                 *float64                `json:"latest_spread"`
	LatestSpreadBPS              *float64                `json:"latest_spread_bps"`
	LatestDifferential           *float64                `json:"latest_differential"`
	LatestDifferentialBPS        *float64                `json:"latest_differential_bps"`
	BaseLatest                   *float64                `json:"base_latest"`
	QuoteLatest                  *float64                `json:"quote_latest"`
	Sources                      map[string]any          `json:"sources"`
	OfficialForwardSourceSupport map[string]any          `json:"official_forward_source_support"`
	DataQuality                  DataQuality             `json:"data_quality"`
	Pagination                   map[string]any          `json:"pagination"`
	Data                         []RateDifferentialPoint `json:"data"`
}

// RateDifferentialPoint is one matched rate differential observation.
type RateDifferentialPoint struct {
	Date                           string   `json:"date"`
	BaseVal                        *float64 `json:"base_val"`
	QuoteVal                       *float64 `json:"quote_val"`
	Spread                         *float64 `json:"spread"`
	SpreadBPS                      *float64 `json:"spread_bps"`
	Differential                   *float64 `json:"differential"`
	DifferentialBPS                *float64 `json:"differential_bps"`
	BaseForwardVal                 *float64 `json:"base_forward_val"`
	QuoteForwardVal                *float64 `json:"quote_forward_val"`
	BaseStartVal                   *float64 `json:"base_start_val"`
	BaseEndVal                     *float64 `json:"base_end_val"`
	QuoteStartVal                  *float64 `json:"quote_start_val"`
	QuoteEndVal                    *float64 `json:"quote_end_val"`
	BaseAnnouncementDatetime       *int64   `json:"base_announcement_datetime"`
	QuoteAnnouncementDatetime      *int64   `json:"quote_announcement_datetime"`
	BaseStartAnnouncementDatetime  *int64   `json:"base_start_announcement_datetime"`
	BaseEndAnnouncementDatetime    *int64   `json:"base_end_announcement_datetime"`
	QuoteStartAnnouncementDatetime *int64   `json:"quote_start_announcement_datetime"`
	QuoteEndAnnouncementDatetime   *int64   `json:"quote_end_announcement_datetime"`
}

// ForexResponse contains daily FX observations and optional technical fields.
type ForexResponse struct {
	Base                    string           `json:"base"`
	Quote                   string           `json:"quote"`
	Source                  string           `json:"source"`
	Provenance              map[string]any   `json:"provenance"`
	PairMetadata            map[string]any   `json:"pair_metadata"`
	DataQuality             DataQuality      `json:"data_quality"`
	StartDate               string           `json:"start_date"`
	EndDate                 string           `json:"end_date"`
	Pagination              map[string]any   `json:"pagination"`
	Data                    []ForexDataPoint `json:"data"`
	Indicators              map[string]any   `json:"indicators"`
	DailyOHLCBasis          map[string]any   `json:"daily_ohlc_basis"`
	TechnicalIndicatorBasis map[string]any   `json:"technical_indicator_basis"`
}

// ForexDataPoint is one FX observation.
type ForexDataPoint struct {
	Date                         string         `json:"date"`
	Val                          *float64       `json:"val"`
	Open                         *float64       `json:"open"`
	High                         *float64       `json:"high"`
	Low                          *float64       `json:"low"`
	Close                        *float64       `json:"close"`
	OHLCPointCount               *int           `json:"ohlc_point_count"`
	OHLCSourceCount              *int           `json:"ohlc_source_count"`
	OHLCTimestampStartUTC        *string        `json:"ohlc_timestamp_start_utc"`
	OHLCTimestampEndUTC          *string        `json:"ohlc_timestamp_end_utc"`
	OHLCType                     *string        `json:"ohlc_type"`
	AnnouncementDatetime         *int64         `json:"announcement_datetime"`
	ObservationDatetime          *int64         `json:"observation_datetime"`
	ObservationDatetimeISO       *string        `json:"observation_datetime_iso"`
	ObservationDatetimePrecision *string        `json:"observation_datetime_precision"`
	Source                       map[string]any `json:"source"`
	SMA20                        *float64       `json:"sma_20"`
	SMA50                        *float64       `json:"sma_50"`
	SMA200                       *float64       `json:"sma_200"`
	EMA12                        *float64       `json:"ema_12"`
	EMA20                        *float64       `json:"ema_20"`
	EMA26                        *float64       `json:"ema_26"`
	EMA50                        *float64       `json:"ema_50"`
	EMA200                       *float64       `json:"ema_200"`
	RSI14                        *float64       `json:"rsi_14"`
	ATR14                        *float64       `json:"atr_14"`
	ADX14                        *float64       `json:"adx_14"`
	StochasticK14                *float64       `json:"stoch_k_14"`
	StochasticD3                 *float64       `json:"stoch_d_3"`
	WilliamsR14                  *float64       `json:"williams_r_14"`
	CCI20                        *float64       `json:"cci_20"`
	DonchianUpper20              *float64       `json:"donchian_upper_20"`
	DonchianMiddle20             *float64       `json:"donchian_middle_20"`
	DonchianLower20              *float64       `json:"donchian_lower_20"`
	MACD                         *float64       `json:"macd"`
	MACDSignal                   *float64       `json:"macd_signal"`
	MACDHistogram                *float64       `json:"macd_histogram"`
	BollingerUpper               *float64       `json:"bb_upper"`
	BollingerMiddle              *float64       `json:"bb_middle"`
	BollingerLower               *float64       `json:"bb_lower"`
}

// FxIntradayReferenceRatesResponse contains subscriber intraday reference rates.
type FxIntradayReferenceRatesResponse struct {
	Pair      string                         `json:"pair"`
	StartTime string                         `json:"start_time"`
	EndTime   string                         `json:"end_time"`
	Data      []FxIntradayReferenceRatePoint `json:"data"`
}

// FxIntradayReferenceRatePoint is one intraday reference-rate observation.
type FxIntradayReferenceRatePoint struct {
	Timestamp        string         `json:"timestamp"`
	Price            float64        `json:"price"`
	ReferenceDate    *string        `json:"reference_date"`
	TimestampType    *string        `json:"timestamp_type"`
	Source           map[string]any `json:"source"`
	SourcePair       *string        `json:"source_pair"`
	DerivationMethod *string        `json:"derivation_method"`
}

// FxSourcesResponse contains public FX source metadata.
type FxSourcesResponse struct {
	SourcePolicy map[string]any   `json:"source_policy"`
	Sources      []map[string]any `json:"sources"`
}

// FxSourceUniverseResponse contains the pair universe available from public FX sources.
type FxSourceUniverseResponse struct {
	SourcePolicy map[string]any   `json:"source_policy"`
	Currency     *string          `json:"currency"`
	Source       *string          `json:"source"`
	Data         []map[string]any `json:"data"`
}

// FactorResponse contains one precomputed currency factor.
type FactorResponse struct {
	Currency            string            `json:"currency"`
	Factor              string            `json:"factor"`
	Methodology         string            `json:"methodology"`
	AsOf                *string           `json:"as_of"`
	Score               *float64          `json:"score"`
	LevelScore          *float64          `json:"level_score"`
	ImpulseScore        *float64          `json:"impulse_score"`
	RateRepricingScore  *float64          `json:"rate_repricing_score"`
	MacroPressureScore  *float64          `json:"macro_pressure_score"`
	Label               *string           `json:"label"`
	StanceContext       *string           `json:"stance_context"`
	LatestAvailableDate *string           `json:"latest_available_date"`
	LastUpdated         *string           `json:"last_updated"`
	StartDate           string            `json:"start_date"`
	EndDate             string            `json:"end_date"`
	DataQuality         DataQuality       `json:"data_quality"`
	Pagination          map[string]any    `json:"pagination"`
	Data                []FactorDataPoint `json:"data"`
}

// FactorDataPoint is one dated factor observation.
type FactorDataPoint struct {
	Date                 string         `json:"date"`
	Val                  *float64       `json:"val"`
	Score                *float64       `json:"score"`
	LevelScore           *float64       `json:"level_score"`
	ImpulseScore         *float64       `json:"impulse_score"`
	RateRepricingScore   *float64       `json:"rate_repricing_score"`
	MacroPressureScore   *float64       `json:"macro_pressure_score"`
	Label                *string        `json:"label"`
	StanceContext        *string        `json:"stance_context"`
	AnnouncementDatetime *int64         `json:"announcement_datetime"`
	CoverageRatio        *float64       `json:"coverage_ratio"`
	ComponentCount       *int           `json:"component_count"`
	PointInTimeSafe      *bool          `json:"point_in_time_safe"`
	Components           map[string]any `json:"components"`
	SourceObservations   map[string]any `json:"source_observations"`
	SourceEndpoints      []string       `json:"source_endpoints"`
}

// MarketSessionsResponse contains the FX market-session snapshot.
type MarketSessionsResponse struct {
	NowUTC      string                 `json:"now_utc"`
	NowUnix     int64                  `json:"now_unix"`
	IsMarketDay bool                   `json:"is_market_day"`
	Sessions    []MarketSession        `json:"sessions"`
	Overlaps    []MarketSessionOverlap `json:"overlaps"`
}

// MarketSession is one major FX market session.
type MarketSession struct {
	Name           string   `json:"name"`
	DisplayName    string   `json:"display_name"`
	Description    string   `json:"description"`
	Currencies     []string `json:"currencies"`
	Timezone       string   `json:"timezone"`
	OpenUTC        string   `json:"open_utc"`
	CloseUTC       string   `json:"close_utc"`
	OpenUnix       int64    `json:"open_unix"`
	CloseUnix      int64    `json:"close_unix"`
	IsOpen         bool     `json:"is_open"`
	SecondsToOpen  *int64   `json:"seconds_to_open"`
	SecondsToClose *int64   `json:"seconds_to_close"`
}

// MarketSessionOverlap is a named overlap between major FX sessions.
type MarketSessionOverlap struct {
	Name           string   `json:"name"`
	Sessions       []string `json:"sessions"`
	Description    string   `json:"description"`
	Priority       string   `json:"priority"`
	NotablePairs   []string `json:"notable_pairs"`
	StartUTC       string   `json:"start_utc"`
	EndUTC         string   `json:"end_utc"`
	StartUnix      int64    `json:"start_unix"`
	EndUnix        int64    `json:"end_unix"`
	IsActive       bool     `json:"is_active"`
	SecondsToStart *int64   `json:"seconds_to_start"`
	SecondsToEnd   *int64   `json:"seconds_to_end"`
	DurationHours  float64  `json:"duration_hours"`
}

// RiskSentimentResponse contains global daily risk-on/risk-off observations.
type RiskSentimentResponse struct {
	StartDate           string                         `json:"start_date"`
	EndDate             string                         `json:"end_date"`
	LatestAvailableDate string                         `json:"latest_available_date"`
	LastUpdated         string                         `json:"last_updated"`
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
	Date                 string             `json:"date"`
	Regime               string             `json:"regime"`
	Score                float64            `json:"score"`
	RiskRegime           string             `json:"risk_regime"`
	Sentiment            string             `json:"sentiment"`
	ComponentCoverage    map[string]bool    `json:"component_coverage"`
	StoredComponentCount int                `json:"stored_component_count"`
	FinancialStressScore *float64           `json:"financial_stress_score"`
	CommodityBetaScore   *float64           `json:"commodity_beta_score"`
	SafeHavenScore       *float64           `json:"safe_haven_score"`
}

// PressReleasesResponse contains official central-bank release items.
type PressReleasesResponse struct {
	Currency   string             `json:"currency"`
	Source     string             `json:"source"`
	SourceURL  string             `json:"source_url"`
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`
	Count      int                `json:"count"`
	Pagination Pagination         `json:"pagination"`
	Data       []PressReleaseItem `json:"data"`
}

// PressReleaseItem is one official central-bank announcement or news item.
type PressReleaseItem struct {
	Title                    string         `json:"title"`
	URL                      string         `json:"url"`
	Date                     string         `json:"date"`
	Summary                  string         `json:"summary"`
	Sentiment                float64        `json:"sentiment"`
	Topics                   []string       `json:"topics"`
	Category                 string         `json:"category"`
	Relevance                float64        `json:"relevance"`
	AISummary                string         `json:"ai_summary"`
	AIStance                 string         `json:"ai_stance"`
	AIStanceScore            *float64       `json:"ai_stance_score"`
	AINextMeetingAction      string         `json:"ai_next_meeting_action"`
	AINextMeetingProbability *float64       `json:"ai_next_meeting_probability"`
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
	Matches    any     `json:"matches"`
}
