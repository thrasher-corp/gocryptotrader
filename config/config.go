package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/connchecker"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/base"
	log "github.com/thrasher-/gocryptotrader/logger"
	"github.com/thrasher-/gocryptotrader/portfolio"
)

// Constants declared here are filename strings and test strings
const (
	FXProviderFixer                        = "fixer"
	EncryptedConfigFile                    = "config.dat"
	ConfigFile                             = "config.json"
	ConfigTestFile                         = "../testdata/configtest.json"
	configFileEncryptionPrompt             = 0
	configFileEncryptionEnabled            = 1
	configFileEncryptionDisabled           = -1
	configPairsLastUpdatedWarningThreshold = 30 // 30 days
	configDefaultHTTPTimeout               = time.Second * 15
	configMaxAuthFailres                   = 3
	defaultNTPAllowedDifference            = 50000000
	defaultNTPAllowedNegativeDifference    = 50000000
)

// Constants here hold some messages
const (
	ErrExchangeNameEmpty                       = "exchange #%d name is empty"
	ErrExchangeAvailablePairsEmpty             = "exchange %s avaiable pairs is emtpy"
	ErrExchangeEnabledPairsEmpty               = "exchange %s enabled pairs is empty"
	ErrExchangeBaseCurrenciesEmpty             = "exchange %s base currencies is empty"
	ErrExchangeNotFound                        = "exchange %s not found"
	ErrNoEnabledExchanges                      = "no exchanges enabled"
	ErrCryptocurrenciesEmpty                   = "cryptocurrencies variable is empty"
	ErrFailureOpeningConfig                    = "fatal error opening %s file. Error: %s"
	ErrCheckingConfigValues                    = "fatal error checking config values. Error: %s"
	ErrSavingConfigBytesMismatch               = "config file %q bytes comparison doesn't match, read %s expected %s"
	WarningWebserverCredentialValuesEmpty      = "webserver support disabled due to empty Username/Password values"
	WarningWebserverListenAddressInvalid       = "webserver support disabled due to invalid listen address"
	WarningExchangeAuthAPIDefaultOrEmptyValues = "exchange %s authenticated API support disabled due to default/empty APIKey/Secret/ClientID values"
	WarningPairsLastUpdatedThresholdExceeded   = "exchange %s last manual update of available currency pairs has exceeded %d days. Manual update required!"
)

// Constants here define unset default values displayed in the config.json
// file
const (
	APIURLNonDefaultMessage              = "NON_DEFAULT_HTTP_LINK_TO_EXCHANGE_API"
	WebsocketURLNonDefaultMessage        = "NON_DEFAULT_HTTP_LINK_TO_WEBSOCKET_EXCHANGE_API"
	DefaultUnsetAPIKey                   = "Key"
	DefaultUnsetAPISecret                = "Secret"
	DefaultUnsetAccountPlan              = "accountPlan"
	DefaultForexProviderExchangeRatesAPI = "ExchangeRates"
)

// Variables here are used for configuration
var (
	Cfg            Config
	IsInitialSetup bool
	testBypass     bool
	m              sync.Mutex
)

// WebserverConfig struct holds the prestart variables for the webserver.
type WebserverConfig struct {
	Enabled                      bool   `json:"enabled"`
	AdminUsername                string `json:"adminUsername"`
	AdminPassword                string `json:"adminPassword"`
	ListenAddress                string `json:"listenAddress"`
	WebsocketConnectionLimit     int    `json:"websocketConnectionLimit"`
	WebsocketMaxAuthFailures     int    `json:"websocketMaxAuthFailures"`
	WebsocketAllowInsecureOrigin bool   `json:"websocketAllowInsecureOrigin"`
}

// Post holds the bot configuration data
type Post struct {
	Data Config `json:"data"`
}

// CurrencyPairFormatConfig stores the users preferred currency pair display
type CurrencyPairFormatConfig struct {
	Uppercase bool   `json:"uppercase"`
	Delimiter string `json:"delimiter,omitempty"`
	Separator string `json:"separator,omitempty"`
	Index     string `json:"index,omitempty"`
}

// Config is the overarching object that holds all the information for
// prestart management of Portfolio, Communications, Webserver and Enabled
// Exchanges
type Config struct {
	Name              string                  `json:"name"`
	EncryptConfig     int                     `json:"encryptConfig"`
	GlobalHTTPTimeout time.Duration           `json:"globalHTTPTimeout"`
	Logging           log.Logging             `json:"logging"`
	Profiler          ProfilerConfig          `json:"profiler"`
	NTPClient         NTPClientConfig         `json:"ntpclient"`
	Currency          CurrencyConfig          `json:"currencyConfig"`
	Communications    CommunicationsConfig    `json:"communications"`
	Portfolio         portfolio.Base          `json:"portfolioAddresses"`
	Webserver         WebserverConfig         `json:"webserver"`
	Exchanges         []ExchangeConfig        `json:"exchanges"`
	BankAccounts      []BankAccount           `json:"bankAccounts"`
	ConnectionMonitor ConnectionMonitorConfig `json:"connectionMonitor"`

	// Deprecated config settings, will be removed at a future date
	CurrencyPairFormat  *CurrencyPairFormatConfig `json:"currencyPairFormat,omitempty"`
	FiatDisplayCurrency currency.Code             `json:"fiatDispayCurrency,omitempty"`
	Cryptocurrencies    currency.Currencies       `json:"cryptocurrencies,omitempty"`
	SMS                 *SMSGlobalConfig          `json:"smsGlobal,omitempty"`
}

// ConnectionMonitorConfig defines the connection monitor variables to ensure
// that there is internet connectivity
type ConnectionMonitorConfig struct {
	DNSList          []string      `json:"preferredDNSList"`
	PublicDomainList []string      `json:"preferredDomainList"`
	CheckInterval    time.Duration `json:"checkInterval"`
}

// ProfilerConfig defines the profiler configuration to enable pprof
type ProfilerConfig struct {
	Enabled bool `json:"enabled"`
}

// NTPClientConfig defines a network time protocol configuration to allow for
// positive and negative differences
type NTPClientConfig struct {
	Level                     int            `json:"enabled"`
	Pool                      []string       `json:"pool"`
	AllowedDifference         *time.Duration `json:"allowedDifference"`
	AllowedNegativeDifference *time.Duration `json:"allowedNegativeDifference"`
}

