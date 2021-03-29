package portfoliomanager

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
)

// vars for the fund manager package
var (
	PortfolioSleepDelay = time.Minute
)

type Manager struct {
	started    int32
	processing int32
	shutdown   chan struct{}
}

func (m *Manager) Started() bool {
	return atomic.LoadInt32(&m.started) == 1
}

func (m *Manager) Start() error {
	if atomic.AddInt32(&m.started, 1) != 1 {
		return errors.New("portfolio manager already started")
	}

	log.Debugln(log.PortfolioMgr, "Portfolio manager starting...")
	engine.Bot.Portfolio = &portfolio.Portfolio
	engine.Bot.Portfolio.Seed(engine.Bot.Config.Portfolio)
	m.shutdown = make(chan struct{})
	portfolio.Verbose = engine.Bot.Settings.Verbose

	go m.run()
	return nil
}
func (m *Manager) Stop() error {
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("portfolio manager %w", subsystems.ErrSubSystemNotStarted)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
	}()

	log.Debugln(log.PortfolioMgr, "Portfolio manager shutting down...")
	close(m.shutdown)
	return nil
}

func (m *Manager) run() {
	log.Debugln(log.PortfolioMgr, "Portfolio manager started.")
	engine.Bot.ServicesWG.Add(1)
	tick := time.NewTicker(engine.Bot.Settings.PortfolioManagerDelay)
	defer func() {
		tick.Stop()
		engine.Bot.ServicesWG.Done()
		log.Debugf(log.PortfolioMgr, "Portfolio manager shutdown.")
	}()

	go m.processPortfolio()
	for {
		select {
		case <-m.shutdown:
			return
		case <-tick.C:
			go m.processPortfolio()
		}
	}
}

func (m *Manager) processPortfolio() {
	if !atomic.CompareAndSwapInt32(&m.processing, 0, 1) {
		return
	}
	pf := portfolio.GetPortfolio()
	data := pf.GetPortfolioGroupedCoin()
	for key, value := range data {
		err := pf.UpdatePortfolio(value, key)
		if err != nil {
			log.Errorf(log.PortfolioMgr,
				"PortfolioWatcher error %s for currency %s\n",
				err,
				key)
			continue
		}

		log.Debugf(log.PortfolioMgr,
			"Portfolio manager: Successfully updated address balance for %s address(es) %s\n",
			key,
			value)
	}
	engine.SeedExchangeAccountInfo(engine.Bot.GetAllEnabledExchangeAccountInfo().Data)
	atomic.CompareAndSwapInt32(&m.processing, 1, 0)
}
