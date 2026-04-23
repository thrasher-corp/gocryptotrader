package mock

import (
	"bytes"
	"io"
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
	t.Parallel()
	items, err := getExcludedItems()
	require.NoError(t, err, "getExcludedItems must not error")

	resp := &http.Response{Request: &http.Request{Header: http.Header{}}}
	resp.Request.Header.Set("Key", "RiskyVals")
	resp.Request.Header.Set("X-Mbx-Apikey", "secret-key")
	resp.Request.Header.Set("Accept", "application/json")

	fMap := GetFilteredHeader(resp, items)
	assert.Empty(t, fMap.Get("Key"), "excluded header Key should be cleared")
	assert.Empty(t, fMap.Get("X-Mbx-Apikey"), "excluded header X-Mbx-Apikey should be cleared")
	assert.Equal(t, "application/json", fMap.Get("Accept"), "non-excluded header should survive")
}

func TestGetFilteredURLVals(t *testing.T) {
	t.Parallel()
	items, err := getExcludedItems()
	require.NoError(t, err, "getExcludedItems must not error")

	vals := url.Values{}
	vals.Set("real_name", "Dr Seuss")
	vals.Set("user", "admin")
	vals.Set("currency", "btc")

	result := GetFilteredURLVals(vals, items)
	assert.NotContains(t, result, "Dr Seuss", "excluded real_name value should be removed")
	assert.NotContains(t, result, "admin", "excluded user value should be removed")
	assert.Contains(t, result, "currency=btc", "non-excluded currency should survive")
}

type checkclass struct {
	Counter int     `json:"counter,omitempty"`
	Numbers []int   `json:"numbers,omitempty"`
	Number  float64 `json:"number,omitempty"`
	Name    string  `json:"name,omitempty"`
}

func TestCheckResponsePayload(t *testing.T) {
	type someJSON struct {
		Secret      string      `json:"secret,omitempty"`
		Data        checkclass  `json:"data"`
		DataPointer *checkclass `json:"datapointer,omitempty"`
		Login       int         `json:"login,omitempty"`
		IsEvenNum   bool        `json:"pass,omitempty"`
		Balance     float64     `json:"bsb,omitempty"`
		RealName    string      `json:"real_name,omitempty"`
	}
	inputs := []struct {
		in  any
		exp []byte
	}{
		{
			in: []someJSON{
				{
					Secret: "REAAAAHHHHH",
					Data: checkclass{
						Name: "the-super-secret-name",
					},
				},
				{},
			}, exp: []byte("[\n {\n  \"data\": {\n   \"name\": \"\"\n  },\n  \"secret\": \"\"\n },\n {\n  \"data\": {}\n }\n]"),
		},
		{
			in: someJSON{}, exp: []byte("{\n \"data\": {}\n}"),
		},
		{in: []*string{}, exp: []byte(`[]`)},
		{in: someJSON{
			Secret:    "",
			Login:     1234,
			IsEvenNum: true,
			Balance:   1234.56,
			RealName:  "sam",
		}, exp: []byte("{\n \"bsb\": 0,\n \"data\": {},\n \"login\": 0,\n \"pass\": true,\n \"real_name\": \"\"\n}")},
	}

	items, err := getExcludedItems()
	require.NoError(t, err, "getExcludedItems must not error")
	assert.NotNil(t, items, "getExcludedItems should not return nil")

	for i := range inputs {
		payload, err := json.Marshal(inputs[i].in)
		require.NoError(t, err, "json marshal must not error")

		data, err := CheckResponsePayload(payload, items, 5)
		assert.NoError(t, err, "CheckResponsePayload should not error")
		assert.Equal(t, inputs[i].exp, data)
	}
}

