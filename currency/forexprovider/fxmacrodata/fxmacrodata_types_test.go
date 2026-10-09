package fxmacrodata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestUnixSecondsJSON(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want string
	}{
		{"pre-1938 release", "-1763461800", "1914-02-13T13:30:00Z"},
		{"1990s release", "916407000", "1999-01-15T13:30:00Z"},
		{"post-2001 release", "1000128600", "2001-09-10T13:30:00Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var ts UnixSeconds
			require.NoError(t, json.Unmarshal([]byte(tc.raw), &ts), "UnixSeconds must decode epoch seconds")
			assert.Equal(t, tc.want, ts.Time().UTC().Format(time.RFC3339), "UnixSeconds should decode the epoch as seconds")

			encoded, err := json.Marshal(ts)
			require.NoError(t, err, "UnixSeconds must encode epoch seconds")
			assert.Equal(t, tc.raw, string(encoded), "UnixSeconds should round-trip the epoch unchanged")
		})
	}
}

func TestUnixSecondsJSONZeroValue(t *testing.T) {
	var ts UnixSeconds
	require.NoError(t, json.Unmarshal([]byte("null"), &ts), "UnixSeconds must accept a null optional value")
	assert.True(t, ts.Time().IsZero(), "UnixSeconds should retain its zero value for null")

	encoded, err := json.Marshal(ts)
	require.NoError(t, err, "UnixSeconds must encode its zero value")
	assert.Equal(t, "null", string(encoded), "UnixSeconds should encode its zero value as null")
}

func TestUnixSecondsJSONRejectsNonInteger(t *testing.T) {
	var ts UnixSeconds
	assert.Error(t, json.Unmarshal([]byte(`1786105800.5`), &ts), "UnixSeconds should reject a fractional timestamp")
	assert.Error(t, json.Unmarshal([]byte(`"2026-08-14T03:55:54Z"`), &ts), "UnixSeconds should reject an RFC 3339 string")
}

func TestUnixNanosJSON(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want string
	}{
		{"pre-1938 publication", "-1763461800000000000", "1914-02-13T13:30:00Z"},
		{"1990s publication", "916407000000000000", "1999-01-15T13:30:00Z"},
		{"18-digit boundary", "999999999000000000", "2001-09-09T01:46:39Z"},
		{"recent publication", "1786105800123456789", "2026-08-07T12:30:00.123456789Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var ts UnixNanos
			require.NoError(t, json.Unmarshal([]byte(tc.raw), &ts), "UnixNanos must decode epoch nanoseconds")
			assert.Equal(t, tc.want, ts.Time().UTC().Format(time.RFC3339Nano), "UnixNanos should decode the epoch as nanoseconds")

			encoded, err := json.Marshal(ts)
			require.NoError(t, err, "UnixNanos must encode epoch nanoseconds")
			assert.Equal(t, tc.raw, string(encoded), "UnixNanos should round-trip the epoch unchanged")
		})
	}
}

func TestUnixNanosJSONZeroValue(t *testing.T) {
	var ts UnixNanos
	require.NoError(t, json.Unmarshal([]byte("null"), &ts), "UnixNanos must accept a null optional value")
	assert.True(t, ts.Time().IsZero(), "UnixNanos should retain its zero value for null")

	encoded, err := json.Marshal(ts)
	require.NoError(t, err, "UnixNanos must encode its zero value")
	assert.Equal(t, "null", string(encoded), "UnixNanos should encode its zero value as null")
}

func TestUnixNanosJSONRejectsNonInteger(t *testing.T) {
	var ts UnixNanos
	assert.Error(t, json.Unmarshal([]byte(`1786105800.5`), &ts), "UnixNanos should reject a fractional timestamp")
	assert.Error(t, json.Unmarshal([]byte(`"1786105800123456789"`), &ts), "UnixNanos should reject a quoted timestamp")
}

func TestUnixMillisJSON(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want string
	}{
		{"pre-1938 chart point", "-1763461800000", "1914-02-13T13:30:00Z"},
		{"1990s chart point", "916407000000", "1999-01-15T13:30:00Z"},
		{"12-digit boundary", "999999999000", "2001-09-09T01:46:39Z"},
		{"recent chart point", "1786105800123", "2026-08-07T12:30:00.123Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var ts UnixMillis
			require.NoError(t, json.Unmarshal([]byte(tc.raw), &ts), "UnixMillis must decode epoch milliseconds")
			assert.Equal(t, tc.want, ts.Time().UTC().Format(time.RFC3339Nano), "UnixMillis should decode the epoch as milliseconds")

			encoded, err := json.Marshal(ts)
			require.NoError(t, err, "UnixMillis must encode epoch milliseconds")
			assert.Equal(t, tc.raw, string(encoded), "UnixMillis should round-trip the epoch unchanged")
		})
	}
}

func TestUnixMillisJSONZeroValue(t *testing.T) {
	var ts UnixMillis
	require.NoError(t, json.Unmarshal([]byte("null"), &ts), "UnixMillis must accept a null optional value")
	assert.True(t, ts.Time().IsZero(), "UnixMillis should retain its zero value for null")

	encoded, err := json.Marshal(ts)
	require.NoError(t, err, "UnixMillis must encode its zero value")
	assert.Equal(t, "null", string(encoded), "UnixMillis should encode its zero value as null")
}

