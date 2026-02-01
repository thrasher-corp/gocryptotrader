package config

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	"github.com/thrasher-corp/gocryptotrader/connchecker"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctscript "github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
)

var (
	errExchangeConfigIsNil = errors.New("exchange config is nil")
	errPairsManagerIsNil   = errors.New("currency pairs manager is nil")
	errDecryptFailed       = errors.New("failed to decrypt config after 3 attempts")
)

// GetCurrencyConfig returns currency configurations
func (c *Config) GetCurrencyConfig() currency.Config {
	return c.Currency
}

// GetExchangeBankAccounts returns banking details associated with an exchange for depositing funds
func (c *Config) GetExchangeBankAccounts(exchangeName, id, depositingCurrency string) (*banking.Account, error) {
	e, err := c.GetExchangeConfig(exchangeName)
	if err != nil {
		return nil, err
	}

	for y := range e.BankAccounts {
		if strings.EqualFold(e.BankAccounts[y].ID, id) {
			if common.StringSliceCompareInsensitive(strings.Split(e.BankAccounts[y].SupportedCurrencies, ","), depositingCurrency) {
				return &e.BankAccounts[y], nil
			}
		}
	}

	return nil, fmt.Errorf("exchange %s bank details not found for %s", exchangeName, depositingCurrency)
}

// UpdateExchangeBankAccounts updates the configuration for the associated
// exchange bank
func (c *Config) UpdateExchangeBankAccounts(exchangeName string, bankCfg []banking.Account) error {
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
func (c *Config) GetClientBankAccounts(exchangeName, targetCurrency string) (*banking.Account, error) {
	m.Lock()
	defer m.Unlock()

	for x := range c.BankAccounts {
		if (strings.Contains(c.BankAccounts[x].SupportedExchanges, exchangeName) ||
			c.BankAccounts[x].SupportedExchanges == "ALL") &&
			strings.Contains(c.BankAccounts[x].SupportedCurrencies, targetCurrency) {
			return &c.BankAccounts[x], nil
		}
	}
	return nil, fmt.Errorf("client banking details not found for %s and currency %s",
		exchangeName,
		targetCurrency)
}

// UpdateClientBankAccounts updates the configuration for a bank
func (c *Config) UpdateClientBankAccounts(bankCfg *banking.Account) error {
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
func (c *Config) CheckClientBankAccounts() {
	m.Lock()
	defer m.Unlock()

	if len(c.BankAccounts) == 0 {
		c.BankAccounts = append(c.BankAccounts,
			banking.Account{
				ID:                  "test-bank-01",
				BankName:            "Test Bank",
				BankAddress:         "42 Bank Street",
				BankPostalCode:      "13337",
				BankPostalCity:      "Satoshiville",
				BankCountry:         "Japan",
				AccountName:         "Satoshi Nakamoto",
				AccountNumber:       "0234",
				SWIFTCode:           "91272837",
				IBAN:                "98218738671897",
				SupportedCurrencies: "USD",
				SupportedExchanges:  "Kraken,Bitstamp",
			},
		)
		return
	}

	for i := range c.BankAccounts {
		if c.BankAccounts[i].Enabled {
			err := c.BankAccounts[i].Validate()
			if err != nil {
				c.BankAccounts[i].Enabled = false
				log.Warnln(log.ConfigMgr, err.Error())
			}
		}
	}
}

// PurgeExchangeAPICredentials purges the stored API credentials
func (c *Config) PurgeExchangeAPICredentials() {
	m.Lock()
	defer m.Unlock()
	for x := range c.Exchanges {
		if !c.Exchanges[x].API.AuthenticatedSupport && !c.Exchanges[x].API.AuthenticatedWebsocketSupport {
			continue
		}
		c.Exchanges[x].API.AuthenticatedSupport = false
		c.Exchanges[x].API.AuthenticatedWebsocketSupport = false

		if c.Exchanges[x].API.CredentialsValidator.RequiresKey {
			c.Exchanges[x].API.Credentials.Key = DefaultAPIKey
		}

		if c.Exchanges[x].API.CredentialsValidator.RequiresSecret {
			c.Exchanges[x].API.Credentials.Secret = DefaultAPISecret
		}

		if c.Exchanges[x].API.CredentialsValidator.RequiresClientID {
			c.Exchanges[x].API.Credentials.ClientID = DefaultAPIClientID
		}

		c.Exchanges[x].API.Credentials.PEMKey = ""
		c.Exchanges[x].API.Credentials.OTPSecret = ""
	}
}

// GetCommunicationsConfig returns the communications configuration
func (c *Config) GetCommunicationsConfig() base.CommunicationsConfig {
	m.Lock()
	comms := c.Communications
	m.Unlock()
	return comms
}

// UpdateCommunicationsConfig sets a new updated version of a Communications
// configuration
func (c *Config) UpdateCommunicationsConfig(config *base.CommunicationsConfig) {
	m.Lock()
	c.Communications = *config
	m.Unlock()
}

// GetCryptocurrencyProviderConfig returns the communications configuration
func (c *Config) GetCryptocurrencyProviderConfig() currency.Provider {
	m.Lock()
	provider := c.Currency.CryptocurrencyProvider
	m.Unlock()
	return provider
}

// UpdateCryptocurrencyProviderConfig returns the communications configuration
func (c *Config) UpdateCryptocurrencyProviderConfig(config currency.Provider) {
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
		c.Communications.SlackConfig = base.SlackConfig{
			Name:              "Slack",
			TargetChannel:     "general",
			VerificationToken: "testtest",
		}
	}

	if c.Communications.SMSGlobalConfig.Name == "" {
		if c.SMS != nil {
			if c.SMS.Contacts != nil {
				c.Communications.SMSGlobalConfig = base.SMSGlobalConfig{
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
				c.Communications.SMSGlobalConfig = base.SMSGlobalConfig{
					Name:     "SMSGlobal",
					From:     c.Name,
					Username: "main",
					Password: "test",

					Contacts: []base.SMSContact{
						{
							Name:    "bob",
							Number:  "1234",
							Enabled: false,
						},
					},
				}
			}
		} else {
			c.Communications.SMSGlobalConfig = base.SMSGlobalConfig{
				Name:     "SMSGlobal",
				Username: "main",
				Password: "test",

				Contacts: []base.SMSContact{
					{
						Name:    "bob",
						Number:  "1234",
						Enabled: false,
					},
				},
			}
		}
	} else {
		if c.Communications.SMSGlobalConfig.From == "" {
			c.Communications.SMSGlobalConfig.From = c.Name
		}

		if len(c.Communications.SMSGlobalConfig.From) > 11 {
			log.Warnf(log.ConfigMgr, "SMSGlobal config supplied from name exceeds 11 characters, trimming.\n")
			c.Communications.SMSGlobalConfig.From = c.Communications.SMSGlobalConfig.From[:11]
		}

		if c.SMS != nil {
			// flush old SMS config
			c.SMS = nil
		}
	}

	if c.Communications.SMTPConfig.Name == "" {
		c.Communications.SMTPConfig = base.SMTPConfig{
			Name:            "SMTP",
			Host:            "smtp.google.com",
			Port:            "537",
			AccountName:     "some",
			AccountPassword: "password",
			RecipientList:   "lol123@gmail.com",
		}
	}

	if c.Communications.TelegramConfig.Name == "" {
		c.Communications.TelegramConfig = base.TelegramConfig{
			Name:              "Telegram",
			VerificationToken: "testest",
		}
	}

	if c.Communications.TelegramConfig.AuthorisedClients == nil {
		c.Communications.TelegramConfig.AuthorisedClients = map[string]int64{"user_example": 0}
	}

	if c.Communications.SlackConfig.Name != "Slack" ||
		c.Communications.SMSGlobalConfig.Name != "SMSGlobal" ||
		c.Communications.SMTPConfig.Name != "SMTP" ||
		c.Communications.TelegramConfig.Name != "Telegram" {
		log.Warnln(log.ConfigMgr, "Communications config name/s not set correctly")
	}
	if c.Communications.SlackConfig.Enabled {
		if c.Communications.SlackConfig.TargetChannel == "" ||
			c.Communications.SlackConfig.VerificationToken == "" ||
			c.Communications.SlackConfig.VerificationToken == "testtest" {
			c.Communications.SlackConfig.Enabled = false
			log.Warnln(log.ConfigMgr, "Slack enabled in config but variable data not set, disabling.")
		}
	}
	if c.Communications.SMSGlobalConfig.Enabled {
		if c.Communications.SMSGlobalConfig.Username == "" ||
			c.Communications.SMSGlobalConfig.Password == "" ||
			len(c.Communications.SMSGlobalConfig.Contacts) == 0 {
			c.Communications.SMSGlobalConfig.Enabled = false
			log.Warnln(log.ConfigMgr, "SMSGlobal enabled in config but variable data not set, disabling.")
		}
	}
	if c.Communications.SMTPConfig.Enabled {
		if c.Communications.SMTPConfig.Host == "" ||
			c.Communications.SMTPConfig.Port == "" ||
			c.Communications.SMTPConfig.AccountName == "" ||
			c.Communications.SMTPConfig.AccountPassword == "" {
			c.Communications.SMTPConfig.Enabled = false
			log.Warnln(log.ConfigMgr, "SMTP enabled in config but variable data not set, disabling.")
		}
	}
	if c.Communications.TelegramConfig.Enabled {
		if _, ok := c.Communications.TelegramConfig.AuthorisedClients["user_example"]; ok ||
			len(c.Communications.TelegramConfig.AuthorisedClients) == 0 ||
			c.Communications.TelegramConfig.VerificationToken == "" ||
			c.Communications.TelegramConfig.VerificationToken == "testest" {
			c.Communications.TelegramConfig.Enabled = false
			log.Warnln(log.ConfigMgr, "Telegram enabled in config but variable data not set, disabling.")
		}
	}
}

// GetExchangeAssetTypes returns the exchanges supported asset types
func (c *Config) GetExchangeAssetTypes(exchName string) (asset.Items, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	if exchCfg.CurrencyPairs == nil {
		return nil, fmt.Errorf("%s %w", exchName, errPairsManagerIsNil)
	}

	return exchCfg.CurrencyPairs.GetAssetTypes(false), nil
}

// SupportsExchangeAssetType returns whether or not the exchange supports the supplied asset type
func (c *Config) SupportsExchangeAssetType(exchName string, assetType asset.Item) error {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return err
	}

	if exchCfg.CurrencyPairs == nil {
		return fmt.Errorf("%s %w", exchName, errPairsManagerIsNil)
	}

	if !assetType.IsValid() {
		return fmt.Errorf("exchange %s invalid asset type %s",
			exchName,
			assetType)
	}

	if !exchCfg.CurrencyPairs.GetAssetTypes(false).Contains(assetType) {
		return fmt.Errorf("exchange %s unsupported asset type %s",
			exchName,
			assetType)
	}
	return nil
}

// SetPairs sets the exchanges currency pairs
func (c *Config) SetPairs(exchName string, assetType asset.Item, enabled bool, pairs currency.Pairs) error {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return err
	}

	err = c.SupportsExchangeAssetType(exchName, assetType)
	if err != nil {
		return err
	}

	return exchCfg.CurrencyPairs.StorePairs(assetType, pairs, enabled)
}

