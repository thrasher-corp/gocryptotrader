package request

import (
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// this doesn't need to be included in binary
func (c *clientTracker) contains(check *http.Client) bool {
	c.Lock()
	defer c.Unlock()
	for x := range c.clients {
		if check == c.clients[x] {
			return true
		}
	}
	return false
}

func TestCheckAndRegister(t *testing.T) {
	t.Parallel()
	err := tracker.checkAndRegister(nil)
	if !errors.Is(err, errHTTPClientIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errHTTPClientIsNil)
	}

	newLovelyClient := new(http.Client)
	err = tracker.checkAndRegister(newLovelyClient)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !tracker.contains(newLovelyClient) {
		t.Fatalf("received: '%v' but expected: '%v'", false, true)
	}

	err = tracker.checkAndRegister(newLovelyClient)
	if !errors.Is(err, errCannotReuseHTTPClient) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errCannotReuseHTTPClient)
	}
}

func TestDeRegister(t *testing.T) {
	t.Parallel()
	err := tracker.deRegister(nil)
	if !errors.Is(err, errHTTPClientIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errHTTPClientIsNil)
	}

	newLovelyClient := new(http.Client)
	err = tracker.deRegister(newLovelyClient)
	if !errors.Is(err, errHTTPClientNotFound) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errHTTPClientNotFound)
	}

	err = tracker.checkAndRegister(newLovelyClient)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !tracker.contains(newLovelyClient) {
		t.Fatalf("received: '%v' but expected: '%v'", false, true)
	}

	err = tracker.deRegister(newLovelyClient)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if tracker.contains(newLovelyClient) {
		t.Fatalf("received: '%v' but expected: '%v'", true, false)
	}
}

func TestNewProtectedClient(t *testing.T) {
	t.Parallel()
	if _, err := newProtectedClient(nil); !errors.Is(err, errHTTPClientIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errHTTPClientIsNil)
	}

	newLovelyClient := new(http.Client)
	protec, err := newProtectedClient(newLovelyClient)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if protec.protected != newLovelyClient {
		t.Fatal("unexpected value")
	}
}

func TestClientSetProxy(t *testing.T) {
	t.Parallel()
	err := (&client{}).setProxy(nil)
	if !errors.Is(err, errNoProxyURLSupplied) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoProxyURLSupplied)
	}
	pp, err := url.Parse("lol.com")
	if err != nil {
		t.Fatal(err)
	}
	err = (&client{protected: new(http.Client)}).setProxy(pp)
	if !errors.Is(err, errTransportNotSet) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errTransportNotSet)
	}
	err = (&client{protected: common.NewHTTPClientWithTimeout(0)}).setProxy(pp)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestClientSetHTTPClientTimeout(t *testing.T) {
	t.Parallel()
	err := (&client{protected: new(http.Client)}).setHTTPClientTimeout(time.Second)
	if !errors.Is(err, errTransportNotSet) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errTransportNotSet)
	}
	err = (&client{protected: common.NewHTTPClientWithTimeout(0)}).setHTTPClientTimeout(time.Second)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestRelease(t *testing.T) {
	t.Parallel()
	newLovelyClient, err := newProtectedClient(common.NewHTTPClientWithTimeout(0))
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !tracker.contains(newLovelyClient.protected) {
		t.Fatalf("received: '%v' but expected: '%v'", false, true)
	}

	err = newLovelyClient.release()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if tracker.contains(newLovelyClient.protected) {
		t.Fatalf("received: '%v' but expected: '%v'", true, false)
	}
}
