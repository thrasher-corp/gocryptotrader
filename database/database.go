package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	// lib/pq used for sqlboiler
	_ "github.com/lib/pq"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/database/models"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/portfolio"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"gopkg.in/volatiletech/null.v6"
)

// ORM is the overarching type across the database package that handles database
// connections and relational mapping
type ORM struct {
	Exec          *sql.DB
	Verbose       bool
	Connected     bool
	ConfigID      int64
	ExchangeID    map[string]int64
	InsertCounter map[string]int64
	sync.Mutex
}

// NewORMConnection makes a connection to the database and returns a pointer
// ORM object
func NewORMConnection(databaseName, host, user, password string, verbose bool) (*ORM, error) {
	dbORM := new(ORM)
	dbORM.Verbose = verbose
	return dbORM, dbORM.SetupConnection(databaseName, host, user, password)
}

// SetupConnection starts the connection to the GoCryptoTrader database
func (o *ORM) SetupConnection(databaseName, host, user, password string) error {
	db, err := sql.Open("postgres",
		fmt.Sprintf("dbname=%s host=%s user=%s password=%s",
			databaseName, host, user, password))
	if err != nil {
		return err
	}
	o.Exec = db

	err = o.Exec.Ping()
	if err != nil {
		return err
	}
	o.SetInsertCounter()
	o.Connected = true
	return nil
}

// Connect connects to a db and loads a specific configuration
func Connect(databaseName, host, user, password, configName string, verbose bool) (*ORM, error) {
	dbORM := new(ORM)
	dbORM.Verbose = verbose

	var err error
	dbORM.Exec, err = sql.Open("postgres",
		fmt.Sprintf("dbname=%s host=%s user=%s password=%s",
			databaseName, host, user, password))
	if err != nil {
		return nil, err
	}

	err = dbORM.Exec.Ping()
	if err != nil {
		return nil, err
	}
	dbORM.SetInsertCounter()
	dbORM.Connected = true
	err = dbORM.LoadConfiguration(configName)
	if err != nil {
		return nil, err
	}

	return dbORM, nil
}

// SetExchangeMap sets exchange map
func (o *ORM) SetExchangeMap() error {
	o.Lock()
	defer o.Unlock()

	exchanges, err := models.Exchanges(o.Exec).All()
	if err != nil {
		return err
	}

	if len(exchanges) < 1 {
		return errors.New("No exchanges loaded into database")
	}

	o.ExchangeID = make(map[string]int64)

	for i := range exchanges {
		o.ExchangeID[exchanges[i].ExchangeName] = exchanges[i].ExchangeID
	}
	return nil
}

// SetInsertCounter allows for the minimization of db calls for potential insert
// intensive functions
func (o *ORM) SetInsertCounter() {
	o.InsertCounter = make(map[string]int64)
	o.InsertCounter["exchangeTradeHistories"] = models.ExchangeTradeHistories(o.Exec).CountP()
	o.InsertCounter["orderHistories"] = models.OrderHistories(o.Exec).CountP()
	o.InsertCounter["taxableEvents"] = models.TaxableEvents(o.Exec).CountP()
}

// LoadConfiguration loads a configuration that has already been loaded
func (o *ORM) LoadConfiguration(configName string) error {
	if !o.Connected {
		if o.Verbose {
			log.Println("cannot load configuration, no database connnection")
		}
		return nil
	}

	if o.checkLoadedConfiguration(configName) {
		var err error
		o.ConfigID, err = o.getLoadedConfigurationID(configName)
		if err != nil {
			return err
		}
		return o.SetExchangeMap()
	}
	return fmt.Errorf("database error could not find loaded configuration %s", configName)
}

// InsertNewConfiguration inserts a new configuration
func (o *ORM) InsertNewConfiguration(cfg *config.Config, password string) error {
	if !o.Connected {
		if o.Verbose {
			log.Println("cannot insert new configuration, no database connnection")
		}
		return nil
	}

	if o.checkLoadedConfiguration(cfg.Name) {
		return errors.New("configuration already loaded")
	}
	err := o.insertMainConfiguration(cfg, password)
	if err != nil {
		return err
	}
	err = o.insertExchangeConfigurations(cfg.Exchanges)
	if err != nil {
		return err
	}
	err = o.insertPortfolioConfiguration(cfg.Portfolio)
	if err != nil {
		return err
	}
	err = o.insertSMSGlobalConfiguration(cfg.SMS.Username, cfg.SMS.Password, cfg.SMS.Enabled)
	if err != nil {
		return err
	}
	if cfg.CurrencyPairFormat == nil {
		return errors.New("config currencypair is nil")
	}
	err = o.insertCurrencyPairConfiguration("config",
		cfg.CurrencyPairFormat.Delimiter,
		cfg.CurrencyPairFormat.Separator,
		cfg.CurrencyPairFormat.Index,
		0,
		cfg.CurrencyPairFormat.Uppercase)
	if err != nil {
		return err
	}
	return nil
}