// GetCurrencyPairConfig returns currency pair config for the desired exchange and asset type
func (c *Config) GetCurrencyPairConfig(exchName string, assetType asset.Item) (*currency.PairStore, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	err = c.SupportsExchangeAssetType(exchName, assetType)
	if err != nil {
		return nil, err
	}

	return exchCfg.CurrencyPairs.Get(assetType)
}

// CheckPairConfigFormats checks to see if the pair config format is valid
func (c *Config) CheckPairConfigFormats(exchName string) error {
	assetTypes, err := c.GetExchangeAssetTypes(exchName)
	if err != nil {
		return err
	}

	for x := range assetTypes {
		assetType := assetTypes[x]
		pairFmt, err := c.GetPairFormat(exchName, assetType)
		if err != nil {
			return err
		}

		// No err checking is required as the above checks the same
		// conditions
		pairs, _ := c.GetCurrencyPairConfig(exchName, assetType)

		if len(pairs.Available) == 0 || len(pairs.Enabled) == 0 {
			continue
		}

		checker := func(enabled bool) error {
			pairsType := "enabled"
			loadedPairs := pairs.Enabled
			if !enabled {
				pairsType = "available"
				loadedPairs = pairs.Available
			}

			for y := range loadedPairs {
				if pairFmt.Delimiter != "" {
					if !strings.Contains(loadedPairs[y].String(), pairFmt.Delimiter) {
						return fmt.Errorf("exchange %s %s %s pairs does not contain delimiter", exchName, pairsType, assetType)
					}
				}
			}
			return nil
		}

		err = checker(true)
		if err != nil {
			return err
		}

		err = checker(false)
		if err != nil {
			return err
		}
	}

	return nil
}

