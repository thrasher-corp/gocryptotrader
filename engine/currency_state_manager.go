package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// CurrencyStateManagementName defines the manager name string
	CurrencyStateManagementName = "currency_state_manager"
	// DefaultStateManagerDelay defines the default duration when the manager
	// fetches and updates each exchange for its currency state
	DefaultStateManagerDelay = time.Minute
)

var enabled = &gctrpc.GenericResponse{Status: "enabled"}

// CurrencyStateManager manages currency states
type CurrencyStateManager struct {
	started  int32
	shutdown chan struct{}
	wg       sync.WaitGroup
	iExchangeManager
	sleep time.Duration
}

// SetupCurrencyStateManager applies configuration parameters before running
func SetupCurrencyStateManager(interval time.Duration, em iExchangeManager) (*CurrencyStateManager, error) {
	if em == nil {
		return nil, errNilExchangeManager
	}
	var c CurrencyStateManager
	if interval <= 0 {
		log.Warnf(log.ExchangeSys,
			"Currency state manager interval is invalid, defaulting to: %s",
			DefaultStateManagerDelay)
		interval = DefaultStateManagerDelay
	}
	c.sleep = interval
	c.iExchangeManager = em
	c.shutdown = make(chan struct{})
	return &c, nil
}

// Start runs the subsystem
func (c *CurrencyStateManager) Start() error {
	log.Debugln(log.ExchangeSys, "Currency state manager starting...")
	if c == nil {
		return fmt.Errorf("%s %w", CurrencyStateManagementName, ErrNilSubsystem)
	}

	if !atomic.CompareAndSwapInt32(&c.started, 0, 1) {
		return fmt.Errorf("%s %w", CurrencyStateManagementName, ErrSubSystemAlreadyStarted)
	}
	c.wg.Add(1)
	go c.monitor()
	log.Debugln(log.ExchangeSys, "Currency state manager started.")
	return nil
}

// Stop stops the subsystem
func (c *CurrencyStateManager) Stop() error {
	if c == nil {
		return fmt.Errorf("%s %w", CurrencyStateManagementName, ErrNilSubsystem)
	}
	if atomic.LoadInt32(&c.started) == 0 {
		return fmt.Errorf("%s %w", CurrencyStateManagementName, ErrSubSystemNotStarted)
	}

	log.Debugf(log.ExchangeSys, "Currency state manager %s", MsgSubSystemShuttingDown)
	close(c.shutdown)
	c.wg.Wait()
	c.shutdown = make(chan struct{})
	log.Debugf(log.ExchangeSys, "Currency state manager %s", MsgSubSystemShutdown)
	atomic.StoreInt32(&c.started, 0)
	return nil
}

// IsRunning safely checks whether the subsystem is running
func (c *CurrencyStateManager) IsRunning() bool {
	if c == nil {
		return false
	}
	return atomic.LoadInt32(&c.started) == 1
}

func (c *CurrencyStateManager) monitor() {
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
					"Currency state manager failed to get exchanges error: %v",
					err)
			}
			for x := range exchs {
				wg.Add(1)
				go c.update(exchs[x], &wg, exchs[x].GetAssetTypes(true))
			}
			wg.Wait() // This causes some variability in the timer due to the
			// longest length of request time. Can do time.Ticker but don't
			// want routines to stack behind, this is more uniform.
			timer.Reset(c.sleep)
		}
	}
}

func (c *CurrencyStateManager) update(exch exchange.IBotExchange, wg *sync.WaitGroup, enabledAssets asset.Items) {
	defer wg.Done()
	for y := range enabledAssets {
		err := exch.UpdateCurrencyStates(context.TODO(), enabledAssets[y])
		if err != nil {
			if errors.Is(err, common.ErrNotYetImplemented) {
				// Deploy default values for outbound gRPC aspects.
				var pairs currency.Pairs
				pairs, err = exch.GetAvailablePairs(enabledAssets[y])
				if err != nil {
					log.Errorf(log.ExchangeSys, "Currency state manager %s %s: %v",
						exch.GetName(),
						enabledAssets[y],
						err)
					return
				}

				// Deploys a full spectrum supported list for the currency states
				update := map[currency.Code]currencystate.Options{}
				for x := range pairs {
					update[pairs[x].Base] = currencystate.Options{}
					update[pairs[x].Quote] = currencystate.Options{}
				}

				b := exch.GetBase()
				if b == nil {
					log.Errorf(log.ExchangeSys, "Currency state manager %s %s: %v",
						exch.GetName(),
						enabledAssets[y],
						"cannot update because base is nil")
					return
				}
				err = b.States.UpdateAll(enabledAssets[y], update)
				if err != nil {
					log.Errorf(log.ExchangeSys, "Currency state manager %s %s: %v",
						exch.GetName(),
						enabledAssets[y],
						err)
				}
				return
			}
			log.Errorf(log.ExchangeSys, "Currency state manager %s %s: %v",
				exch.GetName(),
				enabledAssets[y],
				err)
		}
	}
}

