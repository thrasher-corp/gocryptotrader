package mock

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestMatchURLVals(t *testing.T) {
	testVal, testVal2, testVal3, emptyVal := url.Values{}, url.Values{}, url.Values{}, url.Values{}
	testVal.Add("test", "test")
	testVal2.Add("test2", "test2")
	testVal3.Add("test", "diferentValString")

	nonceVal1, nonceVal2 := url.Values{}, url.Values{}
	nonceVal1.Add("nonce", "012349723587")
	nonceVal2.Add("nonce", "9327373874")

	expected := false
	received := MatchURLVals(testVal, emptyVal)
	assert.Equalf(t, expected, received, "MatchURLVals error expected %v received %v", expected, received)

	received = MatchURLVals(emptyVal, testVal)
	assert.Equalf(t, expected, received, "MatchURLVals error expected %v received %v", expected, received)

	received = MatchURLVals(testVal, testVal2)
	assert.Equalf(t, expected, received, "MatchURLVals error expected %v received %v", expected, received)

	received = MatchURLVals(testVal2, testVal)
	assert.Equalf(t, expected, received, "MatchURLVals error expected %v received %v", expected, received)

	received = MatchURLVals(testVal, testVal3)
	assert.Equalf(t, expected, received, "MatchURLVals error expected %v received %v", expected, received)

	received = MatchURLVals(nonceVal1, testVal2)
	assert.Equalf(t, expected, received, "MatchURLVals error expected %v received %v", expected, received)

	expected = true
	received = MatchURLVals(emptyVal, emptyVal)
	assert.Equalf(t, expected, received, "MatchURLVals error expected %v received %v", expected, received)

	received = MatchURLVals(testVal, testVal)
	assert.Equalf(t, expected, received, "MatchURLVals error expected %v received %v", expected, received)

	received = MatchURLVals(nonceVal1, nonceVal2)
	assert.Equalf(t, expected, received, "MatchURLVals error expected %v received %v", expected, received)
}

func TestDeriveURLValsFromJSON(t *testing.T) {
	test1 := struct {
		Things []string `json:"things"`
		Data   struct {
			Numbers    []int   `json:"numbers"`
			Number     float64 `json:"number"`
			SomeString string  `json:"somestring"`
		} `json:"data"`
	}{
		Things: []string{"hello", "world"},
		Data: struct {
			Numbers    []int   `json:"numbers"`
			Number     float64 `json:"number"`
			SomeString string  `json:"somestring"`
		}{
			Numbers:    []int{1, 3, 3, 7},
			Number:     3.14,
			SomeString: "hello, peoples",
		},
	}

	payload, err := json.Marshal(test1)
	assert.NoErrorf(t, err, "marshal error: %v", err)

	_, err = DeriveURLValsFromJSONMap(payload)
	assert.NoErrorf(t, err, "DeriveURLValsFromJSON error: %v", err)

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
	assert.NoErrorf(t, err, "marshal error: %v", err)

	vals, err := DeriveURLValsFromJSONMap(payload)
	assert.NoErrorf(t, err, "DeriveURLValsFromJSON error: %v", err)
	assert.Equalf(t, "1", vals["val"][0], "DeriveURLValsFromJSON unexpected value: ^%v", vals["val"][0])
}