// CheckPairConsistency checks to see if the enabled pair exists in the
// available pairs list
func (c *Config) CheckPairConsistency(exchName string) error {
	assetTypes, err := c.GetExchangeAssetTypes(exchName)
	if err != nil {
		return err
	}

	var atLeastOneEnabled bool
	for x := range assetTypes {
		enabledPairs, err := c.GetEnabledPairs(exchName, assetTypes[x])
		if err == nil {
			if len(enabledPairs) != 0 {
				atLeastOneEnabled = true
				continue
			}
			var enabled bool
			enabled, err = c.AssetTypeEnabled(assetTypes[x], exchName)
			if err != nil {
				return err
			}

			if !enabled {
				continue
			}

			var availPairs currency.Pairs
			availPairs, err = c.GetAvailablePairs(exchName, assetTypes[x])
			if err != nil {
				return err
			}
			if len(availPairs) == 0 {
				// the other assets may have currency pairs
				continue
			}

			var rPair currency.Pair
			rPair, err = availPairs.GetRandomPair()
			if err != nil {
				return err
			}

			err = c.SetPairs(exchName, assetTypes[x], true, currency.Pairs{rPair})
			if err != nil {
				return err
			}
			atLeastOneEnabled = true
			continue
		}

		// On error an enabled pair is not found in the available pairs list
		// so remove and report
		availPairs, err := c.GetAvailablePairs(exchName, assetTypes[x])
		if err != nil {
			return err
		}

		var pairs, pairsRemoved currency.Pairs
		for x := range enabledPairs {
			if !availPairs.Contains(enabledPairs[x], true) {
				pairsRemoved = append(pairsRemoved, enabledPairs[x])
				continue
			}
			pairs = append(pairs, enabledPairs[x])
		}

		if len(pairsRemoved) == 0 {
			return fmt.Errorf("check pair consistency fault for asset %s, conflict found but no pairs removed",
				assetTypes[x])
		}

		// Flush corrupted/misspelled enabled pairs in config
		err = c.SetPairs(exchName, assetTypes[x], true, pairs)
		if err != nil {
			return err
		}

		log.Warnf(log.ConfigMgr,
			"Exchange %s: [%v] Removing enabled pair(s) %v from enabled pairs list, as it isn't located in the available pairs list.\n",
			exchName,
			assetTypes[x],
			pairsRemoved.Strings())

		if len(pairs) != 0 {
			atLeastOneEnabled = true
			continue
		}

		enabled, err := c.AssetTypeEnabled(assetTypes[x], exchName)
		if err != nil {
			return err
		}

		if !enabled {
			continue
		}

		var rPair currency.Pair
		rPair, err = availPairs.GetRandomPair()
		if err != nil {
			return err
		}

		err = c.SetPairs(exchName, assetTypes[x], true, currency.Pairs{rPair})
		if err != nil {
			return err
		}
		atLeastOneEnabled = true
	}

	// If no pair is enabled across the entire range of assets, then at least
	// enable one and turn on the asset type
	if !atLeastOneEnabled {
		avail, err := c.GetAvailablePairs(exchName, assetTypes[0])
		if err != nil {
			return err
		}

		if len(avail) == 0 {
			return nil
		}

		rPair, err := avail.GetRandomPair()
		if err != nil {
			return err
		}

		err = c.SetPairs(exchName, assetTypes[0], true, currency.Pairs{rPair})
		if err != nil {
			return err
		}
		log.Warnf(log.ConfigMgr,
			"Exchange %s: [%v] No enabled pairs found in available pairs list, randomly added %v pair.\n",
			exchName,
			assetTypes[0],
			rPair)
	}
	return nil
}

// SupportsPair returns true or not whether the exchange supports the supplied
// pair
func (c *Config) SupportsPair(exchName string, p currency.Pair, assetType asset.Item) bool {
	pairs, err := c.GetAvailablePairs(exchName, assetType)
	if err != nil {
		return false
	}
	return pairs.Contains(p, false)
}

// GetPairFormat returns the exchanges pair config storage format
func (c *Config) GetPairFormat(exchName string, assetType asset.Item) (currency.PairFormat, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return currency.EMPTYFORMAT, err
	}

	err = c.SupportsExchangeAssetType(exchName, assetType)
	if err != nil {
		return currency.EMPTYFORMAT, err
	}

	if exchCfg.CurrencyPairs.UseGlobalFormat {
		return *exchCfg.CurrencyPairs.ConfigFormat, nil
	}

	p, err := exchCfg.CurrencyPairs.Get(assetType)
	if err != nil {
		return currency.EMPTYFORMAT, err
	}

	if p == nil {
		return currency.EMPTYFORMAT,
			fmt.Errorf("exchange %s pair store for asset type %s is nil",
				exchName,
				assetType)
	}

	if p.ConfigFormat == nil {
		return currency.EMPTYFORMAT,
			fmt.Errorf("exchange %s pair config format for asset type %s is nil",
				exchName,
				assetType)
	}

	return *p.ConfigFormat, nil
}

// GetAvailablePairs returns a list of currency pairs for a specific exchange
func (c *Config) GetAvailablePairs(exchName string, assetType asset.Item) (currency.Pairs, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	pairFormat, err := c.GetPairFormat(exchName, assetType)
	if err != nil {
		return nil, err
	}

	pairs, err := exchCfg.CurrencyPairs.GetPairs(assetType, false)
	if err != nil {
		return nil, err
	}

	if pairs == nil {
		return nil, nil
	}

	return pairs.Format(pairFormat), nil
}

// GetDefaultSyncManagerConfig returns a config with default values
func GetDefaultSyncManagerConfig() SyncManagerConfig {
	return SyncManagerConfig{
		Enabled:                 true,
		SynchronizeTicker:       true,
		SynchronizeOrderbook:    true,
		SynchronizeTrades:       false,
		SynchronizeContinuously: true,
		TimeoutREST:             DefaultSyncerTimeoutREST,
		TimeoutWebsocket:        DefaultSyncerTimeoutWebsocket,
		NumWorkers:              DefaultSyncerWorkers,
		FiatDisplayCurrency:     currency.USD,
		PairFormatDisplay: &currency.PairFormat{
			Delimiter: "-",
			Uppercase: true,
		},
		Verbose:                 false,
		LogSyncUpdateEvents:     true,
		LogSwitchProtocolEvents: true,
		LogInitialSyncEvents:    true,
	}
}

