package fxmacrodata

import (
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// APIURL is the default FXMacroData API endpoint.
	APIURL = "https://fxmacrodata.com/api/v1/"

	supportedCurrencies = "AUD,BRL,CAD,CHF,CNH,CNY,DKK,EUR,GBP,ILS,JPY,NGN,NOK,NZD,PEN,SEK,THB,USD"
)

// FXMacroData is an FXMacroData foreign exchange and macro data provider.
type FXMacroData struct {
	base.Base
	Requester *request.Requester
	APIURL    string
}

type forexResponse struct {
	Data []ForexRate `json:"data"`
}

// ForexRate is an FX quote returned by the FXMacroData forex endpoint.
type ForexRate struct {
	Val float64 `json:"val"`
}

// ServiceStatusResponse represents a public FXMacroData service status response.
type ServiceStatusResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// DataQuality describes the source and freshness characteristics of a result.
type DataQuality struct {
	IsOfficial                   bool            `json:"is_official"`
	IsProxy                      bool            `json:"is_proxy"`
	IsFallback                   bool            `json:"is_fallback"`
	IsStale                      bool            `json:"is_stale"`
	HasAnnouncementDatetime      bool            `json:"has_announcement_datetime"`
	PointInTimeSafe              bool            `json:"point_in_time_safe"`
	LatestAvailableDate          json.RawMessage `json:"latest_available_date"`
	LastUpdated                  json.RawMessage `json:"last_updated"`
	DataLagDays                  json.RawMessage `json:"data_lag_days"`
	SourceName                   json.RawMessage `json:"source_name"`
	SourceType                   string          `json:"source_type"`
	IsDerived                    bool            `json:"is_derived"`
	RowCount                     int             `json:"row_count"`
	AnnouncementDatetimeCount    int             `json:"announcement_datetime_count"`
	MissingAnnouncementDateCount int             `json:"missing_announcement_datetime_count"`
	QualityScope                 string          `json:"quality_scope"`
	StaleAfterDays               json.RawMessage `json:"stale_after_days"`
}

// PaginationInfo describes a paginated API result.
type PaginationInfo struct {
	Limit                       json.RawMessage `json:"limit"`
	Offset                      int             `json:"offset"`
	ReturnedCount               int             `json:"returned_count"`
	TotalCount                  int             `json:"total_count"`
	HasMore                     bool            `json:"has_more"`
	NextOffset                  json.RawMessage `json:"next_offset"`
	PageIncludesLatestAvailable json.RawMessage `json:"page_includes_latest_available"`
}

// DataCatalogueResponse contains the provider's available data catalogue.
// The OpenAPI contract does not specify the nested catalogue shape, so its
// evolving data payload is retained verbatim rather than decoded into maps.
type DataCatalogueResponse struct {
	Currency   string          `json:"currency"`
	Indicators json.RawMessage `json:"indicators"`
	Data       json.RawMessage `json:"data"`
}

// AnnouncementResponse contains macroeconomic announcement observations.
type AnnouncementResponse struct {
	Currency           string                  `json:"currency"`
	Indicator          string                  `json:"indicator"`
	Name               json.RawMessage         `json:"name"`
	ValueName          json.RawMessage         `json:"value_name"`
	Source             json.RawMessage         `json:"source"`
	SourceURL          json.RawMessage         `json:"source_url"`
	SourceSeriesID     json.RawMessage         `json:"source_series_id"`
	SeasonalAdjustment json.RawMessage         `json:"seasonal_adjustment"`
	IsProxy            bool                    `json:"is_proxy"`
	Provenance         json.RawMessage         `json:"provenance"`
	StartDate          string                  `json:"start_date"`
	EndDate            string                  `json:"end_date"`
	DataQuality        DataQuality             `json:"data_quality"`
	Pagination         PaginationInfo          `json:"pagination"`
	Data               []AnnouncementDataPoint `json:"data"`
}

// AnnouncementDataPoint is an individual macroeconomic observation.
type AnnouncementDataPoint struct {
	AnnouncementID                 json.RawMessage `json:"announcement_id"`
	Date                           string          `json:"date"`
	Val                            json.RawMessage `json:"val"`
	OriginalVal                    json.RawMessage `json:"original_val"`
	OriginalUnit                   json.RawMessage `json:"original_unit"`
	ValMOM                         json.RawMessage `json:"val_mom"`
	ObservationID                  json.RawMessage `json:"observation_id"`
	AnnouncementDatetime           json.RawMessage `json:"announcement_datetime"`
	AnnouncementDatetimeLocal      json.RawMessage `json:"announcement_datetime_local"`
	OfficialPlannedReleaseDatetime json.RawMessage `json:"official_planned_release_datetime"`
	OfficialActualReleaseDatetime  json.RawMessage `json:"official_actual_release_datetime"`
	CollectedAtNS                  json.RawMessage `json:"collected_at_ns"`
	CollectedAtISO                 json.RawMessage `json:"collected_at_iso"`
	IngestionLatencyMS             json.RawMessage `json:"ingestion_latency_ms"`
	PctChange                      json.RawMessage `json:"pct_change"`
	PctChange12M                   json.RawMessage `json:"pct_change_12m"`
	Revisions                      json.RawMessage `json:"revisions"`
}

