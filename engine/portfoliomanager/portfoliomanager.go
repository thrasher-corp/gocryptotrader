package portfoliomanager

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchangemanager "github.com/thrasher-corp/gocryptotrader/engine/exchangemanager"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystems"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
)

// Name is an exported subsystem name
const Name = "portfolio"

var (
	// PortfolioSleepDelay defines the default sleep time between portfolio manager runs
	PortfolioSleepDelay   = time.Minute
	errNilExchangeManager = errors.New("nil exchange manager base received")
	errNilWaitGroup       = errors.New("nil wait group received")
)

// Manager routinely retrieves a user's holdings through exchange APIs as well
// as through addresses provided in the config
type Manager struct {
	started               int32
	processing            int32
	portfolioManagerDelay time.Duration
	exchangeManager       *exchangemanager.Manager
	shutdown              chan struct{}
	base                  *portfolio.Base
}

// Setup creates a new portfolio manager
func Setup(e *exchangemanager.Manager, portfolioManagerDelay time.Duration, cfg *portfolio.Base) (*Manager, error) {
	if e == nil {
		return nil, errNilExchangeManager
	}
	if portfolioManagerDelay <= 0 {
		portfolioManagerDelay = PortfolioSleepDelay
	}
	if cfg == nil {
		cfg = &portfolio.Base{Addresses: []portfolio.Address{}}
	}
	m := &Manager{
		portfolioManagerDelay: portfolioManagerDelay,
		exchangeManager:       e,
		shutdown:              make(chan struct{}),
		base:                  cfg,
	}
	return m, nil
}

// IsRunning safely checks whether the subsystem is running
func (m *Manager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

// Start runs the subsystem
func (m *Manager) Start(wg *sync.WaitGroup) error {
	if m == nil {
		return fmt.Errorf("portfolio manager %w", subsystems.ErrNilSubsystem)
	}
	if wg == nil {
		return errNilWaitGroup
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("portfolio manager %w", subsystems.ErrSubSystemAlreadyStarted)
	}

	log.Debugf(log.PortfolioMgr, "Portfolio manager %s", subsystems.MsgSubSystemStarting)
	m.shutdown = make(chan struct{})
	go m.run(wg)
	return nil
}

// Stop attempts to shutdown the subsystem
func (m *Manager) Stop() error {
	if m == nil {
		return fmt.Errorf("portfolio manager %w", subsystems.ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 1, 0) {
		return fmt.Errorf("portfolio manager %w", subsystems.ErrSubSystemNotStarted)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
	}()

	log.Debugf(log.PortfolioMgr, "Portfolio manager %s", subsystems.MsgSubSystemShuttingDown)
	close(m.shutdown)
	return nil
}

// run periodically will check and update portfolio holdings
func (m *Manager) run(wg *sync.WaitGroup) {
	log.Debugln(log.PortfolioMgr, "Portfolio manager started.")
	wg.Add(1)
	tick := time.NewTicker(m.portfolioManagerDelay)
	defer func() {
		tick.Stop()
		wg.Done()
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

// processPortfolio updates an portfolio holdings
func (m *Manager) processPortfolio() {
	if !atomic.CompareAndSwapInt32(&m.processing, 0, 1) {
		return
	}
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

	d := m.getExchangeAccountInfo(m.exchangeManager.GetExchanges())
	m.seedExchangeAccountInfo(d)
	atomic.CompareAndSwapInt32(&m.processing, 1, 0)
}

// seedExchangeAccountInfo seeds account info
func (m *Manager) seedExchangeAccountInfo(accounts []account.Holdings) {
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
					if accounts[x].Accounts[y].Currencies[z].CurrencyName == currencies[i].CurrencyName {
						currencies[i].Hold += accounts[x].Accounts[y].Currencies[z].Hold
						currencies[i].TotalValue += accounts[x].Accounts[y].Currencies[z].TotalValue
						update = true
					}
				}

				if update {
					continue
				}

				currencies = append(currencies, account.Balance{
					CurrencyName: accounts[x].Accounts[y].Currencies[z].CurrencyName,
					TotalValue:   accounts[x].Accounts[y].Currencies[z].TotalValue,
					Hold:         accounts[x].Accounts[y].Currencies[z].Hold,
				})
			}
		}

		for x := range currencies {
			currencyName := currencies[x].CurrencyName
			total := currencies[x].TotalValue

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
func (m *Manager) getExchangeAccountInfo(exchanges []exchange.IBotExchange) []account.Holdings {
	var response []account.Holdings
	for x := range exchanges {
		if exchanges[x] == nil || !exchanges[x].IsEnabled() {
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
		assetTypes := exchanges[x].GetAssetTypes()
		var exchangeHoldings account.Holdings
		for y := range assetTypes {
			accountHoldings, err := exchanges[x].FetchAccountInfo(assetTypes[y])
			if err != nil {
				log.Errorf(log.PortfolioMgr,
					"Error encountered retrieving exchange account info for %s. Error %s\n",
					exchanges[x].GetName(),
					err)
				continue
			}
			for z := range accountHoldings.Accounts {
				accountHoldings.Accounts[z].AssetType = assetTypes[y]
			}
			exchangeHoldings.Exchange = exchanges[x].GetName()
			exchangeHoldings.Accounts = append(exchangeHoldings.Accounts, accountHoldings.Accounts...)
		}
		response = append(response, exchangeHoldings)
	}
	return response
}

// AddAddress adds a new portfolio address for the portfolio manager to track
func (m *Manager) AddAddress(address, description string, coinType currency.Code, balance float64) error {
	if m == nil {
		return fmt.Errorf("portfolio manager %w", subsystems.ErrNilSubsystem)
	}
	if !m.IsRunning() {
		return fmt.Errorf("portfolio manager %w", subsystems.ErrSubSystemNotStarted)
	}
	return m.base.AddAddress(address, description, coinType, balance)
}

// RemoveAddress removes a portfolio address
func (m *Manager) RemoveAddress(address, description string, coinType currency.Code) error {
	if m == nil {
		return fmt.Errorf("portfolio manager %w", subsystems.ErrNilSubsystem)
	}
	if !m.IsRunning() {
		return fmt.Errorf("portfolio manager %w", subsystems.ErrSubSystemNotStarted)
	}
	return m.base.RemoveAddress(address, description, coinType)
}

// GetPortfolioSummary returns a summary of all portfolio holdings
func (m *Manager) GetPortfolioSummary() portfolio.Summary {
	if m == nil || !m.IsRunning() {
		return portfolio.Summary{}
	}
	return m.base.GetPortfolioSummary()
}

// GetAddresses returns all addresses
func (m *Manager) GetAddresses() []portfolio.Address {
	if m == nil || !m.IsRunning() {
		return nil
	}
	return m.base.Addresses
}

// GetPortfolio returns a copy of the internal portfolio base for
// saving addresses to the config
func (m *Manager) GetPortfolio() *portfolio.Base {
	if m == nil || !m.IsRunning() {
		return nil
	}
	resp := m.base
	return resp
}

// IsWhiteListed checks if an address is whitelisted to withdraw to
func (m *Manager) IsWhiteListed(address string) bool {
	if m == nil || !m.IsRunning() {
		return false
	}
	return m.base.IsWhiteListed(address)
}

// IsExchangeSupported checks if an exchange is supported
func (m *Manager) IsExchangeSupported(exchange, address string) bool {
	if m == nil || !m.IsRunning() {
		return false
	}
	return m.base.IsExchangeSupported(exchange, address)
}