func TestUnixMillisJSONRejectsNonInteger(t *testing.T) {
	var ts UnixMillis
	assert.Error(t, json.Unmarshal([]byte(`1786105800123.5`), &ts), "UnixMillis should reject a fractional timestamp")
	assert.Error(t, json.Unmarshal([]byte(`"1786105800123"`), &ts), "UnixMillis should reject a quoted timestamp")
}

func TestSourceNamesJSON(t *testing.T) {
	var single SourceNames
	require.NoError(t, json.Unmarshal([]byte(`"ECB"`), &single), "SourceNames must decode a single publisher name")
	assert.Equal(t, SourceNames{"ECB"}, single, "SourceNames should wrap a single name in a list")

	var many SourceNames
	require.NoError(t, json.Unmarshal([]byte(`["Federal Reserve","Treasury"]`), &many), "SourceNames must decode a list of publishers")
	assert.Equal(t, SourceNames{"Federal Reserve", "Treasury"}, many, "SourceNames should retain every listed name")

	var absent SourceNames
	require.NoError(t, json.Unmarshal([]byte("null"), &absent), "SourceNames must accept a null optional value")
	assert.Nil(t, absent, "SourceNames should retain its zero value for null")

	assert.Error(t, json.Unmarshal([]byte(`42`), &absent), "SourceNames should reject a non-string value")
}

func TestDateJSON(t *testing.T) {
	var date Date
	require.NoError(t, json.Unmarshal([]byte(`"2026-08-14"`), &date),
		"Date must decode an ISO 8601 calendar date")
	assert.Equal(t, "2026-08-14", date.String(), "Date should retain the calendar date")

	encoded, err := json.Marshal(date)
	require.NoError(t, err, "Date must encode an ISO 8601 calendar date")
	assert.JSONEq(t, `"2026-08-14"`, string(encoded), "Date should encode without a time or timezone")
}

func TestDateJSONZeroValue(t *testing.T) {
	var date Date
	require.NoError(t, json.Unmarshal([]byte("null"), &date), "Date must accept a null optional value")
	assert.Empty(t, date.String(), "Date should retain its zero value for null")
	require.NoError(t, json.Unmarshal([]byte(`""`), &date), "Date must accept an empty optional value")
	assert.Empty(t, date.String(), "Date should retain its zero value for an empty string")

	encoded, err := json.Marshal(date)
	require.NoError(t, err, "Date must encode its zero value")
	assert.Equal(t, "null", string(encoded), "Date should encode its zero value as null")
}

func TestDateJSONRejectsNonString(t *testing.T) {
	var date Date
	err := json.Unmarshal([]byte(`123`), &date)
	assert.Error(t, err, "Date should reject non-string JSON values")
}

func TestDateJSONRejectsDateTime(t *testing.T) {
	var date Date
	err := json.Unmarshal([]byte(`"2026-08-14T03:55:54Z"`), &date)
	assert.Error(t, err, "Date should reject values containing a time or timezone")
}

func TestFreeFormResponseFieldsPreserveJSON(t *testing.T) {
	const raw = `{"integer":9007199254740993,"decimal":0.123456789012345678901,"nested":{"keep":true},"items":[null,"text"]}`
	var forex ForexResponse
	require.NoError(t, json.Unmarshal([]byte(`{"coverage":`+raw+`,"indicators":`+raw+`,"daily_ohlc_basis":`+raw+`,"technical_indicator_basis":`+raw+`}`), &forex),
		"ForexResponse must retain every free-form field")
	var curves CurveAnalyticsResponse
	require.NoError(t, json.Unmarshal([]byte(`{"data":[`+raw+`]}`), &curves),
		"CurveAnalyticsResponse must retain free-form rows")
	require.Len(t, curves.Data, 1, "the curve row must remain available to decode")
	var factor FactorDataPoint
	require.NoError(t, json.Unmarshal([]byte(`{"components":`+raw+`,"source_observations":`+raw+`}`), &factor),
		"FactorDataPoint must retain both free-form fields")

	for name, field := range map[string]json.RawMessage{
		"coverage":                  forex.Coverage,
		"indicators":                forex.Indicators,
		"daily OHLC basis":          forex.DailyOHLCBasis,
		"technical indicator basis": forex.TechnicalIndicatorBasis,
		"curve row":                 curves.Data[0],
		"components":                factor.Components,
		"source observations":       factor.SourceObservations,
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, raw, string(field), "raw JSON should preserve large integers, decimal precision and nested values")
			var callerDefined struct {
				Integer int64           `json:"integer"`
				Decimal json.RawMessage `json:"decimal"`
			}
			require.NoError(t, json.Unmarshal(field, &callerDefined), "callers must be able to decode their own concrete type")
			assert.Equal(t, int64(9007199254740993), callerDefined.Integer, "the integer should remain exact beyond float64 precision")
			assert.Equal(t, "0.123456789012345678901", string(callerDefined.Decimal), "the decimal should remain exact")
			encoded, err := json.Marshal(field)
			require.NoError(t, err, "free-form JSON must remain serialisable")
			assert.Equal(t, raw, string(encoded), "serialisation should retain the original values")
		})
	}
}