// UpdateConfiguration updates an old configuration to new settings
func (o *ORM) UpdateConfiguration(cfg *config.Config) error {
	if !o.Connected {
		if o.Verbose {
			log.Println("cannot insert new configuration, no database connnection")
		}
		return nil
	}

	if o.checkLoadedConfiguration(cfg.Name) {
		//NOTE do stuff
		return nil
	}
	return errors.New("not yet implemented")
}

// CheckLoadedConfiguration checks if a configuration has been loaded in the
// database
func (o *ORM) checkLoadedConfiguration(configName string) bool {
	o.Lock()
	defer o.Unlock()
	return models.Configs(o.Exec, qm.Where("name = ?", configName)).ExistsP()
}

// getLoadedConfigurationID returns ID of loaded configuration
func (o *ORM) getLoadedConfigurationID(configName string) (int64, error) {
	o.Lock()
	defer o.Unlock()
	cfg, err := models.Configs(o.Exec, qm.Where("name = ?", configName)).One()
	if err != nil {
		return 0, err
	}
	return cfg.ConfigID, nil
}

// insertMainConfiguration inserts a new configuration into the database
func (o *ORM) insertMainConfiguration(cfg *config.Config, password string) error {
	o.Lock()
	defer o.Unlock()

	if models.Configs(o.Exec, qm.Where("name = ?", cfg.Name)).ExistsP() {
		return fmt.Errorf("database error configuration already exists with the name %s", cfg.Name)
	}

	count := models.Configs(o.Exec).CountP()

	u := &models.Config{
		ConfigID:            count,
		Name:                cfg.Name,
		Password:            password,
		EncryptConfig:       cfg.EncryptConfig,
		Cryptocurrencies:    cfg.Cryptocurrencies,
		FiatDisplayCurrency: cfg.FiatDisplayCurrency,
		GlobalHTTPTimeout:   int64(cfg.GlobalHTTPTimeout),
	}
	return u.Insert(o.Exec)
}

// insertCurrencyPairConfiguration inserts currency pair information
func (o *ORM) insertCurrencyPairConfiguration(use, delimiter, separator, index string, exchangeID int64, uppercase bool) error {
	o.Lock()
	defer o.Unlock()

	count, err := models.CurrencyPairFormats(o.Exec).Count()
	if err != nil {
		return err
	}

	cpf := &models.CurrencyPairFormat{
		CurrencyPairFormatID: count,
		Name:                 use,
		ExchangeID:           exchangeID,
		Uppercase:            uppercase,
		Delimiter:            delimiter,
		Separator:            separator,
		Index:                index,
	}

	return cpf.Insert(o.Exec)
}

// insertPortfolioConfiguration inserts new portfolio configurations
func (o *ORM) insertPortfolioConfiguration(cfg portfolio.Base) error {
	o.Lock()
	defer o.Unlock()

	count, err := models.Portfolios(o.Exec).Count()
	if err != nil {
		return err
	}

	for i := range cfg.Addresses {
		count++
		p := &models.Portfolio{
			PortfolioID: count,
			ConfigID:    o.ConfigID,
			CoinAddress: cfg.Addresses[i].Address,
			CoinType:    cfg.Addresses[i].CoinType,
			Balance:     cfg.Addresses[i].Balance,
			Description: cfg.Addresses[i].Description,
		}
		err := p.Insert(o.Exec)
		if err != nil {
			return err
		}
	}
	return nil
}

// insertSMSGlobalConfiguration inserts new SMSGlobal configurations
func (o *ORM) insertSMSGlobalConfiguration(username, password string, enabled bool) error {
	o.Lock()
	defer o.Unlock()

	count, err := models.Smsglobals(o.Exec).Count()
	if err != nil {
		return err
	}

	smsglobal := &models.Smsglobal{
		SmsglobalID: count,
		ConfigID:    o.ConfigID,
		Enabled:     enabled,
		Username:    username,
		Password:    password,
	}

	return smsglobal.Insert(o.Exec)
}

