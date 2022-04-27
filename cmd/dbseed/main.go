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
	app = &cli.App{
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
			&cli.BoolFlag{
				Name:        "verbose",
				Usage:       "toggle verbose output",
				Destination: &verbose,
			},
		},
		Commands: []*cli.Command{
			seedExchangeCommand,
			seedCandleCommand,
		},
	}
	workingDir string
	configFile string
	verbose    bool
)

func main() {
	var err error
	workingDir, err = os.Getwd()
	if err != nil {
		log.Println("error getting current working path")
		workingDir = "."
	}

	fmt.Println("GoCryptoTrader database seeding tool")
	fmt.Println(core.Copyright)
	fmt.Println()

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	if dbConn != nil {
		if dbConn.SQL != nil {
			err = dbConn.SQL.Close()
			if err != nil {
				log.Println(err)
			}
		}
	}
}