func TestGetExcludedItems(t *testing.T) {
	t.Parallel()
	exclusionList, err := getExcludedItems()
	require.NoError(t, err, "getExcludedItems must not error")
	assert.NotEmpty(t, exclusionList.Headers, "Headers should not be empty")
	assert.NotEmpty(t, exclusionList.Variables, "Variables should not be empty")
	assert.Contains(t, exclusionList.Headers, "Key", "Key should be in excluded headers")
	assert.Contains(t, exclusionList.Variables, "real_name", "real_name should be in excluded variables")
	assert.Contains(t, exclusionList.Variables, "user", "user should be in excluded variables")
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
			BadVal:   "CriticalBankingStuff",
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
	require.NoError(t, err, "Unmarshal must not error")

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

	service := "mockhttprecord"
	serviceDir := filepath.Join("..", service)
	outputDirPath := filepath.Join("..", service, "testdata")
	err := os.RemoveAll(serviceDir)
	require.NoError(t, err)

	err = os.MkdirAll(outputDirPath, 0o755)
	require.NoError(t, err)

	filePath := filepath.Join(outputDirPath, "http.json")
	err = os.WriteFile(filePath, []byte(`{"routes": null}`), 0o644)
	require.NoError(t, err)

	_, err = os.Stat(filePath)
	require.NoError(t, err, "file not created properly")

	defer func() {
		require.NoErrorf(t, os.RemoveAll(serviceDir), "Remove test exclusion dir %q must not error", serviceDir)
	}()

	content, err := json.Marshal(testVal)
	require.NoError(t, err, "Marshal must not error")
	require.NotNil(t, content, "Marshal must not return nil")

	makeResponse := func(method, rawURL string, body []byte) *http.Response {
		parsed, parseErr := url.Parse(rawURL)
		require.NoError(t, parseErr)
		resp := &http.Response{
			Body: io.NopCloser(bytes.NewReader(nil)),
			Request: &http.Request{
				Method: method,
				URL:    parsed,
			},
		}
		if body != nil {
			resp.Request.Header = map[string][]string{
				contentType: {applicationJSON},
			}
			resp.Request.Body = io.NopCloser(bytes.NewReader(body))
			resp.Request.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(body)), nil
			}
		}
		return resp
	}
	callRecord := func(method, rawURL string, body []byte) error {
		resp := makeResponse(method, rawURL, body)
		defer func() {
			require.NoError(t, resp.Body.Close())
		}()
		return HTTPRecord(resp, service, content, 4)
	}

	assertRouteMethodLen := func(path, method string, expected int) {
		finalPayload, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr, "Read final mock file must not error")
		var finalMock VCRMock
		readErr = json.Unmarshal(finalPayload, &finalMock)
		require.NoError(t, readErr, "Unmarshal final mock file must not error")
		require.Contains(t, finalMock.Routes, path)
		assert.Len(t, finalMock.Routes[path][method], expected)
	}

	// Base GET entry for route initialisation coverage.
	err = callRecord(http.MethodGet, "https://api.abc.com/test/base", nil)
	require.NoError(t, err)

	// For write methods, repeated same request should overwrite existing record.
	for _, method := range []string{http.MethodPost, http.MethodDelete, http.MethodPut} {
		for range 2 {
			err = callRecord(method, "https://api.abc.com/test/payload", content)
			require.NoError(t, err, "HTTPRecord must not error")
		}
		assertRouteMethodLen("/test/payload", method, 1)
	}

	// Shape mismatch should return an explicit error.
	err = callRecord(http.MethodPost, "https://api.abc.com/test/mismatch", content)
	require.NoError(t, err, "HTTPRecord must not error")
	objectBody := []byte(`{"foo":"bar"}`)
	err = callRecord(http.MethodPost, "https://api.abc.com/test/mismatch", objectBody)
	require.ErrorIs(t, err, errMismatchedJSONBodyShape)
	assertRouteMethodLen("/test/mismatch", http.MethodPost, 1)

	existingMockBadParams := VCRMock{
		Routes: map[string]map[string][]HTTPResponse{
			"": {
				http.MethodPost: {
					{
						Headers:    map[string][]string{contentType: {applicationJSON}},
						BodyParams: `{invalid params`,
						Data:       json.RawMessage(`{}`),
					},
				},
			},
		},
	}
	mockData, err := json.Marshal(existingMockBadParams)
	require.NoError(t, err, "Marshal must not error")
	require.NoError(t, os.WriteFile(filePath, mockData, 0o644), "WriteFile must not error")

	invalidBody := []byte(`{invalid json`)
	err = callRecord(http.MethodPost, "", invalidBody)
	require.Error(t, err, "HTTPRecord must error for invalid request body JSON")

	err = callRecord(http.MethodPost, "", nil)
	require.Error(t, err, "HTTPRecord must error for invalid stored BodyParams JSON")

	existingMockBadParams.Routes[""]["ABC"] = []HTTPResponse{
		{
			Headers:    map[string][]string{contentType: {applicationJSON}},
			BodyParams: `{invalid params`,
			Data:       json.RawMessage(`{}`),
		},
	}
	mockData, err = json.Marshal(existingMockBadParams)
	require.NoError(t, err, "Marshal must not error")
	require.NoError(t, os.WriteFile(filePath, mockData, 0o644), "WriteFile must not error")

	err = callRecord("ABC", "", nil)
	require.ErrorContains(t, err, "unhandled request method")

	existingMockBadParams.Routes[""][http.MethodPost] = []HTTPResponse{
		{
			Headers:    map[string][]string{contentType: {"ABC"}},
			BodyParams: `{}`,
			Data:       json.RawMessage(`{}`),
		},
	}
	mockData, err = json.Marshal(existingMockBadParams)
	require.NoError(t, err, "Marshal must not error")
	require.NoError(t, os.WriteFile(filePath, mockData, 0o644), "WriteFile must not error")

	moockResponse := &http.Response{
		Request: &http.Request{
			Method: http.MethodPost,
			Header: http.Header{contentType: {"ABC"}},
			URL:    &url.URL{},
		},
	}
	err = HTTPRecord(moockResponse, service, content, 4)
	require.ErrorContains(t, err, "unhandled content type")
}

