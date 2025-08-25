package mock

import (
	"bytes"
	"net"
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
