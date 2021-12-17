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
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// FeeManagerName defines the manager name string
	FeeManagerName = "fee_manager"
	// DefaultFeeManagerDelay defines the default duration when the manager
	// fetches and updates each exchange for its fees
	DefaultFeeManagerDelay = time.Minute * 10
)

var errNilManager = errors.New("manager has not been set")

// FeeManager manages full fee structures across all enabled exchanges
type FeeManager struct {
	started         int32
	shutdown        chan struct{}
	wg              sync.WaitGroup
	exchangeManager iExchangeManager
	sleep           time.Duration
}

// SetupFeeManager applies configuration parameters before running
func SetupFeeManager(interval time.Duration, em iExchangeManager) (*FeeManager, error) {
	if em == nil {
		return nil, errNilExchangeManager
	}
	var f FeeManager
	if interval <= 0 {
		log.Warnf(log.ExchangeSys,
			"Fee manager interval is invalid, defaulting to: %s",
			DefaultFeeManagerDelay)
		interval = DefaultFeeManagerDelay
	}
	f.sleep = interval
	f.exchangeManager = em

	return &f, nil
}

// Start runs the subsystem
func (f *FeeManager) Start() error {
	log.Debugln(log.ExchangeSys, "Fee manager starting...")
	if f == nil {
		return fmt.Errorf("%s %w", FeeManagerName, ErrNilSubsystem)
	}

	if f.exchangeManager == nil {
		return errNilManager
	}

	if !atomic.CompareAndSwapInt32(&f.started, 0, 1) {
		return fmt.Errorf("%s %w", FeeManagerName, ErrSubSystemAlreadyStarted)
	}
	f.wg.Add(1)
	f.shutdown = make(chan struct{})
	go f.monitor()
	log.Debugln(log.ExchangeSys, "Fee manager started.")
	return nil
}

// Stop stops the subsystem
func (f *FeeManager) Stop() error {
	if f == nil {
		return fmt.Errorf("%s %w", FeeManagerName, ErrNilSubsystem)
	}
	if atomic.LoadInt32(&f.started) == 0 {
		return fmt.Errorf("%s %w", FeeManagerName, ErrSubSystemNotStarted)
	}

	log.Debugf(log.ExchangeSys, "Fee manager %s", MsgSubSystemShuttingDown)
	close(f.shutdown)
	f.wg.Wait()
	f.shutdown = make(chan struct{})
	log.Debugf(log.ExchangeSys, "Fee manager %s", MsgSubSystemShutdown)
	atomic.CompareAndSwapInt32(&f.started, 1, 0)
	return nil
}

// IsRunning safely checks whether the subsystem is running
func (f *FeeManager) IsRunning() bool {
	if f == nil {
		return false
	}
	return atomic.LoadInt32(&f.started) == 1
}

func (f *FeeManager) monitor() {
	defer f.wg.Done()
	timer := time.NewTimer(0) // Prime fireing of channel for initial sync.
	var wg sync.WaitGroup
	for {
		select {
		case <-f.shutdown:
			return
		case <-timer.C:
			exchs, err := f.exchangeManager.GetExchanges()
			if err != nil {
				log.Errorf(log.Global,
					"Fee manager failed to get exchanges error: %v",
					err)
			}
			wg.Add(len(exchs))
			for x := range exchs {
				go func(exch exchange.IBotExchange, wg *sync.WaitGroup) {
					err := update(exch, exch.GetAssetTypes(true))
					if err != nil {
						log.Errorf(log.Global, "Fee manager: %s %v", exch.GetName(), err)
					}
					wg.Done()
				}(exchs[x], &wg)
			}
			// This causes some variability in the timer due to longest length
			// of request time. Can do time.Ticker but don't want routines to
			// stack behind, this is more uniform.
			wg.Wait()
			timer.Reset(f.sleep)
		}
	}
}

func update(exch exchange.IBotExchange, enabledAssets asset.Items) error {
	if (exch.IsRESTAuthenticationRequiredForTradeFees() && exch.IsAuthenticatedRESTSupported()) ||
		!exch.IsRESTAuthenticationRequiredForTradeFees() {
		// Commission fees are maker and taker fees associated with different asset
		// types
		for x := range enabledAssets {
			err := exch.UpdateCommissionFees(context.TODO(), enabledAssets[x])
			if err != nil && !errors.Is(err, common.ErrNotYetImplemented) {
				return fmt.Errorf("update commission fees for %s: %w",
					enabledAssets[x],
					err)
			}
		}
	}

	if exch.IsRESTAuthenticationRequiredForTransferFees() && !exch.IsAuthenticatedRESTSupported() {
		return nil
	}

	// Transfer fees are the common exchange interaction withdrawal and deposit
	// fees
	err := exch.UpdateTransferFees(context.TODO())
	if err != nil && !errors.Is(err, common.ErrNotYetImplemented) {
		return fmt.Errorf("update chain transfer fees: %w", err)
	}

	// Bank fees are the common exchange banking interaction withdrawal and
	// deposit fees
	err = exch.UpdateBankTransferFees(context.TODO())
	if err != nil && !errors.Is(err, common.ErrNotYetImplemented) {
		return fmt.Errorf("update bank transfer fees: %w", err)
	}
	return nil
}
