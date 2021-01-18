# GoCryptoTrader Backtester: Backtester package

<img src="https://github.com/gloriousCode/gocryptotrader/blob/backscratcher/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://travis-ci.org/thrasher-corp/gocryptotrader.svg?branch=master)](https://travis-ci.org/thrasher-corp/gocryptotrader)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This backtester package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)


# GoCryptoTrader Backtester
An event-driven backtesting tool to test and iterate trading strategies using historical or custom data.


## Features
- Works with all GoCryptoTrader exchanges that support trade/candle retrieval
- API data retrieval
- CSV data import
- Database data import
- Proof of concept live data running
- Can run strategies against multiple cryptocurrencies
- Can run strategies that can assess multiple currencies to make complex decisions
- Dollar cost strategy implementation
- RSI strategy implementation
- Strategy customisation via config `.strat` files
- Report generation
- Portfolio manager to help size orders based on config rules, risk and candle volume
- Order manager to place orders with customisable slippage estimator
- Helpful statistics to help determine whether a strategy was effective
- Compliance manager to keep snapshots of every transaction and their changes at every interval

## How does it work?
- The application will load a `.strat` config file as specified at runtime
- The `.strat` config file will contain
  - Start & end dates
  - The strategy to run
  - The candle interval
  - Where the data is to be sourced (API, CSV, database, live)
  - What type of data to use (trade or candle)
  - A nickname for the strategy (to help differentiate between runs/configs using the same strategy)
  - The currency/currencies to use
  - The exchange(s) to run against

# Cool story, how do I use it?
To run the application using the provided dollar cost average strategy, simply run `go run .` from `gocryptotrader/backtester`. An output of the results will be put in the `results` folder


# Things to consider
- There are many more things to add and tweak. Bring up issues on our Github or Slack.



### Please click GoDocs chevron above to view current GoDoc information for this package

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
