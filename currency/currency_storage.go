package currency

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/coinmarketcap"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/base"
	log "github.com/thrasher-/gocryptotrader/logger"
)

func init() {
	storage.SetDefaults()
}

// storage is an overarching type that keeps track of and updates currency,
// currency exchange rates and pairs
var storage Storage

// Storage contains the loaded storage currencies supported on available crypto
// or fiat marketplaces
// NOTE: All internal currencies are upper case
type Storage struct {
	// FiatCurrencies defines the running fiat currencies in the currency
	// storage
	fiatCurrencies Currencies

	// Cryptocurrencies defines the running cryptocurrencies in the currency
	// storage
	cryptocurrencies Currencies

	// CurrencyCodes is a full basket of currencies either crypto, fiat, ico or
	// contract being tracked by the currency storage
	currencyCodes BaseCodes

	// Main convert currency
	baseCurrency Code

	// FXRates defines a protected conversion rate map
	fxRates ConversionRates

	// DefaultBaseCurrency is the base currency used for conversion
	defaultBaseCurrency Code

	// DefaultFiatCurrencies has the default minimum of FIAT values
	defaultFiatCurrencies Currencies

	// DefaultCryptoCurrencies has the default minimum of crytpocurrency values
	defaultCryptoCurrencies Currencies

	// FiatExchangeMarkets defines an interface to access FX data for fiat
	// currency rates
	fiatExchangeMarkets *forexprovider.ForexProviders

	// CurrencyAnalysis defines a full market analysis suite to receieve and
	// define different fiat currencies, cryptocurrencies and markets
	currencyAnalysis *coinmarketcap.Coinmarketcap

	wg             sync.WaitGroup
	shutdownC      chan struct{}
	updaterRunning bool
	Verbose        bool
}

// SetDefaults sets storage defaults for basic package functionality
func (s *Storage) SetDefaults() {
	s.defaultBaseCurrency = USD
	s.baseCurrency = s.defaultBaseCurrency
	s.SetDefaultFiatCurrencies(USD, AUD, EUR, CNY)
	s.SetDefaultCryptocurrencies(BTC, LTC, ETH, DOGE, DASH, XRP, XMR)
	s.SetupConversionRates()
	s.fiatExchangeMarkets = forexprovider.NewDefaultFXProvider()
}

// RunUpdater runs the foreign exchange updater service. This will set up a JSON
// dump file and keep foreign exchange rates updated as fast as possible without
// triggering rate limiters, it will also run a full cryptocurrency check
// through coin market cap and expose analytics for exchange services
func (s *Storage) RunUpdater(overrides BotOverrides, settings MainConfiguration, filePath string, verbose bool) error {
	if !settings.Cryptocurrencies.HasData() {
		return errors.New("currency storage error, no cryptocurrencies loaded")
	}
	s.cryptocurrencies = settings.Cryptocurrencies

	if settings.FiatDisplayCurrency.IsEmpty() {
		return errors.New("currency storage error, no fiat display currency set in config")
	}
	s.baseCurrency = settings.FiatDisplayCurrency
	log.Debugf("Fiat display currency: %s.", s.baseCurrency)

	var fxSettings []base.Settings
	for i := range settings.ForexProviders {
		switch settings.ForexProviders[i].Name {
		case "CurrencyConverter":
			if overrides.FxCurrencyConverter ||
				settings.ForexProviders[i].Enabled {
				settings.ForexProviders[i].Enabled = true
				fxSettings = append(fxSettings,
					base.Settings(settings.ForexProviders[i]))
			}

		case "CurrencyLayer":
			if overrides.FxCurrencyLayer || settings.ForexProviders[i].Enabled {
				settings.ForexProviders[i].Enabled = true
				fxSettings = append(fxSettings,
					base.Settings(settings.ForexProviders[i]))
			}

		case "Fixer":
			if overrides.FxFixer || settings.ForexProviders[i].Enabled {
				settings.ForexProviders[i].Enabled = true
				fxSettings = append(fxSettings,
					base.Settings(settings.ForexProviders[i]))
			}

		case "OpenExchangeRates":
			if overrides.FxOpenExchangeRates ||
				settings.ForexProviders[i].Enabled {
				settings.ForexProviders[i].Enabled = true
				fxSettings = append(fxSettings,
					base.Settings(settings.ForexProviders[i]))
			}

		case "ExchangeRates":
			// TODO ADD OVERRIDE
			if settings.ForexProviders[i].Enabled {
				settings.ForexProviders[i].Enabled = true
				fxSettings = append(fxSettings,
					base.Settings(settings.ForexProviders[i]))
			}
		}
	}

	if len(fxSettings) != 0 {
		var err error
		s.fiatExchangeMarkets, err = forexprovider.StartFXService(fxSettings)
		if err != nil {
			return err
		}

		log.Debugf("Primary foreign exchange conversion provider %s enabled",
			s.fiatExchangeMarkets.Primary.Provider.GetName())

		for i := range s.fiatExchangeMarkets.Support {
			log.Debugf("Support forex conversion provider %s enabled",
				s.fiatExchangeMarkets.Support[i].Provider.GetName())
		}

		go s.ForeignExchangeUpdater()
	} else {
		log.Warnf("No foreign exchange providers enabled in config.json")
	}

	return nil
}