// InsertExchangeConfigurations loads the exchange configurations
func (o *ORM) insertExchangeConfigurations(cfg []config.ExchangeConfig) error {
	o.Lock()
	defer o.Unlock()

	o.ExchangeID = make(map[string]int64)

	for i := range cfg {
		if models.Exchanges(o.Exec, qm.Where("exchange_name = ?", cfg[i].Name)).ExistsP() {
			continue
		}
		e := &models.Exchange{
			ExchangeID:               int64(i),
			ConfigID:                 o.ConfigID,
			ExchangeName:             cfg[i].Name,
			Enabled:                  cfg[i].Enabled,
			IsVerbose:                cfg[i].Verbose,
			Websocket:                cfg[i].Websocket,
			UseSandbox:               cfg[i].UseSandbox,
			RestPollingDelay:         int64(cfg[i].RESTPollingDelay),
			HTTPTimeout:              int64(cfg[i].HTTPTimeout),
			AuthenticatedAPISupport:  cfg[i].AuthenticatedAPISupport,
			APIKey:                   null.NewString(cfg[i].APIKey, true),
			APISecret:                null.NewString(cfg[i].APISecret, true),
			ClientID:                 null.NewString(cfg[i].ClientID, true),
			AvailablePairs:           cfg[i].AvailablePairs,
			EnabledPairs:             cfg[i].EnabledPairs,
			BaseCurrencies:           cfg[i].BaseCurrencies,
			AssetTypes:               null.NewString(cfg[i].AssetTypes, true),
			SupportedAutoPairUpdates: cfg[i].SupportsAutoPairUpdates,
			PairsLastUpdated:         time.Unix(cfg[i].PairsLastUpdated, 0),
		}
		err := e.Insert(o.Exec)
		if err != nil {
			return err
		}
		o.ExchangeID[cfg[i].Name] = int64(i)
	}
	return nil
}

// InsertExchangeTradeHistoryData inserts historic trade data
func (o *ORM) InsertExchangeTradeHistoryData(transactionID int64, exchangeName, currencyPair, assetType, orderType string, amount, rate float64, fulfilledOn time.Time) error {
	o.Lock()
	defer o.Unlock()

	if !o.Connected {
		if o.Verbose {
			log.Println("cannot exchange history data, no database connnection")
		}
		return nil
	}

	dataExists, err := models.ExchangeTradeHistories(o.Exec,
		qm.Where("exchange_id = ?", o.ExchangeID[exchangeName]),
		qm.And("fulfilled_on = ?", fulfilledOn),
		qm.And("currency_pair = ?", currencyPair),
		qm.And("asset_type = ?", assetType),
		qm.And("amount = ?", amount),
		qm.And("rate = ?", rate)).Exists()
	if err != nil {
		return err
	}

	if dataExists {
		log.Println("row already found")
		return nil
	}

	th := &models.ExchangeTradeHistory{
		ExchangeTradeHistoryID: o.InsertCounter["exchangeTradeHistories"],
		ConfigID:               o.ConfigID,
		ExchangeID:             o.ExchangeID[exchangeName],
		FulfilledOn:            fulfilledOn,
		CurrencyPair:           currencyPair,
		AssetType:              assetType,
		OrderType:              orderType,
		Amount:                 amount,
		Rate:                   rate,
	}

	if err := th.Insert(o.Exec); err != nil {
		return err
	}

	o.InsertCounter["exchangeTradeHistories"]++
	return nil
}

// GetEnabledExchanges returns enabled exchanges
func (o *ORM) GetEnabledExchanges() ([]string, error) {
	o.Lock()
	defer o.Unlock()

	if !o.Connected {
		if o.Verbose {
			log.Println("cannot get exchange data, no database connnection")
		}
		return nil, errors.New("no database connection")
	}

	exchanges, err := models.Exchanges(o.Exec, qm.Where("enabled = ?", true)).All()
	if err != nil {
		return nil, err
	}
	var enabledExchanges []string
	for i := range exchanges {
		enabledExchanges = append(enabledExchanges, exchanges[i].ExchangeName)
	}
	return enabledExchanges, nil
}

// GetExchangeTradeHistoryLast returns the last updated time.Time value on a
// trade history item
func (o *ORM) GetExchangeTradeHistoryLast(exchangeName, currencyPair string) (time.Time, error) {
	o.Lock()
	defer o.Unlock()

	if !o.Connected {
		if o.Verbose {
			log.Println("cannot get order history data, no database connnection")
		}
		return time.Time{}, errors.New("no database connection")
	}

	result, err := models.ExchangeTradeHistories(o.Exec,
		qm.Where("exchange_id = ?", o.ExchangeID[exchangeName]),
		qm.And("currency_pair = ?", currencyPair),
		qm.OrderBy("fulfilled_on DESC"),
		qm.Limit(1)).One()
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}

	return result.FulfilledOn, nil
}

