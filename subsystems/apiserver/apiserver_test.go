package apiserver

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"
)

func TestSetup(t *testing.T) {
	t.Parallel()
	_, err := Setup(nil, nil, nil, nil, "")
	if !errors.Is(err, errNilRemoteConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilRemoteConfig)
	}

	_, err = Setup(&config.RemoteControlConfig{}, nil, nil, nil, "")
	if !errors.Is(err, errNilPProfConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilPProfConfig)
	}

	_, err = Setup(&config.RemoteControlConfig{}, &config.Profiler{}, nil, nil, "")
	if !errors.Is(err, errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilExchangeManager)
	}

	_, err = Setup(&config.RemoteControlConfig{}, &config.Profiler{}, &exchangemanager.Manager{}, nil, "")
	if !errors.Is(err, errNilBot) {
		t.Errorf("error '%v', expected '%v'", err, errNilBot)
	}

	_, err = Setup(&config.RemoteControlConfig{}, &config.Profiler{}, &exchangemanager.Manager{}, &fakeBot{}, "")
	if !errors.Is(err, errEmptyConfigPath) {
		t.Errorf("error '%v', expected '%v'", err, errEmptyConfigPath)
	}

	wd, _ := os.Getwd()
	_, err = Setup(&config.RemoteControlConfig{}, &config.Profiler{}, &exchangemanager.Manager{}, &fakeBot{}, wd)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestStartRESTServer(t *testing.T) {
	t.Parallel()
	wd, _ := os.Getwd()
	m, err := Setup(&config.RemoteControlConfig{}, &config.Profiler{}, &exchangemanager.Manager{}, &fakeBot{}, wd)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.StartRESTServer()
	if !errors.Is(err, errServerDisabled) {
		t.Errorf("error '%v', expected '%v'", err, errServerDisabled)
	}
	m.remoteConfig.DeprecatedRPC.Enabled = true
	var wg sync.WaitGroup
	wg.Add(1)
	// this is difficult to test as a webserver actually starts, so quit if an immediate error is not received
	err = m.StartRESTServer()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	wg.Done()
}

func TestStartWebsocketServer(t *testing.T) {
	t.Parallel()
	wd, _ := os.Getwd()
	m, err := Setup(&config.RemoteControlConfig{}, &config.Profiler{}, &exchangemanager.Manager{}, &fakeBot{}, wd)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.StartWebsocketServer()
	if !errors.Is(err, errServerDisabled) {
		t.Errorf("error '%v', expected '%v'", err, errServerDisabled)
	}
	m.remoteConfig.WebsocketRPC.Enabled = true
	err = m.StartWebsocketServer()
	if err != nil && !strings.Contains(err.Error(), "invalid port") {
		t.Error(err)
	} else if err == nil {
		t.Error("expected invalid port error")
	}
	m.websocketListenAddress = "localhost:9051"
	var wg sync.WaitGroup
	wg.Add(1)
	// this is difficult to test as a webserver actually starts, so quit if an immediate error is not received
	err = m.StartWebsocketServer()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	wg.Done()
}

func TestStop(t *testing.T) {
	t.Parallel()
	wd, _ := os.Getwd()
	m, err := Setup(&config.RemoteControlConfig{}, &config.Profiler{}, &exchangemanager.Manager{}, &fakeBot{}, wd)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Stop()
	if !errors.Is(err, subsystems.ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemNotStarted)
	}
	m.started = 1
	m.websocketHttpServer = &http.Server{}
	m.restHttpServer = &http.Server{}
	err = m.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m.restHttpServer != nil {
		t.Error("expected nil")
	}
}

func TestIsRunning(t *testing.T) {
	t.Parallel()
	m := &Manager{}
	if m.IsRunning() {
		t.Error("expected false")
	}
	m.started = 1
	if !m.IsRunning() {
		t.Error("expected true")
	}
	m = nil
	if m.IsRunning() {
		t.Error("expected false")
	}
}

func TestGetAllActiveOrderbooks(t *testing.T) {
	man := exchangemanager.Setup()
	bs, err := man.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	bs.SetDefaults()
	man.Add(bs)
	resp := getAllActiveOrderbooks(man)
	if resp == nil {
		t.Error("expected not nil")
	}
}

func TestGetAllActiveTickers(t *testing.T) {
	t.Parallel()
	man := exchangemanager.Setup()
	bs, err := man.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	bs.SetDefaults()
	man.Add(bs)
	resp := getAllActiveTickers(man)
	if resp == nil {
		t.Error("expected not nil")
	}
}

func TestGetAllActiveAccounts(t *testing.T) {
	t.Parallel()
	man := exchangemanager.Setup()
	bs, err := man.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	bs.SetDefaults()
	man.Add(bs)
	resp := getAllActiveAccounts(man)
	if resp == nil {
		t.Error("expected not nil")
	}
}

func makeHTTPGetRequest(t *testing.T, response interface{}) *http.Response {
	w := httptest.NewRecorder()

	err := writeResponse(w, response)
	if err != nil {
		t.Error("Failed to make response.", err)
	}
	return w.Result()
}

// TestConfigAllJsonResponse test if config/all restful json response is valid
func TestConfigAllJsonResponse(t *testing.T) {
	t.Parallel()
	var c config.Config
	err := c.LoadConfig(config.TestFile, true)
	resp := makeHTTPGetRequest(t, c)
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

	if reflect.DeepEqual(responseConfig, c) {
		t.Error("Json not equal to config")
	}
}

// fakeBot is a basic implementation of the iBot interface used for testing
type fakeBot struct{}

// SetupExchanges is a basic implementation of the iBot interface used for testing
func (f *fakeBot) SetupExchanges() error {
	return nil
}
