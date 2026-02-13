package connchecker

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/log"
)

// DefaultCheckInterval is a const that defines the amount of time between
// checking if the connection is lost
const (
	DefaultCheckInterval = time.Second

	ConnRe       = "Internet connectivity re-established"
	ConnLost     = "Internet connectivity lost"
	ConnFound    = "Internet connectivity found"
	ConnNotFound = "No internet connectivity"
)

// Default check lists
var (
	DefaultDNSList    = []string{"8.8.8.8", "8.8.4.4", "1.1.1.1", "1.0.0.1"}
	DefaultDomainList = []string{"www.google.com", "www.cloudflare.com", "www.facebook.com"}
)

// New returns a new connection checker, if no values set it will default it out
func New(dnsList, domainList []string, checkInterval time.Duration) (*Checker, error) {
	c := new(Checker)
	if len(dnsList) == 0 {
		c.DNSList = DefaultDNSList
	} else {
		c.DNSList = dnsList
	}

	if len(domainList) == 0 {
		c.DomainList = DefaultDomainList
	} else {
		c.DomainList = domainList
	}

	if checkInterval == 0 {
		c.CheckInterval = DefaultCheckInterval
	} else {
		c.CheckInterval = checkInterval
	}

	if err := c.initialCheck(); err != nil {
		return nil, err
	}

	if c.connected {
		log.Debugln(log.Global, ConnFound)
	} else {
		log.Warnln(log.Global, ConnNotFound)
	}

	c.shutdown = make(chan struct{}, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go c.Monitor(&wg)
	wg.Wait()
	return c, nil
}

// Checker defines a struct to determine connectivity to the interwebs
type Checker struct {
	DNSList       []string
	DomainList    []string
	CheckInterval time.Duration
	shutdown      chan struct{}
	wg            sync.WaitGroup
	connected     bool
	mu            sync.Mutex
}

// Shutdown cleanly shutsdown monitor routine
func (c *Checker) Shutdown() {
	c.connected = false
	close(c.shutdown)
	c.wg.Wait()
}

// Monitor determines internet connectivity via a DNS lookup
func (c *Checker) Monitor(wg *sync.WaitGroup) {
	c.wg.Add(1)
	tick := time.NewTicker(c.CheckInterval)
	defer func() { tick.Stop(); c.wg.Done() }()
	wg.Done()
	for {
		select {
		case <-tick.C:
			go c.connectionTest()
		case <-c.shutdown:
			return
		}
	}
}

// initialCheck starts an initial connection check
func (c *Checker) initialCheck() error {
	var connected bool
	for i := range c.DNSList {
		err := c.CheckDNS(c.DNSList[i])
		if err != nil {
			if strings.Contains(err.Error(), "unrecognized address") ||
				strings.Contains(err.Error(), "invalid address") {
				return err
			}
			continue
		}
		if !connected {
			connected = true
		}
	}

	for i := range c.DomainList {
		err := c.CheckHost(c.DomainList[i])
		if err != nil {
			continue
		}
		if !connected {
			connected = true
		}
	}
	c.connected = connected
	return nil
}

// connectionTest determines if a connection to the internet is available by
// iterating over a set list of dns ip and popular domains
func (c *Checker) connectionTest() {
	for i := range c.DNSList {
		err := c.CheckDNS(c.DNSList[i])
		if err == nil {
			c.mu.Lock()
			if !c.connected {
				log.Debugln(log.Global, ConnRe)
				c.connected = true
			}
			c.mu.Unlock()
			return
		}
	}

	for i := range c.DomainList {
		err := c.CheckHost(c.DomainList[i])
		if err == nil {
			c.mu.Lock()
			if !c.connected {
				log.Debugln(log.Global, ConnRe)
				c.connected = true
			}
			c.mu.Unlock()
			return
		}
	}

	c.mu.Lock()
	if c.connected {
		log.Warnln(log.Global, ConnLost)
		c.connected = false
	}
	c.mu.Unlock()
}

// CheckDNS checks current dns for connectivity
func (c *Checker) CheckDNS(dns string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.CheckInterval)
	defer cancel()
	_, err := net.DefaultResolver.LookupAddr(ctx, dns)
	return err
}

// CheckHost checks current host name for connectivity
func (c *Checker) CheckHost(host string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.CheckInterval)
	defer cancel()
	_, err := net.DefaultResolver.LookupHost(ctx, host)
	return err
}

// IsConnected returns if there is internet connectivity
func (c *Checker) IsConnected() bool {
	c.mu.Lock()
	isConnected := c.connected
	c.mu.Unlock()
	return isConnected
}
