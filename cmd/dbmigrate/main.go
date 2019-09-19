package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/goose"

	dbPSQL "github.com/thrasher-corp/gocryptotrader/database/drivers/postgres"
	dbsqlite3 "github.com/thrasher-corp/gocryptotrader/database/drivers/sqlite"
)

var (
	dbConn          *database.Db
	configFile      string
	defaultDataDir  string
	createMigration string
	migrationDir    string
	command         string
	args            string
)

func openDbConnection(driver string) (err error) {
	if driver == "postgres" {
		dbConn, err = dbPSQL.Connect()
		if err != nil {
			return fmt.Errorf("database failed to connect: %v Some features that utilise a database will be unavailable", err)
		}

		dbConn.SQL.SetMaxOpenConns(2)
		dbConn.SQL.SetMaxIdleConns(1)
		dbConn.SQL.SetConnMaxLifetime(time.Hour)

	} else if driver == "sqlite3" || driver == "sqlite" {
		dbConn, err = dbsqlite3.Connect()

		if err != nil {
			return fmt.Errorf("database failed to connect: %v Some features that utilise a database will be unavailable", err)
		}
	}
	return nil
}

func main() {
	fmt.Println("GoCryptoTrader database migration tool")
	fmt.Println(core.Copyright)
	fmt.Println()

	defaultPath, err := config.GetFilePath("")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	flag.StringVar(&command, "command", "status", "command to run")
	flag.StringVar(&args, "args", "", "arguments to pass to goose")

	flag.StringVar(&configFile, "config", defaultPath, "config file to load")
	flag.StringVar(&defaultDataDir, "datadir", common.GetDefaultDataDir(runtime.GOOS), "default data directory for GoCryptoTrader files")
	flag.StringVar(&createMigration, "create", "", "create a new empty migration file")
	flag.StringVar(&migrationDir, "migrationdir", database.MigrationDir, "override migration folder")

	flag.Parse()

	conf := config.GetConfig()

	err = conf.LoadConfig(configFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	err = openDbConnection(conf.Database.Driver)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	drv := driverConvert(conf.Database.Driver)

	if drv == "sqlite" {
		fmt.Printf("Database file: %s\n", conf.Database.Database)
	} else {
		fmt.Printf("Connected to: %s\n", conf.Database.Host)
	}

	if err := goose.Run(command, dbConn.SQL, drv, migrationDir, args); err != nil {
		fmt.Println(err)
	}
}

func driverConvert(in string) (out string) {
	switch strings.ToLower(in) {
	case "postgresql", "postgres", "psql":
		out = "postgres"
	case "sqlite3", "sqlite":
		out = "sqlite"
	}
	return
}
