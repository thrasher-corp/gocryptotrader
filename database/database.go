package database

//go:generate sqlboiler sqlite3 --wipe --no-hooks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/database/models"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/volatiletech/null"
	_ "github.com/volatiletech/sqlboiler-sqlite3/driver"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

// ctx set global context
var ctx = context.Background()

// ORM is the overarching type across the database package that handles database
// connections and relational mapping
type ORM struct {
	DB     *sql.DB
	Config *config.Config

	Verbose   bool
	Connected bool

	ConfigID   int64
	ExchangeID map[string]int64

	sync.Mutex
}

// Connect connects to a db
func Connect(sqlite3Path string, verbose bool, cfg *config.Config) (*ORM, error) {
	if verbose {
		log.Printf("Opening connection to sqlite3 database using PATH: %s",
			sqlite3Path)
	}

	SQLiteDB := new(ORM)
	SQLiteDB.Config = cfg

	var err error
	SQLiteDB.DB, err = sql.Open("sqlite3", sqlite3Path)
	if err != nil {
		return nil, err
	}

	err = SQLiteDB.DB.Ping()
	if err != nil {
		return nil, err
	}

	SQLiteDB.ExchangeID = make(map[string]int64)
	SQLiteDB.Verbose = verbose

	SQLiteDB.Connected = true

	return SQLiteDB, nil
}

// LoadConfigurations loads configuration paramaters
func (o *ORM) LoadConfigurations() error {
	if err := o.UpdateMainConfig(); err != nil {
		return err
	}
	return o.UpdateExchangeConfigurations()
}

// UpdateMainConfig checks, inserts and updates configuration paramaters
func (o *ORM) UpdateMainConfig() error {
	config, err := models.Configs(qm.Where("config_name = ?",
		o.Config.Name)).One(ctx, o.DB)

	if err != nil {
		config = &models.Config{
			ConfigName:                        o.Config.Name,
			GlobalHTTPTimeout:                 o.Config.GlobalHTTPTimeout.Nanoseconds(),
			WebserverAdminPassword:            null.StringFrom(o.Config.Webserver.AdminPassword),
			WebserverAdminUsername:            null.StringFrom(o.Config.Webserver.AdminUsername),
			WebserverAllowInsecureOrigin:      o.Config.Webserver.WebsocketAllowInsecureOrigin,
			WebserverEnabled:                  o.Config.Webserver.Enabled,
			WebserverListenAddress:            null.StringFrom(o.Config.Webserver.ListenAddress),
			WebserverWebsocketConnectionLimit: null.Int64From(int64(o.Config.Webserver.WebsocketConnectionLimit)),
		}

		err = config.Insert(ctx, o.DB, boil.Infer())
		if err != nil {
			return err
		}

		o.ConfigID = config.ID
		return nil
	}

	config.GlobalHTTPTimeout = o.Config.GlobalHTTPTimeout.Nanoseconds()
	config.WebserverAdminPassword = null.StringFrom(o.Config.Webserver.AdminPassword)
	config.WebserverAdminUsername = null.StringFrom(o.Config.Webserver.AdminUsername)
	config.WebserverAllowInsecureOrigin = o.Config.Webserver.WebsocketAllowInsecureOrigin
	config.WebserverEnabled = o.Config.Webserver.Enabled
	config.WebserverListenAddress = null.StringFrom(o.Config.Webserver.ListenAddress)
	config.WebserverWebsocketConnectionLimit = null.Int64From(int64(o.Config.Webserver.WebsocketConnectionLimit))

	_, err = config.Update(ctx, o.DB, boil.Infer())
	if err != nil {
		return err
	}

	o.ConfigID = config.ID
	return nil
}

// UpdateExchangeConfigurations checks, updates and inserts new exchange
// configurations
func (o *ORM) UpdateExchangeConfigurations() error {
	for _, exch := range o.Config.Exchanges {
		config, err := models.ExchangeConfigs(qm.Where("name = ?",
			exch.Name)).One(ctx, o.DB)

		if err != nil {
			config = &models.ExchangeConfig{
				Name:                     exch.Name,
				Enabled:                  exch.Enabled,
				Verbose:                  exch.Verbose,
				WebsocketEnabled:         exch.Websocket,
				UseSandbox:               exch.UseSandbox,
				RestPollingDelay:         int64(exch.RESTPollingDelay),
				HTTPTimeout:              int64(exch.HTTPTimeout),
				AuthenticatedAPISupport:  exch.AuthenticatedAPISupport,
				APIKey:                   null.NewString(exch.APIKey, true),
				APISecret:                null.NewString(exch.APISecret, true),
				ClientID:                 null.NewString(exch.ClientID, true),
				AvailablePairs:           exch.AvailablePairs,
				EnabledPairs:             exch.EnabledPairs,
				BaseCurrencies:           exch.BaseCurrencies,
				AssetTypes:               exch.AssetTypes,
				SupportedAutoPairUpdates: exch.SupportsAutoPairUpdates,
				PairsLastUpdated:         time.Unix(exch.PairsLastUpdated, 0),
				ConfigID:                 o.ConfigID,
			}

			err = config.Insert(ctx, o.DB, boil.Infer())
			if err != nil {
				return err
			}

			o.ExchangeID[config.Name] = config.ID
			continue
		}

		config.Enabled = exch.Enabled
		config.Verbose = exch.Verbose
		config.WebsocketEnabled = exch.Websocket
		config.UseSandbox = exch.UseSandbox
		config.RestPollingDelay = int64(exch.RESTPollingDelay)
		config.HTTPTimeout = int64(exch.HTTPTimeout)
		config.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		config.APIKey = null.NewString(exch.APIKey, true)
		config.APISecret = null.NewString(exch.APISecret, true)
		config.ClientID = null.NewString(exch.ClientID, true)
		config.AvailablePairs = exch.AvailablePairs
		config.EnabledPairs = exch.EnabledPairs
		config.BaseCurrencies = exch.BaseCurrencies
		config.AssetTypes = exch.AssetTypes
		config.SupportedAutoPairUpdates = exch.SupportsAutoPairUpdates
		config.PairsLastUpdated = time.Unix(exch.PairsLastUpdated, 0)
		config.ConfigID = o.ConfigID

		_, err = config.Update(ctx, o.DB, boil.Infer())
		if err != nil {
			return err
		}

		o.ExchangeID[config.Name] = config.ID
	}
	return nil
}

