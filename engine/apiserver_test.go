package engine

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
)

func TestSetupAPIServerManager(t *testing.T) {
	t.Parallel()
	_, err := setupAPIServerManager(nil, nil, nil, nil, nil, "")
	if !errors.Is(err, errNilRemoteConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilRemoteConfig)
	}

	_, err = setupAPIServerManager(&config.RemoteControlConfig{}, nil, nil, nil, nil, "")
	if !errors.Is(err, errNilPProfConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilPProfConfig)
	}

	_, err = setupAPIServerManager(&config.RemoteControlConfig{}, &config.Profiler{}, nil, nil, nil, "")
	if !errors.Is(err, errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilExchangeManager)
	}

	_, err = setupAPIServerManager(&config.RemoteControlConfig{}, &config.Profiler{}, &ExchangeManager{}, nil, nil, "")
	if !errors.Is(err, errNilBot) {
		t.Errorf("error '%v', expected '%v'", err, errNilBot)
	}

	_, err = setupAPIServerManager(&config.RemoteControlConfig{}, &config.Profiler{}, &ExchangeManager{}, &fakeBot{}, nil, "")
	if !errors.Is(err, errEmptyConfigPath) {
		t.Errorf("error '%v', expected '%v'", err, errEmptyConfigPath)
	}

	wd, _ := os.Getwd()
	_, err = setupAPIServerManager(&config.RemoteControlConfig{}, &config.Profiler{}, &ExchangeManager{}, &fakeBot{}, nil, wd)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestStartRESTServer(t *testing.T) {
	t.Parallel()
	wd, _ := os.Getwd()
	m, err := setupAPIServerManager(&config.RemoteControlConfig{}, &config.Profiler{}, &ExchangeManager{}, &fakeBot{}, nil, wd)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.StartRESTServer()
	if !errors.Is(err, errServerDisabled) {
		t.Errorf("error '%v', expected '%v'", err, errServerDisabled)
	}
	m.remoteConfig.DeprecatedRPC.Enabled = true
	err = m.StartRESTServer()
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartWebsocketServer(t *testing.T) {
	t.Parallel()
	wd, _ := os.Getwd()
	m, err := setupAPIServerManager(&config.RemoteControlConfig{}, &config.Profiler{}, &ExchangeManager{}, &fakeBot{}, nil, wd)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.StartWebsocketServer()
	if !errors.Is(err, errServerDisabled) {
		t.Errorf("error '%v', expected '%v'", err, errServerDisabled)
	}
	m.remoteConfig.WebsocketRPC.Enabled = true
	err = m.StartWebsocketServer()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestStopRESTServer(t *testing.T) {
	t.Parallel()
	wd, _ := os.Getwd()
	m, err := setupAPIServerManager(&config.RemoteControlConfig{
		DeprecatedRPC: config.DepcrecatedRPCConfig{
			Enabled:       true,
			ListenAddress: "localhost:9051",
		},
	}, &config.Profiler{}, &ExchangeManager{}, &fakeBot{}, nil, wd)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.StopRESTServer()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	err = m.StartRESTServer()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.StopRESTServer()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	// do it again to ensure things have reset appropriately and no errors occur starting
	err = m.StartRESTServer()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.StopRESTServer()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestWebsocketStop(t *testing.T) {
	t.Parallel()
	wd, _ := os.Getwd()
	m, err := setupAPIServerManager(&config.RemoteControlConfig{
		WebsocketRPC: config.WebsocketRPCConfig{
			Enabled:       true,
			ListenAddress: "localhost:9052",
		},
	}, &config.Profiler{}, &ExchangeManager{}, &fakeBot{}, nil, wd)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.StopWebsocketServer()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	err = m.StartWebsocketServer()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.StopWebsocketServer()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	// do it again to ensure things have reset appropriately and no errors occur starting
	err = m.StartWebsocketServer()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.StopWebsocketServer()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestIsRESTServerRunning(t *testing.T) {
	t.Parallel()
	m := &apiServerManager{}
	if m.IsRESTServerRunning() {
		t.Error("expected false")
	}
	m.restStarted = 1
	if !m.IsRESTServerRunning() {
		t.Error("expected true")
	}
	m = nil
	if m.IsRESTServerRunning() {
		t.Error("expected false")
	}
}

func TestIsWebsocketServerRunning(t *testing.T) {
	t.Parallel()
	m := &apiServerManager{}
	if m.IsWebsocketServerRunning() {
		t.Error("expected false")
	}
	m.websocketStarted = 1
	if !m.IsWebsocketServerRunning() {
		t.Error("expected true")
	}
	m = nil
	if m.IsWebsocketServerRunning() {
		t.Error("expected false")
	}
}

func TestGetAllActiveOrderbooks(t *testing.T) {
	man := SetupExchangeManager()
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
	man := SetupExchangeManager()
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
	man := SetupExchangeManager()
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
	if err != nil {
		t.Error(err)
	}
	resp := makeHTTPGetRequest(t, c)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error("Body not readable", err)
	}
	err = resp.Body.Close()
	if err != nil {
		t.Error("Body not closable", err)
	}

	var responseConfig config.Config
	jsonErr := json.Unmarshal(body, &responseConfig)
	if jsonErr != nil {
		t.Error("Response not parse-able as json", err)
	}

	if !reflect.DeepEqual(responseConfig, c) {
		t.Error("Json not equal to config")
	}
}

// fakeBot is a basic implementation of the iBot interface used for testing
type fakeBot struct{}

// SetupExchanges is a basic implementation of the iBot interface used for testing
func (f *fakeBot) SetupExchanges() error {
	return nil
}
