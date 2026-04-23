package mock

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

type responsePayload struct {
	Price    float64 `json:"price"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

const (
	queryString = "currency=btc&command=getprice"
	testFile    = "test.json"
)

func TestNewVCRServer(t *testing.T) {
	_, _, err := NewVCRServer("")
	assert.ErrorIs(t, err, errJSONMockFilePathRequired)

	// Set up mock data
	test1 := VCRMock{}
	test1.Routes = make(map[string]map[string][]HTTPResponse)
	test1.Routes["/test"] = make(map[string][]HTTPResponse)

	rp, err := json.Marshal(responsePayload{
		Price:    8000.0,
		Amount:   1,
		Currency: "bitcoin",
	})
	require.NoError(t, err, "Marshal must not error")

	testValue := HTTPResponse{Data: rp, QueryString: queryString, BodyParams: queryString}
	test1.Routes["/test"][http.MethodGet] = []HTTPResponse{testValue}

	payload, err := json.Marshal(test1)
	require.NoError(t, err, "Marshal must not error")

	err = os.WriteFile(testFile, payload, os.ModePerm)
	require.NoError(t, err, "WriteFile must not error")

	deets, client, err := NewVCRServer(testFile)
	assert.NoError(t, err, "NewVCRServer should not error")

	err = common.SetHTTPClient(client) // Set common package global HTTP Client
	require.NoError(t, err)

	_, err = common.SendHTTPRequest(t.Context(),
		http.MethodGet,
		"http://localhost:300/somethingElse?"+queryString,
		nil,
		bytes.NewBufferString(""), true)
	var netErr *net.OpError
	assert.ErrorAs(t, err, &netErr, "SendHTTPRequest should return a net.OpError for an invalid host")

	// Expected good outcome
	r, err := common.SendHTTPRequest(t.Context(),
		http.MethodGet,
		deets,
		nil,
		bytes.NewBufferString(""), true)
	assert.NoError(t, err, "SendHTTPRequest should not error")
	assert.Contains(t, string(r), "404 page not found", "SendHTTPRequest return should only contain 404")

	r, err = common.SendHTTPRequest(t.Context(),
		http.MethodGet,
		deets+"/test?"+queryString,
		nil,
		bytes.NewBufferString(""), true)
	assert.NoError(t, err, "SendHTTPRequest should not error")

	var res responsePayload
	err = json.Unmarshal(r, &res)
	assert.NoError(t, err, "Unmarshal should not error")
	assert.Equalf(t, 8000.0, res.Price, "response error expected 8000 but received: %f", res.Price)
	assert.Equalf(t, 1.0, res.Amount, "response error expected 1 but received: %f", res.Amount)
	assert.Equalf(t, "bitcoin", res.Currency, "response error expected \"bitcoin\" but received: %s", res.Currency)

	// clean up test.json file
	err = os.Remove(testFile)
	require.NoError(t, err, "Remove testFile must not error")
}

func TestRegisterHandlerAndMatching(t *testing.T) {
	t.Parallel()
	type payloadAndQuery struct {
		name          string
		method        string
		contentType   string
		requestBody   string
		geminiPayload string
		bodyParams    string
		queryString   string
	}
	tcs := []*payloadAndQuery{
		{
			name:        "post_application_json_array_body",
			method:      http.MethodPost,
			contentType: applicationJSON,
			requestBody: `[{"coin":"btc","amount":1},{"coin":"eth","amount":2}]`,
			bodyParams:  `[{"coin":"btc","amount":1},{"coin":"eth","amount":2}]`,
		},
		{
			name:          "post_text_plain_array_body",
			method:        http.MethodPost,
			contentType:   textPlain,
			requestBody:   "",
			geminiPayload: `[{"coin":"btc","amount":1},{"coin":"eth","amount":2}]`,
			bodyParams:    `[{"coin":"btc","amount":1},{"coin":"eth","amount":2}]`,
		},
		{
			name:        "put_application_json_object",
			method:      http.MethodPut,
			contentType: applicationJSON,
			requestBody: `{"currency":"btc"}`,
			bodyParams:  `{"currency":"btc"}`,
		},
		{
			name:        "delete_query_string",
			method:      http.MethodDelete,
			queryString: "currency=btc&amount=1",
		},
		{
			name:        "post_application_x-www-form-urlencoded",
			method:      http.MethodPost,
			contentType: applicationURLEncoded,
			requestBody: "currency=btc&amount=1",
			bodyParams:  "currency=btc&amount=1",
		},
		{
			name:        "post_empty_content-type_query_string",
			method:      http.MethodPost,
			queryString: "currency=btc",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mux := http.NewServeMux()
			RegisterHandler("/array-shape", map[string][]HTTPResponse{
				tc.method: {
					{
						Headers: http.Header{
							contentType: {tc.contentType},
						},
						QueryString: tc.queryString,
						BodyParams:  tc.bodyParams,
						Data:        json.RawMessage(fmt.Sprintf(`{"match":%q}`, tc.method)),
					},
				},
			}, mux)

			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)

			req, err := http.NewRequestWithContext(t.Context(),
				tc.method,
				srv.URL+"/array-shape"+"?"+tc.queryString,
				strings.NewReader(tc.requestBody))
			require.NoError(t, err)
			req.Header.Set(contentType, tc.contentType)
			if tc.geminiPayload != "" {
				req.Header.Set("X-Gemini-Payload", base64.StdEncoding.EncodeToString([]byte(tc.geminiPayload)))
			}
			if tc.contentType != "" {
				req.Header.Set(contentType, tc.contentType)
			}

			resp, err := srv.Client().Do(req)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, resp.Body.Close())
			})
			require.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.JSONEq(t, fmt.Sprintf(`{"match":%q}`, tc.method), strings.TrimSpace(string(body)))
		})
	}
}

func TestMatchAndGetResponseJSONSlice(t *testing.T) {
	t.Parallel()

	type reqAndMockBody struct {
		name       string
		mockBody   string
		request    []url.Values
		wantErr    bool
		wantResult string
	}
	tcs := []*reqAndMockBody{
		{
			name:     "match_array_payload",
			mockBody: `[{"coin":"btc","amount":1},{"coin":"eth","amount":2}]`,
			request: []url.Values{
				{
					"coin":   {"btc"},
					"amount": {"1"},
				},
				{
					"coin":   {"eth"},
					"amount": {"2"},
				},
			},
			wantResult: `{"match":"array"}`,
		},
		{
			name:     "erroring_match_array_payload",
			mockBody: `[{"coin":"btc","amount":1},{"coin":"eth","amount":2]`,
			request: []url.Values{
				{
					"coin":   {"btc"},
					"amount": {"1"},
				},
				{
					"coin":   {"eth"},
					"amount": {"2"},
				},
			},
			wantErr: true,
		},
		{
			name:     "no_match_array_payload",
			mockBody: `[{"coin":"btc","amount":1},{"coin":"eth","amount":2}]`,
			request: []url.Values{
				{
					"coin":   {"btc"},
					"amount": {"10"},
				},
				{
					"coin":   {"eth"},
					"amount": {"20"},
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockData := []HTTPResponse{
				{
					BodyParams: tc.mockBody,
					Data:       json.RawMessage(`{"match":"array"}`),
				},
			}

			got, err := MatchAndGetResponseJSONSlice(mockData, tc.request, false)
			if tc.wantErr {
				require.ErrorIs(t, err, errNoDataMatched)
				return
			}
			require.NoError(t, err)
			assert.JSONEq(t, tc.wantResult, string(got))
		})
	}
}

func TestMatchAndGetResponse(t *testing.T) {
	t.Parallel()
	type mockResponseAndMatch struct {
		name        string
		mockData    []HTTPResponse
		requestVals url.Values
		isQueryData bool
		wantResult  string
		wantErr     error
	}
	tcs := []*mockResponseAndMatch{
		{
			name:        "query string match",
			mockData:    []HTTPResponse{{QueryString: "currency=btc&amount=1", Data: json.RawMessage(`{"match":"query"}`)}},
			requestVals: url.Values{"currency": {"btc"}, "amount": {"1"}},
			isQueryData: true,
			wantResult:  `{"match":"query"}`,
		},
		{
			name:        "body params match non-json",
			mockData:    []HTTPResponse{{BodyParams: "currency=btc&amount=1", Data: json.RawMessage(`{"match":"body"}`)}},
			requestVals: url.Values{"currency": {"btc"}, "amount": {"1"}},
			isQueryData: false,
			wantResult:  `{"match":"body"}`,
		},
		{
			name:        "json object body with string value",
			mockData:    []HTTPResponse{{BodyParams: `{"currency":"btc"}`, Data: json.RawMessage(`{"match":"string"}`)}},
			requestVals: url.Values{"currency": {"btc"}},
			isQueryData: false,
			wantResult:  `{"match":"string"}`,
		},
		{
			name:        "json object body with bool value",
			mockData:    []HTTPResponse{{BodyParams: `{"active":true}`, Data: json.RawMessage(`{"match":"bool"}`)}},
			requestVals: url.Values{"active": {"true"}},
			isQueryData: false,
			wantResult:  `{"match":"bool"}`,
		},
		{
			name:        "json object body with float64 value",
			mockData:    []HTTPResponse{{BodyParams: `{"amount":1.5}`, Data: json.RawMessage(`{"match":"float"}`)}},
			requestVals: url.Values{"amount": {"1.5"}},
			isQueryData: false,
			wantResult:  `{"match":"float"}`,
		},
		{
			name: "json array body params entry skipped falling through to next",
			mockData: []HTTPResponse{
				{BodyParams: `[1,2,3]`, Data: json.RawMessage(`{"match":"should-not-reach"}`)},
				{BodyParams: "currency=btc", Data: json.RawMessage(`{"match":"fallback"}`)},
			},
			requestVals: url.Values{"currency": {"btc"}},
			isQueryData: false,
			wantResult:  `{"match":"fallback"}`,
		},
		{
			name:        "empty mock data returns errNoDataMatched",
			mockData:    []HTTPResponse{},
			requestVals: url.Values{"currency": {"btc"}},
			isQueryData: false,
			wantErr:     errNoDataMatched,
		},
		{
			name:        "no matching entry returns errNoDataMatched",
			mockData:    []HTTPResponse{{BodyParams: "currency=eth", Data: json.RawMessage(`{"match":"eth"}`)}},
			requestVals: url.Values{"currency": {"btc"}},
			isQueryData: false,
			wantErr:     errNoDataMatched,
		},
		{
			name:        "empty body params matches empty request vals",
			mockData:    []HTTPResponse{{Data: json.RawMessage(`{"match":"empty"}`)}},
			isQueryData: false,
			wantResult:  `{"match":"empty"}`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := MatchAndGetResponse(tc.mockData, tc.requestVals, tc.isQueryData)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "MatchAndGetResponse must return expected error")
				return
			}
			require.NoError(t, err, "MatchAndGetResponse must not error")
			assert.JSONEq(t, tc.wantResult, string(got), "MatchAndGetResponse should return correct data")
		})
	}
}

func TestJSONBodyArrayRegression(t *testing.T) {
	t.Parallel()

	// Regression proof:
	// Before the fix, playback attempted DeriveURLValsFromJSONMap for JSON body
	// payloads, which fails for arrays.
	arrayPayload := []byte(`[{"coin":"btc","amount":1},{"coin":"eth","amount":2}]`)
	_, err := DeriveURLValsFromJSONMap(arrayPayload)
	require.ErrorIs(t, err, errJSONMapPayloadMustBeObject)

	// Fixed behaviour: array payloads are matched via DeriveURLValsFromJSONSlice
	// + MatchAndGetResponseJSONSlice.
	reqVals, err := DeriveURLValsFromJSONArray(arrayPayload)
	require.NoError(t, err)

	mockData := []HTTPResponse{
		{
			BodyParams: string(arrayPayload),
			Data:       json.RawMessage(`{"match":"array"}`),
		},
	}

	got, err := MatchAndGetResponseJSONSlice(mockData, reqVals, false)
	require.NoError(t, err)
	assert.JSONEq(t, `{"match":"array"}`, string(got))
}

func TestMessageWriteJSON(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	MessageWriteJSON(w, http.StatusOK, nil)
	assert.Equal(t, http.StatusOK, w.Code, "status code should be correct")
	assert.Equal(t, applicationJSON, w.Header().Get(contentType), "Content-Type should be set")
	assert.Empty(t, w.Body.String(), "nil data should produce empty body")

	w = httptest.NewRecorder()
	MessageWriteJSON(w, http.StatusCreated, map[string]string{"key": "value"})
	assert.Equal(t, http.StatusCreated, w.Code, "status code should be correct")
	assert.Equal(t, applicationJSON, w.Header().Get(contentType), "Content-Type should be set")
	assert.JSONEq(t, `{"key":"value"}`, strings.TrimSpace(w.Body.String()), "body should contain encoded JSON")

	w = httptest.NewRecorder()
	MessageWriteJSON(w, http.StatusCreated, &struct {
		Key string `json:"key"`
	}{
		Key: "value",
	})
	assert.Equal(t, http.StatusCreated, w.Code, "status code should be correct")
	assert.JSONEq(t, `{"key":"value"}`, strings.TrimSpace(w.Body.String()), "body should contain encoded JSON")

	w = httptest.NewRecorder()
	MessageWriteJSON(w, http.StatusAccepted, json.RawMessage(`{"price":8000}`))
	assert.Equal(t, http.StatusAccepted, w.Code, "status code should be correct")
	assert.JSONEq(t, `{"price":8000}`, strings.TrimSpace(w.Body.String()), "body should match raw JSON")
}
