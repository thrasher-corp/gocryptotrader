package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/database/models"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"

	// External package for SQL queries
	_ "github.com/volatiletech/sqlboiler-sqlite3/driver"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

// DefaultDir is the default directory for the database
var DefaultDir = fmt.Sprintf("%s%sdatabase%s",
	common.GetDefaultDataDir(runtime.GOOS),
	common.GetOSPathSlash(),
	common.GetOSPathSlash())

// DefaultPath is the full default path to the database
var DefaultPath = DefaultDir + "database.db"

// ctx set global context
var ctx = context.Background()

// ORM is the overarching type across the database package that handles database
// connections and relational mapping
type ORM struct {
	DB          *sql.DB
	Path        string
	Config      *config.Config
	sessionID   int64
	sessionCred []byte
	Verbose     bool
	Connected   bool

	sync.Mutex
}

// Setup creates and sets database directory folders and supplementary files
// that works in conjunction with SQLBoiler TODO create new tool that
// autogenrates new database models depending on schema and database.
func Setup(dirPath string, verbose bool) error {
	// Checks to see if default directory is made
	err := common.CheckDir(dirPath, true)
	if err != nil {
		return err
	}

	// Creates a configuration file that points to a database for generating new
	// database models
	_, err = common.ReadFile(dirPath + "sqlboiler.toml")
	if err != nil {
		tomlFile := fmt.Sprintf(
			`[sqlite3]
dbname = "%s"`, DefaultPath)
		err = common.WriteFile(dirPath+"sqlboiler.toml", []byte(tomlFile))
		if err != nil {
			return err
		}

		if verbose {
			log.Printf("Created helper file for SQLBoiler model deployment %s",
				dirPath+"sqlboiler.toml")
		}
	} else {
		if verbose {
			log.Printf("SQLBoiler file found at %s",
				dirPath+"sqlboiler.toml")
		}
	}

	// Creates a schema file for informational deployment
	_, err = common.ReadFile(dirPath + "db.schema")
	if err != nil {
		err = common.WriteFile(dirPath+"db.schema", []byte(fullSchema))
		if err != nil {
			return err
		}
		if verbose {
			log.Printf("Created schema file for database update and SQLBoiler model deployment %s",
				dirPath+"db.schema")
		}
	} else {
		if verbose {
			log.Printf("Schema file found at %s",
				dirPath+"db.schema")
		}
	}
	return nil
}

// Connect initiates a connection to a SQLite database
func Connect(sqlite3Path string, verbose bool) (*ORM, error) {
	if verbose {
		log.Printf("Opening connection to sqlite3 database using PATH: %s",
			sqlite3Path)
	}

	SQLite := new(ORM)
	SQLite.Path = sqlite3Path

	var err error
	SQLite.DB, err = sql.Open("sqlite3", SQLite.Path)
	if err != nil {
		return nil, err
	}

	err = SQLite.DB.Ping()
	if err != nil {
		return nil, err
	}

	// Instantiate tables in new database
	for name, query := range databaseTables {
		rows, err := SQLite.DB.Query(
			fmt.Sprintf(
				"SELECT name FROM sqlite_master WHERE type='table' AND name='%s'",
				name))
		if err != nil {
			return nil, err
		}

		var returnedName string
		for rows.Next() {
			rows.Scan(&returnedName)
		}

		if returnedName == name {
			continue
		}

		stmt, err := SQLite.DB.Prepare(query)
		if err != nil {
			return nil, err
		}

		_, err = stmt.Exec()
		if err != nil {
			return nil, err
		}
	}

	SQLite.Verbose = verbose
	SQLite.Connected = true

	return SQLite, nil
}

