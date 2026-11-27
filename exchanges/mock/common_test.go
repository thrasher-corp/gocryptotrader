package mock

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestMatchURLVals(t *testing.T) { //nolint:tparallel // Cases share their url.Values inputs and run in order
	t.Parallel()
	testVal, testVal2, testVal3, emptyVal := url.Values{}, url.Values{}, url.Values{}, url.Values{}
	testVal.Add("test", "test")
	testVal2.Add("test2", "test2")
	testVal3.Add("test", "diferentValString")

	nonceVal1, nonceVal2 := url.Values{}, url.Values{}
	nonceVal1.Add("nonce", "012349723587")
	nonceVal2.Add("nonce", "9327373874")

	signVal1, signVal2 := url.Values{}, url.Values{}
	signVal1.Add("sign", "8c1b0f6e")
	signVal2.Add("sign", "41a9d2c7")

	// Recordings blank api_key, so a recorded request matches whichever key a test sends
	apiKeyVal1, apiKeyVal2 := url.Values{}, url.Values{}
	apiKeyVal1.Add("api_key", "")
	apiKeyVal2.Add("api_key", "mock-api-key")

	tests := []struct {
		name string
		a    url.Values
		b    url.Values
		exp  bool
	}{
		{"values against no values", testVal, emptyVal, false},
		{"no values against values", emptyVal, testVal, false},
		{"different keys", testVal, testVal2, false},
		{"different keys reversed", testVal2, testVal, false},
		{"different values for a key", testVal, testVal3, false},
		{"nonce against another key", nonceVal1, testVal2, false},
		{"no values against no values", emptyVal, emptyVal, true},
		{"same values", testVal, testVal, true},
		{"different nonces", nonceVal1, nonceVal2, true},
		{"sign against another key", signVal1, testVal, false},
		{"different signs", signVal1, signVal2, true},
		{"blank api_key against another key", apiKeyVal1, testVal, false},
		{"blank api_key against a set one", apiKeyVal1, apiKeyVal2, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MatchURLVals(tc.a, tc.b)
			assert.Equalf(t, tc.exp, got, "MatchURLVals should return correctly for (%q, %q)", tc.a, tc.b)
		})
	}
}

type multiTypeHolder struct {
	Counter    int     `json:"counter"`
	Numbers    []int   `json:"numbers"`
	Number     float64 `json:"number"`
	SomeString string  `json:"somestring"`
	IsEnabled  bool    `json:"isEnabled,omitempty"`
}

type uniqueTypesHolder struct {
	Things     []string        `json:"things"`
	Data       multiTypeHolder `json:"data"`
	Counter    int             `json:"counter"`
	IsEvenNum  bool            `json:"numbers"`
	Number     float64         `json:"number"`
	SomeString string          `json:"somestring"`
}

func TestDeriveURLValsFromJSONMap(t *testing.T) {
	t.Parallel()
	test1 := uniqueTypesHolder{
		Things: []string{"hello", "world"},
		Data: multiTypeHolder{
			Counter:    1,
			Numbers:    []int{1, 3, 3, 7, 9},
			Number:     3.14,
			SomeString: "hello, peoples",
		},
		Counter:    1,
		IsEvenNum:  false,
		Number:     3.14,
		SomeString: "hello, peoples",
	}
	payload, err := json.Marshal(test1)
	require.NoError(t, err, "Marshal must not error")

	values, err := DeriveURLValsFromJSONMap(payload)
	assert.NoError(t, err, "DeriveURLValsFromJSONMap should not error")
	assert.Len(t, values, 6)

	test2 := map[string]string{
		"val":  "1",
		"val2": "2",
		"val3": "3",
		"val4": "4",
		"val5": "5",
		"val6": "6",
		"val7": "7",
	}

	payload, err = json.Marshal(test2)
	require.NoError(t, err, "Marshal must not error")

	values, err = DeriveURLValsFromJSONMap(payload)
	require.NoError(t, err, "DeriveURLValsFromJSONMap must not error")
	require.Equal(t, 7, len(values), "DeriveURLValsFromJSONMap must return the correct number of values")
	for key, val := range values {
		require.Len(t, val, 1)
		assert.Equalf(t, test2[key], val[0], "DeriveURLValsFromJSON should return the correct value for %s", key)
	}

	_, err = DeriveURLValsFromJSONMap([]byte(`{"a":"b",}`))
	assert.Error(t, err)

	t.Run("structured values", func(t *testing.T) {
		t.Parallel()
		values, err := DeriveURLValsFromJSONMap([]byte(`{"object":{"a":1},"array":[1,"b"],"null":null,"text":"{\"a\":1}"}`))
		require.NoError(t, err, "DeriveURLValsFromJSONMap must not error for structured values")
		exp := url.Values{
			"object": {structuredValuePrefix + `{"a":1}`},
			"array":  {structuredValuePrefix + `[1,"b"]`},
			"null":   {structuredValuePrefix + "null"},
			"text":   {`{"a":1}`},
		}
		assert.Equal(t, exp, values, "DeriveURLValsFromJSONMap should mark objects, arrays and null apart from strings")
	})

	t.Run("strings resembling marked values", func(t *testing.T) {
		t.Parallel()
		marked, err := DeriveURLValsFromJSONMap([]byte(`{"null":null,"text":"json:null","marker":"\u0000json:null","escaped":"\u0000string:\u0000json:null"}`))
		require.NoError(t, err, "DeriveURLValsFromJSONMap must not error for strings resembling marked values")
		assert.Equal(t, "json:null", marked.Get("text"), "DeriveURLValsFromJSONMap should keep an ordinary string as sent")
		assert.Equal(t, escapedStringPrefix+structuredValuePrefix+"null", marked.Get("marker"), "DeriveURLValsFromJSONMap should escape a string starting with the marker byte")
		assert.Equal(t, escapedStringPrefix+escapedStringPrefix+structuredValuePrefix+"null", marked.Get("escaped"), "DeriveURLValsFromJSONMap should escape a string starting with the escape marker")
		assert.NotEqual(t, marked.Get("marker"), marked.Get("escaped"), "DeriveURLValsFromJSONMap should never derive two strings as the same value")
		assert.NotEqual(t, marked.Get("null"), marked.Get("marker"), "DeriveURLValsFromJSONMap should never derive a string as a marked null")
		assert.NotEqual(t, marked.Get("null"), marked.Get("text"), "DeriveURLValsFromJSONMap should never derive a string as a marked null")
	})
}