// SetupConversionRates sets default conversion rate values
func (s *Storage) SetupConversionRates() {
	s.fxRates = ConversionRates{
		m: make(map[*Item]map[*Item]*float64),
	}
}

// SetDefaultFiatCurrencies assigns the default fiat currency list and adds it
// to the running list
func (s *Storage) SetDefaultFiatCurrencies(c ...Code) {
	for _, currency := range c {
		s.defaultFiatCurrencies = append(s.defaultFiatCurrencies, currency)
		s.fiatCurrencies = append(s.fiatCurrencies, currency)
	}
}

// SetDefaultCryptocurrencies assigns the default cryptocurrency list and adds
// it to the running list
func (s *Storage) SetDefaultCryptocurrencies(c ...Code) {
	for _, currency := range c {
		s.defaultCryptoCurrencies = append(s.defaultCryptoCurrencies, currency)
		s.cryptocurrencies = append(s.cryptocurrencies, currency)
	}
}

// SetupForexProviders sets up a new instance of the forex providers
func (s *Storage) SetupForexProviders(setting ...base.Settings) error {
	addr, err := forexprovider.StartFXService(setting)
	if err != nil {
		return err
	}

	s.fiatExchangeMarkets = addr
	return nil
}

// ForeignExchangeUpdater is a routine that seeds foreign exchange rate and keeps
// updated as fast as possible
func (s *Storage) ForeignExchangeUpdater() {
	log.Debugf("Foreign exchange updater started seeding Fx rate list..")
	s.wg.Add(1)
	defer func() {
		s.wg.Done()
		s.updaterRunning = false
	}()

	err := s.SeedForeignExchangeRates()
	if err != nil {
		log.Error(err)
	}

	t := time.NewTicker(1 * time.Minute)
	s.updaterRunning = true
	for {
		select {
		case <-s.shutdownC:
			return

		case <-t.C:
			err := s.SeedForeignExchangeRates()
			if err != nil {
				log.Error(err)
			}
		}
	}
}

// SeedForeignExchangeRatesByCurrencies seeds the foreign exchange rates by
// currencies supplied
func (s *Storage) SeedForeignExchangeRatesByCurrencies(c Currencies) error {
	s.fxRates.mtx.Lock()
	defer s.fxRates.mtx.Unlock()
	rates, err := s.fiatExchangeMarkets.GetCurrencyData(s.baseCurrency.String(),
		c.Strings())
	if err != nil {
		return err
	}
	return s.updateExchangeRates(rates)
}

// SeedForeignExchangeRate returns a singular exchange rate
func (s *Storage) SeedForeignExchangeRate(from, to Code) (map[string]float64, error) {
	return s.fiatExchangeMarkets.GetCurrencyData(from.String(),
		[]string{to.String()})
}