// UserLogin creates or logs in to a saved user profile
func (o *ORM) UserLogin() error {
	for {
		username, err := common.PromptForUsername()
		if err != nil {
			return err
		}

		users, err := o.getUser(username)
		if err != nil {
			return err
		}

		if len(users) > 1 {
			return errors.New("duplicate users found in database")
		}

		if len(users) == 1 {
			for tries := 3; tries > 0; tries-- {
				pw, err := common.ComparePassword([]byte(users[0].Password))
				if err != nil {
					fmt.Println("Incorrect password, try again.")
					continue
				}
				return o.SetSessionData(username, pw)
			}
			return fmt.Errorf("Failed to authenticate using password for username %s",
				username)
		}

		var decision string
		fmt.Printf("Username %s not found in database, would you like to create a new user, enter [y,n],\nthen press enter to continue.\n",
			username)
		fmt.Scanln(&decision)

		if common.YesOrNo(decision) {
			pw, err := common.PromptForPassword(true)
			if err != nil {
				return err
			}

			err = o.insertUser(username, pw)
			if err != nil {
				return err
			}
			return o.SetSessionData(username, pw)
		}
	}
}

// getUser returns a slice of users associated witht he username string
func (o *ORM) getUser(username string) (models.GCTUserSlice, error) {
	return models.GCTUsers(qm.Where("name = ?", username)).All(ctx, o.DB)
}

// insertUser inserts a new user by username and password
func (o *ORM) insertUser(username string, password []byte) error {
	exists, err := models.GCTUsers(qm.Where("name = ?", username)).Exists(ctx, o.DB)
	if err != nil {
		return err
	}

	if exists {
		return errors.New("username already found")
	}

	hashPw, err := common.HashPassword(password)
	if err != nil {
		return err
	}

	newuser := &models.GCTUser{
		Name:       username,
		Password:   hashPw,
		InsertedAt: time.Now(),
		AmendedAt:  time.Now(),
	}

	return newuser.Insert(ctx, o.DB, boil.Infer())
}

// SetSessionData sets user data for handling user/database connection
func (o *ORM) SetSessionData(username string, cred []byte) error {
	user, err := models.GCTUsers(qm.Where("name = ?", username)).One(ctx, o.DB)
	if err != nil {
		return err
	}

	o.sessionID = user.ID
	o.sessionCred = cred

	return nil
}

// GetConfig returns a saved configuration
func (o *ORM) GetConfig(configName, configPath string, configOverride, saveConfig bool) (*config.Config, error) {
	switch {
	case configOverride && saveConfig:
		if configPath == "" {
			return nil,
				errors.New("database.go - GetConfig() error - no config path found")
		}

		var cfg = config.GetConfig()
		err := cfg.LoadConfig(configPath)
		if err != nil {
			return nil, fmt.Errorf("database.go - GetConfig() error %s", err)
		}
		return cfg, o.saveConfiguration(cfg)

	case configOverride && !saveConfig:
		if configPath == "" {
			return nil,
				errors.New("database.go - GetConfig() error - no config path found")
		}

		var cfg = config.GetConfig()
		err := cfg.LoadConfig(configPath)
		if err != nil {
			return nil, fmt.Errorf("database.go - GetConfig() error %s", err)
		}
		return cfg, nil

	case !configOverride && saveConfig:
		if configPath == "" {
			return nil,
				errors.New("database.go - GetConfig() error - no config path found")
		}

		var cfg = config.GetConfig()
		err := cfg.LoadConfig(configPath)
		if err != nil {
			return nil, fmt.Errorf("database.go - GetConfig() error %s", err)
		}

		err = o.saveConfiguration(cfg)
		if err != nil {
			return nil, err
		}

		if cfg.Name == configName {
			return cfg, nil
		}

		return o.getSavedConfiguration(configName)

	default:
		return o.getSavedConfiguration(configName)
	}
}

// getSavedConfiguration returns the saved configuration in the database by its
// configuration name
func (o *ORM) getSavedConfiguration(configName string) (*config.Config, error) {
	if configName == "" {
		return nil,
			errors.New("database.go - getSavedConfiguration() error - no config name supplied")
	}

	var configuration config.Config
	cfg, err := models.GCTConfigs(qm.Where("config_name = ?",
		configName)).One(ctx, o.DB)
	if err != nil {
		return nil, err
	}

	decryptedFile, err := o.DeEncryptConfiguration(cfg.ConfigFull)
	if err != nil {
		return nil, err
	}

	return &configuration, common.JSONDecode(decryptedFile, &configuration)
}

