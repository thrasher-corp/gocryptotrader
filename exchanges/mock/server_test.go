package mock

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
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

const queryString = "currency=btc&command=getprice"
const testFile = "test.json"

func TestNewVCRServer(t *testing.T) {
	_, _, err := NewVCRServer("")
	if err == nil {
		t.Error("NewVCRServer error cannot be nil")
	}

	// Set up mock data
	test1 := VCRMock{}
	test1.Routes = make(map[string]map[string][]HTTPResponse)
	test1.Routes["/test"] = make(map[string][]HTTPResponse)

	rp, err := json.Marshal(responsePayload{Price: 8000.0,
		Amount:   1,
		Currency: "bitcoin"})
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

	_, err = common.SendHTTPRequest(context.Background(),
		http.MethodGet,
		"http://localhost:300/somethingElse?"+queryString,
		nil,
		bytes.NewBufferString(""), true)
	if err == nil {
		t.Error("Sending http request expected an error")
	}

	// Expected good outcome
	r, err := common.SendHTTPRequest(context.Background(),
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

	r, err = common.SendHTTPRequest(context.Background(),
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

func TestMatchBatchAndGetResponse(t *testing.T) {
	var response = []HTTPResponse{{
		BodyParams: `[{ "amount":"0.003747662", "ask":"0.000000356", "askQuantity": "13274", "bid":"0.0000003365", "bidQuantity": "2674", "close":"0.0000003373", "closeTime":   "1694855078036", "dailyChange": "0.0224", "displayName": "BTS/BTC", "high": "0.0000003373", "low":"0.0000003299", "markPrice":"0.00000034", "open":"0.0000003299", "quantity":"11111", "startTime":"1694768640000", "symbol":"BTS_BTC", "tradeCount":  "2", "ts":"1694855083780" },{"amount":"0.00334476", "ask":"0.001009", "askQuantity": "4.49", "bid":"0.001006", "bidQuantity": "4.48", "close":"0.001002", "closeTime":   "1694876914787", "dailyChange": "0.0152", "displayName": "DASH/BTC", "high":"0.001015", "low":"0.000987", "markPrice":   "0.001007", "open":"0.000987", "quantity":"3.32", "startTime":"1694790480000", "symbol":"DASH_BTC", "tradeCount": "8", "ts":"1694876923780"}]`,
	}}
	var params = []url.Values{
		{
			"amount":      []string{"0.003747662"},
			"ask":         []string{"0.000000356"},
			"askQuantity": []string{"13274"},
			"bid":         []string{"0.0000003365"},
			"bidQuantity": []string{"2674"},
			"close":       []string{"0.0000003373"},
			"closeTime":   []string{"1694855078036"},
			"dailyChange": []string{"0.0224"},
			"displayName": []string{"BTS/BTC"},
			"high":        []string{"0.0000003373"},
			"low":         []string{"0.0000003299"},
			"markPrice":   []string{"0.00000034"},
			"open":        []string{"0.0000003299"},
			"quantity":    []string{"11111"},
			"startTime":   []string{"1694768640000"},
			"symbol":      []string{"BTS_BTC"},
			"tradeCount":  []string{"2"},
			"ts":          []string{"1694855083780"},
		},
		{
			"amount":      []string{"0.00334476"},
			"ask":         []string{"0.001009"},
			"askQuantity": []string{"4.49"},
			"bid":         []string{"0.001006"},
			"bidQuantity": []string{"4.48"},
			"close":       []string{"0.001002"},
			"closeTime":   []string{"1694876914787"},
			"dailyChange": []string{"0.0152"},
			"displayName": []string{"DASH/BTC"},
			"high":        []string{"0.001015"},
			"low":         []string{"0.000987"},
			"markPrice":   []string{"0.001007"},
			"open":        []string{"0.000987"},
			"quantity":    []string{"3.32"},
			"startTime":   []string{"1694790480000"},
			"symbol":      []string{"DASH_BTC"},
			"tradeCount":  []string{"8"},
			"ts":          []string{"1694876923780"},
		},
	}
	_, err := MatchBatchAndGetResponse(response, params)
	if err != nil {
		t.Error(err)
	}
}