// InsertExchangeTradeHistoryData inserts historic trade data
func (o *ORM) InsertExchangeTradeHistoryData(transactionID int64, exchangeName, currencyPair, assetType, orderType string, amount, rate float64, fulfilledOn time.Time) error {
	if !o.Connected {
		if o.Verbose {
			log.Println("cannot get exchange history data, no database connnection")
		}
		return nil
	}

	o.Lock()
	defer o.Unlock()

	dataExists, err := models.ExchangeTradeHistories(
		qm.Where("exchange_id = ?", o.ExchangeID[exchangeName]),
		qm.And("fulfilled_on = ?", fulfilledOn),
		qm.And("currency_pair = ?", currencyPair),
		qm.And("asset_type = ?", assetType),
		qm.And("amount = ?", amount),
		qm.And("rate = ?", rate)).Exists(ctx, o.DB)
	if err != nil {
		return err
	}

	if dataExists {
		return errors.New("row already found")
	}

	tradeHistory := &models.ExchangeTradeHistory{
		FulfilledOn:  fulfilledOn,
		CurrencyPair: currencyPair,
		AssetType:    assetType,
		OrderType:    orderType,
		Amount:       amount,
		Rate:         rate,
		ContractType: "TEST",
		OrderID:      null.Int64From(transactionID),
		ExchangeID:   o.ExchangeID[exchangeName],
	}
	return tradeHistory.Insert(ctx, o.DB, boil.Infer())
}

// GetExchangeTradeHistoryLast returns the last updated time.Time and tradeID
// values for the most recent trade history data in the set.
func (o *ORM) GetExchangeTradeHistoryLast(exchangeName, currencyPair string) (time.Time, int64, error) {
	if !o.Connected {
		if o.Verbose {
			log.Println("cannot get order history data, no database connnection")
		}
		return time.Time{}, 0, errors.New("no database connection")
	}

	o.Lock()
	defer o.Unlock()

	tradeHistory, err := models.ExchangeTradeHistories(
		qm.Where("exchange_id = ?", o.ExchangeID[exchangeName]),
		qm.And("currency_pair = ?", currencyPair),
		qm.OrderBy("fulfilled_on DESC"),
		qm.Limit(1)).One(ctx, o.DB)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return time.Time{}, 0, nil
		}
		return time.Time{}, 0, err
	}

	return tradeHistory.FulfilledOn, tradeHistory.OrderID.Int64, nil
}

// GetExchangeTradeHistory returns the full trade history by exchange name,
// currency pair and asset class
func (o *ORM) GetExchangeTradeHistory(exchName, currencyPair, assetType string) ([]exchange.TradeHistory, error) {
	o.Lock()
	defer o.Unlock()

	tradeHistory, err := models.ExchangeTradeHistories(
		qm.Where("exchange_id = ?", o.ExchangeID[exchName]),
		qm.And("currency_pair = ?", currencyPair),
		qm.And("asset_type = ?", assetType)).All(ctx, o.DB)
	if err != nil {
		return nil, err
	}

	if len(tradeHistory) == 0 {
		return nil, errors.New("no exchange trade data could be found")
	}

	var fullHistory []exchange.TradeHistory
	for _, trade := range tradeHistory {
		fullHistory = append(fullHistory,
			exchange.TradeHistory{
				Timestamp: trade.FulfilledOn,
				TID:       trade.OrderID.Int64,
				Price:     trade.Rate,
				Amount:    trade.Amount})
	}
	return fullHistory, nil
}

// GetDefaultPath returns the default database path
func GetDefaultPath() string {
	exPath, err := common.GetExecutablePath()
	if err != nil {
		panic(err)
	}

	defaultPath := fmt.Sprintf("%s%s%s%s%s",
		exPath,
		common.GetOSPathSlash(),
		"gocryptotrader_filesystem",
		common.GetOSPathSlash(),
		"database.db")
	return defaultPath
}
