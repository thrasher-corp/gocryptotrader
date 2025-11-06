package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
)

// PortfolioManagerName is an exported subsystem name
const PortfolioManagerName = "portfolio"

// PortfolioSleepDelay defines the default sleep time between portfolio manager runs
var PortfolioSleepDelay = time.Minute

// portfolioManager routinely retrieves a user's holdings through exchange APIs as well
// as through addresses provided in the config
type portfolioManager struct {
	started               int32
	processing            int32
	portfolioManagerDelay time.Duration
	exchangeManager       *ExchangeManager
	shutdown              chan struct{}
	base                  *portfolio.Base
	m                     sync.Mutex
}

// setupPortfolioManager creates a new portfolio manager
func setupPortfolioManager(e *ExchangeManager, portfolioManagerDelay time.Duration, cfg *portfolio.Base) (*portfolioManager, error) {
	if e == nil {
		return nil, errNilExchangeManager
	}
	if portfolioManagerDelay <= 0 {
		portfolioManagerDelay = PortfolioSleepDelay
	}

	if cfg == nil {
		cfg = &portfolio.Base{Addresses: []portfolio.Address{}}
	}
	m := &portfolioManager{
		portfolioManagerDelay: portfolioManagerDelay,
		exchangeManager:       e,
		shutdown:              make(chan struct{}),
		base:                  cfg,
	}
	return m, nil
}

// IsRunning safely checks whether the subsystem is running
func (m *portfolioManager) IsRunning() bool {
	return m != nil && atomic.LoadInt32(&m.started) == 1
}

// Start runs the subsystem
func (m *portfolioManager) Start(wg *sync.WaitGroup) error {
	if m == nil {
		return fmt.Errorf("portfolio manager %w", ErrNilSubsystem)
	}
	if wg == nil {
		return errNilWaitGroup
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("portfolio manager %w", ErrSubSystemAlreadyStarted)
	}

	log.Debugf(log.PortfolioMgr, "Portfolio manager %s", MsgSubSystemStarting)
	m.shutdown = make(chan struct{})
	wg.Add(1)
	go m.run(wg)
	return nil
}

// Stop attempts to shutdown the subsystem
func (m *portfolioManager) Stop() error {
	if m == nil {
		return fmt.Errorf("portfolio manager %w", ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 1, 0) {
		return fmt.Errorf("portfolio manager %w", ErrSubSystemNotStarted)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
	}()

	log.Debugf(log.PortfolioMgr, "Portfolio manager %s", MsgSubSystemShuttingDown)
	close(m.shutdown)
	return nil
}

// run periodically will check and update portfolio holdings
func (m *portfolioManager) run(wg *sync.WaitGroup) {
	log.Debugln(log.PortfolioMgr, "Portfolio manager started.")
	timer := time.NewTimer(0)
	for {
		select {
		case <-m.shutdown:
			if !timer.Stop() {
				<-timer.C
			}
			wg.Done()
			log.Debugf(log.PortfolioMgr, "Portfolio manager shutdown.")
			return
		case <-timer.C:
			// This is run in a go-routine to not prevent the application from
			// shutting down.
			go m.processPortfolio()
			timer.Reset(m.portfolioManagerDelay)
		}
	}
}

// processPortfolio updates portfolio holdings
func (m *portfolioManager) processPortfolio() {
	if !atomic.CompareAndSwapInt32(&m.processing, 0, 1) {
		return
	}
	m.m.Lock()
	defer m.m.Unlock()
	if err := m.updateExchangeBalances(); err != nil {
		log.Errorf(log.PortfolioMgr, "Portfolio updateExchangeBalances error: %v", err)
	}

	data := m.base.GetPortfolioAddressesGroupedByCoin()
	for key, value := range data {
		if err := m.base.UpdatePortfolio(context.TODO(), value, key); err != nil {
			log.Errorf(log.PortfolioMgr, "Portfolio manager: UpdatePortfolio error: %s for currency %s", err, key)
			continue
		}

		log.Debugf(log.PortfolioMgr, "Portfolio manager: Successfully updated address balance for %s address(es) %s", key, value)
	}
	atomic.CompareAndSwapInt32(&m.processing, 1, 0)
}

// updateExchangeBalances calls UpdateAccountBalance on each exchange, and transfers the account balances into portfolio
func (m *portfolioManager) updateExchangeBalances() error {
	if err := common.NilGuard(m); err != nil {
		return err
	}
	exchanges, errs := m.exchangeManager.GetExchanges()
	if errs != nil {
		return fmt.Errorf("portfolio manager cannot get exchanges: %w", errs)
	}
	for _, e := range exchanges {
		if !e.IsEnabled() {
			continue
		}
		if !e.IsRESTAuthenticationSupported() {
			if m.base.Verbose {
				log.Debugf(log.PortfolioMgr, "Portfolio skipping %s due to disabled authenticated API support", e.GetName())
			}
			continue
		}
		assetTypes := asset.Items{asset.Spot}
		if e.HasAssetTypeAccountSegregation() {
			assetTypes = e.GetAssetTypes(true)
		}

		for _, a := range assetTypes {
			if _, err := e.UpdateAccountBalances(context.TODO(), a); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("error updating %s %s account balances: %w", e.GetName(), a, err))
			}
		}
		if err := m.updateExchangeAddressBalances(e); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("error updating %s account balances: %w", e.GetName(), err))
		}
	}
	return errs
}