// GetDefaultForeignExchangeRates returns foreign exchange rates base off
// default fiat currencies.
func (s *Storage) GetDefaultForeignExchangeRates() (Conversions, error) {
	if !s.updaterRunning {
		err := s.SeedDefaultForeignExchangeRates()
		if err != nil {
			return nil, err
		}
	}
	return s.fxRates.GetFullRates(), nil
}

// SeedDefaultForeignExchangeRates seeds the default foreign exchange rates
func (s *Storage) SeedDefaultForeignExchangeRates() error {
	s.fxRates.mtx.Lock()
	defer s.fxRates.mtx.Unlock()
	rates, err := s.fiatExchangeMarkets.GetCurrencyData(
		s.defaultBaseCurrency.String(),
		s.defaultFiatCurrencies.Strings())
	if err != nil {
		return err
	}
	return s.updateExchangeRates(rates)
}

// GetExchangeRates returns storage seeded exchange rates
func (s *Storage) GetExchangeRates() (Conversions, error) {
	if !s.updaterRunning {
		err := s.SeedForeignExchangeRates()
		if err != nil {
			return nil, err
		}
	}
	return s.fxRates.GetFullRates(), nil
}

// SeedForeignExchangeRates seeds the foreign exchange rates from storage config
// currencies
func (s *Storage) SeedForeignExchangeRates() error {
	s.fxRates.mtx.Lock()
	defer s.fxRates.mtx.Unlock()
	rates, err := s.fiatExchangeMarkets.GetCurrencyData(
		s.baseCurrency.String(),
		s.fiatCurrencies.Strings())
	if err != nil {
		return err
	}
	return s.updateExchangeRates(rates)
}

// UpdateForeignExchangeRates sets exchange rates on the FX map
func (s *Storage) updateExchangeRates(m map[string]float64) error {
	return s.fxRates.Update(m)
}

// SetupCryptoProvider sets congiguration paramaters and starts a new instance
// of the currency analyser
func (s *Storage) SetupCryptoProvider(settings coinmarketcap.Settings) error {
	if settings.APIkey == "" ||
		settings.APIkey == "key" ||
		settings.AccountPlan == "" ||
		settings.AccountPlan == "accountPlan" {
		return errors.New("currencyprovider error api key or plan not set in config.json")
	}

	s.currencyAnalysis = new(coinmarketcap.Coinmarketcap)
	s.currencyAnalysis.SetDefaults()
	s.currencyAnalysis.Setup(settings)

	return nil
}

// GetTotalMarketCryptocurrencies returns the total seeded market
// cryptocurrencies
func (s *Storage) GetTotalMarketCryptocurrencies() (Currencies, error) {
	if !s.currencyCodes.HasData() {
		return nil, errors.New("market currency codes not populated")
	}
	return s.currencyCodes.GetCurrencies(), nil
}

// IsDefaultCurrency returns if a currency is a default currency
func (s *Storage) IsDefaultCurrency(c Code) bool {
	t, _ := GetTranslation(c)
	for _, d := range s.defaultFiatCurrencies {
		if d.Item == c.Item || d.Item == t.Item {
			return true
		}
	}
	return false
}

// IsDefaultCryptocurrency returns if a cryptocurrency is a default
// cryptocurrency
func (s *Storage) IsDefaultCryptocurrency(c Code) bool {
	t, _ := GetTranslation(c)
	for _, d := range s.defaultCryptoCurrencies {
		if d.Item == c.Item || d.Item == t.Item {
			return true
		}
	}
	return false
}

// IsFiatCurrency returns if a currency is part of the enabled fiat currency
// list
func (s *Storage) IsFiatCurrency(c Code) bool {
	t, _ := GetTranslation(c)
	for _, d := range s.fiatCurrencies {
		if d.Item == c.Item || d.Item == t.Item {
			return true
		}
	}
	return false
}

// IsCryptocurrency returns if a cryptocurrency is part of the enabled
// cryptocurrency list
func (s *Storage) IsCryptocurrency(c Code) bool {
	t, _ := GetTranslation(c)
	for _, d := range s.cryptocurrencies {
		if d.Item == c.Item || d.Item == t.Item {
			return true
		}
	}
	return false
}

// NewCode validates string against currency list and returns a currency
// code
func (s *Storage) NewCode(newCode string) Code {
	return s.currencyCodes.Register(newCode)
}

