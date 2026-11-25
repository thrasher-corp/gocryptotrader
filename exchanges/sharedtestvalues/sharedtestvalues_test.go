package sharedtestvalues

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
)

func TestNewTestWebsocket(t *testing.T) {
	t.Parallel()
	w := NewTestWebsocket()
	require.NotNil(t, w, "NewTestWebsocket must return a manager")
	require.NotNil(t, w.DataHandler, "NewTestWebsocket must give the manager a DataHandler")
	assert.Equal(t, WebsocketChannelOverrideCapacity, cap(w.DataHandler.C), "NewTestWebsocket should give the DataHandler the test capacity")
}

func TestSkipLiveTestIfCredentialsUnset(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                string
		mockTests           bool
		credentialsSet      bool
		canManipulateOrders []bool
		skip                bool
	}{
		{name: "mocked without credentials", mockTests: true},
		{name: "mocked without permission to manipulate orders", mockTests: true, canManipulateOrders: []bool{false}},
		{name: "live without credentials", skip: true},
		{name: "live without credentials or permission to manipulate orders", canManipulateOrders: []bool{false}, skip: true},
		{name: "live with permission to manipulate orders but no credentials", canManipulateOrders: []bool{true}, skip: true},
		{name: "live with credentials", credentialsSet: true},
		{name: "live with credentials but no permission to manipulate orders", credentialsSet: true, canManipulateOrders: []bool{false}, skip: true},
		{name: "live with credentials and permission to manipulate orders", credentialsSet: true, canManipulateOrders: []bool{true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			exch := new(CustomEx)
			if tc.credentialsSet {
				exch.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			}
			require.Equal(t, tc.credentialsSet, AreAPICredentialsSet(exch), "AreAPICredentialsSet must match the case's credentials")
			var skipped bool
			t.Run("probe", func(t *testing.T) {
				t.Cleanup(func() { skipped = t.Skipped() })
				SkipLiveTestIfCredentialsUnset(t, exch, tc.mockTests, tc.canManipulateOrders...)
			})
			assert.Equal(t, tc.skip, skipped, "SkipLiveTestIfCredentialsUnset should skip only a live test it cannot run")
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
