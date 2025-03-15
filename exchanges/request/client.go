package request

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"
)

var (
	// tracker is the global to maintain sanity between clients across all
	// services using the request package.
	tracker clientTracker

	errNoProxyURLSupplied    = errors.New("no proxy URL supplied")
	errCannotReuseHTTPClient = errors.New("cannot reuse http client")
	errHTTPClientIsNil       = errors.New("http client is nil")
	errHTTPClientNotFound    = errors.New("http client not found")
)

// clientTracker attempts to maintain service/http.Client segregation
type clientTracker struct {
	clients []*http.Client
	sync.Mutex
}

// checkAndRegister stops the sharing of the same http.Client between services.
func (c *clientTracker) checkAndRegister(newClient *http.Client) error {
	if newClient == nil {
		return errHTTPClientIsNil
	}
	c.Lock()
	defer c.Unlock()

	if slices.Contains(c.clients, newClient) {
		return errCannotReuseHTTPClient
	}

	c.clients = append(c.clients, newClient)
	return nil
}

// deRegister removes the *http.Client from being tracked
func (c *clientTracker) deRegister(oldClient *http.Client) error {
	if oldClient == nil {
		return errHTTPClientIsNil
	}
	c.Lock()
	defer c.Unlock()
	for x := range c.clients {
		if oldClient != c.clients[x] {
			continue
		}
		c.clients[x] = c.clients[len(c.clients)-1]
		c.clients[len(c.clients)-1] = nil
		c.clients = c.clients[:len(c.clients)-1]
		return nil
	}
	return errHTTPClientNotFound
}

// client wraps over a http client for better protection
type client struct {
	protected *http.Client
	m         sync.RWMutex
}

// newProtectedClient registers a http.Client to inhibit cross service usage and
// return a thread safe holder (*request.Client) with getter and setters for
// timeouts and transports.
func newProtectedClient(newClient *http.Client) (*client, error) {
	if err := tracker.checkAndRegister(newClient); err != nil {
		return nil, err
	}
	return &client{protected: newClient}, nil
}

// setProxy sets a proxy address for the client transport
func (c *client) setProxy(p *url.URL) error {
	if p == nil || p.String() == "" {
		return errNoProxyURLSupplied
	}
	c.m.Lock()
	defer c.m.Unlock()
	// Check transport first so we don't set something and then error.
	tr, ok := c.protected.Transport.(*http.Transport)
	if !ok {
		return errTransportNotSet
	}
	// This closes idle connections before an attempt at reassignment and
	// boots any dangly routines.
	tr.CloseIdleConnections()
	tr.Proxy = http.ProxyURL(p)
	tr.TLSHandshakeTimeout = proxyTLSTimeout
	return nil
}

// setHTTPClientTimeout sets the timeout value for the exchanges HTTP Client and
// also the underlying transports idle connection timeout
func (c *client) setHTTPClientTimeout(timeout time.Duration) error {
	c.m.Lock()
	defer c.m.Unlock()
	// Check transport first so we don't set something and then error.
	tr, ok := c.protected.Transport.(*http.Transport)
	if !ok {
		return errTransportNotSet
	}
	// This closes idle connections before an attempt at reassignment and
	// boots any dangly routines.
	tr.CloseIdleConnections()
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

// release de-registers the underlying client
func (c *client) release() error {
	c.m.Lock()
	err := tracker.deRegister(c.protected)
	c.m.Unlock()
	return err
}