// NewValidFiatCode inserts a new code and updates the fiat currency list
// TODO: mutex protection
func (s *Storage) NewValidFiatCode(newCode string) Code {
	c := s.currencyCodes.Register(newCode)
	if !s.fiatCurrencies.Contains(c) {
		s.fiatCurrencies = append(s.fiatCurrencies, c)
	}
	return c
}

// NewCryptoCode inserts a new code and updates the crypto currency list
// TODO: mutex protection
func (s *Storage) NewCryptoCode(newCode string) Code {
	c := s.currencyCodes.Register(newCode)
	if !s.cryptocurrencies.Contains(c) {
		s.cryptocurrencies = append(s.cryptocurrencies, c)
	}
	return c
}

// UpdateBaseCurrency changes base currency
func (s *Storage) UpdateBaseCurrency(c Code) error {
	if c.IsFiatCurrency() {
		s.baseCurrency = c
		return nil
	}
	return fmt.Errorf("currency %s not fiat failed to set currency", c)
}

// GetCryptocurrencies returns the cryptocurrency list
func (s *Storage) GetCryptocurrencies() Currencies {
	return s.cryptocurrencies
}

// GetDefaultCryptocurrencies returns a list of default cryptocurrencies
func (s *Storage) GetDefaultCryptocurrencies() Currencies {
	return s.defaultCryptoCurrencies
}

// GetFiatCurrencies returns the fiat currencies list
func (s *Storage) GetFiatCurrencies() Currencies {
	return s.fiatCurrencies
}

// GetDefaultFiatCurrencies returns the default fiat currencies list
func (s *Storage) GetDefaultFiatCurrencies() Currencies {
	return s.defaultFiatCurrencies
}

// GetDefaultBaseCurrency returns the default base currency
func (s *Storage) GetDefaultBaseCurrency() Code {
	return s.defaultBaseCurrency
}

// GetBaseCurrency returns the current storage base currency
func (s *Storage) GetBaseCurrency() Code {
	return s.baseCurrency
}

// UpdateEnabledCryptoCurrencies appends new cryptocurrencies to the enabled
// currency list
func (s *Storage) UpdateEnabledCryptoCurrencies(c Currencies) {
	for _, i := range c {
		if !s.cryptocurrencies.Contains(i) {
			s.cryptocurrencies = append(s.cryptocurrencies, i)
		}
	}
}

// UpdateEnabledFiatCurrencies appends new fiat currencies to the enabled
// currency list
func (s *Storage) UpdateEnabledFiatCurrencies(c Currencies) {
	for _, i := range c {
		if !s.fiatCurrencies.Contains(i) && !s.cryptocurrencies.Contains(i) {
			s.fiatCurrencies = append(s.fiatCurrencies, i)
		}
	}
}

// ConvertCurrency for example converts $1 USD to the equivalent Japanese Yen
// or vice versa.
func (s *Storage) ConvertCurrency(amount float64, from, to Code) (float64, error) {
	if !s.fxRates.HasData() {
		err := s.SeedDefaultForeignExchangeRates()
		if err != nil {
			return 0, err
		}
	}

	r, err := s.fxRates.GetRate(from, to)
	if err != nil {
		return 0, err
	}

	return r * amount, nil
}

// GetStorageRate returns the rate of the conversion value
func (s *Storage) GetStorageRate(from, to Code) (float64, error) {
	if !s.fxRates.HasData() {
		err := s.SeedDefaultForeignExchangeRates()
		if err != nil {
			return 0, err
		}
	}

	return s.fxRates.GetRate(from, to)
}

// NewConversion returns a new conversion object that has a pointer to a related
// rate with its inversion.
func (s *Storage) NewConversion(from, to Code) (Conversion, error) {
	if !s.fxRates.HasData() {
		err := storage.SeedDefaultForeignExchangeRates()
		if err != nil {
			return Conversion{}, err
		}
	}
	return s.fxRates.Register(from, to)
}

// IsVerbose returns if the storage is in verbose mode
func (s *Storage) IsVerbose() bool {
	return s.Verbose
}
