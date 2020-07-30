package main

import (
	"flag"
	"path/filepath"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/urfave/cli/v2"
)

var (
	testConfig = filepath.Join("configtest.json")
	testApp    = &cli.App{
		Name:                 "dbseed",
		Version:              core.Version(false),
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Value: testConfig,
				Usage: "config file to load",
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "toggle verbose output",
				Value: true,
			},
		},
		Commands: []*cli.Command{
			seedExchangeCommand,
			seedCandleCommand,
		},
	}
)

func TestLoad(t *testing.T) {
	config.TestBypass = true
	fs := &flag.FlagSet{}
	fs.String("config", testConfig, "")
	newCtx := cli.NewContext(testApp, fs, &cli.Context{})
	err := Load(newCtx)
	if err != nil {
		t.Fatal(err)
	}
}
