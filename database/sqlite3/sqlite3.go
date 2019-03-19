package sqlite3

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/database/base"
	"github.com/thrasher-/gocryptotrader/database/sqlite3/models"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/gctrpc"
	log "github.com/thrasher-/gocryptotrader/logger"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"

	// External package for SQL queries
	_ "github.com/volatiletech/sqlboiler-sqlite3/driver"
)

// SQLite3 defines a connection to a SQLite3 database
type SQLite3 struct {
	base.RelationalMap
}

// Setup creates and sets database directory, folders and supplementary files
// that works in conjunction with SQLBoiler to regenerate models
func (s *SQLite3) Setup(c *base.ConnDetails) error {
	if c.DirectoryPath == "" {
		return base.ErrDirectoryNotSet
	}

	if c.SQLPath == "" {
		return errors.New("full path to SQLite3 database not set")
	}

	s.PathToDB = c.SQLPath
	s.Verbose = c.Verbose
	s.InstanceName = base.SQLite
	s.PathDBDir = c.DirectoryPath

	if c.MemCacheSize == 0 {
		s.MaxSizeOfCache = base.DefaultMemCache
	} else {
		log.Warnf("Database write buffer size %d is not default", c.MemCacheSize)
		s.MaxSizeOfCache = c.MemCacheSize
	}

	err := s.SetupHelperFiles()
	if err != nil {
		return err
	}

	fullPathToSchema := c.DirectoryPath + base.SQLite3Schema
	// Creates a schema file for informational deployment
	_, err = common.ReadFile(fullPathToSchema)
	if err != nil {
		err = common.WriteFile(fullPathToSchema, []byte(GetSchema()))
		if err != nil {
			return err
		} else if s.Verbose {
			log.Debugf(base.DebugSchemaFileCreated, fullPathToSchema)
		}
	} else if s.Verbose {
		log.Debugf(base.DebugSchemaFileFound, fullPathToSchema)
	}
	return nil
}

// GetSchema returns the full schema ready for file use
func GetSchema() string {
	var fullSchema string
	for i, s := range sqliteSchema {
		fullSchema += s
		if len(sqliteSchema)-1 != i {
			fullSchema += "\n\n"
		}
	}
	return fullSchema
}

// Connect initiates a connection to a SQLite database
func (s *SQLite3) Connect() error {
	if s.PathToDB == "" {
		return fmt.Errorf(base.DBPathNotSet, s.InstanceName)
	}

	if s.Verbose {
		log.Debugf(base.DebugDBConnecting, s.InstanceName, s.PathToDB)
	}

	var err error
	s.C, err = sql.Open(base.SQLite, s.PathToDB)
	if err != nil {
		return err
	}

	err = s.C.Ping()
	if err != nil {
		err = s.Disconnect()
		if err != nil {
			log.Error("Disconnection from sqlite3 db error", err)
		}
		return err
	}

	rows, err := s.C.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var supername string
		rows.Scan(&supername)

		if !common.StringContains(supername, "sqlite3_") {
			log.Warnf(base.WarnTablesExist)
			s.Connected = true
			return nil
		}
	}

	// Instantiate tables in new SQLite3 database if no random tables found
	for _, query := range sqliteSchema {
		stmt, queryErr := s.C.Prepare(query)
		if queryErr != nil {
			return queryErr
		}

		_, queryErr = stmt.Exec()
		if queryErr != nil {
			return queryErr
		}
	}

	err = s.InsertExchanges(base.GetSupportedExchanges())
	if err != nil {
		return err
	}

	s.Connected = true
	return nil
}

// UserLogin creates or logs in to a saved user profile
func (s *SQLite3) UserLogin(newUser bool) error {
	fmt.Println()
	if newUser {
		log.Info(base.InfoInsertUser)
		return s.InsertNewUserByPrompt()
	}

	users, err := models.Users().All(base.Ctx, s.C)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		log.Info(base.InfoNoUsers)
		return s.InsertNewUserByPrompt()
	}

	if len(users) == 1 {
		log.Info(base.InfoSingleUser)
		return s.CheckUserPassword(users[0])
	}

	log.Info(base.InfoMultiUser)
	return s.CheckUsernameAndPassword(users)
}

// CheckUsernameAndPassword matches username and checks user password with
// account
func (s *SQLite3) CheckUsernameAndPassword(c models.UserSlice) error {
	username, err := common.PromptForUsername()
	if err != nil {
		return err
	}

	for i := range c {
		if c[i].UserName == username {
			log.Infof(base.InfoUserNameFound, username)
			return s.CheckUserPassword(c[i])
		}
	}

	return fmt.Errorf(base.UsernameNotFound, username)
}

