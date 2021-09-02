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
	"github.com/thrasher-corp/gocryptotrader/exchanges/state"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	StateManagement          = "StateManagement"
	DefaultStateManagerDelay = time.Minute
)

var (
	errSubsystemNotSetup = errors.New("subsystem not set up")
	errNilInterface      = errors.New("interface is nil")
)

// stateManager provides external package coupling for method application
type stateManager struct {
	started  int32
	shutdown chan struct{}
	wg       sync.WaitGroup
	*state.Manager
	iExchangeManager
	sleep time.Duration
}

// Setup applies configuration paramaters before running
func (s *stateManager) Setup(interval time.Duration, em iExchangeManager) error {
	if em == nil {
		return errNilInterface
	}
	s.sleep = interval
	if interval <= 0 {
		log.Warnf(log.ExchangeSys,
			"%s interval is invalid, defaulting to: %s",
			StateManagement,
			DefaultStateManagerDelay)
		s.sleep = DefaultStateManagerDelay
	}
	s.Manager = state.GetManager()
	s.iExchangeManager = em
	s.shutdown = make(chan struct{})
	return nil
}

// Start runs the subsystem
func (s *stateManager) Start() error {
	if s == nil {
		return fmt.Errorf("%s %w", StateManagement, ErrNilSubsystem)
	}

	if s.Manager == nil {
		return fmt.Errorf("%s %w", StateManagement, errSubsystemNotSetup)
	}

	if !atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		return fmt.Errorf("%s %w", StateManagement, ErrSubSystemAlreadyStarted)
	}
	s.wg.Add(1)
	go s.monitor()
	return nil
}

// Stop stops the subsystem
func (s *stateManager) Stop() error {
	if s == nil {
		return fmt.Errorf("%s %w", StateManagement, ErrNilSubsystem)
	}
	if atomic.LoadInt32(&s.started) == 0 {
		return fmt.Errorf("%s %w", StateManagement, ErrSubSystemNotStarted)
	}

	log.Debugf(log.ExchangeSys, "%s %s", StateManagement, MsgSubSystemShuttingDown)
	close(s.shutdown)
	s.wg.Wait()
	s.shutdown = make(chan struct{})
	log.Debugf(log.ExchangeSys, "%s %s", StateManagement, MsgSubSystemShutdown)
	atomic.CompareAndSwapInt32(&s.started, 1, 0)
	return nil
}

// IsRunning safely checks whether the subsystem is running
func (s *stateManager) IsRunning() bool {
	if s == nil {
		return false
	}
	return atomic.LoadInt32(&s.started) == 1
}

func (s *stateManager) monitor() {
	defer s.wg.Done()
	timer := time.NewTimer(0) // Prime fireing of channel for initial sync.
	for {
		select {
		case <-s.shutdown:
			return

		case <-timer.C:
			var wg sync.WaitGroup
			exchs := s.GetExchanges()
			for x := range exchs {
				wg.Add(1)
				go update(exchs[x], &wg, exchs[x].GetAssetTypes(true))
			}
			wg.Wait() // This causes some variability in the timer due to
			// longest length of request time. Can do time.Ticker but don't
			// want routines to stack behind, this is more uniform.
			timer.Reset(s.sleep)
		}
	}
}

func update(exch exchange.IBotExchange, wg *sync.WaitGroup, enabledAssets asset.Items) {
	defer wg.Done()
	for y := range enabledAssets {
		err := exch.UpdateCurrencyStates(enabledAssets[y])
		if err != nil && !errors.Is(err, common.ErrNotYetImplemented) {
			log.Errorf(log.ExchangeSys, "%s %s %s: %v",
				StateManagement,
				exch.GetName(),
				enabledAssets[y],
				err)
		}
	}
}
