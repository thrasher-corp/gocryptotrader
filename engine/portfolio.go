package engine

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
)

// vars for the fund manager package
var (
	PortfolioSleepDelay = time.Minute
)

type portfolioManager struct {
	started    int32
	processing int32
	shutdown   chan struct{}
}

func (p *portfolioManager) Started() bool {
	return atomic.LoadInt32(&p.started) == 1
}

func (p *portfolioManager) Start() error {
	if atomic.AddInt32(&p.started, 1) != 1 {
		return errors.New("portfolio manager already started")
	}

	log.Debugln(log.PortfolioMgr, "Portfolio manager starting...")
	Bot.Portfolio = &portfolio.Portfolio
	Bot.Portfolio.Seed(Bot.Config.Portfolio)
	p.shutdown = make(chan struct{})
	portfolio.Verbose = Bot.Settings.Verbose

	go p.run()
	return nil
}
func (p *portfolioManager) Stop() error {
	if atomic.LoadInt32(&p.started) == 0 {
		return fmt.Errorf("portfolio manager %w", subsystem.ErrSubSystemNotStarted)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&p.started, 1, 0)
	}()

	log.Debugln(log.PortfolioMgr, "Portfolio manager shutting down...")
	close(p.shutdown)
	return nil
}

func (p *portfolioManager) run() {
	log.Debugln(log.PortfolioMgr, "Portfolio manager started.")
	Bot.ServicesWG.Add(1)
	tick := time.NewTicker(Bot.Settings.PortfolioManagerDelay)
	defer func() {
		tick.Stop()
		Bot.ServicesWG.Done()
		log.Debugf(log.PortfolioMgr, "Portfolio manager shutdown.")
	}()

	go p.processPortfolio()
	for {
		select {
		case <-p.shutdown:
			return
		case <-tick.C:
			go p.processPortfolio()
		}
	}
}

func (p *portfolioManager) processPortfolio() {
	if !atomic.CompareAndSwapInt32(&p.processing, 0, 1) {
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
	SeedExchangeAccountInfo(Bot.GetAllEnabledExchangeAccountInfo().Data)
	atomic.CompareAndSwapInt32(&p.processing, 1, 0)
}