// ExchangeConfig holds all the information needed for each enabled Exchange.
type ExchangeConfig struct {
	Name                      string                    `json:"name"`
	Enabled                   bool                      `json:"enabled"`
	Verbose                   bool                      `json:"verbose"`
	Websocket                 bool                      `json:"websocket"`
	UseSandbox                bool                      `json:"useSandbox"`
	RESTPollingDelay          time.Duration             `json:"restPollingDelay"`
	HTTPTimeout               time.Duration             `json:"httpTimeout"`
	HTTPUserAgent             string                    `json:"httpUserAgent"`
	AuthenticatedAPISupport   bool                      `json:"authenticatedApiSupport"`
	APIKey                    string                    `json:"apiKey"`
	APISecret                 string                    `json:"apiSecret"`
	APIAuthPEMKeySupport      bool                      `json:"apiAuthPemKeySupport,omitempty"`
	APIAuthPEMKey             string                    `json:"apiAuthPemKey,omitempty"`
	APIURL                    string                    `json:"apiUrl"`
	APIURLSecondary           string                    `json:"apiUrlSecondary"`
	ProxyAddress              string                    `json:"proxyAddress"`
	WebsocketURL              string                    `json:"websocketUrl"`
	ClientID                  string                    `json:"clientId,omitempty"`
	AvailablePairs            currency.Pairs            `json:"availablePairs"`
	EnabledPairs              currency.Pairs            `json:"enabledPairs"`
	BaseCurrencies            currency.Currencies       `json:"baseCurrencies"`
	AssetTypes                string                    `json:"assetTypes"`
	SupportsAutoPairUpdates   bool                      `json:"supportsAutoPairUpdates"`
	PairsLastUpdated          int64                     `json:"pairsLastUpdated,omitempty"`
	ConfigCurrencyPairFormat  *CurrencyPairFormatConfig `json:"configCurrencyPairFormat"`
	RequestCurrencyPairFormat *CurrencyPairFormatConfig `json:"requestCurrencyPairFormat"`
	BankAccounts              []BankAccount             `json:"bankAccounts"`
}

// BankAccount holds differing bank account details by supported funding
// currency
type BankAccount struct {
	Enabled             bool   `json:"enabled,omitempty"`
	BankName            string `json:"bankName"`
	BankAddress         string `json:"bankAddress"`
	AccountName         string `json:"accountName"`
	AccountNumber       string `json:"accountNumber"`
	SWIFTCode           string `json:"swiftCode"`
	IBAN                string `json:"iban"`
	BSBNumber           string `json:"bsbNumber,omitempty"`
	SupportedCurrencies string `json:"supportedCurrencies"`
	SupportedExchanges  string `json:"supportedExchanges,omitempty"`
}

// BankTransaction defines a related banking transaction
type BankTransaction struct {
	ReferenceNumber     string `json:"referenceNumber"`
	TransactionNumber   string `json:"transactionNumber"`
	PaymentInstructions string `json:"paymentInstructions"`
}

// CurrencyConfig holds all the information needed for currency related manipulation
type CurrencyConfig struct {
	ForexProviders                []base.Settings           `json:"forexProviders"`
	CryptocurrencyProvider        CryptocurrencyProvider    `json:"cryptocurrencyProvider"`
	Cryptocurrencies              currency.Currencies       `json:"cryptocurrencies"`
	CurrencyPairFormat            *CurrencyPairFormatConfig `json:"currencyPairFormat"`
	FiatDisplayCurrency           currency.Code             `json:"fiatDisplayCurrency"`
	CurrencyFileUpdateDuration    time.Duration             `json:"currencyFileUpdateDuration"`
	ForeignExchangeUpdateDuration time.Duration             `json:"foreignExchangeUpdateDuration"`
}

// CryptocurrencyProvider defines coinmarketcap tools
type CryptocurrencyProvider struct {
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	Verbose     bool   `json:"verbose"`
	APIkey      string `json:"apiKey"`
	AccountPlan string `json:"accountPlan"`
}

// CommunicationsConfig holds all the information needed for each
// enabled communication package
type CommunicationsConfig struct {
	SlackConfig     SlackConfig     `json:"slack"`
	SMSGlobalConfig SMSGlobalConfig `json:"smsGlobal"`
	SMTPConfig      SMTPConfig      `json:"smtp"`
	TelegramConfig  TelegramConfig  `json:"telegram"`
}

// SlackConfig holds all variables to start and run the Slack package
type SlackConfig struct {
	Name              string `json:"name"`
	Enabled           bool   `json:"enabled"`
	Verbose           bool   `json:"verbose"`
	TargetChannel     string `json:"targetChannel"`
	VerificationToken string `json:"verificationToken"`
}

// SMSContact stores the SMS contact info
type SMSContact struct {
	Name    string `json:"name"`
	Number  string `json:"number"`
	Enabled bool   `json:"enabled"`
}

// SMSGlobalConfig structure holds all the variables you need for instant
// messaging and broadcast used by SMSGlobal
type SMSGlobalConfig struct {
	Name     string       `json:"name"`
	Enabled  bool         `json:"enabled"`
	Verbose  bool         `json:"verbose"`
	Username string       `json:"username"`
	Password string       `json:"password"`
	Contacts []SMSContact `json:"contacts"`
}

// SMTPConfig holds all variables to start and run the SMTP package
type SMTPConfig struct {
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	Verbose         bool   `json:"verbose"`
	Host            string `json:"host"`
	Port            string `json:"port"`
	AccountName     string `json:"accountName"`
	AccountPassword string `json:"accountPassword"`
	RecipientList   string `json:"recipientList"`
}

// TelegramConfig holds all variables to start and run the Telegram package
type TelegramConfig struct {
	Name              string `json:"name"`
	Enabled           bool   `json:"enabled"`
	Verbose           bool   `json:"verbose"`
	VerificationToken string `json:"verificationToken"`
}

// GetCurrencyConfig returns currency configurations
func (c *Config) GetCurrencyConfig() CurrencyConfig {
	return c.Currency
}

// GetExchangeBankAccounts returns banking details associated with an exchange
// for depositing funds
func (c *Config) GetExchangeBankAccounts(exchangeName, depositingCurrency string) (BankAccount, error) {
	m.Lock()
	defer m.Unlock()

	for x := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[x].Name, exchangeName) {
			for y := range c.Exchanges[x].BankAccounts {
				if common.StringContains(c.Exchanges[x].BankAccounts[y].SupportedCurrencies,
					depositingCurrency) {
					return c.Exchanges[x].BankAccounts[y], nil
				}
			}
		}
	}
	return BankAccount{}, fmt.Errorf("exchange %s bank details not found for %s",
		exchangeName,
		depositingCurrency)
}