// AnnouncementChangesResponse contains changed announcement events.
type AnnouncementChangesResponse struct {
	Data             []AnnouncementChangeEvent `json:"data"`
	Count            int                       `json:"count"`
	NextCursor       string                    `json:"next_cursor"`
	HasMore          bool                      `json:"has_more"`
	RetentionSeconds int                       `json:"retention_seconds"`
	Scope            json.RawMessage           `json:"scope"`
}

// AnnouncementChangeEvent describes one announcement change notification.
type AnnouncementChangeEvent struct {
	EventID            string          `json:"event_id"`
	Currency           string          `json:"currency"`
	Indicator          string          `json:"indicator"`
	RecordsWritten     json.RawMessage `json:"records_written"`
	Timestamp          json.RawMessage `json:"timestamp"`
	LatestAnnouncement json.RawMessage `json:"latest_announcement"`
}

// CalendarResponse contains scheduled macroeconomic releases.
type CalendarResponse struct {
	Currency    string               `json:"currency"`
	Timezone    json.RawMessage      `json:"timezone"`
	RequestedTZ json.RawMessage      `json:"requested_timezone"`
	Indicator   json.RawMessage      `json:"indicator"`
	StartDate   json.RawMessage      `json:"start_date"`
	EndDate     json.RawMessage      `json:"end_date"`
	DataQuality DataQuality          `json:"data_quality"`
	Data        []CalendarReleaseRow `json:"data"`
}

// CalendarReleaseRow is one scheduled macroeconomic release.
type CalendarReleaseRow struct {
	AnnouncementDatetime                  int64           `json:"announcement_datetime"`
	Release                               string          `json:"release"`
	AnnouncementDatetimeUTC               json.RawMessage `json:"announcement_datetime_utc"`
	AnnouncementDatetimeLocal             json.RawMessage `json:"announcement_datetime_local"`
	AnnouncementDatetimeRequestedTimezone json.RawMessage `json:"announcement_datetime_requested_timezone"`
	ReleaseDateConfirmed                  json.RawMessage `json:"release_date_confirmed"`
	Name                                  json.RawMessage `json:"name"`
	Source                                json.RawMessage `json:"source"`
}

// PredictionsResponse contains model and consensus forecasts for announcements.
type PredictionsResponse struct {
	Currency         string                    `json:"currency"`
	Indicator        json.RawMessage           `json:"indicator"`
	PredictionType   json.RawMessage           `json:"prediction_type"`
	PredictionSource json.RawMessage           `json:"prediction_source"`
	StartDate        json.RawMessage           `json:"start_date"`
	EndDate          json.RawMessage           `json:"end_date"`
	Count            int                       `json:"count"`
	PredictionCount  int                       `json:"prediction_count"`
	DataQuality      DataQuality               `json:"data_quality"`
	Data             []AnnouncementPredictions `json:"data"`
}

// AnnouncementPredictions groups forecasts for a scheduled observation.
type AnnouncementPredictions struct {
	AnnouncementID            string           `json:"announcement_id"`
	ObservationID             json.RawMessage  `json:"observation_id"`
	SelectedSeriesID          json.RawMessage  `json:"selected_series_id"`
	Currency                  string           `json:"currency"`
	Indicator                 string           `json:"indicator"`
	Date                      string           `json:"date"`
	AnnouncementDatetime      json.RawMessage  `json:"announcement_datetime"`
	AnnouncementDatetimeLocal json.RawMessage  `json:"announcement_datetime_local"`
	AnnouncementTiming        json.RawMessage  `json:"announcement_timing"`
	Predictions               []PredictionItem `json:"predictions"`
}

// PredictionItem is one forecast value.
type PredictionItem struct {
	PredictedValue        float64         `json:"predicted_value"`
	PredictionType        json.RawMessage `json:"prediction_type"`
	PredictionSource      json.RawMessage `json:"prediction_source"`
	PredictionSourceLabel json.RawMessage `json:"prediction_source_label"`
	GeneratedAt           json.RawMessage `json:"generated_at"`
	Confidence            json.RawMessage `json:"confidence"`
	PredictionReason      json.RawMessage `json:"prediction_reason"`
}