// GetExchangeTradeHistory returns the full trade history by exchange name,
// currency pair and asset class
func (o *ORM) GetExchangeTradeHistory(exchangeName, currencyPair, assetType string) ([]exchange.TradeHistory, error) {
	o.Lock()
	defer o.Unlock()

	exchangeID, ok := o.ExchangeID[exchangeName]
	if !ok {
		return nil, errors.New("exchange name not found or not enabled")
	}

	exchangeHistory, err := models.ExchangeTradeHistories(o.Exec,
		qm.Where("exchange_id = ?", exchangeID),
		qm.And("currency_pair = ?", currencyPair),
		qm.And("asset_type = ?", assetType)).All()
	if err != nil {
		return nil, err
	}

	if len(exchangeHistory) == 0 {
		return nil, errors.New("no exchange trade data could be found")
	}

	var allExchangeHistory []exchange.TradeHistory
	for i := range exchangeHistory {
		allExchangeHistory = append(allExchangeHistory,
			exchange.TradeHistory{
				Timestamp: exchangeHistory[i].FulfilledOn,
				TID:       exchangeHistory[i].ExchangeTradeHistoryID,
				Price:     exchangeHistory[i].Rate,
				Amount:    exchangeHistory[i].Amount})
	}
	return allExchangeHistory, nil
}

// InsertOrderHistoryData inserts order history
func (o *ORM) InsertOrderHistoryData(exchangeName, currencyPair, assetType, orderType string, amount, rate float64, fulfilledOn time.Time) error {
	o.Lock()
	defer o.Unlock()

	if !o.Connected {
		if o.Verbose {
			log.Println("cannot insert order history data, no database connnection")
		}
		return nil
	}

	th := &models.ExchangeTradeHistory{
		ExchangeTradeHistoryID: o.InsertCounter["orderHistories"],
		ConfigID:               o.ConfigID,
		ExchangeID:             o.ExchangeID[exchangeName],
		FulfilledOn:            time.Now(),
		CurrencyPair:           currencyPair,
		OrderType:              orderType,
		AssetType:              assetType,
		Amount:                 amount,
		Rate:                   rate,
	}

	if err := th.Insert(o.Exec); err != nil {
		return err
	}

	o.InsertCounter["orderHistories"]++
	return nil
}

// InsertTaxableEvents inserts a new taxable event
func (o *ORM) InsertTaxableEvents(from, to string, fromAmount, fromEquivVal, toAmount, toEquivVal, gainloss float64, eventTime time.Time) error {
	o.Lock()
	defer o.Unlock()

	if !o.Connected {
		if o.Verbose {
			log.Println("cannot insert taxable events, no database connnection")
		}
		return nil
	}

	te := &models.TaxableEvent{
		TaxableEventsID:                     o.InsertCounter["taxableEvents"],
		ConfigID:                            o.ConfigID,
		ConversionFrom:                      from,
		ConversionFromAmount:                fromAmount,
		ConversionFromAmountEquivalantValue: fromEquivVal,
		ConversionTo:                        to,
		ConversionToAmount:                  toAmount,
		ConversionToamountEquivalantValue:   toEquivVal,
		ConversionGainLoss:                  gainloss,
		DateAndTime:                         eventTime,
	}

	if err := te.Insert(o.Exec); err != nil {
		return err
	}

	o.InsertCounter["taxableEvents"]++
	return nil
}

// DatabaseFlush removes unused inserted data except loaded configurations
// might be used at a successful sigterm
func (o *ORM) DatabaseFlush() error {
	o.Lock()
	defer o.Unlock()

	if !o.Connected {
		if o.Verbose {
			log.Println("cannot flush database, no database connnection")
		}
		return nil
	}

	err := models.CurrencyPairFormats(o.Exec).DeleteAll()
	if err != nil {
		return err
	}
	err = models.Exchanges(o.Exec).DeleteAll()
	if err != nil {
		return err
	}
	err = models.Portfolios(o.Exec).DeleteAll()
	if err != nil {
		return err
	}
	err = models.Smsglobals(o.Exec).DeleteAll()
	if err != nil {
		return err
	}
	err = models.SmsglobalContacts(o.Exec).DeleteAll()
	if err != nil {
		return err
	}
	err = models.TaxableEvents(o.Exec).DeleteAll()
	if err != nil {
		return err
	}
	err = models.OrderHistories(o.Exec).DeleteAll()
	if err != nil {
		return err
	}
	err = models.ExchangeTradeHistories(o.Exec).DeleteAll()
	if err != nil {
		return err
	}
	err = models.Webservers(o.Exec).DeleteAll()
	if err != nil {
		return err
	}
	return models.Configs(o.Exec).DeleteAll()
}