// CheckSyncManagerConfig checks config for valid values
// sets defaults if values are invalid
func (c *Config) CheckSyncManagerConfig() {
	m.Lock()
	defer m.Unlock()
	if c.SyncManagerConfig == (SyncManagerConfig{}) {
		c.SyncManagerConfig = GetDefaultSyncManagerConfig()
		return
	}
	if c.SyncManagerConfig.TimeoutWebsocket <= 0 {
		log.Warnf(log.ConfigMgr, "Invalid sync manager websocket timeout value %v, defaulting to %v\n", c.SyncManagerConfig.TimeoutWebsocket, DefaultSyncerTimeoutWebsocket)
		c.SyncManagerConfig.TimeoutWebsocket = DefaultSyncerTimeoutWebsocket
	}
	if c.SyncManagerConfig.PairFormatDisplay == nil {
		log.Warnf(log.ConfigMgr, "Invalid sync manager pair format value %v, using default format eg BTC-USD\n", c.SyncManagerConfig.PairFormatDisplay)
		c.SyncManagerConfig.PairFormatDisplay = &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		}
	}
	if c.SyncManagerConfig.TimeoutREST <= 0 {
		log.Warnf(log.ConfigMgr, "Invalid sync manager REST timeout value %v, defaulting to %v\n", c.SyncManagerConfig.TimeoutREST, DefaultSyncerTimeoutREST)
		c.SyncManagerConfig.TimeoutREST = DefaultSyncerTimeoutREST
	}
	if c.SyncManagerConfig.NumWorkers <= 0 {
		log.Warnf(log.ConfigMgr, "Invalid sync manager worker count value %v, defaulting to %v\n", c.SyncManagerConfig.NumWorkers, DefaultSyncerWorkers)
		c.SyncManagerConfig.NumWorkers = DefaultSyncerWorkers
	}
	if c.SyncManagerConfig.FiatDisplayCurrency.IsEmpty() {
		log.Warnf(log.ConfigMgr, "Invalid sync manager fiat display currency value, defaulting to %v\n", currency.USD)
		c.SyncManagerConfig.FiatDisplayCurrency = currency.USD
	}
}

// GetEnabledPairs returns a list of currency pairs for a specific exchange
func (c *Config) GetEnabledPairs(exchName string, assetType asset.Item) (currency.Pairs, error) {
	exchCfg, err := c.GetExchangeConfig(exchName)
	if err != nil {
		return nil, err
	}

	pairFormat, err := c.GetPairFormat(exchName, assetType)
	if err != nil {
		return nil, err
	}

	pairs, err := exchCfg.CurrencyPairs.GetPairs(assetType, true)
	if err != nil {
		return pairs, err
	}

	if pairs == nil {
		return nil, nil
	}

	return pairs.Format(pairFormat), nil
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

// GetCurrencyPairDisplayConfig retrieves the currency pair display preference
func (c *Config) GetCurrencyPairDisplayConfig() *currency.PairFormat {
	return c.Currency.CurrencyPairFormat
}

// GetAllExchangeConfigs returns all exchange configurations
func (c *Config) GetAllExchangeConfigs() []Exchange {
	m.Lock()
	configs := c.Exchanges
	m.Unlock()
	return configs
}

// GetExchangeConfig returns exchange configurations by its individual name
func (c *Config) GetExchangeConfig(name string) (*Exchange, error) {
	m.Lock()
	defer m.Unlock()
	for i := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[i].Name, name) {
			return &c.Exchanges[i], nil
		}
	}
	return nil, fmt.Errorf("%s %w", name, ErrExchangeNotFound)
}

// UpdateExchangeConfig updates exchange configurations
func (c *Config) UpdateExchangeConfig(e *Exchange) error {
	m.Lock()
	defer m.Unlock()
	for i := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[i].Name, e.Name) {
			c.Exchanges[i] = *e
			return nil
		}
	}
	return fmt.Errorf("%s %w", e.Name, ErrExchangeNotFound)
}