// UpdateExchangeBankAccounts updates the configuration for the associated
// exchange bank
func (c *Config) UpdateExchangeBankAccounts(exchangeName string, bankCfg []BankAccount) error {
	m.Lock()
	defer m.Unlock()

	for i := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[i].Name, exchangeName) {
			c.Exchanges[i].BankAccounts = bankCfg
			return nil
		}
	}
	return fmt.Errorf("exchange %s not found",
		exchangeName)
}

// GetClientBankAccounts returns banking details used for a given exchange
// and currency
func (c *Config) GetClientBankAccounts(exchangeName, targetCurrency string) (BankAccount, error) {
	m.Lock()
	defer m.Unlock()

	for x := range c.BankAccounts {
		if (common.StringContains(c.BankAccounts[x].SupportedExchanges, exchangeName) ||
			c.BankAccounts[x].SupportedExchanges == "ALL") &&
			common.StringContains(c.BankAccounts[x].SupportedCurrencies, targetCurrency) {
			return c.BankAccounts[x], nil

		}
	}
	return BankAccount{}, fmt.Errorf("client banking details not found for %s and currency %s",
		exchangeName,
		targetCurrency)
}

// UpdateClientBankAccounts updates the configuration for a bank
func (c *Config) UpdateClientBankAccounts(bankCfg *BankAccount) error {
	m.Lock()
	defer m.Unlock()

	for i := range c.BankAccounts {
		if c.BankAccounts[i].BankName == bankCfg.BankName && c.BankAccounts[i].AccountNumber == bankCfg.AccountNumber {
			c.BankAccounts[i] = *bankCfg
			return nil
		}
	}
	return fmt.Errorf("client banking details for %s not found, update not applied",
		bankCfg.BankName)
}

// CheckClientBankAccounts checks client bank details
func (c *Config) CheckClientBankAccounts() error {
	m.Lock()
	defer m.Unlock()

	if len(c.BankAccounts) == 0 {
		c.BankAccounts = append(c.BankAccounts,
			BankAccount{
				BankName:            "test",
				BankAddress:         "test",
				AccountName:         "TestAccount",
				AccountNumber:       "0234",
				SWIFTCode:           "91272837",
				IBAN:                "98218738671897",
				SupportedCurrencies: "USD",
				SupportedExchanges:  "ANX,Kraken",
			},
		)
		return nil
	}

	for i := range c.BankAccounts {
		if c.BankAccounts[i].Enabled {
			if c.BankAccounts[i].BankName == "" || c.BankAccounts[i].BankAddress == "" {
				return fmt.Errorf("banking details for %s is enabled but variables not set correctly",
					c.BankAccounts[i].BankName)
			}

			if c.BankAccounts[i].AccountName == "" || c.BankAccounts[i].AccountNumber == "" {
				return fmt.Errorf("banking account details for %s variables not set correctly",
					c.BankAccounts[i].BankName)
			}
			if c.BankAccounts[i].IBAN == "" && c.BankAccounts[i].SWIFTCode == "" && c.BankAccounts[i].BSBNumber == "" {
				return fmt.Errorf("critical banking numbers not set for %s in %s account",
					c.BankAccounts[i].BankName,
					c.BankAccounts[i].AccountName)
			}

			if c.BankAccounts[i].SupportedExchanges == "" {
				c.BankAccounts[i].SupportedExchanges = "ALL"
			}
		}
	}
	return nil
}

// GetCommunicationsConfig returns the communications configuration
func (c *Config) GetCommunicationsConfig() CommunicationsConfig {
	m.Lock()
	defer m.Unlock()
	return c.Communications
}

// UpdateCommunicationsConfig sets a new updated version of a Communications
// configuration
func (c *Config) UpdateCommunicationsConfig(config *CommunicationsConfig) {
	m.Lock()
	c.Communications = *config
	m.Unlock()
}

// GetCryptocurrencyProviderConfig returns the communications configuration
func (c *Config) GetCryptocurrencyProviderConfig() CryptocurrencyProvider {
	m.Lock()
	defer m.Unlock()
	return c.Currency.CryptocurrencyProvider
}

// UpdateCryptocurrencyProviderConfig returns the communications configuration
func (c *Config) UpdateCryptocurrencyProviderConfig(config CryptocurrencyProvider) {
	m.Lock()
	c.Currency.CryptocurrencyProvider = config
	m.Unlock()
}

