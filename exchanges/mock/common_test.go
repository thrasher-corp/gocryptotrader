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

func TestDeriveURLValsFromJSON(t *testing.T) {
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

	values, err = DeriveURLValsFromJSONMap([]byte(`{"a":"b",}`))
	require.Error(t, err)
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

func TestDeriveURLValsFromJSONSlice(t *testing.T) {
	t.Parallel()
	_, err := DeriveURLValsFromJSONArray([]byte(``))
	require.NoError(t, err, "DeriveURLValsFromJSONSlice must not error")

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

func TestDeriveURLValsFromJSONArrayAsMap(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal([][]uniqueTypesHolder{tcs})
	require.NoError(t, err, "Marshal must not error")

	values, err := DeriveURLValsFromJSONArray(payload)
	require.NoError(t, err, "DeriveURLValsFromJSONSlice must not error")
	assert.Len(t, values, 1)

	// Testing JSON array as map
	payload, err = json.Marshal(tcs)
	require.NoError(t, err, "Marshal must not error")

	value, err := DeriveURLValsFromJSONArrayAsMap(payload)
	require.NoError(t, err, "DeriveURLValsFromJSONSlice must not error")
	assert.Len(t, value, len(tcs))
}
