package mock

import (
	"bytes"
	"net/http"
	"os"
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
	assert.Error(t, err, "NewVCRServer error cannot be nil")

	// Set up mock data
	test1 := VCRMock{}
	test1.Routes = make(map[string]map[string][]HTTPResponse)
	test1.Routes["/test"] = make(map[string][]HTTPResponse)

	rp, err := json.Marshal(responsePayload{
		Price:    8000.0,
		Amount:   1,
		Currency: "bitcoin",
	})
	require.NoErrorf(t, err, "marshal error: %v", err)

	testValue := HTTPResponse{Data: rp, QueryString: queryString, BodyParams: queryString}
	test1.Routes["/test"][http.MethodGet] = []HTTPResponse{testValue}

	payload, err := json.Marshal(test1)
	require.NoErrorf(t, err, "marshal error: %v", err)

	err = os.WriteFile(testFile, payload, os.ModePerm)
	require.NoErrorf(t, err, "marshal error: %v", err)

	deets, client, err := NewVCRServer(testFile)
	assert.NoErrorf(t, err, "NewVCRServer error: %v", err)

	err = common.SetHTTPClient(client) // Set common package global HTTP Client
	require.NoError(t, err)

	_, err = common.SendHTTPRequest(t.Context(),
		http.MethodGet,
		"http://localhost:300/somethingElse?"+queryString,
		nil,
		bytes.NewBufferString(""), true)
	assert.Error(t, err, "Sending http request expected an error")

	// Expected good outcome
	r, err := common.SendHTTPRequest(t.Context(),
		http.MethodGet,
		deets,
		nil,
		bytes.NewBufferString(""), true)
	assert.NoErrorf(t, err, "Sending http request error: %v", err)
	assert.Containsf(t, "404 page not found", string(r), "Was not expecting any value returned: %s", string(r))

	r, err = common.SendHTTPRequest(t.Context(),
		http.MethodGet,
		deets+"/test?"+queryString,
		nil,
		bytes.NewBufferString(""), true)
	assert.NoErrorf(t, err, "Sending http request error: %v", err)

	var res responsePayload
	err = json.Unmarshal(r, &res)
	assert.NoErrorf(t, err, "unmarshal error: %v", err)
	assert.Equalf(t, 8000.0, res.Price, "response error expected 8000 but received: %f", res.Price)
	assert.Equalf(t, 1.0, res.Amount, "response error expected 1 but received: %f", res.Amount)
	assert.Equalf(t, "bitcoin", res.Currency, "response error expected \"bitcoin\" but received: %s", res.Currency)

	// clean up test.json file
	err = os.Remove(testFile)
	require.NoErrorf(t, err, "Remove error: %v", err)
}
