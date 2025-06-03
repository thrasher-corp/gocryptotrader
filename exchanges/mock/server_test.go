package mock

import (
	"bytes"
	"net/http"
	"os"
	"strings"
	"testing"

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
	if err == nil {
		t.Error("NewVCRServer error cannot be nil")
	}

	// Set up mock data
	test1 := VCRMock{}
	test1.Routes = make(map[string]map[string][]HTTPResponse)
	test1.Routes["/test"] = make(map[string][]HTTPResponse)

	rp, err := json.Marshal(responsePayload{
		Price:    8000.0,
		Amount:   1,
		Currency: "bitcoin",
	})
	if err != nil {
		t.Fatal("marshal error", err)
	}

	testValue := HTTPResponse{Data: rp, QueryString: queryString, BodyParams: queryString}
	test1.Routes["/test"][http.MethodGet] = []HTTPResponse{testValue}

	payload, err := json.Marshal(test1)
	if err != nil {
		t.Fatal("marshal error", err)
	}

	err = os.WriteFile(testFile, payload, os.ModePerm)
	if err != nil {
		t.Fatal("marshal error", err)
	}

	deets, client, err := NewVCRServer(testFile)
	if err != nil {
		t.Error("NewVCRServer error", err)
	}

	err = common.SetHTTPClient(client) // Set common package global HTTP Client
	if err != nil {
		t.Fatal(err)
	}

	_, err = common.SendHTTPRequest(t.Context(),
		http.MethodGet,
		"http://localhost:300/somethingElse?"+queryString,
		nil,
		bytes.NewBufferString(""), true)
	if err == nil {
		t.Error("Sending http request expected an error")
	}

	// Expected good outcome
	r, err := common.SendHTTPRequest(t.Context(),
		http.MethodGet,
		deets,
		nil,
		bytes.NewBufferString(""), true)
	if err != nil {
		t.Error("Sending http request error", err)
	}

	if !strings.Contains(string(r), "404 page not found") {
		t.Error("Was not expecting any value returned:", r)
	}

	r, err = common.SendHTTPRequest(t.Context(),
		http.MethodGet,
		deets+"/test?"+queryString,
		nil,
		bytes.NewBufferString(""), true)
	if err != nil {
		t.Error("Sending http request error", err)
	}

	var res responsePayload
	err = json.Unmarshal(r, &res)
	if err != nil {
		t.Error("unmarshal error", err)
	}

	if res.Price != 8000 {
		t.Error("response error expected 8000 but received:",
			res.Price)
	}

	if res.Amount != 1 {
		t.Error("response error expected 1 but received:",
			res.Amount)
	}

	if res.Currency != "bitcoin" {
		t.Error("response error expected \"bitcoin\" but received:",
			res.Currency)
	}

	// clean up test.json file
	err = os.Remove(testFile)
	if err != nil {
		t.Fatal("Remove error", err)
	}
}