func TestIsExcluded(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		name        string
		key         string
		excluded    []string
		hasExcluded bool
	}{
		{name: "exact match", key: "real_name", excluded: []string{"real_name"}, hasExcluded: true},
		{name: "case insensitive uppercase", key: "REAL_NAME", excluded: []string{"real_name"}, hasExcluded: true},
		{name: "case insensitive mixed", key: "Real_Name", excluded: []string{"real_name"}, hasExcluded: true},
		{name: "no match", key: "currency", excluded: []string{"real_name", "apiKey"}, hasExcluded: false},
		{name: "empty excluded list", key: "real_name", excluded: []string{}, hasExcluded: false},
		{name: "nil excluded list", key: "real_name", excluded: nil, hasExcluded: false},
		{name: "match in middle of list", key: "user", excluded: []string{"bsb", "user", "name"}, hasExcluded: true},
		{name: "empty key no match", key: "", excluded: []string{"real_name"}, hasExcluded: false},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.hasExcluded, IsExcluded(tc.key, tc.excluded), "IsExcluded should return %v for key %q", tc.hasExcluded, tc.key)
		})
	}
}

func TestGetJSONBodyShape(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		name  string
		input string
		want  jsonBodyShape
	}{
		{name: "array", input: `[1,2,3]`, want: jsonBodyArray},
		{name: "object", input: `{"a":"b"}`, want: jsonBodyObject},
		{name: "empty string", input: "", want: jsonBodyUnknown},
		{name: "non-json string", input: "not json", want: jsonBodyUnknown},
		{name: "untrimmed array", input: "  [", want: jsonBodyUnknown},
		{name: "untrimmed object", input: "  {", want: jsonBodyUnknown},
		{name: "null literal", input: "null", want: jsonBodyUnknown},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, getJSONBodyShape(tc.input), "getJSONBodyShape should return correct shape")
		})
	}
}
