package currency

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/currency/coinmarketcap"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// storage is an overarching type that keeps track of and updates currency,
// currency exchange rates and pairs
var storage Storage

func init() {
	storage.SetDefaults()
}

// CurrencyFileUpdateDelay defines the rate at which the currency.json file is
// updated
const (
	DefaultCurrencyFileDelay    = 168 * time.Hour
	DefaultForeignExchangeDelay = 1 * time.Minute
	DefaultStorageFile          = "currency.json"
)

var (
	// ErrFiatDisplayCurrencyIsNotFiat defines an error for when the fiat
	// display currency is not set as a fiat currency.
	ErrFiatDisplayCurrencyIsNotFiat = errors.New("fiat display currency is not a fiat currency")

	errUnexpectedRole                       = errors.New("unexpected currency role")
	errFiatDisplayCurrencyUnset             = errors.New("fiat display currency is unset")
	errNoFilePathSet                        = errors.New("no file path set")
	errInvalidCurrencyFileUpdateDuration    = errors.New("invalid currency file update duration")
	errInvalidForeignExchangeUpdateDuration = errors.New("invalid foreign exchange update duration")
	errNoForeignExchangeProvidersEnabled    = errors.New("no foreign exchange providers enabled")
	errNotFiatCurrency                      = errors.New("not a fiat currency")
	errInvalidAmount                        = errors.New("invalid amount")
)

// SetDefaults sets storage defaults for basic package functionality
func (s *Storage) SetDefaults() {
	s.defaultBaseCurrency = USD
	s.baseCurrency = s.defaultBaseCurrency
	fiatCurrencies := make([]Code, 0, len(symbols))
	for item := range symbols {
		if item == USDT.Item {
			continue
		}
		fiatCurrencies = append(fiatCurrencies, Code{Item: item, upperCase: true})
	}

	err := s.SetDefaultFiatCurrencies(fiatCurrencies)
	if err != nil {
		log.Errorf(log.Currency, "Currency Storage: Setting default fiat currencies error: %s", err)
	}

	err = s.SetStableCoins(stables)
	if err != nil {
		log.Errorf(log.Currency, "Currency Storage: Setting default stable currencies error: %s", err)
	}

	err = s.SetDefaultCryptocurrencies(Currencies{BTC, LTC, ETH, DOGE, DASH, XRP, XMR, USDT, UST})
	if err != nil {
		log.Errorf(log.Currency, "Currency Storage: Setting default cryptocurrencies error: %s", err)
	}
	s.SetupConversionRates()
	s.fiatExchangeMarkets = nil
}

// ForexEnabled returns whether the currency system has any available forex providers enabled
func ForexEnabled() bool {
	return storage.fiatExchangeMarkets != nil
}

