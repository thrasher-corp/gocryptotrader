package request

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
)

func (c *clientTracker) contains(check *http.Client) bool {
	c.Lock()
	defer c.Unlock()
	return slices.Contains(c.clients, check)
}

func TestCheckAndRegister(t *testing.T) {
	t.Parallel()
	err := tracker.checkAndRegister(nil)
	require.ErrorIs(t, err, errHTTPClientIsNil)

	newLovelyClient := new(http.Client)
	err = tracker.checkAndRegister(newLovelyClient)
	require.NoError(t, err)

	if !tracker.contains(newLovelyClient) {
		t.Fatalf("received: '%v' but expected: '%v'", false, true)
	}

	err = tracker.checkAndRegister(newLovelyClient)
	require.ErrorIs(t, err, errCannotReuseHTTPClient)
}

func TestDeRegister(t *testing.T) {
	t.Parallel()
	err := tracker.deRegister(nil)
	require.ErrorIs(t, err, errHTTPClientIsNil)

	newLovelyClient := new(http.Client)
	err = tracker.deRegister(newLovelyClient)
	require.ErrorIs(t, err, errHTTPClientNotFound)

	err = tracker.checkAndRegister(newLovelyClient)
	require.NoError(t, err)

	if !tracker.contains(newLovelyClient) {
		t.Fatalf("received: '%v' but expected: '%v'", false, true)
	}

	err = tracker.deRegister(newLovelyClient)
	require.NoError(t, err)

	if tracker.contains(newLovelyClient) {
		t.Fatalf("received: '%v' but expected: '%v'", true, false)
	}
}

func TestNewProtectedClient(t *testing.T) {
	t.Parallel()
	_, err := newProtectedClient(nil)
	require.ErrorIs(t, err, errHTTPClientIsNil)

	newLovelyClient := new(http.Client)
	protec, err := newProtectedClient(newLovelyClient)
	require.NoError(t, err)

	if protec.protected != newLovelyClient {
		t.Fatal("unexpected value")
	}
}

func TestClientSetProxy(t *testing.T) {
	t.Parallel()
	err := (&client{}).setProxy(nil)
	require.ErrorIs(t, err, errNoProxyURLSupplied)

	pp, err := url.Parse("lol.com")
	if err != nil {
		t.Fatal(err)
	}
	err = (&client{protected: new(http.Client)}).setProxy(pp)
	require.ErrorIs(t, err, errTransportNotSet)

	err = (&client{protected: common.NewHTTPClientWithTimeout(0)}).setProxy(pp)
	require.NoError(t, err)
}

func TestClientSetHTTPClientTimeout(t *testing.T) {
	t.Parallel()
	err := (&client{protected: new(http.Client)}).setHTTPClientTimeout(time.Second)
	require.ErrorIs(t, err, errTransportNotSet)

	err = (&client{protected: common.NewHTTPClientWithTimeout(0)}).setHTTPClientTimeout(time.Second)
	require.NoError(t, err)
}

func TestRelease(t *testing.T) {
	t.Parallel()
	newLovelyClient, err := newProtectedClient(common.NewHTTPClientWithTimeout(0))
	require.NoError(t, err)

	if !tracker.contains(newLovelyClient.protected) {
		t.Fatalf("received: '%v' but expected: '%v'", false, true)
	}

	err = newLovelyClient.release()
	require.NoError(t, err)

	if tracker.contains(newLovelyClient.protected) {
		t.Fatalf("received: '%v' but expected: '%v'", true, false)
	}
}