// updateExchangeAddressBalances fetches and collates all account balances with their deposit addresses
func (m *portfolioManager) updateExchangeAddressBalances(e exchange.IBotExchange) error {
	if err := common.NilGuard(m, e); err != nil {
		return err
	}
	currs, err := e.GetBase().Accounts.CurrencyBalances(nil, asset.All)
	if err != nil {
		return err
	}
	eName := e.GetName()
	for c, b := range currs {
		if !m.base.ExchangeAddressCoinExists(e.GetName(), c) {
			if b.Total <= 0 {
				continue
			}

			log.Debugf(log.PortfolioMgr, "Portfolio: Adding new exchange address: %s, %s, %f, %s", eName, c, b.Total, portfolio.ExchangeAddress)

			m.base.Addresses = append(m.base.Addresses, portfolio.Address{
				Address:     eName,
				CoinType:    c,
				Balance:     b.Total,
				Description: portfolio.ExchangeAddress,
			})
			continue
		}

		if b.Total <= 0 {
			log.Debugf(log.PortfolioMgr, "Portfolio: Removing %s %s entry", eName, c)
			m.base.RemoveExchangeAddress(eName, c)
			continue
		}

		if balance, ok := m.base.GetAddressBalance(eName, portfolio.ExchangeAddress, c); ok && balance != b.Total {
			log.Debugf(log.PortfolioMgr, "Portfolio: Updating %s %s entry with balance %f", eName, c, b.Total)
			m.base.UpdateExchangeAddressBalance(eName, c, b.Total)
		}
	}
	return nil
}

// AddAddress adds a new portfolio address for the portfolio manager to track
func (m *portfolioManager) AddAddress(address, description string, coinType currency.Code, balance float64) error {
	if m == nil {
		return fmt.Errorf("portfolio manager %w", ErrNilSubsystem)
	}
	if !m.IsRunning() {
		return fmt.Errorf("portfolio manager %w", ErrSubSystemNotStarted)
	}
	m.m.Lock()
	defer m.m.Unlock()
	return m.base.AddAddress(address, description, coinType, balance)
}

// RemoveAddress removes a portfolio address
func (m *portfolioManager) RemoveAddress(address, description string, coinType currency.Code) error {
	if m == nil {
		return fmt.Errorf("portfolio manager %w", ErrNilSubsystem)
	}
	if !m.IsRunning() {
		return fmt.Errorf("portfolio manager %w", ErrSubSystemNotStarted)
	}
	m.m.Lock()
	defer m.m.Unlock()
	return m.base.RemoveAddress(address, description, coinType)
}

// GetPortfolioSummary returns a summary of all portfolio holdings
func (m *portfolioManager) GetPortfolioSummary() portfolio.Summary {
	if m == nil || !m.IsRunning() {
		return portfolio.Summary{}
	}
	return m.base.GetPortfolioSummary()
}

// GetAddresses returns all addresses
func (m *portfolioManager) GetAddresses() []portfolio.Address {
	if m == nil || !m.IsRunning() {
		return nil
	}
	return m.base.Addresses
}

// GetPortfolio returns a copy of the internal portfolio base for
// saving addresses to the config
func (m *portfolioManager) GetPortfolio() *portfolio.Base {
	if m == nil || !m.IsRunning() {
		return nil
	}
	resp := m.base
	return resp
}

// IsWhiteListed checks if an address is whitelisted to withdraw to
func (m *portfolioManager) IsWhiteListed(address string) bool {
	if m == nil || !m.IsRunning() {
		return false
	}
	return m.base.IsWhiteListed(address)
}

// IsExchangeSupported checks if an exchange is supported
func (m *portfolioManager) IsExchangeSupported(exch, address string) bool {
	if m == nil || !m.IsRunning() {
		return false
	}
	return m.base.IsExchangeSupported(exch, address)
}
