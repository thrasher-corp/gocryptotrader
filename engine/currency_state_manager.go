package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	CurrencyStateManagementName = "CurrencyStateManagement"
	DefaultStateManagerDelay    = time.Minute
)

var (
	errNilCurrencyStateManager = errors.New("currency state manager is nil")
)

// currencyStateManager manages currency states
type currencyStateManager struct {
	started  int32
	shutdown chan struct{}
	wg       sync.WaitGroup
	*currencystate.Manager
	iExchangeManager
	sleep time.Duration
}

// Setup applies configuration parameters before running
func SetupCurrencyStateManager(interval time.Duration, em iExchangeManager) (*currencyStateManager, error) {
	var c currencyStateManager
	if interval <= 0 {
		log.Warnf(log.ExchangeSys,
			"%s interval is invalid, defaulting to: %s",
			CurrencyStateManagementName,
			DefaultStateManagerDelay)
		c.sleep = DefaultStateManagerDelay
	} else {
		c.sleep = interval
	}

	c.Manager = currencystate.GetManager()
	if c.Manager == nil {
		return nil, errNilCurrencyStateManager
	}
	c.iExchangeManager = em
	c.shutdown = make(chan struct{})
	return &c, nil
}

// Start runs the subsystem
func (c *currencyStateManager) Start() error {
	if c == nil {
		return fmt.Errorf("%s %w", CurrencyStateManagementName, ErrNilSubsystem)
	}

	if c.Manager == nil {
		return fmt.Errorf("%s %w", CurrencyStateManagementName, ErrNilSubsystem)
	}

	if !atomic.CompareAndSwapInt32(&c.started, 0, 1) {
		return fmt.Errorf("%s %w", CurrencyStateManagementName, ErrSubSystemAlreadyStarted)
	}
	c.wg.Add(1)
	go c.monitor()
	return nil
}

// Stop stops the subsystem
func (c *currencyStateManager) Stop() error {
	if c == nil {
		return fmt.Errorf("%s %w", CurrencyStateManagementName, ErrNilSubsystem)
	}
	if atomic.LoadInt32(&c.started) == 0 {
		return fmt.Errorf("%s %w", CurrencyStateManagementName, ErrSubSystemNotStarted)
	}

	log.Debugf(log.ExchangeSys, "%s %s", CurrencyStateManagementName, MsgSubSystemShuttingDown)
	close(c.shutdown)
	c.wg.Wait()
	c.shutdown = make(chan struct{})
	log.Debugf(log.ExchangeSys, "%s %s", CurrencyStateManagementName, MsgSubSystemShutdown)
	atomic.StoreInt32(&c.started, 0)
	return nil
}

// IsRunning safely checks whether the subsystem is running
func (c *currencyStateManager) IsRunning() bool {
	if c == nil {
		return false
	}
	return atomic.LoadInt32(&c.started) == 1
}

func (c *currencyStateManager) monitor() {
	defer c.wg.Done()
	timer := time.NewTimer(0) // Prime firing of channel for initial sync.
	for {
		select {
		case <-c.shutdown:
			return

		case <-timer.C:
			var wg sync.WaitGroup
			exchs, err := c.GetExchanges()
			if err != nil {
				log.Errorf(log.Global,
					"%s failed to get exchanges error: %v",
					CurrencyStateManagementName,
					err)
			}
			for x := range exchs {
				wg.Add(1)
				go c.update(exchs[x], &wg, exchs[x].GetAssetTypes(true))
			}
			wg.Wait() // This causes some variability in the timer due to
			// longest length of request time. Can do time.Ticker but don't
			// want routines to stack behind, this is more uniform.
			timer.Reset(c.sleep)
		}
	}
}

func (c *currencyStateManager) update(exch exchange.IBotExchange, wg *sync.WaitGroup, enabledAssets asset.Items) {
	defer wg.Done()
	for y := range enabledAssets {
		err := exch.UpdateCurrencyStates(context.TODO(), enabledAssets[y])
		if err != nil && !errors.Is(err, common.ErrNotYetImplemented) {
			log.Errorf(log.ExchangeSys, "%s %s %s: %v",
				CurrencyStateManagementName,
				exch.GetName(),
				enabledAssets[y],
				err)
		}
	}
}
