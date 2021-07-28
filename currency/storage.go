package currency

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/currency/coinmarketcap"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/log"
)

func init() {
	storage.SetDefaults()
}

// SetDefaults sets storage defaults for basic package functionality
func (s *Storage) SetDefaults() {
	s.defaultBaseCurrency = USD
	s.baseCurrency = s.defaultBaseCurrency
	var fiatCurrencies []Code
	for item := range symbols {
		if item == USDT.Item {
			continue
		}
		fiatCurrencies = append(fiatCurrencies, Code{Item: item, UpperCase: true})
	}

	err := s.SetDefaultFiatCurrencies(fiatCurrencies...)
	if err != nil {
		log.Errorf(log.Global, "Currency Storage: Setting default fiat currencies error: %s", err)
	}

	err = s.SetDefaultCryptocurrencies(BTC, LTC, ETH, DOGE, DASH, XRP, XMR)
	if err != nil {
		log.Errorf(log.Global, "Currency Storage: Setting default cryptocurrencies error: %s", err)
	}
	s.SetupConversionRates()
	s.fiatExchangeMarkets = forexprovider.NewDefaultFXProvider()
}

// RunUpdater runs the foreign exchange updater service. This will set up a JSON
// dump file and keep foreign exchange rates updated as fast as possible without
// triggering rate limiters, it will also run a full cryptocurrency check
// through coin market cap and expose analytics for exchange services
func (s *Storage) RunUpdater(overrides BotOverrides, settings *MainConfiguration, filePath string) error {
	s.mtx.Lock()
	s.shutdown = make(chan struct{})

	if !settings.Cryptocurrencies.HasData() {
		s.mtx.Unlock()
		return errors.New("currency storage error, no cryptocurrencies loaded")
	}
	s.cryptocurrencies = settings.Cryptocurrencies

	if settings.FiatDisplayCurrency.IsEmpty() {
		s.mtx.Unlock()
		return errors.New("currency storage error, no fiat display currency set in config")
	}
	s.baseCurrency = settings.FiatDisplayCurrency
	log.Debugf(log.Global,
		"Fiat display currency: %s.\n",
		s.baseCurrency)

	if settings.CryptocurrencyProvider.Enabled {
		log.Debugln(log.Global,
			"Setting up currency analysis system with Coinmarketcap...")
		c := &coinmarketcap.Coinmarketcap{}
		c.SetDefaults()
		err := c.Setup(coinmarketcap.Settings{
			Name:        settings.CryptocurrencyProvider.Name,
			Enabled:     settings.CryptocurrencyProvider.Enabled,
			AccountPlan: settings.CryptocurrencyProvider.AccountPlan,
			APIkey:      settings.CryptocurrencyProvider.APIkey,
			Verbose:     settings.CryptocurrencyProvider.Verbose,
		})
		if err != nil {
			log.Errorf(log.Global,
				"Unable to setup CoinMarketCap analysis. Error: %s", err)
			c = nil
			settings.CryptocurrencyProvider.Enabled = false
		} else {
			s.currencyAnalysis = c
		}
	}

	if filePath == "" {
		s.mtx.Unlock()
		return errors.New("currency package runUpdater error filepath not set")
	}

	s.path = filepath.Join(filePath, DefaultStorageFile)

	if settings.CurrencyDelay.Nanoseconds() == 0 {
		s.currencyFileUpdateDelay = DefaultCurrencyFileDelay
	} else {
		s.currencyFileUpdateDelay = settings.CurrencyDelay
	}

	if settings.FxRateDelay.Nanoseconds() == 0 {
		s.foreignExchangeUpdateDelay = DefaultForeignExchangeDelay
	} else {
		s.foreignExchangeUpdateDelay = settings.FxRateDelay
	}

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

		case "ExchangeRateHost":
			if overrides.FxExchangeRateHost || settings.ForexProviders[i].Enabled {
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
			s.mtx.Unlock()
			return err
		}

		log.Debugf(log.Global,
			"Primary foreign exchange conversion provider %s enabled\n",
			s.fiatExchangeMarkets.Primary.Provider.GetName())

		for i := range s.fiatExchangeMarkets.Support {
			log.Debugf(log.Global,
				"Support forex conversion provider %s enabled\n",
				s.fiatExchangeMarkets.Support[i].Provider.GetName())
		}

		// Mutex present in this go routine to lock down retrieving rate data
		// until this system initially updates
		go s.ForeignExchangeUpdater()
	} else {
		log.Warnln(log.Global,
			"No foreign exchange providers enabled in config.json")
		s.mtx.Unlock()
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
func (s *Storage) SetDefaultFiatCurrencies(c ...Code) error {
	for i := range c {
		err := s.currencyCodes.UpdateCurrency("", c[i].String(), "", 0, Fiat)
		if err != nil {
			return err
		}
	}
	s.defaultFiatCurrencies = append(s.defaultFiatCurrencies, c...)
	s.fiatCurrencies = append(s.fiatCurrencies, c...)
	return nil
}

// SetDefaultCryptocurrencies assigns the default cryptocurrency list and adds
// it to the running list
func (s *Storage) SetDefaultCryptocurrencies(c ...Code) error {
	for i := range c {
		err := s.currencyCodes.UpdateCurrency("",
			c[i].String(),
			"",
			0,
			Cryptocurrency)
		if err != nil {
			return err
		}
	}
	s.defaultCryptoCurrencies = append(s.defaultCryptoCurrencies, c...)
	s.cryptocurrencies = append(s.cryptocurrencies, c...)
	return nil
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
	log.Debugln(log.Global,
		"Foreign exchange updater started, seeding FX rate list..")

	s.wg.Add(1)
	defer s.wg.Done()

	err := s.SeedCurrencyAnalysisData()
	if err != nil {
		log.Errorln(log.Global, err)
	}

	err = s.SeedForeignExchangeRates()
	if err != nil {
		log.Errorln(log.Global, err)
	}

	// Unlock main rate retrieval mutex so all routines waiting can get access
	// to data
	s.mtx.Unlock()

	// Set tickers to client defined rates or defaults
	SeedForeignExchangeTick := time.NewTicker(s.foreignExchangeUpdateDelay)
	SeedCurrencyAnalysisTick := time.NewTicker(s.currencyFileUpdateDelay)
	defer SeedForeignExchangeTick.Stop()
	defer SeedCurrencyAnalysisTick.Stop()

	for {
		select {
		case <-s.shutdown:
			return

		case <-SeedForeignExchangeTick.C:
			go func() {
				err := s.SeedForeignExchangeRates()
				if err != nil {
					log.Errorln(log.Global, err)
				}
			}()

		case <-SeedCurrencyAnalysisTick.C:
			go func() {
				err := s.SeedCurrencyAnalysisData()
				if err != nil {
					log.Errorln(log.Global, err)
				}
			}()
		}
	}
}

// SeedCurrencyAnalysisData sets a new instance of a coinmarketcap data.
func (s *Storage) SeedCurrencyAnalysisData() error {
	if s.currencyCodes.LastMainUpdate.IsZero() {
		b, err := ioutil.ReadFile(s.path)
		if err != nil {
			return s.FetchCurrencyAnalysisData()
		}
		var f *File
		err = json.Unmarshal(b, &f)
		if err != nil {
			return err
		}
		err = s.LoadFileCurrencyData(f)
		if err != nil {
			return err
		}
	}

	// Based on update delay update the file
	if time.Now().After(s.currencyCodes.LastMainUpdate.Add(s.currencyFileUpdateDelay)) ||
		s.currencyCodes.LastMainUpdate.IsZero() {
		err := s.FetchCurrencyAnalysisData()
		if err != nil {
			return err
		}
	}

	return nil
}

// FetchCurrencyAnalysisData fetches a new fresh batch of currency data and
// loads it into memory
func (s *Storage) FetchCurrencyAnalysisData() error {
	if s.currencyAnalysis == nil {
		log.Warnln(log.Global,
			"Currency analysis system offline, please set api keys for coinmarketcap if you wish to use this feature.")
		return errors.New("currency analysis system offline")
	}

	return s.UpdateCurrencies()
}

// WriteCurrencyDataToFile writes the full currency data to a designated file
func (s *Storage) WriteCurrencyDataToFile(path string, mainUpdate bool) error {
	data, err := s.currencyCodes.GetFullCurrencyData()
	if err != nil {
		return err
	}

	if mainUpdate {
		t := time.Now()
		data.LastMainUpdate = t.Unix()
		s.currencyCodes.LastMainUpdate = t
	}

	var encoded []byte
	encoded, err = json.MarshalIndent(data, "", " ")
	if err != nil {
		return err
	}

	return file.Write(path, encoded)
}

// LoadFileCurrencyData loads currencies into the currency codes
func (s *Storage) LoadFileCurrencyData(f *File) error {
	for i := range f.Contracts {
		contract := f.Contracts[i]
		contract.Role = Contract
		err := s.currencyCodes.LoadItem(&contract)
		if err != nil {
			return err
		}
	}

	for i := range f.Cryptocurrency {
		crypto := f.Cryptocurrency[i]
		crypto.Role = Cryptocurrency
		err := s.currencyCodes.LoadItem(&crypto)
		if err != nil {
			return err
		}
	}

	for i := range f.Token {
		token := f.Token[i]
		token.Role = Token
		err := s.currencyCodes.LoadItem(&token)
		if err != nil {
			return err
		}
	}

	for i := range f.FiatCurrency {
		fiat := f.FiatCurrency[i]
		fiat.Role = Fiat
		err := s.currencyCodes.LoadItem(&fiat)
		if err != nil {
			return err
		}
	}

	for i := range f.UnsetCurrency {
		unset := f.UnsetCurrency[i]
		unset.Role = Unset
		err := s.currencyCodes.LoadItem(&unset)
		if err != nil {
			return err
		}
	}

	switch t := f.LastMainUpdate.(type) {
	case string:
		parseT, err := time.Parse(time.RFC3339Nano, t)
		if err != nil {
			return err
		}
		s.currencyCodes.LastMainUpdate = parseT
	case float64:
		s.currencyCodes.LastMainUpdate = time.Unix(int64(t), 0)
	default:
		return errors.New("unhandled type conversion for LastMainUpdate time")
	}

	return nil
}

// UpdateCurrencies updates currency role and information using coin market cap
func (s *Storage) UpdateCurrencies() error {
	m, err := s.currencyAnalysis.GetCryptocurrencyIDMap()
	if err != nil {
		return err
	}

	for x := range m {
		if m[x].IsActive != 1 {
			continue
		}

		if m[x].Platform.Symbol != "" {
			err = s.currencyCodes.UpdateCurrency(m[x].Name,
				m[x].Symbol,
				m[x].Platform.Symbol,
				m[x].ID,
				Token)
			if err != nil {
				return err
			}
			continue
		}

		err = s.currencyCodes.UpdateCurrency(m[x].Name,
			m[x].Symbol,
			"",
			m[x].ID,
			Cryptocurrency)
		if err != nil {
			return err
		}
	}
	return nil
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

// GetDefaultForeignExchangeRates returns foreign exchange rates based off
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
	rates, err := s.fiatExchangeMarkets.GetCurrencyData(s.baseCurrency.String(),
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

// SetupCryptoProvider sets congiguration parameters and starts a new instance
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
	for i := range s.defaultFiatCurrencies {
		if s.defaultFiatCurrencies[i].Match(c) ||
			s.defaultFiatCurrencies[i].Match(GetTranslation(c)) {
			return true
		}
	}
	return false
}

// IsDefaultCryptocurrency returns if a cryptocurrency is a default
// cryptocurrency
func (s *Storage) IsDefaultCryptocurrency(c Code) bool {
	for i := range s.defaultCryptoCurrencies {
		if s.defaultCryptoCurrencies[i].Match(c) ||
			s.defaultCryptoCurrencies[i].Match(GetTranslation(c)) {
			return true
		}
	}
	return false
}

// IsFiatCurrency returns if a currency is part of the enabled fiat currency
// list
func (s *Storage) IsFiatCurrency(c Code) bool {
	if c.Item.Role != Unset {
		return c.Item.Role == Fiat
	}

	if c == USDT {
		return false
	}

	for i := range s.fiatCurrencies {
		if s.fiatCurrencies[i].Match(c) ||
			s.fiatCurrencies[i].Match(GetTranslation(c)) {
			return true
		}
	}

	return false
}

// IsCryptocurrency returns if a cryptocurrency is part of the enabled
// cryptocurrency list
func (s *Storage) IsCryptocurrency(c Code) bool {
	if c.Item.Role != Unset {
		return c.Item.Role == Cryptocurrency
	}

	if c == USD {
		return false
	}

	for i := range s.cryptocurrencies {
		if s.cryptocurrencies[i].Match(c) ||
			s.cryptocurrencies[i].Match(GetTranslation(c)) {
			return true
		}
	}

	return false
}

// ValidateCode validates string against currency list and returns a currency
// code
func (s *Storage) ValidateCode(newCode string) Code {
	return s.currencyCodes.Register(newCode)
}

// ValidateFiatCode validates a fiat currency string and returns a currency
// code
func (s *Storage) ValidateFiatCode(newCode string) Code {
	c := s.currencyCodes.RegisterFiat(newCode)
	if !s.fiatCurrencies.Contains(c) {
		s.fiatCurrencies = append(s.fiatCurrencies, c)
	}
	return c
}

// ValidateCryptoCode validates a cryptocurrency string and returns a currency
// code
// TODO: Update and add in RegisterCrypto member func
func (s *Storage) ValidateCryptoCode(newCode string) Code {
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
	for i := range c {
		if !s.cryptocurrencies.Contains(c[i]) {
			s.cryptocurrencies = append(s.cryptocurrencies, c[i])
		}
	}
}

// UpdateEnabledFiatCurrencies appends new fiat currencies to the enabled
// currency list
func (s *Storage) UpdateEnabledFiatCurrencies(c Currencies) {
	for i := range c {
		if !s.fiatCurrencies.Contains(c[i]) &&
			!s.cryptocurrencies.Contains(c[i]) {
			s.fiatCurrencies = append(s.fiatCurrencies, c[i])
		}
	}
}

// ConvertCurrency for example converts $1 USD to the equivalent Japanese Yen
// or vice versa.
func (s *Storage) ConvertCurrency(amount float64, from, to Code) (float64, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

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
	s.mtx.Lock()
	defer s.mtx.Unlock()

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
	s.mtx.Lock()
	defer s.mtx.Unlock()

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

// Shutdown shuts down the currency storage system and saves to currency.json
func (s *Storage) Shutdown() error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	close(s.shutdown)
	s.wg.Wait()
	return s.WriteCurrencyDataToFile(s.path, true)
}
