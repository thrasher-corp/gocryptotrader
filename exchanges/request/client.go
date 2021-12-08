package request

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var (
	trackthis                ClientTracker
	errCannotSetProxy        = errors.New("cannot set proxy")
	errNoProxyURLSupplied    = errors.New("no proxy URL supplied")
	errCannotReuseHTTPClient = errors.New("cannot reuse http client")
	errHTTPClientIsNil       = errors.New("http client is nil")
)

type ClientTracker struct {
	clients []*http.Client
	sync.Mutex
}

// checkAndRegister stops the sharing of the same http.Client between services.
func (c *ClientTracker) checkAndRegister(newClient *http.Client) error {
	if newClient == nil {
		return errHTTPClientIsNil
	}
	c.Lock()
	defer c.Unlock()
	for x := range c.clients {
		if newClient == c.clients[x] {
			return errCannotReuseHTTPClient
		}
	}
	c.clients = append(c.clients, newClient)
	return nil
}

// client wraps over a http client for better protection
type client struct {
	protected *http.Client
	m         sync.RWMutex
}

// NewProtectedClient registers a http.Client to inhibit cross service usage and
// return a thread safe holder (*request.Client) with getter and setters for
// timeouts and transports.
func newProtectedClient(newClient *http.Client) (*client, error) {

	if err := trackthis.checkAndRegister(newClient); err != nil {
		return nil, err
	}
	return &client{protected: newClient}, nil
}

// setProxy sets a proxy address for the client transport
func (c *client) setProxy(p *url.URL) error {
	if p.String() == "" {
		return errNoProxyURLSupplied
	}
	c.m.Lock()
	defer c.m.Unlock()
	t, ok := c.protected.Transport.(*http.Transport)
	if !ok {
		return fmt.Errorf("transport not set: %w", errCannotSetProxy)
	}
	t.Proxy = http.ProxyURL(p)
	t.TLSHandshakeTimeout = proxyTLSTimeout
	return nil
}

// setClientTimeout sets the timeout value for the exchanges HTTP Client and
// also the underlying transports idle connection timeout
func (c *client) setHTTPClientTimeout(timeout time.Duration) error {
	c.m.Lock()
	defer c.m.Unlock()
	// Check transport first so we don't set something and then error.
	tr, ok := c.protected.Transport.(*http.Transport)
	if !ok {
		return errTransportNotSet
	}
	tr.IdleConnTimeout = timeout
	c.protected.Timeout = timeout
	return nil
}

// do sends request in a protected manner
func (c *client) do(request *http.Request) (resp *http.Response, err error) {
	c.m.RLock()
	resp, err = c.protected.Do(request)
	c.m.RUnlock()
	return
}
