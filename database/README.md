# GoCryptoTrader package Database

<img src="https://github.com/thrasher-/gocryptotrader/blob/master/web/src/assets/page-logo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://travis-ci.org/thrasher-/gocryptotrader.svg?branch=master)](https://travis-ci.org/thrasher-/gocryptotrader)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-/gocryptotrader/database)
[![Coverage Status](http://codecov.io/github/thrasher-/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-/gocryptotrader)


This database package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progresss on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://gocryptotrader.herokuapp.com/)

## Database

GoCryptoTrader is using an interim database SQLite3 and using SQLBoiler for its model generation

Big thank you to the team at volatiletech for providing this tool https://github.com/volatiletech/sqlboiler

### Current Features

+ User account creation
+ Insert, Ammend configuration data
+ Encryption of configuration data 
+ Insert, Ammend exchange trade history data

### Features not yet included

+ Order insertion and tracking

### How to enable

+ Using gocryptotrader binary will create db automatically no setup required.
+ To enable loading of exchange history data use the `-history` flag, this will seed historic trade action that has been matched by exchange.
+ To override a saved configuration please use the `-o-config` flag, this will load a supplied config.json into memory.
+ To save a new configuration you can use the `-save-config` flag, please be aware that if you do not change the configuration name it will overwrite your initial configuration loaded in database.
+ To use a saved configuration in database use the `-use-config` flag then appened the name e.g. `-use-config arb_euro`.
+ You can specify a unique path to a database by using the `-db-path` flag e.g. `-db-path ./newdatabase.db`.

+ If enabled via individually importing package, rudimentary example below:

```go
    var verbose bool // Set to true if verbose is required
    err := Setup("Pass in desired test directory", verbose)
	if err != nil {
		// Handle error
	}

    cfg := config.GetConfig() // Get your configuration
	err = cfg.LoadConfig("Path to your congiguration.json file")
	if err != nil {
		// Handle error
	}

	db, err = Connect("Path to your database file", verbose)
	if err != nil {
		// Handle error
	}

    err = db.InsertExchangeTradeHistoryData(...)
    if err != nil {
        // Handle error
    }

    err = db.GetExchangeTradeHistory(...)
    if err != nil {
        // Handle error
    }
```

### Please click GoDocs chevron above to view current GoDoc information for this package

## Contribution

Please feel free to submit any pull requests or suggest any desired features to be added.

When submitting a PR, please abide by our coding guidelines:

+ Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
+ Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
+ Code must adhere to our [coding style](https://github.com/thrasher-/gocryptotrader/blob/master/doc/coding_style.md).
+ Pull requests need to be based on and opened against the `master` branch.

## Donations

<img src="https://github.com/thrasher-/gocryptotrader/blob/master/web/src/assets/donate.png?raw=true" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB***

