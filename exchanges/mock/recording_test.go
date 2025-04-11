package mock

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestGetFilteredHeader(t *testing.T) {
	resp := http.Response{}
	resp.Request = &http.Request{}
	resp.Request.Header = http.Header{}
	resp.Request.Header.Set("Key", "RiskyVals")
	fMap, err := GetFilteredHeader(&resp)
	if err != nil {
		t.Error(err)
	}

	if fMap.Get("Key") != "" {
		t.Error("risky vals where not replaced correctly")
	}
}

func TestGetFilteredURLVals(t *testing.T) {
	superSecretData := "Dr Seuss"
	shadyVals := url.Values{}
	shadyVals.Set("real_name", superSecretData)
	cleanVals, err := GetFilteredURLVals(shadyVals)
	if err != nil {
		t.Error("GetFilteredURLVals error", err)
	}

	if strings.Contains(cleanVals, superSecretData) {
		t.Error("Super secret data found")
	}
}

func TestCheckResponsePayload(t *testing.T) {
	testbody := struct {
		SomeJSON string `json:"stuff"`
	}{
		SomeJSON: "REAAAAHHHHH",
	}

	payload, err := json.Marshal(testbody)
	if err != nil {
		t.Fatal("json marshal error", err)
	}

	data, err := CheckResponsePayload(payload)
	if err != nil {
		t.Error("CheckBody error", err)
	}

	expected := `{
 "stuff": "REAAAAHHHHH"
}`

	if string(data) != expected {
		t.Error("unexpected returned data")
	}
}

type TestStructLevel0 struct {
	StringVal  string           `json:"stringVal"`
	FloatVal   float64          `json:"floatVal"`
	IntVal     int64            `json:"intVal"`
	StructVal  TestStructLevel1 `json:"structVal"`
	MixedSlice []any            `json:"mixedSlice"`
}

type TestStructLevel1 struct {
	OkayVal   string           `json:"okayVal"`
	OkayVal2  float64          `json:"okayVal2"`
	BadVal    string           `json:"user"`
	BadVal2   int              `json:"bsb"`
	OtherData TestStructLevel2 `json:"otherVals"`
}

type TestStructLevel2 struct {
	OkayVal   string           `json:"okayVal"`
	OkayVal2  float64          `json:"okayVal2"`
	BadVal    float32          `json:"name"`
	BadVal2   int32            `json:"real_name"`
	OtherData TestStructLevel3 `json:"moreOtherVals"`
}

type TestStructLevel3 struct {
	OkayVal  string  `json:"okayVal"`
	OkayVal2 float64 `json:"okayVal2"`
	BadVal   int64   `json:"receiver_name"`
	BadVal2  string  `json:"account_number"`
}

func TestCheckJSON(t *testing.T) {
	level3 := TestStructLevel3{
		OkayVal:  "stuff",
		OkayVal2: 129219,
		BadVal:   1337,
		BadVal2:  "Super Secret Password",
	}

	level2 := TestStructLevel2{
		OkayVal:   "stuff",
		OkayVal2:  129219,
		BadVal:    0.222,
		BadVal2:   1337888888,
		OtherData: level3,
	}

	level1 := TestStructLevel1{
		OkayVal:   "stuff",
		OkayVal2:  120938,
		BadVal:    "CritcalBankingStuff",
		BadVal2:   1337,
		OtherData: level2,
	}

	sliceOfPrimitives := []any{
		[]any{float64(1586994000000), "6615.23000000"},
		[]any{float64(1586994300000), "6624.74000000"},
	}

	testVal := TestStructLevel0{
		StringVal:  "somestringstuff",
		FloatVal:   3.14,
		IntVal:     1337,
		StructVal:  level1,
		MixedSlice: sliceOfPrimitives,
	}

	exclusionList, err := GetExcludedItems()
	if err != nil {
		t.Error("GetExcludedItems error", err)
	}

	vals, err := CheckJSON(testVal, &exclusionList)
	if err != nil {
		t.Error("Check JSON error", err)
	}

	payload, err := json.Marshal(vals)
	if err != nil {
		t.Fatal("json marshal error", err)
	}

	newStruct := TestStructLevel0{}
	err = json.Unmarshal(payload, &newStruct)
	if err != nil {
		t.Fatal("Umarshal error", err)
	}

	if newStruct.StructVal.BadVal != "" {
		t.Error("Value not wiped correctly")
	}

	if newStruct.StructVal.BadVal2 != 0 {
		t.Error("Value not wiped correctly")
	}

	if newStruct.StructVal.OtherData.BadVal != 0 {
		t.Error("Value not wiped correctly")
	}

	if newStruct.StructVal.OtherData.BadVal2 != 0 {
		t.Error("Value not wiped correctly")
	}

	if newStruct.StructVal.OtherData.OtherData.BadVal != 0 {
		t.Error("Value not wiped correctly")
	}

	if newStruct.StructVal.OtherData.OtherData.BadVal2 != "" {
		t.Error("Value not wiped correctly")
	}

	vals, err = CheckJSON(sliceOfPrimitives, &exclusionList)
	if err != nil {
		t.Error("Check JSON error", err)
	}

	payload, err = json.Marshal(vals)
	if err != nil {
		t.Fatal("json marshal error", err)
	}

	var newSlice []any
	err = json.Unmarshal(payload, &newSlice)
	if err != nil {
		t.Fatal("Unmarshal error", err)
	}
}

func TestGetExcludedItems(t *testing.T) {
	exclusionList, err := GetExcludedItems()
	if err != nil {
		t.Error("GetExcludedItems error", err)
	}

	if len(exclusionList.Headers) == 0 {
		t.Error("Header exclusion list not popoulated")
	}

	if len(exclusionList.Variables) == 0 {
		t.Error("Variable exclusion list not popoulated")
	}
}
