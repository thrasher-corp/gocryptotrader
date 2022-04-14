package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
)

// PortfolioManagerName is an exported subsystem name
const PortfolioManagerName = "portfolio"

var (
	// PortfolioSleepDelay defines the default sleep time between portfolio manager runs
	PortfolioSleepDelay = time.Minute
)

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
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
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
			m.processPortfolio()
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
	exchanges, err := m.exchangeManager.GetExchanges()
	if err != nil {
		log.Errorf(log.PortfolioMgr, "Portfolio manager cannot get exchanges: %v", err)
	}
	allExchangesHoldings := m.getExchangeAccountInfo(exchanges)
	m.seedExchangeAccountInfo(allExchangesHoldings)

	data := m.base.GetPortfolioGroupedCoin()
	for key, value := range data {
		err := m.base.UpdatePortfolio(value, key)
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
	atomic.CompareAndSwapInt32(&m.processing, 1, 0)
}

// seedExchangeAccountInfo seeds account info
func (m *portfolioManager) seedExchangeAccountInfo(accounts []account.Holdings) {
	if len(accounts) == 0 {
		return
	}
	for x := range accounts {
		exchangeName := accounts[x].Exchange
		var currencies []account.Balance
		for y := range accounts[x].Accounts {
			for z := range accounts[x].Accounts[y].Currencies {
				var update bool
				for i := range currencies {
					if !accounts[x].Accounts[y].Currencies[z].CurrencyName.Equal(currencies[i].CurrencyName) {
						continue
					}
					currencies[i].Hold += accounts[x].Accounts[y].Currencies[z].Hold
					currencies[i].Total += accounts[x].Accounts[y].Currencies[z].Total
					currencies[i].AvailableWithoutBorrow += accounts[x].Accounts[y].Currencies[z].AvailableWithoutBorrow
					currencies[i].Free += accounts[x].Accounts[y].Currencies[z].Free
					currencies[i].Borrowed += accounts[x].Accounts[y].Currencies[z].Borrowed
					update = true
				}
				if update {
					continue
				}
				currencies = append(currencies, account.Balance{
					CurrencyName:           accounts[x].Accounts[y].Currencies[z].CurrencyName,
					Total:                  accounts[x].Accounts[y].Currencies[z].Total,
					Hold:                   accounts[x].Accounts[y].Currencies[z].Hold,
					Free:                   accounts[x].Accounts[y].Currencies[z].Free,
					AvailableWithoutBorrow: accounts[x].Accounts[y].Currencies[z].AvailableWithoutBorrow,
					Borrowed:               accounts[x].Accounts[y].Currencies[z].Borrowed,
				})
			}
		}

		for x := range currencies {
			currencyName := currencies[x].CurrencyName
			total := currencies[x].Total

			if !m.base.ExchangeAddressExists(exchangeName, currencyName) {
				if total <= 0 {
					continue
				}

				log.Debugf(log.PortfolioMgr, "Portfolio: Adding new exchange address: %s, %s, %f, %s\n",
					exchangeName,
					currencyName,
					total,
					portfolio.ExchangeAddress)

				m.base.Addresses = append(
					m.base.Addresses,
					portfolio.Address{Address: exchangeName,
						CoinType:    currencyName,
						Balance:     total,
						Description: portfolio.ExchangeAddress})
			} else {
				if total <= 0 {
					log.Debugf(log.PortfolioMgr, "Portfolio: Removing %s %s entry.\n",
						exchangeName,
						currencyName)
					m.base.RemoveExchangeAddress(exchangeName, currencyName)
				} else {
					balance, ok := m.base.GetAddressBalance(exchangeName,
						portfolio.ExchangeAddress,
						currencyName)
					if !ok {
						continue
					}

					if balance != total {
						log.Debugf(log.PortfolioMgr, "Portfolio: Updating %s %s entry with balance %f.\n",
							exchangeName,
							currencyName,
							total)
						m.base.UpdateExchangeAddressBalance(exchangeName,
							currencyName,
							total)
					}
				}
			}
		}
	}
}

// getExchangeAccountInfo returns all the current enabled exchanges
func (m *portfolioManager) getExchangeAccountInfo(exchanges []exchange.IBotExchange) []account.Holdings {
	var response []account.Holdings
	for x := range exchanges {
		if !exchanges[x].IsEnabled() {
			continue
		}
		if !exchanges[x].GetAuthenticatedAPISupport(exchange.RestAuthentication) {
			if m.base.Verbose {
				log.Debugf(log.PortfolioMgr,
					"skipping %s due to disabled authenticated API support.\n",
					exchanges[x].GetName())
			}
			continue
		}

		assetTypes := asset.Items{asset.Spot}
		if exchanges[x].HasAssetTypeAccountSegregation() {
			// Get entire supported exchange asset types to sync all account
			// information.
			assetTypes = exchanges[x].GetAssetTypes(false)
		}

		exchangeHoldings := account.Holdings{Accounts: make([]account.SubAccount, len(assetTypes))}
		for y := range assetTypes {
			// Update account info to process account updates in memory on
			// every fetch.
			accountHoldings, err := exchanges[x].UpdateAccountInfo(context.TODO(), assetTypes[y])
			if err != nil {
				log.Errorf(log.PortfolioMgr,
					"Error encountered retrieving exchange account info for %s. Error %s\n",
					exchanges[x].GetName(),
					err)
				continue
			}
			exchangeHoldings.Exchange = exchanges[x].GetName()
			exchangeHoldings.Accounts = append(exchangeHoldings.Accounts, accountHoldings.Accounts...)
		}
		response = append(response, exchangeHoldings)
	}
	return response
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
func (m *portfolioManager) IsExchangeSupported(exchange, address string) bool {
	if m == nil || !m.IsRunning() {
		return false
	}
	return m.base.IsExchangeSupported(exchange, address)
}