// RunUpdater runs the foreign exchange updater service. This will set up a JSON
// dump file and keep foreign exchange rates updated as fast as possible without
// triggering rate limiters, it will also run a full cryptocurrency check
// through coin market cap and expose analytics for exchange services
func (s *Storage) RunUpdater(overrides BotOverrides, settings *Config, filePath string) error {
	if settings.FiatDisplayCurrency.IsEmpty() {
		return errFiatDisplayCurrencyUnset
	}

	if !settings.FiatDisplayCurrency.IsFiatCurrency() {
		return fmt.Errorf("%s: %w", settings.FiatDisplayCurrency, ErrFiatDisplayCurrencyIsNotFiat)
	}

	if filePath == "" {
		return errNoFilePathSet
	}

	if settings.CurrencyFileUpdateDuration <= 0 {
		return errInvalidCurrencyFileUpdateDuration
	}

	if settings.ForeignExchangeUpdateDuration <= 0 {
		return errInvalidForeignExchangeUpdateDuration
	}

	s.mtx.Lock()

	// Ensure the forex provider is unset in cases we exit early
	s.fiatExchangeMarkets = nil

	s.shutdown = make(chan struct{})
	s.baseCurrency = settings.FiatDisplayCurrency
	s.path = filepath.Join(filePath, DefaultStorageFile)
	s.currencyFileUpdateDelay = settings.CurrencyFileUpdateDuration
	s.foreignExchangeUpdateDelay = settings.ForeignExchangeUpdateDuration

	log.Debugf(log.Currency, "Fiat display currency: %s.\n", s.baseCurrency)
	var err error
	if overrides.Coinmarketcap {
		if settings.CryptocurrencyProvider.APIKey != "" &&
			settings.CryptocurrencyProvider.APIKey != "Key" {
			log.Debugln(log.Currency, "Setting up currency analysis system with Coinmarketcap...")
			s.currencyAnalysis, err = coinmarketcap.NewFromSettings(coinmarketcap.Settings(settings.CryptocurrencyProvider))
			if err != nil {
				log.Errorf(log.Currency, "Unable to setup CoinMarketCap analysis. Error: %s", err)
			}
		} else {
			log.Warnf(log.Currency, "%s API key not set, disabling. Please set this in your config.json file\n",
				settings.CryptocurrencyProvider.Name)
		}
	}

	fxSettings := make([]base.Settings, 0, len(settings.ForexProviders))
	var primaryProvider bool
	for i := range settings.ForexProviders {
		enabled := (settings.ForexProviders[i].Name == "CurrencyConverter" && overrides.CurrencyConverter) ||
			(settings.ForexProviders[i].Name == "CurrencyLayer" && overrides.CurrencyLayer) ||
			(settings.ForexProviders[i].Name == "Fixer" && overrides.Fixer) ||
			(settings.ForexProviders[i].Name == "OpenExchangeRates" && overrides.OpenExchangeRates) ||
			(settings.ForexProviders[i].Name == "ExchangeRates" && overrides.ExchangeRates)

		if !enabled {
			continue
		}

		if settings.ForexProviders[i].APIKey == "" || settings.ForexProviders[i].APIKey == "Key" {
			log.Warnf(log.Currency, "%s forex provider API key not set, disabling. Please set this in your config.json file\n",
				settings.ForexProviders[i].Name)
			settings.ForexProviders[i].Enabled = false
			settings.ForexProviders[i].PrimaryProvider = false
			continue
		}

		if settings.ForexProviders[i].APIKeyLvl == -1 && settings.ForexProviders[i].Name != "ExchangeRates" {
			log.Warnf(log.Currency, "%s APIKey level not set, functionality is limited. Please review this in your config.json file\n",
				settings.ForexProviders[i].Name)
		}

		if settings.ForexProviders[i].PrimaryProvider {
			if primaryProvider {
				log.Warnf(log.Currency, "%s disabling primary provider, multiple primarys found. Please review providers in your config.json file\n",
					settings.ForexProviders[i].Name)
				settings.ForexProviders[i].PrimaryProvider = false
			} else {
				primaryProvider = true
			}
		}
		fxSettings = append(fxSettings, base.Settings(settings.ForexProviders[i]))
	}

	if len(fxSettings) == 0 {
		s.mtx.Unlock()
		log.Warnln(log.Currency, "No foreign exchange providers enabled, currency conversion will not be available")
		return nil
	}

	if !primaryProvider {
		for x := range settings.ForexProviders {
			if settings.ForexProviders[x].Name == fxSettings[0].Name {
				settings.ForexProviders[x].PrimaryProvider = true
				fxSettings[0].PrimaryProvider = true
				log.Warnf(log.Currency, "No primary foreign exchange provider set. Defaulting to %s.", fxSettings[0].Name)
				break
			}
		}
	}

	s.fiatExchangeMarkets, err = forexprovider.StartFXService(fxSettings)
	if err != nil {
		s.mtx.Unlock()
		return err
	}

	log.Debugf(log.Currency, "Using primary foreign exchange provider %s\n",
		s.fiatExchangeMarkets.Primary.Provider.GetName())

	for i := range s.fiatExchangeMarkets.Support {
		log.Debugf(log.Currency, "Supporting foreign exchange provider %s\n",
			s.fiatExchangeMarkets.Support[i].Provider.GetName())
	}

	// Mutex present in this go routine to lock down retrieving rate data
	// until this system initially updates
	s.wg.Add(1)
	go s.ForeignExchangeUpdater()
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
func (s *Storage) SetDefaultFiatCurrencies(c Currencies) error {
	for i := range c {
		err := s.currencyCodes.UpdateCurrency(&Item{
			ID:         c[i].Item.ID,
			FullName:   c[i].Item.FullName,
			Symbol:     c[i].Item.Symbol,
			Lower:      c[i].Item.Lower,
			Role:       Fiat,
			AssocChain: c[i].Item.AssocChain,
		})
		if err != nil {
			return err
		}
	}
	s.defaultFiatCurrencies = append(s.defaultFiatCurrencies, c...)
	s.fiatCurrencies = append(s.fiatCurrencies, c...)
	return nil
}

// SetStableCoins assigns the stable currency list and adds it to the running
// list
func (s *Storage) SetStableCoins(c Currencies) error {
	for i := range c {
		err := s.currencyCodes.UpdateCurrency(&Item{
			ID:         c[i].Item.ID,
			FullName:   c[i].Item.FullName,
			Symbol:     c[i].Item.Symbol,
			Lower:      c[i].Item.Lower,
			Role:       Stable,
			AssocChain: c[i].Item.AssocChain,
		})
		if err != nil {
			return err
		}
	}
	s.stableCurrencies = append(s.stableCurrencies, c...)
	return nil
}

// SetDefaultCryptocurrencies assigns the default cryptocurrency list and adds
// it to the running list
func (s *Storage) SetDefaultCryptocurrencies(c Currencies) error {
	for i := range c {
		err := s.currencyCodes.UpdateCurrency(&Item{
			ID:         c[i].Item.ID,
			FullName:   c[i].Item.FullName,
			Symbol:     c[i].Item.Symbol,
			Lower:      c[i].Item.Lower,
			Role:       Cryptocurrency,
			AssocChain: c[i].Item.AssocChain,
		})
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
	defer s.wg.Done()
	log.Debugln(log.Currency, "Foreign exchange updater started, seeding FX rate list...")

	err := s.SeedCurrencyAnalysisData()
	if err != nil {
		log.Errorln(log.Currency, err)
	}

	err = s.SeedForeignExchangeRates()
	if err != nil {
		log.Errorln(log.Currency, err)
	}

	// Unlock main rate retrieval mutex so all routines waiting can get access to data
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
					log.Errorln(log.Currency, err)
				}
			}()
		case <-SeedCurrencyAnalysisTick.C:
			go func() {
				err := s.SeedCurrencyAnalysisData()
				if err != nil {
					log.Errorln(log.Currency, err)
				}
			}()
		}
	}
}

