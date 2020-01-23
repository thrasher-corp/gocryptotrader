package engine

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
)

func loadConfig(t *testing.T) *config.Config {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("", true)
	if err != nil {
		t.Error("GetCurrencyConfig LoadConfig error", err)
	}
	configLoaded = true
	return cfg
}

func makeHTTPGetRequest(t *testing.T, response interface{}) *http.Response {
	w := httptest.NewRecorder()

	err := RESTfulJSONResponse(w, response)
	if err != nil {
		t.Error("Failed to make response.", err)
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
		t.Error("Body not readable", err)
	}

	var responseConfig config.Config
	jsonErr := json.Unmarshal(body, &responseConfig)
	if jsonErr != nil {
		t.Error("Response not parseable as json", err)
	}

	if reflect.DeepEqual(responseConfig, cfg) {
		t.Error("Json not equal to config")
	}
}

func TestInvalidHostRequest(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/config/all", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "invalidsite.com"

	resp := httptest.NewRecorder()
	newRouter(true).ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusNotFound {
		t.Errorf("Response returned wrong status code expected %v got %v", http.StatusNotFound, status)
	}
}

func TestValidHostRequest(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/config/all", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "localhost:9050"

	resp := httptest.NewRecorder()
	newRouter(true).ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusOK {
		t.Errorf("Response returned wrong status code expected %v got %v", http.StatusOK, status)
	}
}