// CheckCommunicationsConfig checks to see if the variables are set correctly
// from config.json
func (c *Config) CheckCommunicationsConfig() {
	m.Lock()
	defer m.Unlock()

	// If the communications config hasn't been populated, populate
	// with example settings

	if c.Communications.SlackConfig.Name == "" {
		c.Communications.SlackConfig = SlackConfig{
			Name:              "Slack",
			TargetChannel:     "general",
			VerificationToken: "testtest",
		}
	}

	if c.Communications.SMSGlobalConfig.Name == "" {
		if c.SMS != nil {
			if c.SMS.Contacts != nil {
				c.Communications.SMSGlobalConfig = SMSGlobalConfig{
					Name:     "SMSGlobal",
					Enabled:  c.SMS.Enabled,
					Verbose:  c.SMS.Verbose,
					Username: c.SMS.Username,
					Password: c.SMS.Password,
					Contacts: c.SMS.Contacts,
				}
				// flush old SMS config
				c.SMS = nil
			} else {
				c.Communications.SMSGlobalConfig = SMSGlobalConfig{
					Name:     "SMSGlobal",
					Username: "main",
					Password: "test",

					Contacts: []SMSContact{
						{
							Name:    "bob",
							Number:  "1234",
							Enabled: false,
						},
					},
				}
			}
		} else {
			c.Communications.SMSGlobalConfig = SMSGlobalConfig{
				Name:     "SMSGlobal",
				Username: "main",
				Password: "test",

				Contacts: []SMSContact{
					{
						Name:    "bob",
						Number:  "1234",
						Enabled: false,
					},
				},
			}
		}

	} else {
		if c.SMS != nil {
			// flush old SMS config
			c.SMS = nil
		}
	}

	if c.Communications.SMTPConfig.Name == "" {
		c.Communications.SMTPConfig = SMTPConfig{
			Name:            "SMTP",
			Host:            "smtp.google.com",
			Port:            "537",
			AccountName:     "some",
			AccountPassword: "password",
			RecipientList:   "lol123@gmail.com",
		}
	}

	if c.Communications.TelegramConfig.Name == "" {
		c.Communications.TelegramConfig = TelegramConfig{
			Name:              "Telegram",
			VerificationToken: "testest",
		}
	}

	if c.Communications.SlackConfig.Name != "Slack" ||
		c.Communications.SMSGlobalConfig.Name != "SMSGlobal" ||
		c.Communications.SMTPConfig.Name != "SMTP" ||
		c.Communications.TelegramConfig.Name != "Telegram" {
		log.Warn("Communications config name/s not set correctly")
	}
	if c.Communications.SlackConfig.Enabled {
		if c.Communications.SlackConfig.TargetChannel == "" ||
			c.Communications.SlackConfig.VerificationToken == "" ||
			c.Communications.SlackConfig.VerificationToken == "testtest" {
			c.Communications.SlackConfig.Enabled = false
			log.Warn("Slack enabled in config but variable data not set, disabling.")
		}
	}
	if c.Communications.SMSGlobalConfig.Enabled {
		if c.Communications.SMSGlobalConfig.Username == "" ||
			c.Communications.SMSGlobalConfig.Password == "" ||
			len(c.Communications.SMSGlobalConfig.Contacts) == 0 {
			c.Communications.SMSGlobalConfig.Enabled = false
			log.Warn("SMSGlobal enabled in config but variable data not set, disabling.")
		}
	}
	if c.Communications.SMTPConfig.Enabled {
		if c.Communications.SMTPConfig.Host == "" ||
			c.Communications.SMTPConfig.Port == "" ||
			c.Communications.SMTPConfig.AccountName == "" ||
			c.Communications.SMTPConfig.AccountPassword == "" {
			c.Communications.SMTPConfig.Enabled = false
			log.Warn("SMTP enabled in config but variable data not set, disabling.")
		}
	}
	if c.Communications.TelegramConfig.Enabled {
		if c.Communications.TelegramConfig.VerificationToken == "" {
			c.Communications.TelegramConfig.Enabled = false
			log.Warn("Telegram enabled in config but variable data not set, disabling.")
		}
	}
}

// CheckPairConsistency checks to see if the enabled pair exists in the
// available pairs list
func (c *Config) CheckPairConsistency(exchName string) error {
	enabledPairs, err := c.GetEnabledPairs(exchName)
	if err != nil {
		return err
	}

	availPairs, err := c.GetAvailablePairs(exchName)
	if err != nil {
		return err
	}

	var pairs, pairsRemoved currency.Pairs
	update := false
	for x := range enabledPairs {
		if !availPairs.Contains(enabledPairs[x], true) {
			update = true
			pairsRemoved = append(pairsRemoved, enabledPairs[x])
			continue
		}
		pairs = append(pairs, enabledPairs[x])
	}

	if !update {
		return nil
	}

	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return err
	}

	if len(pairs) == 0 {
		exchCfg.EnabledPairs = append(exchCfg.EnabledPairs,
			availPairs.GetRandomPair())
		log.Debugf("Exchange %s: No enabled pairs found in available pairs, randomly added %v\n",
			exchName,
			exchCfg.EnabledPairs)
	} else {
		exchCfg.EnabledPairs = pairs
	}

	err = c.UpdateExchangeConfig(&exchCfg)
	if err != nil {
		return err
	}

	log.Debugf("Exchange %s: Removing enabled pair(s) %v from enabled pairs as it isn't an available pair",
		exchName,
		pairsRemoved.Strings())
	return nil
}

// SupportsPair returns true or not whether the exchange supports the supplied
// pair
func (c *Config) SupportsPair(exchName string, p currency.Pair) (bool, error) {
	pairs, err := c.GetAvailablePairs(exchName)
	if err != nil {
		return false, err
	}
	return pairs.Contains(p, false), nil
}

// GetAvailablePairs returns a list of currency pairs for a specifc exchange
func (c *Config) GetAvailablePairs(exchName string) (currency.Pairs, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	return exchCfg.AvailablePairs.Format(exchCfg.ConfigCurrencyPairFormat.Delimiter,
		exchCfg.ConfigCurrencyPairFormat.Index,
		exchCfg.ConfigCurrencyPairFormat.Uppercase), nil
}

// GetEnabledPairs returns a list of currency pairs for a specifc exchange
func (c *Config) GetEnabledPairs(exchName string) (currency.Pairs, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	return exchCfg.EnabledPairs.Format(exchCfg.ConfigCurrencyPairFormat.Delimiter,
		exchCfg.ConfigCurrencyPairFormat.Index,
		exchCfg.ConfigCurrencyPairFormat.Uppercase), nil
}

// GetEnabledExchanges returns a list of enabled exchanges
func (c *Config) GetEnabledExchanges() []string {
	var enabledExchs []string
	for i := range c.Exchanges {
		if c.Exchanges[i].Enabled {
			enabledExchs = append(enabledExchs, c.Exchanges[i].Name)
		}
	}
	return enabledExchs
}

// GetDisabledExchanges returns a list of disabled exchanges
func (c *Config) GetDisabledExchanges() []string {
	var disabledExchs []string
	for i := range c.Exchanges {
		if !c.Exchanges[i].Enabled {
			disabledExchs = append(disabledExchs, c.Exchanges[i].Name)
		}
	}
	return disabledExchs
}

// CountEnabledExchanges returns the number of exchanges that are enabled.
func (c *Config) CountEnabledExchanges() int {
	counter := 0
	for i := range c.Exchanges {
		if c.Exchanges[i].Enabled {
			counter++
		}
	}
	return counter
}

// GetConfigCurrencyPairFormat returns the config currency pair format
// for a specific exchange
func (c *Config) GetConfigCurrencyPairFormat(exchName string) (*CurrencyPairFormatConfig, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}
	return exchCfg.ConfigCurrencyPairFormat, nil
}

// GetRequestCurrencyPairFormat returns the request currency pair format
// for a specific exchange
func (c *Config) GetRequestCurrencyPairFormat(exchName string) (*CurrencyPairFormatConfig, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}
	return exchCfg.RequestCurrencyPairFormat, nil
}

// GetCurrencyPairDisplayConfig retrieves the currency pair display preference
func (c *Config) GetCurrencyPairDisplayConfig() *CurrencyPairFormatConfig {
	return c.Currency.CurrencyPairFormat
}