func TestStringValue(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input, exp string
	}{
		{name: "ordinary", input: "hello", exp: "hello"},
		{name: "empty", input: "", exp: ""},
		{name: "JSON text", input: `{"a":1}`, exp: `{"a":1}`},
		{name: "marker byte inside", input: "a\x00json:null", exp: "a\x00json:null"},
		{name: "marker byte alone", input: "\x00", exp: escapedStringPrefix + "\x00"},
		{name: "structured value marker", input: structuredValuePrefix + "null", exp: escapedStringPrefix + structuredValuePrefix + "null"},
		{name: "escape marker", input: escapedStringPrefix + "x", exp: escapedStringPrefix + escapedStringPrefix + "x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, stringValue(tc.input), "stringValue should escape only a string starting with the marker byte")
		})
	}
}

var tcs = []uniqueTypesHolder{
	{
		Things: []string{"hello", "world"},
		Data: multiTypeHolder{
			Counter:    1,
			Numbers:    []int{1, 3, 3, 7, 9},
			Number:     3.14,
			SomeString: "hello, peoples",
		},
		Counter:    1,
		IsEvenNum:  false,
		Number:     3.14,
		SomeString: "case 1",
	},
	{
		Things: []string{"hello", "thrasher"},
		Data: multiTypeHolder{
			Counter:    2,
			Numbers:    []int{1, 9, 9, 9},
			Number:     3.14529,
			SomeString: "hello, team",
		},
		IsEvenNum:  true,
		Number:     3,
		SomeString: "case 2",
	},
	{
		Things: []string{"hello", "Ethiopia"},
		Data: multiTypeHolder{
			Counter:    3,
			Numbers:    []int{2, 0, 1, 8},
			Number:     2018,
			SomeString: "hello, there",
			IsEnabled:  true,
		},
		IsEvenNum:  true,
		Number:     3,
		SomeString: "case 3",
	},
	{
		SomeString: "case 4",
	},
	{},
}

func TestDeriveURLValsFromJSONArray(t *testing.T) {
	t.Parallel()
	_, err := DeriveURLValsFromJSONArray([]byte(``))
	require.NoError(t, err, "DeriveURLValsFromJSONSlice must not error")

	_, err = DeriveURLValsFromJSONArray([]byte(`[invalid`))
	assert.Error(t, err, "invalid JSON should error")

	payload, err := json.Marshal(tcs)
	require.NoError(t, err, "Marshal must not error")

	values, err := DeriveURLValsFromJSONArray(payload)
	require.NoError(t, err, "DeriveURLValsFromJSONSlice must not error")
	assert.Len(t, values, len(tcs))

	for i := range tcs {
		t.Run("Test :"+tcs[i].SomeString, func(t *testing.T) {
			t.Parallel()
			elemPayload, err := json.Marshal(tcs[i])
			require.NoError(t, err, "Marshal must not error")

			val, err := DeriveURLValsFromJSONMap(elemPayload)
			require.NoError(t, err, "DeriveURLValsFromJSONMap must not error")
			assert.True(t, MatchURLVals(values[i], val), "MatchURLVals should be true")
		})
	}
}

