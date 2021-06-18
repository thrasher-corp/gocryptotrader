package engine

import (
	"errors"
	"sync"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// AccountManager defines account management
type AccountManager struct {
	engine                  *Engine
	synchronizationInterval time.Duration
	shutdown                chan struct{}
	wg                      sync.WaitGroup
	m                       sync.Mutex
	verbose                 bool
}

var (
	errEngineIsNil                  = errors.New("engine is nil")
	errAccountManagerNotStarted     = errors.New("account manager not started")
	errAccountManagerAlreadyStarted = errors.New("account manager already started")
	errUnrealisticUpdateInterval    = errors.New("unrealistic update interval should be equal or greater than 10 seconds")
)

// NewAccountManager returns a pointer of a new instance of an account manager
func NewAccountManager(e *Engine, verbose bool) (*AccountManager, error) {
	if e == nil {
		return nil, errEngineIsNil
	}
	return &AccountManager{
		engine:  e,
		verbose: verbose,
	}, nil
}

// Shutdown shuts down the account management instance
func (a *AccountManager) Shutdown() error {
	log.Debugln(log.Accounts, "Account Manager shutting down...")
	a.m.Lock()
	defer a.m.Unlock()
	if a.shutdown == nil {
		return errAccountManagerNotStarted
	}
	close(a.shutdown)
	a.wg.Wait()
	log.Debugln(log.Accounts, "Account Manager stopped.")
	return nil
}

// RunUpdater takes in a sync duration and spawns an update routine.
func (a *AccountManager) RunUpdater(interval time.Duration) error {
	if interval < time.Second*10 {
		return errUnrealisticUpdateInterval
	}
	a.m.Lock()
	defer a.m.Unlock()
	if a.shutdown != nil {
		return errAccountManagerAlreadyStarted
	}
	if a.verbose {
		log.Debugln(log.Accounts, "Account Manager started...")
	}
	a.synchronizationInterval = interval
	a.shutdown = make(chan struct{})
	a.wg.Add(1)
	go a.accountUpdater()
	return nil
}

func (a *AccountManager) accountUpdater() {
	tt := time.NewTicker(a.synchronizationInterval)
	defer a.wg.Done()
	for {
		select {
		case <-tt.C:
			exchs := a.engine.GetExchanges()
			for x := range exchs {
				if a.verbose {
					log.Debugf(log.Accounts,
						"Updating accounts for %s",
						exchs[x].GetName())
				}
				go a.updateAccountForExchange(exchs[x])
			}
		case <-a.shutdown:
			return
		}
	}
}

func (a *AccountManager) updateAccountForExchange(exch exchange.IBotExchange) {
	base := exch.GetBase()
	if base == nil || base.Config == nil || !base.Config.API.AuthenticatedSupport {
		return
	}

	accounts, err := exch.GetAccounts()
	if err != nil {
		log.Errorf(log.Accounts,
			"%s failed to update account holdings for account: %v",
			exch.GetName(),
			err)
		return
	}
	at := exch.GetAssetTypes(true)
	for x := range accounts {
		for y := range at {
			_, err := exch.UpdateAccountInfo(string(accounts[x]), at[y])
			if err != nil {
				log.Errorf(log.Accounts,
					"%s failed to update account holdings for account: %v",
					exch.GetName(),
					err)
			}
		}
	}
}

// IsRunning checks to see if the manager is running
func (a *AccountManager) IsRunning() bool {
	a.m.Lock()
	defer a.m.Unlock()
	return a.shutdown != nil
}
