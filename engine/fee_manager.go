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
	FeeManagerName         = "FeeManager"
	DefaultFeeManagerDelay = time.Minute
)

// FeeManager manages full fee structures across all enabled exchanges
type FeeManager struct {
	started  int32
	shutdown chan struct{}
	wg       sync.WaitGroup
	iExchangeManager
	sleep time.Duration
}

// SetupFeeManager applies configuration parameters before running
func SetupFeeManager(interval time.Duration, em iExchangeManager) (*FeeManager, error) {
	if em == nil {
		return nil, errNilExchangeManager
	}
	var f FeeManager
	if interval <= 0 {
		log.Warnf(log.ExchangeSys,
			"%s interval is invalid, defaulting to: %s",
			FeeManagerName,
			DefaultFeeManagerDelay)
		interval = DefaultFeeManagerDelay
	}
	f.sleep = interval
	f.iExchangeManager = em
	f.shutdown = make(chan struct{})
	return &f, nil
}

// Start runs the subsystem
func (f *FeeManager) Start() error {
	if f == nil {
		return fmt.Errorf("%s %w", FeeManagerName, ErrNilSubsystem)
	}

	if !atomic.CompareAndSwapInt32(&f.started, 0, 1) {
		return fmt.Errorf("%s %w", FeeManagerName, ErrSubSystemAlreadyStarted)
	}
	f.wg.Add(1)
	go f.monitor()
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

	log.Debugf(log.ExchangeSys, "%s %s", FeeManagerName, MsgSubSystemShuttingDown)
	close(f.shutdown)
	f.wg.Wait()
	f.shutdown = make(chan struct{})
	log.Debugf(log.ExchangeSys, "%s %s", FeeManagerName, MsgSubSystemShutdown)
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
	for {
		select {
		case <-f.shutdown:
			return

		case <-timer.C:
			var wg sync.WaitGroup
			exchs, err := f.GetExchanges()
			if err != nil {
				log.Errorf(log.Global,
					"%s failed to get exchanges error: %v",
					FeeManagerName,
					err)
			}
			for x := range exchs {
				if !exchs[x].GetAuthenticatedAPISupport(exchange.RestAuthentication) {
					continue
				}
				wg.Add(1)
				go update(exchs[x], &wg, exchs[x].GetAssetTypes(true))
			}
			wg.Wait() // This causes some variability in the timer due to
			// longest length of request time. Can do time.Ticker but don't
			// want routines to stack behind, this is more uniform.
			timer.Reset(f.sleep)
		}
	}
}

func update(exch exchange.IBotExchange, wg *sync.WaitGroup, enabledAssets asset.Items) {
	defer wg.Done()

	// Commission fees are maker and taker fees associated with different asset
	// types
	for y := range enabledAssets {
		err := exch.UpdateCommissionFees(context.TODO(), enabledAssets[y])
		if err != nil && !errors.Is(err, common.ErrNotYetImplemented) {
			log.Errorf(log.ExchangeSys, "%s %s %s: %v",
				FeeManagerName,
				exch.GetName(),
				enabledAssets[y],
				err)
		}
	}

	// Transfer fees are the common exchange interaction withdrawal and deposit
	// fees
	err := exch.UpdateTransferFees(context.TODO())
	if err != nil && !errors.Is(err, common.ErrNotYetImplemented) {
		log.Errorf(log.ExchangeSys, "%s %s: %v",
			FeeManagerName,
			exch.GetName(),
			err)
	}

	// Bank fees are the common exchange banking interaction withdrawal and
	// deposit fees
	err = exch.UpdateBankTransferFees(context.TODO())
	if err != nil && !errors.Is(err, common.ErrNotYetImplemented) {
		log.Errorf(log.ExchangeSys, "%s %s: %v",
			FeeManagerName,
			exch.GetName(),
			err)
	}

}
