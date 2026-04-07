package mock

import (
	"bytes"
	"encoding/base64"
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

func TestRegisterHandlerArrayBodyMatching(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name          string
		contentType   string
		requestBody   string
		geminiPayload string
	}{
		{
			name:        "application_json_array_body",
			contentType: applicationJSON,
			requestBody: `[{"coin":"btc","amount":1},{"coin":"eth","amount":2}]`,
		},
		{
			name:          "text_plain_array_body",
			contentType:   textPlain,
			requestBody:   "",
			geminiPayload: `[{"coin":"btc","amount":1},{"coin":"eth","amount":2}]`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			arrayPayload := `[{"coin":"btc","amount":1},{"coin":"eth","amount":2}]`
			mux := http.NewServeMux()
			RegisterHandler("/array-shape", map[string][]HTTPResponse{
				http.MethodPost: {
					{
						Headers: http.Header{
							contentType: {tc.contentType},
						},
						BodyParams: arrayPayload,
						Data:       json.RawMessage(`{"match":"array"}`),
					},
				},
			}, mux)

			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)

			req, err := http.NewRequestWithContext(t.Context(),
				http.MethodPost,
				srv.URL+"/array-shape",
				strings.NewReader(tc.requestBody))
			require.NoError(t, err)
			req.Header.Set(contentType, tc.contentType)
			if tc.geminiPayload != "" {
				req.Header.Set("X-Gemini-Payload", base64.StdEncoding.EncodeToString([]byte(tc.geminiPayload)))
			}

			resp, err := srv.Client().Do(req)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, resp.Body.Close())
			})
			require.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			got := map[string]string{}
			err = json.Unmarshal(body, &got)
			require.NoError(t, err)
			assert.Equal(t, "array", got["match"])
		})
	}
}

func TestMatchAndGetResponseJSONSlice(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name       string
		mockBody   string
		request    []url.Values
		wantErr    bool
		wantResult string
	}{
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