// CheckUserPassword matches password and sets user user
func (s *SQLite3) CheckUserPassword(c *models.User) error {
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
func (s *SQLite3) InsertNewUserByPrompt() error {
	username, err := common.PromptForUsername()
	if err != nil {
		return err
	}

	e, err := models.Users(qm.Where(base.QueryUserName,
		username)).Exists(base.Ctx, s.C)
	if err != nil {
		return err
	}

	if e {
		return fmt.Errorf(base.UsernameAlreadyUsed, username)
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

	return newuser.Insert(base.Ctx, s.C, boil.Infer())
}

// InsertPlatformTrades inserts platform matched trades
func (s *SQLite3) InsertPlatformTrades(exchangeName string, trades []*base.PlatformTrades) error {
	s.Lock()
	defer s.Unlock()

	if !s.Connected {
		return base.ErrDatabaseConnection
	}

	e, err := s.insertAndRetrieveExchange(exchangeName)
	if err != nil {
		return err
	}

	tx, err := s.NewTx()
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

	err = s.CommitTx(len(trades))
	if err != nil {
		return err
	}

	return statementErr
}

// InsertAndRetrieveExchange returns the pointer to an exchange model to
// minimise database queries for future insertion, used in conjunction with
// lockable funcs
func (s *SQLite3) insertAndRetrieveExchange(exchName string) (*models.Exchange, error) {
	if s.Exchanges == nil {
		s.Exchanges = make(map[string]interface{})
	}

	e, ok := s.Exchanges[exchName].(*models.Exchange)
	if !ok {
		var err error
		e, err = models.Exchanges(qm.Where(base.QueryExchangeName, exchName)).One(base.Ctx, s.C)
		if err != nil {
			i := &models.Exchange{
				ExchangeName: exchName,
			}

			err = i.Insert(base.Ctx, s.C, boil.Infer())
			if err != nil {
				return nil, err
			}

			err = i.Reload(base.Ctx, s.C)
			if err != nil {
				return nil, err
			}

			e = i
		}
	}

	s.Exchanges[exchName] = e
	return e, nil
}

// GetPlatformTradeLast returns the last updated time.Time and tradeID values
// for the most recent trade history data in the set
func (s *SQLite3) GetPlatformTradeLast(exchangeName, currencyPair, assetType string) (time.Time, string, error) {
	s.Lock()
	err := s.NewQuery()
	if err != nil {
		return time.Time{}, "", err
	}

	defer func() { s.FinishQuery(); s.Unlock() }()

	if !s.Connected {
		return time.Time{}, "", base.ErrDatabaseConnection
	}

	e, err := s.insertAndRetrieveExchange(exchangeName)
	if err != nil {
		return time.Time{}, "", err
	}

	th, err := e.ExchangePlatformTradeHistory(qm.Where(base.QueryCurrencyPair, currencyPair),
		qm.And(base.QueryAssetType, assetType),
		qm.OrderBy(base.OrderByFulfilledDesc),
		qm.Limit(1)).One(base.Ctx, s.C)
	if err != nil {
		return time.Time{}, "", err
	}

	return th.FulfilledOn, th.OrderID, nil
}

// GetPlatformTradeFirst returns the first updated time.Time and tradeID values
// for the initial entry boundary points
func (s *SQLite3) GetPlatformTradeFirst(exchangeName, currencyPair, assetType string) (time.Time, string, error) {
	s.Lock()
	err := s.NewQuery()
	if err != nil {
		return time.Time{}, "", err
	}

	defer func() { s.Unlock(); s.FinishQuery() }()

	if !s.Connected {
		return time.Time{}, "", base.ErrDatabaseConnection
	}

	e, err := s.insertAndRetrieveExchange(exchangeName)
	if err != nil {
		return time.Time{}, "", err
	}

	th, err := e.ExchangePlatformTradeHistory(qm.Where(base.QueryCurrencyPair, currencyPair),
		qm.And(base.QueryAssetType, assetType),
		qm.OrderBy(base.OrderByFullfilledAsc),
		qm.Limit(1)).One(base.Ctx, s.C)
	if err != nil {
		return time.Time{}, "", err
	}

	return th.FulfilledOn, th.OrderID, nil
}

// GetFullPlatformHistory returns the full matched trade history on the
// exchange platform by exchange name, currency pair and asset class
func (s *SQLite3) GetFullPlatformHistory(exchangeName, currencyPair, assetType string) ([]*exchange.PlatformTrade, error) {
	s.Lock()
	err := s.NewQuery()
	if err != nil {
		return nil, err
	}

	defer func() { s.Unlock(); s.FinishQuery() }()

	if !s.Connected {
		return nil, base.ErrDatabaseConnection
	}

	e, err := s.insertAndRetrieveExchange(exchangeName)
	if err != nil {
		return nil, err
	}

	h, err := e.ExchangePlatformTradeHistory(qm.Where(base.QueryCurrencyPair, currencyPair),
		qm.And(base.QueryAssetType, assetType)).All(base.Ctx, s.C)
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
func (s *SQLite3) InsertExchanges(e []string) error {
	s.Lock()
	defer s.Unlock()

	for i := range e {
		exchange := &models.Exchange{
			ExchangeName: e[i],
		}

		err := exchange.Insert(base.Ctx, s.C, boil.Infer())
		if err != nil {
			return err
		}
	}
	return nil
}

// GetUserRPC returns user data
func (s *SQLite3) GetUserRPC(ctx context.Context, username string) (*base.User, error) {
	s.Lock()
	defer s.Unlock()

	var buser *base.User
	c, err := models.Users(qm.Where("user_name = ?", username)).One(ctx, s.C)
	if err != nil {
		return nil, err
	}

	if !c.Enabled {
		return nil, errors.New("user %s disabled, cannot continue")
	}

	buser.UserName = c.UserName
	buser.UpdatedAt = c.UpdatedAt
	buser.PasswordCreatedAt = c.PasswordCreatedAt
	buser.Password = c.Password
	buser.OneTimePassword = c.OneTimePassword.String
	buser.LastLoggedIn = c.LastLoggedIn
	buser.ID = int(c.ID)
	buser.Email = c.Email.String
	buser.CreatedAt = c.CreatedAt
	buser.Enabled = c.Enabled

	return buser, nil
}

// InsertUserRPC inserts a user via RPC with context
func (s *SQLite3) InsertUserRPC(ctx context.Context, username, password string) error {
	s.Lock()
	defer s.Unlock()

	newuser := &models.User{
		UserName:          username,
		Password:          password,
		PasswordCreatedAt: time.Now(),
		LastLoggedIn:      time.Now(),
		Enabled:           true,
	}

	return newuser.Insert(ctx, s.C, boil.Infer())
}

// GetExchangeLoadedDataRPC gets distinct values about data entered
func (s *SQLite3) GetExchangeLoadedDataRPC(ctx context.Context, exchange string) ([]*gctrpc.AvailableData, error) {
	s.Lock()
	// because this is querying the exchange platform table we have to commit
	// transaction in memory TODO: make this friendlier
	err := s.NewQuery()
	if err != nil {
		return nil, err
	}
	defer func() { s.FinishQuery(); s.Unlock() }()

	e, err := s.insertAndRetrieveExchange(exchange)
	if err != nil {
		return nil, err
	}

	h, err := models.ExchangePlatformTradeHistories(
		qm.SQL("SELECT DISTINCT currency_pair, asset_type FROM exchange_platform_trade_history"),
		qm.Where("exchange_id = ?", e.ID),
	).All(ctx, s.C)
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
func (s *SQLite3) GetExchangePlatformHistoryRPC(ctx context.Context, exchange, pair, asset string) ([]*gctrpc.PlatformHistory, error) {
	s.Lock()
	// because this is querying the exchange platform table we have to commit
	// transaction in memory TODO: make this friendlier
	err := s.NewQuery()
	if err != nil {
		return nil, err
	}
	defer func() { s.FinishQuery(); s.Unlock() }()

	e, err := s.insertAndRetrieveExchange(exchange)
	if err != nil {
		return nil, err
	}

	h, err := models.ExchangePlatformTradeHistories(
		qm.Where("exchange_id = ?", e.ID),
		qm.Where("currency_pair = ?", pair),
		qm.Where("asset_type = ?", asset),
	).All(ctx, s.C)
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
func (s *SQLite3) GetUserAuditRPC(ctx context.Context, username string) ([]*base.Audit, error) {
	s.Lock()
	defer s.Unlock()

	c, err := models.Users(qm.Where("user_name = ?", username)).One(ctx, s.C)
	if err != nil {
		return nil, err
	}

	audits, err := c.AuditTrails().All(ctx, s.C)
	if err != nil {
		return nil, err
	}

	var auditTrails []*base.Audit
	for i := range audits {
		auditTrails = append(auditTrails, &base.Audit{
			UserID:       audits[i].UserID,
			Change:       audits[i].Change,
			TimeOfChange: audits[i].CreatedAt,
		})
	}

	return auditTrails, nil
}

// GetUsersRPC returns a full user list that is in the database
func (s *SQLite3) GetUsersRPC(ctx context.Context) ([]*base.User, error) {
	s.Lock()
	defer s.Unlock()

	c, err := models.Users().All(ctx, s.C)
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
func (s *SQLite3) EnableDisableUserRPC(ctx context.Context, username string, enable bool) error {
	s.Lock()
	defer s.Unlock()

	c, err := models.Users(qm.Where("user_name = ?", username)).One(ctx, s.C)
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
	_, err = c.Update(ctx, s.C, boil.Infer())
	return err
}

// SetUserPasswordRPC users sets users password TODO: Add table password
// to make sure expired passwords are not reused
func (s *SQLite3) SetUserPasswordRPC(ctx context.Context, username, password string) error {
	s.Lock()
	defer s.Unlock()

	c, err := models.Users(qm.Where("user_name = ?", username)).One(ctx, s.C)
	if err != nil {
		return err
	}

	c.Password = password
	c.PasswordCreatedAt = time.Now()

	_, err = c.Update(ctx, s.C, boil.Infer())
	return err
}

// ModifyUserRPC modifies user details TODO: expand user table fields to
// include firstname, lastname, address details
func (s *SQLite3) ModifyUserRPC(ctx context.Context, username, email string) error {
	if email == "" {
		return errors.New("no user data set")
	}

	s.Lock()
	defer s.Unlock()

	c, err := models.Users(qm.Where("user_name = ?", username)).One(ctx, s.C)
	if err != nil {
		return err
	}

	c.Email.SetValid(email)

	_, err = c.Update(ctx, s.C, boil.Infer())
	return err
}