// GetAllExchangeConfigs returns all exchange configurations
func (c *Config) GetAllExchangeConfigs() []ExchangeConfig {
	m.Lock()
	defer m.Unlock()
	return c.Exchanges
}

// GetExchangeConfig returns exchange configurations by its indivdual name
func (c *Config) GetExchangeConfig(name string) (ExchangeConfig, error) {
	m.Lock()
	defer m.Unlock()
	for i := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[i].Name, name) {
			return c.Exchanges[i], nil
		}
	}
	return ExchangeConfig{}, fmt.Errorf(ErrExchangeNotFound, name)
}

// GetForexProviderConfig returns a forex provider configuration by its name
func (c *Config) GetForexProviderConfig(name string) (base.Settings, error) {
	m.Lock()
	defer m.Unlock()
	for i := range c.Currency.ForexProviders {
		if strings.EqualFold(c.Currency.ForexProviders[i].Name, name) {
			return c.Currency.ForexProviders[i], nil
		}
	}
	return base.Settings{}, errors.New("provider not found")
}

// GetPrimaryForexProvider returns the primary forex provider
func (c *Config) GetPrimaryForexProvider() string {
	m.Lock()
	defer m.Unlock()
	for i := range c.Currency.ForexProviders {
		if c.Currency.ForexProviders[i].PrimaryProvider {
			return c.Currency.ForexProviders[i].Name
		}
	}
	return ""
}

// UpdateExchangeConfig updates exchange configurations
func (c *Config) UpdateExchangeConfig(e *ExchangeConfig) error {
	m.Lock()
	defer m.Unlock()
	for i := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[i].Name, e.Name) {
			c.Exchanges[i] = *e
			return nil
		}
	}
	return fmt.Errorf(ErrExchangeNotFound, e.Name)
}

// CheckExchangeConfigValues returns configuation values for all enabled
// exchanges
func (c *Config) CheckExchangeConfigValues() error {
	exchanges := 0
	for i := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[i].Name, "GDAX") {
			c.Exchanges[i].Name = "CoinbasePro"
		}

		if c.Exchanges[i].WebsocketURL != WebsocketURLNonDefaultMessage {
			if c.Exchanges[i].WebsocketURL == "" {
				c.Exchanges[i].WebsocketURL = WebsocketURLNonDefaultMessage
			}
		}

		if c.Exchanges[i].APIURL != APIURLNonDefaultMessage {
			if c.Exchanges[i].APIURL == "" {
				// Set default if nothing set
				c.Exchanges[i].APIURL = APIURLNonDefaultMessage
			}
		}

		if c.Exchanges[i].APIURLSecondary != APIURLNonDefaultMessage {
			if c.Exchanges[i].APIURLSecondary == "" {
				// Set default if nothing set
				c.Exchanges[i].APIURLSecondary = APIURLNonDefaultMessage
			}
		}

		if c.Exchanges[i].Enabled {
			if c.Exchanges[i].Name == "" {
				return fmt.Errorf(ErrExchangeNameEmpty, i)
			}
			if len(c.Exchanges[i].AvailablePairs) == 0 {
				return fmt.Errorf(ErrExchangeAvailablePairsEmpty, c.Exchanges[i].Name)
			}
			if len(c.Exchanges[i].EnabledPairs) == 0 {
				return fmt.Errorf(ErrExchangeEnabledPairsEmpty, c.Exchanges[i].Name)
			}
			if len(c.Exchanges[i].BaseCurrencies) == 0 {
				return fmt.Errorf(ErrExchangeBaseCurrenciesEmpty, c.Exchanges[i].Name)
			}
			if c.Exchanges[i].AuthenticatedAPISupport { // non-fatal error
				if c.Exchanges[i].APIKey == "" || c.Exchanges[i].APISecret == "" ||
					c.Exchanges[i].APIKey == DefaultUnsetAPIKey ||
					c.Exchanges[i].APISecret == DefaultUnsetAPISecret {
					c.Exchanges[i].AuthenticatedAPISupport = false
					log.Warnf(WarningExchangeAuthAPIDefaultOrEmptyValues, c.Exchanges[i].Name)
				} else if c.Exchanges[i].Name == "ITBIT" || c.Exchanges[i].Name == "Bitstamp" || c.Exchanges[i].Name == "COINUT" || c.Exchanges[i].Name == "CoinbasePro" {
					if c.Exchanges[i].ClientID == "" || c.Exchanges[i].ClientID == "ClientID" {
						c.Exchanges[i].AuthenticatedAPISupport = false
						log.Warnf(WarningExchangeAuthAPIDefaultOrEmptyValues, c.Exchanges[i].Name)
					}
				}
			}
			if !c.Exchanges[i].SupportsAutoPairUpdates {
				lastUpdated := common.UnixTimestampToTime(c.Exchanges[i].PairsLastUpdated)
				lastUpdated = lastUpdated.AddDate(0, 0, configPairsLastUpdatedWarningThreshold)
				if lastUpdated.Unix() <= time.Now().Unix() {
					log.Warnf(WarningPairsLastUpdatedThresholdExceeded, c.Exchanges[i].Name, configPairsLastUpdatedWarningThreshold)
				}
			}

			if c.Exchanges[i].HTTPTimeout <= 0 {
				log.Warnf("Exchange %s HTTP Timeout value not set, defaulting to %v.", c.Exchanges[i].Name, configDefaultHTTPTimeout)
				c.Exchanges[i].HTTPTimeout = configDefaultHTTPTimeout
			}

			err := c.CheckPairConsistency(c.Exchanges[i].Name)
			if err != nil {
				log.Errorf("Exchange %s: CheckPairConsistency error: %s", c.Exchanges[i].Name, err)
			}

			if len(c.Exchanges[i].BankAccounts) == 0 {
				c.Exchanges[i].BankAccounts = append(c.Exchanges[i].BankAccounts, BankAccount{})
			} else {
				for y := range c.Exchanges[i].BankAccounts {
					bankAccount := &c.Exchanges[i].BankAccounts[y]
					if bankAccount.Enabled {
						if bankAccount.BankName == "" || bankAccount.BankAddress == "" {
							log.Warnf("banking details for %s is enabled but variables not set",
								c.Exchanges[i].Name)
							bankAccount.Enabled = false
						}

						if bankAccount.AccountName == "" || bankAccount.AccountNumber == "" {
							log.Warnf("banking account details for %s variables not set",
								c.Exchanges[i].Name)
							bankAccount.Enabled = false
						}

						if bankAccount.SupportedCurrencies == "" {
							log.Warnf("banking account details for %s acceptable funding currencies not set",
								c.Exchanges[i].Name)
							bankAccount.Enabled = false
						}

						if bankAccount.BSBNumber == "" && bankAccount.IBAN == "" &&
							bankAccount.SWIFTCode == "" {
							log.Warnf("banking account details for %s critical banking numbers not set",
								c.Exchanges[i].Name)
							bankAccount.Enabled = false
						}
					}
				}
			}
			exchanges++
		}
	}
	if exchanges == 0 {
		return errors.New(ErrNoEnabledExchanges)
	}
	return nil
}