// CheckExchangeConfigValues returns configuration values for all enabled
// exchanges
func (c *Config) CheckExchangeConfigValues() error {
	if len(c.Exchanges) == 0 {
		return errNoEnabledExchanges
	}

	exchanges := 0
	for i := range c.Exchanges {
		e := &c.Exchanges[i]

		// Check to see if the old API storage format is used
		if e.APIKey != nil {
			// It is, migrate settings to new format
			e.API.AuthenticatedSupport = *e.AuthenticatedAPISupport
			if e.AuthenticatedWebsocketAPISupport != nil {
				e.API.AuthenticatedWebsocketSupport = *e.AuthenticatedWebsocketAPISupport
			}
			e.API.Credentials.Key = *e.APIKey
			e.API.Credentials.Secret = *e.APISecret

			if e.APIAuthPEMKey != nil {
				e.API.Credentials.PEMKey = *e.APIAuthPEMKey
			}

			if e.APIAuthPEMKeySupport != nil {
				e.API.PEMKeySupport = *e.APIAuthPEMKeySupport
			}

			if e.ClientID != nil {
				e.API.Credentials.ClientID = *e.ClientID
			}

			// Flush settings
			e.AuthenticatedAPISupport = nil
			e.AuthenticatedWebsocketAPISupport = nil
			e.APIKey = nil
			e.APISecret = nil
			e.ClientID = nil
			e.APIAuthPEMKeySupport = nil
			e.APIAuthPEMKey = nil
			e.APIURL = nil
			e.APIURLSecondary = nil
			e.WebsocketURL = nil
		}

		if e.Features == nil {
			e.Features = &FeaturesConfig{}
		}

		if e.SupportsAutoPairUpdates != nil {
			e.Features.Supports.RESTCapabilities.AutoPairUpdates = *e.SupportsAutoPairUpdates
			e.Features.Enabled.AutoPairUpdates = *e.SupportsAutoPairUpdates
			e.SupportsAutoPairUpdates = nil
		}

		if e.Websocket != nil {
			e.Features.Enabled.Websocket = *e.Websocket
			e.Websocket = nil
		}

		if err := e.CurrencyPairs.SetDelimitersFromConfig(); err != nil {
			return fmt.Errorf("%s: %w", e.Name, err)
		}

		assets := e.CurrencyPairs.GetAssetTypes(false)
		if len(assets) == 0 {
			e.Enabled = false
			log.Warnf(log.ConfigMgr, "%s no assets found, disabling...", e.Name)
			continue
		}

		if enabled := e.CurrencyPairs.GetAssetTypes(true); len(enabled) == 0 {
			// turn on an asset if all disabled
			log.Warnf(log.ConfigMgr, "%s assets disabled, turning on asset %s", e.Name, assets[0])
			if err := e.CurrencyPairs.SetAssetEnabled(assets[0], true); err != nil {
				return err
			}
		}

		if !e.Enabled {
			continue
		}
		if e.Name == "" {
			log.Errorf(log.ConfigMgr, "%s: #%d", common.ErrExchangeNameNotSet, i)
			e.Enabled = false
			continue
		}
		if (e.API.AuthenticatedSupport || e.API.AuthenticatedWebsocketSupport) &&
			e.API.CredentialsValidator != nil {
			var failed bool
			if e.API.CredentialsValidator.RequiresKey &&
				(e.API.Credentials.Key == "" || e.API.Credentials.Key == DefaultAPIKey) {
				failed = true
			}

			if e.API.CredentialsValidator.RequiresSecret &&
				(e.API.Credentials.Secret == "" || e.API.Credentials.Secret == DefaultAPISecret) {
				failed = true
			}

			if e.API.CredentialsValidator.RequiresClientID &&
				(e.API.Credentials.ClientID == DefaultAPIClientID || e.API.Credentials.ClientID == "") {
				failed = true
			}

			if failed {
				e.API.AuthenticatedSupport = false
				e.API.AuthenticatedWebsocketSupport = false
				log.Warnf(log.ConfigMgr, warningExchangeAuthAPIDefaultOrEmptyValues, e.Name)
			}
		}
		if !e.Features.Supports.RESTCapabilities.AutoPairUpdates &&
			!e.Features.Supports.WebsocketCapabilities.AutoPairUpdates {
			lastUpdated := time.Unix(e.CurrencyPairs.LastUpdated, 0)
			lastUpdated = lastUpdated.AddDate(0, 0, pairsLastUpdatedWarningThreshold)
			if lastUpdated.Unix() <= time.Now().Unix() {
				log.Warnf(log.ConfigMgr,
					warningPairsLastUpdatedThresholdExceeded,
					e.Name,
					pairsLastUpdatedWarningThreshold)
			}
		}
		if e.HTTPTimeout <= 0 {
			log.Warnf(log.ConfigMgr,
				"Exchange %s HTTP Timeout value not set, defaulting to %v.\n",
				e.Name,
				defaultHTTPTimeout)
			e.HTTPTimeout = defaultHTTPTimeout
		}

		if e.WebsocketResponseCheckTimeout <= 0 {
			log.Warnf(log.ConfigMgr,
				"Exchange %s Websocket response check timeout value not set, defaulting to %v.",
				e.Name,
				DefaultWebsocketResponseCheckTimeout)
			e.WebsocketResponseCheckTimeout = DefaultWebsocketResponseCheckTimeout
		}

		if e.WebsocketResponseMaxLimit <= 0 {
			log.Warnf(log.ConfigMgr,
				"Exchange %s Websocket response max limit value not set, defaulting to %v.",
				e.Name,
				DefaultWebsocketResponseMaxLimit)
			e.WebsocketResponseMaxLimit = DefaultWebsocketResponseMaxLimit
		}
		if e.WebsocketTrafficTimeout <= 0 {
			log.Warnf(log.ConfigMgr,
				"Exchange %s Websocket response traffic timeout value not set, defaulting to %v.",
				e.Name,
				DefaultWebsocketTrafficTimeout)
			e.WebsocketTrafficTimeout = DefaultWebsocketTrafficTimeout
		}
		if e.Orderbook.WebsocketBufferLimit <= 0 {
			log.Warnf(log.ConfigMgr,
				"Exchange %s Websocket orderbook buffer limit value not set, defaulting to %v.",
				e.Name,
				defaultWebsocketOrderbookBufferLimit)
			e.Orderbook.WebsocketBufferLimit = defaultWebsocketOrderbookBufferLimit
		}
		err := c.CheckPairConsistency(e.Name)
		if err != nil {
			log.Errorf(log.ConfigMgr,
				"Exchange %s: CheckPairConsistency error: %s\n",
				e.Name,
				err)
			e.Enabled = false
			continue
		}
		for x := range e.BankAccounts {
			if !e.BankAccounts[x].Enabled {
				continue
			}
			err := e.BankAccounts[x].Validate()
			if err != nil {
				e.BankAccounts[x].Enabled = false
				log.Warnln(log.ConfigMgr, err.Error())
			}
		}

		exchanges++
	}

	if exchanges == 0 {
		return errNoEnabledExchanges
	}

	return nil
}

// CheckBankAccountConfig checks all bank accounts to see if they are valid
func (c *Config) CheckBankAccountConfig() {
	for x := range c.BankAccounts {
		if c.BankAccounts[x].Enabled {
			err := c.BankAccounts[x].Validate()
			if err != nil {
				c.BankAccounts[x].Enabled = false
				log.Warnln(log.ConfigMgr, err.Error())
			}
		}
	}
	banking.SetAccounts(c.BankAccounts...)
}

