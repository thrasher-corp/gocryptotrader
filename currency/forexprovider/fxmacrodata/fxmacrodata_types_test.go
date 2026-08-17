package fxmacrodata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

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
