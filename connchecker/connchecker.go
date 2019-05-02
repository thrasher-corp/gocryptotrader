package connchecker

import (
	"net"
	"sync"
	"time"

	log "github.com/thrasher-/gocryptotrader/logger"
)

// DefaultCheckInterval is a const that defines the amount of time between
// checking if the connection is lost
const DefaultCheckInterval = time.Second

// Default check lists
var (
	DefaultDNSList    = []string{"8.8.8.8", "8.8.4.4", "1.1.1.1", "1.0.0.1"}
	DefaultDomainList = []string{"www.google.com", "www.cloudflare.com", "www.facebook.com"}
)

// New returns a new connection checker, if no values set it will default it out
func New(dnsList, domainList []string, checkInterval time.Duration) *Checker {
	c := &Checker{}
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

	go c.Monitor()
	return c
}

// Checker defines a struct to determine connectivity to the interwebs
type Checker struct {
	DNSList       []string
	DomainList    []string
	CheckInterval time.Duration
	shutdown      chan struct{}
	wg            sync.WaitGroup
	connected     bool
	sync.Mutex
}

// Shutdown cleanly shutsdown monitor routine
func (c *Checker) Shutdown() {
	c.shutdown <- struct{}{}
	c.wg.Wait()
}

// Monitor determines internet connectivity via a DNS lookup
func (c *Checker) Monitor() {
	c.wg.Add(1)
	tick := time.NewTicker(time.Second)
	defer func() { tick.Stop(); c.wg.Done() }()
	c.connectionTest()
	for {
		select {
		case <-tick.C:
			c.connectionTest()
		case <-c.shutdown:
			return
		}
	}
}

// ConnectionTest determines if a connection to the internet is available by
// iterating over a set list of dns ip and popular domains
func (c *Checker) connectionTest() {
	for i := range c.DNSList {
		_, err := net.LookupAddr(c.DNSList[i])
		if err == nil {
			c.Lock()
			if !c.connected {
				log.Warnf("Internet connectivity re-established")
				c.connected = true
			}
			c.Unlock()
			return
		}
	}

	for i := range c.DomainList {
		_, err := net.LookupHost(c.DomainList[i])
		if err == nil {
			c.Lock()
			if !c.connected {
				log.Warnf("Internet connectivity re-established")
				c.connected = true
			}
			c.Unlock()
			return
		}
	}

	c.Lock()
	if c.connected {
		log.Warnf("Internet connectivity lost")
		c.connected = false
	}
	c.Unlock()
}

// IsConnected returns if there is internet connectivity
func (c *Checker) IsConnected() bool {
	c.Lock()
	defer c.Unlock()
	return c.connected
}