// GetAllRPC returns a full snapshot of currency states, whether they are able
// to be withdrawn, deposited or traded on an exchange for RPC.
func (c *CurrencyStateManager) GetAllRPC(exchName string) (*gctrpc.CurrencyStateResponse, error) {
	if !c.IsRunning() {
		return nil, fmt.Errorf("%s %w", CurrencyStateManagementName, ErrSubSystemNotStarted)
	}
	exch, err := c.GetExchangeByName(exchName)
	if err != nil {
		return nil, err
	}

	sh, err := exch.GetCurrencyStateSnapshot()
	if err != nil {
		return nil, err
	}

	resp := &gctrpc.CurrencyStateResponse{}
	for x := range sh {
		resp.CurrencyStates = append(resp.CurrencyStates, &gctrpc.CurrencyState{
			Currency:        sh[x].Code.String(),
			Asset:           sh[x].Asset.String(),
			WithdrawEnabled: sh[x].Withdraw == nil || *sh[x].Withdraw,
			DepositEnabled:  sh[x].Deposit == nil || *sh[x].Deposit,
			TradingEnabled:  sh[x].Trade == nil || *sh[x].Trade,
		})
	}
	return resp, nil
}

// CanWithdrawRPC determines if the currency code is operational for withdrawal
// from an exchange for RPC
func (c *CurrencyStateManager) CanWithdrawRPC(exchName string, cc currency.Code, a asset.Item) (*gctrpc.GenericResponse, error) {
	if !c.IsRunning() {
		return nil, fmt.Errorf("%s %w", CurrencyStateManagementName, ErrSubSystemNotStarted)
	}

	exch, err := c.GetExchangeByName(exchName)
	if err != nil {
		return nil, err
	}

	err = exch.CanWithdraw(cc, a)
	if err != nil {
		return nil, err
	}
	return enabled, nil
}

// CanDepositRPC determines if the currency code is operational for depositing
// to an exchange for RPC
func (c *CurrencyStateManager) CanDepositRPC(exchName string, cc currency.Code, a asset.Item) (*gctrpc.GenericResponse, error) {
	if !c.IsRunning() {
		return nil, fmt.Errorf("%s %w", CurrencyStateManagementName, ErrSubSystemNotStarted)
	}

	exch, err := c.GetExchangeByName(exchName)
	if err != nil {
		return nil, err
	}

	err = exch.CanDeposit(cc, a)
	if err != nil {
		return nil, err
	}
	return enabled, nil
}

// CanTradeRPC determines if the currency code is operational for trading for
// RPC
func (c *CurrencyStateManager) CanTradeRPC(exchName string, cc currency.Code, a asset.Item) (*gctrpc.GenericResponse, error) {
	if !c.IsRunning() {
		return nil, fmt.Errorf("%s %w", CurrencyStateManagementName, ErrSubSystemNotStarted)
	}

	exch, err := c.GetExchangeByName(exchName)
	if err != nil {
		return nil, err
	}

	err = exch.CanTrade(cc, a)
	if err != nil {
		return nil, err
	}
	return enabled, nil
}

// CanTradePairRPC determines if the pair is operational for trading for RPC
func (c *CurrencyStateManager) CanTradePairRPC(exchName string, pair currency.Pair, a asset.Item) (*gctrpc.GenericResponse, error) {
	if !c.IsRunning() {
		return nil, fmt.Errorf("%s %w", CurrencyStateManagementName, ErrSubSystemNotStarted)
	}

	exch, err := c.GetExchangeByName(exchName)
	if err != nil {
		return nil, err
	}
	err = exch.CanTradePair(pair, a)
	if err != nil {
		return nil, err
	}
	return enabled, nil
}
