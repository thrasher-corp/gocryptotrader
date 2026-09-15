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
