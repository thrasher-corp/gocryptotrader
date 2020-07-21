package main

import (
	"fmt"
	"log"
	"os"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/urfave/cli/v2"
)

var (
	configFile string
	dataDir    string
)

func main() {
	fmt.Println("GoCryptoTrader database seeding tool")
	fmt.Println(core.Copyright)
	fmt.Println()

	app := &cli.App{
		Name:                 "dbseed",
		Version:              core.Version(false),
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Value:       config.DefaultFilePath(),
				Usage:       "config file to load",
				Destination: &configFile,
			},
		},
		Commands: []*cli.Command{
			seedExchangeCommand,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	if dbConn.SQL != nil {
		err = dbConn.SQL.Close()
		if err != nil {
			log.Println(err)
		}
	}
}