func TestDeriveURLValsFromJSONArrayOfScalars(t *testing.T) {
	t.Parallel()
	// Lists of order IDs are sent as arrays of strings, whose elements are matched by their JSON text
	values, err := DeriveURLValsFromJSONArray([]byte(`["1", 2, true, null]`))
	require.NoError(t, err, "DeriveURLValsFromJSONArray must not error for an array of scalars")
	exp := []url.Values{{"": {`"1"`}}, {"": {"2"}}, {"": {"true"}}, {"": {"null"}}}
	assert.Equal(t, exp, values, "DeriveURLValsFromJSONArray should key each scalar's JSON text by an empty string")

	request, err := DeriveURLValsFromJSONArray([]byte(`[1,"2",true,null]`))
	require.NoError(t, err, "DeriveURLValsFromJSONArray must not error for an array of scalars")
	assert.False(t, MatchURLVals(values[0], request[0]), "a string should not match the same number")
	assert.False(t, MatchURLVals(values[1], request[1]), "a number should not match the same string")
	assert.True(t, MatchURLVals(values[2], request[2]), "the same scalar should match")
}

func TestDeriveURLValsFromJSONArrayAsMap(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal([][]uniqueTypesHolder{tcs})
	require.NoError(t, err, "Marshal must not error")

	values, err := DeriveURLValsFromJSONArray(payload)
	require.NoError(t, err, "DeriveURLValsFromJSONArray must not error")
	assert.Len(t, values, 1)

	// Testing JSON array as map
	payload, err = json.Marshal(tcs)
	require.NoError(t, err, "Marshal must not error")

	value, err := DeriveURLValsFromJSONArrayAsMap(payload)
	require.NoError(t, err, "DeriveURLValsFromJSONArrayAsMap must not error")
	assert.Len(t, value, len(tcs))

	value, err = DeriveURLValsFromJSONArrayAsMap([]byte{})
	assert.NoError(t, err, "empty payload should not error")
	assert.Empty(t, value, "empty payload should return empty url.Values")

	_, err = DeriveURLValsFromJSONArrayAsMap([]byte(`{"not":"array"}`))
	assert.ErrorIs(t, err, errJSONMapPayloadMustBeArray, "non-array input should return errJSONMapPayloadMustBeObject")

	_, err = DeriveURLValsFromJSONArrayAsMap([]byte(`[invalid`))
	assert.Error(t, err, "invalid JSON should error")

	value, err = DeriveURLValsFromJSONArrayAsMap([]byte(`["btc"]`))
	require.NoError(t, err, "single element array must not error")
	assert.Len(t, value, 1, "single element should produce one entry")
	assert.Equal(t, "btc", value.Get("0"), "element value should be stored under index key")

	value, err = DeriveURLValsFromJSONArrayAsMap([]byte(`[true, 1.5, "hello"]`))
	require.NoError(t, err, "primitive types array must not error")
	assert.Equal(t, "true", value.Get("0"), "bool should be formatted as string")
	assert.Equal(t, "1.5", value.Get("1"), "float64 should be formatted as string")
	assert.Equal(t, "hello", value.Get("2"), "string value should be stored correctly")

	t.Run("structured values", func(t *testing.T) {
		t.Parallel()
		value, err := DeriveURLValsFromJSONArrayAsMap([]byte(`[{"a":1}, [2], null, "{\"a\":1}"]`))
		require.NoError(t, err, "structured types array must not error")
		exp := url.Values{
			"0": {structuredValuePrefix + `{"a":1}`},
			"1": {structuredValuePrefix + "[2]"},
			"2": {structuredValuePrefix + "null"},
			"3": {`{"a":1}`},
		}
		assert.Equal(t, exp, value, "DeriveURLValsFromJSONArrayAsMap should mark objects, arrays and null apart from strings")
	})

	t.Run("strings resembling marked values", func(t *testing.T) {
		t.Parallel()
		value, err := DeriveURLValsFromJSONArrayAsMap([]byte(`[null, "\u0000json:null", "\u0000string:\u0000json:null"]`))
		require.NoError(t, err, "DeriveURLValsFromJSONArrayAsMap must not error for a string resembling a marked value")
		exp := url.Values{
			"0": {structuredValuePrefix + "null"},
			"1": {escapedStringPrefix + structuredValuePrefix + "null"},
			"2": {escapedStringPrefix + escapedStringPrefix + structuredValuePrefix + "null"},
		}
		assert.Equal(t, exp, value, "DeriveURLValsFromJSONArrayAsMap should escape a string starting with the marker byte, the escape marker included")
	})
}
