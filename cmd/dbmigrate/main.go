package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/pressly/goose"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/database"

	dbPSQL "github.com/thrasher-corp/gocryptotrader/database/drivers/postgres"
)

var (
	dbConn          *database.Db
	configFile      string
	defaultDataDir  string
	createMigration string
	migrationDir    string
	command         string
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

	} else if driver == "sqlite" {

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
	flag.StringVar(&command, "command", "", "command to run")
	flag.StringVar(&configFile, "config", defaultPath, "config file to load")
	flag.StringVar(&defaultDataDir, "datadir", common.GetDefaultDataDir(runtime.GOOS), "default data directory for GoCryptoTrader files")
	flag.StringVar(&createMigration, "create", "", "create a new empty migration file")
	flag.StringVar(&migrationDir, "migrationdir", database.MigrationDir, "override migration folder")

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		command = "status"
	} else if len(args) < 2 {
		command = args[0]
	}

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

	fmt.Printf("Connected to: %s\n", conf.Database.Host)

	if err := goose.Run(command, dbConn.SQL, migrationDir, ""); err != nil {
		fmt.Println(err)
	}
}
