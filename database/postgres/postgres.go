package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/database/base"
	"github.com/thrasher-/gocryptotrader/database/postgres/models"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/gctrpc"
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
func (p *Postgres) Setup(c *base.ConnDetails) error {
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

	if c.MemCacheSize == 0 {
		p.MaxSizeOfCache = base.DefaultMemCache
	} else {
		log.Warnf("Database write buffer size %d is not default", c.MemCacheSize)
		p.MaxSizeOfCache = c.MemCacheSize
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
	for i, s := range postgresSchema {
		fullSchema += s
		if len(postgresSchema)-1 != i {
			fullSchema += "\n\n"
		}
	}
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
		stmt, insertErr := p.C.Prepare(query)
		if insertErr != nil {
			return insertErr
		}

		_, insertErr = stmt.Exec()
		if insertErr != nil {
			return insertErr
		}
	}

	err = p.InsertExchanges(base.GetSupportedExchanges())
	if err != nil {
		return err
	}

	p.Connected = true
	return nil
}

// UserLogin creates or logs in to a saved user profile
func (p *Postgres) UserLogin(newuser bool) error {
	fmt.Println()
	if newuser {
		log.Info(base.InfoInsertUser)
		return p.InsertNewUserByPrompt()
	}

	users, err := models.Users().All(base.Ctx, p.C)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		log.Info(base.InfoNoUsers)
		return p.InsertNewUserByPrompt()
	}

	if len(users) == 1 {
		log.Info(base.InfoSingleUser)
		return p.CheckUserPassword(users[0])
	}

	log.Info(base.InfoMultiUser)
	return p.CheckUserUserPassword(users)
}

// CheckUserUserPassword matches username and checks user password with
// account
func (p *Postgres) CheckUserUserPassword(c models.UserSlice) error {
	username, err := common.PromptForUsername()
	if err != nil {
		return err
	}

	for i := range c {
		if c[i].UserName == username {
			log.Infof(base.InfoUserNameFound, username)
			return p.CheckUserPassword(c[i])
		}
	}

	return fmt.Errorf(base.UsernameNotFound, username)
}

// CheckUserPassword matches password and sets user user
func (p *Postgres) CheckUserPassword(c *models.User) error {
	for tries := 3; tries > 0; tries-- {
		_, err := common.ComparePassword([]byte(c.Password))
		if err != nil {
			if tries != 1 {
				log.Warnf(base.WarnWrongPassword, tries-1)
			}
			continue
		}
		return nil
	}
	return fmt.Errorf(base.LoginFailure, c.UserName)
}

// InsertNewUserByPrompt inserts a new user by username and password
// prompt when starting a new gocryptotrader instance
func (p *Postgres) InsertNewUserByPrompt() error {
	username, err := common.PromptForUsername()
	if err != nil {
		return err
	}

	e, err := models.Users(qm.Where(base.QueryUserName,
		username)).Exists(base.Ctx, p.C)
	if err != nil {
		return err
	}

	if e {
		return fmt.Errorf("user username %s already in use", username)
	}

	pw, err := common.PromptForPassword(true)
	if err != nil {
		return err
	}

	hashPw, err := common.HashPassword(pw)
	if err != nil {
		return err
	}

	newuser := &models.User{
		UserName:          username,
		Password:          hashPw,
		PasswordCreatedAt: time.Now(),
		LastLoggedIn:      time.Now(),
		Enabled:           true,
	}

	return newuser.Insert(base.Ctx, p.C, boil.Infer())
}