// saveConfiguration saves the configuration
func (o *ORM) saveConfiguration(c *config.Config) error {
	cfg, err := models.GCTConfigs(qm.Where("config_name = ?", c.Name)).One(ctx, o.DB)
	if err != nil {
		if common.StringContains(err.Error(), "no rows in result set") {
			saveConfig := &models.GCTConfig{}

			encodedConfig, err := common.JSONEncode(*c)
			if err != nil {
				return err
			}

			payload, err := o.EncryptConfiguration(encodedConfig)
			if err != nil {
				return err
			}

			saveConfig.ConfigFull = payload
			saveConfig.ConfigName = c.Name
			t := time.Now()
			saveConfig.InsertedAt = t
			saveConfig.AmendedAt = t
			saveConfig.GCTUserID = o.sessionID

			return saveConfig.Insert(ctx, o.DB, boil.Infer())
		}
		return err
	}

	encodedConfig, err := common.JSONEncode(*c)
	if err != nil {
		return err
	}

	payload, err := o.EncryptConfiguration(encodedConfig)
	if err != nil {
		return err
	}

	cfg.ConfigFull = payload
	cfg.AmendedAt = time.Now()
	cfg.GCTUserID = o.sessionID

	_, err = cfg.Update(ctx, o.DB, boil.Infer())
	return err
}

// EncryptConfiguration encrypts configuration before saving to database
func (o *ORM) EncryptConfiguration(payload []byte) ([]byte, error) {
	return config.EncryptConfigFile(payload, o.sessionCred)
}

// DeEncryptConfiguration unencrypts configuration when retrieving from database
func (o *ORM) DeEncryptConfiguration(payload []byte) ([]byte, error) {
	return config.DecryptConfigFile(payload, o.sessionCred)
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
		qm.Where("exchange_name = ?", exchangeName),
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
		OrderID:      transactionID,
		ExchangeName: exchangeName,
		InsertedAt:   time.Now(),
		AmendedAt:    time.Now(),
	}

	return tradeHistory.Insert(ctx, o.DB, boil.Infer())
}

// GetExchangeTradeHistoryLast returns the last updated time.Time and tradeID
// values for the most recent trade history data in the set.
func (o *ORM) GetExchangeTradeHistoryLast(exchangeName, currencyPair, assetType string) (time.Time, int64, error) {
	if !o.Connected {
		if o.Verbose {
			log.Println("cannot get order history data, no database connnection")
		}
		return time.Time{}, 0, errors.New("no database connection")
	}

	o.Lock()
	defer o.Unlock()

	tradeHistory, err := models.ExchangeTradeHistories(
		qm.Where("exchange_name = ?", exchangeName),
		qm.And("currency_pair = ?", currencyPair),
		qm.And("asset_type = ?", assetType),
		qm.OrderBy("fulfilled_on DESC"),
		qm.Limit(1)).One(ctx, o.DB)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return time.Time{}, 0, nil
		}
		return time.Time{}, 0, err
	}

	return tradeHistory.FulfilledOn, tradeHistory.OrderID, nil
}

// GetExchangeTradeHistory returns the full trade history by exchange name,
// currency pair and asset class
func (o *ORM) GetExchangeTradeHistory(exchName, currencyPair, assetType string) ([]exchange.TradeHistory, error) {
	o.Lock()
	defer o.Unlock()

	tradeHistory, err := models.ExchangeTradeHistories(
		qm.Where("exchange_name = ?", exchName),
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
				Exchange:  trade.ExchangeName,
				Timestamp: trade.FulfilledOn,
				TID:       trade.OrderID,
				Price:     trade.Rate,
				Amount:    trade.Amount,
				Type:      trade.OrderType})
	}
	return fullHistory, nil
}

// Disconnect closes the database connection
func (o *ORM) Disconnect() error {
	return o.DB.Close()
}