// COTResponse contains CFTC positioning observations.
type COTResponse struct {
	Currency    string          `json:"currency"`
	Instrument  string          `json:"instrument"`
	Source      string          `json:"source"`
	SourceURL   string          `json:"source_url"`
	StartDate   string          `json:"start_date"`
	EndDate     string          `json:"end_date"`
	DataQuality DataQuality     `json:"data_quality"`
	Pagination  json.RawMessage `json:"pagination"`
	Data        []COTDataPoint  `json:"data"`
}

// COTDataPoint is one CFTC positioning observation.
type COTDataPoint struct {
	Date                 string          `json:"date"`
	AnnouncementDatetime json.RawMessage `json:"announcement_datetime"`
}

// CommodityResponse contains commodity observations.
type CommodityResponse struct {
	Currency    string               `json:"currency"`
	Indicator   string               `json:"indicator"`
	Source      json.RawMessage      `json:"source"`
	SourceURL   json.RawMessage      `json:"source_url"`
	StartDate   string               `json:"start_date"`
	EndDate     string               `json:"end_date"`
	DataQuality DataQuality          `json:"data_quality"`
	Pagination  json.RawMessage      `json:"pagination"`
	Data        []CommodityDataPoint `json:"data"`
}

// CommodityDataPoint is one commodity observation.
type CommodityDataPoint struct {
	Date                 string          `json:"date"`
	Val                  json.RawMessage `json:"val"`
	AnnouncementDatetime json.RawMessage `json:"announcement_datetime"`
	PctChange            json.RawMessage `json:"pct_change"`
	PctChange12M         json.RawMessage `json:"pct_change_12m"`
}

// CurveSnapshotResponse contains yield-curve nodes.
type CurveSnapshotResponse struct {
	Currency      string           `json:"currency"`
	CurveFamily   string           `json:"curve_family"`
	Metric        string           `json:"metric"`
	RequestedDate string           `json:"requested_date"`
	AsOf          json.RawMessage  `json:"as_of"`
	NodeCount     int              `json:"node_count"`
	Sources       []string         `json:"sources"`
	DataQuality   DataQuality      `json:"data_quality"`
	Data          []CurveNodePoint `json:"data"`
}

// CurveNodePoint is one yield-curve node.
type CurveNodePoint struct {
	Indicator            string          `json:"indicator"`
	Maturity             string          `json:"maturity"`
	Date                 string          `json:"date"`
	Val                  float64         `json:"val"`
	AnnouncementDatetime json.RawMessage `json:"announcement_datetime"`
	Source               json.RawMessage `json:"source"`
}

// CurveProxyResponse contains curve-proxy spreads and inversion state.
type CurveProxyResponse struct {
	Currency      string            `json:"currency"`
	CurveFamily   string            `json:"curve_family"`
	RequestedDate string            `json:"requested_date"`
	AsOf          json.RawMessage   `json:"as_of"`
	NodeCount     int               `json:"node_count"`
	SlopeCount    int               `json:"slope_count"`
	InvertedCount int               `json:"inverted_count"`
	Sources       []string          `json:"sources"`
	DataQuality   DataQuality       `json:"data_quality"`
	Data          []CurveProxyPoint `json:"data"`
}

// CurveProxyPoint is one curve-proxy spread.
type CurveProxyPoint struct {
	Label                     string          `json:"label"`
	ShortMaturity             string          `json:"short_maturity"`
	LongMaturity              string          `json:"long_maturity"`
	ShortIndicator            string          `json:"short_indicator"`
	LongIndicator             string          `json:"long_indicator"`
	ShortVal                  float64         `json:"short_val"`
	LongVal                   float64         `json:"long_val"`
	Slope                     float64         `json:"slope"`
	SlopeBPS                  float64         `json:"slope_bps"`
	Inverted                  bool            `json:"inverted"`
	Date                      string          `json:"date"`
	ShortAnnouncementDatetime json.RawMessage `json:"short_announcement_datetime"`
	LongAnnouncementDatetime  json.RawMessage `json:"long_announcement_datetime"`
}

// ForwardCurveResponse contains forward curve segments.
type ForwardCurveResponse struct {
	Currency      string              `json:"currency"`
	CurveFamily   string              `json:"curve_family"`
	Method        string              `json:"method"`
	RequestedDate string              `json:"requested_date"`
	AsOf          json.RawMessage     `json:"as_of"`
	NodeCount     int                 `json:"node_count"`
	SegmentCount  int                 `json:"segment_count"`
	Sources       []string            `json:"sources"`
	DataQuality   DataQuality         `json:"data_quality"`
	Data          []ForwardCurvePoint `json:"data"`
}

