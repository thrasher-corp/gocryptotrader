package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/database"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
)

var (
	configFile     string
	defaultDataDir string
)

var sqlboilerConfig map[string]driverConfig

type driverConfig struct {
	Dbname    string   `json:"dbname,omitempty"`
	Host      string   `json:"host,omitempty"`
	Port      uint16   `json:"port,omitempty"`
	User      string   `json:"user,omitempty"`
	Pass      string   `json:"pass,omitempty"`
	Schema    string   `json:"schema,omitempty"`
	Sslmode   string   `json:"sslmode,omitempty"`
	Blacklist []string `json:"blacklist,omitempty"`
}

func main() {
	fmt.Println("GoCryptoTrader SQLBoiler config generation tool")
	fmt.Println(core.Copyright)
	fmt.Println()

	defaultPath, err := config.GetFilePath("")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	flag.StringVar(&configFile, "config", defaultPath, "config file to load")
	flag.StringVar(&defaultDataDir, "datadir", common.GetDefaultDataDir(runtime.GOOS), "default data directory for GoCryptoTrader files")

	conf := config.GetConfig()

	err = conf.LoadConfig(configFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	convertGCTtoSQLBoilerConfig(&conf.Database)

	jsonOutput, err := json.MarshalIndent(sqlboilerConfig, "", " ")

	if err != nil {
		fmt.Printf("Marshal failed: %v", err)
		os.Exit(0)
	}

	err = ioutil.WriteFile("sqlboiler.json", jsonOutput, 0644)
	if err != nil {
		fmt.Printf("Write failed: %v", err)
		os.Exit(0)
	}
	fmt.Println("sqlboiler.json file created")
}

func convertGCTtoSQLBoilerConfig(c *database.Config) {
	tempConfig := driverConfig{
		Blacklist: []string{"goose_db_version"},
	}

	sqlboilerConfig = make(map[string]driverConfig)

	dbType := driverConvert(c.Driver)

	if dbType == "sqlite3" {
		tempConfig.Dbname = convertDBName(c.Database)
	} else {
		tempConfig.User = c.Username
		tempConfig.Pass = c.Password
		tempConfig.Port = c.Port
		tempConfig.Host = c.Host
		tempConfig.Dbname = c.Database
		tempConfig.Sslmode = c.SSLMode
	}

	sqlboilerConfig[dbType] = tempConfig
}

func driverConvert(in string) (out string) {
	switch strings.ToLower(in) {
	case "postgresql", "postgres", "psql":
		out = "psql"
	case "sqlite3", "sqlite":
		out = "sqlite3"
	}
	return
}

func convertDBName(in string) (out string) {

	x := in[len(in)-3:]

	if x != ".db" {
		in += ".db"
	}

	return filepath.Join(common.GetDefaultDataDir(runtime.GOOS), "/database", in)
}
