package mock

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
)

type responsePayload struct {
	Price    float64 `json:"price"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

const queryString = "currency=btc&command=getprice"
const testFile = "test.json"

func TestNewVCRServer(t *testing.T) {
	_, err := NewVCRServer("", nil)
	if err == nil {
		t.Error("Test Failed - NewVCRServer error cannot be nil")
	}

	_, err = NewVCRServer("", t)
	if err == nil {
		t.Error("Test Failed - NewVCRServer error cannot be nil")
	}

	// Set up mock data
	test1 := VCRMock{}
	test1.Host = ":3500"
	test1.Routes = make(map[string]map[string][]HTTPResponse)
	test1.Routes["/test"] = make(map[string][]HTTPResponse)

	rp, err := json.Marshal(responsePayload{Price: 8000.0,
		Amount:   1,
		Currency: "bitcoin"})
	if err != nil {
		t.Fatal("Test Failed - marshal error", err)
	}

	testValue := HTTPResponse{Data: rp, QueryString: queryString, BodyParams: queryString}
	test1.Routes["/test"][http.MethodGet] = []HTTPResponse{testValue}

	payload, err := json.Marshal(test1)
	if err != nil {
		t.Fatal("Test Failed - marshal error", err)
	}

	err = ioutil.WriteFile(testFile, payload, os.ModePerm)
	if err != nil {
		t.Fatal("Test Failed - marshal error", err)
	}

	deets, err := NewVCRServer(testFile, t)
	if err != nil {
		t.Error("Test Failed - NewVCRServer error", err)
	}

	_, err = common.SendHTTPRequest(http.MethodGet,
		"http://localhost:300/somethingElse?"+queryString,
		nil,
		bytes.NewBufferString(""))
	if err == nil {
		t.Error("Test Failed - Sending http request expected an error")
	}

	// Expected good outcome
	r, err := common.SendHTTPRequest(http.MethodGet,
		deets,
		nil,
		bytes.NewBufferString(""))
	if err != nil {
		t.Error("Test Failed - Sending http request error", err)
	}

	if !strings.Contains(r, "404 page not found") {
		t.Error("Test Failed - Was not expecting any value returned:", r)
	}

	r, err = common.SendHTTPRequest(http.MethodGet,
		deets+"/test?"+queryString,
		nil,
		bytes.NewBufferString(""))
	if err != nil {
		t.Error("Test Failed - Sending http request error", err)
	}

	var res responsePayload
	err = json.Unmarshal([]byte(r), &res)
	if err != nil {
		t.Error("Test Failed - unmarshal error", err)
	}

	if res.Price != 8000 {
		t.Error("Test Failed - response error expected 8000 but received:",
			res.Price)
	}

	if res.Amount != 1 {
		t.Error("Test Failed - response error expected 1 but received:",
			res.Amount)
	}

	if res.Currency != "bitcoin" {
		t.Error("Test Failed - response error expected \"bitcoin\" but received:",
			res.Currency)
	}

	// clean up test.json file
	err = os.Remove(testFile)
	if err != nil {
		t.Fatal("Test Failed - Remove error", err)
	}
}