// SeedCurrencyAnalysisData sets a new instance of a coinmarketcap data.
func (s *Storage) SeedCurrencyAnalysisData() error {
	if s.currencyCodes.LastMainUpdate.IsZero() {
		b, err := os.ReadFile(s.path)
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
		log.Warnln(log.Currency,
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

func (s *Storage) checkFileCurrencyData(item *Item, role Role) error {
	if item.Role == Unset {
		item.Role = role
	}
	if item.Role != role {
		return fmt.Errorf("%w %s expecting: %s", errUnexpectedRole, item.Role, role)
	}
	return s.currencyCodes.LoadItem(item)
}

// LoadFileCurrencyData loads currencies into the currency codes
func (s *Storage) LoadFileCurrencyData(f *File) error {
	for i := range f.Contracts {
		err := s.checkFileCurrencyData(f.Contracts[i], Contract)
		if err != nil {
			return err
		}
	}

	for i := range f.Cryptocurrency {
		err := s.checkFileCurrencyData(f.Cryptocurrency[i], Cryptocurrency)
		if err != nil {
			return err
		}
	}

	for i := range f.Token {
		err := s.checkFileCurrencyData(f.Token[i], Token)
		if err != nil {
			return err
		}
	}

	for i := range f.FiatCurrency {
		err := s.checkFileCurrencyData(f.FiatCurrency[i], Fiat)
		if err != nil {
			return err
		}
	}

	for i := range f.UnsetCurrency {
		err := s.checkFileCurrencyData(f.UnsetCurrency[i], Unset)
		if err != nil {
			return err
		}
	}

	for i := range f.Stable {
		err := s.checkFileCurrencyData(f.Stable[i], Stable)
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
	currencyUpdates, err := s.currencyAnalysis.GetCryptocurrencyIDMap()
	if err != nil {
		return err
	}

	for x := range currencyUpdates {
		if currencyUpdates[x].IsActive != 1 {
			continue
		}

		update := &Item{
			FullName:   currencyUpdates[x].Name,
			Symbol:     currencyUpdates[x].Symbol,
			AssocChain: currencyUpdates[x].Platform.Symbol,
			ID:         currencyUpdates[x].ID,
			Role:       Cryptocurrency,
		}

		if currencyUpdates[x].Platform.Symbol != "" {
			update.Role = Token
		}

		err = s.currencyCodes.UpdateCurrency(update)
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
	if s.fiatExchangeMarkets == nil {
		return nil
	}
	rates, err := s.fiatExchangeMarkets.GetCurrencyData(s.baseCurrency.String(),
		c.Strings())
	if err != nil {
		return err
	}
	return s.updateExchangeRates(rates)
}

// SeedForeignExchangeRate returns a singular exchange rate
func (s *Storage) SeedForeignExchangeRate(from, to Code) (map[string]float64, error) {
	if s.fiatExchangeMarkets == nil {
		return nil, nil
	}
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
	if s.fiatExchangeMarkets == nil {
		return errNoForeignExchangeProvidersEnabled
	}
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

// SeedForeignExchangeRates seeds the foreign exchange rates from storage config currencies
func (s *Storage) SeedForeignExchangeRates() error {
	s.fxRates.mtx.Lock()
	defer s.fxRates.mtx.Unlock()
	if s.fiatExchangeMarkets == nil {
		return errNoForeignExchangeProvidersEnabled
	}
	rates, err := s.fiatExchangeMarkets.GetCurrencyData(s.baseCurrency.String(),
		s.fiatCurrencies.Strings())
	if err != nil {
		return err
	}
	return s.updateExchangeRates(rates)
}

func (s *Storage) updateExchangeRates(m map[string]float64) error {
	return s.fxRates.Update(m)
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
		if s.defaultFiatCurrencies[i].Equal(c) ||
			s.defaultFiatCurrencies[i].Equal(GetTranslation(c)) {
			return true
		}
	}
	return false
}

// IsDefaultCryptocurrency returns if a cryptocurrency is a default
// cryptocurrency
func (s *Storage) IsDefaultCryptocurrency(c Code) bool {
	for i := range s.defaultCryptoCurrencies {
		if s.defaultCryptoCurrencies[i].Equal(c) ||
			s.defaultCryptoCurrencies[i].Equal(GetTranslation(c)) {
			return true
		}
	}
	return false
}

// ValidateCode validates string against currency list and returns a currency
// code
func (s *Storage) ValidateCode(newCode string) Code {
	return s.currencyCodes.Register(newCode, Unset)
}

// ValidateFiatCode validates a fiat currency string and returns a currency
// code
func (s *Storage) ValidateFiatCode(newCode string) Code {
	c := s.currencyCodes.Register(newCode, Fiat)
	if !s.fiatCurrencies.Contains(c) {
		s.fiatCurrencies = append(s.fiatCurrencies, c)
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
	if s.fiatExchangeMarkets == nil {
		return 0, errNoForeignExchangeProvidersEnabled
	}
	if amount <= 0 {
		return 0, fmt.Errorf("%f %w", amount, errInvalidAmount)
	}
	if !from.IsFiatCurrency() {
		return 0, fmt.Errorf("%s %w", from, errNotFiatCurrency)
	}
	if !to.IsFiatCurrency() {
		return 0, fmt.Errorf("%s %w", to, errNotFiatCurrency)
	}

	if from.Equal(to) { // No need to lock down storage for this rate.
		return amount, nil
	}

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
	if s.shutdown == nil {
		return nil
	}
	close(s.shutdown)
	s.wg.Wait()
	s.shutdown = nil
	return s.WriteCurrencyDataToFile(s.path, true)
}