// CheckWebserverConfigValues checks information before webserver starts and
// returns an error if values are incorrect.
func (c *Config) CheckWebserverConfigValues() error {
	if c.Webserver.AdminUsername == "" || c.Webserver.AdminPassword == "" {
		return errors.New(WarningWebserverCredentialValuesEmpty)
	}

	if !common.StringContains(c.Webserver.ListenAddress, ":") {
		return errors.New(WarningWebserverListenAddressInvalid)
	}

	portStr := common.SplitStrings(c.Webserver.ListenAddress, ":")[1]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return errors.New(WarningWebserverListenAddressInvalid)
	}

	if port < 1 || port > 65355 {
		return errors.New(WarningWebserverListenAddressInvalid)
	}

	if c.Webserver.WebsocketConnectionLimit <= 0 {
		c.Webserver.WebsocketConnectionLimit = 1
	}

	if c.Webserver.WebsocketMaxAuthFailures <= 0 {
		c.Webserver.WebsocketMaxAuthFailures = 3
	}

	return nil
}

// CheckCurrencyConfigValues checks to see if the currency config values are correct or not
func (c *Config) CheckCurrencyConfigValues() error {
	fxProviders := forexprovider.GetAvailableForexProviders()
	if len(fxProviders) == 0 {
		return errors.New("no forex providers available")
	}

	if len(fxProviders) != len(c.Currency.ForexProviders) {
		for x := range fxProviders {
			_, err := c.GetForexProviderConfig(fxProviders[x])
			if err != nil {
				log.Warnf("%s forex provider not found, adding to config..", fxProviders[x])
				c.Currency.ForexProviders = append(c.Currency.ForexProviders, base.Settings{
					Name:             fxProviders[x],
					RESTPollingDelay: 600,
					APIKey:           DefaultUnsetAPIKey,
					APIKeyLvl:        -1,
				})
			}
		}
	}

	count := 0
	for i := range c.Currency.ForexProviders {
		if c.Currency.ForexProviders[i].Enabled {
			if c.Currency.ForexProviders[i].APIKey == DefaultUnsetAPIKey && c.Currency.ForexProviders[i].Name != DefaultForexProviderExchangeRatesAPI {
				log.Warnf("%s enabled forex provider API key not set. Please set this in your config.json file", c.Currency.ForexProviders[i].Name)
				c.Currency.ForexProviders[i].Enabled = false
				c.Currency.ForexProviders[i].PrimaryProvider = false
				continue
			}

			if c.Currency.ForexProviders[i].Name == "CurrencyConverter" {
				if c.Currency.ForexProviders[i].Enabled &&
					c.Currency.ForexProviders[i].PrimaryProvider &&
					(c.Currency.ForexProviders[i].APIKey == "" ||
						c.Currency.ForexProviders[i].APIKey == DefaultUnsetAPIKey) {
					log.Warnf("CurrencyConverter forex provider no longer supports unset API key requests. Switching to ExchangeRates FX provider..")
					c.Currency.ForexProviders[i].Enabled = false
					c.Currency.ForexProviders[i].PrimaryProvider = false
					c.Currency.ForexProviders[i].APIKey = DefaultUnsetAPIKey
					c.Currency.ForexProviders[i].APIKeyLvl = -1
					continue
				}
			}

			if c.Currency.ForexProviders[i].APIKeyLvl == -1 && c.Currency.ForexProviders[i].Name != DefaultForexProviderExchangeRatesAPI {
				log.Warnf("%s APIKey Level not set, functions limited. Please set this in your config.json file",
					c.Currency.ForexProviders[i].Name)
			}
			count++
		}
	}

	if count == 0 {
		for x := range c.Currency.ForexProviders {
			if c.Currency.ForexProviders[x].Name == DefaultForexProviderExchangeRatesAPI {
				c.Currency.ForexProviders[x].Enabled = true
				c.Currency.ForexProviders[x].PrimaryProvider = true
				log.Warn("Using ExchangeRatesAPI for default forex provider.")
			}
		}
	}

	if c.Currency.CryptocurrencyProvider == (CryptocurrencyProvider{}) {
		c.Currency.CryptocurrencyProvider.Name = "CoinMarketCap"
		c.Currency.CryptocurrencyProvider.Enabled = false
		c.Currency.CryptocurrencyProvider.Verbose = false
		c.Currency.CryptocurrencyProvider.AccountPlan = DefaultUnsetAccountPlan
		c.Currency.CryptocurrencyProvider.APIkey = DefaultUnsetAPIKey
	}

	if c.Currency.CryptocurrencyProvider.Enabled {
		if c.Currency.CryptocurrencyProvider.APIkey == "" ||
			c.Currency.CryptocurrencyProvider.APIkey == DefaultUnsetAPIKey {
			log.Warnf("CryptocurrencyProvider enabled but api key is unset please set this in your config.json file")
		}
		if c.Currency.CryptocurrencyProvider.AccountPlan == "" ||
			c.Currency.CryptocurrencyProvider.AccountPlan == DefaultUnsetAccountPlan {
			log.Warnf("CryptocurrencyProvider enabled but account plan is unset please set this in your config.json file")
		}
	} else {
		if c.Currency.CryptocurrencyProvider.APIkey == "" {
			c.Currency.CryptocurrencyProvider.APIkey = DefaultUnsetAPIKey
		}
		if c.Currency.CryptocurrencyProvider.AccountPlan == "" {
			c.Currency.CryptocurrencyProvider.AccountPlan = DefaultUnsetAccountPlan
		}
	}

	if c.Currency.Cryptocurrencies.Join() == "" {
		if c.Cryptocurrencies.Join() != "" {
			c.Currency.Cryptocurrencies = c.Cryptocurrencies
			c.Cryptocurrencies = nil
		} else {
			c.Currency.Cryptocurrencies = currency.GetDefaultCryptocurrencies()
		}
	}

	if c.Currency.CurrencyPairFormat == nil {
		if c.CurrencyPairFormat != nil {
			c.Currency.CurrencyPairFormat = c.CurrencyPairFormat
			c.CurrencyPairFormat = nil
		} else {
			c.Currency.CurrencyPairFormat = &CurrencyPairFormatConfig{
				Delimiter: "-",
				Uppercase: true,
			}
		}
	}

	if c.Currency.FiatDisplayCurrency.IsEmpty() {
		if c.FiatDisplayCurrency.IsEmpty() {
			c.Currency.FiatDisplayCurrency = c.FiatDisplayCurrency
			c.FiatDisplayCurrency = currency.NewCode("")
		} else {
			c.Currency.FiatDisplayCurrency = currency.USD
		}
	}
	return nil
}

