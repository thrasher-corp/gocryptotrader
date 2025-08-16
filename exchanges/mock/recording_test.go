package mock

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestGetFilteredHeader(t *testing.T) {
	items, err := GetExcludedItems()
	require.NoError(t, err, "GetExcludedItems should not error")
	assert.NotNil(t, items)

	resp := http.Response{}
	resp.Request = &http.Request{}
	resp.Request.Header = http.Header{}
	resp.Request.Header.Set("Key", "RiskyVals")
	fMap := GetFilteredHeader(&resp, items)
	assert.Empty(t, fMap.Get("Key"), "risky values should be removed")
}

func TestGetFilteredURLVals(t *testing.T) {
	items, err := GetExcludedItems()
	require.NoError(t, err, "GetExcludedItems should not error")
	assert.NotNil(t, items)

	superSecretData := "Dr Seuss"
	shadyVals := url.Values{}
	shadyVals.Set("real_name", superSecretData)
	cleanVals := GetFilteredURLVals(shadyVals, items)
	assert.NotContains(t, cleanVals, superSecretData, "exclusion real_name should be removed")
}

func TestCheckResponsePayload(t *testing.T) {
	testbody := struct {
		SomeJSON string `json:"stuff"`
	}{
		SomeJSON: "REAAAAHHHHH",
	}

	payload, err := json.Marshal(testbody)
	require.NoError(t, err, "json marshal must not error")

	items, err := GetExcludedItems()
	require.NoError(t, err)
	assert.NotNil(t, items, "GetExcludedItems should not return nil")

	data, err := CheckResponsePayload(payload, items, 5)
	assert.NoError(t, err)

	expected := `{
 "stuff": "REAAAAHHHHH"
}`
	assert.Equal(t, expected, string(data))
}

type TestStructLevel0 struct {
	StringVal  string           `json:"stringVal"`
	FloatVal   float64          `json:"floatVal"`
	IntVal     int64            `json:"intVal"`
	StructVal  TestStructLevel1 `json:"structVal"`
	MixedSlice []any            `json:"mixedSlice"`
}

type TestStructLevel1 struct {
	OkayVal   string             `json:"okayVal"`
	OkayVal2  float64            `json:"okayVal2"`
	BadVal    string             `json:"user"`
	BadVal2   int                `json:"bsb"`
	OtherData []TestStructLevel2 `json:"otherVals"`
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

	level2 := []TestStructLevel2{
		{
			OkayVal:   "stuff",
			OkayVal2:  129219,
			BadVal:    0.222,
			BadVal2:   1337888888,
			OtherData: level3,
		},
		{},
		{},
		{},
		{},
		{},
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

	testVal := []TestStructLevel0{
		{
			StringVal:  "somestringstuff",
			FloatVal:   3.14,
			IntVal:     1337,
			StructVal:  level1,
			MixedSlice: sliceOfPrimitives,
		},
		{
			StringVal: "somestringstuff",
			FloatVal:  3.14,
		},
		{
			StringVal: "somestringstuff",
			FloatVal:  3.14,
			IntVal:    1337,
		},
		{
			StringVal: "somestringstuff",
			FloatVal:  3.14,
			IntVal:    1337,
		},
		{
			StringVal: "somestringstuff",
			IntVal:    1337,
		},
		{},
		{},
		{},
	}

	exclusionList, err := GetExcludedItems()
	assert.NoErrorf(t, err, "GetExcludedItems error: %v", err)

	data, err := json.Marshal(testVal)
	require.NoError(t, err, "json.Marshal nust not error")
	assert.NotNil(t, data, "json.Marshal nust not return nil")

	var input any
	err = json.Unmarshal(data, &input)
	require.NoError(t, err, "json.Unmarshal must not error")

	vals, err := CheckJSON(input, &exclusionList, 4)
	assert.NoErrorf(t, err, "Check JSON error: %v", err)

	payload, err := json.Marshal(vals)
	require.NoErrorf(t, err, "json marshal error: %v", err)

	newStruct := []TestStructLevel0{}
	err = json.Unmarshal(payload, &newStruct)
	require.NoErrorf(t, err, "Umarshal error: %v", err)

	assert.Len(t, newStruct, 4)
	assert.Empty(t, newStruct[0].StructVal.BadVal, "Value not wiped correctly")
	assert.Empty(t, newStruct[0].StructVal.BadVal2, "Value not wiped correctly")
	assert.Empty(t, newStruct[0].StructVal.OtherData[0].BadVal, "Value not wiped correctly")
	assert.Empty(t, newStruct[0].StructVal.OtherData[0].BadVal2, "Value not wiped correctly")
	assert.Empty(t, newStruct[0].StructVal.OtherData[0].OtherData.BadVal, "Value not wiped correctly")
	assert.Empty(t, newStruct[0].StructVal.OtherData[0].OtherData.BadVal2, "Value not wiped correctly")
	assert.Len(t, newStruct[0].StructVal.OtherData, 4)

	vals, err = CheckJSON(sliceOfPrimitives, &exclusionList, 0)
	assert.NoError(t, err)

	payload, err = json.Marshal(vals)
	require.NoError(t, err)

	var newSlice []any
	err = json.Unmarshal(payload, &newSlice)
	require.NoError(t, err)
}

func TestGetExcludedItems(t *testing.T) {
	exclusionList, err := GetExcludedItems()
	require.NoErrorf(t, err, "GetExcludedItems error: %v", err)
	assert.NotEmpty(t, exclusionList.Headers)
	assert.NotEmpty(t, exclusionList.Variables)
}
