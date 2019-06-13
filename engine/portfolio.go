package engine

import (
	"errors"
	"sync/atomic"
	"time"

	log "github.com/thrasher-/gocryptotrader/logger"
	"github.com/thrasher-/gocryptotrader/portfolio"
)

// vars for the fund manager package
var (
	PortfolioSleepDelay = time.Minute
)

type portfolioManager struct {
	started  int32
	stopped  int32
	shutdown chan struct{}
}

func (p *portfolioManager) Started() bool {
	return atomic.LoadInt32(&p.started) == 1
}

func (p *portfolioManager) Start() error {
	if atomic.AddInt32(&p.started, 1) != 1 {
		return errors.New("portfolio manager already started")
	}

	log.Debugln("Portfolio manager starting...")
	Bot.Portfolio = &portfolio.Portfolio
	Bot.Portfolio.Seed(Bot.Config.Portfolio)
	p.shutdown = make(chan struct{})
	go p.run()
	return nil
}
func (p *portfolioManager) Stop() error {
	if atomic.AddInt32(&p.stopped, 1) != 1 {
		return errors.New("portfolio manager is already stopped")
	}

	log.Debugln("Portfolio manager shutting down...")
	close(p.shutdown)
	return nil
}

func (p *portfolioManager) run() {
	log.Debugln("Portfolio manager started.")
	Bot.ServicesWG.Add(1)
	tick := time.NewTicker(PortfolioSleepDelay)
	defer func() {
		atomic.CompareAndSwapInt32(&p.stopped, 1, 0)
		atomic.CompareAndSwapInt32(&p.started, 1, 0)
		tick.Stop()
		Bot.ServicesWG.Done()
		log.Debugf("Portfolio manager shutdown.")
	}()

	for {
		select {
		case <-p.shutdown:
			return

		case <-tick.C:
			p.processPortfolio()
		}
	}
}

func (p *portfolioManager) processPortfolio() {
	pf := portfolio.GetPortfolio()
	data := pf.GetPortfolioGroupedCoin()
	for key, value := range data {
		success := pf.UpdatePortfolio(value, key)
		if success {
			log.Debugf(
				"Portfolio manager: Successfully updated address balance for %s address(es) %s\n",
				key, value,
			)
		}
	}
	SeedExchangeAccountInfo(GetAllEnabledExchangeAccountInfo().Data)
}
