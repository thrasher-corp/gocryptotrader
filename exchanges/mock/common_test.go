package mock

import (
	"net/url"
	"testing"

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
	if received != expected {
		t.Errorf("MatchURLVals error expected %v received %v",
			expected,
			received)
	}

	received = MatchURLVals(emptyVal, testVal)
	if received != expected {
		t.Errorf("MatchURLVals error expected %v received %v",
			expected,
			received)
	}

	received = MatchURLVals(testVal, testVal2)
	if received != expected {
		t.Errorf("MatchURLVals error expected %v received %v",
			expected,
			received)
	}

	received = MatchURLVals(testVal2, testVal)
	if received != expected {
		t.Errorf("MatchURLVals error expected %v received %v",
			expected,
			received)
	}

	received = MatchURLVals(testVal, testVal3)
	if received != expected {
		t.Errorf("MatchURLVals error expected %v received %v",
			expected,
			received)
	}

	received = MatchURLVals(nonceVal1, testVal2)
	if received != expected {
		t.Errorf("MatchURLVals error expected %v received %v",
			expected,
			received)
	}

	expected = true
	received = MatchURLVals(emptyVal, emptyVal)
	if received != expected {
		t.Errorf("MatchURLVals error expected %v received %v",
			expected,
			received)
	}

	received = MatchURLVals(testVal, testVal)
	if received != expected {
		t.Errorf("MatchURLVals error expected %v received %v",
			expected,
			received)
	}

	received = MatchURLVals(nonceVal1, nonceVal2)
	if received != expected {
		t.Errorf("MatchURLVals error expected %v received %v",
			expected,
			received)
	}
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
	if err != nil {
		t.Error("marshal error", err)
	}

	_, err = DeriveURLValsFromJSONMap(payload)
	if err != nil {
		t.Error("DeriveURLValsFromJSON error", err)
	}

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
	if err != nil {
		t.Error("marshal error", err)
	}

	vals, err := DeriveURLValsFromJSONMap(payload)
	if err != nil {
		t.Error("DeriveURLValsFromJSON error", err)
	}

	if vals["val"][0] != "1" {
		t.Error("DeriveURLValsFromJSON unexpected value",
			vals["val"][0])
	}
}
