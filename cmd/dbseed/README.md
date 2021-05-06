# GoCryptoTrader dbseed tool

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/page-logo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/portfolio)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This dbseed tool is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## How to use

#### Prerequisites
##### Configuration

dbseed requires a valid database configuration in your gocryptotrader config

```sh
 "database": {
  "enabled": true,
  "verbose": true,
  "driver": "postgres",
  "connectionDetails": {
   "host": "localhost",
   "port": 5432,
   "username": "gct-dev",
   "password": "gct-dev",
   "database": "gct-dev",
   "sslmode": "disable"
  }
 },
```

By default this will load from the default GoCryptoTrader path 

For Windows users this is:
```%APPDATA%\GoCryptoTrader```

For Linux/macOS users this is:
```$HOME\.gocryptotrader```

and can be overridden with the ```-config``` flag

``` --config value  config file to load (default: "~/.gocryptotrader/config.json")```

#### Usage

#### Sub Commands
##### candle
```
   file     seed candle data from a file
   help, h  Shows a list of commands or help for one command
```
##### command examples
```
dbseed candle file --exchange=binance --base=BTC --quote=USDT --interval=86400 --asset=spot --filename=../../testdata/binance_BTCUSDT_24h_2019_01_01_2020_01_01.csv
```
File structure for import contains the following rows with no headers:

```
timestamp, volume, open, high, low, close
```
An example of this is:
```
1546300800,23741.687033,3701.23,3797.14,3642,3797.14
1546387200,35156.463369,3796.45,3858.56,3750.45,3858.56
1546473600,29406.948359,3857.57,3766.78,3730,3766.78
1546560000,29519.554671,3767.2,3792.01,3703.57,3792.01
1546646400,30490.667751,3790.09,3770.96,3751,3770.96
```
##### exchange
```
   file     seed exchange data from a file
   add      add a single exchange
   default  seed exchange from default list
```
##### command examples
```
dbseed exchange add --name=newexchange
dbseed exchange file --filename=../../testdata/exchangelist.csv
dbseed exchange default
```

File structure for importing contains the following rows with no headers:
```
exchange
```
An example of this is:
```
binance,
ftx,
btc markets,
```

## Contribution

Please feel free to submit any pull requests or suggest any desired features to be added.

When submitting a PR, please abide by our coding guidelines:

+ Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
+ Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
+ Code must adhere to our [coding style](https://github.com/thrasher-corp/gocryptotrader/blob/master/doc/coding_style.md).
+ Pull requests need to be based on and opened against the `master` branch.

## Donations

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/donate.png?raw=true" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***