// ForwardCurvePoint is one forward curve segment.
type ForwardCurvePoint struct {
	Label                     string          `json:"label"`
	StartMaturity             string          `json:"start_maturity"`
	EndMaturity               string          `json:"end_maturity"`
	StartIndicator            string          `json:"start_indicator"`
	EndIndicator              string          `json:"end_indicator"`
	StartYears                float64         `json:"start_years"`
	EndYears                  float64         `json:"end_years"`
	HorizonYears              float64         `json:"horizon_years"`
	Date                      string          `json:"date"`
	Val                       float64         `json:"val"`
	ValBPS                    float64         `json:"val_bps"`
	StartAnnouncementDatetime json.RawMessage `json:"start_announcement_datetime"`
	EndAnnouncementDatetime   json.RawMessage `json:"end_announcement_datetime"`
}

// RateDifferentialResponse contains historical rate differentials.
type RateDifferentialResponse struct {
	Base             string                  `json:"base"`
	Quote            string                  `json:"quote"`
	MeasureRequested string                  `json:"measure_requested"`
	MeasureUsed      string                  `json:"measure_used"`
	BaseIndicator    string                  `json:"base_indicator"`
	QuoteIndicator   string                  `json:"quote_indicator"`
	StartDate        string                  `json:"start_date"`
	EndDate          string                  `json:"end_date"`
	MatchedPoints    int                     `json:"matched_points"`
	Unit             string                  `json:"unit"`
	DataQuality      DataQuality             `json:"data_quality"`
	Pagination       json.RawMessage         `json:"pagination"`
	Data             []RateDifferentialPoint `json:"data"`
}

// RateDifferentialPoint is one matched rate differential observation.
type RateDifferentialPoint struct {
	Date                      string          `json:"date"`
	BaseVal                   float64         `json:"base_val"`
	QuoteVal                  float64         `json:"quote_val"`
	Spread                    float64         `json:"spread"`
	SpreadBPS                 float64         `json:"spread_bps"`
	BaseAnnouncementDatetime  json.RawMessage `json:"base_announcement_datetime"`
	QuoteAnnouncementDatetime json.RawMessage `json:"quote_announcement_datetime"`
}

// ForwardDifferentialResponse contains forward-rate differentials.
type ForwardDifferentialResponse struct {
	Base          string                     `json:"base"`
	Quote         string                     `json:"quote"`
	CurveFamily   string                     `json:"curve_family"`
	StartTenor    string                     `json:"start_tenor"`
	EndTenor      string                     `json:"end_tenor"`
	ForwardLabel  string                     `json:"forward_label"`
	StartDate     string                     `json:"start_date"`
	EndDate       string                     `json:"end_date"`
	MatchedPoints int                        `json:"matched_points"`
	DataQuality   DataQuality                `json:"data_quality"`
	Pagination    json.RawMessage            `json:"pagination"`
	Data          []ForwardDifferentialPoint `json:"data"`
}

// ForwardDifferentialPoint is one matched forward-rate differential observation.
type ForwardDifferentialPoint struct {
	Date                           string          `json:"date"`
	BaseForwardVal                 float64         `json:"base_forward_val"`
	QuoteForwardVal                float64         `json:"quote_forward_val"`
	Differential                   float64         `json:"differential"`
	DifferentialBPS                float64         `json:"differential_bps"`
	BaseStartVal                   float64         `json:"base_start_val"`
	BaseEndVal                     float64         `json:"base_end_val"`
	QuoteStartVal                  float64         `json:"quote_start_val"`
	QuoteEndVal                    float64         `json:"quote_end_val"`
	BaseStartAnnouncementDatetime  json.RawMessage `json:"base_start_announcement_datetime"`
	BaseEndAnnouncementDatetime    json.RawMessage `json:"base_end_announcement_datetime"`
	QuoteStartAnnouncementDatetime json.RawMessage `json:"quote_start_announcement_datetime"`
	QuoteEndAnnouncementDatetime   json.RawMessage `json:"quote_end_announcement_datetime"`
}

// MarketSessionsResponse contains the undocumented market-session payload.
// Data is preserved verbatim until the API publishes a field-level schema.
type MarketSessionsResponse struct {
	Data json.RawMessage `json:"data"`
}

// RiskSentimentResponse contains the undocumented risk-sentiment payload.
type RiskSentimentResponse struct {
	Data json.RawMessage `json:"data"`
}

// NewsResponse contains the undocumented macro-news payload.
type NewsResponse struct {
	Data json.RawMessage `json:"data"`
}

// PressReleasesResponse contains the undocumented official-release payload.
type PressReleasesResponse struct {
	Data json.RawMessage `json:"data"`
}
