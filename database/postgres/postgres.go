package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/database/base"
	"github.com/thrasher-/gocryptotrader/database/postgres/models"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	log "github.com/thrasher-/gocryptotrader/logger"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"

	// External package for SQL queries
	_ "github.com/volatiletech/sqlboiler/drivers/sqlboiler-psql/driver"
)

const (
	conn = "user=%s password=%s dbname=%s host=%s port=%s sslmode=%s"
)

// Postgres defines a connection to a PosgreSQL database
type Postgres struct {
	base.RelationalMap
}

// Setup creates and sets database directory, folders and supplementary files
// that works in conjunction with PosgreSQL to regenerate models
func (p *Postgres) Setup(c base.ConnDetails) error {
	if c.DirectoryPath == "" {
		return base.ErrDirectoryNotSet
	}

	if c.Host == "" {
		return errors.New("host not set for postgres connection, please set in flag -dbhost")
	}

	if c.User == "" {
		return errors.New("username not set for postgres connection, please set in flag -dbuser")
	}

	if c.DBName == "" {
		return errors.New("name not set for postgres connection, please set in flag -dbname")
	}

	if c.Pass == "" {
		log.Warnf("Password not set for the postgreSQL connection, please set in flag -dbpass")
	}

	p.InstanceName = base.Postgres
	p.PathDBDir = c.DirectoryPath
	p.DatabaseName = c.DBName
	p.Host = c.Host
	p.Password = c.Pass
	p.User = c.User
	if c.Port == "" {
		// default port
		p.Port = "5432"
	} else {
		p.Port = c.Port
	}
	p.SSLMode = c.SSLMode
	p.Verbose = c.Verbose

	err := p.SetupHelperFiles()
	if err != nil {
		return err
	}

	fullPathToSchema := p.PathDBDir + base.PostGresSchema
	// Creates a schema file for informational deployment
	_, err = common.ReadFile(fullPathToSchema)
	if err != nil {
		err = common.WriteFile(fullPathToSchema, []byte(GetSchema()))
		if err != nil {
			return err
		} else if p.Verbose {
			log.Debugf(base.DebugSchemaFileCreated, fullPathToSchema)
		}
	} else if p.Verbose {
		log.Debugf(base.DebugSchemaFileFound, fullPathToSchema)
	}
	return nil
}

// GetSchema returns the full schema ready for file use
func GetSchema() string {
	var fullSchema string

	fullSchema += postgresSchema[0] + "\n\n"
	fullSchema += postgresSchema[1] + "\n\n"
	fullSchema += postgresSchema[2] + "\n\n"
	fullSchema += postgresSchema[3]

	return fullSchema
}

// Connect initiates a connection to a PosgreSQL database
func (p *Postgres) Connect() error {
	if p.Host == "" {
		return fmt.Errorf("connect error host not set for %s", p.InstanceName)
	}

	if p.DatabaseName == "" {
		return fmt.Errorf("connect error database not set for %s",
			p.InstanceName)
	}

	if p.User == "" {
		return fmt.Errorf("connect error user not set for %s", p.InstanceName)
	}

	if p.Verbose {
		log.Debugf(base.DebugDBConnecting, p.InstanceName, p.Host)
	}

	var port string
	if p.Port == "" {
		port = "5432"
	} else {
		port = p.Port
	}

	var err error
	p.C, err = sql.Open(base.Postgres,
		fmt.Sprintf(conn,
			p.User,
			p.Password,
			p.DatabaseName,
			p.Host,
			port,
			p.SSLMode))
	if err != nil {
		return err
	}

	err = p.C.Ping()
	if err != nil {
		newErr := p.C.Close()
		if newErr != nil {
			log.Error(newErr)
		}
		return err
	}

	rows, err := p.C.Query("SELECT * FROM information_schema.tables WHERE table_schema='public';")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		log.Warnf(base.WarnTablesExist)
		p.Connected = true
		return nil
	}

	// Instantiate tables in new PostgreSQL database if no random tables found
	for _, query := range postgresSchema {
		stmt, err := p.C.Prepare(query)
		if err != nil {
			return err
		}

		_, err = stmt.Exec()
		if err != nil {
			return err
		}
	}

	p.Connected = true
	return nil
}

// ClientLogin creates or logs in to a saved user profile
func (p *Postgres) ClientLogin(newclient bool) error {
	fmt.Println()
	if newclient {
		log.Info(base.InfoInsertClient)
		return p.InsertNewClient()
	}

	clients, err := models.Clients().All(base.Ctx, p.C)
	if err != nil {
		return err
	}

	if len(clients) == 0 {
		log.Info(base.InfoNoClients)
		return p.InsertNewClient()
	}

	if len(clients) == 1 {
		log.Info(base.InfoSingleClient)
		return p.CheckClientPassword(clients[0])
	}

	log.Info(base.InfoMultiClient)
	return p.CheckClientUserPassword(clients)
}

// CheckClientUserPassword matches username and checks client password with
// account
func (p *Postgres) CheckClientUserPassword(c models.ClientSlice) error {
	username, err := common.PromptForUsername()
	if err != nil {
		return err
	}

	for i := range c {
		if c[i].UserName == username {
			log.Infof(base.InfoUserNameFound, username)
			return p.CheckClientPassword(c[i])
		}
	}

	return fmt.Errorf(base.UsernameNotFound, username)
}

