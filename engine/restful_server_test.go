package engine

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

func loadConfig(t *testing.T) *config.Config {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("")
	if err != nil {
		t.Error("Test failed. GetCurrencyConfig LoadConfig error", err)
	}
	return cfg
}

func makeHTTPGetRequest(t *testing.T, response interface{}) *http.Response {
	w := httptest.NewRecorder()

	err := RESTfulJSONResponse(w, response)
	if err != nil {
		t.Error("Test failed. Failed to make response.", err)
	}
	return w.Result()
}

// TestConfigAllJsonResponse test if config/all restful json response is valid
func TestConfigAllJsonResponse(t *testing.T) {
	cfg := loadConfig(t)
	resp := makeHTTPGetRequest(t, cfg)
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Error("Test failed. Body not readable", err)
	}

	var responseConfig config.Config
	jsonErr := json.Unmarshal(body, &responseConfig)
	if jsonErr != nil {
		t.Error("Test failed. Response not parseable as json", err)
	}

	if reflect.DeepEqual(responseConfig, cfg) {
		t.Error("Test failed. Json not equal to config")
	}
}

func TestInvalidHostRequest(t *testing.T) {
	t.Parallel()
	req, err := http.NewRequest(http.MethodGet, "/config/all", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "invalidsite.com"

	resp := httptest.NewRecorder()
	newRouter(true).ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusNotFound {
		t.Errorf("Test failed. Response returned wrong status code expected %v got %v", http.StatusNotFound, status)
	}
}

func TestValidHostRequest(t *testing.T) {
	t.Parallel()
	req, err := http.NewRequest(http.MethodGet, "/config/all", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = common.ExtractHost(Bot.Config.RemoteControl.DeprecatedRPC.ListenAddress)

	resp := httptest.NewRecorder()
	newRouter(true).ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusOK {
		t.Errorf("Test failed. Response returned wrong status code expected %v got %v", http.StatusOK, status)
	}
}