// RetrieveConfigCurrencyPairs splits, assigns and verifies enabled currency
// pairs either cryptoCurrencies or fiatCurrencies
func (c *Config) RetrieveConfigCurrencyPairs(enabledOnly bool) error {
	cryptoCurrencies := c.Currency.Cryptocurrencies
	fiatCurrencies := currency.GetFiatCurrencies()

	for x := range c.Exchanges {
		if !c.Exchanges[x].Enabled && enabledOnly {
			continue
		}

		baseCurrencies := c.Exchanges[x].BaseCurrencies
		for y := range baseCurrencies {
			if !fiatCurrencies.Contains(baseCurrencies[y]) {
				fiatCurrencies = append(fiatCurrencies, baseCurrencies[y])
			}
		}
	}

	for x := range c.Exchanges {
		var pairs []currency.Pair
		var err error
		if !c.Exchanges[x].Enabled && enabledOnly {
			pairs, err = c.GetEnabledPairs(c.Exchanges[x].Name)
		} else {
			pairs, err = c.GetAvailablePairs(c.Exchanges[x].Name)
		}

		if err != nil {
			return err
		}

		for y := range pairs {
			if !fiatCurrencies.Contains(pairs[y].Base) &&
				!cryptoCurrencies.Contains(pairs[y].Base) {
				cryptoCurrencies = append(cryptoCurrencies, pairs[y].Base)
			}

			if !fiatCurrencies.Contains(pairs[y].Quote) &&
				!cryptoCurrencies.Contains(pairs[y].Quote) {
				cryptoCurrencies = append(cryptoCurrencies, pairs[y].Quote)
			}
		}
	}

	currency.UpdateCurrencies(fiatCurrencies, false)
	currency.UpdateCurrencies(cryptoCurrencies, true)
	return nil
}

// CheckLoggerConfig checks to see logger values are present and valid in config
// if not creates a default instance of the logger
func (c *Config) CheckLoggerConfig() error {
	m.Lock()
	defer m.Unlock()

	// check if enabled is nil or level is a blank string
	if c.Logging.Enabled == nil || c.Logging.Level == "" {
		// Creates a new pointer to bool and sets it as true
		t := func(t bool) *bool { return &t }(true)

		log.Warn("Missing or invalid config settings using safe defaults")

		// Set logger to safe defaults

		c.Logging = log.Logging{
			Enabled:      t,
			Level:        "DEBUG|INFO|WARN|ERROR|FATAL",
			ColourOutput: false,
			File:         "debug.txt",
			Rotate:       false,
		}
		log.Logger = &c.Logging
	} else {
		log.Logger = &c.Logging
	}

	if len(c.Logging.File) > 0 {
		logPath := filepath.Join(common.GetDefaultDataDir(runtime.GOOS), "logs")
		err := common.CreateDir(logPath)
		if err != nil {
			return err
		}
		log.LogPath = logPath
	}
	return nil
}

// CheckNTPConfig checks for missing or incorrectly configured NTPClient and recreates with known safe defaults
func (c *Config) CheckNTPConfig() {
	m.Lock()
	defer m.Unlock()

	if c.NTPClient.AllowedDifference == nil || *c.NTPClient.AllowedDifference == 0 {
		c.NTPClient.AllowedDifference = new(time.Duration)
		*c.NTPClient.AllowedDifference = defaultNTPAllowedDifference
	}

	if c.NTPClient.AllowedNegativeDifference == nil || *c.NTPClient.AllowedNegativeDifference <= 0 {
		c.NTPClient.AllowedNegativeDifference = new(time.Duration)
		*c.NTPClient.AllowedNegativeDifference = defaultNTPAllowedNegativeDifference
	}

	if len(c.NTPClient.Pool) < 1 {
		log.Warn("NTPClient enabled with no servers configured enabling default pool")
		c.NTPClient.Pool = []string{"pool.ntp.org:123"}
	}
}

// DisableNTPCheck allows the user to change how they are prompted for timesync alerts
func (c *Config) DisableNTPCheck(input io.Reader) (string, error) {
	m.Lock()
	defer m.Unlock()

	reader := bufio.NewReader(input)
	log.Warn("Your system time is out of sync this may cause issues with trading")
	log.Warn("How would you like to show future notifications? (a)lert / (w)arn / (d)isable \n")

	var answered = false
	for ok := true; ok; ok = (!answered) {
		answer, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		answer = strings.TrimRight(answer, "\r\n")
		switch answer {
		case "a":
			c.NTPClient.Level = 0
			answered = true
			return "Time sync has been set to alert", nil
		case "w":
			c.NTPClient.Level = 1
			answered = true
			return "Time sync has been set to warn only", nil
		case "d":
			c.NTPClient.Level = -1
			answered = true
			return "Future notications for out time sync have been disabled", nil
		}
	}
	return "", errors.New("something went wrong NTPCheck should never make it this far")
}

// CheckConnectionMonitorConfig checks and if zero value assigns default values
func (c *Config) CheckConnectionMonitorConfig() {
	m.Lock()
	defer m.Unlock()

	if c.ConnectionMonitor.CheckInterval == 0 {
		c.ConnectionMonitor.CheckInterval = connchecker.DefaultCheckInterval
	}

	if len(c.ConnectionMonitor.DNSList) == 0 {
		c.ConnectionMonitor.DNSList = connchecker.DefaultDNSList
	}

	if len(c.ConnectionMonitor.PublicDomainList) == 0 {
		c.ConnectionMonitor.PublicDomainList = connchecker.DefaultDomainList
	}
}

