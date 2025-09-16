package mock

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestMatchURLVals(t *testing.T) {
	t.Parallel()
	testVal, testVal2, testVal3, emptyVal := url.Values{}, url.Values{}, url.Values{}, url.Values{}
	testVal.Add("test", "test")
	testVal2.Add("test2", "test2")
	testVal3.Add("test", "diferentValString")

	nonceVal1, nonceVal2 := url.Values{}, url.Values{}
	nonceVal1.Add("nonce", "012349723587")
	nonceVal2.Add("nonce", "9327373874")

	tests := []struct {
		a   url.Values
		b   url.Values
		exp bool
	}{
		{testVal, emptyVal, false},
		{emptyVal, testVal, false},
		{testVal, testVal2, false},
		{testVal2, testVal, false},
		{testVal, testVal3, false},
		{nonceVal1, testVal2, false},
		{emptyVal, emptyVal, true},
		{testVal, testVal, true},
		{nonceVal1, nonceVal2, true},
	}
	for _, tc := range tests {
		got := MatchURLVals(tc.a, tc.b)
		assert.Equalf(t, tc.exp, got, "MatchURLVals should return correctly for (%q, %q)", tc.a, tc.b)
	}
}

func TestDeriveURLValsFromJSON(t *testing.T) {
	type class struct {
		Numbers    []int   `json:"numbers"`
		Number     float64 `json:"number"`
		SomeString string  `json:"somestring"`
	}
	test1 := struct {
		Things []string `json:"things"`
		Data   class    `json:"data"`
	}{
		Things: []string{"hello", "world"},
		Data: class{
			Numbers:    []int{1, 3, 3, 7},
			Number:     3.14,
			SomeString: "hello, peoples",
		},
	}

	payload, err := json.Marshal(test1)
	require.NoError(t, err, "Marshal must not error")

	values, err := DeriveURLValsFromJSONMap(payload)
	assert.NoError(t, err, "DeriveURLValsFromJSONMap should not error")
	assert.Len(t, values, 2)

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
}