// GetForexProviders returns a list of available forex providers
func (c *Config) GetForexProviders() []currency.FXSettings {
	m.Lock()
	fxProviders := c.Currency.ForexProviders
	m.Unlock()
	return fxProviders
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

// forexProviderExists checks to see if the provider exist.
func (c *Config) forexProviderExists(name string) bool {
	for i := range c.Currency.ForexProviders {
		if strings.EqualFold(c.Currency.ForexProviders[i].Name, name) {
			return true
		}
	}
	return false
}

// CheckCurrencyConfigValues checks to see if the currency config values are
// correct or not
func (c *Config) CheckCurrencyConfigValues() error {
	supported := forexprovider.GetSupportedForexProviders()
	for x := range supported {
		if !c.forexProviderExists(supported[x]) {
			log.Warnf(log.ConfigMgr, "%s forex provider not found, adding to config...\n", supported[x])
			c.Currency.ForexProviders = append(c.Currency.ForexProviders,
				currency.FXSettings{
					Name:      supported[x],
					APIKey:    DefaultUnsetAPIKey,
					APIKeyLvl: -1,
				})
		}
	}

	for i := range c.Currency.ForexProviders {
		if !common.StringSliceContainsInsensitive(supported, c.Currency.ForexProviders[i].Name) {
			log.Warnf(log.ConfigMgr,
				"%s forex provider not supported, please remove from config.\n",
				c.Currency.ForexProviders[i].Name)
			c.Currency.ForexProviders[i].Enabled = false
		}
	}

	if c.Currency.CryptocurrencyProvider == (currency.Provider{}) {
		c.Currency.CryptocurrencyProvider.Name = "CoinMarketCap"
		c.Currency.CryptocurrencyProvider.Enabled = false
		c.Currency.CryptocurrencyProvider.Verbose = false
		c.Currency.CryptocurrencyProvider.AccountPlan = DefaultUnsetAccountPlan
		c.Currency.CryptocurrencyProvider.APIKey = DefaultUnsetAPIKey
	}

	if c.Currency.CryptocurrencyProvider.APIKey == "" {
		c.Currency.CryptocurrencyProvider.APIKey = DefaultUnsetAPIKey
	}
	if c.Currency.CryptocurrencyProvider.AccountPlan == "" {
		c.Currency.CryptocurrencyProvider.AccountPlan = DefaultUnsetAccountPlan
	}

	if c.Currency.CurrencyPairFormat == nil {
		if c.CurrencyPairFormat != nil {
			c.Currency.CurrencyPairFormat = c.CurrencyPairFormat
			c.CurrencyPairFormat = nil
		} else {
			c.Currency.CurrencyPairFormat = &currency.PairFormat{
				Delimiter: "-",
				Uppercase: true,
			}
		}
	}

	if c.Currency.FiatDisplayCurrency.IsEmpty() {
		if c.FiatDisplayCurrency != nil {
			c.Currency.FiatDisplayCurrency = *c.FiatDisplayCurrency
			c.FiatDisplayCurrency = nil
		} else {
			c.Currency.FiatDisplayCurrency = currency.USD
		}
	}

	// Flush old setting which still exists
	if c.FiatDisplayCurrency != nil {
		c.FiatDisplayCurrency = nil
	}

	if c.Currency.CurrencyFileUpdateDuration <= 0 {
		log.Warnf(log.ConfigMgr, "Currency file update duration invalid, defaulting to %s", currency.DefaultCurrencyFileDelay)
		c.Currency.CurrencyFileUpdateDuration = currency.DefaultCurrencyFileDelay
	}

	if c.Currency.ForeignExchangeUpdateDuration <= 0 {
		log.Warnf(log.ConfigMgr, "Currency foreign exchange update duration invalid, defaulting to %s", currency.DefaultForeignExchangeDelay)
		c.Currency.ForeignExchangeUpdateDuration = currency.DefaultForeignExchangeDelay
	}

	return nil
}

// CheckLoggerConfig checks to see logger values are present and valid in config
// if not creates a default instance of the logger
func (c *Config) CheckLoggerConfig() error {
	m.Lock()
	defer m.Unlock()

	if c.Logging.Enabled == nil || c.Logging.Output == "" {
		c.Logging = *log.GenDefaultSettings()
	}

	if c.Logging.AdvancedSettings.ShowLogSystemName == nil {
		c.Logging.AdvancedSettings.ShowLogSystemName = convert.BoolPtr(false)
	}

	if c.Logging.LoggerFileConfig != nil {
		if c.Logging.LoggerFileConfig.FileName == "" {
			c.Logging.LoggerFileConfig.FileName = "log.txt"
		}
		if c.Logging.LoggerFileConfig.Rotate == nil {
			c.Logging.LoggerFileConfig.Rotate = convert.BoolPtr(false)
		}
		if c.Logging.LoggerFileConfig.MaxSize <= 0 {
			log.Warnf(log.ConfigMgr, "Logger rotation size invalid, defaulting to %v", log.DefaultMaxFileSize)
			c.Logging.LoggerFileConfig.MaxSize = log.DefaultMaxFileSize
		}
		log.SetFileLoggingState( /*Is correctly configured*/ true)
	}

	err := log.SetGlobalLogConfig(&c.Logging)
	if err != nil {
		return err
	}

	logPath := c.GetDataPath("logs")
	err = common.CreateDir(logPath)
	if err != nil {
		return err
	}
	return log.SetLogPath(logPath)
}

func (c *Config) checkGCTScriptConfig() error {
	m.Lock()
	defer m.Unlock()

	if c.GCTScript.ScriptTimeout <= 0 {
		c.GCTScript.ScriptTimeout = gctscript.DefaultTimeoutValue
	}

	if c.GCTScript.MaxVirtualMachines == 0 {
		c.GCTScript.MaxVirtualMachines = gctscript.DefaultMaxVirtualMachines
	}

	scriptPath := c.GetDataPath("scripts")
	err := common.CreateDir(scriptPath)
	if err != nil {
		return err
	}

	outputPath := filepath.Join(scriptPath, "output")
	err = common.CreateDir(outputPath)
	if err != nil {
		return err
	}

	gctscript.ScriptPath = scriptPath

	return nil
}

func (c *Config) checkDatabaseConfig() error {
	m.Lock()
	defer m.Unlock()

	if (c.Database == database.Config{}) {
		c.Database.Driver = database.DBSQLite3
		c.Database.Database = database.DefaultSQLiteDatabase
	}

	if !c.Database.Enabled {
		return nil
	}

	if !slices.Contains(database.SupportedDrivers, c.Database.Driver) {
		c.Database.Enabled = false
		return fmt.Errorf("unsupported database driver %v, database disabled", c.Database.Driver)
	}

	if c.Database.Driver == database.DBSQLite || c.Database.Driver == database.DBSQLite3 {
		databaseDir := c.GetDataPath("database")
		err := common.CreateDir(databaseDir)
		if err != nil {
			return err
		}
		database.DB.DataPath = databaseDir
	}

	return database.DB.SetConfig(&c.Database)
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
		log.Warnln(log.ConfigMgr, "NTPClient enabled with no servers configured, enabling default pool.")
		c.NTPClient.Pool = []string{"pool.ntp.org:123"}
	}
}

// SetNTPCheck allows the user to change how they are prompted for timesync alerts
func (c *Config) SetNTPCheck(input io.Reader) (string, error) {
	m.Lock()
	defer m.Unlock()

	reader := bufio.NewReader(input)
	fmt.Println("Your system time is out of sync, this may cause issues with trading")
	fmt.Println("How would you like to show future notifications? (a)lert at startup / (w)arn periodically / (d)isable")

	var resp string
	answered := false
	for !answered {
		answer, err := reader.ReadString('\n')
		if err != nil {
			return resp, err
		}

		answer = strings.TrimRight(answer, "\r\n")
		switch answer {
		case "a":
			c.NTPClient.Level = 0
			resp = "Time sync has been set to alert"
			answered = true
		case "w":
			c.NTPClient.Level = 1
			resp = "Time sync has been set to warn only"
			answered = true
		case "d":
			c.NTPClient.Level = -1
			resp = "Future notifications for out of time sync has been disabled"
			answered = true
		default:
			fmt.Println("Invalid option selected, please try again (a)lert / (w)arn / (d)isable")
		}
	}
	return resp, nil
}

