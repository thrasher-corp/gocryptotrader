package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/database"
	db "github.com/thrasher-corp/gocryptotrader/database/drivers/postgres"
	dbsqlite3 "github.com/thrasher-corp/gocryptotrader/database/drivers/sqlite"
	mg "github.com/thrasher-corp/gocryptotrader/database/migration"
	"github.com/thrasher-corp/gocryptotrader/database/repository/audit"
	auditPSQL "github.com/thrasher-corp/gocryptotrader/database/repository/audit/postgres"
	auditSQLite "github.com/thrasher-corp/gocryptotrader/database/repository/audit/sqlite"
)

var (
	dbConn         *database.Database
	configFile     string
	defaultDataDir string
)

func openDbConnection(driver string) (err error) {
	if driver == "postgres" {
		dbConn, err = db.Connect()
		if err != nil {
			return fmt.Errorf("database failed to connect: %v Some features that utilise a database will be unavailable", err)
		}

		dbConn.SQL.SetMaxOpenConns(2)
		dbConn.SQL.SetMaxIdleConns(1)
		dbConn.SQL.SetConnMaxLifetime(time.Hour)

		audit.Audit = auditPSQL.Audit()
	} else if driver == "sqlite" {
		dbConn, err = dbsqlite3.Connect()

		if err != nil {
			return fmt.Errorf("database failed to connect: %v Some features that utilise a database will be unavailable", err)
		}

		audit.Audit = auditSQLite.Audit()
	}
	return nil
}

type tmpLogger struct{}

func (t tmpLogger) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v)
}

func (t tmpLogger) Println(v ...interface{}) {
	fmt.Println(v)
}

func (t tmpLogger) Errorf(format string, v ...interface{}) {
	fmt.Printf(format, v)
}


func main() {
	fmt.Println("Gocrytotrader database migration tool")
	fmt.Println("© 2019 Thrasher Corporation")
	fmt.Println()

	tempLogger := tmpLogger{}

	temp := mg.Migrator{
		Log: tempLogger,
	}


	err := temp.LoadMigrations()

	if err != nil {
		temp.Log.Println("Failed to load migrations")
		os.Exit(0)
	}

	defaultPath, err := config.GetFilePath("")
	if err != nil {
			temp.Log.Println(err)
		os.Exit(1)
	}

	flag.StringVar(&configFile, "config", defaultPath, "config file to load")
	flag.StringVar(&defaultDataDir, "datadir", common.GetDefaultDataDir(runtime.GOOS), "default data directory for GoCryptoTrader files")

	flag.Parse()

	conf := config.GetConfig()
	err = conf.LoadConfig(configFile)

	if err != nil {
		temp.Log.Println(err)
		os.Exit(0)
	}

	err = openDbConnection(conf.Database.Driver)
	if err != nil {
		temp.Log.Println(err)
		os.Exit(1)
	}

	temp.Log.Printf("Connected to: %s\n", conf.Database.Host)

	temp.Conn = dbConn

	err = temp.RunMigration()

	if err != nil {
		temp.Log.Println(err)
		os.Exit(1)
	}

	if dbConn.SQL != nil {
		err = dbConn.SQL.Close()
		if err != nil {
			temp.Log.Println(err)
		}
	}

}
