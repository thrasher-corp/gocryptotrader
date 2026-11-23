package htx

// Live credentials can be set with apiKey and apiSecret. When both are empty,
// HTX_API_KEY and HTX_API_SECRET supply them. Account changes also require canManipulateRealOrders.

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

const (
	apiKey    = ""
	apiSecret = ""
	// canManipulateRealOrders explicitly enables live tests that change orders, positions or account state.
	canManipulateRealOrders = false
)

var (
	apiCredentials    = liveCredentials(apiKey, apiSecret)
	htxLiveMutationMu sync.RWMutex
)

type htxLiveOptions struct {
	public         bool
	websocket      bool
	changesAccount bool
}

// liveCredentials keeps an explicitly configured key pair together when choosing the credential source.
func liveCredentials(key, secret string) *accounts.Credentials {
	if key == "" && secret == "" {
		key, secret = os.Getenv("HTX_API_KEY"), os.Getenv("HTX_API_SECRET")
	}
	return &accounts.Credentials{Key: key, Secret: secret}
}

func TestLiveCredentials(t *testing.T) {
	// Environment settings are process-wide.
	t.Setenv("HTX_API_KEY", "environment-key")
	t.Setenv("HTX_API_SECRET", "environment-secret")
	for _, tc := range []struct {
		name, key, secret, wantKey, wantSecret string
	}{
		{name: "environment fallback", wantKey: "environment-key", wantSecret: "environment-secret"},
		{name: "configured pair", key: "configured-key", secret: "configured-secret", wantKey: "configured-key", wantSecret: "configured-secret"},
		{name: "configured key only", key: "configured-key", wantKey: "configured-key"},
		{name: "configured secret only", secret: "configured-secret", wantSecret: "configured-secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := liveCredentials(tc.key, tc.secret)
			assert.Equal(t, &accounts.Credentials{Key: tc.wantKey, Secret: tc.wantSecret}, got, "credentials should come from one configured source")
		})
	}
	t.Run("unset sources", func(t *testing.T) {
		t.Setenv("HTX_API_KEY", "")
		t.Setenv("HTX_API_SECRET", "")
		assert.Equal(t, new(accounts.Credentials), liveCredentials("", ""), "missing sources should leave credentials unset")
	})
}

// newHTXLiveExchange keeps credentials off the shared instance used by the ordinary suite.
func newHTXLiveExchange(t *testing.T, creds *accounts.Credentials, options htxLiveOptions) *Exchange {
	t.Helper()
	h := new(Exchange)
	h.SetDefaults()
	h.SetCredentials(creds)
	if options.changesAccount {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
		htxLiveMutationMu.Lock()
		t.Cleanup(htxLiveMutationMu.Unlock)
	} else if !options.public {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
		htxLiveMutationMu.RLock()
		t.Cleanup(htxLiveMutationMu.RUnlock)
	}

	require.NoError(t, testexch.Setup(h), "live HTX exchange must initialise")
	if options.websocket {
		cfg := h.Config
		cfg.API.AuthenticatedSupport = !options.public
		cfg.API.AuthenticatedWebsocketSupport = !options.public
		h.SetDefaults()
		require.NoError(t, h.Setup(cfg), "live HTX websocket connections must initialise")
	}
	h.API.AuthenticatedSupport = !options.public
	h.SetCredentials(creds)
	return h
}

func TestNewHTXLiveExchange(t *testing.T) {
	t.Parallel()
	t.Run("public requests without credentials", func(t *testing.T) {
		t.Parallel()
		h := newHTXLiveExchange(t, new(accounts.Credentials), htxLiveOptions{public: true})
		assert.False(t, h.API.AuthenticatedSupport, "public requests should keep authentication disabled")
	})
	t.Run("account mutation gate", func(t *testing.T) {
		t.Parallel()
		var h *Exchange
		// Run synchronously so the caller can distinguish a skipped setup from a returned instance.
		require.True(t, t.Run("setup", func(t *testing.T) {
			h = newHTXLiveExchange(t, &accounts.Credentials{Key: "test-key", Secret: "test-secret"}, htxLiveOptions{changesAccount: true})
		}), "mutation setup must complete or skip without failing")
		if !canManipulateRealOrders {
			assert.Nil(t, h, "closed mutation gate should prevent returning an authenticated instance")
			return
		}
		require.NotNil(t, h, "open mutation gate must allow setup")
		assert.True(t, h.API.AuthenticatedSupport, "mutation setup should authenticate when the gate is open")
	})
	t.Run("missing credentials", func(t *testing.T) {
		t.Parallel()
		var h *Exchange
		require.True(t, t.Run("setup", func(t *testing.T) {
			h = newHTXLiveExchange(t, new(accounts.Credentials), htxLiveOptions{})
		}), "unauthenticated setup must skip without failing")
		assert.Nil(t, h, "missing credentials should prevent returning a live instance")
	})
	for _, useWebsocket := range []bool{false, true} {
		name := "REST"
		if useWebsocket {
			name = "websocket"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			creds := &accounts.Credentials{Key: "test-key", Secret: "test-secret"}
			h := newHTXLiveExchange(t, creds, htxLiveOptions{websocket: useWebsocket})
			assert.True(t, h.API.AuthenticatedSupport, "REST authentication should be enabled")
			assert.Equal(t, useWebsocket, h.Websocket.CanUseAuthenticatedEndpoints(), "websocket authentication should follow the requested mode")
			assert.Equal(t, *creds, *h.GetDefaultCredentials(), "live instance should retain the provided placeholder credentials")
			assert.False(t, sharedtestvalues.AreAPICredentialsSet(e), "the shared test instance should remain unauthenticated")
		})
	}
}