// CheckDataHistoryMonitorConfig ensures the data history config is
// valid, or sets default values
func (c *Config) CheckDataHistoryMonitorConfig() {
	m.Lock()
	defer m.Unlock()
	if c.DataHistoryManager.CheckInterval <= 0 {
		c.DataHistoryManager.CheckInterval = defaultDataHistoryMonitorCheckTimer
	}
	if c.DataHistoryManager.MaxJobsPerCycle == 0 {
		c.DataHistoryManager.MaxJobsPerCycle = defaultMaxJobsPerCycle
	}
}

// CheckCurrencyStateManager ensures the currency state config is valid, or sets
// default values
func (c *Config) CheckCurrencyStateManager() {
	m.Lock()
	defer m.Unlock()
	if c.CurrencyStateManager.Delay <= 0 {
		c.CurrencyStateManager.Delay = defaultCurrencyStateManagerDelay
	}
	if c.CurrencyStateManager.Enabled == nil { // default on, when being upgraded
		c.CurrencyStateManager.Enabled = convert.BoolPtr(true)
	}
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

// DefaultFilePath returns the default config file path
// MacOS/Linux: $HOME/.gocryptotrader/config.json or config.dat
// Windows: %APPDATA%\GoCryptoTrader\config.json or config.dat
// Helpful for printing application usage
func DefaultFilePath() string {
	foundConfig, _, err := GetFilePath("")
	if err != nil {
		// If there was no config file, show default location for .json
		return filepath.Join(common.GetDefaultDataDir(runtime.GOOS), File)
	}
	return foundConfig
}

// GetAndMigrateDefaultPath returns the target config file
// migrating it from the old default location to new one,
// if it was implicitly loaded from a default location and
// wasn't already in the correct 'new' default location
func GetAndMigrateDefaultPath(configFile string) (string, error) {
	filePath, wasDefault, err := GetFilePath(configFile)
	if err != nil {
		return "", err
	}
	if wasDefault {
		return migrateConfig(filePath, common.GetDefaultDataDir(runtime.GOOS))
	}
	return filePath, nil
}

// GetFilePath returns the desired config file or the default config file name
// and whether it was loaded from a default location (rather than explicitly specified)
func GetFilePath(configFile string) (configPath string, isImplicitDefaultPath bool, err error) {
	if configFile != "" {
		return configFile, false, nil
	}

	exePath, err := common.GetExecutablePath()
	if err != nil {
		return "", false, err
	}
	newDir := common.GetDefaultDataDir(runtime.GOOS)
	defaultPaths := []string{
		filepath.Join(exePath, File),
		filepath.Join(exePath, EncryptedFile),
		filepath.Join(newDir, File),
		filepath.Join(newDir, EncryptedFile),
	}

	for _, p := range defaultPaths {
		if file.Exists(p) {
			configFile = p
			break
		}
	}
	if configFile == "" {
		return "", false, fmt.Errorf("config.json file not found in %s, please follow README.md in root dir for config generation",
			newDir)
	}

	return configFile, true, nil
}

// migrateConfig will move the config file to the target
// config directory as `File` or `EncryptedFile` depending on whether the config
// is encrypted
func migrateConfig(configFile, targetDir string) (string, error) {
	var target string
	if IsFileEncrypted(configFile) {
		target = EncryptedFile
	} else {
		target = File
	}
	target = filepath.Join(targetDir, target)
	if configFile == target {
		return configFile, nil
	}
	if file.Exists(target) {
		log.Warnf(log.ConfigMgr, "Config file already found in %q; not overwriting, defaulting to %s", target, configFile)
		return configFile, nil
	}

	if err := file.Move(configFile, target); err != nil {
		return "", err
	}

	return target, nil
}

// ReadConfigFromFile loads Config from the path
// If encrypted, prompts for encryption key
// Unless dryrun checks if the configuration needs to be encrypted and resaves, prompting for key
func (c *Config) ReadConfigFromFile(path string, dryrun bool) error {
	var err error
	path, _, err = GetFilePath(path)
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := c.readConfig(f); err != nil {
		return err
	}

	if dryrun || c.EncryptConfig != fileEncryptionPrompt || IsFileEncrypted(path) {
		return nil
	}

	return c.saveWithEncryptPrompt(path)
}

// readConfig loads config from a io.Reader into the config object
// versions manager will upgrade/downgrade if appropriate
// If encrypted, prompts for encryption key
func (c *Config) readConfig(d io.Reader) error {
	j, err := io.ReadAll(d)
	if err != nil {
		return err
	}

	if IsEncrypted(j) {
		if j, err = c.decryptConfig(j); err != nil {
			return err
		}
	}

	if j, err = versions.Manager.Deploy(context.Background(), j, versions.UseLatestVersion); err != nil {
		return err
	}

	return json.Unmarshal(j, c)
}

// saveWithEncryptPrompt will prompt the user if they want to encrypt their config
// If they agree, c.EncryptConfig is set to Enabled, the config is encrypted and saved
// Otherwise, c.EncryptConfig is set to Disabled and the file is resaved
func (c *Config) saveWithEncryptPrompt(path string) error {
	if confirm, err := promptForConfigEncryption(os.Stdin); err != nil {
		return nil //nolint:nilerr // Ignore encryption prompt failures; The user will be prompted again
	} else if confirm {
		c.EncryptConfig = fileEncryptionEnabled
		return c.SaveConfigToFile(path)
	}

	c.EncryptConfig = fileEncryptionDisabled
	return c.SaveConfigToFile(path)
}

// decryptConfig reads encrypted configuration and requests key from provider
func (c *Config) decryptConfig(j []byte) ([]byte, error) {
	for range maxAuthFailures {
		f := c.EncryptionKeyProvider
		if f == nil {
			f = PromptForConfigKey
		}
		key, err := f(false)
		if err != nil {
			log.Errorf(log.ConfigMgr, "PromptForConfigKey err: %s", err)
			continue
		}
		d, err := c.decryptConfigData(j, key)
		if err != nil {
			log.Errorln(log.ConfigMgr, "Could not decrypt and deserialise data with given key. Invalid password?", err)
			continue
		}
		return d, nil
	}
	return nil, errDecryptFailed
}

// SaveConfigToFile saves your configuration to your desired path as a JSON object.
// The function encrypts the data and prompts for encryption key, if necessary
func (c *Config) SaveConfigToFile(configPath string) error {
	defaultPath, _, err := GetFilePath(configPath)
	if err != nil {
		return err
	}
	var writer *os.File
	provider := func() (io.Writer, error) {
		writer, err = file.Writer(defaultPath)
		return writer, err
	}
	defer func() {
		if writer != nil {
			err = writer.Close()
			if err != nil {
				log.Errorln(log.ConfigMgr, err)
			}
		}
	}()
	return c.Save(provider)
}

// Save saves your configuration to the writer as a JSON object with encryption, if configured
// If there is an error when preparing the data to store, the writer is never requested
func (c *Config) Save(writerProvider func() (io.Writer, error)) error {
	payload, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return err
	}

	if c.EncryptConfig == fileEncryptionEnabled {
		// Ensure we have the key from session or from user
		if len(c.sessionDK) == 0 {
			f := c.EncryptionKeyProvider
			if f == nil {
				f = PromptForConfigKey
			}
			var key, sessionDK, storedSalt []byte
			if key, err = f(true); err != nil {
				return err
			}
			if sessionDK, storedSalt, err = makeNewSessionDK(key); err != nil {
				return err
			}
			c.sessionDK, c.storedSalt = sessionDK, storedSalt
		}
		payload, err = c.encryptConfigData(payload)
		if err != nil {
			return err
		}
	}
	configWriter, err := writerProvider()
	if err != nil {
		return err
	}
	_, err = io.Copy(configWriter, bytes.NewReader(payload))
	return err
}

