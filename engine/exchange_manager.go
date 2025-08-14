package engine

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// vars related to exchange functions
var (
	ErrNoExchangesLoaded     = errors.New("no exchanges have been loaded")
	ErrExchangeNotFound      = errors.New("exchange not found")
	ErrExchangeAlreadyLoaded = errors.New("exchange already loaded")
	ErrExchangeFailedToLoad  = errors.New("exchange failed to load")

	errExchangeIsNil = errors.New("exchange is nil")
)

// CustomExchangeBuilder interface allows external applications to create
// custom/unsupported exchanges that satisfy the IBotExchange interface.
type CustomExchangeBuilder interface {
	NewExchangeByName(name string) (exchange.IBotExchange, error)
}

// ExchangeManager manages what exchanges are loaded
type ExchangeManager struct {
	mtx       sync.Mutex
	exchanges map[string]exchange.IBotExchange
	Builder   CustomExchangeBuilder
}

// NewExchangeManager creates a new exchange manager
func NewExchangeManager() *ExchangeManager {
	return &ExchangeManager{
		exchanges: make(map[string]exchange.IBotExchange),
	}
}

// Add adds an exchange
func (m *ExchangeManager) Add(exch exchange.IBotExchange) error {
	if m == nil {
		return fmt.Errorf("exchange manager: %w", ErrNilSubsystem)
	}
	if exch == nil {
		return fmt.Errorf("exchange manager: %w", errExchangeIsNil)
	}
	m.mtx.Lock()
	defer m.mtx.Unlock()
	_, ok := m.exchanges[strings.ToLower(exch.GetName())]
	if ok {
		return fmt.Errorf("exchange manager: %s %w", exch.GetName(), ErrExchangeAlreadyLoaded)
	}
	m.exchanges[strings.ToLower(exch.GetName())] = exch
	return nil
}

// GetExchanges returns all stored exchanges
func (m *ExchangeManager) GetExchanges() ([]exchange.IBotExchange, error) {
	if m == nil {
		return nil, fmt.Errorf("exchange manager: %w", ErrNilSubsystem)
	}
	m.mtx.Lock()
	defer m.mtx.Unlock()
	exchs := make([]exchange.IBotExchange, 0, len(m.exchanges))
	for _, exch := range m.exchanges {
		exchs = append(exchs, exch)
	}
	return exchs, nil
}

// RemoveExchange removes an exchange from the manager
func (m *ExchangeManager) RemoveExchange(exchangeName string) error {
	if m == nil {
		return fmt.Errorf("exchange manager: %w", ErrNilSubsystem)
	}

	if exchangeName == "" {
		return fmt.Errorf("exchange manager: %w", common.ErrExchangeNameNotSet)
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()
	exch, ok := m.exchanges[strings.ToLower(exchangeName)]
	if !ok {
		return fmt.Errorf("exchange manager: %s %w", exchangeName, ErrExchangeNotFound)
	}
	err := exch.Shutdown()
	if err != nil {
		return fmt.Errorf("exchange manager: %w", err)
	}
	delete(m.exchanges, strings.ToLower(exchangeName))
	log.Infof(log.ExchangeSys, "%s exchange unloaded successfully.\n", exchangeName)
	return nil
}

// GetExchangeByName returns an exchange by its name if it exists
func (m *ExchangeManager) GetExchangeByName(exchangeName string) (exchange.IBotExchange, error) {
	if m == nil {
		return nil, fmt.Errorf("exchange manager: %w", ErrNilSubsystem)
	}
	if exchangeName == "" {
		return nil, fmt.Errorf("exchange manager: %w", common.ErrExchangeNameNotSet)
	}
	m.mtx.Lock()
	defer m.mtx.Unlock()
	exch, ok := m.exchanges[strings.ToLower(exchangeName)]
	if !ok {
		return nil, fmt.Errorf("exchange manager: %s %w", exchangeName, ErrExchangeNotFound)
	}
	return exch, nil
}

// NewExchangeByName helps create a new exchange to be loaded
func (m *ExchangeManager) NewExchangeByName(name string) (exchange.IBotExchange, error) {
	nameLower := strings.ToLower(name)
	_, err := m.GetExchangeByName(nameLower)
	if err != nil {
		if !errors.Is(err, ErrExchangeNotFound) {
			return nil, fmt.Errorf("exchange manager: %s %w", name, err)
		}
	} else {
		return nil, fmt.Errorf("exchange manager: %s %w", name, ErrExchangeAlreadyLoaded)
	}

	if m.Builder != nil {
		return m.Builder.NewExchangeByName(nameLower)
	}

	return NewSupportedExchangeByName(nameLower)
}

// Shutdown shuts down all exchanges and unloads them
func (m *ExchangeManager) Shutdown(shutdownTimeout time.Duration) error {
	if m == nil {
		return fmt.Errorf("exchange manager: %w", ErrNilSubsystem)
	}

	if shutdownTimeout < 0 {
		shutdownTimeout = 0
	}

	var lockout sync.Mutex
	timer := time.NewTimer(shutdownTimeout)
	var wg sync.WaitGroup

	m.mtx.Lock()
	defer m.mtx.Unlock()

	lockout.Lock()
	for _, exch := range m.exchanges {
		wg.Add(1)
		go func(wg *sync.WaitGroup, mtx *sync.Mutex, exch exchange.IBotExchange) {
			err := exch.Shutdown()
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s failed to shutdown %v.\n", exch.GetName(), err)
			} else {
				mtx.Lock()
				delete(m.exchanges, strings.ToLower(exch.GetName()))
				mtx.Unlock()
			}
			wg.Done()
		}(&wg, &lockout, exch)
	}
	lockout.Unlock()

	ch := make(chan struct{})
	go func(wg *sync.WaitGroup, finish chan<- struct{}) {
		wg.Wait()
		finish <- struct{}{}
	}(&wg, ch)

	select {
	case <-timer.C:
		// Possible deadlock in a number of operating exchanges.
		lockout.Lock()
		for name := range m.exchanges {
			log.Warnf(log.ExchangeSys, "%s has failed to shutdown within %s, please review.\n", name, shutdownTimeout)
		}
		lockout.Unlock()
	case <-ch:
		// Every exchange has finished their shutdown call.
		lockout.Lock()
		for name := range m.exchanges {
			log.Errorf(log.ExchangeSys, "%s has failed to shutdown due to error, please review.\n", name)
		}
		lockout.Unlock()
	}
	return nil
}
