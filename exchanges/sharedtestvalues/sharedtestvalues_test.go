package sharedtestvalues

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// baseEx is a CustomEx that exposes its Base, as a real exchange does
type baseEx struct {
	CustomEx
}

func (b *baseEx) GetBase() *exchange.Base {
	return &b.Base
}

func TestNewTestWebsocket(t *testing.T) {
	t.Parallel()
	w := NewTestWebsocket()
	require.NotNil(t, w, "NewTestWebsocket must return a manager")
	require.NotNil(t, w.DataHandler, "NewTestWebsocket must give the manager a DataHandler")
	assert.Equal(t, WebsocketChannelOverrideCapacity, cap(w.DataHandler.C), "NewTestWebsocket should give the DataHandler the test capacity")
}

func TestSkipLiveTestIfCredentialsUnset(t *testing.T) {
	t.Parallel()
	const restURL = "https://api.example.invalid"
	requester, err := request.New("customex", new(http.Client), request.WithLimiter(request.NewBasicRateLimit(time.Second, 1, 1)))
	require.NoError(t, err, "request.New must not error")
	publicWebsocket, authenticatedWebsocket := NewTestWebsocket(), NewTestWebsocket()
	authenticatedWebsocket.SetCanUseAuthenticatedEndpoints(true)
	// newExchange returns an exchange with what testexch gives a real one: its test config entry, a requester with rate
	// limits, a REST endpoint and a websocket, requiring a key and a secret as Bitstamp, LBank and Poloniex do
	newExchange := func(t *testing.T) *baseEx {
		t.Helper()
		exch := new(baseEx)
		exch.Config = &config.Exchange{
			Enabled: true,
			API: config.APIConfig{
				Credentials:          config.APICredentialsConfig{Key: config.DefaultAPIKey, Secret: config.DefaultAPISecret},
				CredentialsValidator: &config.APICredentialsValidatorConfig{RequiresKey: true, RequiresSecret: true},
				Endpoints:            map[string]string{exchange.RestSpot.String(): restURL},
			},
		}
		exch.API.CredentialsValidator.RequiresKey, exch.API.CredentialsValidator.RequiresSecret = true, true
		exch.Requester = requester
		exch.API.Endpoints = exch.NewEndpoints()
		require.NoError(t, exch.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), restURL), "SetRunningURL must not error")
		exch.Websocket = publicWebsocket
		return exch
	}
	// Only the mode, recording, credentials and permission decide a skip, so each case runs under every combination of
	// the exchange's other settings
	otherSettings := []struct {
		name string
		set  func(*exchange.Base)
	}{
		{name: "LoadedByConfig", set: func(b *exchange.Base) { b.LoadedByConfig = true }},
		{name: "SkipAuthCheck", set: func(b *exchange.Base) { b.SkipAuthCheck = true }},
		{name: "Verbose", set: func(b *exchange.Base) { b.Verbose = true }},
		{name: "HTTPDebugging", set: func(b *exchange.Base) { b.HTTPDebugging = true }},
		{name: "AuthenticatedSupport", set: func(b *exchange.Base) { b.API.AuthenticatedSupport = true }},
		{name: "AuthenticatedWebsocketSupport", set: func(b *exchange.Base) { b.API.AuthenticatedWebsocketSupport = true }},
		{name: "authenticated websocket", set: func(b *exchange.Base) { b.Websocket = authenticatedWebsocket }},
	}
	type credentials struct {
		name   string
		values *accounts.Credentials
		valid  bool
	}
	check := func(t *testing.T, exch exchange.IBotExchange, mockTests bool, creds credentials, canManipulateOrders []bool, skip bool) {
		t.Helper()
		if creds.values != nil {
			exch.SetCredentials(creds.values)
		}
		require.Equal(t, creds.valid, AreAPICredentialsSet(exch), "AreAPICredentialsSet must match the case's credentials")
		var ran, skipped bool
		// Not parallel, as skipped is read as soon as the probe returns
		t.Run("probe", func(t *testing.T) {
			ran = true
			t.Cleanup(func() { skipped = t.Skipped() })
			SkipLiveTestIfCredentialsUnset(t, exch, mockTests, canManipulateOrders...)
		})
		if ran {
			assert.Equal(t, skip, skipped, "SkipLiveTestIfCredentialsUnset should skip only a live or recording test it cannot run")
		}
	}
	for _, mode := range []struct {
		name      string
		mockTests bool
		recording bool
	}{
		{name: "mocked", mockTests: true},
		{name: "mocked while recording", mockTests: true, recording: true},
		{name: "live"},
		{name: "live while recording", recording: true},
	} {
		for _, creds := range []credentials{
			{name: "without credentials"},
			{name: "with credentials", values: &accounts.Credentials{Key: "key", Secret: "secret"}, valid: true},
			{name: "with the config's placeholder credentials", values: &accounts.Credentials{Key: config.DefaultAPIKey, Secret: config.DefaultAPISecret}},
			{name: "with the config's placeholder key", values: &accounts.Credentials{Key: config.DefaultAPIKey, Secret: "secret"}},
			{name: "with the config's placeholder secret", values: &accounts.Credentials{Key: "key", Secret: config.DefaultAPISecret}},
		} {
			for _, permission := range []struct {
				name                string
				canManipulateOrders []bool
			}{
				{name: "and no permission flag"},
				{name: "and no permission to manipulate orders", canManipulateOrders: []bool{false}},
				{name: "and permission to manipulate orders", canManipulateOrders: []bool{true}},
			} {
				// A live or recording test runs only with valid credentials and, where it asks for it, permission
				skip := (!mode.mockTests || mode.recording) && (!creds.valid || slices.Contains(permission.canManipulateOrders, false))
				t.Run(mode.name+" "+creds.name+" "+permission.name, func(t *testing.T) {
					t.Parallel()
					for combination := range 1 << len(otherSettings) {
						exch := newExchange(t)
						settings := make([]string, 0, len(otherSettings))
						for i, s := range otherSettings {
							if combination&(1<<i) != 0 {
								s.set(&exch.Base)
								settings = append(settings, s.name)
							}
						}
						exch.HTTPRecording = mode.recording
						name := "no other settings"
						if len(settings) > 0 {
							name = strings.Join(settings, " and ")
						}
						// Not parallel, as running the combinations in parallel quadruples the test's memory under -race
						// without saving time
						t.Run(name, func(t *testing.T) {
							check(t, exch, mode.mockTests, creds, permission.canManipulateOrders, skip)
						})
					}
				})
			}
		}
	}
	for _, tc := range []struct {
		name      string
		mockTests bool
		skip      bool
	}{
		{name: "mocked on an exchange without a base", mockTests: true},
		{name: "live on an exchange without a base", skip: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			check(t, new(CustomEx), tc.mockTests, credentials{}, nil, tc.skip)
		})
	}
}

// BenchmarkNewTestWebsocket guards against the manager allocating a default DataHandler that NewTestWebsocket discards,
// which every test exchange set up through testexch.Setup pays for
func BenchmarkNewTestWebsocket(b *testing.B) {
	for b.Loop() {
		_ = NewTestWebsocket()
	}
}
