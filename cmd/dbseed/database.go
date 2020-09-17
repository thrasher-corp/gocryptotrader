package main

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/database"
	dbPSQL "github.com/thrasher-corp/gocryptotrader/database/drivers/postgres"
	dbsqlite3 "github.com/thrasher-corp/gocryptotrader/database/drivers/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/urfave/cli/v2"
)

var (
	dbConn *database.Instance
)

func load(c *cli.Context) error {
	var conf config.Config
	err := conf.LoadConfig(c.String("config"), true)
	if err != nil {
		return err
	}

	if !conf.Database.Enabled {
		return database.ErrDatabaseSupportDisabled
	}

	err = openDBConnection(c, conf.Database.Driver)
	if err != nil {
		return err
	}

	drv := repository.GetSQLDialect()
	if drv == database.DBSQLite || drv == database.DBSQLite3 {
		fmt.Printf("Database file: %s\n", conf.Database.Database)
	} else {
		fmt.Printf("Connected to: %s\n", conf.Database.Host)
	}

	return nil
}

func openDBConnection(c *cli.Context, driver string) (err error) {
	if c.IsSet("verbose") {
		boil.DebugMode = true
	}
	if driver == database.DBPostgreSQL {
		dbConn, err = dbPSQL.Connect()
		if err != nil {
			return fmt.Errorf("database failed to connect: %v, some features that utilise a database will be unavailable", err)
		}
		return nil
	} else if driver == database.DBSQLite || driver == database.DBSQLite3 {
		dbConn, err = dbsqlite3.Connect()
		if err != nil {
			return fmt.Errorf("database failed to connect: %v, some features that utilise a database will be unavailable", err)
		}
		return nil
	}
	return errors.New("no connection established")
}