func setDefaultIfZeroWarn[T comparable](scope, name string, p *T, def T) {
	if common.SetIfZero(p, def) {
		log.Warnf(log.ConfigMgr, "%s field %q not set, defaulting to `%v`", scope, name, def)
	}
}

// CheckRemoteControlConfig checks and sets default values for the remote control config
func (c *Config) CheckRemoteControlConfig() {
	m.Lock()
	defer m.Unlock()

	setDefaultIfZeroWarn("Remote control", "username", &c.RemoteControl.Username, DefaultGRPCUsername)
	setDefaultIfZeroWarn("Remote control", "password", &c.RemoteControl.Password, DefaultGRPCPassword)
	setDefaultIfZeroWarn("Remote control gRPC", "listen address", &c.RemoteControl.GRPC.ListenAddress, "localhost:9052")
	setDefaultIfZeroWarn("Remote control gRPC", "gRPC proxy listen address", &c.RemoteControl.GRPC.GRPCProxyListenAddress, "localhost:9053")

	if c.RemoteControl.GRPC.GRPCProxyEnabled && !c.RemoteControl.GRPC.Enabled {
		log.Warnln(log.ConfigMgr, "gRPC proxy cannot be enabled when gRPC is disabled, disabling gRPC proxy")
		c.RemoteControl.GRPC.GRPCProxyEnabled = false
	}
}

// CheckConfig checks all config settings
func (c *Config) CheckConfig() error {
	if err := c.CheckLoggerConfig(); err != nil {
		log.Errorf(log.ConfigMgr, "Failed to configure logger, some logging features unavailable: %s\n", err)
	}

	if err := c.checkDatabaseConfig(); err != nil {
		log.Errorf(log.DatabaseMgr, "Failed to configure database: %v", err)
	}

	if err := c.CheckExchangeConfigValues(); err != nil {
		return fmt.Errorf("%w: %w", errCheckingConfigValues, err)
	}

	if err := c.checkGCTScriptConfig(); err != nil {
		log.Errorf(log.ConfigMgr, "Failed to configure gctscript, feature has been disabled: %s\n", err)
	}

	c.CheckConnectionMonitorConfig()
	c.CheckDataHistoryMonitorConfig()
	c.CheckCurrencyStateManager()
	c.CheckCommunicationsConfig()
	c.CheckClientBankAccounts()
	c.CheckBankAccountConfig()
	c.CheckRemoteControlConfig()
	c.CheckSyncManagerConfig()

	if err := c.CheckCurrencyConfigValues(); err != nil {
		return err
	}

	if c.GlobalHTTPTimeout <= 0 {
		log.Warnf(log.ConfigMgr, "Global HTTP Timeout value not set, defaulting to %v.\n", defaultHTTPTimeout)
		c.GlobalHTTPTimeout = defaultHTTPTimeout
	}

	if c.NTPClient.Level != 0 {
		c.CheckNTPConfig()
	}

	return nil
}

// LoadConfig loads your configuration file into your configuration object
func (c *Config) LoadConfig(configPath string, dryrun bool) error {
	err := c.ReadConfigFromFile(configPath, dryrun)
	if err != nil {
		return fmt.Errorf("%w (%s): %w", ErrFailureOpeningConfig, configPath, err)
	}
	return c.CheckConfig()
}

// UpdateConfig updates the config with a supplied config file
func (c *Config) UpdateConfig(configPath string, newCfg *Config, dryrun bool) error {
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
	c.Exchanges = newCfg.Exchanges

	if !dryrun {
		err = c.SaveConfigToFile(configPath)
		if err != nil {
			return err
		}
	}

	return c.LoadConfig(configPath, dryrun)
}

// GetConfig returns the global shared config instance
func GetConfig() *Config {
	m.Lock()
	defer m.Unlock()
	return &cfg
}

// SetConfig sets the global shared config instance
func SetConfig(c *Config) {
	m.Lock()
	defer m.Unlock()
	cfg = *c
}

// RemoveExchange removes an exchange config
func (c *Config) RemoveExchange(exchName string) bool {
	m.Lock()
	defer m.Unlock()

	for x := range c.Exchanges {
		if strings.EqualFold(c.Exchanges[x].Name, exchName) {
			c.Exchanges = slices.Delete(c.Exchanges, x, x+1)
			return true
		}
	}
	return false
}

// AssetTypeEnabled checks to see if the asset type is enabled in configuration
func (c *Config) AssetTypeEnabled(a asset.Item, exch string) (bool, error) {
	cfg, err := c.GetExchangeConfig(exch)
	if err != nil {
		return false, err
	}

	err = cfg.CurrencyPairs.IsAssetEnabled(a)
	if err != nil {
		return false, nil //nolint:nilerr // non-fatal error
	}
	return true, nil
}

// GetDataPath gets the data path for the given subpath
func (c *Config) GetDataPath(elem ...string) string {
	var baseDir string
	if c.DataDirectory != "" {
		baseDir = c.DataDirectory
	} else {
		baseDir = common.GetDefaultDataDir(runtime.GOOS)
	}
	return filepath.Join(append([]string{baseDir}, elem...)...)
}

// Validate checks if exchange config is valid
func (c *Exchange) Validate() error {
	if c == nil {
		return errExchangeConfigIsNil
	}

	if c.ConnectionMonitorDelay <= 0 {
		c.ConnectionMonitorDelay = DefaultConnectionMonitorDelay
	}

	return nil
}