// CheckClientPassword matches password and sets user client
func (p *Postgres) CheckClientPassword(c *models.Client) error {
	for tries := 3; tries > 0; tries-- {
		_, err := common.ComparePassword([]byte(c.Password))
		if err != nil {
			if tries != 1 {
				log.Warnf(base.WarnWrongPassword, tries-1)
			}
			continue
		}

		p.Client = c
		return nil
	}
	return fmt.Errorf(base.LoginFailure, c.UserName)
}

// InsertNewClient inserts a new client by username and password
func (p *Postgres) InsertNewClient() error {
	username, err := common.PromptForUsername()
	if err != nil {
		return err
	}

	e, err := models.Clients(qm.Where(base.QueryUserName,
		username)).Exists(base.Ctx, p.C)
	if err != nil {
		return err
	}

	if e {
		return fmt.Errorf("client username %s already in use", username)
	}

	pw, err := common.PromptForPassword(true)
	if err != nil {
		return err
	}

	hashPw, err := common.HashPassword(pw)
	if err != nil {
		return err
	}

	newuser := &models.Client{
		UserName:     username,
		Password:     hashPw,
		LastLoggedIn: time.Now(),
	}

	err = newuser.Insert(base.Ctx, p.C, boil.Infer())
	if err != nil {
		return err
	}

	err = newuser.Reload(base.Ctx, p.C)
	if err != nil {
		return err
	}

	p.Client = newuser
	return nil
}

// InsertPlatformTrade inserts platform matched trades
func (p *Postgres) InsertPlatformTrade(orderID, exchangeName, currencyPair, assetType, orderType string, amount, rate float64, fulfilledOn time.Time) error {
	p.Lock()
	defer p.Unlock()

	if !p.Connected {
		return base.ErrDatabaseConnection
	}

	e, err := p.insertAndRetrieveExchange(exchangeName)
	if err != nil {
		return err
	}

	return e.AddExchangePlatformTradeHistories(base.Ctx,
		p.C,
		true,
		&models.ExchangePlatformTradeHistory{
			FulfilledOn:  fulfilledOn,
			CurrencyPair: currencyPair,
			AssetType:    assetType,
			OrderType:    orderType,
			Amount:       amount,
			Rate:         rate,
			OrderID:      orderID,
		})
}

// InsertAndRetrieveExchange returns the pointer to an exchange model to
// minimise database queries for future insertion, used in conjunction with
// lockable funcs
func (p *Postgres) insertAndRetrieveExchange(exchName string) (*models.Exchange, error) {
	if p.Exchanges == nil {
		p.Exchanges = make(map[string]interface{})
	}

	e, ok := p.Exchanges[exchName].(*models.Exchange)
	if !ok {
		var err error
		e, err = models.Exchanges(qm.Where("exchange_name = ?", exchName)).One(base.Ctx, p.C)
		if err != nil {
			i := &models.Exchange{
				ExchangeName: exchName,
			}

			err = i.Insert(base.Ctx, p.C, boil.Infer())
			if err != nil {
				return nil, err
			}

			err = i.Reload(base.Ctx, p.C)
			if err != nil {
				return nil, err
			}

			e = i
		}
	}

	p.Exchanges[exchName] = e
	return e, nil
}

// GetPlatformTradeLast returns the last updated time.Time and tradeID values
// for the most recent trade history data in the set
func (p *Postgres) GetPlatformTradeLast(exchangeName, currencyPair, assetType string) (time.Time, string, error) {
	p.Lock()
	defer p.Unlock()

	if !p.Connected {
		return time.Time{}, "", base.ErrDatabaseConnection
	}

	e, err := p.insertAndRetrieveExchange(exchangeName)
	if err != nil {
		return time.Time{}, "", err
	}

	th, err := e.ExchangePlatformTradeHistories(qm.Where(base.QueryCurrencyPair, currencyPair),
		qm.And(base.QueryAssetType, assetType),
		qm.OrderBy(base.OrderByFulfilledDesc),
		qm.Limit(1)).One(base.Ctx, p.C)
	if err != nil {
		return time.Time{}, "", err
	}

	return th.FulfilledOn, th.OrderID, nil
}

// GetFullPlatformHistory returns the full matched trade history on the
// exchange platform by exchange name, currency pair and asset class
func (p *Postgres) GetFullPlatformHistory(exchangeName, currencyPair, assetType string) ([]exchange.PlatformTrade, error) {
	p.Lock()
	defer p.Unlock()

	if !p.Connected {
		return nil, base.ErrDatabaseConnection
	}

	e, err := p.insertAndRetrieveExchange(exchangeName)
	if err != nil {
		return nil, err
	}

	h, err := e.ExchangePlatformTradeHistories(qm.Where(base.QueryCurrencyPair, currencyPair),
		qm.And(base.QueryAssetType, assetType)).All(base.Ctx, p.C)
	if err != nil {
		return nil, err
	}

	var platformHistory []exchange.PlatformTrade
	for i := range h {
		platformHistory = append(platformHistory,
			exchange.PlatformTrade{
				Exchange:  e.ExchangeName,
				Timestamp: h[i].FulfilledOn,
				TID:       h[i].OrderID,
				Price:     h[i].Rate,
				Amount:    h[i].Amount,
				Type:      h[i].OrderType})
	}

	return platformHistory, nil
}

// GetClientDetails returns a string of current user details
func (p *Postgres) GetClientDetails() (string, error) {
	p.Lock()
	defer p.Unlock()
	return p.Client.(*models.Client).UserName, nil
}
