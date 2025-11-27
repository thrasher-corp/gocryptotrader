package mock

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestGetFilteredHeader(t *testing.T) {
	items, err := getExcludedItems()
	require.NoError(t, err, "getExcludedItems must not error")
	assert.NotNil(t, items, "getExcludedItems should not return nil")

	resp := http.Response{}
	resp.Request = &http.Request{}
	resp.Request.Header = http.Header{}
	resp.Request.Header.Set("Key", "RiskyVals")
	fMap := GetFilteredHeader(&resp, items)
	assert.Empty(t, fMap.Get("Key"), "risky values should be removed")
}

func TestGetFilteredURLVals(t *testing.T) {
	items, err := getExcludedItems()
	require.NoError(t, err, "getExcludedItems must not error")
	assert.NotNil(t, items, "getExcludedItems should not return nil")

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

	items, err := getExcludedItems()
	require.NoError(t, err, "getExcludedItems must not error")
	assert.NotNil(t, items, "getExcludedItems should not return nil")

	data, err := CheckResponsePayload(payload, items, 5)
	assert.NoError(t, err, "CheckResponsePayload should not error")

	expected := `{
 "stuff": "REAAAAHHHHH"
}`
	assert.Equal(t, expected, string(data))
}

func TestGetExcludedItems(t *testing.T) {
	exclusionList, err := getExcludedItems()
	require.NoError(t, err, "getExcludedItems must not error")
	assert.NotEmpty(t, exclusionList.Headers, "Headers should not be empty")
	assert.NotEmpty(t, exclusionList.Variables, "Variables should not be empty")
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
	OkayVal  string         `json:"okayVal"`
	OkayVal2 float64        `json:"okayVal2"`
	OkayVal3 map[string]any `json:"okayVal3"`
	OkayVal4 map[string]any `json:"okayVal4"`
	OkayVal5 []any          `json:"okayVal5"`
	BadVal   int64          `json:"receiver_name"`
	BadVal2  string         `json:"account_number"`
	BadVal3  []any          `json:"secret"`
}

var testVal = []TestStructLevel0{
	{
		StringVal: "somestringstuff",
		FloatVal:  3.14,
		IntVal:    1337,
		StructVal: TestStructLevel1{
			OkayVal:  "stuff",
			OkayVal2: 120938,
			BadVal:   "CritcalBankingStuff",
			BadVal2:  1337,
			OtherData: TestStructLevel2{
				OkayVal:  "stuff",
				OkayVal2: 129219809899009009080980,
				OkayVal3: map[string]any{"a": 1},
				OkayVal4: map[string]any{},
				OkayVal5: []any{},
				BadVal:   1337,
				BadVal2:  "Super Secret Password",
				BadVal3:  []any{123, 'a'},
			},
		},
		MixedSlice: []any{
			[]map[string]any{{"id": 0}, {"id": 2}, {"id": 3}, {"id": 4}, {"id": 5}, {"id": 6}, {}},
			[]any{float64(1586994000000), "6615.23000000", 'a', 1234, false, int64(17866372632), 0},
			"abcd",
		},
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
		IntVal:    1337,
	},
	{},
	{},
	{},
}

func TestCheckJSON(t *testing.T) {
	exclusionList, err := getExcludedItems()
	require.NoError(t, err, "getExcludedItems must not error")
	assert.NotNil(t, exclusionList, "getExcludedItems should not return nil")

	data, err := json.Marshal(testVal)
	require.NoError(t, err, "Marshal must not error")
	require.NotNil(t, data, "Marshal must not return nil")

	var input any
	err = json.Unmarshal(data, &input)
	require.NoError(t, err, "Unmarshal must not error")

	vals, err := CheckJSON(input, &exclusionList, 4)
	assert.NoError(t, err, "CheckJSON should not error")

	payload, err := json.Marshal(vals)
	require.NoError(t, err, "Marshal must not error")

	newStruct := []TestStructLevel0{}
	err = json.Unmarshal(payload, &newStruct)
	require.NoError(t, err, "Umarshal must not error")

	assert.Len(t, newStruct, 4)
	assert.Empty(t, newStruct[0].StructVal.BadVal, "Value not wiped correctly")
	assert.Empty(t, newStruct[0].StructVal.BadVal2, "Value not wiped correctly")
	assert.Empty(t, newStruct[0].StructVal.OtherData.BadVal, "BadVal should be removed")
	assert.Empty(t, newStruct[0].StructVal.OtherData.BadVal2, "BadVal2 should be removed")
	assert.Len(t, newStruct[0].MixedSlice[0], 4)
	assert.Len(t, newStruct[0].MixedSlice[1], 7)
}

func TestHTTPRecord(t *testing.T) {
	t.Parallel()

	service := "mock"
	outputDirPath := filepath.Join("..", service, "testdata")
	err := os.Mkdir(outputDirPath, 0o755)
	require.NoError(t, err)

	filePath := filepath.Join(outputDirPath, "http.json")
	err = os.WriteFile(filePath, []byte(`{"routes": null}`), 0o644)
	require.NoError(t, err)

	_, err = os.Stat(filePath)
	require.NoError(t, err, "file not created properly")

	defer func() {
		require.NoErrorf(t, os.Remove(filePath), "Remove test exclusion file %q must not error", filePath)
		require.NoErrorf(t, os.Remove(outputDirPath), "Remove test exclusion dir %q must not error", outputDirPath)
	}()

	content, err := json.Marshal(testVal)
	require.NoError(t, err, "Marshal must not error")
	require.NotNil(t, content, "Marshal must not return nil")

	response := &http.Response{
		Request: &http.Request{
			Method: http.MethodGet,
			URL:    &url.URL{},
		},
	}
	err = HTTPRecord(response, "mock", content, 4)
	require.NoError(t, err, "HTTPRecord must not error")
}
