package engine

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
)

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
	bot := CreateTestBot(t)
	resp := makeHTTPGetRequest(t, bot.Config)
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

	if reflect.DeepEqual(responseConfig, bot.Config) {
		t.Error("Json not equal to config")
	}
}

func TestInvalidHostRequest(t *testing.T) {
	e := CreateTestBot(t)
	req, err := http.NewRequest(http.MethodGet, "/config/all", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "invalidsite.com"

	resp := httptest.NewRecorder()
	newRouter(e, true).ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusNotFound {
		t.Errorf("Response returned wrong status code expected %v got %v", http.StatusNotFound, status)
	}
}

func TestValidHostRequest(t *testing.T) {
	e := CreateTestBot(t)
	if config.Cfg.Name == "" {
		config.Cfg = *e.Config
	}
	req, err := http.NewRequest(http.MethodGet, "/config/all", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "localhost:9050"

	resp := httptest.NewRecorder()
	newRouter(e, true).ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusOK {
		t.Errorf("Response returned wrong status code expected %v got %v", http.StatusOK, status)
	}
}

func TestProfilerEnabledShouldEnableProfileEndPoint(t *testing.T) {
	e := CreateTestBot(t)
	req, err := http.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Host = "localhost:9050"
	resp := httptest.NewRecorder()
	newRouter(e, true).ServeHTTP(resp, req)
	if status := resp.Code; status != http.StatusNotFound {
		t.Errorf("Response returned wrong status code expected %v got %v", http.StatusNotFound, status)
	}

	e.Config.Profiler.Enabled = true
	e.Config.Profiler.MutexProfileFraction = 5
	req, err = http.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	if err != nil {
		t.Fatal(err)
	}

	mutexValue := runtime.SetMutexProfileFraction(10)
	if mutexValue != 0 {
		t.Fatalf("SetMutexProfileFraction() should be 0 on first set received: %v", mutexValue)
	}

	resp = httptest.NewRecorder()
	newRouter(e, true).ServeHTTP(resp, req)
	mutexValue = runtime.SetMutexProfileFraction(10)
	if mutexValue != 5 {
		t.Fatalf("SetMutexProfileFraction() should be 5 after setup received: %v", mutexValue)
	}
	if status := resp.Code; status != http.StatusOK {
		t.Errorf("Response returned wrong status code expected %v got %v", http.StatusOK, status)
	}
}
