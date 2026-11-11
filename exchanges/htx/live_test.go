package htx

// Authenticated live cases use HTX_API_KEY and HTX_API_SECRET. Cases that change
// account state also require canManipulateRealOrders.

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

// canManipulateRealOrders explicitly enables live tests that change orders, positions or account state.
const canManipulateRealOrders = false

var (
	apiCredentials = &accounts.Credentials{
		Key:    os.Getenv("HTX_API_KEY"),
		Secret: os.Getenv("HTX_API_SECRET"),
	}
	htxLiveMutationMu sync.RWMutex
)

type htxLiveOptions struct {
	websocket      bool
	changesAccount bool
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
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
		htxLiveMutationMu.RLock()
		t.Cleanup(htxLiveMutationMu.RUnlock)
	}

	require.NoError(t, testexch.Setup(h), "live HTX exchange must initialise")
	if options.websocket {
		cfg := h.Config
		cfg.API.AuthenticatedSupport = true
		cfg.API.AuthenticatedWebsocketSupport = true
		h.SetDefaults()
		require.NoError(t, h.Setup(cfg), "live HTX websocket connections must initialise")
	}
	h.API.AuthenticatedSupport = true
	h.SetCredentials(creds)
	return h
}

func TestNewHTXLiveExchange(t *testing.T) {
	t.Parallel()
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