// GetFilePath returns the desired config file or the default config file name
// based on if the application is being run under test or normal mode.
func GetFilePath(file string) (string, error) {
	if file != "" {
		return file, nil
	}

	if flag.Lookup("test.v") != nil && !testBypass {
		return ConfigTestFile, nil
	}

	exePath, err := common.GetExecutablePath()
	if err != nil {
		return "", err
	}

	oldDir := exePath + common.GetOSPathSlash()
	oldDirs := []string{oldDir + ConfigFile, oldDir + EncryptedConfigFile}

	newDir := common.GetDefaultDataDir(runtime.GOOS) + common.GetOSPathSlash()
	err = common.CreateDir(newDir)
	if err != nil {
		return "", err
	}
	newDirs := []string{newDir + ConfigFile, newDir + EncryptedConfigFile}

	// First upgrade the old dir config file if it exists to the corresponding new one
	for x := range oldDirs {
		_, err := os.Stat(oldDirs[x])
		if os.IsNotExist(err) {
			continue
		}
		if filepath.Ext(oldDirs[x]) == ".json" {
			err = os.Rename(oldDirs[x], newDirs[0])
			if err != nil {
				return "", err
			}
			log.Debugf("Renamed old config file %s to %s", oldDirs[x], newDirs[0])
		} else {
			err = os.Rename(oldDirs[x], newDirs[1])
			if err != nil {
				return "", err
			}
			log.Debugf("Renamed old config file %s to %s", oldDirs[x], newDirs[1])
		}
	}

	// Secondly check to see if the new config file extension is correct or not
	for x := range newDirs {
		_, err := os.Stat(newDirs[x])
		if os.IsNotExist(err) {
			continue
		}

		data, err := common.ReadFile(newDirs[x])
		if err != nil {
			return "", err
		}

		if ConfirmECS(data) {
			if filepath.Ext(newDirs[x]) == ".dat" {
				return newDirs[x], nil
			}

			err = os.Rename(newDirs[x], newDirs[1])
			if err != nil {
				return "", err
			}
			return newDirs[1], nil
		}

		if filepath.Ext(newDirs[x]) == ".json" {
			return newDirs[x], nil
		}

		err = os.Rename(newDirs[x], newDirs[0])
		if err != nil {
			return "", err
		}

		return newDirs[0], nil
	}

	return "", errors.New("config default file path error")
}

// ReadConfig verifies and checks for encryption and verifies the unencrypted
// file contains JSON.
func (c *Config) ReadConfig(configPath string) error {
	defaultPath, err := GetFilePath(configPath)
	if err != nil {
		return err
	}

	file, err := common.ReadFile(defaultPath)
	if err != nil {
		return err
	}

	if !ConfirmECS(file) {
		err = ConfirmConfigJSON(file, &c)
		if err != nil {
			return err
		}

		if c.EncryptConfig == configFileEncryptionDisabled {
			return nil
		}

		if c.EncryptConfig == configFileEncryptionPrompt {
			m.Lock()
			IsInitialSetup = true
			m.Unlock()
			if c.PromptForConfigEncryption() {
				c.EncryptConfig = configFileEncryptionEnabled
				return c.SaveConfig(defaultPath)
			}
		}
	} else {
		errCounter := 0
		for {
			if errCounter >= configMaxAuthFailres {
				return errors.New("failed to decrypt config after 3 attempts")
			}
			key, err := PromptForConfigKey(IsInitialSetup)
			if err != nil {
				log.Errorf("PromptForConfigKey err: %s", err)
				errCounter++
				continue
			}

			var f []byte
			f = append(f, file...)
			data, err := DecryptConfigFile(f, key)
			if err != nil {
				log.Errorf("DecryptConfigFile err: %s", err)
				errCounter++
				continue
			}

			err = ConfirmConfigJSON(data, &c)
			if err != nil {
				if errCounter < configMaxAuthFailres {
					log.Errorf("Invalid password.")
				}
				errCounter++
				continue
			}
			break
		}
	}
	return nil
}

// SaveConfig saves your configuration to your desired path
func (c *Config) SaveConfig(configPath string) error {
	defaultPath, err := GetFilePath(configPath)
	if err != nil {
		return err
	}

	payload, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return err
	}

	if c.EncryptConfig == configFileEncryptionEnabled {
		var key []byte

		if IsInitialSetup {
			key, err = PromptForConfigKey(true)
			if err != nil {
				return err
			}
			IsInitialSetup = false
		}

		payload, err = EncryptConfigFile(payload, key)
		if err != nil {
			return err
		}
	}
	return common.WriteFile(defaultPath, payload)
}

// CheckConfig checks all config settings
func (c *Config) CheckConfig() error {
	err := c.CheckExchangeConfigValues()
	if err != nil {
		return fmt.Errorf(ErrCheckingConfigValues, err)
	}

	c.CheckConnectionMonitorConfig()
	c.CheckCommunicationsConfig()

	if c.Webserver.Enabled {
		err = c.CheckWebserverConfigValues()
		if err != nil {
			log.Warnf(ErrCheckingConfigValues, err)
			c.Webserver.Enabled = false
		}
	}

	err = c.CheckCurrencyConfigValues()
	if err != nil {
		return err
	}

	if c.GlobalHTTPTimeout <= 0 {
		log.Warnf("Global HTTP Timeout value not set, defaulting to %v.", configDefaultHTTPTimeout)
		c.GlobalHTTPTimeout = configDefaultHTTPTimeout
	}

	return c.CheckClientBankAccounts()
}

// LoadConfig loads your configuration file into your configuration object
func (c *Config) LoadConfig(configPath string) error {
	err := c.ReadConfig(configPath)
	if err != nil {
		return fmt.Errorf(ErrFailureOpeningConfig, configPath, err)
	}

	return c.CheckConfig()
}

// UpdateConfig updates the config with a supplied config file
func (c *Config) UpdateConfig(configPath string, newCfg *Config) error {
	err := newCfg.CheckConfig()
	if err != nil {
		return err
	}

	c.Name = newCfg.Name
	c.EncryptConfig = newCfg.EncryptConfig
	c.Currency = newCfg.Currency
	c.GlobalHTTPTimeout = newCfg.GlobalHTTPTimeout
	c.Portfolio = newCfg.Portfolio
	c.Communications = newCfg.Communications
	c.Webserver = newCfg.Webserver
	c.Exchanges = newCfg.Exchanges

	err = c.SaveConfig(configPath)
	if err != nil {
		return err
	}

	return c.LoadConfig(configPath)
}

// GetConfig returns a pointer to a configuration object
func GetConfig() *Config {
	return &Cfg
}