// InsertPlatformTrades inserts platform matched trades
func (p *Postgres) InsertPlatformTrades(exchangeName string, trades []*base.PlatformTrades) error {
	p.Lock()
	defer p.Unlock()

	if !p.Connected {
		return base.ErrDatabaseConnection
	}

	e, err := p.insertAndRetrieveExchange(exchangeName)
	if err != nil {
		return err
	}

	tx, err := p.NewTx()
	if err != nil {
		return err
	}

	var statementErr error
	for i := range trades {
		newStatement := &models.ExchangePlatformTradeHistory{
			FulfilledOn:  trades[i].FullfilledOn,
			CurrencyPair: trades[i].Pair,
			AssetType:    trades[i].AssetType,
			OrderType:    trades[i].OrderType,
			Amount:       trades[i].Amount,
			Rate:         trades[i].Rate,
			OrderID:      trades[i].OrderID,
			ExchangeID:   e.ID,
		}
		statementErr = newStatement.Insert(base.Ctx, tx, boil.Infer())
		if statementErr != nil {
			break
		}
	}

	err = p.CommitTx(len(trades))
	if err != nil {
		return err
	}

	return statementErr
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
		e, err = models.Exchanges(qm.Where(base.QueryExchangeName, exchName)).One(base.Ctx, p.C)
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
	err := p.NewQuery()
	if err != nil {
		return time.Time{}, "", err
	}

	defer func() { p.FinishQuery(); p.Unlock() }()

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

// GetPlatformTradeFirst returns the first updated time.Time and tradeID values
// for the initial entry boundary points
func (p *Postgres) GetPlatformTradeFirst(exchangeName, currencyPair, assetType string) (time.Time, string, error) {
	p.Lock()
	err := p.NewQuery()
	if err != nil {
		return time.Time{}, "", err
	}

	defer func() { p.FinishQuery(); p.Unlock() }()

	if !p.Connected {
		return time.Time{}, "", base.ErrDatabaseConnection
	}

	e, err := p.insertAndRetrieveExchange(exchangeName)
	if err != nil {
		return time.Time{}, "", err
	}

	th, err := e.ExchangePlatformTradeHistories(qm.Where(base.QueryCurrencyPair, currencyPair),
		qm.And(base.QueryAssetType, assetType),
		qm.OrderBy(base.OrderByFullfilledAsc),
		qm.Limit(1)).One(base.Ctx, p.C)
	if err != nil {
		return time.Time{}, "", err
	}

	return th.FulfilledOn, th.OrderID, nil
}

// GetFullPlatformHistory returns the full matched trade history on the
// exchange platform by exchange name, currency pair and asset class
func (p *Postgres) GetFullPlatformHistory(exchangeName, currencyPair, assetType string) ([]*exchange.PlatformTrade, error) {
	p.Lock()
	err := p.NewQuery()
	if err != nil {
		return nil, err
	}

	defer func() { p.FinishQuery(); p.Unlock() }()

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

	var platformHistory []*exchange.PlatformTrade
	for i := range h {
		platformHistory = append(platformHistory,
			&exchange.PlatformTrade{
				Exchange:  e.ExchangeName,
				Timestamp: h[i].FulfilledOn,
				TID:       h[i].OrderID,
				Price:     h[i].Rate,
				Amount:    h[i].Amount,
				Type:      h[i].OrderType})
	}

	return platformHistory, nil
}

// InsertExchanges inserts exchange data
func (p *Postgres) InsertExchanges(e []string) error {
	p.Lock()
	defer p.Unlock()

	for i := range e {
		exchange := &models.Exchange{
			ExchangeName: e[i],
		}

		err := exchange.Insert(base.Ctx, p.C, boil.Infer())
		if err != nil {
			return err
		}
	}
	return nil
}

// GetUserRPC returns user data
func (p *Postgres) GetUserRPC(ctx context.Context, username string) (*base.User, error) {
	p.Lock()
	defer p.Unlock()

	var bUser base.User
	c, err := models.Users(qm.Where("user_name = ?", username)).One(ctx, p.C)
	if err != nil {
		return nil, err
	}

	if !c.Enabled {
		return nil, errors.New("user %s disabled, cannot continue")
	}

	bUser.UserName = c.UserName
	bUser.UpdatedAt = c.UpdatedAt
	bUser.PasswordCreatedAt = c.PasswordCreatedAt
	bUser.Password = c.Password
	bUser.OneTimePassword = c.OneTimePassword.String
	bUser.LastLoggedIn = c.LastLoggedIn
	bUser.ID = c.ID
	bUser.Email = c.Email.String
	bUser.CreatedAt = c.CreatedAt
	bUser.Enabled = c.Enabled

	return &bUser, nil
}

// InsertUserRPC inserts a user via RPC with context
func (p *Postgres) InsertUserRPC(ctx context.Context, username, password string) error {
	p.Lock()
	defer p.Unlock()

	newUser := &models.User{
		UserName:          username,
		Password:          password,
		PasswordCreatedAt: time.Now(),
		LastLoggedIn:      time.Now(),
		Enabled:           true,
	}

	return newUser.Insert(ctx, p.C, boil.Infer())
}

// GetExchangeLoadedDataRPC gets distinct values about data entered
func (p *Postgres) GetExchangeLoadedDataRPC(ctx context.Context, exchange string) ([]*gctrpc.AvailableData, error) {
	p.Lock()
	// because this is querying the exchange platform table we have to commit
	// transaction in memory TODO: make this friendlier
	err := p.NewQuery()
	if err != nil {
		return nil, err
	}
	defer func() { p.FinishQuery(); p.Unlock() }()

	e, err := p.insertAndRetrieveExchange(exchange)
	if err != nil {
		return nil, err
	}

	h, err := models.ExchangePlatformTradeHistories(
		qm.SQL("SELECT DISTINCT currency_pair, asset_type FROM exchange_platform_trade_history"),
		qm.Where("exchange_id = ?", e.ID),
	).All(ctx, p.C)
	if err != nil {
		return nil, err
	}

	var avail []*gctrpc.AvailableData
	for i := range h {
		p := currency.NewPairFromString(h[i].CurrencyPair)
		avail = append(avail, &gctrpc.AvailableData{
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			Asset: h[i].AssetType,
		})
	}

	return avail, nil
}

// GetExchangePlatformHistoryRPC returns gctrpc values for RPC export
func (p *Postgres) GetExchangePlatformHistoryRPC(ctx context.Context, exchange, pair, asset string) ([]*gctrpc.PlatformHistory, error) {
	p.Lock()
	// because this is querying the exchange platform table we have to commit
	// transaction in memory TODO: make this friendlier
	err := p.NewQuery()
	if err != nil {
		return nil, err
	}
	defer func() { p.FinishQuery(); p.Unlock() }()

	e, err := p.insertAndRetrieveExchange(exchange)
	if err != nil {
		return nil, err
	}

	h, err := models.ExchangePlatformTradeHistories(
		qm.Where("exchange_id = ?", e.ID),
		qm.Where("currency_pair = ?", pair),
		qm.Where("asset_type = ?", asset),
	).All(ctx, p.C)
	if err != nil {
		return nil, err
	}

	var histories []*gctrpc.PlatformHistory
	for i := range h {
		p := currency.NewPairFromString(h[i].CurrencyPair)
		histories = append(histories, &gctrpc.PlatformHistory{
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			AssetType:    h[i].AssetType,
			Timestamp:    h[i].FulfilledOn.Unix(),
			TradeId:      h[i].OrderID,
			Price:        h[i].Rate,
			Amount:       h[i].Amount,
			TradeType:    h[i].OrderType,
			ExchangeName: exchange,
		})
	}
	return histories, nil
}

// GetUserAuditRPC returns a full user audit list for tracking changes they
// innacted on other users and high security changes
func (p *Postgres) GetUserAuditRPC(ctx context.Context, username string) ([]*base.Audit, error) {
	p.Lock()
	defer p.Unlock()

	c, err := models.Users(qm.Where("user_name = ?", username)).One(ctx, p.C)
	if err != nil {
		return nil, err
	}

	audits, err := c.AuditTrails().All(ctx, p.C)
	if err != nil {
		return nil, err
	}

	var auditTrails []*base.Audit
	for i := range audits {
		auditTrails = append(auditTrails, &base.Audit{
			UserID:       int64(audits[i].UserID),
			Change:       audits[i].Change,
			TimeOfChange: audits[i].CreatedAt,
		})
	}

	return auditTrails, nil
}

// GetUsersRPC returns a full user list that is in the database
func (p *Postgres) GetUsersRPC(ctx context.Context) ([]*base.User, error) {
	p.Lock()
	defer p.Unlock()

	c, err := models.Users().All(ctx, p.C)
	if err != nil {
		return nil, err
	}

	var users []*base.User
	for i := range c {
		users = append(users, &base.User{UserName: c[i].UserName})
	}

	return users, nil
}

// EnableDisableUserRPC changes enabled or disabled state of user
func (p *Postgres) EnableDisableUserRPC(ctx context.Context, username string, enable bool) error {
	p.Lock()
	defer p.Unlock()

	c, err := models.Users(qm.Where("user_name = ?", username)).One(ctx, p.C)
	if err != nil {
		return err
	}

	if c.Enabled == enable {
		if c.Enabled {
			return errors.New("user already enabled")
		}
		return errors.New("user already disabled")
	}

	c.Enabled = enable
	_, err = c.Update(ctx, p.C, boil.Infer())
	return err
}

// SetUserPasswordRPC users sets users password TODO: Add table password
// to make sure expired passwords are not reused
func (p *Postgres) SetUserPasswordRPC(ctx context.Context, username, password string) error {
	p.Lock()
	defer p.Unlock()

	c, err := models.Users(qm.Where("user_name = ?", username)).One(ctx, p.C)
	if err != nil {
		return err
	}

	c.Password = password
	c.PasswordCreatedAt = time.Now()

	_, err = c.Update(ctx, p.C, boil.Infer())
	return err
}

// ModifyUserRPC modifies user details TODO: expand User table fields to
// include firstname, lastname, address details
func (p *Postgres) ModifyUserRPC(ctx context.Context, username, email string) error {
	if email == "" {
		return errors.New("no user data set")
	}

	p.Lock()
	defer p.Unlock()

	c, err := models.Users(qm.Where("user_name = ?", username)).One(ctx, p.C)
	if err != nil {
		return err
	}

	c.Email.SetValid(email)

	_, err = c.Update(ctx, p.C, boil.Infer())
	return err
}
