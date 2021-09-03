package engine

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fee"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	FeeManagement          = "FeeManagement"
	DefaultFeeManagerDelay = time.Minute
)

var (
	errSubsystemNotSetup = errors.New("subsystem not set up")
	errNilInterface      = errors.New("interface is nil")
)

type feeManager struct {
	started  int32
	shutdown chan struct{}
	wg       sync.WaitGroup
	*fee.Manager
	iExchangeManager
	sleep time.Duration
}

// Setup applies configuration parameters before running
func (f *feeManager) Setup(interval time.Duration, em iExchangeManager) error {
	if em == nil {
		return errNilInterface
	}
	f.sleep = interval
	if interval <= 0 {
		log.Warnf(log.ExchangeSys,
			"%s interval is invalid, defaulting to: %s",
			FeeManagement,
			DefaultFeeManagerDelay)
		f.sleep = DefaultFeeManagerDelay
	}
	f.Manager = fee.GetManager()
	f.iExchangeManager = em
	f.shutdown = make(chan struct{})
	return nil
}

// Start runs the subsystem
func (f *feeManager) Start() error {
	if f == nil {
		return fmt.Errorf("%s %w", FeeManagement, ErrNilSubsystem)
	}

	if f.Manager == nil {
		return fmt.Errorf("%s %w", FeeManagement, errSubsystemNotSetup)
	}

	if !atomic.CompareAndSwapInt32(&f.started, 0, 1) {
		return fmt.Errorf("%s %w", FeeManagement, ErrSubSystemAlreadyStarted)
	}
	f.wg.Add(1)
	go f.monitor()
	return nil
}

// Stop stops the subsystem
func (f *feeManager) Stop() error {
	if f == nil {
		return fmt.Errorf("%s %w", FeeManagement, ErrNilSubsystem)
	}
	if atomic.LoadInt32(&f.started) == 0 {
		return fmt.Errorf("%s %w", FeeManagement, ErrSubSystemNotStarted)
	}

	log.Debugf(log.ExchangeSys, "%s %s", FeeManagement, MsgSubSystemShuttingDown)
	close(f.shutdown)
	f.wg.Wait()
	f.shutdown = make(chan struct{})
	log.Debugf(log.ExchangeSys, "%s %s", FeeManagement, MsgSubSystemShutdown)
	atomic.CompareAndSwapInt32(&f.started, 1, 0)
	return nil
}

// IsRunning safely checks whether the subsystem is running
func (f *feeManager) IsRunning() bool {
	if f == nil {
		return false
	}
	return atomic.LoadInt32(&f.started) == 1
}

func (f *feeManager) monitor() {
	defer f.wg.Done()
	timer := time.NewTimer(0) // Prime fireing of channel for initial sync.
	for {
		select {
		case <-f.shutdown:
			return

		case <-timer.C:
			var wg sync.WaitGroup
			exchs := f.GetExchanges()
			for x := range exchs {
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
	for y := range enabledAssets {
		err := exch.UpdateFees(enabledAssets[y])
		if err != nil && !errors.Is(err, common.ErrNotYetImplemented) {
			log.Errorf(log.ExchangeSys, "%s %s %s: %v",
				FeeManagement,
				exch.GetName(),
				enabledAssets[y],
				err)
		}
	}
}
