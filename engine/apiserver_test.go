package engine

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestSetupAPIServerManager(t *testing.T) {
	t.Parallel()
	_, err := setupAPIServerManager(nil, nil, nil, nil, nil, "")
	assert.ErrorIs(t, err, errNilRemoteConfig)

	_, err = setupAPIServerManager(&config.RemoteControlConfig{}, nil, nil, nil, nil, "")
	assert.ErrorIs(t, err, errNilPProfConfig)

	_, err = setupAPIServerManager(&config.RemoteControlConfig{}, &config.Profiler{}, nil, nil, nil, "")
	assert.ErrorIs(t, err, errNilExchangeManager)

	_, err = setupAPIServerManager(&config.RemoteControlConfig{}, &config.Profiler{}, &ExchangeManager{}, nil, nil, "")
	assert.ErrorIs(t, err, errNilBot)

	_, err = setupAPIServerManager(&config.RemoteControlConfig{}, &config.Profiler{}, &ExchangeManager{}, &fakeBot{}, nil, "")
	assert.ErrorIs(t, err, errEmptyConfigPath)

	wd, _ := os.Getwd()
	_, err = setupAPIServerManager(&config.RemoteControlConfig{}, &config.Profiler{}, &ExchangeManager{}, &fakeBot{}, nil, wd)
	assert.NoError(t, err)
}

func TestStartRESTServer(t *testing.T) {
	t.Parallel()
	wd, _ := os.Getwd()
	m, err := setupAPIServerManager(&config.RemoteControlConfig{}, &config.Profiler{}, &ExchangeManager{}, &fakeBot{}, nil, wd)
	assert.NoError(t, err)

	err = m.StartRESTServer()
	assert.ErrorIs(t, err, errServerDisabled)

	m.remoteConfig.DeprecatedRPC.Enabled = true
	err = m.StartRESTServer()
	assert.NoError(t, err)
}

func TestStartWebsocketServer(t *testing.T) {
	t.Parallel()
	wd, _ := os.Getwd()
	m, err := setupAPIServerManager(&config.RemoteControlConfig{}, &config.Profiler{}, &ExchangeManager{}, &fakeBot{}, nil, wd)
	assert.NoError(t, err)

	err = m.StartWebsocketServer()
	assert.ErrorIs(t, err, errServerDisabled)

	m.remoteConfig.WebsocketRPC.Enabled = true
	err = m.StartWebsocketServer()
	assert.NoError(t, err)
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
	assert.NoError(t, err)

	err = m.StopRESTServer()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	err = m.StartRESTServer()
	assert.NoError(t, err)

	err = m.StopRESTServer()
	assert.NoError(t, err)

	// do it again to ensure things have reset appropriately and no errors occur starting
	err = m.StartRESTServer()
	assert.NoError(t, err)

	err = m.StopRESTServer()
	assert.NoError(t, err)
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
	assert.NoError(t, err)

	err = m.StopWebsocketServer()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	err = m.StartWebsocketServer()
	assert.NoError(t, err)

	err = m.StopWebsocketServer()
	assert.NoError(t, err)

	// do it again to ensure things have reset appropriately and no errors occur starting
	err = m.StartWebsocketServer()
	assert.NoError(t, err)

	err = m.StopWebsocketServer()
	assert.NoError(t, err)
}

func TestIsRESTServerRunning(t *testing.T) {
	t.Parallel()
	m := &apiServerManager{}
	assert.False(t, m.IsRESTServerRunning(), "should return correctly with empty type")
	m.restStarted = 1
	assert.True(t, m.IsRESTServerRunning(), "should return correctly with restStarted set")
	assert.False(t, (*apiServerManager)(nil).IsRESTServerRunning(), "should return correctly on nil type")
}

func TestIsWebsocketServerRunning(t *testing.T) {
	t.Parallel()
	m := &apiServerManager{}
	assert.False(t, m.IsWebsocketServerRunning(), "should return correctly with empty type")
	m.websocketStarted = 1
	assert.True(t, m.IsWebsocketServerRunning(), "should return correctly with websocketStarted set")
	assert.False(t, (*apiServerManager)(nil).IsWebsocketServerRunning(), "should return correctly on nil type")
}

func TestGetAllActiveOrderbooks(t *testing.T) {
	man := NewExchangeManager()
	bs, err := man.NewExchangeByName("Bitstamp")
	require.NoError(t, err, "NewExchangeByName must not error")
	bs.SetDefaults()
	err = man.Add(bs)
	require.NoError(t, err)

	resp := getAllActiveOrderbooks(man)
	assert.NotNil(t, resp)
}

func TestGetAllActiveTickers(t *testing.T) {
	t.Parallel()
	man := NewExchangeManager()
	bs, err := man.NewExchangeByName("Bitstamp")
	require.NoError(t, err, "NewExchangeByName must not error")
	bs.SetDefaults()
	err = man.Add(bs)
	require.NoError(t, err)

	resp := getAllActiveTickers(man)
	assert.NotNil(t, resp)
}

func TestGetAllActiveAccounts(t *testing.T) {
	t.Parallel()
	man := NewExchangeManager()
	bs, err := man.NewExchangeByName("Bitstamp")
	require.NoError(t, err, "NewExchangeByName must not error")
	bs.SetDefaults()
	err = man.Add(bs)
	require.NoError(t, err)

	resp := getAllActiveAccounts(man)
	assert.NotNil(t, resp)
}

func makeHTTPGetRequest(t *testing.T, response any) *http.Response {
	t.Helper()
	w := httptest.NewRecorder()

	err := writeResponse(w, response)
	require.NoError(t, err)

	return w.Result()
}

// TestConfigAllJsonResponse test if config/all restful json response is valid
func TestConfigAllJsonResponse(t *testing.T) {
	t.Parallel()
	var c config.Config
	err := c.LoadConfig(config.TestFile, true)
	assert.NoError(t, err, "LoadConfig should not error")

	resp := makeHTTPGetRequest(t, c)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "ReadAll should not error")
	err = resp.Body.Close()
	assert.NoError(t, err, "Close body should not error")

	var responseConfig config.Config
	err = json.Unmarshal(body, &responseConfig)
	assert.NoError(t, err, "Unmarshal should not error")
	for i, e := range responseConfig.Exchanges {
		err = e.CurrencyPairs.SetDelimitersFromConfig()
		assert.NoError(t, err, "SetDelimitersFromConfig should not error")
		// Using require here makes it much easier to isolate differences per-exchange than below
		// We look into pointers separately
		for a, p := range e.CurrencyPairs.Pairs {
			require.Equalf(t, c.Exchanges[i].CurrencyPairs.Pairs[a], p, "%s exchange Config CurrencyManager Pairs for asset %s must match api response", e.Name, a)
		}
		require.Equalf(t, c.Exchanges[i].CurrencyPairs, e.CurrencyPairs, "%s exchange Config CurrencyManager must match api response", e.Name)
		require.Equalf(t, c.Exchanges[i], e, "%s exchange Config must match api response", e.Name) // require here makes it much easier to isolate differences than below
	}
	assert.Equal(t, c, responseConfig, "Config should match api response")
}

// fakeBot is a basic implementation of the iBot interface used for testing
type fakeBot struct{}

// SetupExchanges is a basic implementation of the iBot interface used for testing
func (f *fakeBot) SetupExchanges() error {
	return nil
}
